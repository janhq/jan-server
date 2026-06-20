# API Endpoint Matrix

Complete reference matrix of all API endpoints across Jan Server services.

## Overview

This document provides a comprehensive matrix of all available API endpoints, their HTTP methods, authentication requirements, and descriptions. Use this to understand complete API coverage across all services.

## Legend

| Symbol | Meaning                               |
| ------ | ------------------------------------- |
| ✅     | Fully Implemented & Documented        |
| ⚠️     | Implemented but Limited Documentation |
| 🔒     | Requires Authentication               |
| ❌     | No Authentication Required            |

## LLM API Endpoints

**Base URL:** `http://localhost:8080` (or `http://localhost:8000/v1` via Kong)

### Authentication

| Endpoint              | Method | Auth | Status | Description                              |
| --------------------- | ------ | ---- | ------ | ---------------------------------------- |
| `/auth/guest-login`   | POST   | ❌   | ✅     | Request guest token without credentials  |
| `/auth/register`      | POST   | ❌   | ✅     | Register a new user                      |
| `/auth/login`         | GET    | ❌   | ✅     | Initiate Keycloak OAuth login flow       |
| `/auth/callback`      | GET    | ❌   | ✅     | OAuth callback handler                   |
| `/auth/logout`        | GET/POST | 🔒 | ✅     | Logout current session                   |
| `/auth/refresh-token` | POST   | ❌   | ✅     | Refresh access token                     |
| `/auth/validate`      | POST   | ❌   | ✅     | Validate a Keycloak access token         |
| `/auth/revoke`        | POST   | ❌   | ✅     | Revoke a Keycloak refresh token          |
| `/auth/upgrade`       | POST   | 🔒   | ✅     | Upgrade guest account to permanent       |
| `/auth/me`            | GET    | 🔒   | ✅     | Get current user profile                 |
| `/auth/api-keys`      | GET    | 🔒   | ✅     | List user's API keys                     |
| `/auth/api-keys`      | POST   | 🔒   | ✅     | Create new API key                       |
| `/auth/api-keys/{id}` | DELETE | 🔒   | ✅     | Revoke API key                           |
| `/auth/system-key`    | POST   | 🔒   | ✅     | Get or create the internal system key    |

### Chat Completions

| Endpoint               | Method | Auth | Status | Description                                         |
| ---------------------- | ------ | ---- | ------ | --------------------------------------------------- |
| `/v1/chat/completions` | POST   | 🔒   | ✅     | Send message, get AI response (streaming supported) |

### Conversations

| Endpoint                        | Method | Auth | Status | Description                                   |
| ------------------------------- | ------ | ---- | ------ | --------------------------------------------- |
| `/v1/conversations`             | GET    | 🔒   | ✅     | List all user conversations (paginated)       |
| `/v1/conversations`             | POST   | 🔒   | ✅     | Create new conversation                       |
| `/v1/conversations/{conv_id}`   | GET    | 🔒   | ✅     | Get conversation with all items               |
| `/v1/conversations/{conv_id}`   | POST   | 🔒   | ✅     | Update conversation metadata (title, project) |
| `/v1/conversations/{conv_id}`   | DELETE | 🔒   | ✅     | Delete single conversation                    |
| `/v1/conversations`             | DELETE | 🔒   | ✅     | Delete all of the user's conversations        |

### Conversation Items (Messages)

| Endpoint                                                 | Method | Auth | Status | Description                            |
| -------------------------------------------------------- | ------ | ---- | ------ | -------------------------------------- |
| `/v1/conversations/{conv_id}/items`                      | GET    | 🔒   | ✅     | List items in a conversation           |
| `/v1/conversations/{conv_id}/items`                      | POST   | 🔒   | ✅     | Add message(s) to conversation         |
| `/v1/conversations/{conv_id}/items/{item_id}`            | GET    | 🔒   | ✅     | Get single message details             |
| `/v1/conversations/{conv_id}/items/{item_id}`            | DELETE | 🔒   | ✅     | Delete message from conversation       |
| `/v1/conversations/{conv_id}/items/by-call-id/{call_id}` | PATCH  | 🔒   | ✅     | Update message matched by external call ID |

### Conversation Sharing

| Endpoint                                        | Method | Auth | Status | Description                            |
| ----------------------------------------------- | ------ | ---- | ------ | -------------------------------------- |
| `/v1/conversations/{conv_id}/share`             | POST   | 🔒   | ✅     | Create shareable conversation link     |
| `/v1/conversations/{conv_id}/shares`            | GET    | 🔒   | ✅     | List shares for a conversation         |
| `/v1/conversations/{conv_id}/shares/{share_id}` | DELETE | 🔒   | ✅     | Revoke shareable link                  |
| `/v1/shares`                                    | GET    | 🔒   | ✅     | List all of the user's shares          |
| `/v1/shares/{share_id}`                         | DELETE | 🔒   | ✅     | Revoke one of the user's shares        |
| `/v1/public/shares/{slug}`                      | GET    | ❌   | ✅     | Access a shared conversation (no auth) |

