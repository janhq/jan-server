# Kubernetes Helm Values Generation

Automatic generation of Kubernetes Helm `values.yaml` files from Jan Server configuration.

## Overview

The K8s values generator maps Jan Server's unified configuration to Helm chart values, enabling:

- **Single source of truth**: Configuration drives both local and K8s deployments
- **Environment-specific overrides**: Development, staging, production profiles
- **Consistent deployment**: Same config structure across all environments
- **Type safety**: Generated from Go structs with validation

## Architecture

```
config/defaults.yaml
      ↓
  Config Loader
      ↓
Values Generator ----→ values-development.yaml (1 replica, minimal resources)
      ├-----------→ values-staging.yaml (2 replicas, moderate resources)
      └-----------→ values-production.yaml (3 replicas, full resources, persistence)
```

## Generated Structure

### Global Values
```yaml
global:
  environment: development
  imagePullPolicy: IfNotPresent
  labels:
    app.kubernetes.io/name: jan-server
    app.kubernetes.io/version: 1.0.0
    app.kubernetes.io/environment: development
```

### Service Values
Each service gets:
- **Deployment**: replicas, image, resources
- **Service**: type, ports
- **Health Checks**: liveness/readiness probes
- **Config**: ConfigMaps for non-sensitive config
- **Secrets**: References to K8s secrets

Example:
```yaml
services:
  llm-api:
    enabled: true
    replicaCount: 2
    image:
      repository: jan-llm-api
      tag: 1.0.0
    service:
      type: ClusterIP
      port: 8080
      targetPort: 8080
    resources:
      limits:
        cpu: 1000m
        memory: 1Gi
      requests:
        cpu: 500m
        memory: 512Mi
    healthChecks:
      livenessProbe:
        httpGet:
          path: /health
          port: 8080
        initialDelaySeconds: 30
      readinessProbe:
        httpGet:
          path: /health
          port: 8080
        initialDelaySeconds: 10
    configMap:
      LOG_LEVEL: info
      LOG_FORMAT: json
    secrets:
      - database-credentials
      - keycloak-credentials
```

### Infrastructure Values
Database and auth configuration:
```yaml
infrastructure:
  database:
    postgres:
      enabled: true
      host: api-db
      port: 5432
      database: jan_llm_api
      user: jan_user
      passwordSecret: postgres-password
      maxConnections: 100
      resources:
        limits:
          cpu: 2000m
          memory: 2Gi
      persistence:
        enabled: true
        size: 10Gi
  auth:
    keycloak:
      enabled: true
      baseURL: http://keycloak:8085
      adminUser: admin
      passwordSecret: keycloak-admin-password
      resources:
        limits:
          cpu: 1000m
          memory: 1Gi
```

## Environment Profiles

### Development
- Single replica per service
- Minimal resources (100m CPU, 128Mi RAM requests)
- `imagePullPolicy: Never` (use local images)
- Persistence disabled
- Lower health check thresholds

### Staging
- 2 replicas per service
- Moderate resources (250m CPU, 256Mi RAM requests)
- `imagePullPolicy: IfNotPresent`
- Persistence enabled (20Gi)
- Production-like settings

### Production
- 3 replicas per service
- Full resources (500m CPU, 512Mi RAM requests)
- `imagePullPolicy: Always`
- Persistence enabled (50Gi)
- Strict health checks
- Higher failure thresholds

## Usage

### Programmatic Generation

```go
package main

import (
    "context"
    "github.com/janhq/jan-server/pkg/config"
    "github.com/janhq/jan-server/pkg/config/k8s"
)

func main() {
    // Load configuration
    loader := config.NewConfigLoader("development", "config/defaults.yaml")
    cfg, _ := loader.Load(context.Background())

    // Create generator
    generator := k8s.NewValuesGenerator(cfg)

    // Generate base values
    generator.GenerateToFile("values.yaml")

    // Generate with environment overrides
    values, _ := generator.GenerateWithOverrides("production")
    // Use values...
}
```

### Example Program

See `pkg/config/k8s/examples/generate_values.go`:

```bash
cd pkg/config/k8s/examples
go run generate_values.go
```

