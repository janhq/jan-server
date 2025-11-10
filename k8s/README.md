# Jan Server Kubernetes Deployment

Complete Helm chart for deploying Jan Server on Kubernetes.

## 📋 Overview

This directory contains Helm charts for deploying the entire Jan Server stack:

```
┌─────────────────────────────────────────────────────────┐
│                    Kong API Gateway                      │
│                    (LoadBalancer)                        │
└─────────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
   ┌─────────┐      ┌──────────┐      ┌──────────┐
   │ LLM API │      │Media API │      │MCP Tools │
   └─────────┘      └──────────┘      └──────────┘
        │                  │                  │
        └──────────────────┼──────────────────┘
                           │
                           ▼
                   ┌──────────────┐
                   │  PostgreSQL  │
                   └──────────────┘
                           
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│   Keycloak   │   │    Redis     │   │   SearXNG    │
└──────────────┘   └──────────────┘   └──────────────┘

┌──────────────┐   ┌──────────────┐
│ Vector Store │   │SandboxFusion │
└──────────────┘   └──────────────┘
```

## 🚀 Quick Start

### Prerequisites

```bash
# Kubernetes cluster (1.23+)
kubectl version

# Helm 3.8+
helm version

# Add Bitnami repository
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
```

**Important:** If you don't have a Kubernetes cluster yet, see [SETUP.md](./SETUP.md) for detailed instructions on setting up Docker Desktop Kubernetes or minikube.

### Install Jan Server

**For Minikube Development Setup**, follow these steps:

```bash
# Step 1: Build Go services and Docker images
cd services/llm-api && go mod tidy && docker build -t jan/llm-api:latest .
cd ../media-api && go mod tidy && docker build -t jan/media-api:latest .
cd ../mcp-tools && go mod tidy && docker build -t jan/mcp-tools:latest .
cd ../../keycloak && docker build -t jan/keycloak:latest .
cd ..

# Step 2: Load images into minikube
minikube image load jan/llm-api:latest jan/media-api:latest jan/mcp-tools:latest jan/keycloak:latest
docker pull bitnami/postgresql:latest bitnami/redis:latest
minikube image load bitnami/postgresql:latest bitnami/redis:latest

# Step 3: Build Helm dependencies
cd k8s/jan-server
helm dependency build

# Step 4: Install
cd ..
helm install jan-server ./jan-server \
  --namespace jan-server \
  --create-namespace

# Step 5: Create additional databases
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=postgresql -n jan-server --timeout=300s
kubectl exec -n jan-server jan-server-postgresql-0 -- bash -c "PGPASSWORD=postgres psql -U postgres << 'EOF'
CREATE USER media WITH PASSWORD 'media';
CREATE DATABASE media_api OWNER media;
CREATE USER keycloak WITH PASSWORD 'keycloak';
CREATE DATABASE keycloak OWNER keycloak;
EOF"
```

**For Production with Cloud Kubernetes**, use values-production.yaml:

```bash
helm install jan-server ./jan-server \
  --namespace jan-server \
  --create-namespace \
  --values ./jan-server/values-production.yaml \
  --wait \
  --timeout 15m
```

### Access Services

```bash
# Check deployment status
kubectl get pods -n jan-server

# Port forward to access services locally (in separate terminals)
kubectl port-forward -n jan-server svc/jan-server-llm-api 8080:8080
kubectl port-forward -n jan-server svc/jan-server-media-api 8285:8285
kubectl port-forward -n jan-server svc/jan-server-keycloak 8085:8085

# Test health endpoints
curl http://localhost:8080/healthz
curl http://localhost:8285/healthz

# Access Keycloak Admin Console
# Username: admin, Password: changeme
open http://localhost:8085
```

**Note:** Kong is available but may show restarts due to memory constraints in minikube. For production, increase Kong's memory limits or access services directly.

## 📦 Components

### Core Services

| Service | Port | Description | Status |
|---------|------|-------------|--------|
| LLM API | 8080 | Core LLM orchestration service | ✅ Working |
| Media API | 8285 | Media upload and management | ✅ Working |
| MCP Tools | 8091 | Model Context Protocol tools | ✅ Working |
| Keycloak | 8085 | Authentication server | ✅ Working |
| Kong | 8000 | Unified API Gateway | ⚠️ Optional |

### Supporting Services

| Service | Port | Description | Status |
|---------|------|-------------|--------|
| PostgreSQL | 5432 | Primary database (3 databases) | ✅ Working |
| Redis | 6379 | Caching and sessions | ✅ Working |
| SearXNG | 8080 | Meta search engine | ✅ Working |
| SandboxFusion | 8080 | Code interpreter | ✅ Working |
| Vector Store | 3015 | File search database | 🔴 Disabled by default |

