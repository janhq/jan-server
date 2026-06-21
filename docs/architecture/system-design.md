# System Design

This reference describes how the Jan Server components fit together. Use it when reviewing cross-service changes or planning deployments.

## 1. System Overview

Jan Server is a microservices platform that exposes OpenAI-compatible APIs through Kong. Each service owns a focused domain:

- **LLM API (8080)** - chat completions, conversations, projects, model catalog.
- **Response API (8082)** - multi-step orchestration and MCP tool coordination.
- **Media API (8285)** - binary ingestion, jan\_\* IDs, presigned URL management.
- **MCP Tools (8091)** - JSON-RPC endpoint that proxies web search (Serper -> Exa -> Tavily -> SearXNG fallback chain), scraping, file search, and SandboxFusion execution.
- **Shared infrastructure** - Kong (8000), Keycloak (8085), PostgreSQL (`api-db` 5432, `keycloak-db` 5433), vLLM (8101), observability stack.

Note: Template API (8185) is a development scaffold and not part of the deployed stack.

Kong terminates TLS (in production), validates JWT/API keys, applies rate limits, and forwards requests to the internal services.

## 2. Architecture Layers

| Layer             | Components                                                              | Notes                                                             |
| ----------------- | ----------------------------------------------------------------------- | ----------------------------------------------------------------- |
| **Edge**          | Kong Gateway, Keycloak                                                  | Centralized auth, rate limiting, guest-token endpoint.            |
| **Application**   | LLM API, Response API, Media API, MCP Tools                             | Written in Go using Gin + zerolog.                                |
| **Tooling**       | Serper, Exa, Tavily, SearXNG, SandboxFusion, vector-store               | Only accessible from MCP Tools.                                   |
| **Data/Storage**  | PostgreSQL (`api-db`, `keycloak-db`), S3-compatible storage             | Media files live in object storage; metadata lives in PostgreSQL. |
| **Inference**     | vLLM (local) or remote OpenAI-compatible providers                      | Selected per request using the provider metadata catalog.         |
| **Observability** | OpenTelemetry Collector, Prometheus, Grafana, Jaeger                    | Enabled with `OTEL_ENABLED=true` + `make monitor-up`.             |

## 3. Component Diagram

```
             +------------------------------+
             |  External Clients / SDKs     |
             +---------------+--------------+
                             |
                             v
                   +-------------------+
                   |   Kong Gateway    | 8000
                   +---+----+-----+----+
                       |    |     |
        +--------------+    |     +----------------+
        |                   |                      |
        v                   v                      v
  +-----------+     +---------------+       +---------------+
  |  LLM API  |     |  Response API |       |   Media API   |
  | (8080)    |     |    (8082)     |       |    (8285)     |
  +-----+-----+     +-------+-------+       +---------------+
        |                   |
        |                   v
        |          +-------------------+
        +--------->|    MCP Tools      |
                   |     (8091)        |
                   +----+---+----+-----+
                        |   |    |
                        |   |    +--> SandboxFusion
                        |   +-------> Vector Store
                        +-----------> Serper -> Exa -> Tavily -> SearXNG

Shared dependencies (not shown): PostgreSQL (api-db 5432, keycloak-db 5433), S3/Object storage, Keycloak (JWT issuer), vLLM (8101).
```

## 4. Request Lifecycles

### Chat Completions

1. Client calls `POST /v1/chat/completions` on `http://localhost:8000`.
2. Kong validates the JWT/API key and forwards to `llm-api:8080`.
3. LLM API resolves `jan_*` placeholders via Media API, selects a provider (local vLLM or remote), and streams tokens back to the gateway.
4. Conversations/projects are persisted in PostgreSQL.

### Response Orchestration

1. Client calls `POST /responses/v1/responses` (streaming optional).
2. Response API loads the conversation context and iteratively issues `tools/list` / `tools/call` requests to MCP Tools.
3. Tool executions are capped by `RESPONSE_MAX_TOOL_DEPTH` and `TOOL_EXECUTION_TIMEOUT`.
4. Final synthesis is delegated to LLM API and streamed back to the caller.

### Media Handling

1. Upload via `POST /media/v1/media` (remote URL or data URL) or request a presigned upload with `POST /media/v1/media/prepare-upload`.
2. Media API deduplicates content, issues a `jan_*` ID, and stores metadata in PostgreSQL.
3. Other services embed the `jan_*` ID; LLM API resolves them to presigned URLs right before inference.

### MCP JSON-RPC

1. Response API or external automation sends JSON-RPC requests to `POST /v1/mcp`.
2. MCP Tools validates the method (`tools/list`, `tools/call`, `prompts/*`, `resources/*`) and dispatches to the search (Serper/Exa/Tavily/SearXNG), scrape, and SandboxFusion clients.
3. Results are returned as SSE events (streaming) or plain JSON when the response fits a single chunk.

## 5. Data & Network Topology

- Docker Compose defines two primary networks: `jan-server_default` (Kong + core services + databases) and `jan-server_mcp-network` (MCP-only helpers such as SearXNG, vector store, SandboxFusion).
- Production deployments should mirror this split using Kubernetes namespaces or NetworkPolicies.
- Persistent data:
  - `api-db` (LLM/Response/Media metadata) - each service uses its own schema.
  - `keycloak-db` - Keycloak realm and client configuration.
  - Object storage (S3, MinIO, etc.) - Media files and presigned URLs.

## 6. Deployment Modes

| Mode                    | Description                                                                                             | Commands                                           |
| ----------------------- | ------------------------------------------------------------------------------------------------------- | -------------------------------------------------- |
| **Local (recommended)** | `make quickstart` prompts for providers, writes `.env`, and runs `docker compose up` with all services. | `make quickstart`                                  |
| **Profiles**            | Start a subset of services (API only, MCP only, GPU inference).                                         | `make up-api`, `make up-mcp`, `make up-gpu`        |
| **Monitoring stack**    | Optional Prometheus/Grafana/Jaeger.                                                                     | `make monitor-up`                                  |
| **Kubernetes**          | Use `infra/k8s/jan-server` Helm chart. Values mirror `packages/go-common/config` defaults.                                   | `helm install jan ./infra/k8s/jan-server -f values.yaml` |

## 7. Change Impact Checklist

When modifying the system architecture:

1. Update the relevant service README and API docs.
2. Reflect new ports/paths in Kong configuration.
3. Adjust `docs/architecture/services.md` and `docs/architecture/data-flow.md`.
4. Regenerate configuration artifacts (`make config-generate`) if shared config changes.
5. Update Kubernetes values and Helm defaults as needed.
