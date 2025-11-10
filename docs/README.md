# Jan Server Documentation

Welcome to the Jan Server documentation! This guide will help you find what you need.

> 📍 **New here?** Start with the [Complete Documentation Index & Navigation Guide](INDEX.md) for easy navigation!  
> ✅ **Want quality info?** Check out the [Documentation Checklist](DOCUMENTATION_CHECKLIST.md) for what's verified.

##  New to Jan Server?

**Choose your deployment:**
- **Docker Compose (Local Development)**: [Getting Started Guide](getting-started/README.md)
- **Kubernetes (Production/Staging)**: [Kubernetes Setup Guide](../k8s/SETUP.md)

Quick Docker Compose setup:
```bash
make setup && make up-full
```

Services will be available at: http://localhost:8000

## 📚 Documentation Structure

| Section | Description | Key Files |
|---------|-------------|-----------|
| **[Getting Started](getting-started/)** | Quick setup and first steps | [Quick Start](getting-started/README.md) |
| **[Guides](guides/)** | Development, deployment, monitoring | [Development](guides/development.md), [Testing](guides/testing.md), [Deployment](guides/deployment.md) |
| **[API Reference](api/)** | Complete API documentation | [LLM API](api/llm-api/), [Media API](api/media-api/), [Response API](api/response-api/), [MCP Tools](api/mcp-tools/) |
| **[Architecture](architecture/)** | System design and technical details | [System Design](architecture/system-design.md) |
| **[Conventions](conventions/)** | Code standards and best practices | [Code Conventions](conventions/CONVENTIONS.md) |
| **[Audit Summary](AUDIT_SUMMARY.md)** | Documentation review and updates | [Nov 2025 Audit](AUDIT_SUMMARY.md) |

## 📖 Quick Links

### For New Users
- 🆕 [Quick Start](getting-started/README.md) - Get up and running in 5 minutes
- 📡 [API Overview](api/README.md) - Understanding the APIs
- 🔐 [Authentication](api/llm-api/authentication.md) - How to authenticate

### For Developers
- 💻 [Development Guide](guides/development.md) - Local development workflow
- 🧪 [Testing Guide](guides/testing.md) - Unit and integration tests
- 🔄 [Hybrid Mode](guides/hybrid-mode.md) - Native + Docker development
- 📊 [Monitoring Guide](guides/monitoring.md) - Observability and tracing
- 🧱 [Service Template](guides/services-template.md) - Create new microservices
- 🐛 [Troubleshooting](guides/troubleshooting.md) - Common issues and solutions
- 🖥️ [IDE Setup](guides/ide/) - VS Code debugging and configuration

### For API Consumers
- 📡 **[API Overview](api/)** - All 4 APIs (LLM, Media, Response, MCP)
- 🔤 **[LLM API](api/llm-api/)** - Chat completions, conversations, models
- 🎬 **[Response API](api/response-api/)** - Multi-step tool orchestration
- 🖼️ **[Media API](api/media-api/)** - Media upload and `jan_*` ID resolution
- � **[MCP Tools](api/mcp-tools/)** - Web search, scraping, code execution
- � **[Code Examples](api/#sdk--client-libraries)** - Python, JavaScript examples

### For Operators & DevOps
- 🚀 [Deployment Guide](guides/deployment.md) - Docker Compose, Kubernetes, Hybrid
- ☸️ [Kubernetes Setup](../k8s/SETUP.md) - Step-by-step K8s deployment
- 📊 [Monitoring Guide](guides/monitoring.md) - Prometheus, Grafana, Jaeger
- 🔐 [Security](../SECURITY.md) - Secrets management and best practices
- 🐛 [Troubleshooting](guides/troubleshooting.md) - Common issues and debugging
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
