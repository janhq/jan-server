# Deployment Guide

How Jan Server is built, released, and run. The automated pipeline is **build-and-push only**: it produces container images and a deployed web app, but it does **not** roll those images out to any cluster. Promotion to a running environment is a manual/operator step.

## Release Pipeline (GitHub Actions)

### Production release

Pushing a semver tag (`vX.Y.Z`) triggers `.github/workflows/ci-production-release.yml`, which:

- Builds the four backend service images and pushes them to the Menlo registry:
  - `registry.menlo.ai/jan-server/llm-api:prod-<tag>`
  - `registry.menlo.ai/jan-server/mcp-tools:prod-<tag>`
  - `registry.menlo.ai/jan-server/media-api:prod-<tag>`
  - `registry.menlo.ai/jan-server/response-api:prod-<tag>`
- Builds the web app (`pnpm build:web`) and publishes it to **Cloudflare Pages** (project `jan-server-web`).

There is **no auto-deploy / GitOps step** for the backend. After the images are pushed, updating a running cluster (image tags, manifests, rollout) is done manually by an operator.

### Development builds

Pushes to development branches trigger `ci-backend-dev.yml` and `ci-app-web-dev.yml`, which build and push `:dev-<sha>` images to `registry.menlo.ai/jan-server/*`. These are also build-and-push only.

## Kubernetes / Helm

> **Status: manual and incomplete.** Helm charts and K8s manifests exist in the repo, but there is no automated deployment from CI and the charts are not a turnkey/supported install path. Treat any Kubernetes deployment as operator-driven and expect to fill in gaps.

You can render a starter Helm values file from the canonical config:

```bash
jan-cli config k8s-values
```

Apply and roll out manually (example):

```bash
kubectl rollout restart deployment/llm-api -n jan
kubectl rollout restart deployment/mcp-tools -n jan
```

## Local Docker Compose

For development and testing on a single machine, run the stack with Docker Compose.

### Quick Start

```bash
git clone https://github.com/janhq/jan-server.git
cd jan-server
make quickstart
```

### Start / stop services

```bash
# Start infrastructure only (PostgreSQL, Keycloak, Kong)
make up-infra

# With API services (llm-api, media-api, response-api)
make up-api

# With MCP services (mcp-tools, vector-store)
make up-mcp

# Full stack with Kong + APIs + MCP
make up-full

make down           # Stop all (keeps volumes)
make down-clean     # Stop and remove volumes
```

### Hybrid (dev-full) mode

For fast iteration you can run infrastructure in Docker and a single service natively on the host. That workflow is documented in the [Development Guide](development.md#dev-full-mode-hybrid-debugging) — see `make dev-full` and `jan-cli dev run <service>`.

## Environment Configuration

A single root `.env` drives both Compose and the services (created by `make setup` / `make quickstart`). Below are the key variables per service; see `.env.template` and the [Configuration docs](../configuration/README.md) for the full list.

### LLM API

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

### Media API

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

### MCP Tools

```bash
# Server
HTTP_PORT=8091

# Optional providers
SERPER_API_KEY=your-serper-key
EXA_API_KEY=your-exa-key
```

## Resource Requirements

Minimum guidance for a local/development deployment:

| Component  | CPU  | Memory |
|------------|------|--------|
| LLM API    | 250m | 256Mi  |
| Media API  | 250m | 256Mi  |
| MCP Tools  | 250m | 256Mi  |
| PostgreSQL | 250m | 256Mi  |
| Keycloak   | 500m | 512Mi  |

## Security Checklist

- [ ] Use strong database passwords
- [ ] Configure proper API keys
- [ ] Enable TLS/HTTPS in production
- [ ] Set up monitoring
- [ ] Configure backups

## Monitoring

Enable the observability stack:

```bash
make monitor-up
```

Access:
- Grafana: http://localhost:3331
- Prometheus: http://localhost:9090
- Jaeger: http://localhost:16686

See the [Monitoring Guide](monitoring.md) for details.

## Related Documentation

- [Quickstart](../quickstart.md) - Getting started
- [Development Guide](development.md) - Local development and dev-full mode
- [Monitoring Guide](monitoring.md) - Observability
- [Architecture Overview](../architecture/README.md)