## 🔧 Configuration

### Values Files

- `values.yaml` - Default values (for minikube/development with imagePullPolicy: Never)
- `values-production.yaml` - Production configuration (for cloud with IfNotPresent)
- `values-development.yaml` - Minimal resource allocation

### Key Configuration Areas

#### 1. Image Pull Policy (Important for Minikube)

For minikube with locally built images:
```yaml
llmApi:
  image:
    pullPolicy: Never  # Use local images only

postgresql:
  image:
    tag: "latest"
    pullPolicy: Never  # Use local Bitnami images

redis:
  image:
    tag: "latest"
    pullPolicy: Never  # Use local Bitnami images
```

For production with image registries:
```yaml
llmApi:
  image:
    pullPolicy: IfNotPresent  # Pull if not present
```

#### 2. Database Configuration

PostgreSQL creates the primary database automatically. Additional databases are created manually:
```yaml
postgresql:
  auth:
    username: jan_user
    password: jan_password  # Change in production!
    database: jan_llm_api
    postgresPassword: postgres  # Change in production!
```

**Note:** Media API and Keycloak databases must be created manually after deployment (see SETUP.md).

#### 3. Environment Variables

LLM API requires these key settings:
```yaml
llmApi:
  env:
    JAN_DEFAULT_NODE_SETUP: "false"  # Disable if no Jan provider available
    DATABASE_URL: "postgres://..."   # Auto-configured via secret
    KEYCLOAK_BASE_URL: "http://..."  # Auto-configured
    BACKEND_CLIENT_ID: "llm-api"
    TARGET_CLIENT_ID: "jan-client"
```

#### 4. S3 Storage (Media API)

**Required** for media-api to function:
```yaml
mediaApi:
  secrets:
    s3Endpoint: "https://s3.amazonaws.com"
    s3Bucket: "your-bucket"  # Required!
    s3AccessKey: "YOUR_KEY"   # Required!
    s3SecretKey: "YOUR_SECRET"  # Required!
```

#### 5. Keycloak Admin

```yaml
keycloak:
  admin:
    username: admin
    password: "changeme"  # Change in production!
  database:
    password: keycloak  # Change in production!
```

#### 6. Resource Limits

Adjust based on your environment:
```yaml
llmApi:
  resources:
    requests:
      memory: 256Mi  # Minimum for minikube
      cpu: 250m
    limits:
      memory: 512Mi
      cpu: 500m

# For production, increase limits:
# memory: 1Gi, cpu: 1000m
```

#### 7. Autoscaling (Disabled by default)

```yaml
llmApi:
  autoscaling:
    enabled: false  # Enable for production
    minReplicas: 2
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
```

#### 8. Ingress Configuration

```yaml
llmApi:
  ingress:
    enabled: true
    className: "nginx"
    hosts:
      - host: api.yourdomain.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: api-tls
        hosts:
          - api.yourdomain.com
```

## 🌐 Deployment Scenarios

### Development (Minikube) - Verified Working ✅

```bash
# Start minikube with enough resources
minikube start --cpus=4 --memory=8192 --driver=docker

# Build and load images (see SETUP.md for complete steps)
# ... build services and docker images ...
minikube image load jan/llm-api:latest jan/media-api:latest jan/mcp-tools:latest jan/keycloak:latest

# Install
cd k8s
helm install jan-server ./jan-server \
  --namespace jan-server \
  --create-namespace

# Create databases
kubectl exec -n jan-server jan-server-postgresql-0 -- bash -c "PGPASSWORD=postgres psql -U postgres << 'EOF'
CREATE USER media WITH PASSWORD 'media';
CREATE DATABASE media_api OWNER media;
CREATE USER keycloak WITH PASSWORD 'keycloak';
CREATE DATABASE keycloak OWNER keycloak;
EOF"

# Access via port-forward
kubectl port-forward -n jan-server svc/jan-server-llm-api 8080:8080
```

### Docker Desktop Kubernetes

```bash
# Build images (same as minikube)
# Images are automatically available in Docker Desktop's Kubernetes

# Install with IfNotPresent pull policy
helm install jan-server ./jan-server \
  --namespace jan-server \
  --create-namespace \
  --set llmApi.image.pullPolicy=IfNotPresent \
  --set mediaApi.image.pullPolicy=IfNotPresent \
  --set mcpTools.image.pullPolicy=IfNotPresent \
  --set keycloak.image.pullPolicy=IfNotPresent
```

### Cloud Kubernetes (AKS/EKS/GKE)

