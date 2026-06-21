# Response API Documentation

The Response API executes tools and generates AI responses for complex tasks.

## Quick Start

Examples: [API examples index](../examples/README.md) includes cURL/SDK snippets for orchestration flows.

### URLs

- **Direct access**: http://localhost:8082
- **Through gateway**: http://localhost:8000/responses (Kong prefixes `/responses`)
- **Inside Docker**: http://response-api:8082

### Authentication

All endpoints require authentication through the Kong gateway.

**For complete authentication documentation, see [Authentication Guide](../README.md#authentication)**

**Quick example:**

```bash
# Get guest token
TOKEN=$(curl -s -X POST http://localhost:8000/llm/auth/guest-login | jq -r '.access_token')

# Use in requests
curl -H "Authorization: Bearer $TOKEN" \
 http://localhost:8000/responses/v1/responses
```

## What You Can Do

- **Run tools automatically** - AI decides which tools to use
- **Chain tools together** - Use output from one tool as input to another (depth capped by `RESPONSE_MAX_TOOL_DEPTH`, default 50)
- **Get final answers** - LLM generates natural language response from tool results
- **Track execution** - See which tools ran and how long they took

**Not sure if you need Response API?** See [Decision Guide: LLM API vs Response API](../decision-guides.md#llm-api-vs-response-api) to choose the right approach.

## Service Ports & Configuration

| Component                 | Port | Key Environment Variables                            |
| ------------------------- | ---- | ---------------------------------------------------- |
| **HTTP Server**           | 8082 | `RESPONSE_API_PORT`                                  |
| **Database (PostgreSQL)** | 5432 | `DB_POSTGRESQL_WRITE_DSN`, `DB_POSTGRESQL_READ1_DSN` |
| **LLM API upstream**      | 8080 | `RESPONSE_LLM_API_URL`                               |
| **MCP Tools upstream**    | 8091 | `RESPONSE_MCP_TOOLS_URL`                             |

### Required Environment Variables

```bash
RESPONSE_API_PORT=8082
DB_POSTGRESQL_WRITE_DSN=postgres://response_api:password@api-db:5432/response_api?sslmode=disable
# Optional read replica
DB_POSTGRESQL_READ1_DSN=postgres://response_ro:password@api-db-ro:5432/response_api?sslmode=disable

# Upstream services
RESPONSE_LLM_API_URL=http://llm-api:8080
RESPONSE_MCP_TOOLS_URL=http://mcp-tools:8091

# Tool execution limits
RESPONSE_MAX_TOOL_DEPTH=50      # Max sequential tool calls per response (default 50)
TOOL_EXECUTION_TIMEOUT=300s     # Per-tool-call timeout (default 300s)
```

### Optional Configuration

```bash
RESPONSE_LOG_LEVEL=info
ENABLE_TRACING=false
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317

# Auth (when fronted by Kong or called directly with JWT)
AUTH_ENABLED=true
AUTH_ISSUER=http://localhost:8085/realms/jan
ACCOUNT=account
AUTH_JWKS_URL=http://keycloak:8085/realms/jan/protocol/openid-connect/certs
```

## Main Endpoints

### Create Response (Multi-Step Orchestration)

**POST** `/v1/responses`

Create a new response with automatic tool orchestration.

```bash
curl -X POST http://localhost:8000/responses/v1/responses \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -d '{
 "model": "jan-v1-4b",
 "input": "Search for the latest AI news and summarize the top 3 results",
 "temperature": 0.3,
 "tool_choice": {"type": "auto"},
 "stream": false
 }'
```

**Request Body (subset of `CreateResponseRequest`):**

- `model` _(required)_ - Model identifier understood by the LLM API/catalog
- `input` _(required)_ - User prompt (string or structured object)
- `system_prompt` _(optional)_ - Instruction prepended before each run
- `temperature`, `max_tokens` _(optional)_ - Generation controls
- `tools` _(optional)_ - Override available tools (OpenAI-compatible format)
- `tool_choice` _(optional)_ - `{ "type": "auto" | "none" | "required", "function": {"name": "tool"} }`
- `stream` _(optional)_ - `true` to receive SSE events
- `conversation` _(optional)_ - Attach to an existing conversation ID
- `previous_response_id` _(optional)_ - Continue from a prior response
- `metadata`, `user` _(optional)_ - Free-form payload that is persisted with the response

**Response:**

```json
{
  "id": "resp_01hqr8v9k2x3f4g5h6j7k8m9n0",
  "model": "jan-v1-4b",
  "input": "Search for the latest AI news and summarize the top 3 results",
  "output": "Here are the latest AI news items...",
  "tool_executions": [
    {
      "id": "toolexec_123",
      "tool_name": "google_search",
      "input": { "q": "latest AI news", "num": 3 },
      "output": "...",
      "execution_time_ms": 250,
      "depth": 0
    }
  ],
  "execution_metadata": {
    "max_depth": 50,
    "actual_depth": 1,
    "total_duration_ms": 2500,
    "status": "completed"
  },
  "created_at": "2025-11-10T10:30:00Z",
  "updated_at": "2025-11-10T10:30:02.500Z"
}
```

### Streaming Responses

Enable `stream: true` to receive incremental events (`text/event-stream`), matching the SSE observer in `services/response-api/internal/interfaces/httpserver/handlers/response_handler.go`.

```bash
curl -N http://localhost:8000/responses/v1/responses \
 -H "Authorization: Bearer <token>" \
 -H "Content-Type: application/json" \
 -H "Accept: text/event-stream" \
 -d '{
 "model": "jan-v1-4b",
 "input": "Search for the latest AI news and summarize the top 3 results",
 "stream": true
 }'
```

The stream emits events such as `response.created`, `response.tool_call`, `response.output_text.delta`, and `response.completed`.

### Get Response

**GET** `/v1/responses/{response_id}`

Retrieve a specific response and its execution metadata.

```bash
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/responses/v1/responses/resp_01hqr8v9k2x3f4g5h6j7k8m9n0
```

### Delete Response

**DELETE** `/v1/responses/{response_id}`

```bash
curl -X DELETE -H "Authorization: Bearer <token>" \
 http://localhost:8000/responses/v1/responses/resp_01hqr8v9k2x3f4g5h6j7k8m9n0
```

### Cancel In-Flight Response

**POST** `/v1/responses/{response_id}/cancel`

```bash
curl -X POST -H "Authorization: Bearer <token>" \
 http://localhost:8000/responses/v1/responses/resp_01hqr8v9k2x3f4g5h6j7k8m9n0/cancel
```

### List Input Items (Conversation Replay)

**GET** `/v1/responses/{response_id}/input_items`

Returns the normalized conversation items that were sent to the LLM (useful for replaying the request or for debugging tool runs).

```bash
curl -H "Authorization: Bearer <token>" \
 http://localhost:8000/responses/v1/responses/resp_01hqr8v9k2x3f4g5h6j7k8m9n0/input_items
```

> The Response API does **not** expose a list endpoint for all responses. (Artifacts and agents
> have their own list endpoints — see below.)

### Other Response Routes

| Service path                                | Method | Purpose                                |
| ------------------------------------------- | ------ | -------------------------------------- |
| `/v1/responses/{id}/full`                   | GET    | Response with full execution detail    |
| `/v1/responses/{id}/retry`                  | POST   | Retry a failed response                |
| `/v1/responses/{id}/plan`                   | GET    | Execution plan for the response        |
| `/v1/responses/{id}/plan/details`           | GET    | Plan with step details                 |
| `/v1/responses/{id}/plan/progress`          | GET    | Plan progress                          |
| `/v1/responses/{id}/plan/cancel`            | POST   | Cancel plan execution                  |
| `/v1/responses/{id}/plan/input`             | POST   | Submit user input to a waiting plan    |
| `/v1/responses/{id}/plan/tasks`             | GET    | List plan tasks                        |

### Artifacts and Agents (Kong-routed at `/v1/...`)

Unlike `/responses`, the artifact and agent routes are exposed by Kong **without** path
stripping, so they are reached at `http://localhost:8000/v1/artifacts` and
`http://localhost:8000/v1/agents` directly.

| Path                                          | Method | Purpose                              |
| --------------------------------------------- | ------ | ------------------------------------ |
| `/v1/artifacts`                               | GET    | List artifacts for the current user  |
| `/v1/artifacts/{artifact_id}`                 | GET    | Get an artifact                      |
| `/v1/artifacts/{artifact_id}/versions`        | GET    | List artifact versions               |
| `/v1/artifacts/{artifact_id}/download`        | GET    | Download artifact content            |
| `/v1/artifacts/{artifact_id}`                 | DELETE | Delete an artifact                   |
| `/v1/responses/{id}/artifacts`                | GET    | Artifacts produced by a response     |
| `/v1/responses/{id}/artifacts/latest`         | GET    | Latest artifact for a response       |
| `/v1/agents`                                  | GET    | List available agents                |
| `/v1/agents/capabilities`                     | GET    | List agent capabilities              |
| `/v1/agents/{type}`                           | GET    | Get an agent by type                 |
| `/v1/agents/{type}/schema`                    | GET    | Get an agent input schema            |

### Health Check

**GET** `/healthz`

```bash
# Gateway
curl http://localhost:8000/responses/healthz

# Direct
curl http://localhost:8082/healthz
```

## Tool Execution Flow

### 1. Request Processing

- Validate input parameters
- Check tool availability via MCP Tools

### 2. Tool Discovery

- Query MCP Tools for available tools
- Build tool call graph

### 3. Iterative Execution

- Execute tools in sequence/parallel as needed
- Apply depth limit (`RESPONSE_MAX_TOOL_DEPTH`, default 50)
- Apply timeout per tool (`TOOL_EXECUTION_TIMEOUT`, default 300s)

### 4. LLM Delegation

- Pass tool results to LLM API
- Generate final response using context

### 5. Result Storage

- Store execution trace in PostgreSQL
- Record tool outputs and timing
- Return complete execution metadata

## Tool Execution Parameters

### Max Tool Execution Depth (`RESPONSE_MAX_TOOL_DEPTH`)

Limits how many sequential tool calls a single response may make:

- **Default**: 50 (any value `<= 0` falls back to 50)
- **Meaning**: Maximum number of chained tool calls
- **Example**: search -> extract -> summarize = depth 3

### Tool Execution Timeout (`TOOL_EXECUTION_TIMEOUT`)

Per-tool-call timeout:

- **Value**: Duration string (default: `300s`)
- **Example**: `"30s"`, `"1m"`, `"500ms"`
- **Behavior**: Cancels the tool call if it exceeds the timeout

## Error Handling

| Status | Error                  | Cause                        |
| ------ | ---------------------- | ---------------------------- |
| 400    | Invalid request        | Missing/malformed parameters |
| 404    | Response not found     | Invalid response ID          |
| 408    | Tool execution timeout | Tool exceeded timeout        |
| 500    | Execution error        | Tool or LLM error            |

Example error:

```json
{
  "error": {
    "message": "Tool execution exceeded maximum depth",
    "type": "execution_error",
    "code": "max_depth_exceeded"
  }
}
```

## Related Services

- **LLM API** (Port 8080) - Generates final response
- **MCP Tools** (Port 8091) - Tool execution and discovery
- **Kong Gateway** (Port 8000) - API routing
- **PostgreSQL** - Execution storage

## Configuration Examples

### Quick Response (Single Tool)

```bash
RESPONSE_MAX_TOOL_DEPTH=1 # Single tool call only
TOOL_EXECUTION_TIMEOUT=15s # Short timeout
```

### Complex Workflows (Deep Chains)

```bash
RESPONSE_MAX_TOOL_DEPTH=50 # Allow long chains (default)
TOOL_EXECUTION_TIMEOUT=300s # Long timeout for complex work
```

## See Also

- [MCP Tools API](../mcp-tools/)
- [LLM API](../llm-api/)
- [Architecture Overview](../../architecture/)
- [Development Guide](../../guides/development.md)

## Authentication

Requests routed through Kong (`http://localhost:8000/responses/...`) must include either:

- `Authorization: Bearer <token>` (Keycloak JWT - guest tokens work for local testing)
- `X-API-Key: sk_*` (custom plugin managed by Kong)

When `AUTH_ENABLED=true` the service also validates JWTs on port 8082. Use the gateway path whenever possible for rate limiting and centralized logging.

## Background Mode

The Response API supports OpenAI-compatible background mode for asynchronous response generation. This allows clients to submit long-running requests without holding open HTTP connections.

### Architecture

**Components:**

1. **PostgreSQL-backed Queue**: Uses the `responses` table with `SELECT FOR UPDATE SKIP LOCKED` for reliable task distribution
2. **Worker Pool**: Fixed-size pool of background workers (default: 4) that poll for queued tasks
3. **Webhook Notifications**: HTTP POST callbacks when tasks complete or fail
4. **Graceful Cancellation**: Queued or in-progress tasks can be cancelled

**Task Lifecycle:**

```
Client Request (background=true, store=true)
    ↓
Create Response (status=queued, queued_at=now)
    ↓
Return Response Immediately (201 Created)
    ↓
Worker Dequeues Task
    ↓
Mark Processing (status=in_progress, started_at=now)
    ↓
Execute LLM Orchestration with Tool Calls
    ↓
Update Status (completed/failed, completed_at=now)
    ↓
Send Webhook Notification (async, non-blocking)
```

### Configuration

Add these environment variables to enable background mode:

```bash
# Worker Pool
BACKGROUND_WORKER_COUNT=4        # Number of concurrent workers
BACKGROUND_POLL_INTERVAL=2s      # How often workers check for queued tasks
BACKGROUND_TASK_TIMEOUT=600s     # Max execution time per task (10 minutes)

# Webhook Delivery
WEBHOOK_MAX_RETRIES=3            # Retry attempts for failed webhooks
WEBHOOK_RETRY_DELAY=2s           # Delay between retry attempts
WEBHOOK_TIMEOUT=10s              # HTTP timeout per webhook attempt
WEBHOOK_USER_AGENT=jan-response-api/1.0
```

**Recommended Settings:**

| Environment | Workers | Poll Interval | Task Timeout | Use Case                       |
| ----------- | ------- | ------------- | ------------ | ------------------------------ |
| Development | 2-4     | 2s            | 600s (10m)   | Local testing, fast iteration  |
| Production  | 8-16    | 5s            | 1200s (20m)  | High throughput, complex tasks |
| High-load   | 16-32   | 3s            | 900s (15m)   | Many concurrent tasks          |

### API Usage

#### Creating a Background Response

Add `"background": true` and `"store": true` to any response request:

**Request:**

```bash
curl -X POST http://localhost:8000/responses/v1/responses \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b",
    "input": "Write a comprehensive analysis of quantum computing trends",
    "background": true,
    "store": true,
    "metadata": {
      "webhook_url": "https://example.com/webhooks/responses",
      "user_id": "user_123"
    }
  }'
```

**Response (201 Created):**

```json
{
  "id": "resp_abc123",
  "object": "response",
  "status": "queued",
  "background": true,
  "store": true,
  "queued_at": 1705315800,
  "created_at": 1705315800,
  "model": "jan-v1-4b",
  "input": "Write a comprehensive analysis...",
  "metadata": {
    "webhook_url": "https://example.com/webhooks/responses",
    "user_id": "user_123"
  }
}
```

#### Polling for Status

Use the standard GET endpoint to check task status:

**Request:**

```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8000/responses/v1/responses/resp_abc123
```

**Response (Queued):**

```json
{
  "id": "resp_abc123",
  "status": "queued",
  "queued_at": 1705315800,
  ...
}
```

**Response (In Progress):**

```json
{
  "id": "resp_abc123",
  "status": "in_progress",
  "queued_at": 1705315800,
  "started_at": 1705315805,
  ...
}
```

**Response (Completed):**

```json
{
  "id": "resp_abc123",
  "status": "completed",
  "output": "The comprehensive analysis of quantum computing trends...",
  "usage": {
    "prompt_tokens": 150,
    "completion_tokens": 500,
    "total_tokens": 650
  },
  "queued_at": 1705315800,
  "started_at": 1705315805,
  "completed_at": 1705316122,
  "tool_executions": [...],
  ...
}
```

#### Cancelling a Background Task

Use the cancel endpoint:

**Request:**

```bash
curl -X POST -H "Authorization: Bearer <token>" \
  http://localhost:8000/responses/v1/responses/resp_abc123/cancel
```

**Response:**

```json
{
  "id": "resp_abc123",
  "status": "cancelled",
  "cancelled_at": 1705315860,
  ...
}
```

**Cancellation Behavior:**

- If status is `queued`: Immediately marks cancelled, prevents worker pickup
- If status is `in_progress`: Marks cancelled, but task may complete normally (cooperative cancellation)
- If status is `completed` or `failed`: No-op, returns current state

### Webhook Notifications

When a background task completes or fails, the Response API sends an HTTP POST to the webhook URL specified in `metadata.webhook_url`.

**Webhook Payload (Completed):**

```json
{
  "id": "resp_abc123",
  "event": "response.completed",
  "status": "completed",
  "output": "The response content...",
  "usage": {
    "prompt_tokens": 150,
    "completion_tokens": 500,
    "total_tokens": 650
  },
  "tool_executions": [...],
  "metadata": {
    "webhook_url": "https://example.com/webhooks/responses",
    "user_id": "user_123"
  },
  "queued_at": 1705315800,
  "started_at": 1705315805,
  "completed_at": 1705316122
}
```

**Webhook Payload (Failed):**

```json
{
  "id": "resp_abc123",
  "event": "response.failed",
  "status": "failed",
  "error": {
    "code": "execution_failed",
    "message": "LLM provider timeout after 600s"
  },
  "metadata": {
    "webhook_url": "https://example.com/webhooks/responses",
    "user_id": "user_123"
  },
  "queued_at": 1705315800,
  "started_at": 1705315805,
  "completed_at": 1705316405
}
```

**Webhook HTTP Headers:**

- `Content-Type: application/json`
- `User-Agent: jan-response-api/1.0`
- `X-Jan-Event: response.completed` (or `response.failed`)
- `X-Jan-Response-ID: resp_abc123`

**Webhook Delivery:**

- **Method**: HTTP POST
- **Retries**: Up to 3 attempts with 2-second delays
- **Timeout**: 10 seconds per attempt
- **Non-blocking**: Webhook failures are logged but don't affect task completion
- **Status Codes**: 2xx considered success, all others trigger retry

### Background Mode Constraints

- **Requires store=true**: Background tasks must be persisted to the database
- **API Key Storage**: The user's API key (Bearer token or X-API-Key header) is stored securely and used for LLM API calls during background execution
- **Task Timeout**: Tasks exceeding `BACKGROUND_TASK_TIMEOUT` will be marked as failed
- **Queue Ordering**: Tasks are processed in FIFO order based on `queued_at` timestamp
- **No Streaming**: Background mode is incompatible with `stream: true`
- **Worker Restart**: In-progress tasks may fail if workers restart (status will show `failed`)

### Status Transitions

```
queued → in_progress → completed
queued → in_progress → failed
queued → cancelled
in_progress → cancelled (cooperative)
```

**Valid Status Values:**

- `queued` - Task waiting for worker
- `in_progress` - Worker currently executing
- `completed` - Successfully finished
- `failed` - Error during execution
- `cancelled` - Cancelled by user

### Testing Background Mode

**Quick Test:**

```bash
# 1. Create background task
RESP_ID=$(curl -s -X POST http://localhost:8000/responses/v1/responses \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "jan-v1-4b",
    "input": "Write a haiku about coding",
    "background": true,
    "store": true,
    "metadata": {"webhook_url": "https://webhook.site/your-id"}
  }' | jq -r '.id')

echo "Created task: $RESP_ID"

# 2. Poll until complete
while true; do
  STATUS=$(curl -s -H "Authorization: Bearer <token>" \
    "http://localhost:8000/responses/v1/responses/$RESP_ID" \
    | jq -r '.status')
  echo "Status: $STATUS"
  [[ "$STATUS" == "completed" ]] || [[ "$STATUS" == "failed" ]] && break
  sleep 2
done

# 3. Get final result
curl -s -H "Authorization: Bearer <token>" \
  "http://localhost:8000/responses/v1/responses/$RESP_ID" | jq
```

**Webhook Testing with webhook.site:**

1. Go to https://webhook.site/ to get a unique URL
2. Use that URL as `metadata.webhook_url` in your request
3. View received webhooks in the browser

**Local Webhook Server:**

Create a simple HTTP server that accepts POST requests to `/webhook` and logs the received event data.

```bash
# Run webhook server
python webhook_server.py

# Use http://host.docker.internal:9000/webhook in requests
```

### Automated Testing

Comprehensive test suite at `tests/automation/responses-background-webhook.json`:

**Test Suites:**

1. Setup & Authentication
2. Basic Background Mode
3. Background with Webhooks
4. Background with Tool Calling
5. Cancellation
6. Conversation Continuity
7. Error Handling
8. Complex Scenarios
9. Monitoring & Observability
10. Long-Running Research Task

**Running Tests:**

```bash
# Run all tests
jan-cli api-test run tests/automation/responses-background-webhook.json \
  --timeout-request 60000

# Export results
jan-cli api-test run tests/automation/responses-background-webhook.json \
  --timeout-request 60000 \
  --reporters cli,json
```

### Troubleshooting

#### Tasks Stuck in Queued

**Symptoms**: Tasks remain in `queued` status indefinitely

**Solutions**:

1. Check worker logs: `docker logs <response-api-container> --tail 100`
2. Verify workers started: Look for "worker X started" messages
3. Check `BACKGROUND_WORKER_COUNT > 0`
4. Verify database connectivity
5. Check for database locks: `SELECT * FROM pg_locks WHERE granted = false;`

#### Workers Not Processing Tasks

**Symptoms**: Workers running but queue depth not decreasing

**Solutions**:

1. Verify `BACKGROUND_POLL_INTERVAL` setting
2. Check worker logs for errors
3. Ensure tasks have `background=true` and `store=true`
4. Check LLM API availability: `curl http://llm-api:8080/healthz`

#### Webhook Delivery Failures

**Symptoms**: Tasks complete but webhooks not received

**Solutions**:

1. Test webhook URL: `curl -X POST <webhook_url> -d '{"test":"data"}'`
2. Use `http://host.docker.internal:<port>` for local development
3. Check response-api logs for webhook errors
4. Verify webhook endpoint returns 2xx status
5. Check firewall/network policies

#### Tasks Timing Out

**Symptoms**: Tasks marked as `failed` with timeout errors

**Solutions**:

1. Increase `BACKGROUND_TASK_TIMEOUT` (default: 600s)
2. Optimize prompts to reduce processing time
3. Check LLM API response times
4. Monitor tool execution duration in logs
5. Consider breaking into smaller tasks

#### High Queue Depth

**Symptoms**: Many queued tasks, slow processing

**Solutions**:

1. Increase `BACKGROUND_WORKER_COUNT`
2. Scale horizontally: Run multiple response-api instances
3. Monitor database performance
4. Check LLM API rate limits
5. Optimize tool execution times
