# API Examples

A collection of ready-to-use API examples for Jan Server.

## Quick Navigation

- [LLM API Examples](#llm-api) - Chat, conversations, models
- [Response API Examples](#response-api) - Multi-step tool orchestration
- [Media API Examples](#media-api) - Image uploads and jan\_* IDs
- [MCP Tools Examples](#mcp-tools) - Search, scrape, vector store, code execution
- [Cross-Service Examples](#cross-service-examples) - Integration patterns

## LLM API

- **[Comprehensive Examples](../llm-api/comprehensive-examples.md)** - Full coverage including:
  - Authentication (guest tokens, API keys, JWT refresh)
  - Chat completions (basic, streaming, with context)
  - Conversations (CRUD, pagination, search)
  - Messages (add, list, delete)
  - Models and catalogs (listing, admin operations)

## Response API

- **[Comprehensive Examples](../response-api/comprehensive-examples.md)** - Multi-step orchestration including:
  - Single tool execution
  - Multi-step workflows (chaining tools)
  - Analysis tasks (combining search + scrape)
  - Batch operations
  - Error handling and retries

## Media API

- **[Comprehensive Examples](../media-api/comprehensive-examples.md)** - Image handling including:
  - Ingest from a remote URL
  - Ingest from a base64/data URL
  - Multipart file upload
  - Fetching bytes and metadata by `jan_*` ID
  - Integration with LLM API (vision models)

## MCP Tools

- **[Comprehensive Examples](../mcp-tools/comprehensive-examples.md)** - Tool execution including:
  - Tool discovery (list tools, get schemas)
  - Google search (with filters, location)
  - Web scraping (HTML -> Markdown)
  - Vector search (indexing + querying)
  - Python code execution (sandboxed)
  - Real-world scenarios (research, analysis)

## Cross-Service Examples

### Vision + Chat (Media + LLM)

```bash
# 1. Ingest the image via the Media API (Kong strips /media -> service /media)
IMAGE_RESP=$(curl -s -X POST http://localhost:8000/media/media \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/image.jpg"}')

JAN_ID=$(echo $IMAGE_RESP | jq -r '.id')

# 2. Use in chat completion
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b",
    "messages": [{
      "role": "user",
      "content": [
        {"type": "text", "text": "What is in this image?"},
        {"type": "image_url", "image_url": {"url": "'$JAN_ID'"}}
      ]
    }]
  }'
```

### Search + Response (MCP + Response API)

```bash
# Multi-step: Search, scrape, analyze
# Kong route /responses has strip_path=true, so /responses/v1/responses -> service /v1/responses
curl -X POST http://localhost:8000/responses/v1/responses \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b",
    "input": "Research the latest AI model releases and summarize",
    "tools": [
      {"name": "google_search"},
      {"name": "scrape"}
    ]
  }'
```

## Testing Examples

All examples assume:

1. Jan Server is running (`make up-full`)
2. You have a valid access token
3. Kong Gateway is available at `http://localhost:8000`

**Get an access token:**

```bash
curl -X POST http://localhost:8000/llm/auth/guest-login
```

**Try a basic chat:**

```bash
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

**Try a web search:**

```bash
curl -X POST http://localhost:8000/v1/mcp \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "id": 1,
    "params": {
      "name": "google_search",
      "arguments": {"q": "AI news"}
    }
  }'
```

---

**Back to**: [API Documentation](../README.md) | **Service Docs**: [LLM](../llm-api/) | [Response](../response-api/) | [Media](../media-api/) | [MCP](../mcp-tools/)