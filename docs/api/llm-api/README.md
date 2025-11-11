# LLM API Documentation

The LLM API is the core service for language model completions and conversation management in Jan Server.

## Quick Start

### Base URL
- **Local**: http://localhost:8080
- **Via Gateway**: http://localhost:8000/api/llm
- **Docker**: http://llm-api:8080

### Authentication
All endpoints require authentication enforced by the Kong gateway (`http://localhost:8000`). Kong validates JWTs emitted by Keycloak and also accepts API keys with the `X-API-Key` header, injecting `X-Auth-Method` so downstream services know whether JWT or API key auth was used. Request temporary guest tokens through `/llm/auth/guest-login` and include `Authorization: Bearer <token>` (or `X-API-Key: sk_*`) on every protected endpoint.

```bash
# Get guest token
curl -X POST http://localhost:8000/llm/auth/guest-login

# Response:
{
  "access_token": "eyJhbGc...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "..."
}

# Use token in requests
curl -H "Authorization: Bearer <token>" http://localhost:8000/v1/chat/completions
```

## Key Features

- **OpenAI-Compatible** - Drop-in replacement for OpenAI API
- **Streaming Support** - Real-time response streaming with `stream: true`
- **Conversation Management** - Full CRUD operations on conversations
- **Media Support** - Reference media via `jan_*` IDs
- **Model Abstraction** - Support for vLLM, OpenAI, Anthropic, and more
- **Observability** - OpenTelemetry tracing and structured logging

## Service Ports & Configuration

| Component | Port | Environment Variable |
|-----------|------|---------------------|
| **HTTP Server** | 8080 | `HTTP_PORT` |
| **Database** | 5432 | `DB_DSN` |
| **Keycloak** | 8085 | `KEYCLOAK_BASE_URL` |

### Required Environment Variables

```bash
HTTP_PORT=8080                                    # HTTP listen port
DB_DSN=postgres://jan_user:password@api-db:5432/jan_llm_api?sslmode=disable
LOG_LEVEL=info                                   # debug, info, warn, error
LOG_FORMAT=json                                  # json or text
KEYCLOAK_BASE_URL=http://keycloak:8085          # Keycloak URL
JWKS_URL=http://keycloak:8085/realms/jan/protocol/openid-connect/certs
ISSUER=http://localhost:8090/realms/jan          # Token issuer
AUDIENCE=jan-client                              # JWT audience
```

### Optional Configuration

```bash
OTEL_ENABLED=false                              # Enable OpenTelemetry
OTEL_SERVICE_NAME=llm-api                       # Service name for tracing
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317  # Jaeger endpoint
MEDIA_RESOLVE_URL=http://media-api:8285/v1/media/resolve
MEDIA_RESOLVE_TIMEOUT=5s                        # Media resolution timeout
```

## Main Endpoints

### Chat Completions

**POST** `/v1/chat/completions`

OpenAI-compatible chat completion endpoint.

```bash
# Simple completion
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "temperature": 0.7,
    "max_tokens": 100
  }'

# Streaming completion
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "stream": true
  }'
```

**Request Parameters:**
- `model` (required) - Model identifier (e.g., "jan-v1-4b")
- `messages` (required) - Array of message objects
  - `role` - "system", "user", or "assistant"
  - `content` - Text content (string) or content array (for media)
- `stream` (optional) - Enable streaming responses (default: false)
- `temperature` (optional) - 0.0-2.0, controls randomness (default: 0.7)
- `top_p` (optional) - 0.0-1.0, nucleus sampling (default: 1.0)
- `max_tokens` (optional) - Maximum response length
- `stop` (optional) - Stop sequences

**Response:**
```json
{
  "id": "chatcmpl-...",
  "object": "chat.completion",
  "created": 1699999999,
  "model": "jan-v1-4b",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 12,
    "total_tokens": 22
  }
}
```

### Conversations

**GET** `/v1/conversations`

List all conversations.

```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8000/v1/conversations
```

**POST** `/v1/conversations`

Create a new conversation.

```bash
curl -X POST -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"title": "My Conversation"}' \
  http://localhost:8000/v1/conversations
```

**GET** `/v1/conversations/{id}`

Get a specific conversation with messages.

```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8000/v1/conversations/conv_123
```

### Messages

**GET** `/v1/conversations/{conversation_id}/messages`

List messages in a conversation.

**POST** `/v1/conversations/{conversation_id}/messages`

Add a message to a conversation.

### Models

**GET** `/v1/models`

List available models.

```bash
curl http://localhost:8000/v1/models
```

### Health Check

**GET** `/healthz`

```bash
curl http://localhost:8080/healthz
```

## With Media (Visual Input)

Reference media using `jan_*` IDs from the Media API:

```bash
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b-vision",
    "messages": [
      {
        "role": "user",
        "content": [
          {"type": "text", "text": "What is this?"},
          {
            "type": "image_url",
            "image_url": {
              "url": "jan_01hqr8v9k2x3f4g5h6j7k8m9n0"
            }
          }
        ]
      }
    ]
  }'
```

## Related Services

- **Response API** (Port 8082) - Multi-step orchestration using this service
- **Media API** (Port 8285) - Media resolution for `jan_*` IDs
- **MCP Tools** (Port 8091) - Tool integration for LLM responses
- **Kong Gateway** (Port 8000) - API routing and load balancing

## Error Handling

Common HTTP status codes:

| Code | Meaning |
|------|---------|
| 200 | Success |
| 400 | Invalid request parameters |
| 401 | Unauthorized (invalid/expired token) |
| 403 | Forbidden (insufficient permissions) |
| 404 | Resource not found |
| 429 | Rate limited |
| 500 | Server error |

Example error response:
```json
{
  "error": {
    "message": "Invalid model specified",
    "type": "invalid_request_error",
    "code": "invalid_model"
  }
}
```

## Rate Limiting

Requests are rate-limited per user:
- Default: 100 requests per minute
- Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`

## See Also

- [Architecture Overview](../../architecture/)
- [Development Guide](../../guides/development.md)
- [Testing Guide](../../guides/testing.md)
- [Monitoring Guide](../../guides/monitoring.md)
