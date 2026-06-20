# Development Guide

How to set up, run, and iterate on Jan Server locally. This guide covers the full Docker stack, dev-full mode (hybrid debugging), and native service execution.

## Table of Contents

- [Development Guide](#development-guide)
  - [Table of Contents](#table-of-contents)
  - [Prerequisites](#prerequisites)
  - [Quick Start](#quick-start)
  - [Access Points](#access-points)
  - [Project Layout](#project-layout)
  - [Development Workflows](#development-workflows)
    - [Full Docker Stack (default)](#full-docker-stack-default)
    - [Dev-Full Mode (Hybrid Debugging)](#dev-full-mode-hybrid-debugging)
      - [Why Use Dev-Full Mode?](#why-use-dev-full-mode)
      - [Quick Start](#quick-start-1)
      - [How It Works](#how-it-works)
      - [Running Services Natively in Dev-Full Mode](#running-services-natively-in-dev-full-mode)
      - [Workflow](#workflow)
      - [Environment Variables for Hybrid Mode](#environment-variables-for-hybrid-mode)
      - [Monitoring in Dev-Full Mode](#monitoring-in-dev-full-mode)
    - [Running Services Natively](#running-services-natively)
  - [Configuration](#configuration)
  - [Database \& Tooling](#database--tooling)
  - [Testing](#testing)
  - [IDE Integration](#ide-integration)
  - [Troubleshooting \& Next Steps](#troubleshooting--next-steps)
    - [Common Issues](#common-issues)
    - [Health Checks and Monitoring](#health-checks-and-monitoring)
    - [Cleanup](#cleanup)

## Prerequisites

Install these before running any commands:

- **Docker Desktop 24+** with Docker Compose V2
- **GNU Make** (built in on macOS/Linux, install via Chocolatey/Brew on Windows)
- **Go 1.24+** (only required for native/hybrid execution or when editing Go code; see `go.mod` for the exact version)

> Tip: `make setup` uses `jan-cli dev setup` to verify Docker, copy `.env.template` to `.env`.

## Quick Start

```bash
# 1. Clone and enter the repo
git clone https://github.com/janhq/jan-server.git
cd jan-server

# 2. Create .env and Docker networks
make setup

# 3. Start the full stack (infra + APIs + MCP + optional vLLM)
make up-full

# 4. Verify everything is healthy
make health-check

# 5. Tail logs or iterate
make logs          # all containers
docker compose ps  # status
```

- **Stop containers**: `make down`
- **Remove volumes**: `make down-clean`
- **Restart a single service**: `make restart-api`, `make restart-kong`, `make restart-keycloak`

## Access Points

| Service      | URL                    | Notes                                           |
| ------------ | ---------------------- | ----------------------------------------------- |
| Kong Gateway | http://localhost:8000  | Single entry point for all APIs                 |
| LLM API      | http://localhost:8080  | OpenAI-compatible API, `/healthz` for checks    |
| Response API | http://localhost:8082  | Multi-step orchestration                        |
| Media API    | http://localhost:8285  | File upload/management service                  |
| MCP Tools    | http://localhost:8091  | Native MCP tool bridge                          |
| Keycloak     | http://localhost:8085  | Admin/Admin in development                      |
| PostgreSQL   | localhost:5432         | Database user `jan_user` / password from `.env` |
| Grafana      | http://localhost:3331  | Start with `make monitor-up`                    |
| Prometheus   | http://localhost:9090  | Monitoring profile                              |
| Jaeger       | http://localhost:16686 | Tracing profile                                 |

## Project Layout

```
jan-server/
+-- services/              # llm-api, media-api, response-api, mcp-tools, template-api
+-- tools/jan-cli/         # jan-cli sources (run via `./tools/jan-cli.sh`, `.\tools\jan-cli.ps1` wrappers)
+-- packages/go-common/config/ # Single source of truth for config defaults and schema
+-- infra/docker/          # Compose fragments (infrastructure, services, dev-full, observability)
+-- docker-compose.yml     # Root compose file (includes infra/docker/*.yml via profiles)
+-- docker-compose.dev-full.yml # Extra compose overrides for dev-full
+-- integrations/kong/     # Gateway configuration (kong.yml + kong-dev-full.yml)
+-- docs/                  # Documentation (guides, architecture, configuration)
+-- Makefile               # Canonical automation entry point
+-- .env.template          # Copy to .env and edit per environment
```

## Development Workflows

Jan Server supports three development modes:

1. **Full Docker Stack** - All services in containers (parity with CI/production)
2. **Dev-Full Mode** - Hybrid approach with infrastructure in Docker and optional native service execution
3. **Native Execution** - Run services directly on host without dev-full scaffolding

### Full Docker Stack (default)

```bash
make up-full        # start everything defined by COMPOSE_PROFILES in .env
make logs           # follow logs for every service
make logs-api       # only API services
make logs-mcp       # MCP stack
make down           # stop and remove containers
```

Use this mode for integration testing and parity with CI. `COMPOSE_PROFILES` controls which Compose profiles (`infra,api,mcp,full`) load—edit `.env` if you want to disable GPU/vLLM locally.

### Dev-Full Mode (Hybrid Debugging)

**Dev-full mode is the officially supported hybrid workflow.** It starts every dependency in Docker, configures Kong with `host.docker.internal` upstreams, and lets you replace any service with a native process via `jan-cli dev run <service>`.

#### Why Use Dev-Full Mode?

- Fast iteration without rebuilding Docker images
- Debug with breakpoints while Kong continues to enforce plugins and auth
- Keep PostgreSQL, Keycloak, Kong, and monitoring inside Docker for parity
- Instant rollback: stop your host process and Docker takes over again

#### Quick Start

```bash
make setup         # once per machine
make dev-full      # start infra + APIs + MCP with host routing
```

After the stack is up you can replace a service:

```bash
./tools/jan-cli.sh dev run llm-api   # macOS/Linux
.\tools\jan-cli.ps1 dev run llm-api  # Windows PowerShell
```

`jan-cli dev run` stops the matching container, loads environment variables from `.env` (override with `--env`), and runs `go run ./cmd/server` inside `services/<name>`.

#### How It Works

`make dev-full`:

- Runs `docker compose -f docker-compose.yml -f docker-compose.dev-full.yml --profile full up -d`
- Prints URLs for PostgreSQL, Keycloak, Kong, and every API/MCP service
- Keeps the `jan-network`/`jan-monitoring` networks around for fast restarts

Kong's dual-target upstreams (from `integrations/kong/kong-dev-full.yml`):

```yaml
upstreams:
  - name: llm-api-upstream
    targets:
      - target: llm-api:8080
      - target: host.docker.internal:8080
    healthchecks:
      active:
        http_path: /healthz
```

When you stop the Docker container, Kong automatically fails over to the host target. When your host process stops responding to `/healthz`, traffic returns to Docker.

#### Running Services Natively in Dev-Full Mode

| Service      | Port | Command                        |
| ------------ | ---- | ------------------------------ |
| LLM API      | 8080 | `jan-cli dev run llm-api`      |
| Media API    | 8285 | `jan-cli dev run media-api`    |
| Response API | 8082 | `jan-cli dev run response-api` |
| MCP Tools    | 8091 | `jan-cli dev run mcp-tools`    |

**Options:**

- Use `--build` to compile before running (`jan-cli dev run llm-api --build`)
- Pass `--env <file>` if you keep a dedicated env file for host processes (defaults to `.env`)
- To hand control back to Docker, stop the host process and run `docker compose start <service>`

#### Workflow

1. **Start dev-full mode**

   ```bash
   make dev-full
   ```

2. **Replace a service with native execution**

   ```bash
   ./tools/jan-cli.sh dev run llm-api        # macOS/Linux
   .\tools\jan-cli.ps1 dev run llm-api       # Windows PowerShell
   ```

3. **Iterate and debug**
   - Launch Delve: `dlv debug ./cmd/server --headless --listen=:2345`
   - Use `air` for hot reload inside `services/<name>`
   - Observe requests through Kong at http://localhost:8000 exactly as clients would

4. **Hand control back to Docker**
   - Stop your host process (Ctrl+C)
   - Restart the container if needed: `docker compose start llm-api`

5. **Stop dev-full** when done:
   ```bash
   make dev-full-stop   # keep containers
   make dev-full-down   # remove containers
   ```

#### Environment Variables for Hybrid Mode

| Variable                                     | Purpose                                                            |
| -------------------------------------------- | ------------------------------------------------------------------ |
| `DB_POSTGRESQL_WRITE_DSN`                    | PostgreSQL connection string. Use `localhost` when running on host |
| `KEYCLOAK_BASE_URL` / `ISSUER` / `JWKS_URL`  | Auth endpoints (use `http://localhost:8085`)                       |
| `HTTP_PORT`                                  | Local service port (8080, 8082, 8285, 8091, etc.)                  |
| `LOG_LEVEL` / `LOG_FORMAT`                   | Logging controls                                                   |
| `MCP_*` / `SEARXNG_URL` / `VECTOR_STORE_URL` | Tool integrations for mcp-tools                                    |
| `OTEL_*`                                     | Telemetry export (set `OTEL_ENABLED=true` to emit traces)          |

Need a dedicated hybrid env file? Copy `.env` to a separate file, edit the host-specific values, then run `jan-cli dev run llm-api --env <file>`.

#### Monitoring in Dev-Full Mode

You can bring up observability while using dev-full:

```bash
make monitor-up    # Prometheus + Grafana + Jaeger
make monitor-logs  # follow collector/datasource logs
```

Those containers watch the same `jan-network`, so traces and metrics include both Docker and host services (as long as you set `OTEL_ENABLED=true`).

### Running Services Natively

You can run a service completely outside Docker by providing the same environment variables from `.env`:

```bash
# Example for llm-api
docker compose up -d api-db keycloak kong   # ensure infra is running
cd services/llm-api
export DB_POSTGRESQL_WRITE_DSN="postgres://jan_user:${POSTGRES_PASSWORD}@localhost:5432/jan_llm_api?sslmode=disable"
export KEYCLOAK_BASE_URL="http://localhost:8085"
export JWKS_URL="http://localhost:8085/realms/jan/protocol/openid-connect/certs"
export ISSUER="http://localhost:8085/realms/jan"
export HTTP_PORT=8080
export LOG_LEVEL=debug

go run ./cmd/server
```

> Windows users can run `.\tools\jan-cli.ps1 dev run llm-api --env .env` to load variables automatically.

## Configuration

- Copy `.env.template` to `.env` (or run `make setup`) and edit secrets like `HF_TOKEN`, `SERPER_API_KEY`, and `POSTGRES_PASSWORD`
- Compose loads the root `.env` directly via `env_file`, so all services use the same values
- `packages/go-common/config/defaults.yaml` is the canonical configuration, generated from Go structs in `packages/go-common/config/types.go`
- Helpful jan-cli commands:

```bash
jan-cli config validate --file config/defaults.yaml
jan-cli config show --path services.llm-api
jan-cli config export --format env --output config/generated.env
```

## Database & Tooling

```bash
make db-migrate    # Apply Go migrations for llm-api
make db-reset      # Drop + recreate tables (uses docker compose)
make db-console    # Opens psql inside the api-db container

# Direct Docker examples
docker compose logs api-db
psql "postgres://jan_user:${POSTGRES_PASSWORD}@localhost:5432/jan_llm_api?sslmode=disable"
```

For backups and restores use `make db-backup` / `make db-restore`. The Makefile targets wrap `docker compose` so they work on Windows, macOS, and Linux.

## Testing

- **Full integration suite**: `make test-all` (runs every Postman collection listed in the Makefile)
- **Focused suites**: `make test-auth`, `make test-conversation`, `make test-response`, `make test-media`, `make test-mcp`, `make test-mcp-agents`
- **Unit tests**: run them from each service directory (`go test ./...`)

See [Testing Guide](testing.md) for platform details, CI coverage, and troubleshooting tips.

## IDE Integration

- VS Code launch configurations can depend on a task that runs `make dev-full`
- After the task finishes, `jan-cli dev run <service>` is a good preLaunchCommand
- Debuggers simply connect to the local port (Kong still listens on 8000)
- Hot reload tools (`air`, `reflex`, etc.) live inside `services/<name>`

Example `.vscode/launch.json` for debugging llm-api:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug LLM API",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/services/llm-api/cmd/server",
      "envFile": "${workspaceFolder}/.env",
      "preLaunchTask": "dev-full-start"
    }
  ]
}
```

## Troubleshooting & Next Steps

### Common Issues

| Symptom                                 | Fix                                                                                                   |
| --------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| Kong keeps hitting the Docker container | Ensure the host service listens on the same port and returns 200 on `/healthz`                        |
| Service cannot reach PostgreSQL         | Update `DB_POSTGRESQL_WRITE_DSN` to use `localhost` instead of `api-db` when running on host          |
| Environment variables missing           | Pass `--env .env` (default) or a custom env file to `jan-cli dev run`                                 |
| Port already in use                     | Stop the other listener or change `HTTP_PORT` before running the host service                         |
| Want to debug multiple services         | Run `jan-cli dev run ...` in multiple terminals; each command stops its corresponding container first |

### Health Checks and Monitoring

1. `make health-check` - verifies infrastructure, API, MCP, and optional services
2. `make logs` or `docker compose logs <service>` - inspect failures quickly
3. `make restart-kong` / `make restart-keycloak` - common fixes for gateway/auth issues
4. `make monitor-up` - bring up Grafana/Prometheus/Jaeger if you need observability while debugging

### Cleanup

```bash
make dev-full-stop   # stop containers but keep them around
make dev-full-down   # stop + remove containers
make down            # standard docker workflow stop
make down-clean      # remove volumes and networks for pristine state
```

Need more help? Review [Testing](testing.md), [Troubleshooting](troubleshooting.md), [Monitoring](monitoring.md), and the configuration docs under `docs/configuration/`.
