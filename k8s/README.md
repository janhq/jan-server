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

### Install Jan Server

```bash
# Install with default values (development)
helm install jan-server ./jan-server \
  --namespace jan-server \
  --create-namespace

# Install for production
helm install jan-server ./jan-server \
  --namespace jan-server \
  --create-namespace \
  --values ./jan-server/values-production.yaml
```

### Access Services

```bash
# Check deployment status
kubectl get pods -n jan-server

# Port forward to access services locally
kubectl port-forward -n jan-server svc/jan-server-kong 8000:8000
kubectl port-forward -n jan-server svc/jan-server-keycloak 8085:8085

# Access via Kong API Gateway
curl http://localhost:8000/api/llm/healthz
curl http://localhost:8000/api/media/healthz
curl http://localhost:8000/api/mcp/healthz

# Access Keycloak Admin Console
open http://localhost:8085
```

## 📦 Components

### Core Services

| Service | Port | Description |
|---------|------|-------------|
| LLM API | 8080 | Core LLM orchestration service |
| Media API | 8285 | Media upload and management |
| MCP Tools | 8091 | Model Context Protocol tools |
| Kong | 8000 | Unified API Gateway |
| Keycloak | 8085 | Authentication server |

### Supporting Services

| Service | Port | Description |
|---------|------|-------------|
| PostgreSQL | 5432 | Primary database |
| Redis | 6379 | Caching and sessions |
| SearXNG | 8080 | Meta search engine |
| Vector Store | 3015 | File search database |
| SandboxFusion | 8080 | Code interpreter |

## 🔧 Configuration

### Values Files

- `values.yaml` - Default values (suitable for development)
- `values-production.yaml` - Production configuration
- `values-development.yaml` - Minimal resource allocation

### Key Configuration Areas

#### 1. Database Credentials

```yaml
postgresql:
  auth:
    password: "CHANGE_ME"  # Change in production!
```

#### 2. S3 Storage (Media API)

```yaml
mediaApi:
  secrets:
    s3Endpoint: "https://s3.amazonaws.com"
    s3Bucket: "your-bucket"
    s3AccessKey: "YOUR_KEY"
    s3SecretKey: "YOUR_SECRET"
```

#### 3. Keycloak Admin

```yaml
keycloak:
  admin:
    username: admin
    password: "CHANGE_ME"  # Change in production!
```

#### 4. Resource Limits

```yaml
llmApi:
  resources:
    requests:
      memory: 512Mi
      cpu: 500m
    limits:
      memory: 1Gi
      cpu: 1000m
```

#### 5. Autoscaling

```yaml
llmApi:
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
```

#### 6. Ingress Configuration

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

### Development (Minikube/Kind)

```bash
# Start minikube with enough resources
minikube start --cpus=4 --memory=8192

# Install with development values
helm install jan-server ./jan-server \
  --namespace jan-server-dev \
  --create-namespace \
  --values ./jan-server/values-development.yaml

# Access via port-forward
kubectl port-forward -n jan-server-dev svc/jan-server-kong 8000:8000
```

### Cloud (AWS/GCP/Azure)

```bash
# Create production values
cat > my-values.yaml <<EOF
global:
  storageClass: "gp3"  # AWS EBS

kong:
  service:
    type: LoadBalancer
    annotations:
      service.beta.kubernetes.io/aws-load-balancer-type: "nlb"

llmApi:
  replicaCount: 3
  autoscaling:
    enabled: true
  ingress:
    enabled: true
    className: "nginx"
    hosts:
      - host: api.yourdomain.com
EOF

# Install
helm install jan-server ./jan-server \
  --namespace jan-server \
  --create-namespace \
  --values my-values.yaml
```

### On-Premises

```bash
# Use NodePort for external access
cat > on-prem-values.yaml <<EOF
kong:
  service:
    type: NodePort
    nodePort: 30000

postgresql:
  primary:
    persistence:
      storageClass: "local-storage"
EOF

helm install jan-server ./jan-server \
  --namespace jan-server \
  --create-namespace \
  --values on-prem-values.yaml
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
  --image=postgres:16 \
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
