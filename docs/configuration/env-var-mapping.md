# Environment Variable Mapping

This document maps centralized configuration (`pkg/config/types.go`) environment variables to service-specific variables, facilitating the Sprint 3 migration.

## Infrastructure

### Database (PostgreSQL)

| Centralized Env Var        | Type     | Default        | Services Using | Notes                                           |
| -------------------------- | -------- | -------------- | -------------- | ----------------------------------------------- |
| `POSTGRES_HOST`            | string   | `api-db`       | llm-api        | Used in `DB_POSTGRESQL_WRITE_DSN`               |
| `POSTGRES_PORT`            | int      | `5432`         | llm-api        | Used in `DB_POSTGRESQL_WRITE_DSN`               |
| `POSTGRES_USER`            | string   | `jan_user`     | llm-api        | Used in `DB_POSTGRESQL_WRITE_DSN`               |
| `POSTGRES_PASSWORD`        | string   | `jan_password` | llm-api        | From secrets, used in `DB_POSTGRESQL_WRITE_DSN` |
| `POSTGRES_DB`              | string   | `jan_llm_api`  | llm-api        | Used in `DB_POSTGRESQL_WRITE_DSN`               |
| `POSTGRES_SSL_MODE`        | string   | `disable`      | llm-api        | Used in `DB_POSTGRESQL_WRITE_DSN`               |
| `POSTGRES_MAX_CONNECTIONS` | int      | `100`          | llm-api        | New standardized var                            |
| `POSTGRES_MAX_IDLE_CONNS`  | int      | `5`            | llm-api        | New standardized var                            |
| `POSTGRES_MAX_OPEN_CONNS`  | int      | `15`           | llm-api        | New standardized var                            |
| `DB_CONN_MAX_LIFETIME`     | duration | `30m`          | llm-api        | OK Already aligned                              |

**Migration Notes:**

- All services use `DB_POSTGRESQL_WRITE_DSN` for database connections
- Connection URL is built from components: `postgres://user:password@host:port/database?sslmode=disable`
- This allows better secret management (password separate from URL)

### Authentication (Keycloak)

| Centralized Env Var        | Type     | Default                               | Services Using | Notes                |
| -------------------------- | -------- | ------------------------------------- | -------------- | -------------------- |
| `KEYCLOAK_BASE_URL`        | string   | `http://keycloak:8085`                | llm-api        | OK Already aligned   |
| `KEYCLOAK_REALM`           | string   | `jan`                                 | llm-api        | OK Already aligned   |
| `KEYCLOAK_HTTP_PORT`       | int      | `8085`                                | Infrastructure | New standardized var |
| `KEYCLOAK_ADMIN`           | string   | `admin`                               | llm-api        | OK Already aligned   |
| `KEYCLOAK_ADMIN_PASSWORD`  | string   | (secret)                              | llm-api        | OK Already aligned   |
| `KEYCLOAK_ADMIN_REALM`     | string   | `master`                              | llm-api        | OK Already aligned   |
| `KEYCLOAK_ADMIN_CLIENT_ID` | string   | `admin-cli`                           | llm-api        | OK Already aligned   |
| `BACKEND_CLIENT_ID`        | string   | `backend`                             | llm-api        | OK Already aligned   |
| `BACKEND_CLIENT_SECRET`    | string   | (secret)                              | llm-api        | OK Already aligned   |
| `CLIENT`                   | string   | `jan-client`                          | llm-api        | OK Already aligned   |
| `OAUTH_REDIRECT_URI`       | string   | `http://localhost:8000/auth/callback` | llm-api        | OK Already aligned   |
| `JWKS_URL`                 | string   | (computed)                            | llm-api        | OK Already aligned   |
| `OIDC_DISCOVERY_URL`       | string   | (computed)                            | llm-api        | New standardized var |
| `ISSUER`                   | string   | `http://localhost:8085/realms/jan`    | llm-api        | OK Already aligned   |
| `ACCOUNT`                  | string   | `account`                             | llm-api        | OK Already aligned   |
| `JWKS_REFRESH_INTERVAL`    | duration | `5m`                                  | llm-api        | OK Already aligned   |
| `AUTH_CLOCK_SKEW`          | duration | `60s`                                 | llm-api        | OK Already aligned   |
| `GUEST_ROLE`               | string   | `guest`                               | llm-api        | OK Already aligned   |
| `KEYCLOAK_FEATURES`        | []string | `token-exchange,preview`              | Infrastructure | New standardized var |

### Gateway (Kong)

