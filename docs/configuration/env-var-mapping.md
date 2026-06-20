# Environment Variable Reference

This document lists the environment variables consumed by Jan Server services,
grouped by area. Variable names and defaults are defined in
`packages/go-common/config/types.go` and surfaced in the root `.env` (created
from `.env.template`). Use it as a quick lookup when configuring `.env` or
container overrides.

> All of these can be set in the root `.env` or exported in the shell. Docker
> Compose loads `.env` into every service via `env_file`, and shell/OS values
> take precedence. See [precedence.md](precedence.md) for the full ordering.

## Infrastructure

### Database (PostgreSQL)

| Env Var                    | Type     | Default        | Notes                                           |
| -------------------------- | -------- | -------------- | ----------------------------------------------- |
| `POSTGRES_HOST`            | string   | `api-db`       | Used in `DB_POSTGRESQL_WRITE_DSN`               |
| `POSTGRES_PORT`            | int      | `5432`         | Used in `DB_POSTGRESQL_WRITE_DSN`               |
| `POSTGRES_USER`            | string   | `jan_user`     | Used in `DB_POSTGRESQL_WRITE_DSN`               |
| `POSTGRES_PASSWORD`        | string   | `jan_password` | Secret; used in `DB_POSTGRESQL_WRITE_DSN`       |
| `POSTGRES_DB`              | string   | `jan_llm_api`  | Used in `DB_POSTGRESQL_WRITE_DSN`               |
| `POSTGRES_SSL_MODE`        | string   | `disable`      | Used in `DB_POSTGRESQL_WRITE_DSN`               |
| `POSTGRES_MAX_CONNECTIONS` | int      | `100`          | Connection pool ceiling                         |
| `POSTGRES_MAX_IDLE_CONNS`  | int      | `5`            | Idle connection pool size                       |
| `POSTGRES_MAX_OPEN_CONNS`  | int      | `15`           | Open connection pool size                       |
| `DB_CONN_MAX_LIFETIME`     | duration | `30m`          | Max connection lifetime                         |

**Notes:**

- Services connect via `DB_POSTGRESQL_WRITE_DSN`.
- The connection URL is built from components: `postgres://user:password@host:port/database?sslmode=disable`.
- Keeping the password separate from the URL allows better secret management.

### Authentication (Keycloak)

| Env Var                    | Type     | Default                               | Notes               |
| -------------------------- | -------- | ------------------------------------- | ------------------- |
| `KEYCLOAK_BASE_URL`        | string   | `http://keycloak:8085`                |                     |
| `KEYCLOAK_REALM`           | string   | `jan`                                 |                     |
| `KEYCLOAK_HTTP_PORT`       | int      | `8085`                                | Infrastructure      |
| `KEYCLOAK_ADMIN`           | string   | `admin`                               |                     |
| `KEYCLOAK_ADMIN_PASSWORD`  | string   | (secret)                              |                     |
| `KEYCLOAK_ADMIN_REALM`     | string   | `master`                              |                     |
| `KEYCLOAK_ADMIN_CLIENT_ID` | string   | `admin-cli`                           |                     |
| `BACKEND_CLIENT_ID`        | string   | `backend`                             |                     |
| `BACKEND_CLIENT_SECRET`    | string   | (secret)                              |                     |
| `CLIENT`                   | string   | `jan-client`                          |                     |
| `OAUTH_REDIRECT_URI`       | string   | `http://localhost:8000/auth/callback` |                     |
| `JWKS_URL`                 | string   | (computed)                            |                     |
| `OIDC_DISCOVERY_URL`       | string   | (computed)                            |                     |
| `ISSUER`                   | string   | `http://localhost:8085/realms/jan`    |                     |
| `ACCOUNT`                  | string   | `account`                             |                     |
| `JWKS_REFRESH_INTERVAL`    | duration | `5m`                                  |                     |
| `AUTH_CLOCK_SKEW`          | duration | `60s`                                 |                     |
| `GUEST_ROLE`               | string   | `guest`                               |                     |
| `KEYCLOAK_FEATURES`        | []string | `token-exchange,preview`              | Comma-separated     |

### Gateway (Kong)

| Env Var           | Type   | Default            | Notes          |
| ----------------- | ------ | ------------------ | -------------- |
| `KONG_HTTP_PORT`  | int    | `8000`             | Infrastructure |
| `KONG_ADMIN_PORT` | int    | `8001`             | Infrastructure |
| `KONG_ADMIN_URL`  | string | `http://kong:8001` |                |
| `KONG_LOG_LEVEL`  | string | `info`             | Infrastructure |

## Services

### LLM API

