# MCP Testing Guide

This guide describes how to exercise the MCP tooling end-to-end so we can validate new automations before shipping them.

## 1. Prerequisites

- `make mcp-with-tools` (or `make mcp-full` + `make up-mcp-tools`) to start SearXNG, SandboxFusion, vector-store, and the MCP bridge.
- `newman` CLI installed (`npm install -g newman`).
- Docker services running:
  - SearXNG: `http://localhost:8086`
  - Vector Store: `http://localhost:3015`
  - SandboxFusion: `http://localhost:3010`
  - MCP Tools Bridge: `http://localhost:8091`

## 2. Automated Tests (Newman)

Run the full suite using the Makefile target:

```bash
make newman-mcp
```

Or run directly with newman:

```bash
newman run tests/automation/mcp-postman-scripts.json \
  --env-var mcp_tools_url=http://localhost:8091 \
  --env-var llm_api_url=http://localhost:8000 \
  --env-var searxng_url=http://localhost:8086
```

Key requests inside the collection:

- **Guest Auth - Request Guest Token** – Obtains a guest access token for authenticated MCP requests.
- **Guest Auth - MCP Search Domain Filter** – Confirms `domain_allow_list` enforces example.com citations.
- **Guest Auth - MCP Search Offline Mode** – Forces DuckDuckGo fallback and asserts `cache_status=offline_mode` and `live=false`.
- **MCP Tools - List MCP Tools** – Verifies all available MCP tools are returned (google_search, scrape, file_search_index, file_search_query, python_exec).
- **MCP Tools - Serper Search via MCP** – Verifies structured `{ source_url, snippet, fetched_at, cache_status }` output with results and citations.
- **MCP Tools - Serper Scrape via MCP** – Validates web scraping returns `text`, `text_preview`, `cache_status`, and metadata.
- **MCP Tools - File Search Index** – Calls `file_search_index` to upsert a sample document and verifies indexed status.
- **MCP Tools - File Search Query** – Calls `file_search_query` to ensure the indexed document is returned with a citation-ready preview.
- **MCP Tools - SandboxFusion Python Exec** – Executes Python code via `python_exec` tool and verifies stdout contains expected output. **Note:** Requires `language: "python"` parameter.
- **SearXNG - SearXNG HTML Search** – Direct HTML search via SearXNG to validate the search engine is operational.
- **SearXNG - SearXNG Text Scrape** – Direct text search via SearXNG.

### Test Results

All 28 assertions across 11 requests should pass:
- ✅ Guest authentication and token generation
- ✅ MCP search with domain filtering
- ✅ MCP search in offline mode
- ✅ List all MCP tools
- ✅ Google search via MCP (with fallback support)
- ✅ Web scraping via MCP
- ✅ File search indexing
- ✅ File search querying  
- ✅ SandboxFusion Python code execution
- ✅ Direct SearXNG HTML search
- ✅ Direct SearXNG text scraping

## 3. Manual CURL Checks

### 3.1 SandboxFusion Service

Test Python code execution directly:

```bash
# PowerShell
Invoke-RestMethod -Method Post -Uri "http://localhost:3010/run_code" `
  -Headers @{"Content-Type"="application/json"} `
  -Body '{"code":"print(\"hello from sandbox\")","language":"python"}' | ConvertTo-Json

# Expected response structure:
# {
#   "status": "Success",
#   "run_result": {
#     "status": "Finished",
#     "stdout": "hello from sandbox\n",
#     "stderr": "",
#     "return_code": 0
#   }
# }
```

### 3.2 Vector Store Service

```bash
# Index a document directly
curl -s http://localhost:3015/documents -X POST -H \"Content-Type: application/json\" -d '{
  \"document_id\": \"curl-doc-1\",
  \"text\": \"Curl-based test document for MCP vector store.\",
  \"metadata\": {\"owner\": \"qa\"}
}'

# Query it back
curl -s http://localhost:3015/query -X POST -H \"Content-Type: application/json\" -d '{
  \"text\": \"test document\",
  \"top_k\": 3
}'
```

### 3.3 MCP Bridge

```bash
# List tools
curl -s http://localhost:8091/v1/mcp -X POST -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0", "method": "tools/list", "id": 1
}'

# Execute Python code via MCP (requires language parameter)
curl -s http://localhost:8091/v1/mcp -X POST -H "Content-Type: application/json" -d '{
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
}'

# Index via MCP
curl -s http://localhost:8091/v1/mcp -X POST -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "file_search_index",
    "arguments": {
      "document_id": "curl-doc-2",
      "text": "CLI indexed text"
    }
  }
}'
```

## 4. Expected Outcomes

- Web search responses now always contain structured JSON with citations and cache hints.
- Scrape responses emit `text`, `text_preview`, and `cache_status`.
- File search tools succeed (HTTP 200) and vector store logs show documents being indexed/querying.
- SandboxFusion executes Python code and returns stdout/stderr properly mapped from the nested API response structure.

## 5. Troubleshooting

If any tests fail, check:

1. **SandboxFusion Issues**:
   - Ensure `language: "python"` parameter is included in `python_exec` calls
   - Check SandboxFusion logs: `docker logs jan-server-sandbox-fusion-1`
   - Verify the service is accessible at `http://localhost:3010`

2. **Vector Store Issues**:
   - Verify `VECTOR_STORE_URL` env var inside `mcp-tools` (should be `http://vector-store-mcp:3015`)
   - Check logs: `docker compose logs vector-store-mcp`
   - Ensure the service is accessible at `http://localhost:3015`

3. **MCP Bridge Issues**:
   - Check MCP bridge logs for HTTP errors: `docker logs jan-server-mcp-tools-1`
   - Verify all services are in the same Docker network (`jan-server_mcp-network`)
   - Ensure the bridge is accessible at `http://localhost:8091`

4. **SearXNG Issues**:
   - Check if Redis is running: `docker logs jan-server-redis-searxng-1`
   - Check SearXNG logs: `docker compose logs searxng`
   - Verify the service is accessible at `http://localhost:8086`

## 6. Recent Fixes

### SandboxFusion Integration (Nov 2025)

Fixed the SandboxFusion client to properly handle the API response structure:

- **Issue**: The `python_exec` tool was returning empty stdout/stderr
- **Root Cause**: SandboxFusion API returns a nested structure (`run_result.stdout`) but the client expected flat structure (`stdout`)
- **Solution**: 
  - Added `RunResult` and `SandboxFusionAPIResponse` structs to properly parse the nested API response
  - Updated `RunCode()` method to map `run_result.stdout/stderr` to the expected flat structure
  - Added conversion of execution time from seconds to milliseconds
  - Added proper handling of files map to artifacts array
- **Required Parameter**: The `language` field must be provided (e.g., `"language": "python"`) even though marked as optional in the tool schema
