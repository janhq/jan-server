# MCP Tools API Documentation

The MCP Tools API provides Model Context Protocol tools for web search, scraping, and code execution.

## Quick Start

### Base URL
- **Local**: http://localhost:8091
- **Via Gateway**: http://localhost:8000/api/mcp
- **Docker**: http://mcp-tools:8091

## Key Features

- **JSON-RPC 2.0 Protocol** - Standard protocol for tool interaction
- **Web Search** - Google search via Serper API
- **Web Scraping** - Extract content from URLs
- **Code Execution** - Execute code in sandboxed environment (SandboxFusion)
- **Tool Discovery** - List available tools and parameters
- **Error Handling** - Comprehensive error responses

## Service Ports & Configuration

| Component | Port | Environment Variable |
|-----------|------|---------------------|
| **HTTP Server** | 8091 | `HTTP_PORT` |
| **Serper API** | 443 | `SERPER_API_KEY` |
| **SandboxFusion** | 8080 | `SANDBOX_URL` |

### Required Environment Variables

```bash
HTTP_PORT=8091                                      # HTTP listen port
SERPER_API_KEY=your_serper_api_key                 # Serper API key
OTEL_ENABLED=false                                 # OpenTelemetry
```

### Optional Configuration

```bash
LOG_LEVEL=info                                     # debug, info, warn, error
SANDBOX_URL=http://localhost:3010                 # SandboxFusion URL
SANDBOX_TIMEOUT=30s                               # Execution timeout
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

## Main Endpoints

### Call Tool

**POST** `/v1/mcp`

Execute a tool using JSON-RPC 2.0.

```bash
curl -X POST http://localhost:8091/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "google_search",
      "arguments": {
        "q": "latest AI news",
        "num": 5
      }
    }
  }'
```

### List Tools

**GET** `/v1/mcp/tools`

Get all available tools and their signatures.

```bash
curl http://localhost:8091/v1/mcp/tools
```

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
          "q": {"type": "string", "description": "Search query"},
          "num": {"type": "integer", "description": "Number of results", "default": 10}
        },
        "required": ["q"]
      }
    },
    {
      "name": "web_scraper",
      "description": "Extract content from a URL",
      "inputSchema": {
        "type": "object",
        "properties": {
          "url": {"type": "string", "description": "URL to scrape"}
        },
        "required": ["url"]
      }
    },
    {
      "name": "code_executor",
      "description": "Execute code in a sandboxed environment",
      "inputSchema": {
        "type": "object",
        "properties": {
          "code": {"type": "string", "description": "Code to execute"},
          "language": {"type": "string", "enum": ["python", "javascript"], "default": "python"}
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
curl http://localhost:8091/healthz
```

## Available Tools

### 1. Google Search

**Tool Name**: `google_search`

Search the web using Serper API.

```bash
curl -X POST http://localhost:8091/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "google_search",
      "arguments": {
        "q": "Python async programming",
        "num": 5
      }
    }
  }'
```

**Parameters:**
- `q` (required) - Search query
- `num` (optional) - Number of results (default: 10, max: 20)

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": "[{\"title\":\"...\",\"link\":\"...\",\"snippet\":\"...\"}, ...]",
    "is_error": false
  }
}
```

### 2. Web Scraper

**Tool Name**: `web_scraper`

Extract text content from a URL.

```bash
curl -X POST http://localhost:8091/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "web_scraper",
      "arguments": {
        "url": "https://example.com/article"
      }
    }
  }'
```

**Parameters:**
- `url` (required) - URL to scrape
- `selector` (optional) - CSS selector for specific content
- `timeout` (optional) - Timeout in seconds (default: 10)

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": "Extracted page content...",
    "is_error": false
  }
}
```

### 3. Code Executor

**Tool Name**: `code_executor`

Execute code in a sandboxed environment (SandboxFusion).

```bash
curl -X POST http://localhost:8091/v1/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "code_executor",
      "arguments": {
        "code": "import math\nprint(math.sqrt(16))",
        "language": "python"
      }
    }
  }'
```

**Parameters:**
- `code` (required) - Code to execute
- `language` (optional) - "python" or "javascript" (default: "python")
- `timeout` (optional) - Timeout in seconds (default: 30)

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": "4.0",
    "is_error": false
  }
}
```

## Integration with Response API

The Response API uses MCP Tools for multi-step orchestration:

```bash
# Response API automatically calls MCP tools
curl -X POST http://localhost:8082/v1/responses \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "input": "Search for Python async programming and summarize top 3 results"
  }'

# Response API orchestrates:
# 1. Call google_search tool
# 2. Pass results to LLM API for summarization
# 3. Return final response
```

## Tool Chaining (via Response API)

The Response API enables tool chaining:

```
google_search 
    ↓
web_scraper (on each result)
    ↓
code_executor (if needed for analysis)
    ↓
LLM API (final generation)
```

**Max Depth**: 8 tool calls
**Timeout per Tool**: 45 seconds

## Error Codes

| Code | Message | Meaning |
|------|---------|---------|
| -32700 | Parse error | Invalid JSON |
| -32600 | Invalid Request | Missing method/params |
| -32601 | Method not found | Unknown tool |
| -32602 | Invalid params | Invalid parameters |
| -32603 | Internal error | Tool execution failed |
| -32000 | Timeout | Tool execution timeout |

## Usage Examples

### Example 1: Search and Get First Result

```bash
# Search
curl -X POST http://localhost:8091/v1/mcp \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"google_search","arguments":{"q":"Python documentation","num":1}}}'

# Then scrape top result
curl -X POST http://localhost:8091/v1/mcp \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"web_scraper","arguments":{"url":"https://docs.python.org"}}}'
```

### Example 2: Execute Python Code

```bash
curl -X POST http://localhost:8091/v1/mcp \
  -d '{
    "jsonrpc":"2.0",
    "id":1,
    "method":"tools/call",
    "params":{
      "name":"code_executor",
      "arguments":{
        "code":"data = [1,2,3,4,5]\nprint(f\"Sum: {sum(data)}, Avg: {sum(data)/len(data)}\")",
        "language":"python"
      }
    }
  }'
```

### Example 3: Via Response API (Automated Orchestration)

```bash
curl -X POST http://localhost:8082/v1/responses \
  -d '{
    "model":"gpt-4o-mini",
    "input":"Find current Bitcoin price and explain its recent trend"
  }'

# Automatically orchestrates:
# 1. google_search("Bitcoin price today")
# 2. web_scraper(on top result)
# 3. LLM generation with extracted data
```

## Related Services

- **Response API** (Port 8082) - Tool orchestration
- **LLM API** (Port 8080) - Final generation
- **Kong Gateway** (Port 8000) - API routing
- **SandboxFusion** - Code execution sandbox
- **Serper API** - Web search provider

## See Also

- [Response API Documentation](../response-api/)
- [LLM API Documentation](../llm-api/)
- [Architecture Overview](../../architecture/)
- [MCP Integration Guide](./integration.md)
- [MCP Providers](./providers.md)
