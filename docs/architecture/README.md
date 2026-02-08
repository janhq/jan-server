# Jan Server Architecture

## Overview

Jan Server is built from multiple small services (microservices) that work together. Each service has a specific job.

## Main Parts

1. **[System Design](system-design.md)** - How all the pieces fit together
2. **[Services](services.md)** - What each service does
3. **[Data Flow](data-flow.md)** - How requests move through the system
4. **[Security](security.md)** - How we keep things secure
5. **[Observability](observability.md)** - How we monitor and debug
6. **[Test Flows](test-flows.md)** - How we test everything

## Quick Reference

### Service Ports (Docker Compose defaults)

| Service          | Port  | Access Notes                                                                  |
| ---------------- | ----- | ----------------------------------------------------------------------------- |
| **Kong Gateway** | 8000  | Entry point for `/llm/*`, `/responses/*`, `/media/*`, `/mcp` (routing + auth) |
| **LLM API**      | 8080  | Internal; exposed through Kong routes                                         |
| **Response API** | 8082  | Internal; streaming SSE via Kong `/responses`                                 |
| **Media API**    | 8285  | Internal; proxied by Kong `/media`                                            |
| **MCP Tools**    | 8091  | Internal; routed through Kong `/mcp`                                          |
| **Memory Tools** | 8090  | Internal; semantic memory service                                             |
| **Realtime API** | 8186  | Internal; WebRTC session management                                           |
| **Template API** | 8185  | Dev scaffold (not deployed by default)                                        |
| **Keycloak**     | 8085  | Admin console (protect behind VPN/SSO in production)                          |
| **vLLM**         | 8101  | Inference backend (local GPU/CPU profile)                                     |
| **Prometheus**   | 9090  | Dev-only monitoring UI (`make monitor-up`)                                    |
| **Jaeger**       | 16686 | Trace UI                                                                      |
| **Grafana**      | 3331  | Dashboards (admin/admin in dev)                                               |

### Technology Stack

| Component     | Technology                                   |
| ------------- | -------------------------------------------- |
| API Gateway   | Kong 3.5 + `keycloak-apikey` plugin          |
| Services      | Go 1.21+ (Gin framework, zerolog, wire DI)   |
| MCP Server    | mark3labs/mcp-go v0.7.0                      |
| ORM           | GORM + goose migrations                      |
| Database      | PostgreSQL 15/16 (Docker) / managed service  |
| Auth          | Keycloak (OpenID Connect)                    |
| Inference     | vLLM (OpenAI-compatible) or remote providers |
| Observability | OpenTelemetry Collector                      |
| Metrics       | Prometheus 2.48                              |
| Tracing       | Jaeger 1.51                                  |
| Dashboards    | Grafana 10.2                                 |

## How to Run It

You can run Jan Server in different ways:

### Docker Compose (For Development)

Use Docker Compose to run on your local computer:

- `make quickstart` - interactive wizard (creates `.env`, starts stack)
- `make up-full` - bring up all services (`docker compose.yml` + `infra/docker/*.yml`)
- `make up-vllm-gpu` / `make up-vllm-cpu` - start vLLM profile
- `make monitor-up` - start Prometheus, Grafana, Jaeger
- `make down` / `make down-clean` - stop stack (preserve or remove volumes)

### Kubernetes (For Production)

Use Kubernetes to run in the cloud or on servers:

- **Production**: `k8s/jan-server` Helm chart + managed Postgres + managed Keycloak
- **Hybrid**: Run inference locally while other services run in the cluster

**Note:** Kubernetes documentation is available in the `k8s/` directory.

## References

- [System Design Details](system-design.md)
- [Service Configurations](services.md)
- [Data Flow Patterns](data-flow.md)
- [Security Model](security.md)
- [Observability Guide](observability.md)
- [Test Flows & Diagrams](test-flows.md)
- [API Reference](../api/README.md)
- [Development Guide](../guides/development.md)
