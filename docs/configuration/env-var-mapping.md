# Environment Variable Mapping

This document maps centralized configuration (`pkg/config/types.go`) environment variables to service-specific variables, facilitating the Sprint 3 migration.

## Infrastructure

### Database (PostgreSQL)

| Centralized Env Var | Type | Default | Services Using | Notes |
|---------------------|------|---------|----------------|-------|
| `POSTGRES_HOST` | string | `api-db` | llm-api | Replaces `DATABASE_URL` component |
| `POSTGRES_PORT` | int | `5432` | llm-api | Replaces `DATABASE_URL` component |
| `POSTGRES_USER` | string | `jan_user` | llm-api | Replaces `DATABASE_URL` component |
| `POSTGRES_PASSWORD` | string | `jan_password` | llm-api | From secrets, replaces `DATABASE_URL` component |
| `POSTGRES_DB` | string | `jan_llm_api` | llm-api | Replaces `DATABASE_URL` component |
| `POSTGRES_SSL_MODE` | string | `disable` | llm-api | Replaces `DATABASE_URL` component |
| `POSTGRES_MAX_CONNECTIONS` | int | `100` | llm-api | New standardized var |
| `POSTGRES_MAX_IDLE_CONNS` | int | `5` | llm-api | New standardized var |
| `POSTGRES_MAX_OPEN_CONNS` | int | `15` | llm-api | New standardized var |
| `DB_CONN_MAX_LIFETIME` | duration | `30m` | llm-api | ✅ Already aligned |

**Migration Notes:**
- Services currently using `DATABASE_URL` should transition to component-based env vars
- Connection URL is built from components: `postgres://user:password@host:port/database?sslmode=disable`
- This allows better secret management (password separate from URL)

### Authentication (Keycloak)

| Centralized Env Var | Type | Default | Services Using | Notes |
|---------------------|------|---------|----------------|-------|
| `KEYCLOAK_BASE_URL` | string | `http://keycloak:8085` | llm-api | ✅ Already aligned |
| `KEYCLOAK_REALM` | string | `jan` | llm-api | ✅ Already aligned |
| `KEYCLOAK_HTTP_PORT` | int | `8085` | Infrastructure | New standardized var |
| `KEYCLOAK_ADMIN` | string | `admin` | llm-api | ✅ Already aligned |
| `KEYCLOAK_ADMIN_PASSWORD` | string | (secret) | llm-api | ✅ Already aligned |
| `KEYCLOAK_ADMIN_REALM` | string | `master` | llm-api | ✅ Already aligned |
| `KEYCLOAK_ADMIN_CLIENT_ID` | string | `admin-cli` | llm-api | ✅ Already aligned |
| `BACKEND_CLIENT_ID` | string | `backend` | llm-api | ✅ Already aligned |
| `BACKEND_CLIENT_SECRET` | string | (secret) | llm-api | ✅ Already aligned |
| `TARGET_CLIENT_ID` | string | `jan-client` | llm-api | ✅ Already aligned |
| `OAUTH_REDIRECT_URI` | string | `http://localhost:8000/auth/callback` | llm-api | ✅ Already aligned |
| `JWKS_URL` | string | (computed) | llm-api | ✅ Already aligned |
| `OIDC_DISCOVERY_URL` | string | (computed) | llm-api | New standardized var |
| `ISSUER` | string | `http://localhost:8085/realms/jan` | llm-api | ✅ Already aligned |
| `AUDIENCE` | string | `jan-client` | llm-api | ✅ Already aligned |
| `JWKS_REFRESH_INTERVAL` | duration | `5m` | llm-api | ✅ Already aligned |
| `AUTH_CLOCK_SKEW` | duration | `60s` | llm-api | ✅ Already aligned |
| `GUEST_ROLE` | string | `guest` | llm-api | ✅ Already aligned |
| `KEYCLOAK_FEATURES` | []string | `token-exchange,preview` | Infrastructure | New standardized var |

### Gateway (Kong)

| Centralized Env Var | Type | Default | Services Using | Notes |
|---------------------|------|---------|----------------|-------|
| `KONG_HTTP_PORT` | int | `8000` | Infrastructure | New standardized var |
| `KONG_ADMIN_PORT` | int | `8001` | Infrastructure | New standardized var |
| `KONG_ADMIN_URL` | string | `http://kong:8001` | llm-api | ✅ Already aligned |
| `KONG_LOG_LEVEL` | string | `info` | Infrastructure | New standardized var |

## Services

### LLM API

