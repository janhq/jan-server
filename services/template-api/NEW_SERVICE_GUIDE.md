# New Service Guide (Template API)

This guide explains how to turn `services/template-api` into a real microservice and how to migrate existing services onto the shared scaffolding.

## 1. When to use the template
- **Net-new backend**: always start from the template to inherit logging, tracing, config, Makefile/Docker, DB access, and swagger defaults.
- **Existing service refresh**: when a legacy service still has bespoke bootstrap code, migrate it onto the template to standardize health checks, env handling, observability, and persistence.

## 2. Bootstrapping a new service
1. **Copy the template**
   ```powershell
   scripts\new-service-from-template.ps1 -Name my-service
   ```
   or manually copy `services/template-api` to `services/my-service`.
2. **Rename module path**
   - Update `services/my-service/go.mod` `module jan-server/services/my-service`.
   - Run `go mod tidy`.
3. **Set service metadata**
   - Edit `cmd/server/main.go` placeholders: `ServiceName`, HTTP port, feature flags.
   - Update `internal/config/config.go` with service-specific env vars.
4. **Configure the database**
   - Point `TEMPLATE_DATABASE_URL` at your PostgreSQL instance (rename the variable if you prefer `DATABASE_URL`).
   - Adjust pool knobs (`DB_MAX_IDLE_CONNS`, `DB_MAX_OPEN_CONNS`, `DB_CONN_MAX_LIFETIME`). Migrations/seed data run on startup in `database.AutoMigrate`.
5. **Register routes/crons**
   - Add handlers inside `internal/interfaces/httpserver/handlers`.
   - Expose them via `routes/v1` (or additional versions) and update swagger comments.
6. **Docs & env**
   - Copy `.env.template` entries into the repo root `.env.template`.
   - Create `README.md` describing the service purpose and dependencies.
7. **Smoke test**
   ```bash
   make -C services/my-service run
   curl http://localhost:<port>/healthz
   curl http://localhost:<port>/v1/sample
   ```

## 3. Migrating an existing service
1. **Inventory current bootstrap**: list config structs, logger init, HTTP server, cron, database migrations.
2. **Adopt shared packages**:
   - Import template logger/observability/httpserver packages.
   - Replace local equivalents incrementally (feature flags help).
3. **Align config**:
   - Move env parsing to the template’s `internal/config`.
   - Add service-specific sections (DB creds, upstream API keys) and update env templates (`TEMPLATE_DATABASE_URL`, pool settings, etc.).
4. **Swap entrypoint**:
   - Replace legacy `main.go` with template `cmd/server/main.go`.
   - Ensure DI/wire setup compiles; fix provider bindings as needed.
5. **Update tooling**:
   - Adopt template Makefile targets and Dockerfile if they differ.
   - Update CI/CD to call the new targets (`make build-service SERVICE=my-service`).
6. **Verify & deploy**:
   - Run unit/integration tests.
   - Deploy to staging using the new container image, monitor logs/metrics.
7. **Delete old scaffolding** once parity is confirmed (config loaders, logger utils, custom DB glue).

## 4. Repo-wide updates when adding a service
- **Root Makefile**: append service-specific targets if needed.
- **docker-compose / k8s manifests**: add deployment definitions for the new service.
- **Monitoring/alerting**: reuse template dashboards; only tweak service name & ports.
- **Documentation**: link to the new service README from `docs/services.md`.

## 5. Tips
- Keep the template lean; domain dependencies belong in the individual service.
- Prefer importing shared packages from `pkg/` rather than copying code when possible.
- Dogfood: ensure `services/llm-api` compiles using the same shared packages so the template never drifts.
