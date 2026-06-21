# Development Workflow & Process

> Daily workflow for working on Jan Server. These steps reflect the multi-service layout (`services/<service>/`) and the Makefile targets that exist today.

---

## Table of Contents

1. [Environment & Local Setup](#environment--local-setup)
2. [Git Workflow](#git-workflow)
3. [Testing Strategy](#testing-strategy)
4. [Code Generation](#code-generation)
5. [Security Practices](#security-practices)
6. [Logging Standards](#logging-standards)
7. [Code Review Checklist](#code-review-checklist)
8. [Common Commands](#common-commands)
9. [Deployment Checklist](#deployment-checklist)

---

## Environment & Local Setup

```bash
# 1. Clone
git clone https://github.com/janhq/jan-server.git
cd jan-server

# 2. Create the root .env, verify Docker, etc.
make setup

# 3. Start full stack (PostgreSQL, Keycloak, Kong, APIs, MCP)
make up-full

# 4. Check health
make health-check
```

### Dev-Full Hybrid Workflow

Use dev-full when you want to run a service natively with IDE tooling while the rest stay in Docker.

```bash
make dev-full                 # Boot stack with host.docker.internal routing
tools/jan-cli.sh dev run llm-api  # macOS/Linux (stops container + runs host process)
.\jan-cli.ps1 dev run llm-api # Windows PowerShell
```

Stop dev-full with `make dev-full-stop` (containers paused) or `make dev-full-down` (containers removed).

### Environment Variables

- `.env` (created by `make setup`) is the single source of truth. Copy from `.env.template` and set secrets (HF token, SERPER key, etc.).
- Docker Compose loads this same root `.env` into every service via `env_file: ${ENV_FILE:-../../.env}`---there is no separate `docker/.env`.
- When running a service natively, pass the environment via `jan-cli dev run <service> --env path/to/env` or export manually.

---

## Git Workflow

### Branches & Commits

- Branch format: `type/short-description` (e.g., `feat/dev-full-refresh`).
- Use [Conventional Commits](https://www.conventionalcommits.org/) for messages (`feat:`, `fix:`, `docs:`, `test:`, `chore:`, `refactor:`, etc.).

### Typical Flow

```bash
git checkout -b feat/new-flow
# ...edit files...
go fmt ./...
go test ./services/llm-api/...
make test-all

git add <files>
git commit -m "feat(llm-api): add new flow"
git push origin feat/new-flow
```

### Pull Requests

1. Keep PRs focused on one change.
2. Mention any manual testing performed (e.g., `make test-all`, `make dev-full`, curl commands).
3. Update documentation when commands/configuration change.
4. Wait for CI (GitHub Actions) to pass before merging.

---

## Testing Strategy

| Scope                       | Command                                                                                           |
| --------------------------- | ------------------------------------------------------------------------------------------------- |
| Unit tests (per service)    | `go test ./services/<service>/...`                                                                |
| Full jan-cli api-test suite | `make test-all`                                                                                   |
| Focused integration suites  | `make test-auth`, `make test-conversations`, `make test-media`, `make test-mcp-integration`, etc. |
| Health checks               | `make health-check`                                                                               |

Guidelines:

- Run the service-level Go tests before committing.
- Run the relevant jan-cli api-test collection (or `make test-all`) when changing API routes, Kong config, or MCP tools.
- For MCP-only work, `make test-mcp-integration` plus manual curl checks against `http://localhost:8000/mcp`.

---

## Code Generation

| Task                        | Command                                                         |
| --------------------------- | --------------------------------------------------------------- |
| Swagger docs (all services) | `make swagger`                                                  |
| GORM queries (llm-api)      | `cd services/llm-api && make gormgen`                           |
| Other services              | Add service-local generators if needed (follow llm-api example) |

Run generators whenever you change schema structs or API contracts, then commit the generated files.

---

## Security Practices

- Secrets (`HF_TOKEN`, `SERPER_API_KEY`, `BACKEND_CLIENT_SECRET`, etc.) stay in `.env` only.
- Never log access tokens, API keys, or PII. Use structured logging fields (`logger.With(...)`) instead.
- Kong + Keycloak enforce JWT/API key validation---never bypass them in service routes.
- Use HTTPS when transmitting secrets externally; assume `.env` values will be different in production.

---

## Logging Standards

- Structured logs with `logger.With()` and key/value pairs.
- Include `requestID` (propagated via middleware) in every log and error response.
- Use log levels consistently: `Debug` for development noise, `Info` for state changes, `Warn` for recoverable issues, `Error` for failures.
- Do not log request/response bodies unless behind a debug flag.

---

## Code Review Checklist

**Architecture**

- Domain packages contain business rules only.
- Infrastructure code is injected and has no HTTP imports.
- Routes remain thin; DTOs convert to domain structs immediately.

**Database**

- Schema structs use pointers for zero-value fields.
- `EtoD()` / `DtoE()` cover nil vs default conversion.
- Repositories rely on GORM-gen query builders.

**Errors & Logging**

- Trigger points create `platformerrors.NewError()` instances with UUIDs.
- Errors bubble through `AsError()` and are rendered with `responses.HandleError()`.
- request IDs logged and returned.

**Testing**

- Unit tests cover new code paths.
- Integration/jan-cli api-test suites updated if APIs change.
- Feature flags or env toggles documented.

**Security**

- No credentials committed or logged.
- Input validation present on HTTP DTOs.
- Kong/Keycloak configs updated if auth flow changes.

**Docs**

- README/guides updated when commands or endpoints change.
- Swagger re-generated for API changes.

---

## Common Commands

```bash
# Services & infrastructure
make up-full            # Start everything via Docker
make down               # Stop and remove containers (keeps volumes)
make dev-full           # Start Docker stack with host routing
make dev-full-down      # Remove dev-full containers
make monitor-up         # Observability stack
make monitor-clean      # Remove monitoring volumes

# Database helpers
make db-console         # psql shell into api-db
make db-reset           # Drop & recreate llm-api database

# Testing
make test-all           # All jan-cli api-test suites
make test-auth          # Focused collection
make test-mcp-integration

# Lint/format (run inside the service module)
go fmt ./...
go test ./services/<service>/...
```

---

## Deployment Checklist

- [ ] `go fmt ./...`
- [ ] `go test ./services/<service>/...`
- [ ] `make test-all` (or relevant suites)
- [ ] `make swagger` (if API contracts changed)
- [ ] `(cd services/llm-api && make gormgen)` (if schemas changed)
- [ ] `.env` changes documented but not committed
- [ ] Docker/Kubernetes manifests updated if ports/env vars changed
- [ ] PR reviewed and CI green

Once everything passes, follow the platform release process (merge to `main`, tag if needed, then promote via the deployment pipeline).
