# Guides

Comprehensive how-to guides for working with Jan Server. For incident response and on-call steps, see [../runbooks/README.md](../runbooks/README.md).

## Available Guides

### Development

- **[Development Guide](development.md)** - Complete development workflow, setup, and best practices
- **[Development Guide](development.md)** - Full Docker, dev-full (hybrid), and native execution modes
- **[Testing Guide](testing.md)** - Unit tests, integration tests, and testing best practices

### Operations (how-to)

- **[Monitoring](monitoring.md)** - Observability, metrics, traces, and dashboards
- **[Deployment](deployment.md)** - Kubernetes, Docker Compose, and hybrid deployment strategies
- **[Troubleshooting](troubleshooting.md)** - Common issues and solutions (links out to runbooks where applicable)

### Special Topics

- **[MCP Testing](mcp-testing.md)** - Testing MCP (Model Context Protocol) integration
- **[MCP Tools Configuration](mcp-admin-interface.md)** - Configure search, scrape, and code execution tools

## Quick Links

### For Developers

| Task                    | Guide                                                                              |
| ----------------------- | ---------------------------------------------------------------------------------- |
| Setup local environment | [Development Guide](development.md)                                                |
| Run services natively   | [Development Guide - Dev-Full Mode](development.md#dev-full-mode-hybrid-debugging) |
| Write and run tests     | [Testing Guide](testing.md)                                                        |
| Debug issues            | [Troubleshooting](troubleshooting.md)                                              |

### For DevOps

| Task                 | Guide                                                                  |
| -------------------- | ---------------------------------------------------------------------- |
| Deploy to production | [Deployment Guide](deployment.md)                                      |
| Setup monitoring     | [Monitoring](monitoring.md)                                            |
| Troubleshoot issues  | [Troubleshooting](troubleshooting.md) (see runbooks for on-call steps) |

### For QA

| Task                  | Guide                         |
| --------------------- | ----------------------------- |
| Run integration tests | [Testing Guide](testing.md)   |
| Test MCP tools        | [MCP Testing](mcp-testing.md) |

## Common Tasks

### Development Workflow

```bash
# 1. Setup development environment (.env + infra/docker/.env)
make setup

# 2. Start everything in Docker
make up-full

# 3. Switch a service to native mode (optional)
docker compose stop llm-api
./jan-cli.sh dev run llm-api   # macOS/Linux
.\jan-cli.ps1 dev run llm-api  # Windows

# 4. Run automated tests
make test-all                  # jan-cli api-test integration suites
go test ./services/llm-api/... # Unit tests from source
```

Use [Development Guide](development.md) for the complete workflow including full Docker, dev-full (hybrid), and native execution modes.

### Testing Workflow

```bash
# Integration tests (runs all Postman collections)
make test-all

# Specific test suites
make test-auth
make test-conversations
make test-mcp

# Unit tests from source
go test ./...
```

See [Testing Guide](testing.md) for details.

### Monitoring Setup

```bash
# Start monitoring stack
make monitor-up

# Access dashboards
# - Grafana: http://localhost:3331
# - Prometheus: http://localhost:9090
# - Jaeger: http://localhost:16686

# View logs
make monitor-logs
```

See [Monitoring Guide](monitoring.md) for details.

## Getting Help

Each guide includes:

- Step-by-step instructions
- Code examples
- Common pitfalls
- Troubleshooting tips
- Related resources

### Need More Help?

- Check the [Troubleshooting Guide](troubleshooting.md)
- Review [Architecture Documentation](../architecture/README.md)
- See [API Reference](../api/README.md)
- Ask in [GitHub Discussions](https://github.com/janhq/jan-server/discussions)

---

**Back to**: [Documentation Home](../README.md) | **Next**: Choose a guide above