| Centralized Env Var | Type   | Default            | Services Using | Notes                |
| ------------------- | ------ | ------------------ | -------------- | -------------------- |
| `KONG_HTTP_PORT`    | int    | `8000`             | Infrastructure | New standardized var |
| `KONG_ADMIN_PORT`   | int    | `8001`             | Infrastructure | New standardized var |
| `KONG_ADMIN_URL`    | string | `http://kong:8001` | llm-api        | OK Already aligned   |
| `KONG_LOG_LEVEL`    | string | `info`             | Infrastructure | New standardized var |

## Services

### LLM API

| Centralized Env Var           | Type     | Default                                   | Current Var                   | Status     |
| ----------------------------- | -------- | ----------------------------------------- | ----------------------------- | ---------- |
| `HTTP_PORT`                   | int      | `8080`                                    | `HTTP_PORT`                   | OK Aligned |
| `METRICS_PORT`                | int      | `9091`                                    | `METRICS_PORT`                | OK Aligned |
| `LOG_LEVEL`                   | string   | `info`                                    | `LOG_LEVEL`                   | OK Aligned |
| `LOG_FORMAT`                  | string   | `json`                                    | `LOG_FORMAT`                  | OK Aligned |
| `AUTO_MIGRATE`                | bool     | `true`                                    | `AUTO_MIGRATE`                | OK Aligned |
| `API_KEY_PREFIX`              | string   | `sk_live`                                 | `API_KEY_PREFIX`              | OK Aligned |
| `API_KEY_DEFAULT_TTL`         | duration | `2160h`                                   | `API_KEY_DEFAULT_TTL`         | OK Aligned |
| `API_KEY_MAX_TTL`             | duration | `2160h`                                   | `API_KEY_MAX_TTL`             | OK Aligned |
| `API_KEY_MAX_PER_USER`        | int      | `5`                                       | `API_KEY_MAX_PER_USER`        | OK Aligned |
| `MODEL_PROVIDER_SECRET`       | string   | `jan-model-provider-secret-2024`          | `MODEL_PROVIDER_SECRET`       | OK Aligned |
| `MODEL_SYNC_ENABLED`          | bool     | `true`                                    | `MODEL_SYNC_ENABLED`          | OK Aligned |
| `MODEL_SYNC_INTERVAL_MINUTES` | int      | `60`                                      | `MODEL_SYNC_INTERVAL_MINUTES` | OK Aligned |
| `MEDIA_RESOLVE_URL`           | string   | `http://kong:8000/media/v1/media/resolve` | `MEDIA_RESOLVE_URL`           | OK Aligned |
| `MEDIA_RESOLVE_TIMEOUT`       | duration | `5s`                                      | `MEDIA_RESOLVE_TIMEOUT`       | OK Aligned |

**Provider Config:**
| Centralized Env Var | Type | Default | Current Var | Status |
|---------------------|------|---------|-------------|--------|
| `JAN_PROVIDER_CONFIGS_FILE` | string | `configs/providers.yml` | `JAN_PROVIDER_CONFIGS_FILE` | TODO Path may differ |
| `JAN_PROVIDER_CONFIG_SET` | string | `default` | `JAN_PROVIDER_CONFIG_SET` | OK Aligned |
| `JAN_PROVIDER_CONFIGS` | bool | `true` | `JAN_PROVIDER_CONFIGS` | OK Aligned |

### MCP Tools

