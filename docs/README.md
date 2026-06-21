# Jan Server Documentation

Jan Server is an enterprise-grade microservices LLM API platform with Model Context Protocol (MCP) tool integration.

---

## Quick Start

| Step | Action | Command |
|------|--------|---------|
| 1 | Clone repository | `git clone https://github.com/janhq/server.git` |
| 2 | Run setup wizard | `cd jan-server && make quickstart` |
| 3 | Verify health | `make health-check` |

---

## Documentation Overview

### Getting Started

| Document | Description |
|----------|-------------|
| [Quickstart](quickstart.md) | Get up and running in 5 minutes |
| [Architecture](architecture/README.md) | System design and component responsibilities |
| [First API Call](api/llm-api/README.md#quick-start) | Sample curl request with authentication |

### API Documentation

| Document | Description |
|----------|-------------|
| [API Overview](api/README.md) | Authentication, base URLs, and service mapping |
| [LLM API](api/llm-api/README.md) | OpenAI-compatible chat completions, conversations, models |
| [Response API](api/response-api/README.md) | Multi-step tool orchestration |
| [Media API](api/media-api/README.md) | File uploads, jan_* ID resolution |
| [MCP Tools](api/mcp-tools/README.md) | JSON-RPC tool providers |
| [Endpoint Matrix](api/endpoint-matrix.md) | Complete endpoint reference |
| [API Examples](api/examples/README.md) | Ready-to-use code snippets |

### Development

| Document | Description |
|----------|-------------|
| [Development Guide](guides/development.md) | Local setup, Make targets, hybrid mode |
| [Testing Guide](guides/testing.md) | Integration tests, unit tests |
| [Configuration](configuration/README.md) | Environment variables, config precedence |
| [Service Template](guides/services-template.md) | Generate new microservices |
| [Conventions](conventions/conventions.md) | Code style, patterns, workflow |

### Deployment & Operations

| Document | Description |
|----------|-------------|
| [Deployment Guide](guides/deployment.md) | Docker Compose deployment |
| [Monitoring Guide](guides/monitoring.md) | Prometheus, Grafana, Jaeger |
| [Authentication](guides/authentication.md) | Kong + Keycloak integration |
| [Runbooks](runbooks/README.md) | On-call playbooks and procedures |
| [Troubleshooting](guides/troubleshooting.md) | Common issues and solutions |

---

## Architecture Summary

### Services

| Service | Port | Purpose |
|---------|------|---------|
| Kong Gateway | 8000 | API entry point, auth, routing |
| LLM API | 8080 | Chat completions, conversations |
| Response API | 8082 | Tool orchestration |
| Media API | 8285 | File storage, jan_* IDs |
| MCP Tools | 8091 | Search, scrape, code execution |

### Technology Stack

| Component | Technology |
|-----------|------------|
| Gateway | Kong 3.5 |
| Services | Go 1.24, Gin, Zerolog, Wire |
| Database | PostgreSQL 18 |
| Auth | Keycloak 24.0.5 |
| MCP | mark3labs/mcp-go v0.7.0 |
| Observability | OpenTelemetry, Prometheus, Grafana |

---

## Directory Structure

```
docs/
├── README.md                 # This file
├── quickstart.md             # Getting started guide
├── api/                      # API documentation
│   ├── README.md            # API overview
│   ├── llm-api/             # LLM API reference
│   ├── response-api/        # Response API reference
│   ├── media-api/           # Media API reference
│   ├── mcp-tools/           # MCP Tools reference
│   ├── examples/            # Code examples
│   └── endpoint-matrix.md   # All endpoints
├── architecture/            # Architecture docs
│   ├── README.md            # Architecture overview
│   ├── system-design.md     # How the pieces fit together
│   ├── services.md          # Service breakdown
│   ├── data-flow.md         # Request flow
│   ├── security.md          # Security model
│   ├── observability.md     # Monitoring and tracing
│   └── test-flows.md        # Test suite flows
├── guides/                  # How-to guides
│   ├── README.md            # Guides overview
│   ├── development.md       # Development setup
│   ├── testing.md           # Testing guide
│   ├── deployment.md        # Deployment guide
│   ├── authentication.md    # Auth integration
│   ├── monitoring.md        # Monitoring setup
│   └── troubleshooting.md   # Common issues
├── configuration/           # Configuration
│   ├── README.md            # Config overview
│   ├── env-var-mapping.md   # Environment variables
│   └── docker-compose.md    # Docker setup
├── conventions/             # Code standards
│   ├── conventions.md       # Coding standards
│   └── workflow.md          # Git workflow
└── runbooks/                # Operational procedures
    └── README.md            # Runbooks overview
```

---

## External Resources

- [OpenAI API Reference](https://platform.openai.com/docs/api-reference)
- [Model Context Protocol](https://modelcontextprotocol.io/)
- [Kong Gateway](https://konghq.com/)
- [Keycloak](https://www.keycloak.org/)
- [OpenTelemetry](https://opentelemetry.io/)

---

## Contributing

See the following files for contribution guidelines:

- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution process
- [CHANGELOG.md](../CHANGELOG.md) - Release notes
- [CLAUDE.md](../CLAUDE.md) - AI assistant guidelines