# Jan Server Architecture

## Overview

Jan Server is a modular, microservices-based LLM API platform with enterprise-grade authentication, API gateway routing, and flexible inference backend support. The system provides OpenAI-compatible API endpoints for chat completions, conversations, and model management.

## Architecture Components

The system is organized into several layers:

1. **[System Design](system-design.md)** - Overall architecture diagram and layer descriptions
2. **[Services](services.md)** - Detailed service descriptions and configurations
3. **[Data Flow](data-flow.md)** - Request flow patterns and interactions
4. **[Security](security.md)** - Authentication, authorization, and security considerations
5. **[Observability](observability.md)** - Monitoring, tracing, and logging

## Quick Reference

### Service Ports

| Service | Port | Access |
|---------|------|--------|
| Kong Gateway | 8000 | External (public API) |
| LLM-API | 8080 | Internal |
| MCP-Tools | 8091 | Internal |
| Keycloak | 8085 | External (admin console) |
| vLLM | 8000 | Internal |
| Prometheus | 9090 | External (monitoring) |
| Jaeger | 16686 | External (tracing UI) |
| Grafana | 3001 | External (dashboards) |

### Technology Stack

| Component       | Technology                     |
|-----------------|--------------------------------|
| API Gateway     | Kong 3.5                       |
| Services        | Go 1.21+ (Gin framework)       |
| MCP Server      | mark3labs/mcp-go v0.7.0        |
| ORM             | GORM                           |
| Database        | PostgreSQL 16                  |
| Auth            | Keycloak (OpenID Connect)      |
| Inference       | vLLM (OpenAI-compatible)       |
| Observability   | OpenTelemetry Collector        |
| Metrics         | Prometheus 2.48                |
| Tracing         | Jaeger 1.51                    |
| Dashboards      | Grafana 10.2                   |

## Deployment Profiles

Docker Compose services are organized by profiles for flexible deployment:

- **Infrastructure only**: `make up` (api-db, keycloak-db, keycloak)
- **With LLM API**: `make up-llm-api` (+ llm-api service)
- **With Kong**: `make up-kong` (+ kong gateway)
- **Full stack**: `make up-full` (all services)
- **GPU inference**: `make up-gpu` (+ vllm with GPU)
- **CPU inference**: `make up-cpu` (+ vllm CPU-only)
- **Monitoring stack**: `make monitor-up` (prometheus, jaeger, grafana, otel-collector)

## References

- [System Design Details](system-design.md)
- [Service Configurations](services.md)
- [Data Flow Patterns](data-flow.md)
- [Security Model](security.md)
- [Observability Guide](observability.md)
- [API Reference](../api/README.md)
- [Development Guide](../guides/development.md)
