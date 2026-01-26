# CLAUDE.md - AI Assistant Guidelines for Jan Server

> Instructions for AI coding assistants (Claude, GitHub Copilot, Cursor, etc.) working on this codebase.

## Project Overview

**Jan Server** is an enterprise-grade microservices LLM API platform with Model Context Protocol (MCP) tool integration. It provides OpenAI-compatible APIs, multi-step tool orchestration, media management, and full observability.

**Repository**: Monorepo using pnpm workspaces (frontend) and Go modules (backend)
**Architecture**: Clean Architecture with microservices

---

## Quick Reference

```bash
# Setup & Run
make quickstart              # Interactive setup wizard
make setup                   # Generate .env from template
make up-full                 # Start all services in Docker
make health-check            # Verify all services are healthy

# Development
make dev-full                # Docker stack with host.docker.internal for debugging
go fmt ./...                 # Format Go code
make swagger                 # Regenerate OpenAPI docs
make test-all                # Run all integration tests

# Cleanup
make down                    # Stop containers (keep volumes)
make down-clean              # Stop containers and remove volumes
```

---

## Technology Stack

### Backend (Go Services)

| Component        | Technology           | Version   |
|------------------|----------------------|-----------|
| Language         | Go                   | 1.25.0    |
| HTTP Framework   | Gin                  | v1.11.0   |
| ORM              | GORM + GORM Gen      | v1.26.0   |
| Database         | PostgreSQL           | 18        |
| Cache            | Redis                | Latest    |
| Auth             | Keycloak (OIDC)      | 24.0.5    |
| API Gateway      | Kong                 | 3.5       |
| DI               | Wire                 | v0.7.0    |
| Logging          | Zerolog              | v1.31.0   |
| Observability    | OpenTelemetry        | v1.24.0   |
| MCP Protocol     | mark3labs/mcp-go     | v0.7.0    |

### Frontend (TypeScript/React)

| Component        | Technology           | Version   |
|------------------|----------------------|-----------|
| Web App          | React + Vite         | 19.2.0    |
| Platform App     | Next.js              | 16.0.7    |
| Router           | TanStack Router      | v1.140.0  |
| Styling          | Tailwind CSS         | 4.1.17    |
| UI Components    | Radix UI + shadcn/ui | Latest    |
| AI SDK           | Vercel AI SDK        | 5.0.108   |
| State            | Zustand              | v5.0.9    |
| Package Manager  | pnpm                 | 9.15.4    |

---

## Repository Structure

```
jan-server/
├── apps/                        # Frontend applications
│   ├── web/                     # Chat UI (React + Vite, port 3001)
│   └── platform/                # Admin & docs site (Next.js, port 3000)
├── packages/                    # Shared packages
│   ├── interfaces/              # Shared UI components (@janhq/interfaces)
│   └── go-common/               # Shared Go utilities (config, errors, observability)
├── services/                    # Backend microservices (Go)
│   ├── llm-api/                 # OpenAI-compatible chat completions (port 8080)
│   ├── response-api/            # Multi-step tool orchestration (port 8082)
│   ├── media-api/               # S3 storage, jan_* IDs (port 8285)
│   ├── mcp-tools/               # MCP tool providers (port 8091)
│   ├── memory-tools/            # Semantic memory with BGE-M3 (port 8090)
│   ├── realtime-api/            # WebRTC via LiveKit (port 8186)
│   └── template-api/            # Service scaffold template
├── tools/jan-cli/               # CLI tool for development & testing
├── config/                      # Environment templates and schemas
├── infra/docker/                # Docker Compose fragments
├── integrations/                # Kong plugins, Keycloak config
├── docs/                        # Documentation (80+ files)
├── tests/                       # Integration test collections
├── Makefile                     # Build automation (100+ targets)
├── docker-compose.yml           # Root compose file
└── .env.template                # Environment template
```

### Service Internal Structure

Each Go service under `services/<name>/` follows this layout:

```
services/<service>/
├── cmd/server/                  # Main entrypoint + wire.go
├── internal/
│   ├── domain/                  # Business logic (NO HTTP/DB imports)
│   │   └── <entity>/            # Entity, service, filter, dto
│   ├── infrastructure/          # External integrations
│   │   ├── config/              # Service config loading
│   │   ├── database/            # GORM schemas, repositories
│   │   │   ├── dbschema/        # Schema structs with EtoD/DtoE
│   │   │   └── repository/      # Repository implementations
│   │   └── <provider>/          # External API clients
│   └── interfaces/httpserver/   # HTTP layer
│       ├── routes/              # Gin route handlers
│       ├── middlewares/         # Auth, logging, CORS
│       ├── requests/            # Request DTOs
│       └── responses/           # Response DTOs
├── migrations/                  # SQL migrations (goose)
├── docs/swagger/                # Generated OpenAPI specs
└── Makefile                     # Service-local targets
```

---

## Architecture Rules

### Clean Architecture Layers

```
Interfaces (HTTP handlers, routes)
        ↓
Domain (entities, services, business logic)
        ↓
Infrastructure (repositories, external clients)
```

**Critical Rules:**
1. **Domain packages NEVER import HTTP or database packages**
2. Domain defines interfaces; infrastructure implements them
3. HTTP handlers are thin - convert DTOs to domain structs, call services
4. Business logic lives in domain services, NOT in handlers

---

## Code Conventions

### Naming

| Context           | Convention       | Example                           |
|-------------------|------------------|-----------------------------------|
| Exported Go       | PascalCase       | `UserService`, `CreateUser`       |
| Unexported Go     | camelCase        | `userRepo`, `buildQuery`          |
| Database columns  | snake_case       | `created_at`, `user_id`           |
| JSON fields       | snake_case       | `"user_id"`, `"created_at"`       |
| Environment vars  | SCREAMING_SNAKE  | `SERPER_API_KEY`, `DATABASE_URL`  |

**Avoid stuttering:** Use `provider.ID`, NOT `provider.ProviderID`

### Import Order

```go
import (
    // Standard library
    "context"
    "fmt"

    // Third-party packages
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    // Internal packages
    "jan-server/services/llm-api/internal/domain/user"
)
```

## Git Instructions

### Branch Naming (REQUIRED)
- Always create branches with prefix: `feat/`, `fix/`, or `chore/`
- Example: `feat/add-payment-module`, `fix/resolve-timeout-issue`
- Never commit directly to `main` or `develop`

### Commits
- Do NOT include any AI attribution in commit messages
- Use conventional commit format: `feat:`, `fix:`, `chore:`

---

## Critical Patterns

### GORM Zero-Value Handling (CRITICAL)

**Problem:** GORM's `.Save()` skips fields with zero values (`false`, `0`, `0.0`, `""`).

**Solution:** Use pointer types for fields that can legitimately be zero:

```go
// BAD: Cannot set Enabled to false
type User struct {
    Enabled bool `gorm:"not null;default:true"`
}

// GOOD: Use pointer for zero-affected fields
type User struct {
    Enabled *bool `gorm:"not null;default:true"`
}

// Conversion: Schema to Domain (EtoD)
func (u *User) EtoD() *domain.User {
    enabled := true // default
    if u.Enabled != nil {
        enabled = *u.Enabled
    }
    return &domain.User{Enabled: enabled}
}

// Conversion: Domain to Schema (NewSchema*)
func NewSchemaUser(d *domain.User) *User {
    enabled := d.Enabled
    return &User{Enabled: &enabled}
}
```

### Error Handling (3-Layer Pattern)

1. **Repository (trigger point):** Create errors with `platformerrors.NewError()` and unique UUID
2. **Domain:** Use `platformerrors.AsError()` to add context OR pass through
3. **Route:** Use `responses.HandleError()` for consistent HTTP responses

```go
// Repository - trigger point with unique UUID
return nil, platformerrors.NewError(ctx,
    platformerrors.LayerRepository,
    platformerrors.ErrorTypeNotFound,
    "user not found",
    err,
    "3e47b618-b750-4064-9b22-ece9e244019d")

// Route - handle error for HTTP response
if err != nil {
    responses.HandleError(c, err)
    return
}
```