| Env Var                       | Type     | Default                                   | Notes |
| ----------------------------- | -------- | ----------------------------------------- | ----- |
| `HTTP_PORT`                   | int      | `8080`                                    |       |
| `METRICS_PORT`                | int      | `9091`                                    |       |
| `LOG_LEVEL`                   | string   | `info`                                    |       |
| `LOG_FORMAT`                  | string   | `json`                                    |       |
| `AUTO_MIGRATE`                | bool     | `true`                                    |       |
| `API_KEY_PREFIX`              | string   | `sk_live`                                 |       |
| `API_KEY_DEFAULT_TTL`         | duration | `2160h`                                   |       |
| `API_KEY_MAX_TTL`             | duration | `2160h`                                   |       |
| `API_KEY_MAX_PER_USER`        | int      | `5`                                       |       |
| `MODEL_PROVIDER_SECRET`       | string   | `jan-model-provider-secret-2024`          | Secret |
| `MODEL_SYNC_ENABLED`          | bool     | `true`                                    |       |
| `MODEL_SYNC_INTERVAL_MINUTES` | int      | `60`                                      |       |
| `MEDIA_RESOLVE_URL`           | string   | `http://kong:8000/media/v1/media/resolve` |       |
| `MEDIA_RESOLVE_TIMEOUT`       | duration | `5s`                                      |       |
| `DOCUMENT_OCR_ENABLED`        | bool     | `false`                                   |       |
| `DOCUMENT_OCR_TIMEOUT`        | duration | `120s`                                    |       |
| `DOCUMENT_OCR_MODEL`          | string   | `docling-v1`                              |       |
| `DOCUMENT_MAX_BYTES`          | int      | `52428800`                                |       |
| `DOCUMENT_SUPPORTED_TYPES`    | string   | (list)                                    |       |
| `DOCLING_ENABLED`             | bool     | `false`                                   |       |
| `DOCLING_PROVIDER_URL`        | string   | (empty)                                   |       |
| `DOCLING_API_KEY`             | string   | (secret)                                  |       |
| `PREFERENCES_DEFAULT_HIDE_CONNECTORS` | bool | `true`                            |       |
| `PREFERENCES_DEFAULT_HIDE_ARTIFACTS`  | bool | `true`                            |       |

**Provider Config:**

| Env Var                     | Type   | Default                 | Notes |
| --------------------------- | ------ | ----------------------- | ----- |
| `JAN_PROVIDER_CONFIGS_FILE` | string | `configs/providers.yml` |       |
| `JAN_PROVIDER_CONFIG_SET`   | string | `default`               |       |
| `JAN_PROVIDER_CONFIGS`      | bool   | `true`                  |       |

### MCP Tools

| Env Var                        | Type     | Default                         | Notes |
| ------------------------------ | -------- | ------------------------------- | ----- |
| `MCP_TOOLS_HTTP_PORT`          | int      | `8091`                          |       |
| `MCP_TOOLS_LOG_LEVEL`          | string   | `info`                          |       |
| `MCP_TOOLS_LOG_FORMAT`         | string   | `json`                          |       |
| `MCP_SEARCH_ENGINE`            | string   | `serper`                        |       |
| `SERPER_ENABLED`               | bool     | `true`                          |       |
| `SERPER_API_KEY`               | string   | (secret)                        |       |
| `EXA_ENABLED`                  | bool     | `false`                         |       |
| `EXA_API_KEY`                  | string   | (secret)                        |       |
| `EXA_SEARCH_ENDPOINT`          | string   | `https://api.exa.ai/search`     |       |
| `EXA_TIMEOUT`                  | duration | `15s`                           |       |
| `TAVILY_ENABLED`               | bool     | `false`                         |       |
| `TAVILY_API_KEY`               | string   | (secret)                        |       |
| `TAVILY_SEARCH_ENDPOINT`       | string   | `https://api.tavily.com/search` |       |
| `TAVILY_TIMEOUT`               | duration | `15s`                           |       |
| `SEARXNG_URL`                  | string   | `http://searxng:8080`           |       |
| `SEARXNG_ENABLED`              | bool     | `false`                         |       |
| `VECTOR_STORE_URL`             | string   | `http://vector-store:3015`      |       |
| `SANDBOXFUSION_URL`            | string   | `http://sandboxfusion:8080`     |       |
| `MCP_SANDBOX_REQUIRE_APPROVAL` | bool     | `true`                          |       |
| `MCP_CONFIG_FILE`              | string   | `configs/mcp-providers.yml`     |       |
| `MCP_AGENT_PROXY_ENABLED`      | bool     | `true`                          |       |

**Notes:**

- Search providers cascade in order: Serper -> Exa -> Tavily -> SearXNG. Each requires both its `*_ENABLED=true` flag and valid credentials.

### Media API

