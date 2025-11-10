# Service Template Overview

The `services/template-api` directory contains a production-ready skeleton for new Jan microservices. Highlights:

- Go module with config/logger/observability/http packages mirroring established patterns.
- GORM/PostgreSQL wiring (connection pool, migrations, seed data, repository example).
- Optional Keycloak JWT guard controlled via `AUTH_ENABLED`.
- Makefile + Dockerfile for local dev and CI.
- Wire entrypoint plus example env and docs.
- Automation script `scripts/new-service-from-template.ps1 -Name my-service` to scaffold new services.

## Getting Started
1. Run the scaffold script (or copy the folder manually).
2. Update `go.mod`, the service section inside `.env.template`, and `cmd/server/server.go` with your service-specific names and dependencies.
3. Configure `TEMPLATE_DATABASE_URL` (or rename it) and run `make run` so migrations seed the database.
4. Decide whether to enable JWT auth (`AUTH_ENABLED`, `AUTH_ISSUER`, `AUTH_AUDIENCE`, `AUTH_JWKS_URL`).
5. Register your handlers inside `internal/interfaces/httpserver`.
6. Add domain packages and migrations as needed.
7. Update root `.env.template`, README, and deployment manifests to include your service.

See `services/template-api/NEW_SERVICE_GUIDE.md` for a detailed checklist covering both greenfield and migration workflows.
