# CONVENTIONS - Quick Reference

> Start here for the current Jan Server standards. Detailed references live in the other files inside `docs/conventions/`.

---

## Documentation Map

| File                         | Purpose                                    |
| ---------------------------- | ------------------------------------------ |
| `conventions.md` (this file) | TL;DR and quick links                      |
| `architecture-patterns.md`   | Repository and service layout patterns     |
| `design-patterns.md`         | Code-level guidance (DB, errors, entities) |
| `workflow.md`                | Daily workflow: git, testing, deployments  |

---

## TL;DR Rules

### Language & Tooling

- **Go version:** `1.25.0` (matches the root `go.mod`). Run `go fmt ./...` and `go test ./...` before committing.
- **Dependency hygiene:** `go mod tidy` inside the service you changed.

### Architecture

- Each service lives under `services/<name>/` and follows the same structure (`cmd/`, `internal/`, `migrations/`, etc.).
- Clean architecture still applies: Interfaces (HTTP) -> Domain -> Infrastructure. Domain packages never import database or HTTP packages.

### Database

- GORM zero-value issue still exists. Use pointer fields (`*bool`, `*float64`, etc.) in schema structs and convert to plain types in domain models.
- When schemas change, run `make gormgen` from the service directory (e.g., `cd services/llm-api && make gormgen`).

### Error Handling

- Trigger point (repository) creates errors via `platformerrors.NewError()`.
- Handlers call `responses.HandleError()` so every response includes `requestID`.
- Never log secrets or the raw error from external providers.

### Naming

- Exported symbols: `PascalCase`. Unexported: `camelCase`.
- Database columns: `snake_case`.
- Avoid stuttering (`provider.ID`, not `provider.ProviderID`).
- Avoid single word naming, must meaningful, easy to read and understand

### Git & Commits

- Conventional commits: `feat:`, `fix:`, `docs:`, `test:`, `chore:`, etc.
- Branches: `type/short-description` (e.g., `feat/dev-full-refresh`).

### Security

- Secrets only live in `.env`/environment variables. `.env` is created from `.env.template` via `make setup` and never committed.
- Kong + Keycloak handle auth; do not bypass JWT/API-key validation in services.

---

## Common Commands

```bash
# Setup & environments
make setup              # Copy .env.template -> .env (single root .env)
make up-full            # Start infra + APIs + MCP in Docker
make dev-full           # Start Docker stack with host routing for native services
tools/jan-cli.sh dev run llm-api   # Run a service on host (macOS/Linux)
.\jan-cli.ps1 dev run llm-api  # Same on Windows

# Monitoring & tooling
make monitor-up         # Prometheus + Grafana + Jaeger
make monitor-clean      # Stop monitoring and remove volumes

# Testing
make test-all           # Run every jan-cli api-test collection
make test-auth          # Focused suite (see Makefile for others)
go test ./services/llm-api/...    # Service-level unit tests

# Code generation
(cd services/llm-api && make gormgen)   # Regenerate GORM queries after schema changes
make swagger            # Rebuild Swagger docs for all services

# Database helpers
make db-console         # Open psql shell inside api-db
make db-reset           # Drop + recreate llm-api database
```

---

## Critical Checklist Before Pushing

1. `go fmt ./...` in every service you touched.
2. `go test ./...` (unit) and `make test-all` or the relevant jan-cli api-test suites if you changed APIs.
3. `make swagger` if REST contracts changed.
4. `(cd services/<name> && make gormgen)` if DB schemas changed.
5. `.env`/secrets unchanged and never committed.
6. Conventional commit message, CI passes locally (`make up-full && make health-check`).

---

## Need More Detail?

- **Structure & file placement:** `architecture-patterns.md`
- **Code patterns (DB, entities, errors):** `design-patterns.md`
- **Daily workflow (git, CI/CD, deployment):** `workflow.md`

Always keep docs and commands in sync with the Makefile, jan-cli, and the actual service directories. If a command does not exist locally, update the documentation first.
