# Deployment Guide

Comprehensive guide for deploying Jan Server to various environments.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Deployment Options](#deployment-options)
  - [Kubernetes (Recommended)](#kubernetes-recommended)
  - [Docker Compose](#docker-compose)
  - [Hybrid Mode](#hybrid-mode)
- [Environment Configuration](#environment-configuration)
- [Security Considerations](#security-considerations)
- [Monitoring and Observability](#monitoring-and-observability)

## Overview

Jan Server supports multiple deployment strategies to accommodate different use cases:

| Environment | Use Case | Orchestrator | Recommended For |
|-------------|----------|--------------|-----------------|
| **Kubernetes** | Production, Staging | Kubernetes/Helm | Scalable production deployments |
| **Docker Compose** | Development, Testing | Docker Compose | Local development and testing |
| **Hybrid Mode** | Development | Native + Docker | Fast iteration and debugging |

## Prerequisites

### All Deployments

- Docker 24+ and Docker Compose V2
- PostgreSQL 18+ (managed or in-cluster)
- Redis 7+ (managed or in-cluster)
- S3-compatible storage (for media-api)

### Kubernetes Deployments

- Kubernetes 1.27+
- Helm 3.12+
- kubectl configured
- Sufficient cluster resources (see [Resource Requirements](#resource-requirements))

## Deployment Options

### Kubernetes (Recommended)

Kubernetes deployment uses Helm charts for full orchestration and scalability.

#### 1. Development (Minikube)

For local development and testing:

```bash
# Prerequisites
minikube start --cpus=4 --memory=8192 --driver=docker

# Build and load images
cd services/llm-api && go mod tidy && cd ../..
cd services/media-api && go mod tidy && cd ../..
cd services/mcp-tools && go mod tidy && cd ../..

docker build -t jan/llm-api:latest -f services/llm-api/Dockerfile .
docker build -t jan/media-api:latest -f services/media-api/Dockerfile .
docker build -t jan/mcp-tools:latest -f services/mcp-tools/Dockerfile .
docker build -t jan/keycloak:latest -f keycloak/Dockerfile keycloak

# Load images into minikube
minikube image load jan/llm-api:latest jan/media-api:latest jan/mcp-tools:latest jan/keycloak:latest
minikube image load bitnami/postgresql:latest bitnami/redis:latest

# Deploy
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

# Verify deployment
kubectl get pods -n jan-server

# Access services
kubectl port-forward -n jan-server svc/jan-server-llm-api 8080:8080
curl http://localhost:8080/healthz
```

**Complete guide:** See [k8s/SETUP.md](../../k8s/SETUP.md)

#### 2. Cloud Kubernetes (AKS/EKS/GKE)

For production cloud deployments:

```bash
# Option A: With cloud-managed databases (recommended)
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
  --set llmApi.replicaCount=3 \
  --set llmApi.image.pullPolicy=Always \
  --set mediaApi.image.pullPolicy=Always \
  --set mcpTools.image.pullPolicy=Always

# Option B: With in-cluster databases
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

**Configuration guide:** See [k8s/README.md](../../k8s/README.md)

#### 3. On-Premises Kubernetes

For on-premises production:

```bash
# Use production values with external databases
helm install jan-server ./jan-server \
  --namespace jan-server \
  --create-namespace \
  --values ./jan-server/values-production.yaml \
  --set postgresql.enabled=false \
  --set redis.enabled=false \
  --set global.postgresql.host=postgres.internal \
  --set global.redis.host=redis.internal
```

### Docker Compose

For local development and testing environments.

#### Development Mode

```bash
# Start infrastructure only
make up

# With LLM API
make up-llm-api

# Full stack with Kong
make up-full

# With GPU inference
make up-gpu
```

**Complete guide:** See [Development Guide](development.md)

#### Testing Environment

```bash
# Load testing configuration
cp config/testing.env .env
source .env

# Start services
docker compose up -d

# Run tests
make test-automation
```

### Hybrid Mode

For fast iteration during development:

```bash
# Start infrastructure (PostgreSQL, Redis, Keycloak)
docker compose --profile infrastructure up -d

# Run LLM API natively
./scripts/hybrid-run-api.sh  # or .ps1 on Windows

# Run Media API natively
./scripts/hybrid-run-media-api.sh

# Run MCP Tools natively
./scripts/hybrid-run-mcp.sh
```

**Complete guide:** See [Hybrid Mode Guide](hybrid-mode.md)

## Environment Configuration

### Required Environment Variables

#### LLM API

```bash
# Database
DATABASE_URL=postgres://jan_user:jan_password@localhost:5432/jan_llm_api?sslmode=disable

# Keycloak/Auth
KEYCLOAK_BASE_URL=http://localhost:8085
BACKEND_CLIENT_ID=llm-api
BACKEND_CLIENT_SECRET=your-secret
TARGET_CLIENT_ID=jan-client

# Optional
JAN_DEFAULT_NODE_SETUP=false  # Disable if no Jan provider
HTTP_PORT=8080
LOG_LEVEL=debug
```

#### Media API

```bash
# Database
DATABASE_URL=postgres://media:media@localhost:5432/media_api?sslmode=disable

# S3 Storage (Required)
S3_ENDPOINT=https://s3.amazonaws.com
S3_BUCKET=your-bucket
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key

# Server
HTTP_PORT=8081
LOG_LEVEL=info
```

#### MCP Tools

```bash
# Server
HTTP_PORT=8091
LOG_LEVEL=info

# Optional providers
EXA_API_KEY=your-exa-key
BRAVE_API_KEY=your-brave-key
```

### Configuration Files

Environment-specific configuration files in `config/`:

- `defaults.env` - Default values for all environments
- `development.env` - Local development settings
- `testing.env` - Test environment settings
- `production.env.example` - Production template (copy and customize)
- `secrets.env.example` - Secrets template (never commit actual secrets)

## Security Considerations

### Production Checklist

- [ ] **Secrets Management**
  - Use external secrets operator (e.g., AWS Secrets Manager, Azure Key Vault)
  - Never commit secrets to version control
  - Rotate credentials regularly

- [ ] **Network Security**
  - Enable network policies to restrict pod-to-pod communication
  - Use TLS for all external endpoints
  - Configure ingress with proper SSL certificates

- [ ] **Authentication**
  - Change default Keycloak admin password
  - Configure proper realm settings
  - Enable token exchange for client-to-client auth

- [ ] **Database Security**
  - Use managed database services when possible
  - Enable SSL/TLS connections
  - Implement backup and disaster recovery

- [ ] **Pod Security**
  - Apply pod security standards (restricted profile)
  - Use non-root containers
  - Enable security context constraints

### Example: External Secrets

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

## Resource Requirements

### Minimum (Development)

| Component | CPU | Memory |
|-----------|-----|--------|
| LLM API | 250m | 256Mi |
| Media API | 250m | 256Mi |
| MCP Tools | 250m | 256Mi |
| PostgreSQL | 250m | 256Mi |
| Redis | 100m | 128Mi |
| Keycloak | 500m | 512Mi |
| **Total** | **~1.5 CPU** | **~2Gi** |

### Recommended (Production)

| Component | CPU | Memory | Replicas |
|-----------|-----|--------|----------|
| LLM API | 1000m | 1Gi | 3 |
| Media API | 500m | 512Mi | 2 |
| MCP Tools | 500m | 512Mi | 2 |
| PostgreSQL | 2000m | 4Gi | 1 (or managed) |
| Redis | 500m | 1Gi | 3 (cluster) |
| Keycloak | 1000m | 1Gi | 2 |

### Storage Requirements

- PostgreSQL: 50Gi minimum (100Gi+ for production)
- Redis: 10Gi for persistence
- PVCs for media uploads (if not using S3)

## Monitoring and Observability

### Enable Monitoring Stack

```bash
# Start monitoring services
docker compose --profile monitoring up -d

# Access dashboards
# Prometheus: http://localhost:9090
# Grafana: http://localhost:3001
# Jaeger: http://localhost:16686
```

### Key Metrics to Monitor

- **Service Health**: Endpoint availability, response times
- **Database**: Connection pool usage, query performance
- **Resource Usage**: CPU, memory, disk I/O
- **Request Rates**: Throughput, error rates
- **Authentication**: Token issuance, validation failures

**Complete guide:** See [Monitoring Guide](monitoring.md)

## Troubleshooting

### Common Issues

#### Pods Not Starting

```bash
# Check pod status
kubectl get pods -n jan-server

# View pod logs
kubectl logs -n jan-server <pod-name>

# Describe pod for events
kubectl describe pod -n jan-server <pod-name>
```

#### Database Connection Failures

```bash
# Verify PostgreSQL is running
kubectl exec -n jan-server jan-server-postgresql-0 -- psql -U postgres -c '\l'

# Check database exists
kubectl exec -n jan-server jan-server-postgresql-0 -- psql -U postgres -c '\l' | grep media_api

# Test connection from service pod
kubectl exec -n jan-server <service-pod> -- nc -zv jan-server-postgresql 5432
```

#### Image Pull Failures

For minikube:
```bash
# Verify images are loaded
minikube image ls | grep jan/

# Reload if missing
minikube image load jan/llm-api:latest
```

For production:
```bash
# Check image pull policy
kubectl get deployment -n jan-server jan-server-llm-api -o yaml | grep pullPolicy

# Should be "Always" or "IfNotPresent" for registry images
```

## Related Documentation

- [Kubernetes Setup Guide](../../k8s/SETUP.md) - Complete k8s deployment steps
- [Kubernetes Configuration](../../k8s/README.md) - Helm chart configuration reference
- [Development Guide](development.md) - Local development setup
- [Hybrid Mode](hybrid-mode.md) - Native service execution
- [Monitoring Guide](monitoring.md) - Observability setup
- [Architecture Overview](../architecture/README.md) - System architecture

## Support

For additional help:
- Review [Getting Started](../getting-started/README.md)
- Check [Troubleshooting Guide](troubleshooting.md)
- See [Architecture Documentation](../architecture/README.md)
