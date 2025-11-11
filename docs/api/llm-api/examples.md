# LLM API Examples

All examples assume `make up-full` is running locally so that Kong Gateway is available at `http://localhost:8000`.

## Prerequisites
1. Create `.env` from `.env.template` and run `make setup`.
2. Start the stack: `make up-full`.
3. Grab a guest token through the Kong gateway:
   ```bash
   curl -s -X POST http://localhost:8000/llm/auth/guest-login | jq -r .access_token
   ```
   Save the token as `ACCESS_TOKEN`.

All `/v1/*` requests are routed through Kong, which validates Keycloak JWTs or the custom API key plugin (`X-API-Key: sk_*`) before forwarding to the LLM API.

## 1. Basic Chat Completion (cURL)
```bash
curl -s -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b",
    "messages": [
      {"role": "user", "content": "Give me a fun fact about Saturn."}
    ]
  }' | jq .
```

## 2. Streaming Response (cURL)
```bash
curl -N -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b",
    "messages": [
      {"role": "user", "content": "Explain transformers in two sentences."}
    ],
    "stream": true
  }'
```

## 3. Conversation Management
```bash
# Create a conversation
curl -s -X POST http://localhost:8000/v1/conversations \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Docs Demo"}' | jq .

# List conversations
curl -s http://localhost:8000/v1/conversations \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

## 4. Python (openai>=1.0)
```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8000/v1",
    api_key="YOUR_GUEST_TOKEN"
)

response = client.chat.completions.create(
    model="jan-v1-4b",
    messages=[{"role": "user", "content": "List three cities in France."}]
)

print(response.choices[0].message.content)
```

## 5. JavaScript (openai@4)
```javascript
import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "http://localhost:8000/v1",
  apiKey: process.env.ACCESS_TOKEN,
});

const response = await client.chat.completions.create({
  model: "jan-v1-4b",
  messages: [{ role: "user", content: "What is the Jan Server stack?" }],
});

console.log(response.choices[0].message.content);
```

## 6. With Media (jan_* ID)
```bash
curl -s -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b-vision",
    "messages": [
      {
        "role": "user",
        "content": [
          {"type": "text", "text": "Describe this image"},
          {"type": "image_url", "image_url": {"url": "jan_01hr0..." }}
        ]
      }
    ]
  }'
```
Replace `jan_01hr0...` with a real `jan_*` ID from Media API.

## 7. Projects Management
```bash
# Create a project
curl -s -X POST http://localhost:8000/v1/projects \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Marketing Campaign",
    "instruction": "You are a marketing expert."
  }' | jq .

# List all projects
curl -s http://localhost:8000/v1/projects \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq .

# Get a specific project
PROJECT_ID="proj_123"
curl -s http://localhost:8000/v1/projects/$PROJECT_ID \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq .

# Update project
curl -s -X PATCH http://localhost:8000/v1/projects/$PROJECT_ID \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Project Name",
    "archived": false
  }' | jq .

# Create conversation in project
curl -s -X POST http://localhost:8000/v1/conversations \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Project Conversation",
    "project_id": "'$PROJECT_ID'"
  }' | jq .
```

## 8. API Key Management
```bash
# Create an API key
curl -s -X POST http://localhost:8000/llm/auth/api-keys \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production Key",
    "scopes": ["read", "write"]
  }' | jq .

# Save the returned API key (only shown once)
API_KEY="sk_test_..."

# Use API key instead of Bearer token
curl -s http://localhost:8000/v1/models \
  -H "X-API-Key: $API_KEY" | jq .

# List all API keys
curl -s http://localhost:8000/llm/auth/api-keys \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq .

# Delete an API key
KEY_ID="key_123"
curl -s -X DELETE http://localhost:8000/llm/auth/api-keys/$KEY_ID \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

## 9. Admin - Provider Management
```bash
# Requires admin token
ADMIN_TOKEN="your_admin_token"

# List all providers
curl -s http://localhost:8000/v1/admin/providers \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .

# Register a new provider
curl -s -X POST http://localhost:8000/v1/admin/providers \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "OpenAI",
    "base_url": "https://api.openai.com",
    "api_key": "sk-..."
  }' | jq .

# Update provider
PROVIDER_ID="prov_123"
curl -s -X PATCH http://localhost:8000/v1/admin/providers/$PROVIDER_ID \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}' | jq .
```

## 10. Admin - Model Catalog Management
```bash
# List all catalog models
curl -s http://localhost:8000/v1/admin/models/catalogs \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .

# Get specific model
curl -s http://localhost:8000/v1/admin/models/catalogs/jan-v1-4b \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .

# Update model catalog
curl -s -X PATCH http://localhost:8000/v1/admin/models/catalogs/jan-v1-4b \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"enabled": true, "featured": true}' | jq .

# Bulk toggle models
curl -s -X POST http://localhost:8000/v1/admin/models/catalogs/bulk-toggle \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model_ids": ["jan-v1-4b", "gpt-4"],
    "enabled": true
  }' | jq .
```

## 11. Conversation Items (Messages)
```bash
# Add items to conversation
CONV_ID="conv_123"
curl -s -X POST http://localhost:8000/v1/conversations/$CONV_ID/items \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {
        "type": "message",
        "role": "user",
        "content": [
          {"type": "input_text", "text": "What is AI?"}
        ]
      }
    ]
  }' | jq .

# List conversation items
curl -s http://localhost:8000/v1/conversations/$CONV_ID/items \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq .

# Get specific item
ITEM_ID="item_456"
curl -s http://localhost:8000/v1/conversations/$CONV_ID/items/$ITEM_ID \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq .

# Delete item
curl -s -X DELETE http://localhost:8000/v1/conversations/$CONV_ID/items/$ITEM_ID \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

## 12. Python - Projects and Conversations
```python
import requests

BASE_URL = "http://localhost:8000"
headers = {"Authorization": f"Bearer {ACCESS_TOKEN}"}

# Create project
project_resp = requests.post(
    f"{BASE_URL}/v1/projects",
    headers=headers,
    json={"name": "AI Research", "instruction": "Focus on recent developments"}
)
project_id = project_resp.json()["id"]

# Create conversation in project
conv_resp = requests.post(
    f"{BASE_URL}/v1/conversations",
    headers=headers,
    json={"title": "GPT-4 Discussion", "project_id": project_id}
)
conv_id = conv_resp.json()["id"]

# Add message to conversation
requests.post(
    f"{BASE_URL}/v1/conversations/{conv_id}/items",
    headers=headers,
    json={
        "items": [{
            "type": "message",
            "role": "user",
            "content": [{"type": "input_text", "text": "Explain GPT-4"}]
        }]
    }
)

# Get conversation with messages
conv = requests.get(f"{BASE_URL}/v1/conversations/{conv_id}", headers=headers)
print(conv.json())
```

Use these snippets as templates for SDK integrations and tests.
