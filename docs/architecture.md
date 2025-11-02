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
│  │  • Route: /v1/* → llm-api-svc                                         │  │
│  │  • Plugins:                                                            │  │
│  │    - Key-Auth (X-API-Key) → Injects X-Consumer-* headers             │  │
│  │    - CORS                                                              │  │
│  │  • Port: 8000                                                          │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         APPLICATION LAYER                                    │
│                                                                              │
│  ┌─────────────────────────────────────┐   ┌────────────────────────────┐  │
│  │         LLM-API Service             │   │   GuestAuth Service        │  │
│  │  (Port: 8080, Internal)             │   │   (Port: 8080, Exposed)    │  │
│  │                                     │   │                            │  │
│  │  • REST API (Gin Framework)        │   │  • REST API (Gin)          │  │
│  │  • OpenAPI/Swagger                  │   │  • Guest User Creation     │  │
│  │  • Authentication:                  │   │  • Account Upgrade         │  │
│  │    - JWT (Keycloak JWKS)           │   │  • Keycloak Integration    │  │
│  │    - API Key (Kong consumer)       │   │                            │  │
│  │  • Middleware:                      │   │  Endpoints:                │  │
│  │    - Auth                           │   │  POST /auth/guest          │  │
│  │    - Request ID                     │   │  POST /auth/upgrade        │  │
│  │    - SSE Support                    │   │                            │  │
│  │  • Idempotency Store                │   │                            │  │
│  │  • OpenTelemetry Integration        │   │                            │  │
│  │                                     │   │                            │  │
│  │  Endpoints:                         │   └────────────────────────────┘  │
│  │  • GET  /v1/models                  │                                    │
│  │  • GET  /v1/models/:id              │                                    │
│  │  • POST /v1/chat/completions        │                                    │
│  │  • POST /v1/completions             │                                    │
│  │  • POST /v1/conversations           │                                    │
│  │  • GET  /v1/conversations           │                                    │
│  │  • GET  /v1/conversations/:id       │                                    │
│  │  • POST /v1/conversations/:id/msgs  │                                    │
│  │  • GET  /v1/conversations/:id/msgs  │                                    │
│  │  • POST /v1/conversations/:id/runs  │                                    │
│  │  • POST /v1/responses               │                                    │
│  └─────────────────────────────────────┘                                    │
│             │              │                                                 │
│             │              └──────────────────┐                              │
│             ▼                                 ▼                              │
│  ┌──────────────────────┐         ┌─────────────────────────┐              │
│  │  Provider Registry   │         │   Repository Layer      │              │
│  │                      │         │                         │              │
│  │  • providers.yaml    │         │  • ModelRepository      │              │
│  │  • Default: vllm     │         │  • ConversationRepo     │              │
│  │  • Model routing     │         │  • MessageRepository    │              │
│  │  • Capability flags  │         │  • GORM ORM             │              │
│  └──────────────────────┘         └─────────────────────────┘              │
│             │                                 │                              │
└─────────────┼─────────────────────────────────┼──────────────────────────────┘
              │                                 │
              ▼                                 ▼
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
│  │           OpenTelemetry Collector (Port: 4318)                         │  │
│  │  • Traces, metrics, logs collection                                    │  │
│  │  • Exporters configured via otel-collector.yaml                        │  │
│  │  • Connected to llm-api telemetry                                      │  │
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
  │ ✓ Key-auth (skipped, anonymous consumer)
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

### Pattern 2: Guest User Creation

```
Client
  │
  │ POST /auth/guest
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

### Pattern 3: Conversation Management

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
  - `llm-api-svc` → `http://llm-api:8080`
- **Routes**: `/v1/*`
- **Plugins**:
  - `key-auth`: Validates `X-API-Key`, creates anonymous consumer if missing
  - `cors`: Allows cross-origin requests
- **Consumer Injection**: Sets `X-Consumer-Username`, `X-Consumer-ID` headers

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
- **Port**: 4318 (OTLP HTTP receiver)
- **Config**: `docs/otel-collector.yaml`
- **Purpose**: Centralized telemetry collection from llm-api

---

## Configuration Management

### Environment Variables (.env)
```bash
# Database
POSTGRES_USER=jan
POSTGRES_PASSWORD=<secret>
POSTGRES_DB=jan_api
DATABASE_URL=postgres://jan:<secret>@api-db:5432/jan_api

# LLM-API
HTTP_PORT=8080
LOG_LEVEL=info
LOG_FORMAT=json
AUTO_MIGRATE=true
PROVIDERS_CONFIG_PATH=/app/config/providers.yaml

# Keycloak
KEYCLOAK_BASE_URL=http://keycloak:8080
KEYCLOAK_REALM=jan
KC_BOOTSTRAP_ADMIN_USERNAME=admin
KC_BOOTSTRAP_ADMIN_PASSWORD=<secret>

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

### Full Stack + GPU
```bash
make up-gpu
# Starts: api-db, llm-api, kong, keycloak, keycloak-db, vllm-llama, otel-collector
```

### Full Stack + CPU
```bash
make up-cpu
# Same as GPU but uses vllm-cpu profile (no GPU requirements)
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
- `keycloak-db:5432` (PostgreSQL)
- `keycloak:8080` (Keycloak)
- `vllm-llama:8000` (vLLM Inference)
- `otel-collector:4318` (OTLP HTTP)

### Exposed Ports
- `8000` -> Kong Gateway (public API)
- `8080` -> LLM API (guest and v1 endpoints)
- `8085` -> Keycloak Admin Console

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
   - Only Kong, Keycloak, GuestAuth exposed externally

3. **Authentication Layers**:
   - Kong: API key validation (optional)
   - LLM-API: JWT validation (required for user data)
   - vLLM: Internal bearer token

4. **CORS**:
   - Configured in Kong plugin
   - Allows all origins in development (should be restricted in production)

---

## Observability

### Metrics & Traces
- **OpenTelemetry** integration in llm-api
- Traces sent to `otel-collector:4318`
- Configurable exporters (Jaeger, Prometheus, etc.)

### Logging
- Structured JSON logs (zerolog)
- Log levels configurable via `LOG_LEVEL`
- Request IDs propagated via `X-Request-Id`

### Health Checks
- `GET /healthz` on all services
- Docker healthchecks configured for readiness

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
curl -X POST http://localhost:8080/auth/guest

# Use token
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"model":"jan-v1-4b","messages":[{"role":"user","content":"Hello"}]}'
```

### 6. Cleanup
```bash
make down  # Removes containers and volumes
```

---

## Technology Stack

| Component       | Technology                     |
|-----------------|--------------------------------|
| API Gateway     | Kong 3.5                       |
| Services        | Go 1.21+ (Gin framework)       |
| ORM             | GORM                           |
| Database        | PostgreSQL 16                  |
| Auth            | Keycloak (OpenID Connect)      |
| Inference       | vLLM (OpenAI-compatible)       |
| Observability   | OpenTelemetry Collector        |
| Migrations      | golang-migrate                 |
| Containerization| Docker Compose                 |
| Documentation   | OpenAPI 3.0 (Swagger)          |

---

## Future Enhancements

- [ ] Redis-based idempotency store (currently in-memory)
- [ ] Rate limiting per user/API key
- [ ] Multi-provider support (OpenAI, Anthropic, etc.)
- [ ] WebSocket support for bidirectional streaming
- [ ] Admin API for model/provider management
- [ ] Prometheus metrics exporter
- [ ] Distributed tracing visualization (Jaeger UI)
- [ ] Horizontal scaling for llm-api (stateless design ready)
- [ ] S3/blob storage for conversation exports
- [ ] Fine-tuning job management

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
- Guest token: Verify guest endpoints are running: `curl http://localhost:8080/healthz`

---

## References

- [Kong Declarative Config](https://docs.konghq.com/gateway/latest/production/deployment-topologies/db-less-and-declarative-config/)
- [Keycloak Token Exchange](https://www.keycloak.org/docs/latest/securing_apps/#_token-exchange)
- [vLLM Documentation](https://docs.vllm.ai/)
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
- [GORM Documentation](https://gorm.io/docs/)

