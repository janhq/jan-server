# LLM API Comprehensive Examples

Complete working examples for LLM API endpoints with JavaScript and cURL.

## Table of Contents

- [Authentication](#authentication)
  - [Get Bearer Token](#get-bearer-token)
  - [Refresh Token](#refresh-token)
  - [Revoke Token](#revoke-token)
  - [API Key Management](#api-key-management)
- [Conversations](#conversations)
- [Messages](#messages)
- [Chat Completions](#chat-completions)
- [Models & Catalogs](#models--catalogs)
- [User Settings](#user-settings)
- [Admin Operations](#admin-operations)
  - [Provider Models](#provider-models-admin)
  - [MCP Tools](#mcp-tools-admin)

---

## Authentication

### Get Bearer Token

**JavaScript:**

```javascript
// Get guest token
const authResponse = await fetch("http://localhost:8000/llm/auth/guest-login", {
  method: "POST",
});
const authData = await authResponse.json();
const token = authData.access_token;

// Use in subsequent requests
const headers = { Authorization: `Bearer ${token}` };
```

**cURL:**

```bash
# Get token
TOKEN=$(curl -s -X POST http://localhost:8000/llm/auth/guest-login | jq -r '.access_token')

# Use in requests
curl -H "Authorization: Bearer $TOKEN" http://localhost:8000/v1/conversations
```

### Refresh Token

**JavaScript:**

```javascript
// Refresh expired token
const refreshResponse = await fetch(
  "http://localhost:8000/llm/auth/refresh-token",
  {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  },
);

const newTokens = await refreshResponse.json();
console.log(`Token refreshed, expires in: ${newTokens.expires_in}s`);
```

**cURL:**

```bash
curl -X POST http://localhost:8000/llm/auth/refresh-token \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "'"$REFRESH_TOKEN"'"}'
```

### Revoke Token

**JavaScript:**

```javascript
// Revoke token
const revokeResponse = await fetch("http://localhost:8000/llm/auth/revoke", {
  method: "POST",
  headers: {
    Authorization: `Bearer ${token}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ token: token }),
});

if (revokeResponse.ok) {
  console.log("Token revoked successfully");
}
```

**cURL:**

```bash
curl -X POST http://localhost:8000/llm/auth/revoke \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"token": "'"$TOKEN"'"}'
```

### API Key Management

#### Create API Key

**JavaScript:**

```javascript
// Create API key
const createKeyResponse = await fetch(
  "http://localhost:8000/llm/auth/api-keys",
  {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      name: "Mobile App Key",
      expires_in_days: 365,
      scopes: ["read", "write"],
    }),
  },
);

const keyData = (await createKeyResponse.json()).data;
console.log(`API Key: ${keyData.key}`);
console.log(`Key ID: ${keyData.id}`);

// Save this key - it's only shown once!
const apiKey = keyData.key;
```

**cURL:**

```bash
curl -X POST http://localhost:8000/llm/auth/api-keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CLI Tool Key",
    "expires_in_days": 30,
    "scopes": ["read"]
  }'
```

#### List API Keys

**JavaScript:**

```javascript
// List API keys
const listKeysResponse = await fetch(
  "http://localhost:8000/llm/auth/api-keys",
  {
    headers: { Authorization: `Bearer ${token}` },
  },
);

const keys = (await listKeysResponse.json()).data;
console.log(`Total API keys: ${keys.length}`);
keys.forEach((key) => {
  const status = key.is_active ? "Active" : "Revoked";
  console.log(`  - ${key.name} (${key.id}) - ${status}`);
});
```

**cURL:**

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8000/llm/auth/api-keys | jq
```

#### Revoke API Key

**JavaScript:**

```javascript
// Revoke API key
const keyId = "key_abc123";

const revokeKeyResponse = await fetch(
  `http://localhost:8000/llm/auth/api-keys/${keyId}`,
  {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  },
);

if (revokeKeyResponse.ok) {
  console.log(`API key ${keyId} revoked`);
}
```

**cURL:**

```bash
KEY_ID="key_abc123"
curl -X DELETE "http://localhost:8000/llm/auth/api-keys/$KEY_ID" \
  -H "Authorization: Bearer $TOKEN"
```

#### Use API Key

**JavaScript:**

```javascript
// Use API key for authentication
const apiKey = "sk_prod_abc123xyz...";
const apiHeaders = { "X-API-Key": apiKey };

const response = await fetch("http://localhost:8000/v1/conversations", {
  headers: apiHeaders,
});

const conversations = (await response.json()).data;
console.log(`Found ${conversations.length} conversations`);
```

**cURL:**

```bash
API_KEY="sk_prod_abc123xyz..."
curl -H "X-API-Key: $API_KEY" \
  http://localhost:8000/v1/conversations
```

---

## Conversations

### Create Conversation

**JavaScript:**

```javascript
const token = "your-token-here";
const headers = { Authorization: `Bearer ${token}` };

const response = await fetch("http://localhost:8000/v1/conversations", {
  method: "POST",
  headers: {
    ...headers,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    title: `Chat - ${new Date().toISOString().split("T")[0]}`,
    metadata: {
      source: "javascript-example",
      project_id: "proj_123",
    },
  }),
});

const { data: conversation } = await response.json();
console.log(`Created conversation: ${conversation.id}`);
```

**cURL:**

```bash
curl -X POST http://localhost:8000/v1/conversations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My First Conversation",
    "metadata": {
      "source": "curl-example"
    }
  }' | jq '.data.id'
```

### List Conversations

**JavaScript:**

Pagination is cursor-based: pass `after` with the `next_after` value from the previous page.

```javascript
const response = await fetch(
  "http://localhost:8000/v1/conversations?limit=20",
  { headers },
);
const { data: conversations, next_after } = await response.json();

conversations.forEach((conv) => {
  console.log(`- ${conv.title} (${conv.id})`);
});

// Fetch the next page (if any)
if (next_after) {
  await fetch(
    `http://localhost:8000/v1/conversations?limit=20&after=${next_after}`,
    { headers },
  );
}
```

**cURL:**

```bash
curl "http://localhost:8000/v1/conversations?limit=20" \
  -H "Authorization: Bearer $TOKEN" | jq '.data[] | {id, title}'

# Next page
curl "http://localhost:8000/v1/conversations?limit=20&after=conv_456" \
  -H "Authorization: Bearer $TOKEN" | jq '.data[] | {id, title}'
```

### Get Single Conversation

**JavaScript:**

```javascript
const conversationId = "conv_abc123";

const response = await fetch(
  `http://localhost:8000/v1/conversations/${conversationId}`,
  { headers },
);
const { data: conversation } = await response.json();

console.log(`Title: ${conversation.title}`);
console.log(`Messages: ${conversation.items.length}`);
```

### Update Conversation

**JavaScript:**

Conversation updates use `POST` (not PATCH) on the conversation resource.

```javascript
const response = await fetch(
  `http://localhost:8000/v1/conversations/${conversationId}`,
  {
    method: "POST",
    headers: {
      ...headers,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      title: "Updated Title",
      metadata: { project_id: "proj_new" },
    }),
  },
);
const { data: updated } = await response.json();
console.log(`Updated: ${updated.title}`);
```

### Delete Conversation

**JavaScript:**

```javascript
const response = await fetch(
  `http://localhost:8000/v1/conversations/${conversationId}`,
  {
    method: "DELETE",
    headers,
  },
);

if (response.status === 204) {
  console.log("Conversation deleted");
}
```

### Delete All Conversations

There is no per-ID bulk-delete endpoint. A `DELETE` on the collection removes **all** of the
authenticated user's conversations.

**JavaScript:**

```javascript
const response = await fetch("http://localhost:8000/v1/conversations", {
  method: "DELETE",
  headers,
});

if (response.ok) {
  console.log("All conversations deleted");
}
```

### Share Conversation

**JavaScript:**

```javascript
const response = await fetch(
  `http://localhost:8000/v1/conversations/${conversationId}/share`,
  {
    method: "POST",
    headers: {
      ...headers,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      expires_in: 7 * 24 * 3600, // 7 days
      read_only: true,
      allow_feedback: false,
    }),
  },
);

const { data: shareData } = await response.json();
console.log(`Share link: ${shareData.share_link}`);
```

---

## Messages

### Add Message(s)

Items are added as an `items` array. Each message carries a typed `content` array.

**JavaScript:**

```javascript
const response = await fetch(
  `http://localhost:8000/v1/conversations/${conversationId}/items`,
  {
    method: "POST",
    headers: {
      ...headers,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      items: [
        {
          type: "message",
          role: "user",
          content: [{ type: "input_text", text: "Hello! How are you?" }],
        },
      ],
    }),
  },
);

