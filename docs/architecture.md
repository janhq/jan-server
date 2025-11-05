# Jan Server Architecture

## System Overview

Jan Server is a modular, microservices-based LLM API platform with enterprise-grade authentication, API gateway routing, and flexible inference backend support. The system provides OpenAI-compatible API endpoints for chat completions, conversations, and model management.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLIENT LAYER                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                      │
│  │   Web App    │  │  Mobile App  │  │  CLI Client  │                      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘                      │
└─────────┼──────────────────┼──────────────────┼────────────────────────────┘
          │                  │                  │
          │ HTTP/SSE         │ HTTP/SSE         │ HTTP/SSE
          │ Port 8000        │ Port 8000        │ Port 8000
          │                  │                  │
          └──────────────────┴──────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          API GATEWAY LAYER                                   │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                          KONG API Gateway                              │  │
│  │  • Declarative Config (kong.yml)                                      │  │
│  │  • Services:                                                           │  │
│  │    - llm-api-svc → http://host.docker.internal:8080                  │  │
│  │    - mcp-tools-svc → http://host.docker.internal:8091                │  │
│  │  • Routes:                                                             │  │
│  │    - /v1/* (excl. /v1/mcp) → llm-api-svc                             │  │
│  │    - /v1/mcp → mcp-tools-svc                                          │  │
│  │    - /auth → llm-api-svc                                              │  │
│  │  • Plugins:                                                            │  │
│  │    - CORS (with Mcp-Session-Id, mcp-protocol-version headers)        │  │
│  │  • Port: 8000                                                          │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                    │                              │
                    ▼                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         APPLICATION LAYER                                    │
│                                                                              │
│  ┌─────────────────────────────────────┐   ┌────────────────────────────┐  │
│  │         LLM-API Service             │   │     MCP-Tools Service      │  │
│  │  (Port: 8080, Internal)             │   │  (Port: 8091, Internal)    │  │
│  │                                     │   │                            │  │
│  │  • REST API (Gin Framework)        │   │  • MCP Protocol (HTTP)     │  │
│  │  • OpenAPI/Swagger                  │   │  • Stateless mode          │  │
│  │  • Authentication:                  │   │  • No authentication       │  │
│  │    - JWT (Keycloak JWKS)           │   │    (internal service)      │  │
│  │    - API Key (Kong consumer)       │   │                            │  │
│  │  • Middleware:                      │   │  MCP Tools:                │  │
│  │    - Auth                           │   │  • google_search           │  │
│  │    - Request ID                     │   │    (Serper API)            │  │
│  │    - SSE Support                    │   │  • scrape                  │  │
│  │  • Idempotency Store                │   │    (Web scraping)          │  │
│  │  • OpenTelemetry Integration        │   │                            │  │
│  │                                     │   │  Endpoint:                 │  │
│  │  Endpoints:                         │   │  POST /v1/mcp              │  │
│  │  • GET  /v1/models                  │   │                            │  │
│  │  • GET  /v1/models/:id              │   │  Allowed Methods:          │  │
│  │  • POST /v1/chat/completions        │   │  • initialize              │  │
│  │  • POST /v1/completions             │   │  • ping                    │  │
│  │  • POST /v1/conversations           │   │  • tools/list              │  │
│  │  • GET  /v1/conversations           │   │  • tools/call              │  │
│  │  • GET  /v1/conversations/:id       │   │                            │  │
│  │  • POST /v1/conversations/:id/msgs  │   │  Health:                   │  │
│  │  • GET  /v1/conversations/:id/msgs  │   │  GET /healthz, /readyz     │  │
│  │  • POST /v1/conversations/:id/runs  │   └────────────────────────────┘  │
│  │  • POST /v1/responses               │              │                     │
│  │  • POST /auth/guest                 │              │                     │
│  │  • POST /auth/upgrade               │              ▼                     │
│  └─────────────────────────────────────┘   ┌────────────────────────────┐  │
│             │              │                │   Serper Service           │  │
│             │              │                │  • Search API client       │  │
│             │              └──────────┐     │  • Scrape API client       │  │
│             ▼                         ▼     └────────────────────────────┘  │
│  ┌──────────────────────┐  ┌─────────────────────────┐                     │
│  │  Provider Registry   │  │   Repository Layer      │                     │
│  │                      │  │                         │                     │
│  │  • providers.yaml    │  │  • ModelRepository      │                     │
│  │  • Default: vllm     │  │  • ConversationRepo     │                     │
│  │  • Model routing     │  │  • MessageRepository    │                     │
│  │  • Capability flags  │  │  • GORM ORM             │                     │
│  └──────────────────────┘  └─────────────────────────┘                     │
│             │                          │                                     │
└─────────────┼──────────────────────────┼─────────────────────────────────────┘
              │                          │
              ▼                          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        INFERENCE LAYER                                       │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                        vLLM Inference Server                           │  │
│  │  (Port: 8000, Internal)                                                │  │
│  │                                                                         │  │
│  │  • OpenAI-Compatible API                                               │  │
│  │  • Model Profiles:                                                     │  │
│  │    - GPU: vllm-llama (AWQ quantization, default)                      │  │
│  │    - CPU: vllm-cpu (bfloat16)                                         │  │
│  │  • Default Model: Qwen2.5-3B-Instruct-AWQ / jan-v1-4b                 │  │
│  │  • Features:                                                           │  │
│  │    - Auto tool calling                                                 │  │
│  │    - KV cache optimization                                             │  │
│  │    - Token streaming                                                   │  │
│  │  • Auth: Bearer token (VLLM_INTERNAL_KEY)                             │  │
│  │  • Volume: HuggingFace model cache                                     │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                         AUTHENTICATION LAYER                                 │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                         Keycloak (Port: 8085)                          │  │
│  │                                                                         │  │
│  │  Realm: jan                                                            │  │
│  │  Clients:                                                               │  │
│  │  • backend (service account)                                           │  │
│  │    - Client secret auth                                                │  │
│  │    - Token exchange enabled                                            │  │
│  │    - Guest user creation                                               │  │
│  │  • llm-api (public client)                                             │  │
│  │    - Direct access grants                                              │  │
│  │    - Standard flow                                                     │  │
│  │    - Custom claims: preferred_username, guest flag                    │  │
│  │                                                                         │  │
│  │  Roles:                                                                 │  │
│  │  • guest (temporary access)                                            │  │
│  │  • user (upgraded accounts)                                            │  │
│  │                                                                         │  │
│  │  JWKS Endpoint: /realms/jan/protocol/openid-connect/certs             │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                               │                                              │
│                               ▼                                              │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                   Keycloak DB (PostgreSQL 16)                          │  │
│  │  • User identities                                                     │  │
│  │  • Client configurations                                               │  │
│  │  • Sessions & tokens                                                   │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                          PERSISTENCE LAYER                                   │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                     API Database (PostgreSQL 16)                       │  │
│  │                                                                         │  │
│  │  Tables:                                                                │  │
│  │  • conversations (id, user_id, title, metadata, timestamps)           │  │
│  │  • messages (id, conversation_id, role, content, timestamps)          │  │
│  │  • models (id, provider, display_name, capabilities)                  │  │
│  │                                                                         │  │
│  │  Managed by:                                                            │  │
│  │  • GORM ORM (application)                                              │  │
│  │  • golang-migrate (schema migrations)                                  │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                        OBSERVABILITY LAYER                                   │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │           OpenTelemetry Collector (Port: 4318/4317)                    │  │
│  │  • Traces, metrics, logs collection                                    │  │
│  │  • OTLP HTTP (4318) and gRPC (4317) receivers                         │  │
│  │  • Exporters: Prometheus, Jaeger, Console                             │  │
│  │  • Connected to llm-api telemetry                                      │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                    │                                  │                       │
│                    ▼                                  ▼                       │
│  ┌────────────────────────────┐      ┌──────────────────────────────────┐  │
│  │  Prometheus (Port: 9090)   │      │   Jaeger (Port: 16686)           │  │
│  │  • Metrics storage         │      │   • Distributed tracing          │  │
│  │  • Time-series database    │      │   • Trace visualization          │  │
│  │  • PromQL queries          │      │   • Service dependency graph     │  │
│  │  • 15s scrape interval     │      │   • Performance analysis         │  │
│  └────────────────────────────┘      └──────────────────────────────────┘  │
│                    │                                  │                       │
│                    └──────────────┬───────────────────┘                       │
│                                   ▼                                           │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    Grafana (Port: 3001)                                │  │
│  │  • Unified dashboards for metrics and traces                          │  │
│  │  • Pre-configured datasources (Prometheus, Jaeger)                    │  │
│  │  • Custom dashboards for Jan Server services                          │  │
│  │  • Alerting and notifications                                          │  │
│  │  • Default credentials: admin/admin                                    │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Request Flow Patterns

### Pattern 1: Chat Completion (Authenticated User)

```
Client
  │
  │ POST /v1/chat/completions
  │ Headers: Authorization: Bearer <JWT>
  │ Body: {model, messages, stream: true}
  │
  ▼
Kong Gateway
  │ ✓ CORS check
  │ ✓ Route to llm-api-svc
  │
  ▼
LLM-API Service
  │ ✓ JWT validation (JWKS from Keycloak)
  │ ✓ Extract principal (user_id, scopes)
  │ ✓ Idempotency check (Idempotency-Key header)
  │ ✓ Model resolution (providers.yaml)
  │
  ▼
vLLM Server
  │ POST /v1/chat/completions
  │ Headers: Authorization: Bearer <VLLM_INTERNAL_KEY>
  │
  ▼
Response (SSE Stream)
  │ data: {id, choices[{delta}], ...}
  │ data: {id, choices[{delta}], ...}
  │ data: [DONE]
  │
  ▼
LLM-API
  │ ✓ Store idempotency result
  │ ✓ Emit telemetry
  │
  ▼
Client (streaming response)
```

### Pattern 2: MCP Tool Execution

```
Client (MCP Client)
  │
  │ POST /v1/mcp
  │ Body: {jsonrpc: "2.0", method: "tools/call", params: {name: "google_search", arguments: {q: "..."}}}
  │
  ▼
Kong Gateway
  │ ✓ CORS check (allows Mcp-Session-Id, mcp-protocol-version)
  │ ✓ Route to mcp-tools-svc
  │
  ▼
MCP-Tools Service
  │ ✓ Method guard (validate allowed MCP method)
  │ ✓ Parse MCP request
  │
  ▼
MCP Server (mark3labs/mcp-go)
  │ ✓ Stateless mode (no session management)
  │ ✓ Route to tool handler
  │
  ▼
Tool Handler (e.g., google_search)
  │ ✓ Extract arguments
  │ ✓ Call Serper API
  │
  ▼
Serper API (External)
  │ HTTP Request to api.serper.dev
  │
  ▼
Response (MCP JSON-RPC)
  │ {jsonrpc: "2.0", id: ..., result: {content: [{type: "text", text: "..."}]}}
  │
  ▼
Client
```

### Pattern 3: Guest User Creation

```
Client
  │
  │ POST /auth/guest
  │
  ▼
Kong Gateway
  │ ✓ Route to llm-api-svc
  │
  ▼
LLM-API (/auth endpoints on Port 8080)
  │
  ▼
Keycloak
  │ 1. Service account login (backend client)
  │ 2. Create user with guest=true attribute
  │ 3. Assign 'guest' role
  │ 4. Token exchange to user token
  │
  ▼
Response
  │ {access_token, refresh_token, expires_in}
  │
  ▼
Client (can now call /v1/chat/completions with JWT)
```

### Pattern 4: Conversation Management

```
Client
  │ POST /v1/conversations
  │ Headers: Authorization: Bearer <JWT>
  │ Body: {title, metadata}
  │
  ▼
Kong → LLM-API
  │ ✓ Auth
  │ ✓ Extract principal
  │
  ▼
ConversationRepository
  │ INSERT into conversations
  │
  ▼
Response: {id, user_id, title, created_at}

─── Later ───

Client
  │ POST /v1/conversations/:id/messages
  │ Body: {role: "user", content: "..."}
  │
  ▼
LLM-API
  │ MessageRepository.Create()
  │
  ▼
Response: {message_id, conversation_id, ...}

─── Then ───

Client
  │ POST /v1/conversations/:id/runs
  │
  ▼
LLM-API
  │ 1. Fetch all messages in conversation
  │ 2. Format as chat completion request
  │ 3. Call vLLM
  │ 4. Store assistant response as new message
  │
  ▼
Response (SSE or JSON)
```

---

## Component Details

### Kong API Gateway
- **Image**: `kong:3.5`
- **Config**: Declarative (`kong.yml`)
- **Services**:
  - `llm-api-svc` → `http://host.docker.internal:8080`
  - `mcp-tools-svc` → `http://host.docker.internal:8091`
- **Routes**: 
  - `/v1/*` (except `/v1/mcp`) → llm-api-svc
  - `/v1/mcp` → mcp-tools-svc
  - `/auth` → llm-api-svc
  - Health checks → respective services
- **Plugins**:
  - `cors`: Allows cross-origin requests with MCP-specific headers (Mcp-Session-Id, mcp-protocol-version)
- **Port**: 8000 (exposed)

### LLM-API Service
- **Language**: Go
- **Framework**: Gin (HTTP), GORM (ORM)
- **Port**: 8080 (internal)
- **Dependencies**:
  - PostgreSQL (api-db)
  - Keycloak (JWT validation)
  - vLLM (inference)
  - OpenTelemetry Collector (traces/metrics)
- **Key Features**:
  - Dual auth: JWT (Keycloak JWKS) or API Key (Kong consumer)
  - Idempotency support for POST requests
  - SSE streaming for chat completions
  - Provider abstraction (supports multiple backends)
  - Conversation & message persistence
  - Embedded database migrations applied on startup
  - Guest authentication endpoints

### MCP-Tools Service
- **Language**: Go
- **Framework**: Gin (HTTP server), mcp-go (MCP protocol)
- **Port**: 8091 (internal)
- **Protocol**: Model Context Protocol (MCP) over HTTP
- **Mode**: Stateless (no session management required)
- **Dependencies**:
  - Serper API (external search service)
- **Architecture**: Clean Architecture
  - Domain: Business logic (SerperService)
  - Infrastructure: External clients (SerperClient)
  - Interfaces: HTTP routes, MCP handlers
  - Utils: Error handling, MCP utilities
- **Tools Available**:
  - `google_search`: Web search via Serper API
    - Parameters: q (query), gl (region), hl (language), location, num, tbs, page, autocorrect
  - `scrape`: Web page scraping
    - Parameters: url, includeMarkdown
- **MCP Methods Supported**:
  - `initialize`: MCP handshake
  - `ping`: Health check
  - `tools/list`: List available tools
  - `tools/call`: Execute a tool
- **Key Features**:
  - No authentication (internal service)
  - Method guard middleware
  - Enhanced error logging
  - CORS support for MCP headers

### Guest Authentication (within llm-api)
- **Language**: Go (part of llm-api binary)
- **Framework**: Gin
- **Port**: 8080 (exposed)
- **Purpose**: Guest user lifecycle management
- **Endpoints**:
  - `POST /auth/guest`: Create guest user, return JWT
  - `POST /auth/upgrade`: Convert guest to permanent account
- **Integration**: Keycloak Admin API & Token Exchange (through embedded client)

### Keycloak
- **Image**: Custom Dockerfile (based on official Keycloak)
- **Port**: 8085 (exposed)
- **Realm**: `jan`
- **Init Script**: `enable-token-exchange.sh` (runs on startup)
- **Clients**:
  - `backend`: Service account used by llm-api guest provisioning flows
  - `llm-api`: Public client for user authentication
- **Roles**: `guest`, `user`
- **Custom Claims**: `preferred_username`, `guest` (boolean)

### vLLM Inference
- **Image**: `vllm/vllm-openai:v0.10.1` (GPU) / `vllm/vllm-openai:latest` (CPU)
- **Port**: 8000 (internal)
- **Profiles**:
  - `gpu`: Default AWQ quantized model (Qwen2.5-3B-Instruct-AWQ)
  - `cpu`: Fallback model (janhq/Jan-v1-4b)
- **Auth**: Bearer token (`VLLM_INTERNAL_KEY`)
- **Volume**: HuggingFace cache mounted to `/root/.cache/huggingface`
- **Environment**: Requires `HF_TOKEN` for model downloads

### Databases
- **API DB** (PostgreSQL 16):
  - Volume: `api-db-data`
  - Schema: conversations, messages, models
  - Migrations: embedded SQL migrations applied by llm-api
- **Keycloak DB** (PostgreSQL 16):
  - Volume: `keycloak-db-data`
  - Managed by Keycloak

### OpenTelemetry Collector
- **Image**: `otel/opentelemetry-collector-contrib:0.90.1`
- **Ports**: 
  - 4318 (OTLP HTTP receiver)
  - 4317 (OTLP gRPC receiver)
  - 8889 (Prometheus metrics exporter)
- **Config**: `docs/otel-collector.yaml`
- **Purpose**: Centralized telemetry collection from llm-api
- **Exporters**:
  - **Prometheus**: Metrics at `:8889/metrics`
  - **OTLP (Jaeger)**: Traces to Jaeger via OTLP gRPC (port 4317)
  - **Logging**: Console output for debugging

### Prometheus
- **Image**: `prom/prometheus:v2.48.0`
- **Port**: 9090 (exposed)
- **Config**: `docs/prometheus.yml`
- **Purpose**: Metrics storage and querying
- **Scrape Targets**:
  - otel-collector:8889 (OpenTelemetry metrics)
  - llm-api:8080/metrics (if exposed)
  - mcp-tools:8091/metrics (if exposed)
- **Storage**: Persistent volume `prometheus-data`
- **Features**:
  - Time-series database
  - PromQL query language
  - 15s scrape interval

### Jaeger
- **Image**: `jaegertracing/all-in-one:1.51`
- **Port**: 16686 (UI, exposed)
- **Purpose**: Distributed tracing backend and UI
- **Features**:
  - Trace collection via OTLP
  - Service dependency graph
  - Trace search and visualization
  - Performance analysis
  - Root cause analysis

### Grafana
- **Image**: `grafana/grafana:10.2.2`
- **Port**: 3001 (exposed, mapped from internal 3000)
- **Purpose**: Unified observability dashboard
- **Default Credentials**: admin/admin (configurable via env)
- **Datasources** (auto-provisioned):
  - Prometheus (metrics)
  - Jaeger (traces)
- **Storage**: Persistent volume `grafana-data`
- **Features**:
  - Custom dashboards for Jan Server
  - Metrics visualization
  - Trace correlation
  - Alerting
  - Dashboard provisioning

---

## Configuration Management

### Environment Variables (.env)
```bash
# Database
POSTGRES_USER=jan_user
POSTGRES_PASSWORD=<secret>
POSTGRES_DB=jan_llm_api
DATABASE_URL=postgres://jan_user:<secret>@api-db:5432/jan_llm_api

# LLM-API
HTTP_PORT=8080
LOG_LEVEL=info
LOG_FORMAT=json
AUTO_MIGRATE=true

# MCP-Tools
MCP_TOOLS_HTTP_PORT=8091
SERPER_API_KEY=<your-serper-api-key>

# Keycloak
KEYCLOAK_BASE_URL=http://keycloak:8080
KEYCLOAK_REALM=jan
KEYCLOAK_HTTP_PORT=8085
KEYCLOAK_ADMIN=admin
KEYCLOAK_ADMIN_PASSWORD=<secret>

# Guest Provisioning (handled by llm-api)
BACKEND_CLIENT_ID=backend
BACKEND_CLIENT_SECRET=backend-secret
TARGET_CLIENT_ID=llm-api
GUEST_ROLE=guest

# vLLM
VLLM_MODEL=Qwen2.5-3B-Instruct-AWQ
VLLM_SERVED_NAME=qwen2.5-3b-awq
VLLM_INTERNAL_KEY=changeme
VLLM_GPU_UTIL=0.95
VLLM_MAX_LEN=512
HF_TOKEN=<huggingface-token>

# OpenTelemetry
OTEL_ENABLED=false
OTEL_SERVICE_NAME=llm-api
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318

# Observability Stack
PROMETHEUS_PORT=9090
JAEGER_UI_PORT=16686
GRAFANA_PORT=3001
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=<secret>
```

### Provider Configuration (providers.yaml)
```yaml
providers:
  - name: vllm-local
    kind: openai
    base_url: http://vllm-llama:8000
    headers:
      Authorization: "Bearer ${VLLM_INTERNAL_KEY}"
    models:
      - id: jan-v1-4b
        served_name: ${VLLM_SERVED_NAME}
        capabilities: [chat, completions, embeddings]
routing:
  default_provider: vllm-local
```

---

## Data Models

### Domain Entities (GORM)

**Conversation**
```go
type Conversation struct {
    ID        string    `gorm:"primaryKey"`
    UserID    string    `gorm:"index"`
    Title     string
    Metadata  JSON      // Arbitrary JSON
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

**Message**
```go
type Message struct {
    ID             string `gorm:"primaryKey"`
    ConversationID string `gorm:"index"`
    Role           string // "user", "assistant", "system"
    Content        string
    CreatedAt      time.Time
}
```

**Model**
```go
type Model struct {
    ID           string   `gorm:"primaryKey"`
    Provider     string
    DisplayName  string
    Capabilities []string `gorm:"type:text[]"` // PostgreSQL array
}
```

---

## Authentication & Authorization

### Auth Methods

1. **JWT (Keycloak)**:
   - Validated using JWKS endpoint
   - Claims: `sub` (user_id), `preferred_username`, `guest`, `realm_access.roles`
   - Principal built from JWT claims
   - Response header: `X-Auth-Method: jwt`

2. **API Key (Kong)**:
   - Validated by Kong key-auth plugin
   - Kong injects `X-Consumer-Username`, `X-Consumer-ID`
   - Principal built from consumer headers
   - Response header: `X-Auth-Method: api_key`

### Principal Propagation
```go
type Principal struct {
    ID       string
    Username string
    Scopes   []string
    IsGuest  bool
}
```
Headers injected by llm-api:
- `X-Principal-Id`
- `X-Auth-Method`
- `X-Scopes` (space-separated)

---

## Deployment Profiles

Docker Compose services are organized by profiles for flexible deployment:

- **Infrastructure only**: `make up` (api-db, keycloak-db, keycloak)
- **With LLM API**: `make up-llm-api` (+ llm-api service)
- **With Kong**: `make up-kong` (+ kong gateway)
- **Full stack**: `make up-full` (all services)
- **GPU inference**: `make up-gpu` (+ vllm with GPU)
- **CPU inference**: `make up-cpu` (+ vllm CPU-only)
- **Monitoring stack**: `make monitor-up` (prometheus, jaeger, grafana, otel-collector)

The monitoring stack is completely separate in `docker-compose.monitor.yml` and can be started/stopped independently.

### Full Stack + GPU
```bash
make up-gpu
# Starts: api-db, llm-api, kong, keycloak, keycloak-db, vllm-llama, mcp-tools
```

### Full Stack + CPU
```bash
make up-cpu
# Same as GPU but uses vllm-cpu profile (no GPU requirements)
```

### With Observability
```bash
make monitor-up
# Starts: prometheus, jaeger, grafana, otel-collector

# Or manually:
docker compose -f docker-compose.monitor.yml up -d
```

### Stop Observability
```bash
make monitor-down
# Stops monitoring stack, keeps data volumes

make monitor-down-v
# Stops and removes data volumes (fresh start)
```

### Inference Only (GPU)
```bash
make up-gpu-only
# Starts: vllm-llama only (for development/testing)
```

### Inference Only (CPU)
```bash
make up-cpu-only
# Starts: vllm-cpu only
```

---

## Network Topology

### Internal Services (Docker Network)
- `api-db:5432` (PostgreSQL)
- `llm-api:8080` (LLM API Service)
- `mcp-tools:8091` (MCP Tools Service)
- `keycloak-db:5432` (PostgreSQL)
- `keycloak:8080` (Keycloak)
- `vllm-llama:8000` (vLLM Inference)
- `otel-collector:4318/4317` (OTLP HTTP/gRPC)
- `prometheus:9090` (Metrics storage)
- `jaeger:16686` (Trace backend)
- `grafana:3000` (Dashboards)

### Exposed Ports
- `8000` -> Kong Gateway (public API)
- `8080` -> LLM API (for direct access if needed)
- `8085` -> Keycloak Admin Console
- `8091` -> MCP Tools (for direct MCP access if needed)
- `9090` -> Prometheus UI
- `16686` -> Jaeger UI
- `3001` -> Grafana (dashboards)

---

## Security Considerations

1. **Secrets Management**:
   - All sensitive values in `.env` (gitignored)
   - Keycloak client secrets
   - Database passwords
   - vLLM internal API key
   - HuggingFace tokens

2. **Network Isolation**:
   - Internal services communicate via Docker network
   - Only Kong exposed externally (all API traffic goes through gateway)
   - MCP-Tools has no authentication (internal service only)

3. **Authentication Layers**:
   - Kong: CORS and routing (no key-auth in current setup)
   - LLM-API: JWT validation (required for user data)
   - MCP-Tools: No authentication (stateless, internal only)
   - vLLM: Internal bearer token

4. **CORS**:
   - Configured in Kong plugin
   - Allows all origins in development (should be restricted in production)

---

## Observability

### Complete Observability Stack
The platform includes a full observability stack with metrics, traces, and visualization:

- **OpenTelemetry Collector**: Receives telemetry from services
- **Prometheus**: Stores and queries metrics
- **Jaeger**: Distributed tracing backend
- **Grafana**: Unified dashboard for metrics and traces

### Metrics & Traces
- **OpenTelemetry** integration in llm-api
- Metrics and traces sent to `otel-collector:4318` (HTTP) or `otel-collector:4317` (gRPC)
- **Prometheus** scrapes metrics from:
  - OpenTelemetry Collector (`:8889/metrics`)
  - Services with Prometheus endpoints
- **Jaeger** receives traces via OTLP from OpenTelemetry Collector
- **Grafana** visualizes both metrics (from Prometheus) and traces (from Jaeger)

### Accessing Observability UIs
- **Grafana**: http://localhost:3001 (admin/admin)
  - Pre-configured dashboards for Jan Server
  - Metrics from Prometheus
  - Traces from Jaeger
- **Prometheus**: http://localhost:9090
  - Direct PromQL queries
  - Metrics exploration
- **Jaeger**: http://localhost:16686
  - Trace search and analysis
  - Service dependency graph

### Logging
- Structured JSON logs (zerolog)
- Log levels configurable via `LOG_LEVEL`
- Request IDs propagated via `X-Request-Id`
- Trace IDs in logs for correlation

### Health Checks
- `GET /healthz` on all services
- Docker healthchecks configured for readiness
- Prometheus service discovery and health monitoring

---

## Migration & Initialization

### Database Migrations
- Migrations located in: `services/llm-api/infrastructure/db/migrations/`
- Applied automatically on llm-api startup (set `AUTO_MIGRATE=false` to disable)
- Inspect progress via `docker compose logs llm-api`

### Keycloak Setup
- Realm imported from `keycloak/import/realm-jan.json`
- Token exchange enabled via `keycloak/init/enable-token-exchange.sh`
- Runs automatically on container startup

### Model Bootstrapping
- On llm-api startup, models from `providers.yaml` are upserted to database
- Ensures model registry stays in sync with configuration

---

## API Conventions

### Error Handling
```json
{
  "type": "invalid_request_error|auth_error|rate_limit_error|internal_error",
  "code": "string",
  "message": "human-friendly message",
  "param": "optional field name",
  "request_id": "uuid"
}
```

### Idempotency
- Header: `Idempotency-Key: <uuid>`
- Supported on: POST `/v1/chat/completions`, `/v1/responses`, `/v1/conversations/*/runs`
- Cached responses returned for duplicate keys

### Pagination
- Query params: `limit`, `after`
- Response: `{data: [...], next_after: "cursor|null"}`

### Streaming (SSE)
- Query param: `?stream=true`
- Response: `Content-Type: text/event-stream`
- Format: `data: {json}\n\n`
- Terminator: `data: [DONE]\n\n`

---

## Development Workflow

### 1. Initial Setup
```bash
cp .env.example .env
# Edit .env with your secrets and HF_TOKEN
```

### 2. Start Services
```bash
make up-gpu     # or make up-cpu
```

### 3. Verify
```bash
curl http://localhost:8000/v1/models
```

### 4. Generate Docs
```bash
make swag  # Merges OpenAPI specs
# Open: http://localhost:8000/v1/swagger/index.html
```

### 5. Test Chat
```bash
# Get guest token
curl -X POST http://localhost:8000/auth/guest

