# AGENTS.md - AI Coding Agent Guidelines

> Instructions for AI coding assistants (GitHub Copilot, Claude, Cursor, etc.) working on Jan Server.

## Project Overview

**Jan Server** is an enterprise-grade microservices LLM API platform with MCP (Model Context Protocol) tool integration. It provides OpenAI-compatible APIs, multi-step tool orchestration, media management, and full observability.

### Tech Stack

| Component | Technology |
|-----------|------------|
| **Backend** | |
| Language | Go 1.25+ |
| Framework | Gin (HTTP), zerolog (logging), Wire (DI) |
| ORM | GORM + goose migrations |
| Database | PostgreSQL (ankane/pgvector:latest for API, postgres:18 for Keycloak) |
| API Gateway | Kong 3.5 + Keycloak (OIDC) |
| Inference | vLLM (OpenAI-compatible) or remote providers |
| MCP Server | mark3labs/mcp-go v0.7.0 |
| Observability | OpenTelemetry, Prometheus, Jaeger, Grafana |
| Container | Docker Compose (dev), Kubernetes/Helm (prod) |
| **Frontend** | |
| Web App | React 19 + Vite + TanStack Router + Tailwind CSS 4 |
| UI Components | Radix UI + shadcn/ui patterns |
| State | Zustand |
| AI SDK | Vercel AI SDK (@ai-sdk/react) |
| Package Manager | npm (monorepo with workspaces) |

---

## Repository Structure

```
jan-server/
├── apps/                        # Frontend applications
│   ├── web/                     # Chat UI (React + Vite, port 3001)
│   └── platform/                # Admin & docs site (Next.js, port 3000)
├── packages/                    # Shared packages
│   ├── interfaces/              # Shared UI components (@janhq/interfaces)
│   └── go-common/               # Shared Go utilities
├── services/                    # Backend microservices (Go)
│   ├── llm-api/                 # OpenAI-compatible chat completions (port 8080)
│   ├── response-api/            # Multi-step tool orchestration (port 8082)
│   ├── media-api/               # S3 storage, jan_* IDs (port 8285)
│   ├── mcp-tools/               # MCP tool providers (port 8091)
│   ├── memory-tools/            # Semantic memory with BGE-M3 (port 8090)
│   ├── realtime-api/            # WebRTC via LiveKit (port 8186)
│   └── template-api/            # Service scaffold template
├── tools/jan-cli/               # CLI tool sources
├── config/                      # Shared configuration (defaults.yaml, schemas)
├── infra/docker/                # Docker Compose fragments
├── docs/                        # Documentation
├── k8s/                         # Kubernetes Helm charts
├── tests/                       # Integration test collections
├── Makefile                     # Build automation (100+ targets)
├── docker-compose.yml           # Root compose file
└── .env.template                # Environment template
```

### Service Internal Structure

Each service under `services/<name>/` follows this layout:

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

## Frontend Applications

### Web App (`apps/web/`)

Chat interface built with React 19 + Vite.

```
apps/web/
├── src/
│   ├── components/          # UI components (chat-input, sidebar, settings)
│   ├── routes/              # TanStack Router pages
│   ├── stores/              # Zustand state stores
│   ├── services/            # API client services
│   ├── hooks/               # Custom React hooks
│   ├── lib/                 # Utilities
│   └── App.tsx              # Root component
├── package.json
├── vite.config.ts
└── tsconfig.json
```

**Key Technologies:**
- **Router:** TanStack Router (file-based routing)
- **State:** Zustand
- **AI:** Vercel AI SDK (`@ai-sdk/react`, `ai`)
- **UI:** Radix UI + Tailwind CSS 4
- **Build:** Vite (rolldown)

**Environment Variables:**
```env
JAN_API_BASE_URL=http://localhost:8000    # Kong gateway URL
VITE_GA_ID=                               # Google Analytics (optional)
```

