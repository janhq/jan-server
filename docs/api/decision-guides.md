# API Decision Guides

Quick reference guides to help you choose the right API and approach for your use case.

## When to Use Which API?

### LLM API vs Response API

**Use LLM API when:**

- You need direct chat completions
- Single-turn or simple multi-turn conversations
- You want to manage conversation history yourself
- Streaming responses in real-time
- Simple Q&A without external tools
- Building a chat interface

**Use Response API when:**

- You need multi-step tool orchestration (search → scrape → analyze)
- Automatic tool selection and chaining
- Complex workflows with several chained tool calls (depth capped by `RESPONSE_MAX_TOOL_DEPTH`, default 50)
- Background processing with a completion callback (`metadata.webhook_url`)
- Want AI to decide which tools to use
- Need execution tracking and monitoring

### Media Upload Methods

The Media API has exactly two ingest endpoints. There is no presigned/client-side S3 upload.

**Use POST /v1/media with a `url` (remote URL) when:**

- Image is already hosted publicly
- You want the server to fetch and store it
- Content deduplication is important (identical bytes return the same `jan_*` ID)

**Use POST /v1/media with a `data:` URL (base64) when:**

- Image was generated client-side (canvas, screenshots)
- Base64 data is already available

**Use POST /v1/media/upload (multipart) when:**

- You have a raw file from a file picker or form
- You would rather stream the bytes than base64-encode them

All three return a `jan_*` ID plus a direct URL. Limits are governed by `MEDIA_MAX_BYTES`
(default 50 MB).

**Decision flowchart:**

```
Do you have a public URL?
├─ Yes → POST /v1/media   with { "url": "https://..." }
└─ No → Do you have the file as base64?
    ├─ Yes → POST /v1/media        with { "url": "data:image/png;base64,..." }
    └─ No  → POST /v1/media/upload  (multipart/form-data, field: file)
```

### Authentication Method Selection

**Use Bearer Tokens when:**

- Development and testing
- Short-lived sessions (5-60 minutes)
- User-facing applications with login flows
- Need token refresh capability
- Guest access is acceptable

**Use API Keys when:**

- Production deployments
- Server-to-server communication
- Long-lived credentials (30-365 days)
- Service accounts and automation
- No user interaction needed
- Simplified authentication flow

**Use Direct Service Ports (8080/8082/8285/8091) when:**

- Internal service-to-service calls within Docker network
- Health checks and monitoring
- Debugging and development
- Want to bypass Kong gateway
- Still requires valid JWT token

## Response API Patterns

### Synchronous vs Background Mode

**Use Synchronous Mode when:**

- Quick operations (<30 seconds expected)
- Need immediate response
- Client can wait for completion
- Simple single-tool calls
- Real-time user interfaces

**Use Background Mode when:**

- Long-running operations (>30 seconds)
- Multiple tool chains (3+ tools)
- Client can poll `GET /v1/responses/{id}` or receive a `metadata.webhook_url` callback
- Want to prevent timeouts
- Building async workflows
- Need to queue multiple requests

### Tool Execution Depth

The orchestrator caps how many sequential tool calls a single response may make. This ceiling
is set by `RESPONSE_MAX_TOOL_DEPTH` (default **50**); the per-call timeout is
`TOOL_EXECUTION_TIMEOUT` (default **300s**).

```
1 hop:  User input → Tool call → Response
3 hops: User input → Tool 1 → Tool 2 → Tool 3 → Response
```

**Visual example:**

```
Query: "Find the latest news on quantum computing and analyze sentiment"

┌─────────┐    ┌───────────────┐    ┌────────┐    ┌─────────────┐    ┌──────────┐
│  Input  │───▶│ google_search │───▶│ scrape │───▶│ LLM Analyze │───▶│ Response │
└─────────┘    └───────────────┘    └────────┘    └─────────────┘    └──────────┘
```

Most workflows finish in a handful of hops; the high default ceiling exists so complex
research chains are not cut off prematurely. Keep chains short where you can — each hop adds
latency and cost.

## Media API Patterns

### Jan ID System

**What are jan\_\* IDs?**

- Unique identifiers for stored media, formatted as `jan_` + a lowercase ULID
  (for example `jan_01hqr8v9k2x3f4g5h6j7k8m9n0`)
- Content-addressed: identical bytes deduplicate to the same ID
- Portable: reference the same media across conversations and requests

**How to use an ID:**

- Fetch the bytes: `GET /v1/media/{id}` (streams the content, or returns a direct URL when
  `MEDIA_PROXY_DOWNLOAD=false`)
- Inspect metadata: `GET /v1/media/{id}/metadata`
- Embed publicly in HTML: use the Kong public route `GET /api/media/{id}`

**Best practices:**

1. Store the `jan_*` ID, not a rendered URL — IDs are stable, URLs may change
2. For `<img>` tags, use the public `/api/media/{id}` path
3. Let the LLM API resolve `jan_*` references in chat content for you

## Rate Limiting Strategy

**Understanding limits:**

- Rate limiting is enforced by the Kong gateway in **all** environments — there is no
  "no limits in development" mode.
- A global limit (600/min, 10000/hour by IP) applies, with tighter per-route overrides
  (e.g. `/v1` 120/min, `/responses` 100/min, `/mcp` 200/min, `/media` 60/min).
- Exceeding a limit returns HTTP 429. See
  [integrations/kong/kong.yml](../../integrations/kong/kong.yml) for exact values.

**Strategies:**

1. **Exponential backoff** on 429, honoring any `Retry-After` header
2. **Batch operations** to reduce request count
3. **Cache** responses that do not change often

## Error Handling Patterns

### Common Error Scenarios

- **401 Unauthorized** — Missing/expired token or invalid API key. Refresh the token
  (`POST /auth/refresh-token`) or re-authenticate.
- **404 Not Found** — The resource ID does not exist or is not owned by the caller.
- **429 Too Many Requests** — A Kong rate limit was hit; back off and retry.
- **500 Internal Server Error** — Upstream failure; retry with backoff and capture the
  `request_id` from the response for support.

## See Also

- [API Examples](examples/README.md) - Working code samples
- [Endpoint Matrix](endpoint-matrix.md) - Full endpoint inventory
- Rate limits: [integrations/kong/kong.yml](../../integrations/kong/kong.yml)
