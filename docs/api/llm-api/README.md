# LLM API Documentation

The LLM API lets you send messages to AI models and get responses. It works like the OpenAI API.

## Quick Start

Examples: [API examples index](../examples/README.md) covers cURL/SDK snippets for every endpoint.

### URLs

- **Direct access**: http://localhost:8080
- **Through gateway** (recommended): http://localhost:8000
- **Inside Docker**: http://llm-api:8080

### Authentication

All endpoints require authentication through the Kong gateway.

**For complete authentication documentation, see [Authentication Guide](../README.md#authentication)**

**Quick guest token:**

```bash
# Get guest token
curl -X POST http://localhost:8000/llm/auth/guest-login

# Use token in requests
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/v1/chat/completions
```

**Key points:**

- Use Kong gateway (port 8000) for all client requests
- Bearer tokens from `/llm/auth/guest-login` work for development
- API keys (`X-API-Key: sk_*`) available via Kong for production
- Direct service access (port 8080) still requires valid JWT

## What You Can Do

- **Chat with AI** - Send messages and get responses (like ChatGPT)
- **Stream responses** - Get word-by-word output in real-time
- **Save conversations** - Keep chat history for later
- **Add images** - Reference images using jan\_\* IDs
- **Multiple models** - Works with vLLM, OpenAI, Anthropic, and others
- **Track everything** - Built-in logging and monitoring
- **(Future) Prompt Orchestration** - Dynamic prompt composition with memory, templates, and conditional modules (see `docs/todo/prompt-orchestration-todo.md`)

## Service Ports & Configuration

| Component       | Port | Environment Variable      |
| --------------- | ---- | ------------------------- |
| **HTTP Server** | 8080 | `HTTP_PORT`               |
| **Database**    | 5432 | `DB_POSTGRESQL_WRITE_DSN` |
| **Keycloak**    | 8085 | `KEYCLOAK_BASE_URL`       |

### Required Environment Variables

```bash
HTTP_PORT=8080 # HTTP listen port
DB_POSTGRESQL_WRITE_DSN=postgres://jan_user:password@api-db:5432/jan_llm_api?sslmode=disable
LOG_LEVEL=info # debug, info, warn, error
LOG_FORMAT=json # json or text
KEYCLOAK_BASE_URL=http://keycloak:8085 # Keycloak URL
JWKS_URL=http://keycloak:8085/realms/jan/protocol/openid-connect/certs
ISSUER=http://localhost:8085/realms/jan # Token issuer (must match Keycloak realm issuer)
ACCOUNT=account # JWT audience/account claim
```

### Optional Configuration

```bash
OTEL_ENABLED=false # Enable OpenTelemetry
OTEL_SERVICE_NAME=llm-api # Service name for tracing
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 # Jaeger endpoint
MEDIA_RESOLVE_URL=http://media-api:8285/v1/media # Media API base used to resolve jan_* IDs
MEDIA_RESOLVE_TIMEOUT=5s # Media resolution timeout
```

> `MEDIA_RESOLVE_URL` points at the Media API `/v1/media` base; the client appends
> `/<id>/metadata` to look an ID up. To route through Kong instead, set it to
> `http://kong:8000/media/media`.

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

List all conversations for the authenticated user.

```bash
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/v1/conversations
```

**Query Parameters:**

- `limit` (optional) - Number of conversations to return (default: 20)
- `after` (optional) - Cursor for pagination
- `order` (optional) - Sort order: "asc" or "desc" (default: "desc")

**POST** `/v1/conversations`

Create a new conversation.

```bash
curl -X POST -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "title": "My Conversation",
 "project_id": "proj_123"
 }' \
 http://localhost:8000/v1/conversations
```

**GET** `/v1/conversations/{conv_public_id}`

Get a specific conversation with its items.

```bash
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/v1/conversations/conv_123
```

**POST** `/v1/conversations/{conv_public_id}`

Update a conversation (title, archived status).

```bash
curl -X POST -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{"title": "Updated Title"}' \
 http://localhost:8000/v1/conversations/conv_123
```

**DELETE** `/v1/conversations/{conv_public_id}`

Delete a conversation.

```bash
curl -X DELETE -H "Authorization: Bearer <token>" \
 http://localhost:8000/v1/conversations/conv_123
```

### Conversation Items (Messages)

**GET** `/v1/conversations/{conv_public_id}/items`

List all items (messages) in a conversation.

```bash
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/v1/conversations/conv_123/items
```

**POST** `/v1/conversations/{conv_public_id}/items`

Add items (messages) to a conversation.

```bash
curl -X POST -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "items": [
 {
 "type": "message",
 "role": "user",
 "content": [
 {"type": "input_text", "text": "Hello!"}
 ]
 }
 ]
 }' \
 http://localhost:8000/v1/conversations/conv_123/items
```

**GET** `/v1/conversations/{conv_public_id}/items/{item_id}`

Get a specific item from a conversation.

```bash
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/v1/conversations/conv_123/items/item_456
```

**DELETE** `/v1/conversations/{conv_public_id}/items/{item_id}`

Delete an item from a conversation.

```bash
curl -X DELETE -H "Authorization: Bearer <token>" \
 http://localhost:8000/v1/conversations/conv_123/items/item_456
```

### Projects

Projects help organize conversations into logical groups.

**POST** `/v1/projects`

Create a new project.

```bash
curl -X POST -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "name": "Marketing Campaign",
 "instruction": "You are a marketing expert."
 }' \
 http://localhost:8000/v1/projects
```

**GET** `/v1/projects`

List all projects for the authenticated user.

```bash
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/v1/projects
```

**Query Parameters:**

- `limit` (optional) - Number of projects to return
- `after` (optional) - Cursor for pagination
- `order` (optional) - Sort order: "asc" or "desc"

**GET** `/v1/projects/{project_id}`

Get a specific project by ID.

```bash
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/v1/projects/proj_123
```

**PATCH** `/v1/projects/{project_id}`

Update a project's name, instruction, or archived status.

```bash
curl -X PATCH -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "name": "Updated Project Name",
 "instruction": "New instruction text",
 "archived": false
 }' \
 http://localhost:8000/v1/projects/proj_123
```

**DELETE** `/v1/projects/{project_id}`

Soft-delete a project.

```bash
curl -X DELETE -H "Authorization: Bearer <token>" \
 http://localhost:8000/v1/projects/proj_123
```

### Models

**GET** `/v1/models`

List all available models.

```bash
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/v1/models
```

**GET** `/v1/models/catalogs/{model_public_id}`

Get details for a specific model from the catalog.

```bash
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/v1/models/catalogs/jan-v1-4b
```

**GET** `/v1/models/providers`

List all available model providers.

```bash
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/v1/models/providers
```

### Health Checks

**GET** `/v1/healthz`

Basic health check endpoint.

```bash
curl http://localhost:8080/v1/healthz
```

**GET** `/v1/readyz`

Readiness check endpoint (service ready to accept traffic).

```bash
curl http://localhost:8080/v1/readyz
```

**GET** `/v1/version`

Get API version and build information.

```bash
curl http://localhost:8080/v1/version
```

### User Settings

**GET** `/v1/users/me/settings`

Retrieves the current user's settings. If no settings exist, returns defaults.

```bash
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/v1/users/me/settings
```

**Response:**

```json
{
  "id": 1,
  "user_id": 123,
  "memory_config": {
    "enabled": true,
    "observe_enabled": true,
    "inject_user_core": true,
    "inject_semantic": true,
    "inject_episodic": false,
    "max_user_items": 3,
    "max_project_items": 5,
    "max_episodic_items": 3,
    "min_similarity": 0.75
  },
  "profile_settings": {
    "base_style": "Friendly",
    "custom_instructions": "",
    "nick_name": "",
    "occupation": "",
    "more_about_you": ""
  },
  "advanced_settings": {
    "web_search": false,
    "code_enabled": false
  },
  "enable_trace": false,
  "enable_tools": true,
  "preferences": {}
}
```

**PATCH** `/v1/users/me/settings`

Updates user settings. Only provided fields are updated (partial update).

```bash
curl -X PATCH http://localhost:8000/v1/users/me/settings \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
   "memory_config": {
     "enabled": true,
     "max_user_items": 5
   },
   "profile_settings": {
     "base_style": "Professional",
     "nick_name": "Dev"
   }
 }'
```

**Settings Groups:**

| Group               | Fields                                                                           | Purpose                       |
| ------------------- | -------------------------------------------------------------------------------- | ----------------------------- |
| `memory_config`     | `enabled`, `observe_enabled`, `inject_*`, `max_*`, `min_similarity`              | Memory and retrieval controls |
| `profile_settings`  | `base_style`, `custom_instructions`, `nick_name`, `occupation`, `more_about_you` | User profile and preferences  |
| `advanced_settings` | `web_search`, `code_enabled`                                                     | Advanced feature toggles      |
| Top-level           | `enable_trace`, `enable_tools`, `preferences`                                    | System features               |

**Memory Configuration:**

- `enabled` - Master toggle for all memory features
- `observe_enabled` - Automatically learn from conversations
- `inject_user_core` - Include user core facts in prompts
- `inject_semantic` - Include semantic project facts
- `inject_episodic` - Include episodic conversation history
- `max_user_items` (0-20) - Maximum user memory items to retrieve
- `max_project_items` (0-50) - Maximum project facts to retrieve
- `max_episodic_items` (0-20) - Maximum episodic events to retrieve
- `min_similarity` (0.0-1.0) - Minimum relevance score for memory retrieval

**Profile Settings:**

- `base_style` - Conversation style: `"Concise"`, `"Friendly"`, or `"Professional"`
- `custom_instructions` - Additional behavior and tone preferences
- `nick_name` - What should Jan call you? (alias: `nickname` accepted on input)
- `occupation` - Your occupation or role
- `more_about_you` - Additional information about yourself

**Advanced Settings:**

- `web_search` - Enable automatic web search (privacy consideration)
- `code_enabled` - Enable code execution features (security consideration)

## With Media (Visual Input)

Reference media using `jan_*` IDs from the Media API:

```bash
curl -X POST http://localhost:8000/v1/chat/completions \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "model": "jan-v2-30b",
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

Use any vision-capable model you have configured (for local-only setups, point `jan-cli` at a remote provider such as OpenAI, Anthropic, or Qwen VL).

## Related Services

- **Response API** (Port 8082) - Multi-step orchestration using this service. See [Decision Guide](../decision-guides.md#llm-api-vs-response-api) for when to use each.
- **Media API** (Port 8285) - Media resolution for `jan_*` IDs. See [Media Upload Guide](../decision-guides.md#media-upload-methods) and [Jan ID System](../decision-guides.md#jan-id-system).
- **MCP Tools** (Port 8091) - Tool integration for LLM responses. See [MCP Tools Documentation](../mcp-tools/).
- **Kong Gateway** (Port 8000) - API routing and load balancing. See [Authentication Guide](../README.md#authentication).

## Error Handling

Common HTTP status codes:

| Code | Meaning                              |
| ---- | ------------------------------------ |
| 200  | Success                              |
| 400  | Invalid request parameters           |
| 401  | Unauthorized (invalid/expired token) |
| 403  | Forbidden (insufficient permissions) |
| 404  | Resource not found                   |
| 429  | Rate limited                         |
| 500  | Server error                         |

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

Requests routed through Kong inherit its rate-limiting plugin (enforced in all environments):

- A global limit of 600/min (10000/hour) per client IP applies, plus per-route overrides.
- The `/v1` LLM API routes are capped at 120/min; the `/llm` proxy at 120/min per consumer.
- Exceeding a limit returns HTTP 429.

Exact values live in [integrations/kong/kong.yml](../../../integrations/kong/kong.yml). Calling
the service directly on port 8080 bypasses the gateway rate limiter (useful for internal health
checks) but still requires a valid JWT.

## See Also

- [Architecture Overview](../../architecture/)
- [Development Guide](../../guides/development.md)
- [Testing Guide](../../guides/testing.md)
- [Monitoring Guide](../../guides/monitoring.md)
