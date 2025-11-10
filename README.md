# Jan Server

> A microservices LLM API platform with MCP tool integration

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-required-2496ED?logo=docker)](https://www.docker.com/)

## Quick Start

```bash
make setup && make up-full
```

**Services running at:**
- **API Gateway**: http://localhost:8000 (Kong)
- **LLM API**: http://localhost:8080 (OpenAI-compatible)
- **Response API**: http://localhost:8082 (Multi-step orchestration)
- **Media API**: http://localhost:8285 (Media management)
- **MCP Tools**: http://localhost:8091 (Tool integration)
- **API Documentation**: http://localhost:8000/v1/swagger/
- **Keycloak Console**: http://localhost:8085 (admin/admin)

**Full setup guide**: [Getting Started →](docs/getting-started/README.md)

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

-  **OpenAI-compatible chat completions API** with streaming support
-  **Response API** for multi-step tool orchestration (max depth: 8, timeout: 45s)
-  **Media API** with S3 storage, `jan_*` ID system, and presigned URLs
-  **MCP tools** (google_search, web scraping, code execution via SandboxFusion)
-  **Conversation & message management** with PostgreSQL persistence
-  **Guest & user authentication** via Keycloak OIDC
-  **API gateway routing** via Kong (v3.5)
-  **Distributed tracing** with Jaeger and OpenTelemetry
-  **Metrics & dashboards** with Prometheus + Grafana
-  **Hybrid development mode** for native execution with hot reload
-  **Comprehensive testing suite** with 6 Newman/Postman collections
-  **Service template system** for rapid microservice creation

## Documentation

📚 **Complete Documentation Hub**: [docs/README.md](docs/README.md)

**Quick Navigation**:
- 🚀 [Getting Started](docs/getting-started/README.md) - Setup & first steps (5 min)
- 📖 [Complete Index & Navigation](docs/INDEX.md) - Find what you need
- 📡 [API Reference](docs/api/README.md) - All 4 APIs documented
  - [LLM API](docs/api/llm-api/README.md) - Chat, conversations, models
  - [Response API](docs/api/response-api/README.md) - Multi-step orchestration
  - [Media API](docs/api/media-api/README.md) - File uploads, storage
  - [MCP Tools API](docs/api/mcp-tools/README.md) - Web search, scraping, code execution
- 💻 [Development Guide](docs/guides/development.md) - Local development
- 🧪 [Testing Guide](docs/guides/testing.md) - Test suites & procedures
- 🚀 [Deployment Guide](docs/guides/deployment.md) - Docker, Kubernetes, Hybrid
- 📊 [Monitoring Guide](docs/guides/monitoring.md) - Observability stack
- 🐛 [Troubleshooting Guide](docs/guides/troubleshooting.md) - Common issues & solutions
- 🏗️ [Architecture](docs/architecture/README.md) - System design
- ⚙️ [Configuration](config/README.md) - Environment setup
- ☸️ [Kubernetes Setup](k8s/SETUP.md) - Production deployment

**Documentation Quality**: ✅ All 82 .md files reviewed & updated for v0.2.0  
**Latest Audit**: [Project Completion Report](docs/PROJECT_COMPLETION_REPORT.md)

## Project Structure

```
jan-server/
├── services/              # Microservices (Go)
│   ├── llm-api/          # Core LLM orchestration (port 8080)
│   ├── response-api/     # Multi-step tool orchestration (port 8082)
│   ├── media-api/        # Media management with S3 (port 8285)
│   ├── mcp-tools/        # MCP protocol tools (port 8091)
│   └── template-api/     # Service template for new services
├── docker/               # Docker Compose configurations
│   ├── infrastructure.yml   # PostgreSQL, Redis, Keycloak
│   ├── services-api.yml     # API microservices
│   ├── services-mcp.yml     # MCP tools services
│   └── inference.yml        # vLLM inference
├── monitoring/           # Observability configs
│   ├── grafana/         # Dashboards & provisioning
│   ├── prometheus.yml   # Metrics collection
│   └── otel-collector.yaml # OpenTelemetry
├── k8s/                 # Kubernetes/Helm charts
│   └── jan-server/      # Helm chart v1.1.0
├── docs/                # Documentation
│   ├── getting-started/
│   ├── guides/
│   ├── api/
│   └── architecture/
├── kong/                # API gateway config
├── keycloak/            # Auth server config
└── Makefile             # 100+ build & run commands
```

## Microservices Architecture

Jan Server consists of 4 core microservices:

### 1. **LLM API** (Port 8080)
Core LLM orchestration service providing OpenAI-compatible endpoints.

**Features:**
- Chat completions with streaming support
- Conversation and message management
- Model provider abstraction (vLLM, OpenAI, Anthropic, etc.)
- Media resolution via Media API
- JWT authentication with Keycloak
- PostgreSQL persistence with GORM

**Endpoints:**
- `POST /v1/chat/completions` - Chat completion (streaming/non-streaming)
- `GET /v1/conversations` - List conversations
- `GET /v1/models` - List available models
- `GET /healthz` - Health check

### 2. **Response API** (Port 8082)
Multi-step tool orchestration for complex workflows.

**Features:**
- OpenAI Responses contract implementation
- Multi-step tool execution (max depth: 8)
- Tool timeout management (default: 45s)
- Integration with MCP Tools for tool discovery
- LLM API delegation for language generation
- PostgreSQL persistence for responses and executions

**Configuration:**
- `MAX_TOOL_EXECUTION_DEPTH=8` - Max recursive tool chain depth
- `TOOL_EXECUTION_TIMEOUT=45s` - Per-tool call timeout
- `LLM_API_URL` - LLM API base URL
- `MCP_TOOLS_URL` - MCP Tools base URL

**Endpoints:**
- `POST /v1/responses` - Create response with tool orchestration
- `GET /v1/responses/:id` - Get response by ID
- `GET /healthz` - Health check

### 3. **Media API** (Port 8285)
Media ingestion and resolution with S3 storage.

**Features:**
- `jan_*` ID system for persistent media references
- S3-compatible storage integration (Menlo S3)
- Presigned URL generation (5-minute TTL)
- Direct upload and remote URL ingestion
- Data URL support (base64 images)
- Deduplication by content hash
- PostgreSQL metadata store
- API key authentication (`X-Media-Service-Key`)

**Storage Flow:**
1. Client uploads via data URL or remote URL
2. Media API pushes to S3 (`platform-dev` bucket)
3. Returns `jan_id` + presigned URL
4. Other services resolve `jan_*` IDs to presigned URLs

**Endpoints:**
- `POST /v1/media` - Upload media (direct or remote)
- `POST /v1/media/prepare-upload` - Get presigned upload URL
- `POST /v1/media/resolve` - Resolve `jan_*` IDs to URLs
- `GET /healthz` - Health check

### 4. **MCP Tools** (Port 8091)
Model Context Protocol tools integration.

**Features:**
- JSON-RPC 2.0 protocol
- Google Search via Serper API
- Web scraping and content extraction
- Code execution via SandboxFusion
- Tool discovery and listing
- Extensible tool architecture

**Available Tools:**
- `google_search` - Web search with Serper API
- `web_scraper` - Extract content from URLs
- `code_executor` - Execute code in sandboxed environment

**Endpoints:**
- `POST /v1/mcp` - MCP JSON-RPC endpoint
- `GET /v1/mcp/tools` - List available tools
- `GET /healthz` - Health check

## Service Template

Create new microservices quickly with the template system:

```bash
# Generate new service from template
./scripts/new-service-from-template.ps1 -Name my-new-service

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
- Template guide: `docs/guides/services-template.md`
- Template README: `services/template-api/NEW_SERVICE_GUIDE.md`

## Development

### Quick Commands

```bash
# Start services
make up-full              # Full stack (all 4 APIs + infrastructure)
make up-gpu               # With GPU inference (vLLM)
make up-cpu               # CPU-only inference
make up                   # Infrastructure only (DB, Keycloak, Redis)

# Build services
make build-llm-api        # Build LLM API
make build-response-api   # Build Response API
make build-media-api      # Build Media API
make build-mcp            # Build MCP Tools

# Development
make hybrid-dev           # Setup hybrid environment
make test-all             # Run all test suites
make swag                 # Generate API docs

# Testing
make test-auth            # Authentication tests
make test-conversations   # Conversation tests
make test-response        # Response API tests
make test-media           # Media API tests
make test-mcp             # MCP tools tests
make test-e2e             # Gateway E2E tests

# Monitoring
make monitor-up           # Start monitoring stack
make monitor-logs         # View monitoring logs

# Logs & Status
make logs-llm-api         # View LLM API logs
make logs-response-api    # View Response API logs
make logs-media-api       # View Media API logs
make logs-mcp             # View MCP Tools logs
make health-check         # Check all services health

# Database
make db-migrate           # Run migrations
make db-reset             # Reset database
make db-seed              # Seed test data

# Cleanup
make down                 # Stop services
make clean                # Clean artifacts
make clean-all            # Clean everything (including volumes)
```

### Hybrid Development Mode

Run services natively for faster iteration:

```bash
make hybrid-dev           # Setup hybrid environment
# Run API/MCP natively with hot reload
```

See [Development Guide](docs/guides/development.md) for details.

## API Examples

### 1. Authentication

```bash
# Get guest token (no registration required)
curl -X POST http://localhost:8000/auth/guest

# Response:
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
  -H "X-Media-Service-Key: changeme-media-key" \
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
  -H "X-Media-Service-Key: changeme-media-key" \
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

More examples: [API Documentation →](docs/api/)

## Deployment

### Docker Compose Profiles

```bash
make up-full              # All services
make up-gpu               # With GPU inference
make up-cpu               # CPU-only inference
make monitor-up           # Add monitoring stack
```

### Environment Configuration

```bash
# Quick setup with defaults
make setup

# Or manually configure
cp config/secrets.env.example config/secrets.env
# Edit config/secrets.env with your API keys:
# - HF_TOKEN (HuggingFace token for model downloads)
# - SERPER_API_KEY (for Google Search tool)
# - POSTGRES_PASSWORD (database password)
# - KEYCLOAK_ADMIN_PASSWORD (Keycloak admin password)

# Available environment configs:
# - config/defaults.env       - Base configuration
# - config/development.env    - Docker development
# - config/testing.env        - Testing configuration
# - config/hybrid.env         - Native development
# - config/production.env.example - Production template
```

**Required secrets:**
- `HF_TOKEN` - HuggingFace token (get from https://huggingface.co/settings/tokens)
- `SERPER_API_KEY` - Serper API key (get from https://serper.dev)

See [Deployment Guide](docs/guides/deployment.md) for production setup.

## Testing

```bash
# Run all tests (6 Newman/Postman collections)
make test-all

# Specific test suites
make test-auth            # Authentication flows (guest + user)
make test-conversations   # Conversation management
make test-response        # Response API orchestration
make test-media           # Media API operations
make test-mcp             # MCP tools integration
make test-e2e             # Gateway end-to-end tests

# Test reports
# - JSON reports: newman.json
# - CLI output: Detailed results with assertions
```

**Test Collections:**
- `tests/automation/auth-postman-scripts.json` - Auth flows
- `tests/automation/conversations-postman-scripts.json` - Conversations
- `tests/automation/responses-postman-scripts.json` - Response API
- `tests/automation/media-postman-scripts.json` - Media API
- `tests/automation/mcp-postman-scripts.json` - MCP tools
- `tests/automation/test-all.postman.json` - Complete E2E suite

Testing guide: [docs/guides/testing.md](docs/guides/testing.md)

## Monitoring

Access monitoring dashboards:

- **Grafana**: http://localhost:3001 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Jaeger**: http://localhost:16686

See [Monitoring Guide](docs/guides/monitoring.md) for configuration.

## Technology Stack

| Layer | Technology | Version |
|-------|------------|---------|
| **API Gateway** | Kong | 3.5 |
| **Services** | Go (Gin framework) | 1.21+ |
| **Database** | PostgreSQL | 18 |
| **Cache** | Redis | Latest |
| **Auth** | Keycloak (OIDC) | Latest |
| **Inference** | vLLM | Latest |
| **Search** | SearXNG | Latest |
| **Code Execution** | SandboxFusion | Latest |
| **Observability** | OpenTelemetry | Latest |
| **Metrics** | Prometheus | Latest |
| **Tracing** | Jaeger | Latest |
| **Dashboards** | Grafana | Latest |
| **MCP Protocol** | mark3labs/mcp-go | Latest |
| **Container** | Docker Compose | 2.0+ |
| **Orchestration** | Kubernetes + Helm | 1.28+ |

**Microservices:**
- LLM API: Go 1.21+ with Gin, GORM, Wire DI
- Response API: Go 1.21+ with Gin, GORM, Wire DI
- Media API: Go 1.21+ with Gin, GORM, S3 SDK
- MCP Tools: Go 1.21+ with JSON-RPC 2.0

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## License

[License information]

## Support

- 📚 [Documentation](docs/README.md)
- 🐛 [Issue Tracker](https://github.com/janhq/jan-server/issues)
- 💬 [Discussions](https://github.com/janhq/jan-server/discussions)

---

**Quick Start**: `make setup && make up-full` | **Documentation**: [docs/](docs/) | **API Docs**: http://localhost:8000/v1/swagger/