### Models

| Endpoint                                | Method | Auth | Status | Description                                       |
| --------------------------------------- | ------ | ---- | ------ | ------------------------------------------------- |
| `/v1/models`                            | GET    | 🔒   | ✅     | List available models                             |
| `/v1/models/catalogs/{model_public_id}` | GET    | 🔒   | ✅     | Get model catalog details (supported parameters)  |

### Projects

| Endpoint                                  | Method | Auth | Status | Description                   |
| ----------------------------------------- | ------ | ---- | ------ | ----------------------------- |
| `/v1/projects`                            | GET    | 🔒   | ✅     | List all projects             |
| `/v1/projects`                            | POST   | 🔒   | ✅     | Create new project            |
| `/v1/projects/{project_id}`               | GET    | 🔒   | ✅     | Get project details           |
| `/v1/projects/{project_id}`               | PATCH  | 🔒   | ✅     | Update project metadata       |
| `/v1/projects/{project_id}`               | DELETE | 🔒   | ✅     | Soft-delete project           |
| `/v1/projects/{project_id}/conversations` | GET    | 🔒   | ✅     | List conversations in project |

### User Settings

| Endpoint                            | Method | Auth | Status | Description                           |
| ----------------------------------- | ------ | ---- | ------ | ------------------------------------- |
| `/v1/users/me/settings`             | GET    | 🔒   | ✅     | Get user settings                     |
| `/v1/users/me/settings`             | PATCH  | 🔒   | ✅     | Update user settings (partial update) |
| `/v1/users/me/settings/preferences` | GET    | 🔒   | ✅     | Get user preferences                  |
| `/v1/users/me/settings/preferences` | PATCH  | 🔒   | ✅     | Update user preferences               |

### Admin Endpoints (Model & Provider Management)

| Endpoint                                                | Method | Auth | Status | Description                          |
| ------------------------------------------------------- | ------ | ---- | ------ | ------------------------------------ |
| `/v1/admin/models/catalogs`                             | GET    | 🔒   | ✅     | List all model catalogs (admin view) |
| `/v1/admin/models/catalogs/{model_public_id}`           | GET    | 🔒   | ✅     | Get catalog details (admin view)     |
| `/v1/admin/models/catalogs/{model_public_id}`           | PATCH  | 🔒   | ✅     | Update catalog configuration         |
| `/v1/admin/models/catalogs/bulk-toggle`                 | POST   | 🔒   | ✅     | Enable/disable multiple models       |
| `/v1/admin/models/provider-models`                      | GET    | 🔒   | ✅     | List provider models (admin)         |
| `/v1/admin/models/provider-models/{provider_model_id}`  | GET    | 🔒   | ✅     | Get provider model details           |
| `/v1/admin/models/provider-models/{provider_model_id}`  | PATCH  | 🔒   | ✅     | Update provider model config         |
| `/v1/admin/models/provider-models/bulk-toggle`          | POST   | 🔒   | ✅     | Toggle multiple provider models      |
| `/v1/admin/providers`                                   | GET    | 🔒   | ✅     | List upstream providers              |
| `/v1/admin/providers`                                   | POST   | 🔒   | ✅     | Register a provider                  |
| `/v1/admin/providers/{provider_public_id}`              | GET/PATCH/DELETE | 🔒 | ✅ | Manage a single provider             |

### Health & Status

| Endpoint                                | Method | Auth | Status | Description               |
| --------------------------------------- | ------ | ---- | ------ | ------------------------- |
| `/healthz`, `/readyz`, `/v1/version`    | GET    | ❌   | ✅     | Health / readiness / build version |

## Response API Endpoints

**Base URL:** `http://localhost:8082` (service paths shown below are under `/v1`).

> Via Kong the `/responses` route uses `strip_path`, so a service path like `/v1/responses`
> is reached at `http://localhost:8000/responses/v1/responses`. The `/v1/artifacts` and
> `/v1/agents` routes are **not** stripped and are reached at their path directly.

### Response Execution