| Centralized Env Var            | Type     | Default                         | Current Var                | Status           |
| ------------------------------ | -------- | ------------------------------- | -------------------------- | ---------------- |
| `MCP_TOOLS_HTTP_PORT`          | int      | `8091`                          | `HTTP_PORT`                | TODO Need prefix |
| `MCP_TOOLS_LOG_LEVEL`          | string   | `info`                          | `LOG_LEVEL`                | TODO Need prefix |
| `MCP_TOOLS_LOG_FORMAT`         | string   | `json`                          | `LOG_FORMAT`               | TODO Need prefix |
| `MCP_SEARCH_ENGINE`            | string   | `serper`                        | `SEARCH_ENGINE`            | TODO Need prefix |
| `SERPER_ENABLED`               | bool     | `true`                          | `SERPER_ENABLED`           | New              |
| `SERPER_API_KEY`               | string   | (secret)                        | `SERPER_API_KEY`           | OK Aligned       |
| `EXA_ENABLED`                  | bool     | `false`                         | `EXA_ENABLED`              | New              |
| `EXA_API_KEY`                  | string   | (secret)                        | `EXA_API_KEY`              | New              |
| `EXA_SEARCH_ENDPOINT`          | string   | `https://api.exa.ai/search`     | `EXA_SEARCH_ENDPOINT`      | New              |
| `EXA_TIMEOUT`                  | duration | `15s`                           | `EXA_TIMEOUT`              | New              |
| `TAVILY_ENABLED`               | bool     | `false`                         | `TAVILY_ENABLED`           | New              |
| `TAVILY_API_KEY`               | string   | (secret)                        | `TAVILY_API_KEY`           | New              |
| `TAVILY_SEARCH_ENDPOINT`       | string   | `https://api.tavily.com/search` | `TAVILY_SEARCH_ENDPOINT`   | New              |
| `TAVILY_TIMEOUT`               | duration | `15s`                           | `TAVILY_TIMEOUT`           | New              |
| `SEARXNG_URL`                  | string   | `http://searxng:8080`           | `SEARXNG_URL`              | OK Aligned       |
| `SEARXNG_ENABLED`              | bool     | `false`                         | `SEARXNG_ENABLED`          | New              |
| `VECTOR_STORE_URL`             | string   | `http://vector-store:3015`      | `VECTOR_STORE_URL`         | OK Aligned       |
| `SANDBOXFUSION_URL`            | string   | `http://sandboxfusion:8080`     | `SANDBOXFUSION_URL`        | OK Aligned       |
| `MCP_SANDBOX_REQUIRE_APPROVAL` | bool     | `true`                          | `SANDBOX_REQUIRE_APPROVAL` | TODO Need prefix |
| `MCP_CONFIG_FILE`              | string   | `configs/mcp-providers.yml`     | `MCP_CONFIG_FILE`          | OK Aligned       |

**Migration Notes:**

- Add `MCP_` or `MCP_TOOLS_` prefix to disambiguate from other services
- HTTP_PORT collision with llm-api when running in same environment

### Memory Tools

| Centralized Env Var          | Type     | Default                | Current Var                  | Status            |
| ---------------------------- | -------- | ---------------------- | ---------------------------- | ----------------- |
| `MEMORY_TOOLS_PORT`          | int      | `8090`                 | `MEMORY_TOOLS_PORT`          | OK Aligned        |
| `DB_POSTGRESQL_WRITE_DSN`    | string   | (computed)             | -                            | ✅ Standard       |
| `DB_POSTGRESQL_READ1_DSN`    | string   | -                      | -                            | ✅ New (optional) |
| `MEMORY_LOG_LEVEL`           | string   | `info`                 | `LOG_LEVEL`                  | TODO Need prefix  |
| `MEMORY_LOG_FORMAT`          | string   | `json`                 | `LOG_FORMAT`                 | TODO Need prefix  |
| `EMBEDDING_SERVICE_URL`      | string   | -                      | `EMBEDDING_SERVICE_URL`      | OK Aligned        |
| `EMBEDDING_CACHE_TYPE`       | string   | `memory`               | `EMBEDDING_CACHE_TYPE`       | OK Aligned        |
| `EMBEDDING_CACHE_REDIS_URL`  | string   | `redis://redis:6379/3` | `EMBEDDING_CACHE_REDIS_URL`  | OK Aligned        |
| `EMBEDDING_CACHE_KEY_PREFIX` | string   | `emb:`                 | `EMBEDDING_CACHE_KEY_PREFIX` | OK Aligned        |
| `EMBEDDING_CACHE_MAX_SIZE`   | int      | `10000`                | `EMBEDDING_CACHE_MAX_SIZE`   | OK Aligned        |
| `EMBEDDING_CACHE_TTL`        | duration | `1h`                   | `EMBEDDING_CACHE_TTL`        | OK Aligned        |

**Migration Notes:**

- Database configuration uses `DB_POSTGRESQL_WRITE_DSN` and optional `DB_POSTGRESQL_READ1_DSN` for read replicas
- Supports read/write splitting for better scalability
- Can share database with other services or use separate database
- Read replica is optional; falls back to write DSN if not configured

### Realtime API

