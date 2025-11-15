# Service Overview

Jan Server ships four core services plus shared infrastructure. Use this document to understand how they fit together and where to look in the codebase.

## Core Services

| Service | Purpose | Port(s) | Source | Primary Docs |
|---------|---------|---------|--------|--------------|
| **LLM API** | OpenAI-compatible chat completions, conversation storage, model management | 8080 (direct), 8000 via Kong | `services/llm-api` | [api/llm-api/README.md](api/llm-api/README.md), [api/llm-api/examples.md](api/llm-api/examples.md) |
| **Response API** | Multi-step orchestration, tool chaining, integration with MCP Tools | 8082 | `services/response-api` | [api/response-api/README.md](api/response-api/README.md) |
| **Media API** | Binary ingestion, jan_* IDs, S3 storage and resolution | 8285 | `services/media-api` | [api/media-api/README.md](api/media-api/README.md) |
| **MCP Tools** | Model Context Protocol tools (web search, scraping, file search, python exec) | 8091 | `services/mcp-tools` | [api/mcp-tools/README.md](api/mcp-tools/README.md), [services/mcp-tools/README.md](../services/mcp-tools/README.md) |

## Configuration

All services use the **centralized configuration system** at `pkg/config/`:

- **Type-safe:** Go structs with compile-time validation
- **YAML defaults:** `config/defaults.yaml` for base configuration
- **Environment overrides:** Service-specific env vars (e.g., `LLM_API_HTTP_PORT`)
- **Kubernetes values:** Auto-generated from configuration structs
- **CLI tool:** `jan-cli config` for validation and inspection

See [Configuration Documentation](configuration/README.md) for details.

## Infrastructure Components
- **Kong Gateway (8000)**: exposes public APIs, enforces rate limits, validates Keycloak JWTs/API keys (custom plugin), and proxies `/llm/auth/guest-login` for guest tokens.
- **Keycloak (8085)**: handles OAuth2/OIDC flows; see `keycloak/`.
- **PostgreSQL**: `api-db` (LLM/Response/Media data) and `keycloak-db` (Keycloak state).
- **vLLM (8101)**: inference backend reachable from llm-api.
- **Observability stack**: Prometheus (9090), Grafana (3001), Jaeger (16686), OpenTelemetry Collector.
- **MCP support services**: SearXNG (search), Vector Store (file search), SandboxFusion (python execution).

## Creating a New Service

### Quick Start

```bash
# Generate from template
scripts/new-service-from-template.ps1 -Name my-service
```

### Configuration Setup

New services should use the centralized configuration system:

1. **Define service config in `pkg/config/types.go`:**
   ```go
   type ServiceConfig struct {
       HTTP     HTTPConfig     `yaml:"http"`
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
   import "jan-server/pkg/config"
   
   cfg, _ := config.Load()
   serviceCfg, _ := cfg.GetServiceConfig("my-service")
   ```

4. **Update deployment configs:**
   - Add service to `docker/services-api.yml`
   - Generate K8s values: `jan-cli config k8s-values --env production`

See [Configuration System](configuration/README.md) and [Service Template](../services/template-api/NEW_SERVICE_GUIDE.md) for complete guide.

### Documentation Requirements

1. Update `docs/services.md` (this file) with new service row
2. Create `docs/api/<service>/README.md` with API reference
3. Add service to `docs/INDEX.md` navigation
4. Update `k8s/jan-server/values.yaml` if deploying to Kubernetes

## Service Interactions
- **LLM API -> Media API**: LLM API resolves `jan_*` IDs before sending payloads to vLLM or upstream providers (`MEDIA_RESOLVE_URL` env var).
- **Response API -> LLM API**: Response API delegates final language generation to LLM API (`LLM_API_URL`).
- **Response API -> MCP Tools**: orchestrated tool calls are issued via JSON-RPC (`MCP_TOOLS_URL`).
- **MCP Tools -> Infrastructure**: uses SearXNG, Vector Store, and SandboxFusion to execute user requests.

Keep this document updated whenever a service is added, renamed, or retires.