| Env Var                      | Type     | Default               | Notes  |
| ---------------------------- | -------- | --------------------- | ------ |
| `MEDIA_API_PORT`             | int      | `8285`                |        |
| `MEDIA_API_LOG_LEVEL`        | string   | `info`                |        |
| `MEDIA_MAX_UPLOAD_BYTES`     | int      | `20971520`            |        |
| `MEDIA_RETENTION_DAYS`       | int      | `30`                  |        |
| `MEDIA_PROXY_DOWNLOAD`       | bool     | `true`                |        |
| `MEDIA_REMOTE_FETCH_TIMEOUT` | duration | `15s`                 |        |
| `MEDIA_S3_ENDPOINT`          | string   | `https://s3.menlo.ai` |        |
| `MEDIA_S3_PUBLIC_ENDPOINT`   | string   | (empty)               |        |
| `MEDIA_S3_URL_ENABLED`       | bool     | `false`               |        |
| `MEDIA_S3_REGION`            | string   | `us-west-2`           |        |
| `MEDIA_S3_BUCKET`            | string   | `platform-dev`        |        |
| `MEDIA_S3_USE_PATH_STYLE`    | bool     | `true`                |        |
| `MEDIA_S3_PRESIGN_TTL`       | duration | `168h`                |        |
| `MEDIA_S3_ACCESS_KEY_ID`     | string   | (secret)              |        |
| `MEDIA_S3_SECRET_ACCESS_KEY` | string   | (secret)              |        |

### Response API

| Env Var                                   | Type     | Default                 | Notes |
| ----------------------------------------- | -------- | ----------------------- | ----- |
| `RESPONSE_API_PORT`                       | int      | `8082`                  |       |
| `RESPONSE_API_LOG_LEVEL`                  | string   | `info`                  |       |
| `RESPONSE_LLM_API_URL`                    | string   | `http://llm-api:8080`   |       |
| `RESPONSE_MCP_TOOLS_URL`                  | string   | `http://mcp-tools:8091` |       |
| `RESPONSE_MEDIA_API_URL`                  | string   | `http://media-api:8285` |       |
| `RESPONSE_MAX_TOOL_DEPTH`                 | int      | `8`                     |       |
| `TOOL_EXECUTION_TIMEOUT`                   | duration | `300s`                   |       |
| `RESPONSE_LLM_DISABLE_CUSTOM_TEMPERATURE` | bool     | `false`                 |       |
| `RESPONSE_LLM_STREAM_MODE`                | string   | `auto`                  |       |

## Monitoring

### OpenTelemetry

| Env Var                       | Type   | Default                      | Notes        |
| ----------------------------- | ------ | ---------------------------- | ------------ |
| `OTEL_ENABLED`                | bool   | `false`                      | All services |
| `OTEL_SERVICE_NAME`           | string | (per service)                | All services |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | string | `http://otel-collector:4318` | All services |
| `OTEL_HTTP_PORT`              | int    | `4318`                       | Infrastructure |
| `OTEL_GRPC_PORT`              | int    | `4317`                       | Infrastructure |

### Prometheus

| Env Var           | Type | Default | Notes          |
| ----------------- | ---- | ------- | -------------- |
| `PROMETHEUS_PORT` | int  | `9090`  | Infrastructure |

### Grafana

| Env Var                  | Type   | Default  | Notes          |
| ------------------------ | ------ | -------- | -------------- |
| `GRAFANA_PORT`           | int    | `3331`   | Infrastructure |
| `GRAFANA_ADMIN_USER`     | string | `admin`  | Infrastructure |
| `GRAFANA_ADMIN_PASSWORD` | string | (secret) | Infrastructure |

### Jaeger

| Env Var          | Type | Default | Notes          |
| ---------------- | ---- | ------- | -------------- |
| `JAEGER_UI_PORT` | int  | `16686` | Infrastructure |

## Inference

### vLLM

| Env Var                | Type   | Default                      | Notes          |
| ---------------------- | ------ | ---------------------------- | -------------- |
| `VLLM_ENABLED`         | bool   | `true`                       | Infrastructure |
| `VLLM_PORT`            | int    | `8001`                       |                |
| `VLLM_MODEL`           | string | `Qwen/Qwen2.5-0.5B-Instruct` | Infrastructure |
| `VLLM_SERVED_NAME`     | string | `qwen2.5-0.5b-instruct`      | Infrastructure |
| `VLLM_GPU_UTILIZATION` | float  | `0.66`                       | Infrastructure |

## See Also

- [Configuration Precedence](./precedence.md)
- [Configuration Types Reference](../../packages/go-common/config/types.go)
- [`.env.template`](../../.env.template) - The authoritative list of configurable values
