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
- Complex workflows with up to 8 tool calls
- Background processing with webhooks
- Want AI to decide which tools to use
- Need execution tracking and monitoring

**Example comparison:**

### Media Upload Methods

**Use POST /v1/media (remote_url) when:**

- Image is already hosted publicly
- You want to avoid client-side uploads
- Working with URLs from external sources
- Content deduplication is important

**Use POST /v1/media/prepare-upload (presigned URL) when:**

- Large file uploads (>10MB)
- Need client-side direct S3 upload
- Want to minimize server load
- Building mobile/web apps with file pickers

**Use POST /v1/media (data_url) when:**

- Small images (<5MB)
- Image generated client-side (canvas, screenshots)
- Base64 data already available
- Simple quick uploads

**Decision flowchart:**

```
Do you have a public URL?
├─ Yes → Use remote_url method
└─ No → Is file >10MB?
    ├─ Yes → Use prepare-upload (presigned)
    └─ No → Is it base64?
        ├─ Yes → Use data_url
        └─ No → Use prepare-upload
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
- Client can poll or use webhooks
- Want to prevent timeouts
- Building async workflows
- Need to queue multiple requests

**Pattern comparison:**

### Tool Execution Depth

**Understanding depth parameter:**

```
depth=1: User input → Tool call → Response
depth=3: User input → Tool 1 → Tool 2 → Tool 3 → Response
depth=8: Maximum chain length
```

**Visual example:**

```
Query: "Find the latest news on quantum computing and analyze sentiment"

Depth 2:
┌─────────┐    ┌───────────────┐    ┌─────────────┐    ┌──────────┐
│  Input  │───▶│ google_search │───▶│ LLM Analyze │───▶│ Response │
└─────────┘    └───────────────┘    └─────────────┘    └──────────┘

Depth 4:
┌─────────┐    ┌───────────────┐    ┌────────┐    ┌─────────────┐    ┌──────────┐
│  Input  │───▶│ google_search │───▶│ scrape │───▶│ LLM Analyze │───▶│ Response │
└─────────┘    └───────────────┘    └────────┘    └─────────────┘    └──────────┘
```

**Choosing depth:**

- `depth=1`: Single tool call, simple operations
- `depth=2-3`: Standard workflows, most use cases (recommended)
- `depth=4-6`: Complex research, multi-stage analysis
- `depth=7-8`: Advanced pipelines, use sparingly (cost/latency)

## Media API Patterns

### Jan ID System

**What are jan\_\* IDs?**

- Unique identifiers for uploaded media: `jan_01hqr8v9k2x3f4g5h6j7k8m9n0`
- Content-addressed: Same image = same ID (deduplication)
- Portable: Use across conversations and requests
- Resolvable: Convert to presigned URLs on demand

**When to resolve IDs:**

**Best practices:**

1. Store `jan_*` IDs in your database, not presigned URLs (URLs expire)
2. Resolve only when needed (presigned URLs valid 7 days)
3. Use batch resolution for multiple images
4. Let LLM API handle resolution when possible

### Presigned URL Workflow

**Decision flow:**

```
Need to display image?
├─ Stored as jan_* ID?
│  └─ Call /v1/media/resolve → Get presigned URL → Display
└─ Already have presigned URL?
   ├─ Check expiry (valid 7 days)
   │  ├─ Expired? → Call /v1/media/{id}/presign → Get new URL
   │  └─ Valid? → Use directly
   └─ Unknown? → Resolve to be safe
```

**Error handling:**

## Rate Limiting Strategy

**Understanding limits:**

- Kong gateway: 100 req/min per IP (development)
- Headers: `X-RateLimit-Limit-minute`, `X-RateLimit-Remaining-minute`
- HTTP 429 when exceeded

**Strategies:**

1. **Exponential Backoff**
2. **Check Headers**
3. **Batch Operations**

## Error Handling Patterns

### Common Error Scenarios

**401 Unauthorized:**
**404 Not Found:**
**429 Rate Limited:**
**500 Server Error:**

## See Also

- [API Examples](examples/README.md) - Working code samples
- [Rate Limiting](rate-limiting.md) - Rate limits and quotas
