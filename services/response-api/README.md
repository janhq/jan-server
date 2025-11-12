# response-api

`response-api` is the Jan Responses API microservice. It follows the OpenAI Responses contract, orchestrates multi-step tool calls through `mcp-tools`, and delegates language generation to `llm-api`.

Key capabilities:

- Environment-driven config with sensible defaults (see `internal/config`).
- Structured Zerolog logging plus optional OTEL tracing.
- PostgreSQL persistence for responses, conversations, and tool executions (GORM).
- JSON-RPC integration with `services/mcp-tools` for tool discovery/calls.
- HTTP client for `services/llm-api` chat completions.
- Gin HTTP server exposing `/v1/responses` CRUD plus SSE streaming stub.
- Optional Keycloak/OIDC JWT enforcement.
- Wire-ready DI entrypoint, Dockerfile, Makefile, and example env file.

## Quick start

```bash
# From repo root
make env-create            # populates .env from .env.template

cd services/response-api
go mod tidy
make run

# Smoke check
curl http://localhost:8082/healthz
curl -X POST http://localhost:8082/v1/responses \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","input":"ping"}'

```

Useful targets:

- `make wire` – regenerate DI after editing `cmd/server/wire.go`.
- `make swagger` – regenerate OpenAPI docs from annotations.
- `make test` – unit/integration test suite.

## Configuration

| Variable | Description | Default |
| --- | --- | --- |
| `SERVICE_NAME` | Logical service name | `response-api` |
| `HTTP_PORT` | HTTP listen port | `8082` |
| `RESPONSE_DATABASE_URL` | PostgreSQL DSN | `postgres://postgres:postgres@localhost:5432/response_api?sslmode=disable` |
| `LLM_API_URL` | Base URL for `llm-api` | `http://localhost:8080` |
| `MCP_TOOLS_URL` | Base URL for `mcp-tools` | `http://localhost:8091` |
| `MAX_TOOL_EXECUTION_DEPTH` | Max recursive tool chain depth | `8` |
| `TOOL_EXECUTION_TIMEOUT` | Per-tool call timeout | `45s` |
| `AUTH_ENABLED` + `AUTH_*` | Toggle and configure OIDC validation | disabled |

See `.env.template` in the repo root for the full list including tracing/logging knobs.

## Database

On startup the service runs migrations for:

- `responses`
- `conversations`
- `conversation_items`
- `tool_executions`

Each table uses JSONB columns for flexible payload storage. Point `RESPONSE_DATABASE_URL` at your cluster before starting the service.

## Authentication

- Set `AUTH_ENABLED=true` to enforce Bearer tokens. Provide `AUTH_ISSUER`, `AUTH_AUDIENCE`, and `AUTH_JWKS_URL`.
- With auth disabled the service treats callers as `guest` unless a `user` field is provided in the request body.
