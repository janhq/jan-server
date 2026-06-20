# MCP Tools API Documentation

The MCP Tools API provides AI tools for web search, web scraping, and code execution.

## Quick Start

Examples: [API examples index](../examples/README.md) includes discovery and call samples.

### URLs

- **Direct access**: http://localhost:8091
- **Through gateway**: http://localhost:8000/v1/mcp (Kong also exposes `/mcp/*` and forwards to `/v1/...`)
- **Inside Docker**: http://mcp-tools:8091

### Authentication

All endpoints require authentication through the Kong gateway.

**For complete authentication documentation, see [Authentication Guide](../README.md#authentication)**

**Quick example:**

```bash
# Get guest token
TOKEN=$(curl -s -X POST http://localhost:8000/llm/auth/guest-login | jq -r '.access_token')

# Use in requests
curl -H "Authorization: Bearer $TOKEN" \
 -H "Content-Type: application/json" \
 -d '{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}' \
 http://localhost:8000/v1/mcp
```

## Available Tools

- **google_search** - Web search with optional filters and location hints. Backed by a provider fallback chain: **Serper -> Exa -> Tavily -> SearXNG** (the first enabled provider with valid credentials handles the query).
- **scrape** - Fetch and parse a web page, optionally returning Markdown
- **file_search_index** / **file_search_query** - Lightweight vector store to index custom text and run similarity queries
- **python_exec** - Execute trusted code through SandboxFusion (optional approval flag)
- **External providers** - Additional tools declared in [`services/mcp-tools/mcp-providers.md`](../../services/mcp-tools/mcp-providers.md) are loaded automatically

## How It Works

All tools use the JSON-RPC 2.0 protocol over the single `POST /v1/mcp` endpoint. You send a
request naming the tool and its arguments, and get back the result. See
[JSON-RPC 2.0 Protocol](#json-rpc-20-protocol) below for the request/response format.

## Service Ports & Configuration

| Component            | Port | Key Environment Variables                                                          |
| -------------------- | ---- | ---------------------------------------------------------------------------------- |
| **HTTP Server**      | 8091 | `MCP_TOOLS_HTTP_PORT`                                                               |
| **Search Providers** | 443  | `SERPER_API_KEY`, `EXA_API_KEY`, `TAVILY_API_KEY`, `SEARXNG_URL` (+ each `*_ENABLED`) |
| **Vector Store**     | 3015 | `VECTOR_STORE_URL`                                                                  |
| **SandboxFusion**    | 8080 | `SANDBOXFUSION_URL`, `MCP_SANDBOX_REQUIRE_APPROVAL`                                 |

### Required Environment Variables

```bash
MCP_TOOLS_HTTP_PORT=8091

# Search provider fallback chain: Serper -> Exa -> Tavily -> SearXNG.
# Each provider needs *_ENABLED=true AND valid credentials to participate.
SERPER_API_KEY=your_serper_api_key   SERPER_ENABLED=true
EXA_API_KEY=your_exa_api_key         EXA_ENABLED=true
TAVILY_API_KEY=your_tavily_api_key   TAVILY_ENABLED=true
SEARXNG_URL=http://searxng:8080      SEARXNG_ENABLED=true

VECTOR_STORE_URL=http://vector-store-mcp:3015
SANDBOXFUSION_URL=http://sandbox-fusion:8080
OTEL_ENABLED=false

# Auth (optional)
AUTH_ENABLED=true
AUTH_ISSUER=http://localhost:8085/realms/jan
ACCOUNT=account
AUTH_JWKS_URL=http://keycloak:8085/realms/jan/protocol/openid-connect/certs
```

### Optional Configuration

```bash
MCP_TOOLS_LOG_LEVEL=info
MCP_TOOLS_LOG_FORMAT=json          # json | console
SANDBOXFUSION_TIMEOUT=30s
SERPER_DOMAIN_FILTER=example.com,another.com
SERPER_LOCATION_HINT=California, United States
SERPER_OFFLINE_MODE=false
MCP_SANDBOX_REQUIRE_APPROVAL=true  # force clients to set `approved: true`
```

## JSON-RPC 2.0 Protocol

All tool calls use JSON-RPC 2.0 format.

### Request Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "tool_name",
    "arguments": {
      "arg1": "value1",
      "arg2": "value2"
    }
  }
}
```

### Response Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": "Tool output",
    "is_error": false
  }
}
```

### Error Response

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32603,
    "message": "Internal error",
    "data": "Tool execution failed"
  }
}
```

## MCP Endpoint (`POST /v1/mcp`)

All JSON-RPC requests (initialize, tools/list, tools/call, prompts/list, etc.) go through a single streaming endpoint. When calling through Kong, use `http://localhost:8000/v1/mcp`; direct calls go to `http://localhost:8091/v1/mcp`.

```bash
curl -N http://localhost:8000/v1/mcp \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "jsonrpc": "2.0",
 "id": 1,
 "method": "tools/list"
 }'
```

