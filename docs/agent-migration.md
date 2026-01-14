# Agent Migration Guide

This guide documents how to poll response execution via `response_id`, retrieve step-by-step progress, and control execution (cancel, pause, resume).

## Overview

When you create a response with `background: true`, the API returns immediately with a `response_id`. You can then poll for status, monitor detailed progress, and control execution using the endpoints below.

## Table of Contents

- [Response Lifecycle](#response-lifecycle)
- [Polling Endpoints](#polling-endpoints)
  - [Basic Polling](#basic-polling)
  - [Full Response with Plan](#full-response-with-plan)
  - [Plan Progress](#plan-progress)
  - [Plan Details with Steps](#plan-details-with-steps)
- [Execution Control](#execution-control)
  - [Cancel Execution](#cancel-execution)
  - [Submit User Input (Resume)](#submit-user-input-resume)
- [Complete Polling Examples](#complete-polling-examples)
- [Status Reference](#status-reference)

---

## Response Lifecycle

```
Client Request (background=true, store=true)
    ↓
Create Response (status=queued)
    ↓
Return Response Immediately (201 Created with response_id)
    ↓
Worker Dequeues Task
    ↓
Mark Processing (status=in_progress)
    ↓
Create Plan (status=planning → in_progress)
    ↓
Execute Tasks & Steps (track progress)
    ↓
   [If user input required: status=wait_for_user]
    ↓
Update Status (completed/failed)
    ↓
Send Webhook Notification (if configured)
```

---

## Polling Endpoints

### Basic Polling

Get current response status without plan details.

**Endpoint:** `GET /v1/responses/{response_id}`

**Request:**

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8000/responses/v1/responses/resp_abc123
```

**Response (Queued):**

```json
{
  "id": "resp_abc123",
  "object": "response",
  "status": "queued",
  "model": "jan-v2-30b",
  "input": "Research quantum computing trends",
  "queued_at": 1705315800,
  "created_at": 1705315800
}
```

**Response (In Progress):**

```json
{
  "id": "resp_abc123",
  "object": "response",
  "status": "in_progress",
  "model": "jan-v2-30b",
  "input": "Research quantum computing trends",
  "queued_at": 1705315800,
  "started_at": 1705315805,
  "created_at": 1705315800
}
```

**Response (Completed):**

```json
{
  "id": "resp_abc123",
  "object": "response",
  "status": "completed",
  "model": "jan-v2-30b",
  "input": "Research quantum computing trends",
  "output": "The comprehensive analysis of quantum computing trends...",
  "usage": {
    "prompt_tokens": 150,
    "completion_tokens": 500,
    "total_tokens": 650
  },
  "queued_at": 1705315800,
  "started_at": 1705315805,
  "completed_at": 1705316122,
  "created_at": 1705315800
}
```

---

### Full Response with Plan

Get response with complete plan details (tasks and steps) in a single call.

**Endpoint:** `GET /v1/responses/{response_id}/full`

**Request:**

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8000/responses/v1/responses/resp_abc123/full
```

**Response:**

```json
{
  "id": "resp_abc123",
  "object": "response",
  "status": "in_progress",
  "model": "jan-v2-30b",
  "input": "Research quantum computing trends",
  "queued_at": 1705315800,
  "started_at": 1705315805,
  "plan": {
    "id": "plan_xyz789",
    "object": "plan",
    "response_id": "resp_abc123",
    "status": "in_progress",
    "progress": 0.45,
    "agent_type": "research",
    "estimated_steps": 6,
    "completed_steps": 3,
    "current_task_id": "task_002",
    "created_at": 1705315805,
    "updated_at": 1705315850,
    "tasks": [
      {
        "id": "task_001",
        "object": "task",
        "plan_id": "plan_xyz789",
        "sequence": 1,
        "task_type": "search",
        "status": "completed",
        "title": "Search for quantum computing resources",
        "steps": [
          {
            "id": "step_001",
            "object": "step",
            "task_id": "task_001",
            "sequence": 1,
            "action": "web_search",
            "status": "completed",
            "retry_count": 0,
            "max_retries": 3,
            "duration_ms": 1250,
            "planned_params": {"query": "quantum computing trends 2025"},
            "output_data": {"results": [...]},
            "started_at": 1705315805,
            "completed_at": 1705315807
          }
        ],
        "created_at": 1705315805,
        "completed_at": 1705315810
      },
      {
        "id": "task_002",
        "object": "task",
        "plan_id": "plan_xyz789",
        "sequence": 2,
        "task_type": "analyze",
        "status": "in_progress",
        "title": "Analyze search results",
        "steps": [
          {
            "id": "step_002",
            "object": "step",
            "task_id": "task_002",
            "sequence": 1,
            "action": "llm_analyze",
            "status": "in_progress",
            "retry_count": 0,
            "max_retries": 3,
            "started_at": 1705315812
          }
        ],
        "created_at": 1705315810
      }
    ]
  }
}
```

---

### Plan Progress

Get lightweight progress information (useful for progress bars).

**Endpoint:** `GET /v1/responses/{response_id}/plan/progress`

**Request:**

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8000/responses/v1/responses/resp_abc123/plan/progress
```

**Response:**

```json
{
  "plan_id": "plan_xyz789",
  "status": "in_progress",
  "progress": 0.45,
  "estimated_steps": 6,
  "completed_steps": 3,
  "failed_steps": 0,
  "current_task": {
    "task_id": "task_002",
    "title": "Analyze search results",
    "status": "in_progress"
  }
}
```

---

### Plan Details with Steps

Get the full plan structure with all tasks and steps.

**Endpoint:** `GET /v1/responses/{response_id}/plan/details`

**Request:**

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8000/responses/v1/responses/resp_abc123/plan/details
```

**Response:**

```json
{
  "id": "plan_xyz789",
  "object": "plan",
  "response_id": "resp_abc123",
  "status": "in_progress",
  "progress": 0.45,
  "agent_type": "research",
  "estimated_steps": 6,
  "completed_steps": 3,
  "current_task_id": "task_002",
  "created_at": 1705315805,
  "updated_at": 1705315850,
  "tasks": [
    {
      "id": "task_001",
      "object": "task",
      "plan_id": "plan_xyz789",
      "sequence": 1,
      "task_type": "search",
      "status": "completed",
      "title": "Search for quantum computing resources",
      "steps": [
        {
          "id": "step_001",
          "object": "step",
          "task_id": "task_001",
          "sequence": 1,
          "action": "web_search",
          "status": "completed",
          "retry_count": 0,
          "max_retries": 3,
          "duration_ms": 1250,
          "planned_params": {"query": "quantum computing trends 2025"},
          "output_data": {"results": [...]},
          "started_at": 1705315805,
          "completed_at": 1705315807
        }
      ],
      "created_at": 1705315805,
      "completed_at": 1705315810
    }
  ]
}
```

---

## Execution Control

### Cancel Execution

Cancel a queued or in-progress response/plan.

#### Cancel Response

**Endpoint:** `POST /v1/responses/{response_id}/cancel`

**Request:**

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8000/responses/v1/responses/resp_abc123/cancel
```

**Response:**

```json
{
  "id": "resp_abc123",
  "object": "response",
  "status": "cancelled",
  "model": "jan-v2-30b",
  "cancelled_at": 1705315860,
  "created_at": 1705315800
}
```

#### Cancel Plan

Cancel the plan execution specifically (useful when plan is separate from response lifecycle).

**Endpoint:** `POST /v1/responses/{response_id}/plan/cancel`

**Request:**

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"reason": "User requested cancellation"}' \
  http://localhost:8000/responses/v1/responses/resp_abc123/plan/cancel
```

**Request Body (optional):**

```json
{
  "reason": "User requested cancellation"
}
```

**Response:**

```json
{
  "id": "plan_xyz789",
  "object": "plan",
  "response_id": "resp_abc123",
  "status": "cancelled",
  "progress": 0.45,
  "agent_type": "research",
  "estimated_steps": 6,
  "completed_steps": 3,
  "error": "User requested cancellation",
  "created_at": 1705315805,
  "updated_at": 1705315860
}
```

**Cancellation Behavior:**

| Current Status | Behavior |
|----------------|----------|
| `queued` | Immediately marked cancelled, prevents worker pickup |
| `in_progress` | Marked cancelled, task may complete normally (cooperative cancellation) |
| `wait_for_user` | Immediately cancelled |
| `completed` | No-op, returns current state |
| `failed` | No-op, returns current state |
| `cancelled` | No-op, returns current state |

---

### Submit User Input (Resume)

When a plan is in `wait_for_user` status, submit user input to resume execution.

**Endpoint:** `POST /v1/responses/{response_id}/plan/input`

**Request:**

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"selection": "option_1", "approval": true}' \
  http://localhost:8000/responses/v1/responses/resp_abc123/plan/input
```

**Request Body:**

```json
{
  "selection": "option_1",
  "approval": true,
  "message": "Additional context from user"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `selection` | string | User's selection from provided options |
| `approval` | boolean | Approval for pending action (true/false) |
| `message` | string | Additional free-form message from user |

**Response:**

```json
{
  "id": "plan_xyz789",
  "object": "plan",
  "response_id": "resp_abc123",
  "status": "in_progress",
  "progress": 0.45,
  "agent_type": "research",
  "estimated_steps": 6,
  "completed_steps": 3,
  "created_at": 1705315805,
  "updated_at": 1705315900
}
```

**Note:** The plan status changes from `wait_for_user` to `in_progress` after submitting input.

---

## Complete Polling Examples

### JavaScript: Polling with Progress Callback

```javascript
async function pollForCompletion(responseId, headers, options = {}) {
  const {
    maxWait = 300000,      // 5 minutes
    pollInterval = 2000,   // 2 seconds
    onProgress = null,     // Progress callback
    onStepComplete = null, // Step completion callback
  } = options;

  const startTime = Date.now();
  let lastProgress = 0;

  while (Date.now() - startTime < maxWait) {
    // Get full response with plan details
    const response = await fetch(
      `http://localhost:8000/responses/v1/responses/${responseId}/full`,
      { headers }
    );
    const result = await response.json();

    // Report progress
    if (onProgress && result.plan) {
      const { progress, completed_steps, estimated_steps, current_task } = result.plan;
      if (progress !== lastProgress) {
        onProgress({
          progress,
          completedSteps: completed_steps,
          estimatedSteps: estimated_steps,
          currentTask: current_task?.title,
        });
        lastProgress = progress;
      }
    }

    // Check terminal states
    if (['completed', 'failed', 'cancelled', 'expired'].includes(result.status)) {
      return result;
    }

    // Handle waiting for user input
    if (result.status === 'wait_for_user' || result.plan?.status === 'wait_for_user') {
      return result;
    }

    await new Promise(resolve => setTimeout(resolve, pollInterval));
  }

  throw new Error('Task did not complete in time');
}

// Usage
const result = await pollForCompletion('resp_abc123', headers, {
  onProgress: ({ progress, currentTask }) => {
    console.log(`Progress: ${(progress * 100).toFixed(0)}% - ${currentTask}`);
  },
});

if (result.plan?.status === 'wait_for_user') {
  console.log('User input required!');
  // Submit user input to resume
  await fetch(
    `http://localhost:8000/responses/v1/responses/${result.id}/plan/input`,
    {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ selection: 'approve' }),
    }
  );
}
```

### Bash: Simple Polling Loop

```bash
#!/bin/bash

TOKEN="your_access_token"
RESPONSE_ID="resp_abc123"
BASE_URL="http://localhost:8000/responses/v1/responses"

# Poll until completion
while true; do
  RESULT=$(curl -s -H "Authorization: Bearer $TOKEN" \
    "$BASE_URL/$RESPONSE_ID")

  STATUS=$(echo $RESULT | jq -r '.status')
  echo "Status: $STATUS"

  # Check for terminal states
  if [[ "$STATUS" == "completed" ]] || \
     [[ "$STATUS" == "failed" ]] || \
     [[ "$STATUS" == "cancelled" ]]; then
    echo "Final result:"
    echo $RESULT | jq
    break
  fi

  # Check for user input required
  if [[ "$STATUS" == "wait_for_user" ]]; then
    echo "User input required. Submitting approval..."
    curl -s -X POST \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d '{"approval": true}' \
      "$BASE_URL/$RESPONSE_ID/plan/input"
  fi

  sleep 2
done
```

### Bash: Step-by-Step Progress Monitoring

```bash
#!/bin/bash

TOKEN="your_access_token"
RESPONSE_ID="resp_abc123"
BASE_URL="http://localhost:8000/responses/v1/responses"

echo "Monitoring response: $RESPONSE_ID"
echo "================================"

LAST_STEP=""

while true; do
  # Get full response with plan
  FULL=$(curl -s -H "Authorization: Bearer $TOKEN" \
    "$BASE_URL/$RESPONSE_ID/full")

  STATUS=$(echo $FULL | jq -r '.status')
  PLAN_STATUS=$(echo $FULL | jq -r '.plan.status // empty')
  PROGRESS=$(echo $FULL | jq -r '.plan.progress // 0')
  COMPLETED=$(echo $FULL | jq -r '.plan.completed_steps // 0')
  TOTAL=$(echo $FULL | jq -r '.plan.estimated_steps // 0')
  CURRENT_TASK=$(echo $FULL | jq -r '.plan.tasks[] | select(.status == "in_progress") | .title' 2>/dev/null | head -1)
  CURRENT_STEP=$(echo $FULL | jq -r '.plan.tasks[].steps[] | select(.status == "in_progress") | .action' 2>/dev/null | head -1)

  # Show progress
  PERCENT=$(echo "$PROGRESS * 100" | bc | cut -d. -f1)
  echo -ne "\rProgress: [$COMPLETED/$TOTAL] ${PERCENT}% - Task: ${CURRENT_TASK:-N/A} - Step: ${CURRENT_STEP:-N/A}    "

  # Log step completions
  COMPLETED_STEPS=$(echo $FULL | jq -r '.plan.tasks[].steps[] | select(.status == "completed") | .action')
  for STEP in $COMPLETED_STEPS; do
    if [[ ! "$LAST_STEP" == *"$STEP"* ]]; then
      echo -e "\n✓ Completed: $STEP"
      LAST_STEP="$LAST_STEP $STEP"
    fi
  done

  # Check terminal states
  if [[ "$STATUS" == "completed" ]]; then
    echo -e "\n\n✓ Response completed!"
    echo "Output:"
    echo $FULL | jq -r '.output'
    break
  elif [[ "$STATUS" == "failed" ]]; then
    echo -e "\n\n✗ Response failed!"
    echo $FULL | jq '.plan.tasks[].steps[] | select(.status == "failed")'
    break
  elif [[ "$STATUS" == "cancelled" ]]; then
    echo -e "\n\n⚠ Response cancelled"
    break
  fi

  sleep 1
done
```

### Python: Async Polling with WebSocket-like Updates

```python
import asyncio
import aiohttp
import json

async def poll_response(response_id: str, token: str, callback=None):
    """Poll response with real-time progress updates."""
    base_url = "http://localhost:8000/responses/v1/responses"
    headers = {"Authorization": f"Bearer {token}"}

    async with aiohttp.ClientSession() as session:
        while True:
            # Get full response with plan
            async with session.get(
                f"{base_url}/{response_id}/full",
                headers=headers
            ) as resp:
                data = await resp.json()

            status = data.get("status")
            plan = data.get("plan", {})

            # Call progress callback
            if callback:
                await callback({
                    "status": status,
                    "plan_status": plan.get("status"),
                    "progress": plan.get("progress", 0),
                    "completed_steps": plan.get("completed_steps", 0),
                    "estimated_steps": plan.get("estimated_steps", 0),
                    "current_task": next(
                        (t["title"] for t in plan.get("tasks", [])
                         if t["status"] == "in_progress"),
                        None
                    ),
                })

            # Handle terminal states
            if status in ["completed", "failed", "cancelled", "expired"]:
                return data

            # Handle user input required
            if status == "wait_for_user":
                return data

            await asyncio.sleep(2)


async def submit_user_input(response_id: str, token: str, selection: str = None, approval: bool = None):
    """Submit user input to resume a waiting plan."""
    base_url = "http://localhost:8000/responses/v1/responses"
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }

    body = {}
    if selection:
        body["selection"] = selection
    if approval is not None:
        body["approval"] = approval

    async with aiohttp.ClientSession() as session:
        async with session.post(
            f"{base_url}/{response_id}/plan/input",
            headers=headers,
            json=body
        ) as resp:
            return await resp.json()


async def cancel_response(response_id: str, token: str, reason: str = None):
    """Cancel a response execution."""
    base_url = "http://localhost:8000/responses/v1/responses"
    headers = {"Authorization": f"Bearer {token}"}

    async with aiohttp.ClientSession() as session:
        async with session.post(
            f"{base_url}/{response_id}/cancel",
            headers=headers
        ) as resp:
            return await resp.json()


# Usage example
async def main():
    token = "your_access_token"
    response_id = "resp_abc123"

    async def on_progress(update):
        progress = update["progress"] * 100
        print(f"[{progress:.0f}%] {update['current_task'] or 'Processing...'}")

    result = await poll_response(response_id, token, callback=on_progress)

    if result["status"] == "wait_for_user":
        print("User input required!")
        result = await submit_user_input(response_id, token, approval=True)
        # Continue polling after input
        result = await poll_response(response_id, token, callback=on_progress)

    print(f"Final status: {result['status']}")
    if result.get("output"):
        print(f"Output: {result['output'][:200]}...")


asyncio.run(main())
```

---

## Status Reference

### Response Status

| Status | Description | Terminal |
|--------|-------------|----------|
| `queued` | Waiting in queue for worker | No |
| `in_progress` | Being processed by worker | No |
| `completed` | Successfully finished | Yes |
| `failed` | Execution failed | Yes |
| `cancelled` | Cancelled by user/system | Yes |

### Plan Status

| Status | Description | Terminal |
|--------|-------------|----------|
| `pending` | Plan created, not started | No |
| `planning` | Agent creating execution plan | No |
| `in_progress` | Tasks being executed | No |
| `wait_for_user` | Waiting for user input | No |
| `completed` | All tasks finished successfully | Yes |
| `failed` | Execution failed (may be retryable) | Yes |
| `cancelled` | Cancelled by user | Yes |
| `expired` | Timed out waiting for user | Yes |
| `skipped` | Step was skipped | Yes |

### Valid Status Transitions

```
pending → planning, in_progress, failed, cancelled
planning → in_progress, failed, cancelled
in_progress → wait_for_user, completed, failed, cancelled
wait_for_user → in_progress, expired, cancelled
failed → in_progress (retry)
```

---

## API Reference Summary

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/responses/{response_id}` | GET | Get response status |
| `/v1/responses/{response_id}/full` | GET | Get response with plan details |
| `/v1/responses/{response_id}/plan` | GET | Get plan metadata |
| `/v1/responses/{response_id}/plan/details` | GET | Get plan with all tasks and steps |
| `/v1/responses/{response_id}/plan/progress` | GET | Get lightweight progress info |
| `/v1/responses/{response_id}/plan/tasks` | GET | List all tasks in a plan |
| `/v1/responses/{response_id}/cancel` | POST | Cancel response |
| `/v1/responses/{response_id}/plan/cancel` | POST | Cancel plan execution |
| `/v1/responses/{response_id}/plan/input` | POST | Submit user input to resume |

---

## Best Practices

1. **Use `/full` endpoint for UI updates**: Single call returns everything needed for rich progress display.

2. **Use `/plan/progress` for progress bars**: Lightweight endpoint, minimal data transfer.

3. **Handle `wait_for_user` status**: Always check for this status and prompt user or auto-approve.

4. **Implement exponential backoff**: Don't hammer the server with rapid polls.

5. **Set reasonable timeouts**: Background tasks can take minutes; set appropriate max wait times.

6. **Check terminal states first**: Exit polling loop immediately when terminal state detected.

7. **Use webhooks for production**: Instead of polling, configure `webhook_url` in metadata for push notifications.
