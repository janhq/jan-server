# Data Flow Reference

## 1. Chat Completion (LLM API)
1. **Client** calls Kong Gateway `POST /v1/chat/completions` with Bearer token.
2. **Kong** forwards to `llm-api:8080` (internal DNS) and injects request headers.
3. **LLM API**:
   - Validates JWT via Keycloak JWKS.
   - Resolves any `jan_*` media IDs by calling Media API `/v1/media/resolve`.
   - Selects a provider (local vLLM or configured upstream) and forwards the request.
4. **Provider** (vLLM) streams tokens back to LLM API.
5. **LLM API** streams data to the client (SSE) via Kong and persists conversation rows in PostgreSQL.

## 2. Response API Orchestration
1. **Client** calls `POST /v1/responses`.
2. **Response API** looks up conversation state and enqueues tool steps.
3. For each tool call:
   - Executes JSON-RPC request against MCP Tools (`/v1/mcp`).
   - Records execution metadata in PostgreSQL.
   - Applies depth/timeout limits (`MAX_TOOL_EXECUTION_DEPTH`, `TOOL_EXECUTION_TIMEOUT`).
4. Final synthesis request is sent to LLM API.
5. Completed response is stored and returned to the caller (supporting streaming in future releases).

## 3. Media Upload and Resolution
1. Client uploads via:
   - `POST /v1/media` (server-proxied, data URL or remote fetch), or
   - `POST /v1/media/prepare-upload` followed by direct S3 upload.
2. Media API stores metadata rows and issues `jan_<snowflake>` IDs.
3. Other services reference those IDs instead of exposing raw S3 URLs.
4. Before inference, LLM API calls `/v1/media/resolve` with the request payload; Media API rewrites each placeholder with a fresh presigned URL.

## 4. MCP Tool Execution
1. Response API or external clients send MCP JSON-RPC requests to `mcp-tools:8091`.
2. MCP Tools selects the proper backend:
   - Web search -> Serper or SearXNG (via redis-searxng cache)
   - Scrape -> HTTP fetcher with metadata
   - File search -> vector-store service
   - Python exec -> SandboxFusion container
3. Results are returned synchronously; streaming support is planned via incremental notifications.

## 5. Observability Pipeline
1. Services emit traces and metrics via OTLP (4317).
2. The OpenTelemetry Collector forwards metrics to Prometheus and traces to Jaeger.
3. Logs are structured JSON printed to stdout; Docker/ Kubernetes aggregates them for your logging stack.
4. Grafana dashboards connect to Prometheus and Jaeger for live inspection.

Use this file when onboarding engineers or mapping changes that span multiple services.
