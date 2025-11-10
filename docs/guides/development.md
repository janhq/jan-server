# Jan Server - Development Guide

Welcome to the Jan Server development guide! This document provides comprehensive information for developers working on the jan-server project.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Project Structure](#project-structure)
3. [Development Workflows](#development-workflows)
4. [Configuration](#configuration)
5. [Testing](#testing)
6. [Troubleshooting](#troubleshooting)

## Quick Start

### Prerequisites

- **Docker** & **Docker Compose V2** (required)
- **Make** (required for convenience)
- **Go 1.25+** (for hybrid development)
- **Newman** (for integration tests)

### Initial Setup

```bash
# 1. Clone the repository
git clone https://github.com/janhq/jan-server.git
cd jan-server

# 2. Run setup script
make setup

# 3. Start all services
make up-full

# 4. Verify services are running
make health-check
```

### Access Points

After starting services:

- **LLM API**: http://localhost:8080
- **Media API**: http://localhost:8285
- **Keycloak**: http://localhost:8085 (admin/admin)
- **Kong Gateway**: http://localhost:8000
- **MCP Tools**: http://localhost:8091
- **SearXNG**: http://localhost:8086
- **Vector Store**: http://localhost:3015
- **SandboxFusion**: http://localhost:3010
- **Grafana**: http://localhost:3001 (admin/admin) - if monitoring enabled

## Project Structure

```
jan-server/
├── services/
│   ├── llm-api/           # Main LLM API service (Go)
│   ├── media-api/         # Media upload and management (Go)
│   ├── response-api/      # Response API service (Go)
│   └── mcp-tools/         # MCP tools integration service (Go)
├── scripts/
│   ├── lib/               # Helper scripts (bash & PowerShell)
│   ├── setup.sh/.ps1      # Initial setup scripts
│   └── hybrid-run-*.sh    # Hybrid development runners
├── config/
│   ├── defaults.env       # Default configuration
│   ├── development.env    # Development environment
│   ├── testing.env        # Testing environment
│   └── hybrid.env         # Hybrid development mode
├── docker/
│   ├── infrastructure.yml # Core infrastructure (DB, Auth, Gateway)
│   ├── services-api.yml   # LLM API service
│   ├── services-mcp.yml   # MCP services
│   ├── dev-hybrid.yml     # Hybrid development mode
│   ├── inference.yml      # vLLM GPU/CPU inference
│   └── observability.yml  # Monitoring stack
├── docker-compose.yml     # Main compose file (includes docker/*.yml)
└── Makefile               # Build system with all targets

```

## Development Workflows

### 1. Standard Docker Development

Best for: Full integration testing, production-like environment

```bash
# Start all services
make up-full

# View logs
make logs-api          # API logs only
make logs-mcp          # MCP logs only
make logs              # All logs

# Restart specific service
make restart-api
make restart-kong

# Stop everything
make down
```

### 2. Hybrid Development (Recommended for Active Development)

Best for: Debugging, hot-reloading, native IDE integration

#### Hybrid API Development

```bash
# 1. Start infrastructure in Docker
make hybrid-dev-api

# 2. Run APIs natively
make hybrid-run-api
make hybrid-run-media

# Or manually:
cd services/llm-api
source ../../config/hybrid.env
go run .
```

#### Hybrid MCP Development

```bash
# 1. Start MCP infrastructure
make hybrid-dev-mcp

# 2. Run MCP Tools natively
make hybrid-run-mcp
```

#### Debugging with Delve

```bash
# Terminal 1: Start infrastructure
make hybrid-infra-up

# Terminal 2: Run with debugger
make hybrid-debug-api    # API on port 2345
make hybrid-debug-mcp    # MCP on port 2346

# Connect your IDE debugger to localhost:2345 or localhost:2346
```

### 3. Testing Workflow

```bash
# Setup test environment
make test-setup

# Run all tests
make test-all              # Integration tests
make test                  # Unit tests
make test-coverage         # With coverage report

# Run specific test suites
make test-auth             # Authentication tests
make test-conversations    # Conversation API tests
make test-mcp-integration  # MCP integration tests

# Debug tests
make newman-debug

# Cleanup
make test-teardown
```

## Configuration

### Environment Files

| File | Purpose |
|------|---------|
| `.env.template` | Template with all variables documented |
| `config/defaults.env` | Default values for all environments |
| `config/development.env` | Local Docker development |
| `config/testing.env` | Integration test configuration |
| `config/hybrid.env` | Hybrid development (native services) |

### Switching Environments

```bash
# List available environments
make env-list

# Switch environment
make env-switch ENV=development
make env-switch ENV=testing
make env-switch ENV=hybrid

# Validate current environment
make env-validate
```

### Key Configuration Variables

#### Database
```bash
POSTGRES_USER=jan_user
POSTGRES_PASSWORD=jan_password
POSTGRES_DB=jan_llm_api
DB_DSN=postgres://jan_user:jan_password@localhost:5432/jan_llm_api
```

#### Authentication
```bash
KEYCLOAK_BASE_URL=http://localhost:8085
JWKS_URL=http://localhost:8085/realms/jan/protocol/openid-connect/certs
ISSUER=http://localhost:8090/realms/jan
AUDIENCE=jan-client
REFRESH_JWKS_INTERVAL=5m
```

#### API Services
```bash
HTTP_PORT=8080                    # LLM API port
MCP_TOOLS_HTTP_PORT=8091          # MCP Tools port
MEDIA_API_PORT=8285               # Media API port
```

#### MCP Services
```bash
SEARXNG_URL=http://localhost:8086
VECTOR_STORE_URL=http://localhost:3015
SANDBOXFUSION_URL=http://localhost:3010
```

#### Logging & Observability
```bash
LOG_LEVEL=debug                   # debug, info, warn, error
LOG_FORMAT=console                # console or json
OTEL_ENABLED=false                # OpenTelemetry tracing
AUTO_MIGRATE=true                 # Auto-run database migrations
```

## Testing

### Unit Tests

```bash
# Run all unit tests
make test

# Run tests for specific service
make test-api
make test-mcp

# Generate coverage report
make test-coverage
# Opens coverage.html in browser
```

### Integration Tests

Integration tests use Newman (Postman CLI) to test the full API:

```bash
# Run all integration tests
make test-all

# Run specific test suites
make test-auth              # OAuth/OIDC authentication
make test-conversations     # Conversation CRUD operations
make test-response          # Response API tests
make test-media             # Media API tests
make test-mcp-integration   # MCP tools functionality
make test-e2e               # Gateway end-to-end tests
```

### CI/CD

```bash
# Run full CI pipeline locally
make ci-test    # All tests
make ci-lint    # Linting
make ci-build   # Build verification
```

## Makefile Commands Reference

### Quick Reference

```bash
make help              # Show all commands
make help-core         # Setup & environment commands
make help-build        # Build & quality commands
make help-run          # Service management commands
make help-test         # Testing commands
make help-hybrid       # Hybrid development commands
make help-dev          # Developer utility commands
```

### Most Used Commands

| Command | Description |
|---------|-------------|
| `make setup` | Initial project setup |
| `make up-full` | Start all services |
| `make down` | Stop all services |
| `make logs-api` | View API logs |
| `make test-all` | Run all integration tests |
| `make health-check` | Check service health |
| `make hybrid-dev-api` | Start hybrid API development |
| `make swagger` | Generate API documentation |

## Troubleshooting

### Services Won't Start

```bash
# Check Docker is running
make check-deps

# Check service status
make dev-status

# Reset everything
make dev-reset
```

### Database Issues

```bash
# Reset database
make db-reset

# View database logs
docker compose logs api-db

# Open database console
make db-console
```

### Port Conflicts

If you see "port already in use" errors:

```bash
# Check what's using the ports
# Windows
netstat -ano | findstr ":8080"

# Linux/Mac
lsof -i :8080

# Use different ports by modifying .env:
HTTP_PORT=8081
KEYCLOAK_HTTP_PORT=8086
```

### Docker Issues

```bash
# Clean up Docker system
make docker-prune

# Remove all networks
make network-clean
make network-create

# Remove volumes ( DELETES DATA)
make volumes-clean
```

### Test Failures

```bash
# Run tests with debugging
make newman-debug

# Check service health before testing
make health-check

# View error logs
make logs-error
```

## Best Practices

### 1. Always Use Environment Files

Don't hardcode configuration. Use `.env` files and switch between environments:

```bash
# Development
make env-switch ENV=development
make up-full

# Testing
make env-switch ENV=testing
make test-all
```

### 2. Use Hybrid Mode for Active Development

Faster iteration, better debugging:

```bash
make hybrid-dev-api
# In another terminal:
make hybrid-run-api
make hybrid-run-media
```

### 3. Run Tests Before Committing

```bash
make fmt          # Format code
make lint         # Check for issues
make test         # Unit tests
make test-all     # Integration tests
```

### 4. Keep Documentation Updated

```bash
# After API changes, regenerate swagger
make swagger

# After schema changes
make db-migrate
```

### 5. Clean Up Regularly

```bash
# Clean build artifacts
make dev-clean

# Clean Docker resources
make docker-prune
```

## Additional Resources

- [Testing Guide](TESTING.md) - Comprehensive testing documentation
- [Hybrid Mode Guide](HYBRID_MODE.md) - Detailed hybrid development workflow
- [Architecture Documentation](architecture.md) - System architecture overview
- [MCP Integration](../services/mcp-tools/README.md) - MCP tools documentation

## Getting Help

1. Check this documentation
2. Run `make help` for command reference
3. Check logs: `make logs-error`
4. Review [troubleshooting](#troubleshooting) section
5. Ask in team chat or create an issue

---
