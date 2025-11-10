# Service Architecture Details

## LLM API
- **Language**: Go (Gin, Wire DI, GORM)
- **Responsibilities**:
  - OpenAI-compatible `/v1/chat/completions`, `/v1/conversations`, `/v1/models`
  - Token verification using Keycloak JWKS
  - Media resolution via `MEDIA_RESOLVE_URL`
  - Provider abstraction for vLLM, OpenAI, Anthropic (configured via env vars)
- **Data stores**:
  - PostgreSQL (`jan_llm_api`)
  - Optional Redis (provider specific caches, future use)
- **Observability**:
  - Structured logging (zerolog)
  - OpenTelemetry exporter (OTLP gRPC)

## Response API
- **Purpose**: Implements the OpenAI Responses contract with tool orchestration.
- **Workflow**:
  1. Validates request and conversation context.
  2. Executes tool calls recursively via MCP Tools (`MAX_TOOL_EXECUTION_DEPTH`, `TOOL_EXECUTION_TIMEOUT`).
  3. Invokes LLM API for synthesis.
  4. Streams partial results (SSE roadmap) and stores final response rows.
- **State**: PostgreSQL (`response_api` database) storing responses, tool executions, and metadata.

## Media API
- **Purpose**: Central media ingestion and jan_id registry.
- **Key features**:
  - Accepts remote URLs, base64 payloads, or presigned upload reservations.
  - Stores metadata plus S3 object references.
  - Issues `jan_*` IDs with deterministic prefixes.
  - Resolves IDs to S3 presigned URLs before handing payloads to LLM services.
- **Dependencies**:
  - PostgreSQL (`media_api`)
  - S3-compatible object storage (configurable endpoint, bucket, credentials)

## MCP Tools
- **Purpose**: Implements the Model Context Protocol for tool calling.
- **Tools**:
  - `google_search` (Serper or SearXNG backend)
  - `scrape` web fetcher
  - `file_search_index` + `file_search_query` (vector store helper)
  - `python_exec` (SandboxFusion)
- **Runtime**:
  - Go HTTP server exposing MCP JSON-RPC endpoints at `/v1/mcp`
  - Connects to helper services on the dedicated MCP Docker network.

## Shared Infrastructure
- **Kong Gateway**: Exposes `/v1/*` endpoints, handles rate limiting, request IDs, auth enforcement.
- **Keycloak**: Single realm (`jan`) with admin console for managing users and clients.
- **PostgreSQL clusters**:
  - `api-db`: multi-schema database for LLM/Response/Media
  - `keycloak-db`: Keycloak persistence
- **vLLM**: Hosted inference backend accessible via API key.
- **Observability stack**:
  - OpenTelemetry Collector (4317)
  - Prometheus (9090)
  - Jaeger (16686)
  - Grafana (3001)

Update this file whenever a service gains new dependencies or exposes additional ports.