| Endpoint                                  | Method | Auth | Status | Description                                       |
| ----------------------------------------- | ------ | ---- | ------ | ------------------------------------------------- |
| `/v1/responses`                           | POST   | 🔒   | ✅     | Create response (multi-step tool orchestration)   |
| `/v1/responses/{response_id}`             | GET    | 🔒   | ✅     | Get response details and execution status         |
| `/v1/responses/{response_id}/full`        | GET    | 🔒   | ✅     | Get response with full execution detail           |
| `/v1/responses/{response_id}`             | DELETE | 🔒   | ✅     | Delete response                                   |
| `/v1/responses/{response_id}/cancel`      | POST   | 🔒   | ✅     | Cancel an in-progress response                    |
| `/v1/responses/{response_id}/retry`       | POST   | 🔒   | ✅     | Retry a failed response                           |
| `/v1/responses/{response_id}/input_items` | GET    | 🔒   | ✅     | List input items for a response                   |

> Webhooks are **not** a CRUD API. To receive a callback when a background response
> completes, pass `metadata.webhook_url` in the create-response request.

### Plan (nested under a response)

| Endpoint                                    | Method | Auth | Status | Description                          |
| ------------------------------------------- | ------ | ---- | ------ | ------------------------------------ |
| `/v1/responses/{response_id}/plan`          | GET    | 🔒   | ✅     | Get the execution plan               |
| `/v1/responses/{response_id}/plan/details`  | GET    | 🔒   | ✅     | Get plan with step details           |
| `/v1/responses/{response_id}/plan/progress` | GET    | 🔒   | ✅     | Get plan progress                    |
| `/v1/responses/{response_id}/plan/cancel`   | POST   | 🔒   | ✅     | Cancel plan execution                |
| `/v1/responses/{response_id}/plan/input`    | POST   | 🔒   | ✅     | Submit user input to a waiting plan  |
| `/v1/responses/{response_id}/plan/tasks`    | GET    | 🔒   | ✅     | List plan tasks                      |

### Artifacts

| Endpoint                                       | Method | Auth | Status | Description                          |
| ---------------------------------------------- | ------ | ---- | ------ | ------------------------------------ |
| `/v1/artifacts`                                | GET    | 🔒   | ✅     | List artifacts for the current user  |
| `/v1/artifacts/{artifact_id}`                  | GET    | 🔒   | ✅     | Get artifact                         |
| `/v1/artifacts/{artifact_id}/versions`         | GET    | 🔒   | ✅     | List artifact versions               |
| `/v1/artifacts/{artifact_id}/download`         | GET    | 🔒   | ✅     | Download artifact content            |
| `/v1/artifacts/{artifact_id}`                  | DELETE | 🔒   | ✅     | Delete artifact                      |
| `/v1/responses/{response_id}/artifacts`        | GET    | 🔒   | ✅     | List artifacts for a response        |
| `/v1/responses/{response_id}/artifacts/latest` | GET    | 🔒   | ✅     | Get the latest artifact for response |

### Agent Discovery

| Endpoint                      | Method | Auth | Status | Description                          |
| ----------------------------- | ------ | ---- | ------ | ------------------------------------ |
| `/v1/agents`                  | GET    | 🔒   | ✅     | List available agents                |
| `/v1/agents/capabilities`     | GET    | 🔒   | ✅     | List agent capabilities              |
| `/v1/agents/{type}`           | GET    | 🔒   | ✅     | Get agent by type                    |
| `/v1/agents/{type}/schema`    | GET    | 🔒   | ✅     | Get agent input schema               |

### Health & Status

| Endpoint   | Method | Auth | Status | Description          |
| ---------- | ------ | ---- | ------ | -------------------- |
| `/healthz` | GET    | ❌   | ✅     | Service health check |

## Media API Endpoints

**Base URL:** `http://localhost:8285` (via Kong: `/media` for management, `/api/media` for public serving)

### Media Operations

| Endpoint                     | Method | Auth | Status | Description                                                  |
| ---------------------------- | ------ | ---- | ------ | ------------------------------------------------------------ |
| `/v1/media`                  | POST   | 🔒   | ✅     | Ingest a data URL or remote URL; returns the `jan_*` ID      |
| `/v1/media/upload`           | POST   | 🔒   | ✅     | Multipart file upload; returns the `jan_*` ID                |
| `/v1/media/{id}`             | GET    | 🔒   | ✅     | Stream media bytes (or a direct URL if proxying is disabled) |
| `/v1/media/{id}/metadata`    | GET    | 🔒   | ✅     | Get media metadata (id, url, content_type, filename, size)   |
| `/api/media/{id}`            | GET    | ❌   | ✅     | Public read-only serving via Kong (used in `img src`)        |

> `/v1/files`, `/v1/files/upload`, `/v1/files/{id}`, and `/v1/files/{id}/metadata` are
> registered as aliases of the `/v1/media*` routes above.
>
> There is no presigned-upload, resolve, or bulk-delete endpoint.