**Commands:**
```bash
cd apps/web
npm install
npm run dev      # Start dev server on port 3001
npm run build    # Production build
npm run lint     # ESLint
```

### Shared Package (`packages/interfaces/`)

Shared UI components library used by the web app.

```
packages/interfaces/
├── src/
│   ├── ui/                  # shadcn/ui components (button, input, dialog, etc.)
│   ├── hooks/               # Shared React hooks
│   ├── lib/                 # Utilities (cn, constants)
│   ├── svgs/                # SVG icon components
│   └── ai-elements/         # AI-specific UI components
└── package.json
```

**Usage in apps:**
```tsx
import { Button } from "@janhq/interfaces/ui/button"
import { useMobile } from "@janhq/interfaces/hooks/use-mobile"
import { cn } from "@janhq/interfaces/lib"
```

### Frontend Development Workflow

```bash
# Start full stack (backend + frontend)
make up-full
make up-web          # Start web app container

# Or run frontend locally
cd apps/web && npm run dev       # http://localhost:3001

# Build for production
cd apps/web && npm run build
```

---

## Coding Patterns

### GORM Zero-Value Handling (CRITICAL)

GORM's `.Updates()` with structs skips zero values (`false`, `0`, `0.0`, `""`). Use pointer types for fields that can legitimately be zero:

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

### Error Handling

```go
// Repository creates errors with platformerrors
return nil, platformerrors.NewError(ctx, 
    platformerrors.LayerRepository,
    platformerrors.ErrorTypeNotFound, 
    "user not found", 
    err,
    "unique-uuid-here")

// Handler uses responses.HandleError for consistent responses
if err != nil {
    responses.HandleError(c, err)
    return
}
```

### Naming Conventions

| Context | Convention | Example |
|---------|------------|---------|
| Exported Go | PascalCase | `UserService`, `CreateUser` |
| Unexported Go | camelCase | `userRepo`, `buildQuery` |
| Database columns | snake_case | `created_at`, `user_id` |
| JSON fields | snake_case | `"user_id"`, `"created_at"` |
| Environment vars | SCREAMING_SNAKE | `SERPER_API_KEY` |

**Avoid stuttering:** Use `provider.ID`, NOT `provider.ProviderID`

---

## Common Commands

```bash
# Setup & Run
make setup              # Create .env, docker/.env, networks
make up-full            # Start all services in Docker
make dev-full           # Hybrid mode (Docker + native debugging)
make health-check       # Verify services are healthy

# Development
go fmt ./...                          # Format code
go test ./services/<svc>/...          # Unit tests
make test-all                         # All integration tests
make swagger                          # Regenerate OpenAPI specs
cd services/<svc> && make gormgen     # Regenerate GORM queries

# Monitoring
make monitor-up         # Start Prometheus, Grafana, Jaeger
make logs               # Tail all container logs

# Cleanup
make down               # Stop containers (keep volumes)
make down-clean         # Remove containers AND volumes
```

---

## Service Ports

| Service | Port | Description |
|---------|------|-------------|
| **Frontend** | | |
| Web App | 3001 | Chat UI (React + Vite) |
| Platform App | 3000 | Admin panel & docs (Next.js) |
| **Backend** | | |
| Kong Gateway | 8000 | API entry point (routes to all services) |
| LLM API | 8080 | Chat completions, conversations, models |
| Response API | 8082 | Multi-step tool orchestration |
| Media API | 8285 | File upload, jan_* ID resolution |
| MCP Tools | 8091 | MCP protocol tools (search, scrape, exec) |
| Memory Tools | 8090 | Semantic memory service |
| Realtime API | 8186 | WebRTC session management |
| **Infrastructure** | | |
| Keycloak | 8085 | Auth admin console |
| PostgreSQL | 5432 | Database |
| Grafana | 3331 | Dashboards |
| Prometheus | 9090 | Metrics |
| Jaeger | 16686 | Tracing |

