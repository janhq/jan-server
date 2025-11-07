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
- API Gateway: http://localhost:8000
- API Documentation: http://localhost:8000/v1/swagger/
- Keycloak Console: http://localhost:8085

**Full setup guide**: [Getting Started →](docs/getting-started/README.md)

## What is Jan Server?

Jan Server is an enterprise-grade LLM API platform that provides:
- **OpenAI-compatible API** for chat completions and conversations
- **MCP (Model Context Protocol)** tools for web search, scraping, and more
- **OAuth/OIDC authentication** via Keycloak
- **Full observability** with traces, metrics, and logs
- **Flexible deployment** with Docker Compose profiles

## Features

- ✅ OpenAI-compatible chat completions API
- ✅ MCP tools (google_search, web scraping)
- ✅ Conversation & message management
- ✅ Guest & user authentication (Keycloak)
- ✅ API gateway routing (Kong)
- ✅ Distributed tracing (Jaeger)
- ✅ Metrics & dashboards (Prometheus + Grafana)
- ✅ Hybrid development mode
- ✅ Comprehensive testing suite

## Documentation

- 🚀 [**Getting Started**](docs/getting-started/) - Setup & first steps
- 📖 [**Guides**](docs/guides/) - Development, testing, deployment
- 📡 [**API Reference**](docs/api/) - Endpoint documentation
- 🏗️ [**Architecture**](docs/architecture/) - System design
- 📋 [**Conventions**](docs/conventions/) - Code standards

**Full documentation**: [docs/README.md](docs/README.md)

## Project Structure

```
jan-server/
├── services/          # Microservices
│   ├── llm-api/      # LLM API service (Go)
│   └── mcp-tools/    # MCP tools service (Go)
├── monitoring/        # Observability configs
│   ├── grafana/      # Dashboards & provisioning
│   ├── prometheus.yml
│   └── otel-collector.yaml
├── docs/             # Documentation
│   ├── getting-started/
│   ├── guides/
│   ├── api/
│   └── architecture/
├── kong/             # API gateway config
├── keycloak/         # Auth server config
└── Makefile          # Build & run commands
```

## Development

### Quick Commands

```bash
# Start services
make up-full              # Full stack with Docker
make up-gpu               # With GPU inference
make up-cpu               # With CPU inference

# Development
make build-llm-api        # Build LLM API
make test                 # Run tests
make swag                 # Generate API docs

# Monitoring
make monitor-up           # Start monitoring stack
make monitor-logs         # View monitoring logs

# Logs & Status
make logs-llm-api         # View API logs
make health-check         # Check service health

# Cleanup
make down                 # Stop services
make clean                # Clean artifacts
```

### Hybrid Development Mode

Run services natively for faster iteration:

```bash
make hybrid-dev           # Setup hybrid environment
# Run API/MCP natively with hot reload
```

See [Development Guide](docs/guides/development.md) for details.

## API Examples

### Chat Completion

```bash
# Get guest token
curl -X POST http://localhost:8000/auth/guest

# Chat completion
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'
```

### MCP Tools

```bash
# Google search
curl -X POST http://localhost:8000/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "google_search",
      "arguments": {"q": "latest AI news"}
    }
  }'
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
cp .env.example .env
# Edit .env with your configuration
make setup
```

See [Deployment Guide](docs/guides/deployment.md) for production setup.

## Testing

```bash
# Run all tests
make test-all

# Specific test suites
make test-auth            # Authentication tests
make test-conversations   # Conversation tests
make test-mcp             # MCP tools tests
```

Testing guide: [docs/guides/testing.md](docs/guides/testing.md)

## Monitoring

Access monitoring dashboards:

- **Grafana**: http://localhost:3001 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Jaeger**: http://localhost:16686

See [Monitoring Guide](docs/guides/monitoring.md) for configuration.

## Technology Stack

| Layer | Technology |
|-------|------------|
| API Gateway | Kong 3.5 |
| Services | Go 1.21+ (Gin) |
| Database | PostgreSQL 16 |
| Auth | Keycloak (OIDC) |
| Inference | vLLM |
| Observability | OpenTelemetry, Prometheus, Jaeger, Grafana |
| MCP Protocol | mark3labs/mcp-go |

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
