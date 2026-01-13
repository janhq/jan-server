# response-api

`response-api` is the Jan Responses API microservice. It follows the OpenAI Responses contract, orchestrates multi-step tool calls through `mcp-tools`, and delegates language generation to `llm-api`.

## Key Features

- **OpenAI-Compatible Background Mode**: Async response generation with webhook notifications
- **Environment-driven config** with sensible defaults (see `internal/config`)
- **Structured Zerolog logging** plus optional OTEL tracing
- **PostgreSQL persistence** for responses, conversations, and tool executions (GORM)
- **PostgreSQL-backed Task Queue**: Reliable background job processing using `FOR UPDATE SKIP LOCKED`
- **Worker Pool**: Configurable concurrent workers for background tasks (default: 4)
- **Webhook Notifications**: HTTP POST callbacks on task completion/failure
- **JSON-RPC integration** with `services/mcp-tools` for tool discovery/calls
- **HTTP client** for `services/llm-api` chat completions
- **Gin HTTP server** exposing `/v1/responses` CRUD plus SSE streaming
- **Optional Keycloak/OIDC JWT** enforcement
- **Wire-ready DI** entrypoint, Dockerfile, Makefile, and example env file

## Quick start

```bash
# From repo root
make env-create            # populates .env from .env.template

cd services/response-api
go mod tidy
make run

# Smoke check
curl http://localhost:8082/healthz
curl -X POST http://localhost:8082/v1/responses \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","input":"ping"}'

```

Useful targets:

- `make wire` - regenerate DI after editing `cmd/server/wire.go`.
- `make swagger` - regenerate OpenAPI docs from annotations.
- `make test` - unit/integration test suite.

## Configuration

| Variable                   | Description                             | Default                                                                    |
| -------------------------- | --------------------------------------- | -------------------------------------------------------------------------- |
| `SERVICE_NAME`             | Logical service name                    | `response-api`                                                             |
| `HTTP_PORT`                | HTTP listen port                        | `8082`                                                                     |
| `DB_POSTGRESQL_WRITE_DSN`  | PostgreSQL DSN                          | `postgres://postgres:postgres@localhost:5432/response_api?sslmode=disable` |
| `LLM_API_URL`              | Base URL for `llm-api`                  | `http://localhost:8080`                                                    |
| `MCP_TOOLS_URL`            | Base URL for `mcp-tools`                | `http://localhost:8091`                                                    |
| `MAX_TOOL_EXECUTION_DEPTH` | Max recursive tool chain depth          | `8`                                                                        |
| `TOOL_EXECUTION_TIMEOUT`   | Per-tool call timeout                   | `45s`                                                                      |
| `BACKGROUND_WORKER_COUNT`  | Number of concurrent background workers | `4`                                                                        |
| `BACKGROUND_TASK_TIMEOUT`  | Max execution time per background task  | `600s`                                                                     |
| `BACKGROUND_POLL_INTERVAL` | How often workers poll for tasks        | `2s`                                                                       |
| `WEBHOOK_TIMEOUT`          | HTTP timeout for webhook delivery       | `10s`                                                                      |
| `WEBHOOK_MAX_RETRIES`      | Number of webhook retry attempts        | `3`                                                                        |
| `WEBHOOK_RETRY_DELAY`      | Delay between webhook retries           | `2s`                                                                       |
| `AUTH_ENABLED` + `AUTH_*`  | Toggle and configure OIDC validation    | disabled                                                                   |

See `.env.template` in the repo root for the full list including tracing/logging knobs.

### Recommended Settings

**Development:**

```bash
BACKGROUND_WORKER_COUNT=2
BACKGROUND_TASK_TIMEOUT=300s
```

**Production:**

```bash
BACKGROUND_WORKER_COUNT=8
BACKGROUND_TASK_TIMEOUT=600s
# Monitor queue depth and adjust worker count as needed
```

## Database

On startup the service runs migrations for:

- `responses`
- `conversations`
- `conversation_items`
- `tool_executions`

Each table uses JSONB columns for flexible payload storage. Point `DB_POSTGRESQL_WRITE_DSN` at your cluster before starting the service.

## Authentication

- Set `AUTH_ENABLED=true` to enforce Bearer tokens. Provide `AUTH_ISSUER`, `ACCOUNT`, and `AUTH_JWKS_URL`.
- With auth disabled the service treats callers as `guest` unless a `user` field is provided in the request body.

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

### API Usage

#### Creating a Background Response

**Request:**

```http
POST /v1/responses
Content-Type: application/json
Authorization: Bearer <token>

{
  "model": "gpt-4",
  "input": "Write a comprehensive analysis of...",
  "background": true,
  "store": true,
  "metadata": {
    "webhook_url": "https://example.com/webhooks/responses"
  }
}
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
  "metadata": {
    "webhook_url": "https://example.com/webhooks/responses"
  }
}
```

#### Polling for Status

**Request:**

```http
GET /v1/responses/resp_abc123
Authorization: Bearer <token>
```

