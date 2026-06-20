# Jan Server

> A microservices LLM API platform with MCP tool integration

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24.0-00ADD8?logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-required-2496ED?logo=docker)](https://www.docker.com/)

## Prerequisites

Before running Jan Server locally make sure you have:

- **Docker Desktop** (Windows/macOS) or **Docker Engine + docker compose V2** (Linux)
- **Make** (installed by default on Linux/macOS, [install on Windows](https://gnuwin32.sourceforge.net/packages/make.htm))
- **Git** for cloning the repository
- **8 GB RAM minimum** (12 GB recommended for all services)
- Optional: **NVIDIA GPU + recent drivers** if you plan to run local vLLM inference

## Quick Start

```bash
# Clone and enter the repo
git clone https://github.com/janhq/server.git
cd server

# Interactive setup (runs jan-cli wizard and docker compose)
make quickstart
```

The `quickstart` target wraps `jan-cli` and guides you through:

- Selecting the LLM provider (local vLLM vs remote OpenAI-compatible endpoint)
- Choosing the MCP search provider (Serper, SearXNG, or disabled)
- Enabling or disabling the Media API

Need to rerun the wizard? Execute `make quickstart` again and accept the prompt to update your `.env`.

Prefer a scripted setup? Run:

```bash
make setup   # Generates/updates .env via jan-cli
make up-full # Starts every service defined in docker-compose.yml
```

**More detail**: [Quickstart Documentation](docs/quickstart.md)

**Services running after `make up-full`:**

- **API Gateway**: http://localhost:8000 (Kong)
- **LLM API**: http://localhost:8080 (OpenAI-compatible)
- **Response API**: http://localhost:8082 (Multi-step orchestration)
- **Media API**: http://localhost:8285 (Media management)
- **MCP Tools**: http://localhost:8091 (Tool integration)
- **API Documentation**: http://localhost:8000/api/swagger/index.html
- **Keycloak Console**: http://localhost:8085 (admin/admin)

> Keycloak now runs directly from the official `quay.io/keycloak/keycloak:24.0.5` image with our realm/import scripts bind-mounted at runtime - no bundled Keycloak source tree is required.

## What is Jan Server?

Jan Server is an enterprise-grade LLM API platform that provides:

- **OpenAI-compatible API** for chat completions and conversations
- **Multi-step tool orchestration** with Response API for complex workflows
- **Media management** with S3 integration and `jan_*` ID resolution
- **MCP (Model Context Protocol)** tools for web search, scraping, and code execution
- **OAuth/OIDC authentication** via Keycloak with guest access
- **Full observability** with OpenTelemetry, Prometheus, Jaeger, and Grafana
- **Flexible deployment** with Docker Compose profiles and Kubernetes support

## Features

- **OpenAI-compatible chat completions API** with streaming support
- **Response API** for multi-step tool orchestration (max depth: 8, timeout: 45s)
- **Media API** with S3 storage, jan\_\* ID system, and presigned URLs
- **MCP tools** (google_search, web scraping, code execution via SandboxFusion)
- **Conversation and message management** with PostgreSQL persistence
- **Guest and user authentication** via Keycloak OIDC enforced by Kong gateway (JWT + custom API key plugin)
- **API gateway routing** via Kong v3.5
- **Distributed tracing** with Jaeger and OpenTelemetry
- **Metrics and dashboards** with Prometheus and Grafana
- **Development mode** with host.docker.internal support for flexible debugging
- **Comprehensive testing suite** with 6 jan-cli api-test collections
- **Service template system** for rapid microservice creation

## Documentation

Primary entry points:

- [docs/README.md](docs/README.md) - Documentation hub overview and navigation map grouped by audience
- [docs/architecture/services.md](docs/architecture/services.md) - Service responsibilities and ports
- [docs/api/README.md](docs/api/README.md) - API reference hub
- [docs/quickstart.md](docs/quickstart.md) - Interactive setup walkthrough and commands

Governance and quality:

- [CHANGELOG.md](CHANGELOG.md) - Release history and notable changes
- [CONTRIBUTING.md](CONTRIBUTING.md) - Development workflow expectations
- [docs/architecture/security.md](docs/architecture/security.md) - Security posture and hardening guidance

## Project Structure

```text
jan-server/
|-- apps/                  # Frontend applications
|   |-- web/               # Chat UI (React + Vite, port 3001)
|-- services/              # Go microservices
|   |-- llm-api/
|   |-- response-api/
|   |-- media-api/
|   |-- mcp-tools/
|   |-- template-api/      # Service scaffold template
|-- packages/              # Shared packages
|   |-- interfaces/        # Shared UI components (@janhq/interfaces)
|   |-- go-common/         # Shared Go utilities
|-- tools/jan-cli/         # Unified CLI for setup, ops, and development
|-- docs/                  # Documentation hub
|-- infra/docker/          # Compose fragments (infra, api, mcp, inference)
|-- integrations/          # Kong plugins, Keycloak realm/config
|-- config/                # Environment templates and schemas
|-- Makefile               # Build, test, deploy targets
|-- docker-compose.yml     # Root compose file
|-- .env.template          # Environment template
```

Key directories:

- `apps/` - frontend application (React + Vite chat UI).
- `services/` - source for each microservice plus local docs.
- `packages/` - shared packages (`interfaces` UI components, `go-common` Go utilities).
- `tools/jan-cli/` - unified CLI (`tools/jan-cli.sh` / `tools/jan-cli.ps1` wrappers).
- `docs/` - user, operator, and developer documentation (see [docs/README.md](docs/README.md)).
- `infra/docker/` - compose files included via `docker-compose.yml`.
- `integrations/` - Kong gateway and Keycloak configuration.
- `config/` - environment templates and schemas.

### Frontend Applications

| Application | Purpose                                      | Port | Source           | Tech Stack                          |
| ----------- | -------------------------------------------- | ---- | ---------------- | ----------------------------------- |
| Web App     | Chat UI for conversations                    | 3001 | `apps/web`       | React 19, Vite, TanStack Router     |

**Shared Package:** `packages/interfaces` - UI components (shadcn/ui), hooks, and utilities used by the web app.

```bash
# Run frontend app
cd apps/web && npm install && npm run dev       # http://localhost:3001

# Or via Docker
make up-web       # Start web app container
```

### Microservices Overview

| Service      | Purpose                                                            | Port(s)                      | Source                  | Docs                              |
| ------------ | ------------------------------------------------------------------ | ---------------------------- | ----------------------- | --------------------------------- |
| LLM API      | OpenAI-compatible chat, conversations, models                      | 8080 (direct), 8000 via Kong | `services/llm-api`      | `docs/api/llm-api/README.md`      |
| Response API | Multi-step orchestration using MCP tools                           | 8082                         | `services/response-api` | `docs/api/response-api/README.md` |
| Media API    | jan\_\* IDs, S3 ingest, media resolution                           | 8285                         | `services/media-api`    | `docs/api/media-api/README.md`    |
| MCP Tools    | Model Context Protocol tools (search, scrape, file search, python) | 8091                         | `services/mcp-tools`    | `docs/api/mcp-tools/README.md`    |

See [docs/architecture/services.md](docs/architecture/services.md) for dependency graphs and integration notes.

## Service Template

Create new microservices quickly with the template system:

```bash
# Generate new service from template
jan-cli dev scaffold my-new-service

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

**Documentation:**

- Template guide: [docs/guides/services-template.md](docs/guides/services-template.md)

## Development

### Quick Commands

```bash
make up-full              # Start the full stack (all APIs + infrastructure)
make health-check         # Verify all services are healthy
make swagger              # Regenerate OpenAPI docs
make logs                 # Tail container logs
make down                 # Stop containers (keep volumes)
make down-clean           # Stop containers and remove volumes
```

The Makefile exposes 100+ targets for building, running profiles, monitoring,
and database operations. See [docs/quickstart.md](docs/quickstart.md) for the
full command reference.

### Development Mode

Run services on your host for debugging:

```bash
# Start all services in Docker with host.docker.internal support
make dev-full

# Stop any service and run it on your host
docker compose stop llm-api
jan-cli dev run llm-api
```

See [Development Guide](docs/guides/development.md) for details on full Docker, dev-full (hybrid), and native execution modes.

## CLI Tool

Jan Server includes a unified CLI tool for configuration management, service operations, and development tasks.

### Quick Install

```bash
# Install globally (recommended)
make cli-install

# Add to PATH as instructed, then run from anywhere
jan-cli --version
jan-cli config validate
jan-cli service list
```

### Quick Usage (Without Installation)

Use the wrapper scripts under `tools/`:

```bash
# Linux/macOS/WSL
tools/jan-cli.sh config validate
tools/jan-cli.sh service list
tools/jan-cli.sh dev setup

# Windows PowerShell
tools\jan-cli.ps1 config validate
tools\jan-cli.ps1 service list
tools\jan-cli.ps1 dev setup
```

The wrapper scripts automatically build the CLI if needed.

### Available Commands

**Configuration Management:**

```bash
jan-cli config validate              # Validate configuration
jan-cli config export --format env   # Export as environment variables
jan-cli config show llm-api          # Show service configuration
jan-cli config k8s-values --env prod # Generate Kubernetes values
```

**Service Operations:**

```bash
jan-cli service list                 # List all services
jan-cli service logs llm-api         # Show service logs
jan-cli service status               # Check service health
```

**Development Tools:**

```bash
jan-cli dev setup                    # Setup development environment
jan-cli dev scaffold my-service      # Create new service from template
```

**Documentation:**

- Complete guide: [docs/guides/jan-cli.md](docs/guides/jan-cli.md)
- Command reference: [tools/jan-cli/README.md](tools/jan-cli/README.md)

## API Examples

### 1. Authentication

Kong (`http://localhost:8000`) fronts all `/llm/*` services and enforces Keycloak-issued JWTs or the custom API key plugin (`X-API-Key: sk_*`). Acquire temporary guest tokens at `POST /llm/auth/guest-login`, then include `Authorization: Bearer <token>` (or `X-API-Key`) on subsequent requests.

```bash
# Get guest token (no registration required)
curl -X POST http://localhost:8000/llm/auth/guest-login

# Sample response:
# {
#   "access_token": "eyJhbGc...",
#   "token_type": "Bearer",
#   "expires_in": 3600,
#   "refresh_token": "...",
#   "user_id": "guest-..."
# }
```

### 2. Chat Completion

```bash
# Simple chat completion
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'

# With media (using jan_* ID)
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b",
    "messages": [{
      "role": "user",
      "content": [
        {"type": "text", "text": "What's in this image?"},
        {"type": "image_url", "image_url": {"url": "jan_01hqr8v9k2x3f4g5h6j7k8m9n0"}}
      ]
    }]
  }'
```

### 3. Media Upload & Resolution

```bash
# Upload media (remote URL)
curl -X POST http://localhost:8285/v1/media \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "source": {
      "type": "remote_url",
      "url": "https://example.com/image.jpg"
    },
    "user_id": "user123"
  }'

# Response:
# {
#   "id": "jan_01hqr8v9k2x3f4g5h6j7k8m9n0",
#   "mime": "image/jpeg",
#   "bytes": 45678,
#   "presigned_url": "https://s3.menlo.ai/platform-dev/..."
# }

# Resolve jan_* ID to presigned URL
curl -X POST http://localhost:8285/v1/media/resolve \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"ids": ["jan_01hqr8v9k2x3f4g5h6j7k8m9n0"]}'
```

### 4. MCP Tools

```bash
# Google search
curl -X POST http://localhost:8000/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "google_search",
      "arguments": {"q": "latest AI news", "num": 5}
    }
  }'

# List available tools
curl -X GET http://localhost:8091/v1/mcp/tools
```

### 5. Response API (Multi-step Orchestration)

```bash
# Create response with tool execution
curl -X POST http://localhost:8082/v1/responses \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "input": "Search for the latest AI news and summarize the top 3 results"
  }'

# Response includes:
# - Tool execution trace
# - Final generated response
# - Execution metadata (depth, duration, etc.)
```

More examples: [API Documentation ->](docs/api/)

## Deployment

### Docker Compose Profiles

```bash
make up-full              # All services (including optional ones)
make up-gpu               # With GPU inference
make up-cpu               # CPU-only inference
make monitor-up           # Add monitoring stack

# Optional services (enabled via profiles)
docker compose --profile sandbox up -d  # Start sandbox for code execution
```

### Environment Configuration

Jan Server uses a single `.env` file at the repository root, generated from
`.env.template`:

```bash
# Create or update the root .env (idempotent)
make setup

# Then edit .env and choose which services run via COMPOSE_PROFILES
```

**Common secrets to set in `.env`:**

- `HF_TOKEN` - HuggingFace token for model downloads (https://huggingface.co/settings/tokens)
- `SERPER_API_KEY` - Serper API key for the Google Search tool (https://serper.dev)
- `POSTGRES_PASSWORD` - application database password
- `KEYCLOAK_ADMIN_PASSWORD` - Keycloak admin password

See [Deployment Guide](docs/guides/deployment.md) for production setup.

## Testing

Integration tests run through jan-cli api-test collections:

```bash
make test-all             # Run all collections
make test-auth            # Authentication flows (guest + user)
make test-conversation    # Conversation management
make test-response        # Response API orchestration
make test-media           # Media API operations
make test-mcp             # MCP tools integration
```

Testing guide: [docs/guides/testing.md](docs/guides/testing.md)

## Monitoring

Access monitoring dashboards:

- **Grafana**: http://localhost:3331 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Jaeger**: http://localhost:16686

See [Monitoring Guide](docs/guides/monitoring.md) for configuration.

## Technology Stack

| Layer              | Technology         | Version |
| ------------------ | ------------------ | ------- |
| **API Gateway**    | Kong               | 3.5     |
| **Services**       | Go (Gin framework) | 1.24.0  |
| **Database**       | PostgreSQL         | 18      |
| **Cache**          | Redis              | Latest  |
| **Auth**           | Keycloak (OIDC)    | 24.0.5  |
| **Inference**      | vLLM               | Latest  |
| **Search**         | SearXNG            | Latest  |
| **Code Execution** | SandboxFusion      | Latest  |
| **Observability**  | OpenTelemetry      | Latest  |
| **Metrics**        | Prometheus         | Latest  |
| **Tracing**        | Jaeger             | Latest  |
| **Dashboards**     | Grafana            | Latest  |
| **MCP Protocol**   | mark3labs/mcp-go   | Latest  |
| **Container**      | Docker Compose     | 2.0+    |
| **Orchestration**  | Kubernetes + Helm  | 1.28+   |

**Microservices:**

- LLM API: Go 1.24.0 with Gin, GORM, Wire DI
- Response API: Go 1.24.0 with Gin, GORM, Wire DI
- Media API: Go 1.24.0 with Gin, GORM, S3 SDK
- MCP Tools: Go 1.24.0 with JSON-RPC 2.0

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## License

[License information]

## Support

- Documentation: [docs/README.md](docs/README.md)
- Issue Tracker: https://github.com/janhq/jan-server/issues
- Discussions: https://github.com/janhq/jan-server/discussions

---

**Quick Start**: `make setup && make up-full` | **Documentation**: [docs/](docs/) | **API Docs**: http://localhost:8000/api/swagger/index.html