const { data: items } = await response.json();
console.log(`Added ${items.length} item(s)`);
```

### Get Messages

**JavaScript:**

```javascript
const response = await fetch(
  `http://localhost:8000/v1/conversations/${conversationId}/items?limit=50`,
  { headers },
);

const { data: messages, next_after } = await response.json();
messages.forEach((msg) => {
  console.log(`${msg.role.toUpperCase()}: ${JSON.stringify(msg.content)}`);
});
```

### Delete Message

**JavaScript:**

```javascript
const response = await fetch(
  `http://localhost:8000/v1/conversations/${conversationId}/items/${messageId}`,
  {
    method: "DELETE",
    headers,
  },
);

if (response.status === 204) {
  console.log("Message deleted");
}
```

---

## Chat Completions

### Stream Response (OpenAI Compatible)

**JavaScript:**

```javascript
const response = await fetch("http://localhost:8000/v1/chat/completions", {
  method: "POST",
  headers: {
    ...headers,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "jan-v1-4b",
    messages: [
      { role: "system", content: "You are a helpful assistant." },
      { role: "user", content: "Write a short poem about AI." },
    ],
    temperature: 0.7,
    max_tokens: 200,
    stream: true,
  }),
});

const reader = response.body.getReader();
const decoder = new TextDecoder();