| Centralized Env Var        | Type     | Default                         | Current Var                | Status           |
| -------------------------- | -------- | ------------------------------- | -------------------------- | ---------------- |
| `REALTIME_API_PORT`        | int      | `8186`                          | `REALTIME_API_PORT`        | OK Aligned       |
| `REALTIME_LOG_LEVEL`       | string   | `info`                          | `LOG_LEVEL`                | TODO Need prefix |
| `LIVEKIT_WS_URL`           | string   | `wss://your-livekit-server.com` | `LIVEKIT_WS_URL`           | OK Aligned       |
| `LIVEKIT_API_KEY`          | string   | (secret)                        | `LIVEKIT_API_KEY`          | OK Aligned       |
| `LIVEKIT_API_SECRET`       | string   | (secret)                        | `LIVEKIT_API_SECRET`       | OK Aligned       |
| `LIVEKIT_TOKEN_TTL`        | duration | `24h`                           | `LIVEKIT_TOKEN_TTL`        | OK Aligned       |
| `SESSION_STALE_TTL`        | duration | `10m`                           | `SESSION_STALE_TTL`        | OK Aligned       |
| `SESSION_CLEANUP_INTERVAL` | duration | `15s`                           | `SESSION_CLEANUP_INTERVAL` | OK Aligned       |
| `REALTIME_AUTH_ENABLED`    | bool     | `true`                          | `AUTH_ENABLED`             | TODO Need prefix |

**Migration Notes:**

- LiveKit configuration already uses proper prefixes
- Session management variables are clear and aligned
- Only generic LOG*LEVEL and AUTH_ENABLED need REALTIME* prefix for consistency

### Media API

| Centralized Env Var          | Type     | Default               | Current Var             | Status                  |
| ---------------------------- | -------- | --------------------- | ----------------------- | ----------------------- |
| `MEDIA_API_PORT`             | int      | `8285`                | `HTTP_PORT`             | TODO Need rename        |
| `MEDIA_API_LOG_LEVEL`        | string   | `info`                | `LOG_LEVEL`             | TODO Need prefix        |
| `MEDIA_MAX_UPLOAD_BYTES`     | int      | `20971520`            | `MAX_UPLOAD_SIZE`       | TODO Rename needed      |
| `MEDIA_RETENTION_DAYS`       | int      | `30`                  | `RETENTION_DAYS`        | TODO Need prefix        |
| `MEDIA_PROXY_DOWNLOAD`       | bool     | `true`                | `PROXY_DOWNLOAD`        | TODO Need prefix        |
| `MEDIA_REMOTE_FETCH_TIMEOUT` | duration | `15s`                 | `FETCH_TIMEOUT`         | TODO Rename needed      |
| `MEDIA_S3_ENDPOINT`          | string   | `https://s3.menlo.ai` | `S3_ENDPOINT`           | TODO Need prefix        |
| `MEDIA_S3_PUBLIC_ENDPOINT`   | string   | (empty)               | `S3_PUBLIC_ENDPOINT`    | TODO Need prefix        |
| `MEDIA_S3_URL_ENABLED`       | bool     | `false`               | `S3_URL_ENABLED`        | New                     |
| `MEDIA_S3_REGION`            | string   | `us-west-2`           | `S3_REGION`             | TODO Need prefix        |
| `MEDIA_S3_BUCKET`            | string   | `platform-dev`        | `S3_BUCKET`             | TODO Need prefix        |
| `MEDIA_S3_USE_PATH_STYLE`    | bool     | `true`                | `S3_PATH_STYLE`         | TODO Rename needed      |
| `MEDIA_S3_PRESIGN_TTL`       | duration | `168h`                | `PRESIGN_TTL`           | TODO Need prefix        |
| `MEDIA_S3_ACCESS_KEY_ID`     | string   | (secret)              | `AWS_ACCESS_KEY_ID`     | TODO Rename for clarity |
| `MEDIA_S3_SECRET_ACCESS_KEY` | string   | (secret)              | `AWS_SECRET_ACCESS_KEY` | TODO Rename for clarity |

**Migration Notes:**

- Most env vars need `MEDIA_` prefix to avoid conflicts
- S3 vars should use `MEDIA_S3_` prefix for clarity
- Consider AWS credential standardization

### Response API

| Centralized Env Var       | Type     | Default                 | Current Var      | Status           |
| ------------------------- | -------- | ----------------------- | ---------------- | ---------------- |
| `RESPONSE_API_PORT`       | int      | `8082`                  | `HTTP_PORT`      | TODO Need rename |
| `RESPONSE_API_LOG_LEVEL`  | string   | `info`                  | `LOG_LEVEL`      | TODO Need prefix |
| `RESPONSE_LLM_API_URL`    | string   | `http://llm-api:8080`   | `LLM_API_URL`    | TODO Need prefix |
| `RESPONSE_MCP_TOOLS_URL`  | string   | `http://mcp-tools:8091` | `MCP_TOOLS_URL`  | TODO Need prefix |
| `RESPONSE_MAX_TOOL_DEPTH` | int      | `8`                     | `MAX_TOOL_DEPTH` | TODO Need prefix |
| `RESPONSE_TOOL_TIMEOUT`   | duration | `45s`                   | `TOOL_TIMEOUT`   | TODO Need prefix |

