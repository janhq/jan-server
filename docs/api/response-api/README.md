# Response API Documentation

The Response API handles multi-step tool orchestration for complex workflows.

## Quick Start

### Base URL
- **Local**: http://localhost:8082
- **Via Gateway**: http://localhost:8000/api/responses
- **Docker**: http://response-api:8082

## Key Features

- **Multi-Step Tool Orchestration** - Chain tools together (max depth: 8)
- **Tool Timeout Management** - Per-tool timeouts (default: 45s)
- **LLM Integration** - Delegates to LLM API for language generation
- **MCP Tools** - Full integration with MCP tools for tool discovery
- **PostgreSQL Persistence** - Stores all executions and results
- **OpenAI Responses Contract** - Compatible with OpenAI responses format

## Service Ports & Configuration

| Component | Port | Environment Variable |
|-----------|------|---------------------|
| **HTTP Server** | 8082 | `HTTP_PORT` |
| **Database** | 5432 | `RESPONSE_DATABASE_URL` |
| **LLM API** | 8080 | `LLM_API_URL` |
| **MCP Tools** | 8091 | `MCP_TOOLS_URL` |

### Required Environment Variables

```bash
HTTP_PORT=8082                                                # HTTP listen port
RESPONSE_DATABASE_URL=postgres://response_api:password@api-db:5432/response_api?sslmode=disable
LLM_API_URL=http://llm-api:8080                             # LLM API base URL
MCP_TOOLS_URL=http://mcp-tools:8091                         # MCP Tools URL
MAX_TOOL_EXECUTION_DEPTH=8                                   # Max tool chain depth
TOOL_EXECUTION_TIMEOUT=45s                                   # Per-tool timeout
```

### Optional Configuration

```bash
LOG_LEVEL=info                                              # debug, info, warn, error
ENABLE_TRACING=false                                        # Enable OpenTelemetry
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317          # Jaeger endpoint
AUTH_ENABLED=false                                          # Enable JWT validation
AUTH_ISSUER=http://localhost:8090/realms/jan               # Token issuer
AUTH_AUDIENCE=jan-client                                    # JWT audience
AUTH_JWKS_URL=http://keycloak:8085/realms/jan/protocol/openid-connect/certs
```

## Main Endpoints

### Create Response (Multi-Step Orchestration)

**POST** `/v1/responses`

Create a new response with automatic tool orchestration.

```bash
curl -X POST http://localhost:8082/v1/responses \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "input": "Search for the latest AI news and summarize the top 3 results"
  }'
```

**Request Parameters:**
- `model` (required) - Model to use for generation
- `input` (required) - User input/prompt

**Response:**
```json
{
  "id": "resp_01hqr8v9k2x3f4g5h6j7k8m9n0",
  "model": "gpt-4o-mini",
  "input": "Search for the latest AI news and summarize the top 3 results",
  "output": "Here are the latest AI news items...",
  "tool_executions": [
    {
      "id": "toolexec_123",
      "tool": "google_search",
      "input": {"q": "latest AI news", "num": 3},
      "output": "...",
      "duration_ms": 250
    }
  ],
  "execution_metadata": {
    "max_depth": 8,
    "actual_depth": 1,
    "total_duration_ms": 2500,
    "status": "completed"
  },
  "created_at": "2025-11-10T10:30:00Z",
  "updated_at": "2025-11-10T10:30:02.500Z"
}
```

### Get Response

**GET** `/v1/responses/{id}`

Retrieve a specific response.

```bash
curl http://localhost:8082/v1/responses/resp_01hqr8v9k2x3f4g5h6j7k8m9n0
```

### List Responses

**GET** `/v1/responses`

List all responses with pagination.

```bash
curl http://localhost:8082/v1/responses?limit=10&offset=0
```

### Health Check

**GET** `/healthz`

```bash
curl http://localhost:8082/healthz
```

## Tool Execution Flow

### 1. Request Processing
- Validate input parameters
- Check tool availability via MCP Tools

### 2. Tool Discovery
- Query MCP Tools for available tools
- Build tool call graph

### 3. Iterative Execution
- Execute tools in sequence/parallel as needed
- Apply depth limit (max 8)
- Apply timeout per tool (45s)

### 4. LLM Delegation
- Pass tool results to LLM API
- Generate final response using context

### 5. Result Storage
- Store execution trace in PostgreSQL
- Record tool outputs and timing
- Return complete execution metadata

## Tool Execution Parameters

### Max Tool Execution Depth
Limits how deep tool calls can chain:
- **Value**: 1-15 (default: 8)
- **Meaning**: Maximum recursive depth of tool calls
- **Example**: search → extract → summarize = depth 2

### Tool Execution Timeout
Per-tool call timeout:
- **Value**: Duration string (default: 45s)
- **Example**: "30s", "1m", "500ms"
- **Behavior**: Cancels tool if it exceeds timeout

## Error Handling

| Status | Error | Cause |
|--------|-------|-------|
| 400 | Invalid request | Missing/malformed parameters |
| 404 | Response not found | Invalid response ID |
| 408 | Tool execution timeout | Tool exceeded timeout |
| 500 | Execution error | Tool or LLM error |

Example error:
```json
{
  "error": {
    "message": "Tool execution exceeded maximum depth",
    "type": "execution_error",
    "code": "max_depth_exceeded"
  }
}
```

## Related Services

- **LLM API** (Port 8080) - Generates final response
- **MCP Tools** (Port 8091) - Tool execution and discovery
- **Kong Gateway** (Port 8000) - API routing
- **PostgreSQL** - Execution storage

## Configuration Examples

### Quick Response (Single Tool)
```bash
MAX_TOOL_EXECUTION_DEPTH=1          # Single tool call only
TOOL_EXECUTION_TIMEOUT=15s          # Short timeout
```

### Complex Workflows (Deep Chains)
```bash
MAX_TOOL_EXECUTION_DEPTH=8          # Allow up to 8 levels
TOOL_EXECUTION_TIMEOUT=120s         # Long timeout for complex work
```

## See Also

- [MCP Tools API](../mcp-tools/)
- [LLM API](../llm-api/)
- [Architecture Overview](../../architecture/)
- [Development Guide](../../guides/development.md)