### Logging Standards

```go
// Use structured logging with zerolog
log.Info().
    Str("user_id", userID).
    Str("action", "create_conversation").
    Msg("Conversation created")

// Always include request_id from middleware context
log.Error().
    Err(err).
    Str("request_id", requestID).
    Msg("Failed to process request")
```

**Log levels:**
- `Debug` - Development noise
- `Info` - State changes, business events
- `Warn` - Recoverable issues
- `Error` - Failures requiring attention

---

## Service Ports

| Service         | Port  | Description                          |
|-----------------|-------|--------------------------------------|
| **Frontend**    |       |                                      |
| Web App         | 3001  | Chat UI (React + Vite)               |
| Platform App    | 3000  | Admin panel & docs (Next.js)         |
| **Backend**     |       |                                      |
| Kong Gateway    | 8000  | API entry point (routes to services) |
| LLM API         | 8080  | Chat completions, conversations      |
| Response API    | 8082  | Multi-step tool orchestration        |
| Media API       | 8285  | File upload, jan_* ID resolution     |
| MCP Tools       | 8091  | Search, scrape, code exec tools      |
| Memory Tools    | 8090  | Semantic memory service              |
| Realtime API    | 8186  | WebRTC session management            |
| **Infra**       |       |                                      |
| Keycloak        | 8085  | Auth admin console                   |
| PostgreSQL      | 5432  | Database                             |
| Grafana         | 3331  | Dashboards                           |
| Prometheus      | 9090  | Metrics                              |
| Jaeger          | 16686 | Tracing                              |

---

## Common Commands

### Service Management

```bash
make up-full              # Start all services (based on COMPOSE_PROFILES in .env)
make up-infra             # Start infrastructure only (DB, Keycloak, Kong)
make up-api               # Start API services
make up-mcp               # Start MCP tools
make down                 # Stop containers (keep volumes)
make down-clean           # Stop and remove volumes
make health-check         # Check all service health
```

### Development

```bash
# Hybrid development (Docker infra + native service)
make dev-full                    # Start Docker with host routing
./jan-cli.sh dev run llm-api     # Run service natively (Linux/Mac)
.\jan-cli.ps1 dev run llm-api    # Run service natively (Windows)

# Code quality
go fmt ./...                     # Format Go code
make lint                        # Run linters
make swagger                     # Regenerate OpenAPI docs

# Database
make db-console                  # Open PostgreSQL shell
make db-reset                    # Reset database
cd services/llm-api && make gormgen  # Regenerate GORM queries
```

### Testing

```bash
make test-all              # All integration test collections
make test-auth             # Authentication tests
make test-conversation     # Conversation tests
make test-response         # Response API tests
make test-media            # Media API tests
make test-mcp              # MCP tools tests
go test ./services/<svc>/...  # Unit tests for specific service
```

### Frontend Development

```bash
# Run frontend apps locally
cd apps/web && npm run dev       # http://localhost:3001
cd apps/platform && npm run dev  # http://localhost:3000

# Or via Docker
make up-web                      # Start web app container
make up-platform                 # Start platform app container

# Build
pnpm build                       # Build all packages/apps
```

---

## Adding New Features

### Adding a New Domain Entity

1. Create `services/<svc>/internal/domain/<entity>/entity.go`
2. Create `services/<svc>/internal/domain/<entity>/service.go`
3. Add schema in `internal/infrastructure/database/dbschema/`
4. Add repository in `internal/infrastructure/database/repository/<entity>repo/`
5. Add HTTP routes in `internal/interfaces/httpserver/routes/`
6. Run `make gormgen` and `make swagger`

### Adding a New HTTP Endpoint

1. Add route handler in `internal/interfaces/httpserver/routes/<area>/`
2. Create request/response DTOs if needed
3. Call domain service (not direct DB access)
4. Use `responses.HandleError()` for errors
5. Add Swagger annotations
6. Run `make swagger`

### Adding a New Environment Variable

