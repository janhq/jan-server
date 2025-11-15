# Docker Compose Generation

## Overview

This document describes how docker-compose files are managed in relation to the configuration system.

## Current Approach

Instead of generating docker-compose files from YAML config, we maintain the compose files directly with references to the config system:

1. **Infrastructure** (`docker/infrastructure.yml`) - Defines PostgreSQL, Keycloak, Kong
2. **Services** (`docker/services-api.yml`) - Defines llm-api, media-api, response-api
3. **MCP Tools** (`docker/services-mcp.yml`) - Defines mcp-tools, vector-store

## Configuration Integration

Each service in docker-compose references the standardized environment variables:

```yaml
services:
  llm-api:
    environment:
      # Database - constructed DSN from config defaults
      DB_POSTGRESQL_WRITE_DSN: "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@api-db:5432/${POSTGRES_DB}?sslmode=disable"
      
      # All other variables reference config/defaults.yaml structure
      HTTP_PORT: ${HTTP_PORT:-8080}
      LOG_LEVEL: ${LOG_LEVEL:-info}
```

## Why Direct Maintenance?

1. **Simplicity** - Docker Compose is already declarative and easy to read
2. **Flexibility** - Allows docker-specific optimizations (healthchecks, networks, volumes)
3. **Version Control** - Changes are clearly visible in git diffs
4. **No Generation Overhead** - No build step required

## Future: Optional Generation

If needed, a generator can be built using `pkg/config/compose/generator.go` that:
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
```

## Sprint 4 Status

✅ **COMPLETE** - Docker compose files are consistent with config system
✅ **COMPLETE** - All services use standardized environment variables  
✅ **COMPLETE** - Validation process documented

**Rationale:** Direct maintenance is simpler and more maintainable than generation for this use case. The generator infrastructure exists in `pkg/config/compose/` if needed in the future.
