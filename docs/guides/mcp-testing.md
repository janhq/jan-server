# MCP Testing Guide

Validate the MCP (Model Context Protocol) toolchain end to end. Every command below maps directly to current Makefile targets and compose services, so you can run it without editing scripts.

## 1. Prerequisites

- `make up-full` (or `make up-mcp` + `make up-api`) so Kong, MCP Tools, and vector-store are running
- `SERPER_API_KEY` (if `SERPER_ENABLED=true`) or alternative provider keys (`EXA_API_KEY`, `TAVILY_API_KEY`) set in `.env`
- Provider flags set as needed: `SERPER_ENABLED`, `EXA_ENABLED`, `TAVILY_ENABLED`, `SEARXNG_ENABLED`
- Services reachable on:
  - Kong Gateway: http://localhost:8000
  - MCP Tools: http://localhost:8091 (direct) or http://localhost:8000/mcp (via Kong)
  - Vector Store: http://localhost:3015
  - SandboxFusion (optional): http://localhost:3010

Check health quickly:

```bash
make health-check       # full stack health summary
curl http://localhost:8091/healthz
curl http://localhost:3015/healthz || true   # returns 404 because the vector store uses custom routes
```

## 2. Automated Suite (jan-cli api-test)

Run everything through the Makefile target:

```bash
make test-mcp
```

The target executes:

```bash
jan-cli api-test run \
  tests/e2e/automation/collections/mcp-runtime.postman.json \
  tests/e2e/automation/collections/mcp-admin.postman.json \
  --env-var "kong_url=http://localhost:8000" \
  --env-var "mcp_tools_url=http://localhost:8000/mcp" \
  --verbose --reporters cli
```

Expectations:

- Guest token requests succeed (`/llm/auth/guest-login`)
- MCP search variants (domain filter, offline) return structured JSON or explicit errors when all providers fail
- Tool list includes `google_search`, `scrape`, `file_search_index`, `file_search_query`, `python_exec`
- File index/query flows return 200 and include the previously indexed document
- SandboxFusion executions return stdout/stderr

## 3. Manual Checks

### 3.1 Kong -> MCP Tools

```bash
# list tools through Kong (authenticated by the gateway)
curl -s http://localhost:8000/mcp -X POST -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list"
}' | jq .

# call python_exec via Kong
curl -s http://localhost:8000/mcp -X POST -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "python_exec",
    "arguments": {
      "code": "print(\"Hello from MCP\")",
      "language": "python",
      "approved": true
    }
  }
}' | jq .
```

### 3.2 Direct Service Endpoints

```bash
# MCP Tools (direct port)
curl -s http://localhost:8091/v1/mcp -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {"name": "file_search_index", "arguments": {"document_id": "cli-doc", "text": "CLI test"}}
}' | jq .

# Vector store
curl -s http://localhost:3015/documents -X POST -H "Content-Type: application/json" -d '{
  "document_id": "curl-doc",
  "text": "Curl-based MCP test",
  "metadata": {"owner": "qa"}
}'

curl -s http://localhost:3015/query -X POST -H "Content-Type: application/json" -d '{
  "text": "MCP test",
  "top_k": 3
}' | jq .

# SandboxFusion (optional)
curl -s http://localhost:3010/run_code -H "Content-Type: application/json" -d '{
  "code": "print(\"sandbox\")",
  "language": "python"
}' | jq .
```

## 4. Logs and Troubleshooting

| Component     | Logs                                             | Notes                                                       |
| ------------- | ------------------------------------------------ | ----------------------------------------------------------- |
| Kong          | `make logs` or `docker compose logs kong`        | Confirms `/mcp` route, auth headers, upstream failures      |
| MCP Tools     | `make logs-mcp`                                  | Watch tool dispatch, vector store responses, sandbox output |
| Vector Store  | `docker compose logs vector-store`               | Service name is `vector-store` in `infra/docker/services-mcp.yml` |
| SandboxFusion | `docker compose logs sandboxfusion` (if enabled) | Verify HTTP 200s and stdout capturing                       |

Common fixes:

- **401/403**: ensure guest token exists or provide API key headers when hitting Kong
- **Timeouts to vector store**: confirm service is part of the `mcp` profile (`COMPOSE_PROFILES` includes `mcp`)
- **Sandbox errors**: include the required `language` parameter; see `services/mcp-tools/internal/sandboxfusion`

## 5. Summary Checklist

- [ ] `make up-full` (or `make up-mcp` + `make up-api`) running
- [ ] `make test-mcp` passes locally
- [ ] Manual curl checks through Kong and direct service succeed
- [ ] Logs show healthy MCP tool executions

Document these results in your PR or QA notes so MCP coverage stays verifiable.