while (true) {
  const { done, value } = await reader.read();
  if (done) break;

  const text = decoder.decode(value);
  const lines = text.split("\n");

  for (const line of lines) {
    if (line.startsWith("data: ")) {
      const data = JSON.parse(line.replace("data: ", ""));
      const content = data.choices[0]?.delta?.content || "";
      process.stdout.write(content);
    }
  }
}
```

### Non-Streaming Response

**JavaScript:**

```javascript
const response = await fetch("http://localhost:8000/v1/chat/completions", {
  method: "POST",
  headers: {
    ...headers,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "jan-v1-4b",
    messages: [{ role: "user", content: "Explain quantum computing briefly." }],
    temperature: 0.7,
    max_tokens: 150,
    stream: false,
  }),
});

const result = await response.json();
const content = result.choices[0].message.content;
console.log(`Response: ${content}`);
console.log(`Tokens: ${result.usage.total_tokens}`);
```

---

## Models & Catalogs

### List Available Models

`GET /v1/models` returns the models available to the caller (OpenAI-compatible shape).

**JavaScript:**

```javascript
const response = await fetch("http://localhost:8000/v1/models", { headers });

const { data: models } = await response.json();
models.forEach((model) => {
  console.log(`- ${model.id}`);
});
```

### Get a Model Catalog Entry

`GET /v1/models/catalogs/{model_public_id}` returns catalog detail (supported parameters) for
one model.

**JavaScript:**

```javascript
const response = await fetch(
  "http://localhost:8000/v1/models/catalogs/jan-v1-4b",
  { headers },
);

const catalog = await response.json();
console.log(catalog);
```

---

## User Settings

### Get User Settings

**JavaScript:**

```javascript
const response = await fetch("http://localhost:8000/v1/users/me/settings", {
  headers,
});

const { data: settings } = await response.json();
console.log(`Base style: ${settings.profile_settings.base_style}`);
console.log(`Nickname: ${settings.profile_settings.nick_name}`);
```

### Update User Settings

**JavaScript:**

```javascript
const response = await fetch("http://localhost:8000/v1/users/me/settings", {
  method: "PATCH",
  headers: {
    ...headers,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    profile_settings: {
      base_style: "Professional",
      nick_name: "Assistant Pro",
      occupation: "Software Engineer",
    },
    memory_config: {
      min_similarity: 0.85,
      max_user_items: 5,
      max_project_items: 10,
    },
    advanced_settings: {
      web_search: true,
      code_enabled: false,
    },
  }),
});

const updated = await response.json();
console.log("Settings updated successfully");
```

> Field names must match the [settings schema in the README](README.md#user-settings):
> `memory_config` uses `max_user_items` / `max_project_items` / `max_episodic_items`
> (not `max_items`), and `advanced_settings` uses `web_search` / `code_enabled`.

---

## Admin Operations

### Provider Models (Admin)

#### List Provider Models

**JavaScript:**

```javascript
const adminToken = "your-admin-token";
const adminHeaders = { Authorization: `Bearer ${adminToken}` };

const response = await fetch(
  "http://localhost:8000/v1/admin/models/provider-models",
  { headers: adminHeaders },
);

const providerModels = (await response.json()).data;
console.log(`Total provider models: ${providerModels.length}`);
providerModels.forEach((model) => {
  const status = model.enabled ? "Enabled" : "Disabled";
  console.log(`  - ${model.model_id} (${model.provider}) - ${status}`);
});
```

**cURL:**

```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://localhost:8000/v1/admin/models/provider-models | jq
```

#### Get Provider Model Details

**JavaScript:**

```javascript
const modelId = "gpt-4o-mini";

const response = await fetch(
  `http://localhost:8000/v1/admin/models/provider-models/${modelId}`,
  { headers: adminHeaders },
);

const model = (await response.json()).data;
console.log(`Model: ${model.model_id}`);
console.log(`Provider: ${model.provider}`);
console.log(`Context Window: ${model.context_window} tokens`);
console.log(`Capabilities: ${model.capabilities}`);
```

**cURL:**

```bash
MODEL_ID="gpt-4o-mini"
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:8000/v1/admin/models/provider-models/$MODEL_ID" | jq
```

#### Update Provider Model Configuration

**JavaScript:**

```javascript
const modelId = "claude-3-5-sonnet-20241022";

