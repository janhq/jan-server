# template-api

`template-api` is a Go microservice skeleton that mirrors the production layout used by Jan services. Copy this directory when creating a new backend to inherit:

- Environment-driven config loader (`internal/config`).
- Structured logging via Zerolog.
- Optional OpenTelemetry tracing.
- PostgreSQL access via GORM with auto-migrations and seed helpers.
- Missing databases are auto-created when using standard `postgres://` URLs.
- Gin-powered HTTP server with health endpoints.
- Wire-ready dependency injection entrypoint.
- Makefile, Dockerfile, and example environment file for local dev.

## Quick start

```bash
cd services/template-api
go mod tidy
make run
curl http://localhost:8185/healthz
# Optional:
make wire      # regenerate dependency injection
make swagger   # regenerate OpenAPI docs
```

Set configuration values in your shell or using `config/example.env`. See `services/template-api/NEW_SERVICE_GUIDE.md` for detailed migration steps.

## Database

- Point `TEMPLATE_DATABASE_URL` at your PostgreSQL instance (default assumes `postgres:postgres@localhost:5432/template_api`).
- On startup the service runs GORM auto-migrations for the `samples` table and seeds a single row, which powers the `/v1/sample` endpoint.