---

## Key Documentation

| Topic | Location |
|-------|----------|
| Quick start | [docs/quickstart.md](docs/quickstart.md) |
| Architecture | [docs/architecture/README.md](docs/architecture/README.md) |
| Services | [docs/architecture/services.md](docs/architecture/services.md) |
| API Reference | [docs/api/README.md](docs/api/README.md) |
| Development | [docs/guides/development.md](docs/guides/development.md) |
| Testing | [docs/guides/testing.md](docs/guides/testing.md) |
| Conventions | [docs/conventions/conventions.md](docs/conventions/conventions.md) |
| Design Patterns | [docs/conventions/design-patterns.md](docs/conventions/design-patterns.md) |
| Configuration | [docs/configuration/README.md](docs/configuration/README.md) |

---

## Before Committing

1. **Format:** `go fmt ./services/<svc>/...` for changed services
2. **Test:** `go test ./services/<svc>/...` for unit tests
3. **Integration:** `make test-all` if APIs changed
4. **Swagger:** `make swagger` if HTTP contracts changed
5. **GORM:** `cd services/<svc> && make gormgen` if schemas changed
6. **Secrets:** Never commit `.env` files
7. **Commit message:** Use [Conventional Commits](https://www.conventionalcommits.org/)
   - `feat:`, `fix:`, `docs:`, `test:`, `chore:`, `refactor:`
   - Example: `feat(llm-api): add conversation branching`

---

## MCP Tools Service Specifics

The `mcp-tools` service implements Model Context Protocol tools with cascading fallback:

### Search Fallback Chain
```
Serper → Exa → Tavily → SearXNG → Error
```

### Scrape Fallback Chain
```
Serper → Exa → Tavily → Direct HTTP → Error
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

## Important Patterns to Follow

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

---

## Logging Standards

```go
// Use structured logging with zerolog
log.Info().
    Str("user_id", userID).
    Str("action", "create_conversation").
    Msg("Conversation created")

// Always include request_id (from middleware context)
log.Error().
    Err(err).
    Str("request_id", requestID).
    Msg("Failed to process request")

// Log levels:
// Debug - Development noise
// Info  - State changes, business events
// Warn  - Recoverable issues
// Error - Failures requiring attention
```

---

## Security Rules

1. **Secrets only in `.env`** - Never hardcode API keys
2. **Kong + Keycloak handle auth** - Don't bypass JWT validation
3. **Never log secrets** - No API keys, tokens, or PII in logs
4. **Validate inputs** - Use validator tags on request structs
5. **Use HTTPS** - For external communication

---

## Testing Strategy

| Scope | Command |
|-------|---------|
| Unit tests | `go test ./services/<svc>/...` |
| Full integration | `make test-all` |
| Auth tests | `make test-auth` |
| Conversation tests | `make test-conversations` |
| MCP tests | `make test-mcp-integration` |
| Health checks | `make health-check` |

---

## Troubleshooting

### Common Issues

1. **Service won't start:** Check `make health-check`, verify `.env` exists
2. **Database errors:** Run `make db-migrate` to apply migrations
3. **Auth failures:** Verify Keycloak is running (`http://localhost:8085`)
4. **Port conflicts:** Check `docker ps` for conflicting containers

### Useful Commands

```bash
make logs                    # All container logs
docker compose logs <svc>    # Specific service logs
make db-console              # PostgreSQL shell
make monitor-up              # Start observability stack
```

---

## Quick Reference

### File Naming

- Entity files: `entity.go`, `service.go`, `filter.go`
- Schema files: `<entity>.go` in `dbschema/`
- Repository: `<entity>_repository.go` in `<entity>repo/`
- Handlers: `<resource>_handler.go` or `<resource>.go`

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

### Git Branch Naming

```
feat/short-description
fix/issue-description
docs/update-readme
test/add-integration-tests
chore/update-deps
```