| Centralized Env Var | Type | Default | Current Var | Status |
|---------------------|------|---------|-------------|--------|
| `HTTP_PORT` | int | `8080` | `HTTP_PORT` | ✅ Aligned |
| `METRICS_PORT` | int | `9091` | `METRICS_PORT` | ✅ Aligned |
| `LOG_LEVEL` | string | `info` | `LOG_LEVEL` | ✅ Aligned |
| `LOG_FORMAT` | string | `json` | `LOG_FORMAT` | ✅ Aligned |
| `AUTO_MIGRATE` | bool | `true` | `AUTO_MIGRATE` | ✅ Aligned |
| `API_KEY_PREFIX` | string | `sk_live` | `API_KEY_PREFIX` | ✅ Aligned |
| `API_KEY_DEFAULT_TTL` | duration | `2160h` | `API_KEY_DEFAULT_TTL` | ✅ Aligned |
| `API_KEY_MAX_TTL` | duration | `2160h` | `API_KEY_MAX_TTL` | ✅ Aligned |
| `API_KEY_MAX_PER_USER` | int | `5` | `API_KEY_MAX_PER_USER` | ✅ Aligned |
| `MODEL_PROVIDER_SECRET` | string | `jan-model-provider-secret-2024` | `MODEL_PROVIDER_SECRET` | ✅ Aligned |
| `MODEL_SYNC_ENABLED` | bool | `true` | `MODEL_SYNC_ENABLED` | ✅ Aligned |
| `MODEL_SYNC_INTERVAL_MINUTES` | int | `60` | `MODEL_SYNC_INTERVAL_MINUTES` | ✅ Aligned |
| `MEDIA_RESOLVE_URL` | string | `http://kong:8000/media/v1/media/resolve` | `MEDIA_RESOLVE_URL` | ✅ Aligned |
| `MEDIA_RESOLVE_TIMEOUT` | duration | `5s` | `MEDIA_RESOLVE_TIMEOUT` | ✅ Aligned |

**Provider Config:**
| Centralized Env Var | Type | Default | Current Var | Status |
|---------------------|------|---------|-------------|--------|
| `JAN_PROVIDER_CONFIGS_FILE` | string | `config/providers.yml` | `JAN_PROVIDER_CONFIGS_FILE` | ⚠️ Path may differ |
| `JAN_PROVIDER_CONFIG_SET` | string | `default` | `JAN_PROVIDER_CONFIG_SET` | ✅ Aligned |
| `JAN_PROVIDER_CONFIGS` | bool | `true` | `JAN_PROVIDER_CONFIGS` | ✅ Aligned |

### MCP Tools

| Centralized Env Var | Type | Default | Current Var | Status |
|---------------------|------|---------|-------------|--------|
| `MCP_TOOLS_HTTP_PORT` | int | `8091` | `HTTP_PORT` | ⚠️ Need prefix |
| `MCP_TOOLS_LOG_LEVEL` | string | `info` | `LOG_LEVEL` | ⚠️ Need prefix |
| `MCP_TOOLS_LOG_FORMAT` | string | `json` | `LOG_FORMAT` | ⚠️ Need prefix |
| `MCP_SEARCH_ENGINE` | string | `serper` | `SEARCH_ENGINE` | ⚠️ Need prefix |
| `SEARXNG_URL` | string | `http://searxng:8080` | `SEARXNG_URL` | ✅ Aligned |
| `VECTOR_STORE_URL` | string | `http://vector-store:3015` | `VECTOR_STORE_URL` | ✅ Aligned |
| `SANDBOXFUSION_URL` | string | `http://sandboxfusion:8080` | `SANDBOXFUSION_URL` | ✅ Aligned |
| `MCP_SANDBOX_REQUIRE_APPROVAL` | bool | `true` | `SANDBOX_REQUIRE_APPROVAL` | ⚠️ Need prefix |
| `MCP_CONFIG_FILE` | string | `configs/mcp-providers.yml` | `MCP_CONFIG_FILE` | ✅ Aligned |

**Migration Notes:**
- Add `MCP_` or `MCP_TOOLS_` prefix to disambiguate from other services
- HTTP_PORT collision with llm-api when running in same environment

### Media API

| Centralized Env Var | Type | Default | Current Var | Status |
|---------------------|------|---------|-------------|--------|
| `MEDIA_API_PORT` | int | `8285` | `HTTP_PORT` | ⚠️ Need rename |
| `MEDIA_API_LOG_LEVEL` | string | `info` | `LOG_LEVEL` | ⚠️ Need prefix |
| `MEDIA_MAX_UPLOAD_BYTES` | int | `20971520` | `MAX_UPLOAD_SIZE` | ⚠️ Rename needed |
| `MEDIA_RETENTION_DAYS` | int | `30` | `RETENTION_DAYS` | ⚠️ Need prefix |
| `MEDIA_PROXY_DOWNLOAD` | bool | `true` | `PROXY_DOWNLOAD` | ⚠️ Need prefix |
| `MEDIA_REMOTE_FETCH_TIMEOUT` | duration | `15s` | `FETCH_TIMEOUT` | ⚠️ Rename needed |
| `MEDIA_S3_ENDPOINT` | string | `https://s3.menlo.ai` | `S3_ENDPOINT` | ⚠️ Need prefix |
| `MEDIA_S3_REGION` | string | `us-west-2` | `S3_REGION` | ⚠️ Need prefix |
| `MEDIA_S3_BUCKET` | string | `platform-dev` | `S3_BUCKET` | ⚠️ Need prefix |
| `MEDIA_S3_USE_PATH_STYLE` | bool | `true` | `S3_PATH_STYLE` | ⚠️ Rename needed |
| `MEDIA_S3_PRESIGN_TTL` | duration | `5m` | `PRESIGN_TTL` | ⚠️ Need prefix |
| `MEDIA_S3_ACCESS_KEY_ID` | string | (secret) | `AWS_ACCESS_KEY_ID` | ⚠️ Rename for clarity |
| `MEDIA_S3_SECRET_ACCESS_KEY` | string | (secret) | `AWS_SECRET_ACCESS_KEY` | ⚠️ Rename for clarity |

