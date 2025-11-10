# Documentation Index & Navigation Guide

**Last Updated**: November 10, 2025  
**Documentation Version**: v0.2.0  
**Status**: Production Ready ✅

---

## 🎯 Start Here

### First Time with Jan Server?
1. **[Quick Start (5 minutes)](getting-started/README.md)** - Get up and running
2. **[Architecture Overview](architecture/README.md)** - Understand the system
3. **[API Overview](api/README.md)** - Learn the available APIs
4. **[Your First Request](api/llm-api/README.md#quick-start)** - Make your first API call

---

## 📚 Complete Documentation Map

### For New Users
```
Getting Started in 5 Minutes
├── [Quick Start Guide](getting-started/README.md)
├── [System Architecture Overview](architecture/README.md)
├── [Quick Reference (100+ commands)](QUICK_REFERENCE.md)
└── [First API Call Example](api/llm-api/README.md#quick-start)
```

### For Developers
```
Development & Contributions
├── [Development Guide](guides/development.md)
│   ├── Local Development Setup
│   ├── Hybrid Mode (Native + Docker)
│   ├── Building Services
│   └── Configuration Management
├── [Testing Guide](guides/testing.md)
│   ├── Unit Testing
│   ├── Integration Testing
│   ├── Test Suites (6 total)
│   └── Coverage Reporting
├── [IDE Setup](guides/ide/)
│   └── VS Code Debugging & Configuration
├── [Service Creation](guides/services-template.md)
│   └── Build a new microservice
├── [Troubleshooting](guides/troubleshooting.md)
│   └── Common issues & solutions
├── [Conventions](conventions/CONVENTIONS.md)
│   ├── Code Standards
│   ├── Design Patterns
│   ├── Architecture Patterns
│   └── Development Workflow
└── [System Design Deep Dive](architecture/system-design.md)
```

### For API Consumers
```
API Documentation
├── [API Overview & Authentication](api/README.md)
├── [LLM API Reference](api/llm-api/README.md)
│   ├── Chat Completions (OpenAI-compatible)
│   ├── Conversation Management
│   ├── Model Listing
│   └── Streaming Responses
├── [Response API Reference](api/response-api/README.md)
│   ├── Multi-Step Tool Execution
│   ├── Tool Orchestration
│   ├── Response Management
│   └── Configuration
├── [Media API Reference](api/media-api/README.md)
│   ├── Media Upload
│   ├── Presigned URLs
│   ├── jan_* ID Resolution
│   ├── S3 Storage
│   └── Media Deduplication
└── [MCP Tools API Reference](api/mcp-tools/README.md)
    ├── Web Search
    ├── Web Scraping
    ├── Code Execution
    ├── Tool Chaining
    └── Integration with Response API
```

### For DevOps & Operations
```
Deployment & Infrastructure
├── [Deployment Guide](guides/deployment.md)
│   ├── Docker Compose Setup
│   ├── Kubernetes Deployment
│   ├── Minikube Configuration
│   ├── Cloud Deployment
│   ├── Hybrid Mode Deployment
│   └── Environment Configuration
├── [Kubernetes Setup Guide](../k8s/SETUP.md)
│   ├── Helm Chart Overview (v1.1.0)
│   ├── Step-by-Step Kubernetes Setup
│   ├── Database Configuration
│   ├── Port-Forward Examples
│   └── Production Deployment
├── [Kubernetes README](../k8s/README.md)
│   └── Helm chart reference
├── [Monitoring & Observability](guides/monitoring.md)
│   ├── OpenTelemetry Collector
│   ├── Prometheus Setup
│   ├── Jaeger Tracing
│   ├── Grafana Dashboards
│   └── Service Health Monitoring
├── [Configuration Management](../config/README.md)
│   ├── Default Configuration
│   ├── Development Environment
│   ├── Testing Environment
│   ├── Hybrid Mode Environment
│   └── Production Secrets
├── [Troubleshooting Guide](guides/troubleshooting.md)
│   ├── Service Startup Issues
│   ├── Database Problems
│   ├── API Errors
│   ├── Authentication Issues
│   ├── Docker Issues
│   ├── Kubernetes Issues
│   └── Performance Issues
└── [Security Best Practices](../SECURITY.md)
    ├── Vulnerability Reporting
    ├── Secrets Management
    ├── Environment Setup
    └── Security Checklist
```

### For Architects & Technical Leaders
```
Architecture & System Design
├── [System Architecture Overview](architecture/README.md)
├── [Detailed System Design](architecture/system-design.md)
│   ├── Microservices Architecture
│   ├── Service Interaction Flows
│   ├── Data Flow Diagrams
│   └── Technology Stack
├── [Architecture Patterns & Conventions](conventions/conventions-architecture.md)
├── [Design Patterns & Conventions](conventions/conventions-patterns.md)
├── [Kubernetes Architecture](../k8s/SETUP.md)
├── [Service Structure](guides/services-template.md)
└── [Complete Audit Summary](AUDIT_SUMMARY.md)
    ├── All Changes Made
    ├── Coverage Statistics
    ├── Quality Verification
    └── Next Steps
```

---

## 🎯 Quick Navigation by Task

### "I want to..."

#### ...get started quickly
→ [Quick Start Guide](getting-started/README.md) (5 minutes)

#### ...understand the system
→ [Architecture Overview](architecture/README.md)

#### ...learn the APIs
→ [API Overview](api/README.md) then pick your API:
- [LLM API](api/llm-api/README.md) - Chat, conversations, models
- [Response API](api/response-api/README.md) - Multi-step orchestration
- [Media API](api/media-api/README.md) - File uploads, storage
- [MCP Tools API](api/mcp-tools/README.md) - Web search, scraping, code execution

#### ...make my first API call
→ [LLM API Quick Start](api/llm-api/README.md#quick-start)

#### ...set up development environment
→ [Development Guide](guides/development.md)

#### ...test my changes
→ [Testing Guide](guides/testing.md)

#### ...use hybrid mode (native + Docker)
→ [Hybrid Mode Guide](guides/hybrid-mode.md)

#### ...debug issues
→ [VS Code Setup](guides/ide/)

#### ...run tests
→ [Testing Guide](guides/testing.md)

#### ...create a new service
→ [Service Template Guide](guides/services-template.md)

#### ...resolve an issue
→ [Troubleshooting Guide](guides/troubleshooting.md)

#### ...monitor the system
→ [Monitoring Guide](guides/monitoring.md)

#### ...deploy to production
→ [Deployment Guide](guides/deployment.md)

#### ...deploy to Kubernetes
→ [Kubernetes Setup](../k8s/SETUP.md)

#### ...understand security
→ [Security Guide](../SECURITY.md)

#### ...find all commands
→ [Quick Reference (100+ commands)](QUICK_REFERENCE.md)

#### ...report a vulnerability
→ [Security Guide - Reporting](../SECURITY.md#reporting-security-vulnerabilities)

---

## 📑 Documentation Files by Type

### Main Documentation
- 📖 [README.md](../README.md) - Project overview
- 📋 [CHANGELOG.md](../CHANGELOG.md) - Version history and release notes
- 🔒 [SECURITY.md](../SECURITY.md) - Security best practices
- 👥 [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution guidelines

### Documentation Hub
- 📚 [docs/README.md](README.md) - Documentation index
- 📊 [docs/AUDIT_SUMMARY.md](AUDIT_SUMMARY.md) - Audit results and changes
- ✅ [docs/DOCUMENTATION_CHECKLIST.md](DOCUMENTATION_CHECKLIST.md) - Quality checklist
- ⚡ [docs/QUICK_REFERENCE.md](QUICK_REFERENCE.md) - 100+ commands

### Getting Started
- 🚀 [docs/getting-started/README.md](getting-started/README.md) - 5-minute setup

### API Documentation
- 📡 [docs/api/README.md](api/README.md) - All APIs overview
- 🤖 [docs/api/llm-api/README.md](api/llm-api/README.md) - LLM API (Chat, conversations, models)
- 🔄 [docs/api/response-api/README.md](api/response-api/README.md) - Response API (Multi-step orchestration)
- 🖼️ [docs/api/media-api/README.md](api/media-api/README.md) - Media API (Upload, storage, resolution)
- 🧠 [docs/api/mcp-tools/README.md](api/mcp-tools/README.md) - MCP Tools (Search, scraping, execution)

### Guides
- 💻 [docs/guides/development.md](guides/development.md) - Local development
- 🧪 [docs/guides/testing.md](guides/testing.md) - Testing procedures
- 🚀 [docs/guides/deployment.md](guides/deployment.md) - Deployment guide
- 📊 [docs/guides/monitoring.md](guides/monitoring.md) - Observability stack
- 🔄 [docs/guides/hybrid-mode.md](guides/hybrid-mode.md) - Native development
- 🧬 [docs/guides/mcp-testing.md](guides/mcp-testing.md) - MCP tools testing
- 🏗️ [docs/guides/services-template.md](guides/services-template.md) - Service creation
- 🐛 [docs/guides/troubleshooting.md](guides/troubleshooting.md) - Common issues
- 🖥️ [docs/guides/ide/](guides/ide/) - IDE setup (VS Code)

### Architecture
- 🏗️ [docs/architecture/README.md](architecture/README.md) - Architecture overview
- 📐 [docs/architecture/system-design.md](architecture/system-design.md) - System design

### Conventions
- 📋 [docs/conventions/CONVENTIONS.md](conventions/CONVENTIONS.md) - Code standards
- 🎯 [docs/conventions/conventions-patterns.md](conventions/conventions-patterns.md) - Design patterns
- 🏛️ [docs/conventions/conventions-architecture.md](conventions/conventions-architecture.md) - Architecture patterns
- 🔄 [docs/conventions/conventions-workflow.md](conventions/conventions-workflow.md) - Dev workflow

### Infrastructure
- ⚙️ [../config/README.md](../config/README.md) - Configuration files
- ☸️ [../k8s/README.md](../k8s/README.md) - Kubernetes overview
- ☸️ [../k8s/SETUP.md](../k8s/SETUP.md) - Kubernetes setup guide

### Service Documentation
- 🤖 [../services/llm-api/README.md](../services/llm-api/README.md) - LLM API service
- 🔄 [../services/response-api/README.md](../services/response-api/README.md) - Response API service
- 🖼️ [../services/media-api/README.md](../services/media-api/README.md) - Media API service
- 🧠 [../services/mcp-tools/README.md](../services/mcp-tools/README.md) - MCP Tools service
- 🧬 [../services/mcp-tools/INTEGRATION.md](../services/mcp-tools/INTEGRATION.md) - MCP Integration
- ⚙️ [../services/mcp-tools/MCP_PROVIDERS.md](../services/mcp-tools/MCP_PROVIDERS.md) - MCP Providers
- 📦 [../services/template-api/README.md](../services/template-api/README.md) - Template service
- 🏗️ [../services/template-api/NEW_SERVICE_GUIDE.md](../services/template-api/NEW_SERVICE_GUIDE.md) - Service creation

---

## 🔗 External Resources

### API Standards
- [OpenAI API Documentation](https://platform.openai.com/docs/api-reference) - Referenced by LLM API
- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification) - Used by MCP Tools
- [Model Context Protocol](https://modelcontextprotocol.io/) - Foundation for MCP Tools

### Technologies
- [Kong Gateway](https://konghq.com/)
- [Keycloak](https://www.keycloak.org/)
- [PostgreSQL](https://www.postgresql.org/)
- [OpenTelemetry](https://opentelemetry.io/)
- [Kubernetes](https://kubernetes.io/)
- [Helm](https://helm.sh/)

---

## 📊 Documentation Statistics

- **Total Files**: 82 .md files
- **Total Lines**: 15,000+ lines
- **API Documentation**: 1,500 lines
- **Guides**: 2,500 lines
- **Architecture**: 500 lines
- **Conventions**: 400 lines
- **Coverage**: 100% of services, APIs, and deployment methods

---

## ✨ Documentation Quality

- ✅ All documentation is up-to-date with v0.2.0 code
- ✅ No "Coming Soon" placeholders
- ✅ All 4 microservices documented
- ✅ All APIs documented with examples
- ✅ All deployment methods covered
- ✅ Comprehensive troubleshooting guide
- ✅ 100% service coverage
- ✅ Production-ready documentation

---

## 🔄 Documentation Maintenance

### Last Major Audit
- **Date**: November 10, 2025
- **Status**: ✅ Complete
- **Coverage**: 100%
- **Summary**: [Audit Summary](AUDIT_SUMMARY.md)
- **Checklist**: [Documentation Checklist](DOCUMENTATION_CHECKLIST.md)

### Next Review
- **Schedule**: Q1 2026
- **Type**: Comprehensive audit
- **Focus**: Consistency with latest code

---

## 💡 Tips for Finding Information

1. **Use this page** as your navigation hub
2. **Use Ctrl+F (or Cmd+F)** to search within pages
3. **Check the table of contents** at the top of each file
4. **Follow the links** to related documentation
5. **Check [Quick Reference](QUICK_REFERENCE.md)** for common commands

---

## 🆘 Need Help?

- **First time user?** → [Quick Start](getting-started/README.md)
- **Having issues?** → [Troubleshooting Guide](guides/troubleshooting.md)
- **Need a command?** → [Quick Reference](QUICK_REFERENCE.md)
- **Looking for an API?** → [API Overview](api/README.md)
- **Security issue?** → [Security Guide](../SECURITY.md)
- **Found a bug?** → [Contributing Guide](../CONTRIBUTING.md)

---

**Last Updated**: November 10, 2025  
**Status**: ✅ Production Ready  
**Next Review**: Q1 2026
