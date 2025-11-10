# Kubernetes Setup and Deployment Guide

This guide walks through setting up a Kubernetes cluster and deploying Jan Server using Helm.

## Prerequisites

- Docker Desktop OR minikube
- kubectl CLI
- Helm 3.8+
- Go 1.23+ (for building services)

## Services Overview

Jan Server includes the following microservices:

- **LLM API** (port 8080) - Core LLM orchestration service
- **Media API** (port 8285) - Media upload and management
- **Response API** (port 8082) - Response generation with tool orchestration
- **MCP Tools** (port 8091) - Model Context Protocol tools integration
- **Keycloak** (port 8085) - Authentication server
- **Kong** (port 8000) - API Gateway
- **PostgreSQL** (port 5432) - Database (shared by all services)
- **Redis** (port 6379) - Caching
- **SearXNG** (port 8080) - Meta search engine
- **SandboxFusion** (port 8080) - Code interpreter

## Option 1: Minikube (Recommended for Development)

### Install Minikube

```powershell
# Using Chocolatey
choco install minikube

# Or download from: https://minikube.sigs.k8s.io/docs/start/
```

### Start Minikube

```powershell
# Start with sufficient resources
minikube start --cpus=4 --memory=8192 --driver=docker
```

### Verify Installation

```powershell
kubectl cluster-info
kubectl get nodes
minikube status
```

## Option 2: Docker Desktop Kubernetes

### Enable Kubernetes in Docker Desktop

1. Open Docker Desktop
2. Go to Settings → Kubernetes
3. Check "Enable Kubernetes"
4. Click "Apply & Restart"
5. Wait for Kubernetes to start (green indicator)

### Verify Installation

```powershell
kubectl cluster-info
kubectl get nodes
```

You should see:
```
Kubernetes control plane is running at https://kubernetes.docker.internal:6443
```

## Building and Deploying Jan Server

### Step 1: Fix Go Modules

```powershell
cd d:\Working\Menlo\jan-server

# Fix go modules for all services
cd services\llm-api
go mod tidy

cd ..\media-api
go mod tidy

cd ..\mcp-tools
go mod tidy

cd ..\..
```

### Step 2: Build Docker Images

```powershell
cd d:\Working\Menlo\jan-server

# Build LLM API
cd services\llm-api
docker build -t jan/llm-api:latest .

# Build Media API
cd ..\media-api
docker build -t jan/media-api:latest .

# Build Response API
cd ..\response-api
docker build -t jan/response-api:latest .

# Build MCP Tools
cd ..\mcp-tools
docker build -t jan/mcp-tools:latest .

# Build Keycloak
cd ..\..\keycloak
docker build -t jan/keycloak:latest .

cd ..
```

### Step 3: Load Images into Minikube (Minikube Only)

If using minikube, load the images into the cluster:

```powershell
# Load custom images
minikube image load jan/llm-api:latest
minikube image load jan/media-api:latest
minikube image load jan/response-api:latest
minikube image load jan/mcp-tools:latest
minikube image load jan/keycloak:latest

# Pull and load Bitnami images
docker pull bitnami/postgresql:latest
docker pull bitnami/redis:latest
minikube image load bitnami/postgresql:latest
minikube image load bitnami/redis:latest

# Verify images are loaded
minikube image ls | Select-String "jan/|bitnami"
```

### Step 4: Add Bitnami Repository

```powershell
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
```

### Step 5: Build Chart Dependencies

```powershell
cd d:\Working\Menlo\jan-server\k8s\jan-server
helm dependency build
```

This downloads PostgreSQL and Redis charts from Bitnami.

### Step 6: Install Jan Server

```powershell
cd d:\Working\Menlo\jan-server\k8s

# Install with default values
helm install jan-server ./jan-server `
  --namespace jan-server `
  --create-namespace
```

### Step 7: Create Additional Databases

PostgreSQL needs additional databases for media-api and keycloak:

```powershell
# Wait for PostgreSQL to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=postgresql -n jan-server --timeout=300s

# Create databases
kubectl exec -n jan-server jan-server-postgresql-0 -- bash -c "PGPASSWORD=postgres psql -U postgres << 'EOF'
CREATE USER media WITH PASSWORD 'media';
CREATE DATABASE media_api OWNER media;
CREATE USER keycloak WITH PASSWORD 'keycloak';
CREATE DATABASE keycloak OWNER keycloak;
EOF"
```

### Step 8: Verify Deployment

```powershell
# Check all resources
kubectl get all -n jan-server

# Check pods status
kubectl get pods -n jan-server

# Check services
kubectl get svc -n jan-server
```

Wait until all pods show `Running` status (this may take 2-5 minutes):
```
NAME                                    READY   STATUS    RESTARTS   AGE
jan-server-keycloak-xxx                 1/1     Running   0          3m
jan-server-kong-xxx                     1/1     Running   0          3m
jan-server-llm-api-xxx                  1/1     Running   0          3m
jan-server-media-api-xxx                1/1     Running   0          3m
jan-server-response-api-xxx             1/1     Running   0          3m
jan-server-mcp-tools-xxx                1/1     Running   0          3m
jan-server-postgresql-0                 1/1     Running   0          3m
jan-server-redis-master-0               1/1     Running   0          3m
jan-server-redis-replicas-0             1/1     Running   0          3m
jan-server-searxng-xxx                  1/1     Running   0          3m
jan-server-sandboxfusion-xxx            1/1     Running   0          3m
```

**Note:** `vector-store` is disabled by default. Kong may show restarts due to low memory limits (this is expected).

### Step 9: Access Services via Port-Forward

Open multiple PowerShell terminals and run:

```powershell
# Terminal 1: Kong API Gateway (Main entry point)
kubectl port-forward -n jan-server svc/jan-server-kong 8000:8000

# Terminal 2: Keycloak Authentication
kubectl port-forward -n jan-server svc/jan-server-keycloak 8085:8085

# Optional: Direct service access
kubectl port-forward -n jan-server svc/jan-server-llm-api 8080:8080
kubectl port-forward -n jan-server svc/jan-server-media-api 8285:8285
kubectl port-forward -n jan-server svc/jan-server-response-api 8082:8082
kubectl port-forward -n jan-server svc/jan-server-mcp-tools 8091:8091
```

### Step 10: Test API Endpoints

```powershell
# Test via Kong API Gateway
curl http://localhost:8000/api/llm/healthz
curl http://localhost:8000/api/media/healthz
curl http://localhost:8000/api/responses/healthz
curl http://localhost:8000/api/mcp/healthz

# Or test direct service access
curl http://localhost:8080/healthz  # LLM API
curl http://localhost:8285/healthz  # Media API
curl http://localhost:8082/healthz  # Response API
curl http://localhost:8091/healthz  # MCP Tools

# Access Keycloak Admin
# Open browser: http://localhost:8085
# Username: admin
# Password: changeme
```

## Common Commands

### View Logs

```powershell
# View logs for a specific pod
kubectl logs -n jan-server <pod-name>

# Follow logs
kubectl logs -n jan-server <pod-name> -f

# View logs from all containers in a pod
kubectl logs -n jan-server <pod-name> --all-containers
```

### Describe Resources

```powershell
# Describe a pod (shows events and status)
kubectl describe pod -n jan-server <pod-name>

# Describe a service
kubectl describe svc -n jan-server jan-server-kong
```

### Execute Commands in Pods

```powershell
# Connect to PostgreSQL
kubectl exec -it -n jan-server jan-server-postgresql-0 -- psql -U jan_user -d jan_llm_api

# Shell into a pod
kubectl exec -it -n jan-server <pod-name> -- /bin/sh
```

### Restart a Deployment