```bash
# Option 1: Use cloud-managed databases (recommended)
helm install jan-server ./jan-server \
  --namespace jan-server \
  --create-namespace \
  --set postgresql.enabled=false \
  --set redis.enabled=false \
  --set global.postgresql.host=your-managed-postgres.cloud \
  --set global.redis.host=your-managed-redis.cloud \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.hosts[0].host=jan.yourdomain.com \
  --set llmApi.autoscaling.enabled=true \
  --set llmApi.replicaCount=3

# Option 2: Use in-cluster databases with persistent storage
helm install jan-server ./jan-server \
  --namespace jan-server \
  --create-namespace \
  --set postgresql.persistence.enabled=true \
  --set postgresql.persistence.size=50Gi \
  --set postgresql.persistence.storageClass=gp3 \
  --set redis.master.persistence.enabled=true \
  --set ingress.enabled=true \
  --set llmApi.autoscaling.enabled=true
```

### Production On-Premises

```bash
# Use production values template
helm install jan-server ./jan-server \
  --namespace jan-server \
  --create-namespace \
  --values ./jan-server/values-production.yaml \
  --set postgresql.enabled=false \
  --set redis.enabled=false \
  --set global.postgresql.host=postgres.internal \
  --set global.redis.host=redis.internal
```

## 🔒 Security Best Practices

### 1. Use External Secrets

```bash
# Install external-secrets operator
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets \
  --namespace external-secrets-system \
  --create-namespace

# Create SecretStore for AWS Secrets Manager
kubectl apply -f - <<EOF
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: aws-secretsmanager
  namespace: jan-server
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-west-2
EOF
```

### 2. Enable Network Policies

```bash
# Create network policy to restrict pod communication
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: jan-server-netpol
  namespace: jan-server
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: jan-server
EOF
```

### 3. Pod Security Standards

```bash
# Label namespace with pod security standard
kubectl label namespace jan-server \
  pod-security.kubernetes.io/enforce=baseline \
  pod-security.kubernetes.io/audit=restricted \
  pod-security.kubernetes.io/warn=restricted
```

### 4. Enable TLS

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Create ClusterIssuer for Let's Encrypt
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: your-email@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

## 📊 Monitoring

### Prometheus & Grafana

```bash
# Add Prometheus stack
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace

# Access Grafana
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80
# Default: admin/prom-operator
```

### Logging

```bash
# Install Loki stack
helm repo add grafana https://grafana.github.io/helm-charts
helm install loki grafana/loki-stack \
  --namespace logging \
  --create-namespace \
  --set grafana.enabled=true
```

## 🔄 Upgrade & Maintenance

### Upgrade Helm Release

```bash
# Upgrade with new values
helm upgrade jan-server ./jan-server \
  --namespace jan-server \
  --values my-values.yaml

# Rollback if needed
helm rollback jan-server -n jan-server
```

### Database Migrations

```bash
# Run migrations manually before upgrade
kubectl run migration-job \
  --namespace jan-server \
  --image=jan/llm-api:latest \
  --restart=Never \
  --command -- /app/migrate
```

### Backup PostgreSQL

```bash
# Create backup
kubectl exec -n jan-server jan-server-postgresql-0 -- \
  pg_dump -U jan_user jan_llm_api > backup-$(date +%Y%m%d).sql

# Restore backup
kubectl exec -i -n jan-server jan-server-postgresql-0 -- \
  psql -U jan_user jan_llm_api < backup-20250109.sql
```

## 🐛 Troubleshooting

### Common Issues

#### Pods Not Starting

```bash
# Check events
kubectl describe pod -n jan-server <pod-name>

# Check logs
kubectl logs -n jan-server <pod-name> --previous
```

#### Database Connection Errors

```bash
# Verify PostgreSQL is running
kubectl get pods -n jan-server -l app.kubernetes.io/name=postgresql

# Test connection
kubectl run -n jan-server psql-test --rm -it \
  --image=postgres:18 \
  -- psql -h jan-server-postgresql -U jan_user -d jan_llm_api
```

#### Service Not Accessible

```bash
# Check service endpoints
kubectl get endpoints -n jan-server

# Test service internally
kubectl run -n jan-server curl-test --rm -it \
  --image=curlimages/curl \
  -- curl http://jan-server-llm-api:8080/healthz
```

## 📚 Additional Resources

- [Helm Documentation](https://helm.sh/docs/)
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
- [Jan Server Documentation](https://docs.jan.ai)
- [Bitnami Charts](https://github.com/bitnami/charts)

## 🤝 Support

For issues and questions:
- GitHub Issues: https://github.com/janhq/jan-server/issues
- Documentation: https://docs.jan.ai
- Community: https://discord.gg/jan

## 📄 License

See the main project LICENSE file.
