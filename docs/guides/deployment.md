# Deployment Guide

Deploy Jan Server to various environments using Docker Compose.

## Quick Start

```bash
# Clone and setup
git clone https://github.com/janhq/jan-server.git
cd jan-server
make quickstart
```

## Deployment Options

| Environment | Use Case | Recommended For |
|-------------|----------|-----------------|
| Docker Compose | Development, Testing | Local development |
| Hybrid Mode | Development | Fast iteration |

## Docker Compose Deployment

### Development Mode

```bash
# Start infrastructure only (PostgreSQL, Keycloak, Kong)
make up-infra

# With API services (llm-api, media-api, response-api)
make up-api

# With MCP services (mcp-tools, vector-store)
make up-mcp

# Full stack with Kong + APIs + MCP
make up-full
```

### Stop Services

```bash
make down           # Stop all (keeps volumes)
make down-clean     # Stop and remove volumes
```

## Hybrid Mode

For fast iteration during development:

```bash
make dev-full                 # Start stack with host routing

# Replace a service with a host-native process
./jan-cli.sh dev run llm-api  # macOS/Linux
.\jan-cli.ps1 dev run llm-api # Windows PowerShell

# Stop dev-full when done
make dev-full-stop            # Keep containers
make dev-full-down            # Remove containers
```

## Environment Configuration

### Required Environment Variables

#### LLM API

```bash
# Database
DB_POSTGRESQL_WRITE_DSN=postgres://jan_user:jan_password@localhost:5432/jan_llm_api?sslmode=disable

# Keycloak/Auth
KEYCLOAK_BASE_URL=http://localhost:8085
BACKEND_CLIENT_ID=llm-api
BACKEND_CLIENT_SECRET=your-secret

# Provider
VLLM_ENABLED=true
VLLM_PROVIDER_URL=http://localhost:8101/v1
REMOTE_LLM_ENABLED=false
```

#### Media API

```bash
# Database
DB_POSTGRESQL_WRITE_DSN=postgres://media:media@localhost:5432/media_api?sslmode=disable

# S3 Storage
MEDIA_S3_ENDPOINT=https://s3.amazonaws.com
MEDIA_S3_REGION=us-east-1
MEDIA_S3_BUCKET=your-bucket
MEDIA_S3_ACCESS_KEY_ID=your-access-key
MEDIA_S3_SECRET_ACCESS_KEY=your-secret-key
```

#### MCP Tools

```bash
# Server
HTTP_PORT=8091

# Optional providers
SERPER_API_KEY=your-serper-key
EXA_API_KEY=your-exa-key
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

## Security Checklist

- [ ] Use strong database passwords
- [ ] Configure proper API keys
- [ ] Enable TLS/HTTPS in production
- [ ] Set up monitoring
- [ ] Configure backups

## Monitoring

Enable monitoring stack:

```bash
make monitor-up
```

Access:
- Grafana: http://localhost:3331
- Prometheus: http://localhost:9090
- Jaeger: http://localhost:16686

## Related Documentation

- [Quickstart](../quickstart.md) - Getting started
- [Development Guide](development.md) - Local development
- [Monitoring Guide](monitoring.md) - Observability
- [Architecture Overview](../architecture/README.md)