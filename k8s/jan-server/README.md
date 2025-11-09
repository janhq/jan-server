# Jan Server Helm Chart

This Helm chart deploys the complete Jan Server platform on Kubernetes, including:

- **LLM API** - Core LLM orchestration service
- **Media API** - Media upload and management service
- **MCP Tools** - Model Context Protocol tools and utilities
- **Keycloak** - Authentication and authorization server
- **Kong** - API Gateway for unified API endpoint
- **PostgreSQL** - Primary database (via Bitnami chart)
- **Redis** - Caching and session store (via Bitnami chart)
- **SearXNG** - Meta search engine for MCP
- **Vector Store** - Lightweight vector database for file search
- **SandboxFusion** - Code interpreter and execution environment

## Prerequisites

- Kubernetes 1.23+
- Helm 3.8+
- PV provisioner support in the underlying infrastructure (for persistent volumes)
- LoadBalancer support (for Kong ingress) or Ingress Controller

## Installing the Chart

### Add Bitnami Repository (for PostgreSQL and Redis)

```bash
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
```

### Install from local directory

```bash
# From the k8s directory
helm install jan-server ./jan-server \
  --namespace jan-server \
  --create-namespace \
  --values ./jan-server/values.yaml
```

### Install with custom values

```bash
helm install jan-server ./jan-server \
  --namespace jan-server \
  --create-namespace \
  --values ./jan-server/values-production.yaml
```

## Configuration

The following table lists the configurable parameters and their default values.

### Global Settings

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.imageRegistry` | Global Docker registry | `""` |
| `global.imagePullSecrets` | Global image pull secrets | `[]` |
| `global.storageClass` | Global storage class | `""` |

### PostgreSQL (Bitnami)

| Parameter | Description | Default |
|-----------|-------------|---------|
| `postgresql.enabled` | Enable PostgreSQL | `true` |
| `postgresql.auth.username` | PostgreSQL username | `jan_user` |
| `postgresql.auth.password` | PostgreSQL password | `jan_password` |
| `postgresql.auth.database` | PostgreSQL database | `jan_llm_api` |
| `postgresql.primary.persistence.size` | PVC size | `10Gi` |

### LLM API

| Parameter | Description | Default |
|-----------|-------------|---------|
| `llmApi.enabled` | Enable LLM API | `true` |
| `llmApi.replicaCount` | Number of replicas | `2` |
| `llmApi.image.repository` | Image repository | `jan/llm-api` |
| `llmApi.image.tag` | Image tag | `latest` |
| `llmApi.service.type` | Service type | `ClusterIP` |
| `llmApi.service.port` | Service port | `8080` |
| `llmApi.resources.requests.memory` | Memory request | `256Mi` |
| `llmApi.resources.requests.cpu` | CPU request | `250m` |
| `llmApi.autoscaling.enabled` | Enable autoscaling | `false` |
| `llmApi.ingress.enabled` | Enable ingress | `false` |

### Media API

| Parameter | Description | Default |
|-----------|-------------|---------|
| `mediaApi.enabled` | Enable Media API | `true` |
| `mediaApi.replicaCount` | Number of replicas | `2` |
| `mediaApi.image.repository` | Image repository | `jan/media-api` |
| `mediaApi.image.tag` | Image tag | `latest` |
| `mediaApi.service.port` | Service port | `8285` |
| `mediaApi.secrets.s3Endpoint` | S3 endpoint URL | `https://s3.menlo.ai` |
| `mediaApi.secrets.s3Bucket` | S3 bucket name | `platform-dev` |
| `mediaApi.secrets.s3AccessKey` | S3 access key | `XXXXX` |
| `mediaApi.secrets.s3SecretKey` | S3 secret key | `YYYY` |

### MCP Tools

| Parameter | Description | Default |
|-----------|-------------|---------|
| `mcpTools.enabled` | Enable MCP Tools | `true` |
| `mcpTools.replicaCount` | Number of replicas | `2` |
| `mcpTools.service.port` | Service port | `8091` |
| `mcpTools.secrets.serperApiKey` | Serper API key | `""` |

### Keycloak

| Parameter | Description | Default |
|-----------|-------------|---------|
| `keycloak.enabled` | Enable Keycloak | `true` |
| `keycloak.admin.username` | Admin username | `admin` |
| `keycloak.admin.password` | Admin password | `changeme` |
| `keycloak.service.port` | Service port | `8085` |

### Kong API Gateway

