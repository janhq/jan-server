# Guides

Comprehensive how-to guides for working with Jan Server.

## Available Guides

### Development
- **[Development Guide](development.md)** - Complete development workflow, setup, and best practices
- **[VS Code Guide](ide/vscode.md)** - VS Code debugging, tasks, and environment configuration
- **[Hybrid Mode](hybrid-mode.md)** - Run services natively for faster iteration and debugging
- **[Testing Guide](testing.md)** - Unit tests, integration tests, and testing best practices

### IDE Setup
- **[VS Code](ide/vscode.md)** - Complete VS Code configuration, debugging, and tasks

### Operations
- **[Monitoring](monitoring.md)** - Observability, metrics, traces, and dashboards
- **[Deployment](deployment.md)** - Kubernetes, Docker Compose, and hybrid deployment strategies
- **[Troubleshooting](troubleshooting.md)** - Common issues and solutions

### Special Topics
- **[MCP Testing](mcp-testing.md)** - Testing MCP (Model Context Protocol) integration

## Quick Links

### For Developers
| Task | Guide |
|------|-------|
| Setup local environment | [Development Guide](development.md) |
| Debug with VS Code | [VS Code Guide](ide/vscode.md) |
| Run services natively | [Hybrid Mode](hybrid-mode.md) |
| Write and run tests | [Testing Guide](testing.md) |
| Debug issues | [Troubleshooting](troubleshooting.md) |

### For DevOps
| Task | Guide |
|------|-------|
| Deploy to production | [Deployment Guide](deployment.md) |
| Setup monitoring | [Monitoring](monitoring.md) |
| Troubleshoot issues | [Troubleshooting](troubleshooting.md) |

### For QA
| Task | Guide |
|------|-------|
| Run integration tests | [Testing Guide](testing.md) |
| Test MCP tools | [MCP Testing](mcp-testing.md) |

## Common Tasks

### Development Workflow

```bash
# 1. Setup development environment
make setup
make hybrid-dev

# 2. Run service natively
cd services/llm-api
air  # hot reload

# 3. Make changes and test
make test

# 4. Run integration tests
make test-all
```

See [Development Guide](development.md) for details.

### Testing Workflow

```bash
# Unit tests
make test

# Integration tests  
make test-all

# Specific test suites
make test-auth
make test-conversations
make test-mcp
```

See [Testing Guide](testing.md) for details.

### Monitoring Setup

```bash
# Start monitoring stack
make monitor-up

# Access dashboards
# - Grafana: http://localhost:3001
# - Prometheus: http://localhost:9090
# - Jaeger: http://localhost:16686

# View logs
make monitor-logs
```

See [Monitoring Guide](monitoring.md) for details.

## Getting Help

Each guide includes:
-  Step-by-step instructions
-  Code examples
-  Common pitfalls
-  Troubleshooting tips
-  Related resources

### Need More Help?

- Check the [Troubleshooting Guide](troubleshooting.md)
- Review [Architecture Documentation](../architecture/README.md)
- See [API Reference](../api/README.md)
- Ask in [GitHub Discussions](https://github.com/janhq/jan-server/discussions)

---

**Back to**: [Documentation Home](../README.md) | **Next**: Choose a guide above
