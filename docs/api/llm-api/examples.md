# LLM API Examples

All examples assume `make up-full` is running locally so that Kong Gateway is available at `http://localhost:8000`.

## Prerequisites
1. Create `.env` from `.env.template` and run `make setup`.
2. Start the stack: `make up-full`.
3. Grab a guest token:
   ```bash
   curl -s -X POST http://localhost:8000/auth/guest | jq -r .access_token
   ```
   Save the token as `ACCESS_TOKEN`.

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
          {"type": "image_url", "image_url": {"url": "data:image/png;jan_01hr0..." }}
        ]
      }
    ]
  }'
```
Replace `data:image/png;jan_01hr0...` with a real `jan_*` ID from Media API.

Use these snippets as templates for SDK integrations and tests.