**Migration Notes:**
- Most env vars need `MEDIA_` prefix to avoid conflicts
- S3 vars should use `MEDIA_S3_` prefix for clarity
- Consider AWS credential standardization

### Response API

| Centralized Env Var | Type | Default | Current Var | Status |
|---------------------|------|---------|-------------|--------|
| `RESPONSE_API_PORT` | int | `8082` | `HTTP_PORT` | ⚠️ Need rename |
| `RESPONSE_API_LOG_LEVEL` | string | `info` | `LOG_LEVEL` | ⚠️ Need prefix |
| `RESPONSE_LLM_API_URL` | string | `http://llm-api:8080` | `LLM_API_URL` | ⚠️ Need prefix |
| `RESPONSE_MCP_TOOLS_URL` | string | `http://mcp-tools:8091` | `MCP_TOOLS_URL` | ⚠️ Need prefix |
| `RESPONSE_MAX_TOOL_DEPTH` | int | `8` | `MAX_TOOL_DEPTH` | ⚠️ Need prefix |
| `RESPONSE_TOOL_TIMEOUT` | duration | `45s` | `TOOL_TIMEOUT` | ⚠️ Need prefix |

## Monitoring

### OpenTelemetry

| Centralized Env Var | Type | Default | Services Using | Status |
|---------------------|------|---------|----------------|--------|
| `OTEL_ENABLED` | bool | `false` | All services | ✅ Standard |
| `OTEL_SERVICE_NAME` | string | `llm-api` | All services | ⚠️ Service-specific |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | string | `http://otel-collector:4318` | All services | ✅ Standard |
| `OTEL_HTTP_PORT` | int | `4318` | Infrastructure | New |
| `OTEL_GRPC_PORT` | int | `4317` | Infrastructure | New |

### Prometheus

| Centralized Env Var | Type | Default | Services Using | Status |
|---------------------|------|---------|----------------|--------|
| `PROMETHEUS_PORT` | int | `9090` | Infrastructure | New |

### Grafana

| Centralized Env Var | Type | Default | Services Using | Status |
|---------------------|------|---------|----------------|--------|
| `GRAFANA_PORT` | int | `3001` | Infrastructure | New |
| `GRAFANA_ADMIN_USER` | string | `admin` | Infrastructure | New |
| `GRAFANA_ADMIN_PASSWORD` | string | (secret) | Infrastructure | New |

### Jaeger

| Centralized Env Var | Type | Default | Services Using | Status |
|---------------------|------|---------|----------------|--------|
| `JAEGER_UI_PORT` | int | `16686` | Infrastructure | New |

## Inference

### vLLM

| Centralized Env Var | Type | Default | Services Using | Status |
|---------------------|------|---------|-------------|--------|
| `VLLM_ENABLED` | bool | `true` | Infrastructure | New |
| `VLLM_PORT` | int | `8001` | llm-api | New |
| `VLLM_MODEL` | string | `Qwen/Qwen2.5-0.5B-Instruct` | Infrastructure | New |
| `VLLM_SERVED_NAME` | string | `qwen2.5-0.5b-instruct` | Infrastructure | New |
| `VLLM_GPU_UTILIZATION` | float | `0.66` | Infrastructure | New |

## Migration Priority

### Phase 1: Critical (Sprint 3.1)
✅ **Already Aligned - No Changes Needed:**
- llm-api authentication vars (Keycloak)
- llm-api API key management
- llm-api model sync
- Database connection timeouts

### Phase 2: High Priority (Sprint 3.2)
⚠️ **Requires Prefix/Rename:**
- Service-specific HTTP_PORT → {SERVICE}_PORT
- Service-specific LOG_LEVEL → {SERVICE}_LOG_LEVEL
- Database URL components (transition from DATABASE_URL)

### Phase 3: Medium Priority (Sprint 3.3)
⚠️ **New Variables - Add Support:**
- Infrastructure monitoring ports (Prometheus, Grafana, Jaeger)
- vLLM inference configuration
- Kong gateway ports
- Database connection pool settings

### Phase 4: Low Priority (Sprint 3.4)
⚠️ **Nice to Have:**
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

1. **Immediate:** Revert docker-compose.yml to use old env vars
2. **Service-Level:** Keep backward compatibility (read both old and new vars)
3. **Gradual Migration:** Migrate one service at a time, not all at once

## See Also

- [Service Migration Strategy](./service-migration-strategy.md)
- [Configuration Precedence](./precedence.md)
- [Configuration Types Reference](../../pkg/config/types.go)
