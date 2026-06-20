# API Reference

Complete API documentation for Jan Server services.

## Available APIs

### 1. LLM API (Port 8080)

OpenAI-compatible API for chat completions, conversations, and models.

**What it does:**

- Generate AI responses to user messages
- Manage conversations and chat history
- Organize conversations in projects
- List available AI models
- Handle user authentication
- Support images via jan\_\* IDs
- Generate images from text prompts

**Documentation:**

- **[Complete Documentation](llm-api/)** - Full API reference, endpoints, examples
- **[Authentication](llm-api/#authentication)** - Auth methods, API keys, and token management
- **[Chat Completions](llm-api/#chat-completions)** - Main completion endpoint
- **[Conversations](llm-api/#conversations)** - Conversation CRUD operations
- **[Projects](llm-api/#projects)** - Project management for organizing conversations
- **[Admin Endpoints](llm-api/#admin-endpoints)** - Provider and model catalog management
- **[With Media](llm-api/#with-media-visual-input)** - Media references using `jan_*` IDs
- **[Examples](llm-api/comprehensive-examples.md)** - cURL, Python, and JavaScript snippets

### 2. Response API (Port 8082)

Executes tools and generates AI responses for complex tasks.

**What it does:**

- Run multiple tools in sequence (tool depth capped by `RESPONSE_MAX_TOOL_DEPTH`, default 50)
- Chain tool outputs together
- Generate final answers using LLM
- Track execution time and status

**Documentation:**

- **[Complete Documentation](response-api/)** - Full API reference, configuration, examples
- **[Create Response](response-api/#create-response-multi-step-orchestration)** - Main orchestration endpoint
- **[Tool Execution Flow](response-api/#tool-execution-flow)** - How tools are executed
- **[Configuration](response-api/#tool-execution-parameters)** - Depth and timeout settings

### 3. Media API (Port 8285)

Handles image uploads and storage.

**What it does:**

- Upload images from URLs or base64 data
- Store images in S3 cloud storage
- Generate jan\_\* IDs for images
- Create temporary download links
- Prevent duplicate uploads

**Documentation:**

- **[Complete Documentation](media-api/)** - Full API reference, storage flow, examples
- **[Upload Media](media-api/#endpoints)** - Ingest from remote URL / data URL, or multipart upload
- **[Jan ID System](media-api/#jan-id-system)** - Understanding `jan_*` identifiers

### 4. MCP Tools API (Port 8091)

Provides Model Context Protocol tools for search, scraping, lightweight vector search, and sandboxed execution.

**Available Tools:**

- **google_search** - Web search with a provider fallback chain (Serper -> Exa -> Tavily -> SearXNG), filters, and location hints
- **scrape** - Fetch and parse a web page (optional Markdown output)
- **file_search_index / file_search_query** - Index custom text into the bundled vector store and run similarity queries
- **python_exec** - Run trusted code via SandboxFusion, returning stdout/stderr/artifacts

All tools are invoked through a single JSON-RPC 2.0 endpoint, `POST /v1/mcp`, using the `tools/list` and `tools/call` methods.

**Documentation:**

- **[Complete Documentation](mcp-tools/)** - Full API reference, tool descriptions, examples
- **[JSON-RPC Protocol](mcp-tools/#json-rpc-20-protocol)** - Standard protocol format
- **[Available Tools](mcp-tools/#available-tools)** - Tool names and parameters
- **[Providers](mcp-tools/#providers)** - MCP provider configuration

## API Guides

- **[Decision Guides](decision-guides.md)** - When to use which API, choosing upload methods, memory configuration
- [Endpoint Matrix](endpoint-matrix.md) - Full endpoint inventory
- [Examples Index](examples/README.md) - cURL/SDK samples across services

Rate limits are enforced by the Kong gateway; see [integrations/kong/kong.yml](../../integrations/kong/kong.yml).

## Quick Reference

### Base URLs

| Environment | LLM API               | Response API             | Media API             | MCP Tools             | Gateway               |
| ----------- | --------------------- | ------------------------ | --------------------- | --------------------- | --------------------- |
| **Local**   | http://localhost:8080 | http://localhost:8082    | http://localhost:8285 | http://localhost:8091 | http://localhost:8000 |
| **Docker**  | http://llm-api:8080   | http://response-api:8082 | http://media-api:8285 | http://mcp-tools:8091 | http://kong:8000      |

**Recommended**: Point all public clients at the Kong gateway (port 8000) so authentication, rate limiting, and routing stay consistent. Direct service ports remain available for internal tests but still require JWT/API key headers.

### Authentication

All API endpoints require authentication. The Kong gateway (port 8000) validates your credentials and forwards requests to backend services.

#### Authentication Methods

**1. Bearer Token (Recommended for Development)**

Get a guest token from Keycloak and use it in the `Authorization` header:

```bash
# Request a guest token
curl -X POST http://localhost:8000/llm/auth/guest-login

# Response
{
 "access_token": "eyJhbGci...",
 "refresh_token": "eyJhbGci...",
 "expires_in": 300,
 "token_type": "Bearer"
}

# Use the token in requests
curl -H "Authorization: Bearer eyJhbGci..." \
 http://localhost:8000/v1/chat/completions
```

**2. API Key (For Production Clients)**

Use the `X-API-Key` header with your API key:

```bash
curl -H "X-API-Key: sk_your_api_key_here" \
 http://localhost:8000/v1/chat/completions
```

#### Token Management

**Refresh Tokens:**

```bash
curl -X POST http://localhost:8000/llm/auth/refresh-token \
 -H "Content-Type: application/json" \
 -d '{"refresh_token": "eyJhbGci..."}'
```

**Revoke Tokens:**

```bash
curl -X POST http://localhost:8000/llm/auth/revoke \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{"token": "eyJhbGci..."}'
```

#### Direct Service Access

When calling services directly (ports 8080/8082/8285/8091) instead of through Kong:

- You still need a valid Keycloak JWT
- Use the same `Authorization: Bearer <token>` header
- API key authentication is NOT available (Kong-only feature)

**Example direct call:**

```bash
# Still requires JWT token from Keycloak
curl -H "Authorization: Bearer <token>" \
 http://localhost:8080/v1/chat/completions
```

#### Authentication Flow

1. Client requests guest login or uses API key
2. Kong validates credentials (JWT signature + expiry, or API key lookup)
3. Kong forwards request to backend service with JWT in header
4. Backend service validates JWT signature and claims
5. Request is processed and response returned

> **Best Practice**: Always use the Kong gateway (port 8000) for client applications. Direct service ports are for internal communication and debugging only.

### Quick Examples

#### Chat Completion

```bash
curl -X POST http://localhost:8000/v1/chat/completions \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "model": "jan-v1-4b",
 "messages": [
 {"role": "user", "content": "Hello!"}
 ]
 }'
```

#### Google Search (MCP)

```bash
curl -X POST http://localhost:8000/v1/mcp \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "jsonrpc": "2.0",
 "method": "tools/call",
 "params": {
 "name": "google_search",
 "arguments": {"q": "AI news"}
 }
 }'
```

> Calling MCP Tools directly (e.g., `http://localhost:8091/v1/mcp`) is supported for internal testing, but the gateway-provided JWT/API key is still required when Kong proxies the request.

#### List Models

```bash
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/v1/models
```

## API Conventions

### Response Format

All successful responses return JSON:

```json
{
 "data": {...},
 "meta": {...}
}
```

### Error Format

All errors follow this structure:

```json
{
  "error": {
    "type": "invalid_request_error",
    "code": "invalid_parameter",
    "message": "Parameter 'model' is required",
    "param": "model",
    "request_id": "req_123xyz"
  }
}
```

### Error Types

| Type                    | Description                | HTTP Status |
| ----------------------- | -------------------------- | ----------- |
| `invalid_request_error` | Invalid request parameters | 400         |
| `auth_error`            | Authentication failed      | 401         |
| `permission_error`      | Insufficient permissions   | 403         |
| `not_found_error`       | Resource not found         | 404         |
| `rate_limit_error`      | Too many requests          | 429         |
| `internal_error`        | Server error               | 500         |

### Headers

**Request Headers**:

- `Authorization: Bearer <token>` - Required for authenticated endpoints
- `Content-Type: application/json` - For POST/PUT requests
- `Idempotency-Key: <uuid>` - Optional, for idempotent POST requests
- `X-Request-Id: <uuid>` - Optional, for request tracing

**Response Headers**:

- `X-Request-Id` - Request identifier for tracing
- `X-Auth-Method` - Authentication method used (jwt or api_key)
- `Content-Type: application/json` - JSON response
- `Content-Type: text/event-stream` - SSE streaming response

### Pagination

List endpoints support pagination:

```bash
curl "http://localhost:8000/v1/conversations?limit=10&after=conv_123"
```

Response:

```json
{
 "data": [...],
 "next_after": "conv_456"
}
```

### Streaming

Chat completions support Server-Sent Events (SSE) streaming:

```bash
curl -X POST http://localhost:8000/v1/chat/completions \
 -H "Authorization: Bearer <token>" \
 -d '{"model":"jan-v1-4b","messages":[...],"stream":true}'
```

Response:

```
data: {"id":"chat-123","choices":[{"delta":{"content":"Hello"}}]}

data: {"id":"chat-123","choices":[{"delta":{"content":"!"}}]}

data: [DONE]
```

## Interactive API Documentation

Access the interactive Swagger UI:

**Local**: http://localhost:8000/api/swagger/index.html

Try API calls directly from your browser with built-in authentication.

## SDK & Client Libraries

### Official SDKs

Official SDKs are coming soon. In the meantime, use OpenAI-compatible clients with the Jan Server base URL.

### Community SDKs

Contributions welcome! Jan Server is OpenAI-compatible, so most OpenAI client libraries work with minor configuration changes.

#### JavaScript Example (OpenAI SDK)

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "http://localhost:8000/v1",
  apiKey: "your_guest_token_here",
});

const response = await client.chat.completions.create({
  model: "jan-v1-4b",
  messages: [{ role: "user", content: "Hello!" }],
});

console.log(response.choices[0].message.content);
```

## Rate Limits

Rate limiting is enforced by the Kong gateway (port 8000) in all environments, using Kong's
`rate-limiting` plugin with `policy: local`. There is a global limit plus tighter per-route
overrides. Current values (see [integrations/kong/kong.yml](../../integrations/kong/kong.yml)):

| Scope                         | Limit               | Counted by |
| ----------------------------- | ------------------- | ---------- |
| Global (all routes)           | 600/min, 10000/hour | IP         |
| `/llm` proxy                  | 120/min             | consumer   |
| `/v1` (LLM API)               | 120/min             | IP         |
| `/responses` (Response API)   | 100/min             | IP         |
| `/v1/artifacts`               | 100/min             | IP         |
| `/v1/agents`                  | 200/min             | IP         |
| `/mcp` (MCP Tools)            | 200/min             | IP         |
| `/media` (protected)          | 60/min              | IP         |
| `/api/media` (public serving) | 300/min             | IP         |
| `/v1/public/shares`           | 100/min, 1000/hour  | IP         |

When a limit is exceeded, Kong returns `429 Too Many Requests`. Tune these in `kong.yml`.

## API Versioning

All APIs are versioned using URL path versioning:

- Current version: `/v1/`
- Future versions will be: `/v2/`, `/v3/`, etc.

Breaking changes will only occur in new major versions.

## Support

- Docs [Full Documentation](../README.md)
- Bug [Report API Issues](https://github.com/janhq/jan-server/issues)
- Discussion [API Discussions](https://github.com/janhq/jan-server/discussions)

---

**Explore APIs**: [LLM API ->](llm-api/) | [MCP Tools ->](mcp-tools/) | **Interactive Docs**: [Swagger UI ->](http://localhost:8000/api/swagger/index.html)
