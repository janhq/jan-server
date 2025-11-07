# Jan Server Documentation

Welcome to the Jan Server documentation! This guide will help you find what you need.

## 🚀 New to Jan Server?

Start here: **[Getting Started Guide](getting-started/README.md)**

Quick commands:
```bash
make setup && make up-full
```

Services will be available at: http://localhost:8000

## 📚 Documentation Structure

| Section | Description |
|---------|-------------|
| **[Getting Started](getting-started/)** | Quick setup and first steps |
| **[Guides](guides/)** | Development, testing, deployment, and troubleshooting |
| **[API Reference](api/)** | Complete API documentation and examples |
| **[Architecture](architecture/)** | System design and technical details |
| **[Conventions](conventions/)** | Code standards and best practices |

## 📖 Quick Links

### For New Users
- 🆕 [Quick Start](getting-started/README.md) - Get up and running in 5 minutes
- 📡 [API Overview](api/README.md) - Understanding the APIs
- 🔐 [Authentication](api/llm-api/authentication.md) - How to authenticate

### For Developers
- 💻 [Development Guide](guides/development.md) - Local development workflow
- 🧪 [Testing Guide](guides/testing.md) - Running tests
- 🔄 [Hybrid Mode](guides/hybrid-mode.md) - Hybrid development setup
- 📊 [Monitoring](guides/monitoring.md) - Observability and monitoring

### For API Consumers
- 📡 [LLM API](api/llm-api/) - Chat completions and conversations
- 🛠️ [MCP Tools](api/mcp-tools/) - Model Context Protocol tools
- 💡 [Examples](api/llm-api/examples.md) - Code samples

### For Architects
- 🏗️ [Architecture Overview](architecture/README.md) - System architecture
- 🔒 [Security Model](architecture/security.md) - Security considerations
- 📈 [Observability](architecture/observability.md) - Monitoring stack

## 🆘 Need Help?

| Issue | Resource |
|-------|----------|
| **Service won't start** | [Troubleshooting Guide](guides/troubleshooting.md) |
| **API errors** | [API Documentation](api/README.md) |
| **Authentication issues** | [Auth Guide](api/llm-api/authentication.md) |
| **Performance problems** | [Monitoring Guide](guides/monitoring.md) |

## 🗂️ Common Tasks

### Setup & Installation
```bash
# Initial setup
make setup

# Start full stack
make up-full

# Start with monitoring
make up-full && make monitor-up
```

### Development
```bash
# Build LLM API
make build-llm-api

# Run tests
make test

# Generate API docs
make swag
```

### Monitoring
```bash
# Start monitoring stack
make monitor-up

# View dashboards
# Grafana: http://localhost:3001 (admin/admin)
# Prometheus: http://localhost:9090
# Jaeger: http://localhost:16686
```

## 📝 Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for contribution guidelines.

## 📋 Conventions

All code follows the conventions documented in [conventions/](conventions/):
- [Architecture Conventions](conventions/architecture.md)
- [Code Patterns](conventions/patterns.md)
- [Workflow](conventions/workflow.md)

## 🔄 What's New

See [CHANGELOG.md](../CHANGELOG.md) for version history and changes.

---

**Can't find what you're looking for?** Check the full documentation structure above or search within specific sections.