# Use token for chat
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"model":"jan-v1-4b","messages":[{"role":"user","content":"Hello"}]}'
```

### 6. Test MCP Tools
```bash
# List available tools
curl -X POST http://localhost:8000/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list"
  }'

# Execute google_search tool
curl -X POST http://localhost:8000/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "google_search",
      "arguments": {
        "q": "latest AI news"
      }
    }
  }'
```

### 7. View Observability Dashboards (Optional)
```bash
# Start monitoring stack
make monitor-up

# Access dashboards
# Grafana (unified dashboard): http://localhost:3001 (admin/admin)
# Prometheus (metrics): http://localhost:9090
# Jaeger (traces): http://localhost:16686

# View monitoring logs
make monitor-logs

# Stop monitoring stack
make monitor-down
```

### 8. Cleanup
```bash
make down  # Removes containers and volumes
```

---

## Technology Stack

| Component       | Technology                     |
|-----------------|--------------------------------|
| API Gateway     | Kong 3.5                       |
| Services        | Go 1.21+ (Gin framework)       |
| MCP Server      | mark3labs/mcp-go v0.7.0        |
| ORM             | GORM                           |
| Database        | PostgreSQL 16                  |
| Auth            | Keycloak (OpenID Connect)      |
| Inference       | vLLM (OpenAI-compatible)       |
| Observability   | OpenTelemetry Collector        |
| Metrics         | Prometheus 2.48                |
| Tracing         | Jaeger 1.51                    |
| Dashboards      | Grafana 10.2                   |
| Migrations      | golang-migrate                 |
| Containerization| Docker Compose                 |
| Documentation   | OpenAPI 3.0 (Swagger)          |
| External APIs   | Serper API (search/scrape)     |

---

## Future Enhancements

- [ ] Redis-based idempotency store (currently in-memory)
- [ ] Rate limiting per user/API key
- [ ] Multi-provider support (OpenAI, Anthropic, etc.)
- [ ] WebSocket support for bidirectional streaming
- [ ] Admin API for model/provider management
- [x] ~~Prometheus metrics exporter~~ (Implemented)
- [x] ~~Distributed tracing visualization (Jaeger UI)~~ (Implemented)
- [ ] Custom Grafana dashboards for business metrics
- [ ] Alerting rules in Prometheus
- [ ] Log aggregation with Loki
- [ ] Horizontal scaling for llm-api (stateless design ready)
- [ ] S3/blob storage for conversation exports
- [ ] Fine-tuning job management
- [ ] Additional MCP tools (e.g., calculator, file system, database)
- [ ] MCP resources support (for dynamic content)
- [ ] MCP prompts support (for template management)
- [ ] Session management for MCP (if needed for stateful workflows)
- [ ] Authentication for MCP endpoints (if exposing externally)

---

## Troubleshooting

### vLLM GPU Issues
- Ensure NVIDIA drivers installed
- Verify Docker has GPU access: `docker run --rm --gpus all nvidia/cuda:11.8.0-base-ubuntu22.04 nvidia-smi`
- Check `NVIDIA_VISIBLE_DEVICES` in compose file

### Keycloak Not Starting
- Check `keycloak-db` health: `docker compose logs keycloak-db`
- Verify `KC_BOOTSTRAP_ADMIN_PASSWORD` is set in `.env`
- Review init script: `docker compose logs keycloak | grep enable-token-exchange`

### Migration Failures
- Verify `DATABASE_URL` format: `postgres://user:pass@host:port/db`
- Check `api-db` is healthy: `docker compose ps api-db`
- Restart llm-api to retry migrations: `docker compose restart llm-api`