| Parameter | Description | Default |
|-----------|-------------|---------|
| `kong.enabled` | Enable Kong | `true` |
| `kong.service.type` | Service type | `LoadBalancer` |
| `kong.service.port` | Service port | `8000` |

## Upgrading

```bash
helm upgrade jan-server ./jan-server \
  --namespace jan-server \
  --values ./jan-server/values-production.yaml
```

## Uninstalling

```bash
helm uninstall jan-server --namespace jan-server
```

To delete the PVCs as well:

```bash
kubectl delete pvc -n jan-server --all
```

## Examples

### Production Deployment

```bash
# Create a production values file
cat > values-prod.yaml <<EOF
global:
  storageClass: "gp3"

postgresql:
  primary:
    persistence:
      size: 50Gi
  auth:
    password: "STRONG_PASSWORD_HERE"

llmApi:
  replicaCount: 3
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10
  resources:
    requests:
      memory: 512Mi
      cpu: 500m
    limits:
      memory: 1Gi
      cpu: 1000m
  ingress:
    enabled: true
    className: "nginx"
    hosts:
      - host: api.yourdomain.com
        paths:
          - path: /
            pathType: Prefix

mediaApi:
  secrets:
    s3AccessKey: "YOUR_ACCESS_KEY"
    s3SecretKey: "YOUR_SECRET_KEY"
    serviceKey: "YOUR_SERVICE_KEY"

keycloak:
  admin:
    password: "STRONG_ADMIN_PASSWORD"
  ingress:
    enabled: true
    hosts:
      - host: auth.yourdomain.com
EOF

# Install with production values
helm install jan-server ./jan-server \
  --namespace jan-server \
  --create-namespace \
  --values values-prod.yaml
```

### Development Deployment (Minimal Resources)

```bash
cat > values-dev.yaml <<EOF
llmApi:
  replicaCount: 1
  resources:
    requests:
      memory: 128Mi
      cpu: 100m

mediaApi:
  replicaCount: 1
  resources:
    requests:
      memory: 128Mi
      cpu: 100m

mcpTools:
  replicaCount: 1

postgresql:
  primary:
    persistence:
      size: 5Gi
    resources:
      requests:
        memory: 128Mi
        cpu: 100m
EOF

helm install jan-server ./jan-server \
  --namespace jan-server-dev \
  --create-namespace \
  --values values-dev.yaml
```

## Accessing Services

After installation, you can access services via:

### Via Kong API Gateway (Recommended)

```bash
# Get Kong external IP
kubectl get svc -n jan-server jan-server-kong

# Access services through Kong
curl http://<KONG_IP>:8000/api/llm/healthz
curl http://<KONG_IP>:8000/api/media/healthz
curl http://<KONG_IP>:8000/api/mcp/healthz
```

### Direct Service Access (Port Forward)

```bash
# LLM API
kubectl port-forward -n jan-server svc/jan-server-llm-api 8080:8080

# Media API
kubectl port-forward -n jan-server svc/jan-server-media-api 8285:8285

# Keycloak Admin Console
kubectl port-forward -n jan-server svc/jan-server-keycloak 8085:8085
# Visit: http://localhost:8085
```

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n jan-server
```

### View Logs

```bash
# LLM API logs
kubectl logs -n jan-server -l app.kubernetes.io/component=llm-api --tail=100

# Media API logs
kubectl logs -n jan-server -l app.kubernetes.io/component=media-api --tail=100

# Keycloak logs
kubectl logs -n jan-server -l app.kubernetes.io/component=keycloak --tail=100
```

### Check Service Connectivity

```bash
# Test internal service connectivity
kubectl run -n jan-server test-pod --rm -it --image=curlimages/curl -- sh

# Inside the pod:
curl http://jan-server-llm-api:8080/healthz
curl http://jan-server-media-api:8285/healthz
curl http://jan-server-keycloak:8085
```

### Database Connection Issues

```bash
# Check PostgreSQL status
kubectl get pods -n jan-server -l app.kubernetes.io/name=postgresql

# Connect to PostgreSQL
kubectl exec -it -n jan-server jan-server-postgresql-0 -- psql -U jan_user -d jan_llm_api
```

## Security Considerations

1. **Change default passwords** in production
2. **Enable TLS/HTTPS** for all ingresses
3. **Use Kubernetes Secrets** for sensitive data
4. **Enable Network Policies** to restrict pod-to-pod communication
5. **Use Pod Security Policies** or Pod Security Standards
6. **Regular security audits** and updates

## Support

For issues and questions:
- GitHub: https://github.com/janhq/jan-server
- Documentation: https://docs.jan.ai

## License

See the main project LICENSE file.