## Monitoring

### OpenTelemetry

| Centralized Env Var           | Type   | Default                      | Services Using | Status                |
| ----------------------------- | ------ | ---------------------------- | -------------- | --------------------- |
| `OTEL_ENABLED`                | bool   | `false`                      | All services   | OK Standard           |
| `OTEL_SERVICE_NAME`           | string | `llm-api`                    | All services   | TODO Service-specific |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | string | `http://otel-collector:4318` | All services   | OK Standard           |
| `OTEL_HTTP_PORT`              | int    | `4318`                       | Infrastructure | New                   |
| `OTEL_GRPC_PORT`              | int    | `4317`                       | Infrastructure | New                   |

### Prometheus

| Centralized Env Var | Type | Default | Services Using | Status |
| ------------------- | ---- | ------- | -------------- | ------ |
| `PROMETHEUS_PORT`   | int  | `9090`  | Infrastructure | New    |

### Grafana

| Centralized Env Var      | Type   | Default  | Services Using | Status |
| ------------------------ | ------ | -------- | -------------- | ------ |
| `GRAFANA_PORT`           | int    | `3331`   | Infrastructure | New    |
| `GRAFANA_ADMIN_USER`     | string | `admin`  | Infrastructure | New    |
| `GRAFANA_ADMIN_PASSWORD` | string | (secret) | Infrastructure | New    |

### Jaeger

| Centralized Env Var | Type | Default | Services Using | Status |
| ------------------- | ---- | ------- | -------------- | ------ |
| `JAEGER_UI_PORT`    | int  | `16686` | Infrastructure | New    |

## Inference

### vLLM

| Centralized Env Var    | Type   | Default                      | Services Using | Status |
| ---------------------- | ------ | ---------------------------- | -------------- | ------ |
| `VLLM_ENABLED`         | bool   | `true`                       | Infrastructure | New    |
| `VLLM_PORT`            | int    | `8001`                       | llm-api        | New    |
| `VLLM_MODEL`           | string | `Qwen/Qwen2.5-0.5B-Instruct` | Infrastructure | New    |
| `VLLM_SERVED_NAME`     | string | `qwen2.5-0.5b-instruct`      | Infrastructure | New    |
| `VLLM_GPU_UTILIZATION` | float  | `0.66`                       | Infrastructure | New    |

## Migration Priority

### Phase 1: Critical (Sprint 3.1)

OK **Already Aligned - No Changes Needed:**

- llm-api authentication vars (Keycloak)
- llm-api API key management
- llm-api model sync
- Database connection timeouts

### Phase 2: High Priority (Sprint 3.2)

TODO **Requires Prefix/Rename:**

- Service-specific HTTP_PORT -> \{SERVICE\}\_PORT
- Service-specific LOG_LEVEL -> \{SERVICE\}\_LOG_LEVEL
- Database URL components

### Phase 3: Medium Priority (Sprint 3.3)

TODO **New Variables - Add Support:**

- Infrastructure monitoring ports (Prometheus, Grafana, Jaeger)
- vLLM inference configuration
- Kong gateway ports
- Database connection pool settings

### Phase 4: Low Priority (Sprint 3.4)

TODO **Nice to Have:**

- Media API S3 prefixing
- Response API prefixing
- MCP Tools prefixing

## Testing Strategy

### Per-Service Testing

For each service after env var migration:

1. **Unit Tests:** Verify config loading with new env vars
2. **Integration Tests:** Test with Docker Compose
3. **Precedence Tests:** Verify env vars override defaults
4. **Backward Compatibility:** Old env vars still work (deprecation warnings)

### Test Script Template

```bash
#!/bin/bash
# Test service with new env vars

# Set centralized env vars
export POSTGRES_HOST=testdb
export POSTGRES_PORT=5432
export POSTGRES_USER=testuser
export POSTGRES_PASSWORD=testpass
export POSTGRES_DB=testdb
export POSTGRES_SSL_MODE=disable

# Run service
./service-binary

# Verify config loaded correctly
curl http://localhost:8080/health
```

## Rollback Plan

If migration causes issues:

1. **Immediate:** Revert docker compose.yml to use old env vars
2. **Service-Level:** Keep backward compatibility (read both old and new vars)
3. **Gradual Migration:** Migrate one service at a time, not all at once

## See Also

- [Service Migration Strategy](./service-migration-strategy.md)
- [Configuration Precedence](./precedence.md)
- [Configuration Types Reference](../../pkg/config/types.go)