Generates:
- `values-dev.yaml` - Development environment
- `values-prod.yaml` - Production environment

## Configuration Mapping

| Config Path | Helm Values Path | Notes |
|-------------|-----------------|-------|
| `meta.version` | `global.labels["app.kubernetes.io/version"]`, `*.image.tag` | Version across all components |
| `meta.environment` | `global.environment`, `global.labels["app.kubernetes.io/environment"]` | Environment name |
| `services.llm_api.http_port` | `services.llm-api.service.port` | Service port mapping |
| `services.llm_api.log_level` | `services.llm-api.configMap.LOG_LEVEL` | Config as ConfigMap |
| `infrastructure.database.postgres.*` | `infrastructure.database.postgres.*` | Direct mapping |
| `infrastructure.auth.keycloak.*` | `infrastructure.auth.keycloak.*` | Direct mapping |

## Secret Management

**Important**: The generator creates **references** to secrets, not the secrets themselves.

Secret references in values:
```yaml
services:
  llm-api:
    secrets:
      - database-credentials  # → maps to K8s Secret
      - keycloak-credentials
```

You must create K8s secrets separately:
```bash
kubectl create secret generic database-credentials \
  --from-literal=password=<db-password>

kubectl create secret generic keycloak-credentials \
  --from-literal=admin-password=<keycloak-password>
```

Secrets are managed by DevOps via Kubernetes Secrets, HashiCorp Vault, or environment variables.

## Resource Sizing

### Default Resources

| Service | CPU Request | CPU Limit | Memory Request | Memory Limit |
|---------|------------|-----------|----------------|--------------|
| llm-api | 500m | 1000m | 512Mi | 1Gi |
| mcp-tools | 250m | 500m | 256Mi | 512Mi |
| media-api | 250m | 500m | 256Mi | 512Mi |
| response-api | 250m | 500m | 256Mi | 512Mi |
| postgres | 1000m | 2000m | 1Gi | 2Gi |
| keycloak | 500m | 1000m | 512Mi | 1Gi |

### Scaling Recommendations

For production workloads:
- **High traffic**: Increase replicas (3-5+)
- **Heavy AI workloads**: Increase llm-api resources (2-4 CPU, 2-4Gi RAM)
- **Large databases**: Increase postgres resources and persistence size

## Health Checks

### Liveness Probes
- Detect crashed containers
- Restart unhealthy containers
- Higher failure threshold (3-5)

### Readiness Probes
- Detect startup completion
- Remove from service during issues
- Lower failure threshold (3)

Generated probes:
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
```

## Integration with Helm

### Deploy with Generated Values

```bash
# Generate values
go run pkg/config/k8s/examples/generate_values.go

# Install chart with generated values
helm install jan-server k8s/jan-server \
  -f pkg/config/k8s/examples/values-prod.yaml
```

### Override Specific Values

```bash
# Use generated base + custom overrides
helm install jan-server k8s/jan-server \
  -f values-prod.yaml \
  --set services.llm-api.replicaCount=5
```

### CI/CD Integration

```yaml
# .github/workflows/deploy.yml
- name: Generate Helm Values
  run: |
    go run pkg/config/k8s/examples/generate_values.go
    
- name: Deploy to K8s
  run: |
    helm upgrade --install jan-server k8s/jan-server \
      -f values-prod.yaml \
      --namespace jan-server
```

## Limitations

- **Not a replacement for Helm templating**: Generator creates values, Helm chart still needed
- **Basic resource sizing**: May need tuning based on actual workload
- **Secrets not generated**: K8s secrets must be created separately by DevOps team
- **No autoscaling**: HPA configuration not generated (add manually)

## Future Enhancements

- [ ] Autoscaling (HPA) configuration
- [ ] Ingress generation
- [ ] Network policies
- [ ] Pod disruption budgets
- [ ] Service mesh integration (Istio, Linkerd)
- [ ] Custom resource definitions (CRDs)

## See Also

- [Configuration System](README.md)
- [Helm Chart Documentation](../../k8s/jan-server/README.md)
- [Deployment Guide](../../docs/guides/deployment.md)