```powershell
kubectl rollout restart deployment -n jan-server jan-server-llm-api
```

## Upgrade Deployment

```powershell
# Upgrade with new values
helm upgrade jan-server ./jan-server `
  --namespace jan-server `
  --wait `
  --timeout 10m

# Upgrade with custom values
helm upgrade jan-server ./jan-server `
  --namespace jan-server `
  --values ./jan-server/values-development.yaml `
  --wait
```

## Uninstall

```powershell
# Uninstall the release
helm uninstall jan-server -n jan-server

# Delete the namespace (including PVCs)
kubectl delete namespace jan-server
```

## Troubleshooting

### Pods in CrashLoopBackOff

```powershell
# Check pod logs
kubectl logs -n jan-server <pod-name> --previous

# Check pod events
kubectl describe pod -n jan-server <pod-name>
```

### ImagePullBackOff Error

This means Docker images are not available. You need to:

1. Build the Docker images locally:
   ```powershell
   cd d:\Working\Menlo\jan-server
   docker compose build
   ```

2. Update values.yaml to use local images or configure image pull policy:
   ```yaml
   llmApi:
     image:
       registry: ""
       repository: jan/llm-api
       tag: latest
       pullPolicy: IfNotPresent
   ```

### PostgreSQL Not Starting

```powershell
# Check PostgreSQL logs
kubectl logs -n jan-server jan-server-postgresql-0

# Check PVC
kubectl get pvc -n jan-server
```

### Services Not Accessible

```powershell
# Check if service has endpoints
kubectl get endpoints -n jan-server

# Test internal connectivity
kubectl run -n jan-server curl-test --rm -it --image=curlimages/curl -- curl http://jan-server-llm-api:8080/healthz
```

## Next Steps

Once the infrastructure is deployed and all pods are running:

1. Run the port-forward script (use the helper in `k8s/port-forward.ps1`)
2. Configure Keycloak realm and clients
3. Run automation tests from `tests/automation/`

## For Minikube Users

### Access LoadBalancer Services

Minikube doesn't support LoadBalancer type services by default. Use one of these methods:

```powershell
# Method 1: Use minikube tunnel (requires admin privileges)
minikube tunnel

# Method 2: Change Kong service to NodePort in values.yaml
kong:
  service:
    type: NodePort
    nodePort: 30000

# Then access via: http://$(minikube ip):30000
```

### Enable Metrics Server

```powershell
minikube addons enable metrics-server
```

## Production Deployment

For production deployment, create a custom values file:

```yaml
# my-production-values.yaml
postgresql:
  auth:
    password: "STRONG_PASSWORD_HERE"
    postgresPassword: "STRONG_POSTGRES_PASSWORD"

keycloak:
  admin:
    password: "STRONG_ADMIN_PASSWORD"
  database:
    password: "STRONG_DB_PASSWORD"

mediaApi:
  secrets:
    serviceKey: "YOUR_SERVICE_KEY"
    apiKey: "YOUR_API_KEY"
    s3Endpoint: "https://your-s3-endpoint.com"
    s3Bucket: "your-bucket"
    s3AccessKey: "YOUR_ACCESS_KEY"
    s3SecretKey: "YOUR_SECRET_KEY"

responseApi:
  secrets:
    databaseUrl: "postgres://jan_user:STRONG_PASSWORD@jan-server-postgresql:5432/jan_llm_api?sslmode=disable"
  env:
    maxToolExecutionDepth: "5"
    toolExecutionTimeout: "30s"

kong:
  service:
    type: LoadBalancer

llmApi:
  replicaCount: 3
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10

responseApi:
  replicaCount: 2
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 8

mediaApi:
  replicaCount: 2
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 6
```

Deploy with:
```powershell
helm install jan-server ./jan-server `
  --namespace jan-server `
  --create-namespace `
  --values my-production-values.yaml `
  --wait `
  --timeout 15m
```