**Response (In Progress):**

```json
{
  "id": "resp_abc123",
  "status": "in_progress",
  "started_at": 1705315805,
  ...
}
```

**Response (Completed):**

```json
{
  "id": "resp_abc123",
  "status": "completed",
  "output": "The comprehensive analysis...",
  "usage": {
    "prompt_tokens": 150,
    "completion_tokens": 500,
    "total_tokens": 650
  },
  "started_at": 1705315805,
  "completed_at": 1705316122,
  ...
}
```

#### Cancelling a Background Task

**Request:**

```http
POST /v1/responses/resp_abc123/cancel
Authorization: Bearer <token>
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

**Webhook Payload (Completed):**

```json
{
  "id": "resp_abc123",
  "event": "response.completed",
  "status": "completed",
  "output": "The response content...",
  "metadata": {...},
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
    "message": "LLM provider timeout"
  },
  "metadata": {...}
}
```

**Webhook Delivery:**

- **Method**: HTTP POST
- **Content-Type**: `application/json`
- **Headers**:
  - `User-Agent: jan-response-api/1.0`
  - `X-Jan-Event: response.completed` (or `response.failed`)
  - `X-Jan-Response-ID: resp_abc123`
- **Retries**: Up to 3 attempts with 2-second delays
- **Timeout**: 10 seconds per attempt
- **Non-blocking**: Webhook failures are logged but don't affect task completion

### Constraints

- **Background mode requires store=true**: Background tasks must be persisted to the database
- **API Key Authentication**: The user's API key (Bearer token or X-API-Key header) is stored securely and used for LLM API calls during background execution
- **Task Timeout**: Tasks exceeding `BACKGROUND_TASK_TIMEOUT` will be marked as failed
- **Queue Ordering**: Tasks are processed in FIFO order based on `queued_at` timestamp

## Testing

### Quick Test

```bash
# Create background task
curl -X POST http://localhost:8082/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "model": "gpt-4",
    "input": "Write a story",
    "background": true,
    "store": true,
    "metadata": {"webhook_url": "https://webhook.site/your-id"}
  }'

# Poll status (replace resp_xxx with actual ID)
curl http://localhost:8082/v1/responses/resp_xxx \
  -H "Authorization: Bearer <token>"
```

### Automated Tests

Comprehensive test suite available at `tests/automation/responses-background-webhook.json`:

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

# Run with verbose output
jan-cli api-test run tests/automation/responses-background-webhook.json \
  --timeout-request 60000 \
  --verbose

# Export results to JSON
jan-cli api-test run tests/automation/responses-background-webhook.json \
  --timeout-request 60000 \
  --reporters cli,json
```

### Test Webhook Server

Use webhook.site for testing:

```bash
# Get unique webhook URL
curl https://webhook.site/token

# Use in request
{
  "model": "gpt-4",
  "input": "Test input",
  "background": true,
  "store": true,
  "metadata": {
    "webhook_url": "https://webhook.site/<your-unique-id>"
  }
}

# View received webhooks at https://webhook.site/<your-unique-id>
```

**Local Webhook Server:**

```python
# webhook_server.py
from flask import Flask, request
import json

app = Flask(__name__)

@app.route('/webhook', methods=['POST'])
def webhook():
    data = request.get_json()
    print(json.dumps(data, indent=2))
    return '', 200

if __name__ == '__main__':
    app.run(port=9000)
```

```bash
# Run local webhook server
python webhook_server.py

# Use http://host.docker.internal:9000/webhook in Docker
```

### CI/CD Integration

```yaml
# .github/workflows/test-background-mode.yml
name: Background Mode Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Start services
        run: docker-compose up -d response-api

      - name: Wait for health check
        run: |
          timeout 60 bash -c 'until curl -f http://localhost:8082/healthz; do sleep 2; done'

      - name: Run tests
        run: |
          jan-cli api-test run tests/automation/responses-background-webhook.json \
            --timeout-request 60000 \
            --reporters cli

      - name: Publish test results
        uses: EnricoMi/publish-unit-test-result-action@v2
        if: always()
        with:
          files: test-results.xml
```

## Troubleshooting

### Background Tasks Stuck in Queued

**Issue**: Tasks remain in `queued` status and never get processed

**Solutions**:

1. Check worker logs: `docker logs <container> --tail 100`
2. Verify workers are running: Look for "worker started" messages
3. Check database connectivity: Workers need access to PostgreSQL
4. Verify `BACKGROUND_WORKER_COUNT > 0`

### Workers Not Picking Up Tasks

**Issue**: Workers running but not dequeuing tasks

**Solutions**:

1. Check for database locks: `SELECT * FROM pg_locks WHERE granted = false;`
2. Verify `BACKGROUND_POLL_INTERVAL` setting
3. Check worker logs for errors
4. Verify tasks have `background=true` and `store=true`

### Webhook Delivery Failures

**Issue**: Tasks complete but webhooks not received

