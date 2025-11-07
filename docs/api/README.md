# API Reference

Complete API documentation for Jan Server services.

## Available APIs

### LLM API
OpenAI-compatible API for chat completions, conversations, and models.

- **[Overview](llm-api/overview.md)** - Service description and capabilities (Coming Soon)
- **[Endpoints](llm-api/endpoints.md)** - Complete endpoint reference (Coming Soon)
- **[Authentication](llm-api/authentication.md)** - Auth methods and token management (Coming Soon)
- **[Examples](llm-api/examples.md)** - Code samples and use cases (Coming Soon)
- **[OpenAPI Spec](llm-api/openapi.md)** - Swagger/OpenAPI documentation (Coming Soon)

### MCP Tools API
Model Context Protocol tools for web search, scraping, and more.

- **[Overview](mcp-tools/overview.md)** - MCP service description (Coming Soon)
- **[Tools Reference](mcp-tools/tools-reference.md)** - Available tools and parameters (Coming Soon)
- **[Providers](mcp-tools/providers.md)** - MCP provider integration
- **[Integration](mcp-tools/integration.md)** - How to integrate MCP tools

## Quick Reference

### Base URLs

| Environment | LLM API | MCP Tools | Gateway |
|-------------|---------|-----------|---------|
| **Local** | http://localhost:8080 | http://localhost:8091 | http://localhost:8000 |
| **Docker** | http://llm-api:8080 | http://mcp-tools:8091 | http://kong:8000 |

**Recommended**: Use the Kong Gateway (port 8000) for all API calls.

### Authentication

All LLM API endpoints require authentication:

```bash
# Get guest token
curl -X POST http://localhost:8000/auth/guest

# Response
{
  "access_token": "eyJhbGci...",
  "refresh_token": "eyJhbGci...",
  "expires_in": 300
}

# Use in requests
curl -H "Authorization: Bearer <access_token>" \
  http://localhost:8000/v1/models
```

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

| Type | Description | HTTP Status |
|------|-------------|-------------|
| `invalid_request_error` | Invalid request parameters | 400 |
| `auth_error` | Authentication failed | 401 |
| `permission_error` | Insufficient permissions | 403 |
| `not_found_error` | Resource not found | 404 |
| `rate_limit_error` | Too many requests | 429 |
| `internal_error` | Server error | 500 |

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

**Local**: http://localhost:8000/v1/swagger/index.html

Try API calls directly from your browser with built-in authentication.

## SDK & Client Libraries

### Official SDKs

(Coming Soon)

### Community SDKs

Contributions welcome! Jan Server is OpenAI-compatible, so most OpenAI client libraries work with minor configuration changes.

#### Python Example (OpenAI SDK)

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8000/v1",
    api_key="your_guest_token_here"
)

response = client.chat.completions.create(
    model="jan-v1-4b",
    messages=[
        {"role": "user", "content": "Hello!"}
    ]
)

print(response.choices[0].message.content)
```

#### JavaScript Example (OpenAI SDK)

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
    baseURL: 'http://localhost:8000/v1',
    apiKey: 'your_guest_token_here',
});

const response = await client.chat.completions.create({
    model: 'jan-v1-4b',
    messages: [
        { role: 'user', content: 'Hello!' }
    ],
});

console.log(response.choices[0].message.content);
```

## Rate Limits

Currently, Jan Server does not enforce rate limits in development mode. 

Production deployments should configure rate limiting via Kong Gateway.

## API Versioning

All APIs are versioned using URL path versioning:

- Current version: `/v1/`
- Future versions will be: `/v2/`, `/v3/`, etc.

Breaking changes will only occur in new major versions.

## Support

- 📚 [Full Documentation](../README.md)
- 🐛 [Report API Issues](https://github.com/janhq/jan-server/issues)
- 💬 [API Discussions](https://github.com/janhq/jan-server/discussions)

---

**Explore APIs**: [LLM API →](llm-api/) | [MCP Tools →](mcp-tools/) | **Interactive Docs**: [Swagger UI →](http://localhost:8000/v1/swagger/)