const response = await fetch(
  `http://localhost:8000/v1/admin/models/provider-models/${modelId}`,
  {
    method: "PATCH",
    headers: {
      ...adminHeaders,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      enabled: true,
      default_temperature: 0.7,
      rate_limit: {
        requests_per_minute: 50,
      },
      tags: ["production"],
    }),
  },
);

const updatedModel = (await response.json()).data;
console.log(`Model ${modelId} updated`);
```

**cURL:**

```bash
curl -X PATCH "http://localhost:8000/v1/admin/models/provider-models/claude-3-5-sonnet-20241022" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "default_temperature": 0.7,
    "tags": ["production", "high-quality"]
  }'
```

#### Bulk Toggle Provider Models

**JavaScript:**

```javascript
const response = await fetch(
  "http://localhost:8000/v1/admin/models/provider-models/bulk-toggle",
  {
    method: "POST",
    headers: {
      ...adminHeaders,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      model_ids: ["gpt-4o-mini", "gpt-4o", "claude-3-5-sonnet-20241022"],
      enabled: true,
      reason: "Enabling primary production models",
    }),
  },
);

const result = (await response.json()).data;
console.log(`Updated ${result.count} models`);
```

**cURL:**

```bash
curl -X POST http://localhost:8000/v1/admin/models/provider-models/bulk-toggle \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model_ids": ["gpt-4o-mini", "gpt-4o"],
    "enabled": true,
    "reason": "Production deployment"
  }'
```

#### List Model Catalogs (Admin)

**JavaScript:**

```javascript
const response = await fetch(
  "http://localhost:8000/v1/admin/models/catalogs?provider=openai&enabled=true",
  { headers: adminHeaders },
);

const catalogs = (await response.json()).data;
catalogs.forEach((catalog) => {
  console.log(`Catalog: ${catalog.public_id}`);
  console.log(`  Models: ${catalog.models.length}`);
  console.log(`  Usage: ${catalog.total_requests} requests`);
});
```

**cURL:**

```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:8000/v1/admin/models/catalogs?provider=openai" | jq
```

#### Bulk Toggle Model Catalogs

**cURL:**

```bash
curl -X POST http://localhost:8000/v1/admin/models/catalogs/bulk-toggle \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "catalog_ids": ["openai-gpt4"],
    "enabled": false,
    "reason": "Cost control"
  }'
```

### MCP Tools (Admin)

MCP tool admin configuration is served by the LLM API under `/v1/admin/mcp-tools`.

#### List MCP Tools (Admin)

```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://localhost:8000/v1/admin/mcp-tools | jq
```

#### Enable / Disable an MCP Tool (Admin)

```bash
# Toggle a tool's enabled state by its id
curl -X PATCH "http://localhost:8000/v1/admin/mcp-tools/$TOOL_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'
```

---

## Complete Example: Multi-Turn Conversation

```bash
# 1. Authenticate
TOKEN=$(curl -s -X POST http://localhost:8000/llm/auth/guest-login | jq -r '.access_token')

# 2. Create a conversation
CONV_ID=$(curl -s -X POST http://localhost:8000/v1/conversations \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"title": "Trip planning"}' | jq -r '.data.id')

# 3. Add a user message
curl -s -X POST "http://localhost:8000/v1/conversations/$CONV_ID/items" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"items":[{"type":"message","role":"user",
        "content":[{"type":"input_text","text":"Suggest a 3-day itinerary for Kyoto."}]}]}'

# 4. Get a completion
curl -s -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"model":"jan-v1-4b","messages":[{"role":"user","content":"Suggest a 3-day itinerary for Kyoto."}]}' \
  | jq -r '.choices[0].message.content'

# 5. List the stored items
curl -s "http://localhost:8000/v1/conversations/$CONV_ID/items?limit=50" \
  -H "Authorization: Bearer $TOKEN" | jq '.data'
```

## Related Documentation

- [LLM API Reference](README.md) - Full endpoint documentation
- [Decision Guide: LLM vs Response API](../decision-guides.md#llm-api-vs-response-api) - Choose the right API
- [Decision Guide: Authentication Methods](../decision-guides.md#authentication-method-selection) - Choose auth approach
- [Endpoint Matrix](../endpoint-matrix.md) - Full endpoint inventory
- [Response API](../response-api/) - Multi-step tool orchestration
- [Media API](../media-api/) - Image uploads and jan\_\* IDs
- [MCP Tools](../mcp-tools/) - Available tools
- [Examples Index](../examples/README.md) - Cross-service examples