### Authentication Errors
- JWT validation: Check `KEYCLOAK_BASE_URL` and `KEYCLOAK_REALM` in `.env`
- JWKS fetch: Ensure llm-api can reach Keycloak: `docker compose exec llm-api curl http://keycloak:8080/realms/jan/protocol/openid-connect/certs`
- Guest token: Verify guest endpoints are running: `curl http://localhost:8000/auth/guest`

### MCP Tools Issues
- Serper API: Verify `SERPER_API_KEY` is set correctly
- Tool execution: Check mcp-tools logs: `docker compose logs mcp-tools`
- Stateless mode: No session ID required - if getting session errors, check that `WithStateLess(true)` is set
- Method not allowed: Verify method is in `allowedMCPMethods` list
- CORS errors: Check Kong CORS configuration includes MCP headers

### Observability Issues
- **Grafana not accessible**: Start monitoring stack with `make monitor-up`
- **No metrics in Prometheus**: 
  - Ensure monitoring stack is running: `docker compose -f docker-compose.monitor.yml ps`
  - Verify OTEL collector is running: `make monitor-logs`
  - Check Prometheus targets: http://localhost:9090/targets
  - Ensure `OTEL_ENABLED=true` in llm-api environment
- **No traces in Jaeger**: 
  - Verify Jaeger is receiving data: `make monitor-logs | grep jaeger`
  - Check OTEL collector exports: `make monitor-logs | grep otel-collector`
  - Ensure traces are being generated from llm-api
- **Grafana datasources not configured**: 
  - Check provisioning: `make monitor-logs | grep grafana`
  - Verify `docs/grafana/provisioning` is mounted correctly
  - Restart monitoring stack: `make monitor-down && make monitor-up`

---

## References

- [Kong Declarative Config](https://docs.konghq.com/gateway/latest/production/deployment-topologies/db-less-and-declarative-config/)
- [Keycloak Token Exchange](https://www.keycloak.org/docs/latest/securing_apps/#_token-exchange)
- [vLLM Documentation](https://docs.vllm.ai/)
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
- [GORM Documentation](https://gorm.io/docs/)
- [Model Context Protocol (MCP)](https://modelcontextprotocol.io/)
- [mcp-go Library](https://github.com/mark3labs/mcp-go)
- [Serper API Documentation](https://serper.dev/docs)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)