### Health & Status

| Endpoint   | Method | Auth | Status | Description          |
| ---------- | ------ | ---- | ------ | -------------------- |
| `/healthz` | GET    | ❌   | ✅     | Service health check |

## MCP Tools API Endpoints

**Base URL:** `http://localhost:8091`

### Tool Operations

All tools are exposed through a **single JSON-RPC 2.0 endpoint**. The method (`tools/list`,
`tools/call`, `initialize`, `ping`, etc.) is selected via the request body, not the URL path.

| Endpoint  | Method | Auth | Status | Description                                                            |
| --------- | ------ | ---- | ------ | --------------------------------------------------------------------- |
| `/v1/mcp` | POST   | 🔒   | ✅     | JSON-RPC 2.0 endpoint for all MCP methods (`tools/list`, `tools/call`) |

> Via Kong this is reached at `POST /mcp`. Search uses a provider fallback chain:
> Serper -> Exa -> Tavily -> SearXNG.

### Admin Tools (managed in LLM API)

MCP tool admin configuration lives in the **LLM API**, not the MCP Tools service.

| Endpoint                        | Method | Auth | Status | Description                       |
| ------------------------------- | ------ | ---- | ------ | --------------------------------- |
| `/v1/admin/mcp-tools`           | GET    | 🔒   | ✅     | List MCP tools with admin config  |
| `/v1/admin/mcp-tools/{id}`      | GET    | 🔒   | ✅     | Get tool admin configuration      |
| `/v1/admin/mcp-tools/{id}`      | PATCH  | 🔒   | ✅     | Update tool enable/disable status |

### Health & Status

| Endpoint   | Method | Auth | Status | Description          |
| ---------- | ------ | ---- | ------ | -------------------- |
| `/healthz` | GET    | ❌   | ✅     | Service health check |

## Template API

`template-api` is a **scaffold/reference service** used as the starting point for new
microservices (`jan-cli dev scaffold`). It is not part of the running platform
and is not routed through Kong, so its endpoints are intentionally omitted from this inventory.

## Summary Statistics

### By Service

| Service          | Status                                            |
| ---------------- | ------------------------------------------------- |
| **LLM API**      | ✅ Comprehensive (auth, chat, conversations, …)   |
| **Response API** | ✅ Responses, plan, artifacts, agent discovery    |
| **Media API**    | ✅ Ingest, upload, fetch, metadata                |
| **MCP Tools**    | ✅ Single JSON-RPC endpoint                       |
| **Template API** | Scaffold only (not routed)                        |

## API Versioning

All endpoints use path-based versioning:

```
/v1/  ← Current production version
/v2/  ← Future version (not yet available)
```

Breaking changes only occur in major version increments.

## Gateway Routing (Kong)

When using the Kong gateway (port 8000), routes map as follows (see
[integrations/kong/kong.yml](../../integrations/kong/kong.yml)):

| Kong path             | Upstream service       | Path handling                                   |
| --------------------- | ---------------------- | ----------------------------------------------- |
| `/llm/*`              | LLM API (8080)         | `strip_path` (e.g. `/llm/auth/...` → `/auth/...`) |
| `/v1/*`               | LLM API (8080)         | preserved                                       |
| `/auth/*`             | LLM API (8080)         | preserved                                       |
| `/responses/*`        | Response API (8082)    | `strip_path`                                    |
| `/v1/artifacts`       | Response API (8082)    | preserved                                       |
| `/v1/agents`          | Response API (8082)    | preserved                                       |
| `/media/*`            | Media API (8285)       | `strip_path` (→ `/media/...` on the service)    |
| `/api/media/*`        | Media API (8285)       | preserved (public read)                         |
| `/mcp`                | MCP Tools (8091)       | `strip_path` → service `/v1/mcp`                |

## Error Response Format

All errors follow standard format across services:

```json
{
  "error": {
    "type": "error_type",
    "code": "error_code",
    "message": "Human-readable error message",
    "param": "parameter_name",
    "request_id": "req_xyz"
  }
}
```

## Rate Limiting

Rate limits are enforced by the Kong gateway in **all** environments (global limit plus
per-route overrides). See the rate-limit table in the
[API Reference](README.md#rate-limits) and the source of truth,
[integrations/kong/kong.yml](../../integrations/kong/kong.yml).

## Related Documentation

- [API Reference](README.md) - Base URLs, auth, conventions
- [LLM API Reference](llm-api/README.md) - Complete LLM API documentation
- [Response API Reference](response-api/README.md) - Response orchestration guide
- [Media API Reference](media-api/README.md) - Media handling guide
- [MCP Tools Reference](mcp-tools/README.md) - Tool execution guide