1. Add to service's `internal/infrastructure/config/config.go`
2. Add to `.env.template` with documentation
3. Update `config/defaults.yaml` if applicable
4. Document in `docs/configuration/env-var-mapping.md`

### Creating a New Microservice

```bash
./scripts/new-service-from-template.sh my-service  # Linux/Mac
./scripts/new-service-from-template.ps1 -Name my-service  # Windows

# Template includes:
# - Go service skeleton with Gin HTTP server
# - Configuration management (Viper)
# - Structured logging (Zerolog)
# - OpenTelemetry tracing support
# - PostgreSQL with GORM
# - Dependency injection with Wire
# - Docker and Makefile setup
# - Health check endpoint
```

---

## MCP Tools Specifics

The `mcp-tools` service implements Model Context Protocol tools with cascading fallback:

### Search Fallback Chain
```
Serper → Exa → Tavily → SearXNG → Error
```

### Key Config Variables
```bash
SERPER_API_KEY=xxx      SERPER_ENABLED=true
EXA_API_KEY=xxx         EXA_ENABLED=true
TAVILY_API_KEY=xxx      TAVILY_ENABLED=true
SEARXNG_URL=xxx         SEARXNG_ENABLED=true
```

Each provider requires BOTH `*_ENABLED=true` AND valid credentials.

---

## Security Rules

1. **Secrets only in `.env`** - Never hardcode API keys
2. **Kong + Keycloak handle auth** - Don't bypass JWT validation
3. **Never log secrets** - No API keys, tokens, or PII in logs
4. **Validate inputs** - Use validator tags on request structs
5. **Use HTTPS** - For external communication

---

## Before Committing Checklist

1. **Format:** `go fmt ./services/<svc>/...` for changed services
2. **Test:** `go test ./services/<svc>/...` for unit tests
3. **Integration:** `make test-all` if APIs changed
4. **Swagger:** `make swagger` if HTTP contracts changed
5. **GORM:** `cd services/<svc> && make gormgen` if schemas changed
6. **Secrets:** Never commit `.env` files
7. **Docs:** Update relevant guides if behavior changed

---

## Key Documentation

| Topic           | Location                                      |
|-----------------|-----------------------------------------------|
| Quick Start     | `docs/quickstart.md`                          |
| Architecture    | `docs/architecture/README.md`                 |
| Services        | `docs/architecture/services.md`               |
| API Reference   | `docs/api/README.md`                          |
| Development     | `docs/guides/development.md`                  |
| Testing         | `docs/guides/testing.md`                      |
| Conventions     | `docs/conventions/conventions.md`             |
| Design Patterns | `docs/conventions/design-patterns.md`         |
| Configuration   | `docs/configuration/README.md`                |
| CLI Tool        | `tools/jan-cli/README.md`                     |

---

## Troubleshooting

### Common Issues

1. **Service won't start:** Check `make health-check`, verify `.env` exists
2. **Database errors:** Run `make db-migrate` to apply migrations
3. **Auth failures:** Verify Keycloak is running (`http://localhost:8085`)
4. **Port conflicts:** Check `docker ps` for conflicting containers

### Useful Commands

```bash
make logs                       # All container logs
docker compose logs <svc>       # Specific service logs
make db-console                 # PostgreSQL shell
make monitor-up                 # Start observability stack
```

---

## AI Assistant Tips

When working on this codebase:

1. **Read before editing:** Always read files before suggesting modifications
2. **Respect architecture:** Keep domain logic in domain packages, HTTP in interfaces
3. **Use GORM pointers:** For boolean/numeric fields that can be zero
4. **Error handling:** Use the 3-layer pattern with unique UUIDs
5. **Test changes:** Mention relevant test commands for your changes
6. **Update docs:** If you change APIs or configuration, update the docs
7. **Check conventions:** Follow the naming and structure conventions in `docs/conventions/`

For detailed patterns, see:
- `docs/conventions/design-patterns.md` - Code patterns
- `docs/conventions/architecture-patterns.md` - Structure patterns
- `docs/conventions/workflow.md` - Development workflow
- `AGENTS.md` - Detailed agent guidelines
