# Troubleshooting Guide

Solutions to common issues when developing and deploying Jan Server.

## Table of Contents

1. [Service Startup Issues](#service-startup-issues)
2. [Database Issues](#database-issues)
3. [API Issues](#api-issues)
4. [Authentication Issues](#authentication-issues)
5. [Docker Issues](#docker-issues)
6. [Kubernetes Issues](#kubernetes-issues)
7. [Performance Issues](#performance-issues)
8. [Getting Help](#getting-help)

## Service Startup Issues

### Port Already in Use

**Error**: `Address already in use` or port conflict

**Solutions**:
```bash
# Find what's using the port (Linux/macOS)
lsof -i :8080
lsof -i :8082
lsof -i :8285
lsof -i :8091

# Find what's using the port (Windows)
netstat -ano | findstr :8080
taskkill /PID <PID> /F

# Or change ports in .env
HTTP_PORT=8081
RESPONSE_API_PORT=8083
MEDIA_API_PORT=8286
```

### Service Fails to Start

**Error**: Container exits immediately after starting

**Debug Steps**:
```bash
# View logs
make logs-llm-api
make logs-response-api
make logs-media-api
make logs-mcp

# Or with docker directly
docker logs <container-id>
```

**Common Causes**:
- Missing environment variables
- Database not ready
- Incorrect configuration

**Fix**:
```bash
# Verify all required env vars are set
cat .env | grep -E "HTTP_PORT|DATABASE|API_KEY"

# Wait for database to be ready
make health-check

# Restart service
docker restart <container-name>
```

### Services Won't Connect

**Error**: `connection refused` or `cannot reach service`

**Solutions**:
```bash
# Verify all services are running
docker ps

# Check network connectivity
docker network ls
docker network inspect jan-server_default

# Test connectivity between services
docker exec llm-api curl http://media-api:8285/healthz

# Verify DNS resolution
docker exec llm-api nslookup media-api
```

## Database Issues

### Database Connection Failed

**Error**: `dial tcp: connect: connection refused` or `database connection error`

**Solutions**:
```bash
# Check if PostgreSQL is running
docker ps | grep postgres

# Check database credentials in .env
cat .env | grep DATABASE

# Test connection
docker exec api-db psql -U jan_user -d jan_llm_api -c "SELECT 1"

# Verify database exists
docker exec api-db psql -U postgres -l | grep jan
```

### Missing Database

**Error**: `database "jan_llm_api" does not exist`

**Solutions**:
```bash
# Create database
docker exec api-db psql -U postgres -c "CREATE DATABASE jan_llm_api"

# Or run migrations
docker exec llm-api /app/llm-api migrate

# Or with make
make db-migrate
```

### Table Migration Failed

**Error**: Migration errors or schema mismatch

**Solutions**:
```bash
# View migrations
docker exec api-db psql -U jan_user -d jan_llm_api \
  -c "SELECT name, version FROM schema_migrations"

# Reset database (WARNING: destroys data!)
make db-reset

# Re-migrate
make db-migrate
```

### Database Disk Full

**Error**: `No space left on device`

**Solutions**:
```bash
# Check disk usage
df -h

# Clean up Docker volumes
docker system prune -a --volumes

# Or manually remove volume
docker volume ls
docker volume rm <volume-name>
```

## API Issues

### 401 Unauthorized

**Error**: All requests return 401

**Solutions**:
```bash
# Get a guest token
curl -X POST http://localhost:8000/llm/auth/guest-login

# Use token in requests
curl -H "Authorization: Bearer <token>" \
  http://localhost:8000/v1/models

# Check Keycloak is running
docker ps | grep keycloak

# Verify token is valid
jwt decode <token>  # requires jwt-cli
```

### 404 Not Found

**Error**: Endpoints return 404

**Check**:
```bash
# Verify service is running and healthy
curl http://localhost:8080/healthz      # LLM API
curl http://localhost:8082/healthz      # Response API
curl http://localhost:8285/healthz      # Media API
curl http://localhost:8091/healthz      # MCP Tools

# Verify Kong is routing correctly
curl http://localhost:8000/                # Kong health
curl http://localhost:8000/v1/models       # Via Kong
```

### Timeout Errors

**Error**: `408 Request Timeout` or connection hangs

**Solutions**:
```bash
# Increase timeout in .env
TOOL_EXECUTION_TIMEOUT=120s
MEDIA_REMOTE_FETCH_TIMEOUT=30s

# Check service performance
docker stats llm-api media-api response-api mcp-tools

# Look for stuck processes
make logs-llm-api | grep -i timeout
```

### 500 Internal Server Error

**Error**: Unexpected server error

**Debug**:
```bash
# View detailed logs
docker logs <service-name> --tail=100 -f

# Check if service crashed
docker inspect <service-name>

# Restart service
docker restart <service-name>

# Or full restart
make down && make up-full
```

## Authentication Issues

### Keycloak Not Responding

**Error**: `Failed to connect to Keycloak` or auth endpoints fail

**Solutions**:
```bash
# Check if Keycloak is running
docker ps | grep keycloak

# Verify it's accessible
curl http://localhost:8085/admin

# Check logs
docker logs keycloak

# Restart Keycloak
docker restart keycloak
```

### Invalid JWT Token

**Error**: Token is expired or invalid

**Solutions**:
```bash
# Get new token
curl -X POST http://localhost:8000/llm/auth/guest-login

# Check token expiration
jwt decode <token>

# For user auth, check credentials
curl -X POST http://localhost:8085/auth/realms/jan/protocol/openid-connect/token \
  -d "client_id=llm-api&grant_type=password&username=admin&password=admin"
```

## Docker Issues

### Out of Memory

**Error**: `OOMKilled` or memory errors

**Solutions**:
```bash
# Check memory usage
docker stats

# Increase Docker memory limit
# In Docker Desktop: Settings > Resources > Memory

# Or reduce services running
make down
```

### Disk Space Low

**Error**: `no space left on device`

**Solutions**:
```bash
# Clean up unused images and volumes
docker system prune -a --volumes

# Remove old containers
docker container prune

# Check image sizes
docker images --format "table {{.Repository}}\t{{.Size}}"
```

### Network Issues

**Error**: Services can't communicate

**Solutions**:
```bash
# Verify network exists
docker network ls

# Check network configuration
docker network inspect jan-server_default

# Recreate network if needed
docker network rm jan-server_default
docker network create jan-server_default
```

## Kubernetes Issues

### Pod Stuck in Pending

**Error**: Pod stays in Pending state

**Debug**:
```bash
# Check events
kubectl describe pod -n jan-server <pod-name>

# Check node resources
kubectl top nodes

# Check available storage
kubectl get pvc -n jan-server
```

### ImagePullBackOff

**Error**: Can't pull image

**Solutions**:
```bash
# Verify image exists
minikube image ls | grep jan

# Rebuild image
cd services/llm-api
docker build -t jan/llm-api:latest .
minikube image load jan/llm-api:latest

# Or update imagePullPolicy in values.yaml
imagePullPolicy: Never  # For minikube
```

### Service Not Accessible

**Error**: Service endpoints not working

**Debug**:
```bash
# Check service exists
kubectl get svc -n jan-server

# Port forward for access
kubectl port-forward -n jan-server svc/jan-server-llm-api 8080:8080

# Check service endpoints
kubectl get endpoints -n jan-server
```

## Performance Issues

### High Memory Usage

**Symptoms**: Services use lots of memory

**Solutions**:
```bash
# Monitor memory
docker stats llm-api

# Reduce batch size or concurrency
# Check service configuration in docker compose

# Look for memory leaks in logs
docker logs llm-api | grep -i memory
```

### High CPU Usage

**Symptoms**: CPU usage maxed out

**Solutions**:
```bash
# Monitor CPU
docker stats

# Reduce concurrent requests
# Set rate limits in configuration

# Check for infinite loops or busy waits
docker logs llm-api | grep -i error
```

### Slow Responses

**Symptoms**: API requests are slow

**Solutions**:
```bash
# Check database performance
docker exec api-db psql -U jan_user -d jan_llm_api \
  -c "SELECT query, calls, total_time FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10"

# Enable query logging
# Set LOG_LEVEL=debug in .env

# Use monitoring stack for traces
make monitor-up
# Visit http://localhost:16686 (Jaeger)
```

## Getting Help

### Gathering Debug Information

Before asking for help, collect this information:

```bash
# System info
docker version
docker-compose version
go version
kubectl version

# Service status
make health-check

# Logs from all services
make logs > debug-logs.txt

# Configuration (without secrets)
cat .env | grep -v _KEY | grep -v _PASSWORD > config.txt

# Docker system status
docker system df
docker ps -a
```

### Requesting Support

When reporting issues, include:
1. Error messages and logs
2. Steps to reproduce
3. Your environment (OS, Docker version, etc.)
4. Configuration (sanitized)
5. What you've already tried

**Resources**:
- [GitHub Issues](https://github.com/janhq/jan-server/issues)
- [Discussions](https://github.com/janhq/jan-server/discussions)
- [Architecture Documentation](../architecture/)
- [Development Guide](./development.md)