The service runs the MCP streamable HTTP handler in **stateless, JSON-response mode**, so a
normal `tools/call` returns a single JSON body (you do not need `-N`). The handler still accepts
`Accept: application/json, text/event-stream`, which is set automatically if you omit it.

**Response:**

```json
{
  "tools": [
    {
      "name": "google_search",
      "description": "Search Google for query results",
      "inputSchema": {
        "type": "object",
        "properties": {
          "q": { "type": "string", "description": "Search query" },
          "num": {
            "type": "integer",
            "description": "Number of results",
            "default": 10
          }
        },
        "required": ["q"]
      }
    },
    {
      "name": "scrape",
      "description": "Extract content from a URL",
      "inputSchema": {
        "type": "object",
        "properties": {
          "url": { "type": "string", "description": "URL to scrape" }
        },
        "required": ["url"]
      }
    },
    {
      "name": "python_exec",
      "description": "Execute code in a sandboxed environment",
      "inputSchema": {
        "type": "object",
        "properties": {
          "code": { "type": "string", "description": "Code to execute" },
          "language": {
            "type": "string",
            "enum": ["python", "javascript"],
            "default": "python"
          }
        },
        "required": ["code"]
      }
    }
  ]
}
```

### Health Check

**GET** `/healthz`

```bash
# Via gateway
curl http://localhost:8000/mcp/healthz

# Direct
curl http://localhost:8091/healthz
```

## Integration with Response API

The Response API uses MCP Tools for multi-step orchestration:

```bash
# Response API automatically calls MCP tools
# Kong route /responses uses strip_path, so /responses/v1/responses -> service /v1/responses
curl -X POST http://localhost:8000/responses/v1/responses \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "model": "jan-v1-4b",
 "input": "Search for Python async programming and summarize top 3 results",
 "stream": true
 }'

# Response API orchestrates:
# 1. Call google_search
# 2. Optionally call scrape/file_search_query
# 3. Stream deltas as the LLM produces the final response
```

## Tool Chaining (via Response API)

The Response API enables tool chaining:

```
google_search
 v
scrape (on each result)
 v
python_exec (if needed for analysis)
 v
LLM API (final generation)
```

**Max tool depth**: capped by `RESPONSE_MAX_TOOL_DEPTH` (default 50)
**Timeout per tool**: `TOOL_EXECUTION_TIMEOUT` (default 300s)

## Error Codes

| Code   | Message          | Meaning                |
| ------ | ---------------- | ---------------------- |
| -32700 | Parse error      | Invalid JSON           |
| -32600 | Invalid Request  | Missing method/params  |
| -32601 | Method not found | Unknown tool           |
| -32602 | Invalid params   | Invalid parameters     |
| -32603 | Internal error   | Tool execution failed  |
| -32000 | Timeout          | Tool execution timeout |

## Related Services

- **Response API** (Port 8082) - Tool orchestration
- **LLM API** (Port 8080) - Final generation
- **Kong Gateway** (Port 8000) - API routing
- **SandboxFusion** - Code execution sandbox
- **Search providers** - Serper, Exa, Tavily, SearXNG (fallback chain)
- **Provider Configuration**: [services/mcp-tools/mcp-providers.md](../../services/mcp-tools/mcp-providers.md)

## Examples

### Example: List Available Tools

```bash
curl -s http://localhost:8000/v1/mcp \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "jsonrpc": "2.0",
 "id": 42,
 "method": "tools/list"
 }' | jq
```

### Example: `google_search`

```bash
curl -s http://localhost:8000/v1/mcp \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "jsonrpc": "2.0",
 "id": 2,
 "method": "tools/call",
 "params": {
 "name": "google_search",
 "arguments": {"q": "latest AI news", "num": 5}
 }
 }'
```

### Example: `scrape`

```bash
curl -s http://localhost:8000/v1/mcp \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "jsonrpc": "2.0",
 "id": 3,
 "method": "tools/call",
 "params": {
 "name": "scrape",
 "arguments": {"url": "https://docs.python.org/3/", "includeMarkdown": true}
 }
 }'
```

### Example: Vector Store

```bash
# Index a note
curl -s http://localhost:8000/v1/mcp \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "jsonrpc": "2.0",
 "id": 4,
 "method": "tools/call",
 "params": {
 "name": "file_search_index",
 "arguments": {
   "document_id": "notes-1",
   "text": "Menlo Platform docs live in jan-server/docs/*",
   "tags": ["docs","menlo"]
 }
 }
 }'

# Query it later
curl -s http://localhost:8000/v1/mcp \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "jsonrpc": "2.0",
 "id": 5,
 "method": "tools/call",
 "params": {
   "name": "file_search_query",
   "arguments": {
     "query": "Where do the Menlo Platform docs live?",
     "top_k": 3
   }
 }
 }'
```

### Example: Sandbox (`python_exec`)

```bash
curl -s http://localhost:8000/v1/mcp \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "jsonrpc": "2.0",
 "id": 6,
 "method": "tools/call",
 "params": {
 "name": "python_exec",
 "arguments": {"code": "import math; print(math.pi)"}
 }
 }'
```
