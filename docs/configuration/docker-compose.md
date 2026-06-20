# Docker Compose Generation

## Overview

This document describes how docker-compose files are managed in relation to the configuration system.

## Current Approach

Instead of generating docker-compose files from YAML config, we maintain the compose files directly with references to the config system:

1. **Infrastructure** (`infra/docker/infrastructure.yml`) - PostgreSQL, Keycloak, Kong, shared networks/volumes.
2. **API Services** (`infra/docker/services-api.yml`) - llm-api, media-api, response-api.
3. **MCP Services** (`infra/docker/services-mcp.yml`) - mcp-tools, vector-store, sandbox helpers.
4. **Observability** (`infra/docker/observability.yml`) - Prometheus, Grafana, Jaeger, OTEL collector.
5. **Inference** (`infra/docker/inference.yml`) - vLLM GPU/CPU profiles.
6. **Development overlay** (`infra/docker/dev-full.yml`) - adds `host.docker.internal` mapping for hybrid workflows.

The root `docker-compose.yml` stitches the profiles together (infrastructure + services + MCP + observability). Profiles such as `full`, `mcp`, `monitor`, and `dev-full` map directly to the files above.

## Configuration Integration

Each service loads the **entire root `.env`** via Compose's `env_file` directive, then layers per-variable `${VAR:-default}` overrides on top. The `env_file` path is itself indirected through `ENV_FILE` so it always resolves to the single root `.env` unless explicitly overridden:

```yaml
services:
  llm-api:
    # Load every key/value from the root .env into the container
    env_file:
      - ${ENV_FILE:-../../.env}
    environment:
      # Database - constructed DSN from .env values
      DB_POSTGRESQL_WRITE_DSN: "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@api-db:5432/${POSTGRES_DB}?sslmode=disable"

      # Per-variable overrides fall back to defaults that mirror
      # config/defaults.yaml and packages/go-common/config/types.go
      HTTP_PORT: ${HTTP_PORT:-8080}
      LOG_LEVEL: ${LOG_LEVEL:-info}
```

The root `.env` is generated from `.env.template` by `make setup` / `make quickstart`. There is exactly one `.env` at the repository root---no `docker/.env` and no per-environment `config/<env>.env` files.

## Why Direct Maintenance?

1. **Simplicity** - Docker Compose is already declarative and easy to read
2. **Flexibility** - Allows docker-specific optimizations (healthchecks, extra hosts, bind mounts)
3. **Version Control** - Changes are clearly visible in git diffs
4. **No Generation Overhead** - No build step required
5. **Profiles** - Different dev/prod/monitoring stacks can be launched with a single `make` target

## Future: Optional Generation

If needed, a generator can be built using `packages/go-common/config/compose/` that:

- Reads `config/defaults.yaml`
- Applies environment overrides
- Generates docker-compose YAML files
- Validates output

## Validation

To validate compose files:

```bash
# Validate syntax
docker compose -f docker/infrastructure.yml config

# Validate with current environment
docker compose -f docker-compose.yml config

# Dry-run full stack
docker compose --profile full config

# Verify dev overlay
docker compose --profile dev-full config
```

Typical networks and volumes:

- Networks: `jan-server_default` (core), `jan-server_mcp-network` (MCP helpers)
- Volumes: `api-db-data`, `keycloak-db-data`, `vector-store-data`, `grafana-data`

All of the above are declared in the compose snippets so they can be inspected with `docker compose config`.

**Rationale:** Direct maintenance is simpler and more maintainable than generation for this use case. The generator infrastructure exists in `packages/go-common/config/compose/` if needed in the future.
