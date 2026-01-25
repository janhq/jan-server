# Serper Image Search MCP Tool

Image search via the Serper API, integrated through the Model Context Protocol (MCP).

## Overview

The Serper image search tool (`image_search`) enables agents to search for images across the web using the Serper Images API. It returns image results with metadata including URLs, dimensions, thumbnails, and source information.

## Prerequisites

- Valid Serper API key (obtain from [serper.dev](https://serper.dev))
- MCP tools service running on port 8091
- `SERPER_ENABLED=true` in configuration

## Configuration

### Environment Variables

```bash
SERPER_API_KEY=your_serper_api_key_here
SERPER_ENABLED=true
```

### Docker Compose

```yaml
services:
  mcp-tools:
    environment:
      SERPER_API_KEY: ${SERPER_API_KEY}
      SERPER_ENABLED: true
```

## MCP Tool Definition

```json
{
  "name": "image_search",
  "description": "Search for images using Serper API. Returns image URLs, dimensions, thumbnails, and source information.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "q": {
        "type": "string",
        "description": "Search query (required)"
      },
      "num": {
        "type": "integer",
        "description": "Number of results (default: 10, max: 100)"
      },
      "gl": {
        "type": "string",
        "description": "Country code (e.g., 'us', 'uk', 'de')"
      },
      "hl": {
        "type": "string",
        "description": "Language code (e.g., 'en', 'de', 'fr')"
      },
      "autocorrect": {
        "type": "boolean",
        "description": "Enable autocorrect (default: true)"
      },
      "offline_mode": {
        "type": "boolean",
        "description": "Force offline mode (default: false)"
      }
    },
    "required": ["q"]
  }
}
```

## API Response Format

### Tool Payload Response

```json
{
  "query": "mountain landscape",
  "engine": "serper",
  "live": true,
  "cache_status": "live",
  "metadata": {
    "q": "mountain landscape",
    "gl": "us",
    "hl": "en",
    "type": "images",
    "engine": "serper"
  },
  "results": [
    {
      "position": 1,
      "title": "Beautiful Mountain Landscape",
      "image_url": "https://example.com/image.jpg",
      "image_width": 1200,
      "image_height": 800,
      "thumbnail_url": "https://encrypted-tbn0.gstatic.com/...",
      "thumbnail_width": 275,
      "thumbnail_height": 183,
      "source": "Example Photography",
      "domain": "example.com",
      "link": "https://example.com/gallery/mountain",
      "creator": "John Doe",
      "credit": "Example Photography"
    }
  ]
}
```

### Raw Serper API Response

The `raw` field contains the original Serper API response:

```json
{
  "searchParameters": {
    "q": "lion",
    "gl": "us",
    "hl": "en",
    "type": "images",
    "autocorrect": true,
    "engine": "google",
    "num": 10
  },
  "images": [
    {
      "title": "Image title",
      "imageUrl": "https://example.com/image.jpg",
      "imageWidth": 850,
      "imageHeight": 834,
      "thumbnailUrl": "https://encrypted-tbn0.gstatic.com/...",
      "thumbnailWidth": 227,
      "thumbnailHeight": 222,
      "source": "Example Source",
      "domain": "example.com",
      "link": "https://example.com/page",
      "googleUrl": "https://www.google.com/imgres?...",
      "position": 1
    }
  ]
}
```

## Usage

### Example MCP Request

```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "image_search",
    "arguments": {
      "q": "mountain landscape",
      "num": 10,
      "gl": "us",
      "hl": "en"
    }
  },
  "id": 1
}
```

### Example via HTTP

```bash
curl -X POST http://localhost:8091/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "image_search",
      "arguments": {
        "q": "cat",
        "num": 5
      }
    },
    "id": 1
  }'
```

## Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `q` | string | required | Search query string |
| `num` | integer | 10 | Number of results to return (1-100) |
| `gl` | string | - | Geographic location (ISO 3166-1 alpha-2 country code) |
| `hl` | string | - | Result language (ISO 639-1 language code) |
| `autocorrect` | boolean | true | Enable query autocorrect |
| `offline_mode` | boolean | false | Force offline mode (returns error if enabled) |

## Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `position` | int | Result position in search results |
| `title` | string | Image title/caption |
| `image_url` | string | Direct URL to the full image |
| `image_width` | int | Image width in pixels |
| `image_height` | int | Image height in pixels |
| `thumbnail_url` | string | URL to thumbnail version |
| `thumbnail_width` | int | Thumbnail width in pixels |
| `thumbnail_height` | int | Thumbnail height in pixels |
| `source` | string | Image source/attribution |
| `domain` | string | Domain hosting the image |
| `link` | string | Link to page containing the image |
| `creator` | string | Image creator/photographer (if available) |
| `credit` | string | Image credit attribution (if available) |

## Provider Support

Currently, only Serper API supports image search. If Serper is disabled or the API key is missing, the tool returns an error.

| Provider | Image Search Support |
|----------|---------------------|
| Serper | ✅ Supported |
| Exa | ❌ Not supported |
| Tavily | ❌ Not supported |
| SearXNG | ❌ Not supported |

## Rate Limiting

- Serper API applies rate limits based on your subscription plan
- Monitor API usage in your Serper dashboard
- The client uses exponential backoff retry for transient failures
- Circuit breaker protection prevents cascading failures

## Error Handling

```json
{
  "query": "test",
  "engine": "error",
  "live": false,
  "cache_status": "error",
  "metadata": {
    "error": "image search unavailable: Serper provider not enabled or missing API key"
  },
  "results": []
}
```

Common errors:
- `image search unavailable: offline mode is enabled` - Offline mode requested
- `image search unavailable: Serper provider not enabled or missing API key` - Serper not configured
- `Serper image search API error (status 401)` - Invalid API key
- `Serper image search API error (status 429)` - Rate limit exceeded
- `serper circuit breaker is open` - Too many recent failures

## Comparison with Web Search

| Feature | `google_search` | `image_search` |
|---------|----------------|----------------|
| Result type | Web pages (organic) | Images |
| Metadata | Title, snippet, link | Image URL, dimensions, source |
| Providers | Serper, Exa, Tavily, SearXNG | Serper only |
| Fallback chain | ✅ Yes | ❌ No |
| Use case | General information | Visual content discovery |

## Best Practices

1. **Use descriptive queries** - More specific queries yield better image results
2. **Set reasonable result limits** - Default 10 is usually sufficient for most use cases
3. **Handle missing metadata** - Some fields (`creator`, `credit`) may be empty
4. **Cache results** - Avoid duplicate searches for the same query
5. **Respect image rights** - Verify licensing before using images in your application
6. **Use thumbnails for previews** - Use `thumbnail_url` for UI previews to reduce bandwidth

## Metrics & Observability

The tool records the following metrics:

- `mcp_tool_call_duration_seconds{tool="image_search",provider="serper",status="success|error"}`
- `mcp_tool_tokens{tool="image_search",provider="serper"}` - Estimated token count
- `external_provider_request_total{operation="image_search",provider="serper"}`
- `external_provider_latency_seconds{provider="serper"}`

## Integration with Response API

The image search tool can be used in multi-step orchestrations via the Response API:

```json
{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "Find images of the Eiffel Tower"}],
  "tools": [
    {"type": "function", "function": {"name": "image_search"}}
  ]
}
```

## Related Documentation

- [MCP Tools Service README](../../services/mcp-tools/README.md)
- [Search Tool Documentation](./search.md)
- [Serper API Documentation](https://serper.dev/docs)
