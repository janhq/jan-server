# Service Template Overview

The `services/template-api` directory contains a production-ready skeleton for new Jan microservices. Highlights:

- Go module with config/logger/observability/http packages mirroring established patterns.
- GORM/PostgreSQL wiring (connection pool, migrations, seed data, repository example).
- Optional Keycloak JWT guard controlled via `AUTH_ENABLED`.
- Makefile + Dockerfile for local dev and CI.
- Wire entrypoint plus example env and docs.
- Use `jan-cli dev scaffold <service-name>` to copy the template with placeholders replaced.

## Getting Started

1. Run `jan-cli dev scaffold my-service` (or copy `services/template-api` manually).
2. Update `go.mod`, the service section inside `.env.template`, and `cmd/server/server.go` with your service-specific names and dependencies.
3. Configure the database DSN (`DB_POSTGRESQL_WRITE_DSN`) and run `go run ./cmd/server` once so migrations seed the database.
4. Decide whether to enable JWT auth (`AUTH_ENABLED`, `AUTH_ISSUER`, `ACCOUNT`, `AUTH_JWKS_URL`).
5. Register your handlers inside `internal/interfaces/httpserver`.
6. Add domain packages and migrations as needed.
7. Update root `.env.template`, README, and deployment manifests to include your service.
