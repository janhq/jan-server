# Documentation Hub

Welcome to the Jan Server documentation. Use this page as a map to the rest of the guides.

> New to the project? Start with the [Documentation Index](INDEX.md).  
> Need to know what was reviewed? See the [Documentation Checklist](DOCUMENTATION_CHECKLIST.md).

## Structure

| Section | Description | Key Files |
|---------|-------------|-----------|
| **Getting Started** | Five minute setup for Docker Compose | [getting-started/README.md](getting-started/README.md) |
| **Guides** | Development, deployment, monitoring, IDE, troubleshooting | [guides/](guides/) |
| **API Reference** | LLM, Response, Media, MCP Tools APIs | [api/README.md](api/README.md) |
| **Services** | Responsibilities, ports, dependencies | [services.md](services.md) |
| **Architecture** | System design, security, observability, data flow | [architecture/](architecture/) |
| **Conventions** | Code standards and workflows | [conventions/CONVENTIONS.md](conventions/CONVENTIONS.md) |
| **Audits** | Latest documentation review | [AUDIT_SUMMARY.md](AUDIT_SUMMARY.md) |

## Quick Links

### New users
- [Quick Start](getting-started/README.md)
- [API Overview](api/README.md)
- [Authentication](api/llm-api/README.md#authentication)

### Developers
- [Development Guide](guides/development.md)
- [Testing Guide](guides/testing.md)
- [Hybrid Mode](guides/hybrid-mode.md)
- [Service Template](guides/services-template.md)
- [IDE Setup](guides/ide/vscode.md)

### API consumers
- [LLM API](api/llm-api/README.md)
- [Response API](api/response-api/README.md)
- [Media API](api/media-api/README.md)
- [MCP Tools](api/mcp-tools/README.md)
- [LLM Examples](api/llm-api/examples.md)

### Operators
- [Deployment Guide](guides/deployment.md)
- [Kubernetes Setup](../k8s/SETUP.md)
- [Monitoring Guide](guides/monitoring.md)
- [Authentication & Gateway](guides/authentication.md)
- [Troubleshooting](guides/troubleshooting.md)
- [Security Policy](../SECURITY.md)
- [Architecture Security](architecture/security.md)
- [Observability](architecture/observability.md)

## Need help?

| Issue | Resource |
|-------|----------|
| Services fail to start | [Troubleshooting Guide](guides/troubleshooting.md) |
| API errors | [API Reference](api/README.md) |
| Auth problems | [LLM API Auth](api/llm-api/README.md#authentication) |
| Performance issues | [Monitoring Guide](guides/monitoring.md) |

## Contributing and Updates
- Contribution process: [../CONTRIBUTING.md](../CONTRIBUTING.md)
- Security process: [../SECURITY.md](../SECURITY.md)
- Release notes: [../CHANGELOG.md](../CHANGELOG.md)

Still lost? Jump to the [Documentation Index](INDEX.md) or search within this directory.
