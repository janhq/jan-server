# Service Overview

Jan Server ships four core services plus shared infrastructure. Use this document to understand how they fit together and where to look in the codebase.

| Service | Purpose | Port(s) | Source | Primary Docs |
|---------|---------|---------|--------|--------------|
| **LLM API** | OpenAI-compatible chat completions, conversation storage, model management | 8080 (direct), 8000 via Kong | `services/llm-api` | [api/llm-api/README.md](api/llm-api/README.md), [api/llm-api/examples.md](api/llm-api/examples.md) |
| **Response API** | Multi-step orchestration, tool chaining, integration with MCP Tools | 8082 | `services/response-api` | [api/response-api/README.md](api/response-api/README.md) |
| **Media API** | Binary ingestion, jan_* IDs, S3 storage and resolution | 8285 | `services/media-api` | [api/media-api/README.md](api/media-api/README.md) |
| **MCP Tools** | Model Context Protocol tools (web search, scraping, file search, python exec) | 8091 | `services/mcp-tools` | [api/mcp-tools/README.md](api/mcp-tools/README.md), [services/mcp-tools/README.md](../services/mcp-tools/README.md) |

## Infrastructure Components
- **Kong Gateway (8000)**: exposes public APIs, enforces rate limits, validates Keycloak JWTs/API keys (custom plugin), and proxies `/llm/auth/guest-login` for guest tokens.
- **Keycloak (8085)**: handles OAuth2/OIDC flows; see `keycloak/`.
- **PostgreSQL**: `api-db` (LLM/Response/Media data) and `keycloak-db` (Keycloak state).
- **vLLM (8101)**: inference backend reachable from llm-api.
- **Observability stack**: Prometheus (9090), Grafana (3001), Jaeger (16686), OpenTelemetry Collector.
- **MCP support services**: SearXNG (search), Vector Store (file search), SandboxFusion (python execution).

## Creating a New Service
1. Copy `services/template-api` via `scripts/new-service-from-template.ps1 -Name my-service`.
2. Update `go.mod`, `cmd/server/main.go`, and `internal/config`.
3. Add your env vars to `.env.template`, `config/defaults.env`, and service README.
4. Extend `docs/services.md` and `docs/api/<service>/README.md` with new coverage.
5. Wire the service into:
   - `docker/services-api.yml` or a new compose file
   - `k8s/jan-server/values.yaml` (if deploying to Kubernetes)
   - `docs/INDEX.md` navigation

## Service Interactions
- **LLM API -> Media API**: LLM API resolves `jan_*` IDs before sending payloads to vLLM or upstream providers (`MEDIA_RESOLVE_URL` env var).
- **Response API -> LLM API**: Response API delegates final language generation to LLM API (`LLM_API_URL`).
- **Response API -> MCP Tools**: orchestrated tool calls are issued via JSON-RPC (`MCP_TOOLS_URL`).
- **MCP Tools -> Infrastructure**: uses SearXNG, Vector Store, and SandboxFusion to execute user requests.

Keep this document updated whenever a service is added, renamed, or retires.
