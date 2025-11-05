# Architecture Comparison: Jan Server vs Reference Platform

> **Document Purpose**: This document provides a comprehensive comparison between the new `jan-server` architecture and the reference `platform` codebase, highlighting key differences in design philosophy, technical stack, and implementation patterns.

---

## Table of Contents

1. [High-Level Overview](#high-level-overview)
2. [Architecture Philosophy](#architecture-philosophy)
3. [Technical Stack Comparison](#technical-stack-comparison)
4. [Layer-by-Layer Comparison](#layer-by-layer-comparison)
5. [Request Flow Patterns](#request-flow-patterns)
6. [Code Organization](#code-organization)
7. [Key Feature Differences](#key-feature-differences)
8. [Migration Considerations](#migration-considerations)

---

## High-Level Overview

### Jan Server
**Type**: Microservices-based LLM API platform  
**Purpose**: OpenAI-compatible API endpoints for chat completions, conversations, and model management  
**Target**: Production-ready LLM inference service with enterprise authentication

### Reference Platform
**Type**: Monolithic enterprise platform  
**Purpose**: Full-featured business platform with billing, user management, and model provider catalogs  
**Target**: Multi-tenant SaaS application with complex business logic

---

## Architecture Philosophy

| Aspect | Jan Server | Reference Platform |
|--------|-----------|-------------------|
| **Architecture Style** | Microservices (2+ services) | Modular Monolith |
| **Service Boundaries** | Service-level separation (llm-api, mcp-tools) | Package-level separation (domain modules) |
| **API Gateway** | Kong (declarative config) | None (direct HTTP) |
| **Design Pattern** | Pragmatic layering with infrastructure helpers | Strict Clean Architecture |
| **Complexity** | Minimal, focused on LLM operations | Complex, full business platform |
| **Domain Purity** | Domain can inject thin helpers (cache, etc.) | Domain layer completely pure (no external deps) |
| **Deployment** | Docker Compose with profiles | Docker Compose for local dev |

---

## Technical Stack Comparison

### Programming & Frameworks

| Component | Jan Server | Reference Platform |
|-----------|-----------|-------------------|
| **Go Version** | 1.23.0 | 1.24.6 |
| **HTTP Framework** | Gin | Gin |
| **ORM** | GORM | GORM |
| **Code Generation** | GORM Gen | GORM Gen + Wire (DI) |
| **API Documentation** | Swagger (OpenAPI) | Swagger (OpenAPI) |
| **Dependency Injection** | Manual (via constructors) | Wire (automated) |
| **Testing** | Standard Go tests | Table-driven tests (convention) |

### Infrastructure & Services

| Component | Jan Server | Reference Platform |
|-----------|-----------|-------------------|
| **API Gateway** | ✅ Kong 3.5 (declarative) | ❌ None |
| **Authentication** | Keycloak (OpenID Connect) | Keycloak (assumed) |
| **Database** | PostgreSQL 16 (2 instances) | PostgreSQL (single) |
| **Cache** | ❌ None | ✅ Redis |
| **Message Queue** | ❌ None | ✅ Kafka |
| **Inference Backend** | vLLM (OpenAI-compatible) | External model providers (OpenAI, etc.) |
| **Observability** | OpenTelemetry + Prometheus + Jaeger + Grafana | Minimal (basic logging) |
| **MCP Protocol** | ✅ Native support (mcp-tools service) | ❌ None |

### Monitoring & Observability

| Component | Jan Server | Reference Platform |
|-----------|-----------|-------------------|
| **Tracing** | ✅ OpenTelemetry → Jaeger | ❌ None |
| **Metrics** | ✅ Prometheus | ❌ None |
| **Dashboards** | ✅ Grafana | ❌ None |
| **OTEL Collector** | ✅ Full pipeline (OTLP HTTP/gRPC) | ❌ None |
| **Log Correlation** | ✅ Trace ID + Span ID in logs | ✅ Request ID only |
| **Middleware** | TracingMiddleware + LoggingMiddleware | Request ID only |

---

## Layer-by-Layer Comparison

### 1. Client/Entry Layer

#### Jan Server
```
┌─────────────────────────────────────┐
│   Kong API Gateway (Port 8000)      │
│   • Declarative routing             │
│   • CORS with MCP headers           │
│   • Service-level routing           │
│     - /v1/* → llm-api               │
│     - /v1/mcp → mcp-tools           │
│     - /auth → llm-api               │
└─────────────────────────────────────┘
```

#### Reference Platform
```
┌─────────────────────────────────────┐
│   Direct HTTP (Port 8080)           │
│   • No API gateway                  │
│   • Application-level routing       │
│   • Route handlers in code          │
└─────────────────────────────────────┘
```

**Key Difference**: Jan Server uses Kong for centralized routing and CORS, enabling microservices. Reference Platform uses direct HTTP routing.

---

### 2. Application Layer

#### Jan Server - Services

**llm-api Service (Port 8080)**
```
internal/
├── config/              # Environment config
├── domain/
│   ├── conversation/    # Conversation entity + service
│   ├── message/         # Message entity + service
│   └── model/           # Model entity + service
├── infrastructure/
│   ├── database/        # GORM repositories
│   ├── httpclients/     # vLLM HTTP client
│   ├── keycloak/        # Auth client
│   └── observability/   # OTEL helpers
├── interfaces/
│   └── httpserver/
│       ├── handlers/    # Minimal reusable helpers
│       ├── middlewares/ # Auth, Tracing, Logging
│       └── routes/      # Main orchestration
└── utils/              # Logger, errors, etc.
```

**mcp-tools Service (Port 8091)**
```
domain/
  └── serper/           # Search service
infrastructure/
  └── serper/           # Serper API client
interfaces/
  └── httpserver/
      ├── handlers/     # MCP handlers
      └── routes/       # MCP routes
utils/                  # MCP utilities
```

#### Reference Platform - Modular Monolith

```
cmd/
  ├── server/           # Main entry point
  └── gormgen/          # Code generator
internal/
├── domain/             # Business logic modules
│   ├── apikey/         # API key management
│   ├── auth/           # Authentication
│   ├── billing/        # Billing & payments
│   ├── model/          # Model provider catalogs
│   ├── query/          # Query history
│   └── user/           # User management
├── infrastructure/
│   ├── database/       # DB schemas + repos
│   ├── cache/          # Redis cache
│   └── messagequeue/   # Kafka producers
├── interfaces/
│   ├── httpserver/     # HTTP layer
│   ├── eventconsumers/ # Kafka consumers
│   └── crontab/        # Scheduled jobs
└── utils/              # Cross-cutting helpers
```

**Key Differences**:
- **Services**: Jan Server has 2 separate services; Platform is monolithic
- **Scope**: Jan Server focused on LLM operations; Platform has billing, payments, etc.
- **Handlers**: Jan Server minimizes handlers (orchestration in routes); Platform may use handlers as reusable helpers
- **Infrastructure**: Platform has Redis + Kafka; Jan Server uses OTEL instead

---

### 3. Domain Layer

#### Jan Server Domain
```go
// Example: Conversation domain
type Conversation struct {
    ID        string
    UserID    string
    Title     string
    Metadata  map[string]interface{}
    CreatedAt time.Time
    UpdatedAt time.Time
}

type ConversationService struct {
    repo *repository.ConversationRepository
    // Can inject thin infrastructure helpers
}

func (s *ConversationService) Create(ctx context.Context, userID, title string) (*Conversation, error) {
    // Business logic with repository calls
}
```

**Philosophy**: Domain can depend on thin infrastructure helpers (pragmatic approach)

#### Reference Platform Domain
```go
// Example: User domain
type User struct {
    PublicID  string
    Email     string
    Roles     []string
    CreatedAt time.Time
}

type UserService struct {
    // NO infrastructure dependencies
    // Only domain interfaces
}

func (s *UserService) CreateUser(ctx context.Context, email string) (*User, error) {
    // Pure business logic
    // No DB, cache, or external calls
}
```

**Philosophy**: Domain is completely pure (strict Clean Architecture)

**Key Difference**: Jan Server allows injecting infrastructure helpers into domain services for pragmatism. Reference Platform enforces strict domain purity.

---

### 4. Infrastructure Layer

#### Jan Server Infrastructure

| Component | Implementation | Purpose |
|-----------|---------------|---------|
| **Database** | GORM + Gen | Postgres repositories |
| **HTTP Clients** | Resty v3 | vLLM inference calls |
| **Keycloak** | Custom client | JWT validation, user creation |
| **Observability** | OTEL SDK | Traces, metrics, helper functions |

**No Cache, No MQ**: Simpler infrastructure focused on LLM operations

#### Reference Platform Infrastructure

| Component | Implementation | Purpose |
|-----------|---------------|---------|
| **Database** | GORM + Gen | Postgres repositories |
| **Cache** | Redis | Performance optimization, session storage |
| **Message Queue** | Kafka | Event-driven architecture, async jobs |
| **External APIs** | Stripe, OpenAI, etc. | Payment processing, AI providers |

**Comprehensive Infrastructure**: Full enterprise stack with caching and messaging

**Key Difference**: Reference Platform has more infrastructure components (Redis, Kafka) for enterprise features. Jan Server keeps it minimal.

---

### 5. Interface Layer

#### Jan Server Routes

```go
// routes/v1/conversations/route.go
func RegisterRoutes(r *gin.RouterGroup, svc *conversation.ConversationService) {
    r.POST("", func(c *gin.Context) {
        // Orchestration logic here
        principal := c.MustGet("principal").(auth.Principal)
        conv, err := svc.Create(c.Request.Context(), principal.UserID, req.Title)
        // ...
    })
}
```

**Pattern**: Routes handle orchestration directly, minimal handlers

#### Reference Platform Routes

```go
// routes/v1/management/users/route.go
func SetupUserRoutes(r *gin.RouterGroup, handler *handlers.UserHandler) {
    r.POST("", handler.CreateUser)
    r.GET("/:id", handler.GetUser)
}

// handlers/user_handler.go
func (h *UserHandler) CreateUser(c *gin.Context) {
    // Reusable handler logic
    // Calls service layer
}
```

**Pattern**: Handlers as optional reusable helpers across routes

**Key Difference**: Jan Server routes do orchestration directly. Reference Platform may use handlers for shared logic (though conventions say "avoid unnecessary wrappers").

---

## Request Flow Patterns

### Jan Server: Chat Completion

```
Client
  │
  │ POST /v1/chat/completions
  │ Authorization: Bearer <JWT>
  │
  ▼
Kong Gateway (Port 8000)
  │ ✓ CORS check
  │ ✓ Route to llm-api-svc
  │
  ▼
LLM-API Service (Port 8080)
  │ ✓ TracingMiddleware (start span)
  │ ✓ LoggingMiddleware (add trace_id to logs)
  │ ✓ AuthMiddleware (JWT validation via Keycloak JWKS)
  │ ✓ Extract principal (user_id, scopes)
  │ ✓ Idempotency check
  │ ✓ Model resolution (providers.yaml)
  │
  ▼
vLLM Server (Port 8000, internal)
  │ POST /v1/chat/completions
  │ Authorization: Bearer <VLLM_INTERNAL_KEY>
  │
  ▼
Response (SSE Stream)
  │ ✓ OpenTelemetry span tracking (tokens, duration, finish_reason)
  │ ✓ Store idempotency result
  │ ✓ Emit telemetry to OTEL Collector
  │
  ▼
Client (streaming response)
```

### Reference Platform: Similar Operation

```
Client
  │
  │ POST /v1/completions
  │ Authorization: Bearer <JWT>
  │
  ▼
Platform Service (Port 8080)
  │ ✓ RequestIDMiddleware
  │ ✓ AuthMiddleware (JWT validation)
  │ ✓ Extract principal
  │ ✓ Service orchestration
  │
  ▼
External Provider (OpenAI, etc.)
  │ POST /v1/chat/completions
  │ Authorization: Bearer <API_KEY>
  │
  ▼
Response
  │ ✓ Cache result (Redis)
  │ ✓ Publish event (Kafka)
  │ ✓ Update billing (domain service)
  │
  ▼
Client
```

**Key Differences**:
- **Gateway**: Jan Server uses Kong; Platform direct routing
- **Observability**: Jan Server has full OTEL tracing; Platform has basic request ID
- **Caching**: Platform uses Redis; Jan Server uses idempotency store
- **Events**: Platform publishes to Kafka; Jan Server emits OTEL events
- **Backend**: Jan Server uses vLLM; Platform uses external APIs

---

## Code Organization

### Project Structure Comparison

| Aspect | Jan Server | Reference Platform |
|--------|-----------|-------------------|
| **Mono/Multi Repo** | Monorepo (services/*) | Single service |
| **Service Count** | 2 (llm-api, mcp-tools) | 1 (platform) |
| **Cmd/** | ✅ server + gormgen | ✅ server + gormgen |
| **Config/** | ✅ Environment-based | ✅ Version + environment |
| **Domain/** | ✅ Entity + service pattern | ✅ Entity + service pattern |
| **Infrastructure/** | Database, HTTP clients, OTEL | Database, Cache, MQ, External APIs |
| **Interfaces/** | httpserver only | httpserver, eventconsumers, crontab |
| **Utils/** | Logger, errors, validators | Logger, errors, crypto, validators |
| **Migrations/** | ✅ Embedded in binary | ❌ Separate migration tool |
| **Tests/** | ✅ Integration tests (Postman/Newman) | ✅ Unit + integration |

### File Naming Conventions

| Aspect | Jan Server | Reference Platform |
|--------|-----------|-------------------|
| **Files** | `lowercase.go` | `lowercase.go` or `user_service.go` |
| **Directories** | `lowercase` | `lowercase` (no underscores) |
| **Packages** | Single word | Single word |
| **Variables** | `camelCase` | `camelCase` |
| **Exported** | `PascalCase` | `PascalCase` |
| **DB Columns** | `snake_case` | `snake_case` |

**Similarity**: Both follow Go conventions and avoid stuttering (e.g., `user.ID` not `user.UserID`)

---

## Key Feature Differences

### 1. Authentication

| Feature | Jan Server | Reference Platform |
|---------|-----------|-------------------|
| **Provider** | Keycloak (OpenID Connect) | Keycloak (assumed) |
| **Auth Methods** | JWT + API Key (Kong consumer) | JWT (primary) |
| **Guest Users** | ✅ Native support (/auth/guest, /auth/upgrade) | ❌ Not mentioned |
| **Token Exchange** | ✅ Backend client → user token | ❓ Unknown |
| **JWKS Validation** | ✅ Dynamic refresh (5m interval) | ❓ Unknown |
| **Custom Claims** | `preferred_username`, `guest` flag | ❓ Unknown |

### 2. Model Management

| Feature | Jan Server | Reference Platform |
|---------|-----------|-------------------|
| **Provider Abstraction** | ✅ providers.yaml (vllm default) | ✅ Model provider catalogs |
| **Default Provider** | vLLM (local inference) | External APIs (OpenAI, etc.) |
| **Model Switching** | Dynamic routing via config | ❓ Unknown |
| **Capability Flags** | ✅ (streaming, tool_calling, etc.) | ✅ |
| **Model Database** | ✅ Persisted in PostgreSQL | ✅ Persisted |

### 3. Observability

| Feature | Jan Server | Reference Platform |
|---------|-----------|-------------------|
| **OpenTelemetry** | ✅ Full integration (traces + metrics) | ❌ None |
| **Tracing Backend** | ✅ Jaeger (OTLP) | ❌ None |
| **Metrics Backend** | ✅ Prometheus | ❌ None |
| **Dashboards** | ✅ Grafana (pre-configured) | ❌ None |
| **OTEL Collector** | ✅ Centralized pipeline | ❌ None |
| **Middleware** | TracingMiddleware + LoggingMiddleware | RequestIDMiddleware only |
| **Span Attributes** | LLM-specific (tokens, model, duration) | N/A |
| **Monitoring Stack** | Separate docker-compose.monitor.yml | N/A |

**Winner**: Jan Server has enterprise-grade observability; Platform has minimal logging

### 4. MCP (Model Context Protocol)

| Feature | Jan Server | Reference Platform |
|---------|-----------|-------------------|
| **MCP Support** | ✅ Native (dedicated mcp-tools service) | ❌ None |
| **MCP Library** | mark3labs/mcp-go v0.7.0 | N/A |
| **Mode** | Stateless (no sessions) | N/A |
| **Tools** | google_search, scrape | N/A |
| **External Service** | Serper API | N/A |
| **Routing** | Kong → /v1/mcp → mcp-tools | N/A |
| **CORS Headers** | Mcp-Session-Id, mcp-protocol-version | N/A |

**Winner**: Jan Server is MCP-native; Platform has no MCP support

### 5. Persistence

| Feature | Jan Server | Reference Platform |
|---------|-----------|-------------------|
| **Database** | PostgreSQL 16 (2 instances) | PostgreSQL |
| **ORM** | GORM + Gen | GORM + Gen |
| **Migrations** | ✅ Embedded (golang-migrate) | Separate tool |
| **Schema Management** | dbschema/ + gormgen/ | dbschema/ + gormgen/ |
| **Entities** | Conversations, Messages, Models | Users, API Keys, Billing, Organizations |
| **Zero-Value Handling** | ✅ Pointers for bool/float64 | ✅ Pointers (convention enforced) |

**Similarity**: Both use GORM Gen with similar patterns. Reference Platform has stricter conventions.

### 6. Caching & Messaging

| Feature | Jan Server | Reference Platform |
|---------|-----------|-------------------|
| **Cache** | ❌ None (uses idempotency store) | ✅ Redis |
| **Message Queue** | ❌ None | ✅ Kafka |
| **Event Publishing** | OTEL events only | Kafka events |
| **Async Jobs** | ❌ None | ✅ Kafka consumers |
| **Scheduled Tasks** | ✅ Cron (minimal) | ✅ Crontab (extensive) |

**Winner**: Reference Platform has more infrastructure for enterprise features

### 7. API Design

| Feature | Jan Server | Reference Platform |
|---------|-----------|-------------------|
| **API Style** | OpenAI-compatible REST | Custom REST |
| **Versioning** | /v1/* | /v1/* |
| **Swagger** | ✅ Auto-generated | ✅ Auto-generated |
| **Request DTOs** | requests/ package | requests/ package |
| **Response DTOs** | responses/ package | responses/ package |
| **Error Format** | PlatformError + HTTP codes | PlatformError + HTTP codes |
| **Idempotency** | ✅ Idempotency-Key header | ❌ Not mentioned |

**Similarity**: Both use versioned REST with Swagger. Jan Server adds idempotency.

---

## Migration Considerations

### Moving from Reference Platform to Jan Server

#### ✅ Easy Migrations
- **Domain entities**: Similar patterns, easy to port
- **GORM schemas**: Same dbschema + gormgen approach
- **Error handling**: Both use PlatformError pattern
- **Middleware**: Easy to add (auth, request ID, etc.)
- **Swagger docs**: Same annotation style

#### ⚠️ Moderate Effort
- **Remove Redis caching**: Redesign without cache or use idempotency
- **Remove Kafka**: Convert to synchronous or OTEL events
- **Wire → Manual DI**: Replace Wire with manual dependency injection
- **Handlers → Routes**: Move orchestration logic directly into routes
- **Add Kong**: Introduce API gateway layer

#### 🔴 High Effort
- **Monolith → Microservices**: Split into multiple services (llm-api, mcp-tools)
- **Add observability**: Implement full OTEL stack (Collector, Prometheus, Jaeger, Grafana)
- **External APIs → vLLM**: Switch from external providers to local inference
- **Add MCP support**: Implement MCP protocol and tools
- **Separate monitoring**: Extract to docker-compose.monitor.yml

### Moving from Jan Server to Reference Platform

#### ✅ Easy Migrations
- **Domain patterns**: Port conversation/message logic to platform style
- **GORM schemas**: Same approach, easy to integrate
- **Authentication**: Keycloak already compatible
- **Middleware**: Keep existing patterns

#### ⚠️ Moderate Effort
- **Merge services**: Consolidate llm-api and mcp-tools into monolith
- **Remove Kong**: Direct HTTP routing
- **Add Wire**: Implement automated dependency injection
- **Add handlers**: Extract reusable logic from routes

#### 🔴 High Effort
- **Add Redis caching**: Implement cache layer
- **Add Kafka**: Implement message queue + consumers
- **Add billing**: Full payment processing
- **Add organizations**: Multi-tenancy support
- **Remove OTEL**: Replace with basic logging (if not keeping observability)

---

## Architecture Decision Comparison

### When to Use Jan Server Architecture

✅ **Choose Jan Server if you need:**
- OpenAI-compatible LLM API
- Local inference with vLLM
- MCP (Model Context Protocol) support
- Microservices architecture
- Enterprise observability (OTEL + Jaeger + Prometheus)
- API gateway routing (Kong)
- Guest authentication flows
- Minimal infrastructure (no Redis, no Kafka)
- Rapid deployment with Docker Compose
- Streaming SSE responses
- Idempotency support

### When to Use Reference Platform Architecture

✅ **Choose Reference Platform if you need:**
- Full enterprise SaaS platform
- Billing and payment processing
- Multi-tenancy / Organizations
- User management
- API key management
- Caching layer (Redis)
- Event-driven architecture (Kafka)
- Strict Clean Architecture (pure domain)
- Wire dependency injection
- Automated code generation (Wire + GORM)
- Extensive business logic modules
- Scheduled background jobs
- External API integrations (Stripe, OpenAI, etc.)

---

## Conventions Comparison

### Code Style

| Convention | Jan Server | Reference Platform |
|-----------|-----------|-------------------|
| **Go Version** | 1.23.0 | 1.24.6 |
| **Formatting** | gofmt | gofmt (pre-commit hook) |
| **Import Order** | stdlib → external → internal | stdlib → external → internal |
| **Naming** | camelCase/PascalCase | camelCase/PascalCase |
| **DB Columns** | snake_case | snake_case |
| **Error Wrapping** | ✅ AsError() pattern | ✅ AsError() pattern |
| **Zero-Value Fields** | ✅ Pointers | ✅ Pointers (strict) |
| **Commit Messages** | Free-form | Conventional Commits (feat:, fix:) |
| **Pre-commit Hooks** | ❌ None | ✅ Codegen + Swagger |

### Testing

| Convention | Jan Server | Reference Platform |
|-----------|-----------|-------------------|
| **Unit Tests** | Standard Go tests | Table-driven tests (convention) |
| **Integration Tests** | ✅ Newman (Postman) | ✅ Go integration tests |
| **Test Location** | tests/automation/ | Next to code (convention) |
| **Mocking** | Minimal | Mock external deps (convention) |
| **Coverage** | Not enforced | Not enforced (but encouraged) |

### Documentation

| Convention | Jan Server | Reference Platform |
|-----------|-----------|-------------------|
| **Architecture Docs** | ✅ Comprehensive (architecture.md) | ✅ Comprehensive (conventions/*.md) |
| **API Docs** | ✅ Swagger (auto-generated) | ✅ Swagger (auto-generated) |
| **Code Comments** | Minimal | Encouraged |
| **README** | ✅ Quickstart + setup | ✅ Quickstart + conventions |
| **Conventions Doc** | ❌ None | ✅ Extensive (3 files) |

---

## Summary Matrix

### Quick Comparison Table

| Feature | Jan Server | Reference Platform |
|---------|:----------:|:-----------------:|
| **Architecture** | Microservices | Modular Monolith |
| **Services** | 2 (llm-api, mcp-tools) | 1 (platform) |
| **API Gateway** | ✅ Kong | ❌ None |
| **Observability** | ✅ Full OTEL stack | ❌ Minimal |
| **Caching** | ❌ None | ✅ Redis |
| **Message Queue** | ❌ None | ✅ Kafka |
| **MCP Support** | ✅ Native | ❌ None |
| **Inference** | ✅ vLLM (local) | External APIs |
| **Guest Auth** | ✅ Native | ❌ Unknown |
| **Idempotency** | ✅ Built-in | ❌ Unknown |
| **Billing** | ❌ None | ✅ Full stack |
| **Organizations** | ❌ None | ✅ Multi-tenant |
| **DI Method** | Manual | Wire (automated) |
| **Domain Purity** | Pragmatic | Strict |
| **Pre-commit Hooks** | ❌ None | ✅ Codegen |
| **Conventions Docs** | ❌ None | ✅ Extensive |
| **Deployment** | Docker Compose | Docker Compose |

---

## Recommendations

### For New LLM Projects
👉 **Use Jan Server** if you need OpenAI-compatible APIs with local inference and enterprise observability.

### For Enterprise SaaS Platforms
👉 **Use Reference Platform** as a foundation and add LLM features from Jan Server patterns.

### For Hybrid Approach
👉 **Combine both**:
1. Use Reference Platform conventions (Clean Architecture, Wire, pre-commit hooks)
2. Add Jan Server features (Kong, OTEL, MCP, vLLM)
3. Keep strict domain purity from Platform
4. Add observability stack from Jan Server
5. Use Kong for microservices if needed

---

## Conclusion

Both architectures serve different purposes:

**Jan Server** excels at:
- LLM-focused operations
- Microservices deployment
- Enterprise observability
- Minimal infrastructure
- Rapid deployment

**Reference Platform** excels at:
- Complex business logic
- Multi-tenancy
- Billing and payments
- Event-driven architecture
- Strict architectural patterns
- Extensive conventions

The choice depends on your project requirements. For pure LLM inference services, Jan Server provides a production-ready foundation. For full enterprise platforms, Reference Platform offers comprehensive patterns and infrastructure.

Both share common strengths:
- Clean Architecture principles
- GORM Gen for type-safe queries
- Gin for HTTP routing
- PostgreSQL for persistence
- Swagger for API documentation
- Go best practices

---

**Document Version**: 1.0  
**Last Updated**: November 6, 2025  
**Maintained By**: Jan Server Team
