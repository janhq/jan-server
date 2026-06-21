# Service Overview

Jan Server ships four core services plus a scaffold template and shared infrastructure. Use this document to understand how they fit together and where to look in the codebase.

## Core Services

| Service                     | Purpose                                                                       | Port(s)                      | Source                  | Primary Docs                                                     |
| --------------------------- | ----------------------------------------------------------------------------- | ---------------------------- | ----------------------- | ---------------------------------------------------------------- |
| **LLM API**                 | OpenAI-compatible chat completions, conversation storage, model management    | 8080 (direct), 8000 via Kong | `services/llm-api`      | [../api/llm-api/README.md](../api/llm-api/README.md)             |
| **Response API**            | Multi-step orchestration, tool chaining, integration with MCP Tools           | 8082                         | `services/response-api` | [../api/response-api/README.md](../api/response-api/README.md)   |
| **Media API**               | Binary ingestion, jan\_\* IDs, S3 storage and resolution                      | 8285                         | `services/media-api`    | [../api/media-api/README.md](../api/media-api/README.md)         |
| **MCP Tools**               | Model Context Protocol tools (web search, scraping, file search, python exec) | 8091                         | `services/mcp-tools`    | [../api/mcp-tools/README.md](../api/mcp-tools/README.md)         |
| **Template API (scaffold)** | Go microservice blueprint for new services (not deployed)                     | 8185                         | `services/template-api` | [../guides/services-template.md](../guides/services-template.md) |

## Configuration

All services use the **centralized configuration system** at `packages/go-common/config/`:

- **Type-safe:** Go structs with compile-time validation
- **YAML defaults:** `config/defaults.yaml` for base configuration
- **Environment overrides:** Service-specific env vars (e.g., `LLM_API_HTTP_PORT`)
- **Kubernetes values:** Auto-generated from configuration structs
- **CLI tool:** `jan-cli config` for validation and inspection

See [Configuration Documentation](../configuration/README.md) for details.

## Infrastructure Components

For detailed infrastructure architecture (Kong, Keycloak, databases, observability), see [System Design](system-design.md#2-architecture-layers).

## Creating a New Service

### Quick Start

```bash
# Generate from template
jan-cli dev scaffold my-service
```

### Configuration Setup

New services should use the centralized configuration system:

1. **Define service config in `packages/go-common/config/types.go`:**

```go
type ServiceConfig struct {
HTTP HTTPConfig `yaml:"http"`
Database DatabaseConfig `yaml:"database"`
// Add service-specific fields
}
```

2. **Regenerate config files:**

```bash
make config-generate
```

3. **Load config in your service:**

```go
import "jan-server/packages/go-common/config"

cfg, _:= config.Load()
serviceCfg, _:= cfg.GetServiceConfig("my-service")
```

4. **Update deployment configs:**

- Add service to `docker/services-api.yml`
- Generate K8s values: `jan-cli config k8s-values --env production`

See [Configuration System](../configuration/README.md) and [Service Template Guide](../guides/services-template.md) for the scaffold and full instructions.

### Documentation Requirements

1. Update `docs/architecture/services.md` (this file) with new service row
2. Create `docs/api/<service>/README.md` with API reference
3. Add service to `docs/README.md` navigation
4. Update `k8s/jan-server/values.yaml` if deploying to Kubernetes

## Service Interactions

- **LLM API -> Media API**: LLM API resolves `jan_*` IDs before sending payloads to vLLM or upstream providers (`MEDIA_RESOLVE_URL` env var).
- **Response API -> LLM API**: Response API delegates final language generation to LLM API (`LLM_API_URL`).
- **Response API -> MCP Tools**: orchestrated tool calls are issued via JSON-RPC (`MCP_TOOLS_URL`).
- **MCP Tools -> Infrastructure**: web search uses a cascading fallback chain (Serper -> Exa -> Tavily -> SearXNG); also uses Vector Store and SandboxFusion to execute user requests.

Keep this document updated whenever a service is added, renamed, or retires.