**Solutions**:

1. Check webhook URL is accessible from Docker network
2. Use `http://host.docker.internal:<port>` for local development
3. Check response-api logs for webhook delivery errors
4. Verify webhook endpoint returns 2xx status code
5. Test webhook URL with curl: `curl -X POST <webhook_url> -d '{"test":"data"}'`

### Tasks Timing Out

**Issue**: Tasks marked as failed with timeout errors

**Solutions**:

1. Increase `BACKGROUND_TASK_TIMEOUT` (default: 600s)
2. Optimize LLM prompts to reduce processing time
3. Check LLM API availability and response times
4. Monitor tool execution times in logs

### High Queue Depth

**Issue**: Many tasks queued, slow processing

**Solutions**:

1. Increase `BACKGROUND_WORKER_COUNT`
2. Scale horizontally: Run multiple response-api instances
3. Monitor database performance
4. Check LLM API rate limits
5. Consider task prioritization (future enhancement)

## Agent Response System

The Response API includes an advanced agent response system for orchestrating multi-step AI workflows like deep research and slide generation.

### Plans and Tasks

When processing complex requests that require multiple steps (e.g., researching a topic, generating slides), the service creates an execution **Plan** consisting of:

- **Tasks**: High-level work units (search, analyze, generate, etc.)
- **Steps**: Individual operations within each task
- **Step Details**: Execution records with timing and retry information

### Plan API

| Endpoint                                   | Method | Description                                 |
| ------------------------------------------ | ------ | ------------------------------------------- |
| `/v1/responses/:response_id/plan`          | GET    | Get plan for a response                     |
| `/v1/responses/:response_id/plan/details`  | GET    | Get plan with all tasks and steps           |
| `/v1/responses/:response_id/plan/progress` | GET    | Get execution progress (percentage, counts) |
| `/v1/responses/:response_id/plan/cancel`   | POST   | Cancel plan execution                       |
| `/v1/responses/:response_id/plan/input`    | POST   | Submit user input for waiting plans         |
| `/v1/responses/:response_id/plan/tasks`    | GET    | List all tasks in a plan                    |

### Plan Statuses

- `pending`: Plan created, not yet started
- `planning`: Agent is creating execution plan
- `in_progress`: Tasks being executed
- `wait_for_user`: Waiting for user input
- `completed`: All tasks finished successfully
- `failed`: Execution failed (may be retryable)
- `cancelled`: Cancelled by user
- `expired`: Timed out waiting for user input

### Artifacts

Plans can produce **Artifacts** - structured output files like slide decks, research documents, or code files.

| Endpoint                                      | Method | Description                        |
| --------------------------------------------- | ------ | ---------------------------------- |
| `/v1/artifacts/:artifact_id`                  | GET    | Get artifact metadata              |
| `/v1/artifacts/:artifact_id/versions`         | GET    | Get all versions of an artifact    |
| `/v1/artifacts/:artifact_id/download`         | GET    | Download artifact content          |
| `/v1/artifacts/:artifact_id`                  | DELETE | Delete artifact                    |
| `/v1/responses/:response_id/artifacts`        | GET    | Get all artifacts for a response   |
| `/v1/responses/:response_id/artifacts/latest` | GET    | Get latest artifact for a response |

### Artifact Content Types

- `slides`: Presentation slides (JSON structure)
- `document`: Research documents, reports
- `image`: Generated images
- `code`: Code files
- `data`: Structured data (CSV, JSON)

### Example: Deep Research Flow

```http
POST /v1/responses
Content-Type: application/json

{
  "model": "gpt-4",
  "input": "Create a deep research report on renewable energy trends",
  "background": true,
  "store": true
}
```

The service will:

1. Create a Plan with the `research` agent type
2. Execute search tasks to gather information
3. Analyze and synthesize findings
4. Generate a research document artifact
5. Store the artifact and update the response

## Observability

### Prometheus Metrics

The service exposes metrics at `/metrics`:

**Plan Metrics:**

- `response_api_plans_total{agent_type, status}` - Total plans by type and status
- `response_api_plan_duration_seconds{agent_type}` - Plan execution duration
- `response_api_plan_steps_total{task_type, status}` - Steps by task type and status
- `response_api_plan_retries_total{step_action}` - Retry attempts by step action
- `response_api_plans_active{agent_type}` - Currently active plans
- `response_api_plans_waiting_for_user` - Plans waiting for user input

**Artifact Metrics:**

- `response_api_artifacts_total{content_type}` - Total artifacts by type
- `response_api_artifact_size_bytes{content_type}` - Artifact sizes
- `response_api_artifact_versions_total` - Version counts
- `response_api_artifact_downloads_total{content_type}` - Download counts

### OpenTelemetry Tracing

The service creates spans for:

- Plan lifecycle (`plan.create`, `plan.execute`, `plan.complete`)
- Task execution (`task.execute`)
- Step execution (`step.execute`)
- Artifact operations (`artifact.create`, `artifact.download`)
