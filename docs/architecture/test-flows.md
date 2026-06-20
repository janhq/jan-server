# Test Flows Architecture & Diagrams

This document provides visual representations of test flows, dependencies, and service interactions across the jan-server test suite. See [System Design](system-design.md) for the complete system architecture.

---

## Overview

The jan-server test suite consists of Postman collections with 100+ individual test cases covering:

- Authentication flows (guest, JWT, API keys)
- Conversation and project management
- Model catalog and prompt templates
- Tool orchestration via MCP (admin and runtime)
- Media file operations
- Response generation with tool calling
- User management

All tests follow dependency chains: Health -> Auth -> Setup -> Main Tests -> Cleanup

For complete system architecture diagrams, see [System Design](system-design.md).

---

## Test Collections Overview

### 1. Auth Tests (`auth.postman.json`)

**Focus**: Authentication flows and token management

```
Health Checks
 v
Setup [Guest Token] -> [Keycloak Admin] -> [Create User] -> [Set Password]
 v
Main Tests (Parallel)
+- Guest Login Flow [Request Token, Upgrade Account]
+- JWT Login Flow [Keycloak Auth, User Management]
+- API Key Flow [Create, List, Use, Revoke]
 v
Cleanup [Delete User]
```

**20+ Test Cases**

---

### 2. Conversation Tests (`conversation.postman.json`)

**Focus**: Conversation and project management

```
Health & Auth Setup
 v
Model Discovery [List Available Models]
 v
Project Management (Parallel)
+- Create Projects (3 types)
+- CRUD Operations
+- List & Pagination
+- Update (Name, Instructions, Favorite, Archive)
+- Validation Tests
 v
Conversation Flow
+- Create Conversation
+- Verify Title
+- Start Chat (First Message)
+- Continue Chat (Follow-ups)
+- Get Details
+- List Conversations
 v
Cleanup [Delete All Resources]
```

**30+ Test Cases**

---

### 3. Model Tests (`model.postman.json`)

**Focus**: Model catalog operations

```
Authentication
 v
Model Catalog Operations
+- List Models
+- Get Model Details
+- Model Metadata
```

**Test Cases**

---

### 4. Model Prompt Templates Tests (`model-prompt-templates.postman.json`)

**Focus**: Prompt template management

```
Authentication
 v
Template Operations
+- List Templates
+- Get Template
+- Create/Update Templates
```

**Test Cases**

---

### 5. MCP Admin Tests (`mcp-admin.postman.json`)

**Focus**: MCP administrative operations

```
Authentication
 v
MCP Configuration
+- List Available Tools
+- Tool Configuration
+- Admin Settings
```

**Test Cases**

---

### 6. MCP Runtime Tests (`mcp-runtime.postman.json`)

**Focus**: MCP tool execution at runtime

```
Guest Authentication
 v
Tool Discovery [List Available Tools]
 v
Individual Tool Tests
+- Serper Search [Query with domain filters]
+- Web Scraping [Scrape URLs]
+- File Search Index [Index & Query documents]
+- Python Execution [Sandboxed code execution]
+- SearXNG Direct [Meta-search integration]
```

**8+ Test Cases**

---

### 7. Media API Tests (`media.postman.json`)

**Focus**: File upload, storage, and resolution

```
Authentication
 v
Upload Operations
+- Presigned URL Generation
+- Remote URL Ingestion
+- Data URL Ingestion
+- Deduplication Testing
 v
Resolution & Download
+- Payload Resolution (with jan_* placeholders)
+- Direct Stream Download
+- Error Cases (404, 400, 401)
```

**11 Test Cases**

---

### 8. Response API Tests (`response.postman.json`)

**Focus**: Response generation with tool orchestration

```
Authentication & Setup
 v
Health & Service Checks
+- Response API Health
+- MCP Tools Availability
+- LLM API Smoke Test
 v
Response Generation (Parallel)
+- Basic Text Responses [No tools]
+- Single Tool Calling [Search integration]
+- Multi-Step Tool Chains [Search + Scrape]
+- File Search Workflows [Index + Query]
+- Conversation Continuity [Multi-turn with context]
+- Error Handling [Invalid tools, missing params]
+- Complex Scenarios [Search + Scrape + Analyze]
```

**25+ Test Cases**

---

### 9. User Management Tests (`user-management.postman.json`)

**Focus**: User administration and account management

```
Authentication
 v
User Operations
+- Create Users
+- List Users
+- Update User Details
+- Delete Users
+- Role Management
```

**Test Cases**

---

### 10. Full Regression (`make test-all`)

**Focus**: Runs all collections in `tests/e2e/automation/collections/` sequentially for CI/regression. There is no single `test-all` collection file; the `make test-all` target chains the individual collections.

```
Bootstrap (Health + Auth)
 v
[Auth Collection]
 v
[Conversations Collection]
 v
[Media Collection]
 v
[MCP Tools Collection]
 v
[Response API Collection]
 v
Cleanup + Report
```

**1 Flow, 100+ Assertions**

Use `make test-all` when running the full suite or in CI pipelines; it reuses the shared environment file and preserves the dependency order shown above.

---

## Test Flow Sequence Diagrams

### Authentication Sequence

```
Test Runner Services
 | |
 +--> Health Check -> LLM API
 <-- 200 OK <--+
 | |
 +--> Guest Login -> /auth/guest-login
 <-- {access_token,...} <--+
 | |
 +--> Get Keycloak Token -> Keycloak
 <-- {admin_token} <--+
 | |
 +--> Create User -> /admin/realms/{realm}/users
 <-- 201 Created <--+
 | |
 +--> Set Password -> /admin/realms/{realm}/users/{id}
 <-- 204 No Content <--+
 | |
 +--> Obtain User Token -> /realms/{realm}/token
 <-- {user_token} <--+
 | |
 +--> Main Tests -> [Services]
```

---

### Conversation Flow

```
Test Runner Services
 | |
 +--> Authenticate -> LLM API
 <-- {access_token} <--+
 | |
 +--> List Models -> /v1/models
 <-- [{model},...] <--+
 | |
 +--> Create Project -> /v1/projects
 <-- {project_id} <--+
 | |
 +--> Create Conversation -> /v1/conversations
 <-- {conversation_id} <--+
 | |
 +--> Start Chat -> /v1/chat/completions
 | (with conversation) (with conversation.id)
 <-- {message, choices} <--+
 | |
 +--> Continue Chat -> /v1/chat/completions
 | (with history) (with prior messages)
 <-- {message, choices} <--+
 | |
 +--> Cleanup -> [Delete Resources]
```

---

### Response API with Tool Calling

```
Test Runner Services
 | |
 +--> Health Checks -> Response API, MCP Tools
 <-- OK <--+
 | |
 +--> Create Response -> Response API
 | (with tool config) /responses (POST)
 <-- {response_id} <--+
 | |
 | Response Service Calls Tools (Internal)
 | +-> MCP Tools
 | | /tools/call (search)
 | | <- {results}
 | |
 | +-> MCP Tools
 | | /tools/call (scrape)
 | | <- {content}
 | |
 | +-> LLM API
 | /v1/chat/completions
 | <- {final_response}
 |
 +--> Get Response -> /responses/{id}
 <-- {id, content,...} <--+
 | |
 +--> Verify Results OK Success
```

---

### Media Processing Flow

```
Test Runner Services
 | |
 +--> Authenticate -> LLM API
 <-- {access_token} <--+
 | |
 +--> Get Presigned URL -> Media API
 +--> /media/presign (Client uploads to S3/Object Storage)
 <-- {presigned_url} <--+
 | |
 +--> Ingest from URL -> Media API
 +--> /media/ingest (source=url)
 <-- {media_id, hash} <--+
 | |
 +--> Test Deduplication -> Media API
 +--> /media/ingest (same url)
 <-- {media_id: same, deduped: true} <--+
 | |
 +--> Resolve Placeholder -> Media API
 +--> /media/resolve ({{jan_media_{id}}})
 <-- {content: resolved_url} <--+
 | |
 +--> Download -> Media API
 /media/{id} (Stream binary data)
```

---

### MCP Tools Workflow

```
Test Runner Services
 | |
 +--> List Tools -> MCP Tools
 | /tools/list
 <-- {tools: [...]} <--+
 | |
 +--> Execute Serper -> MCP Tools
 | /tools/call (search) (calls Serper API)
 <-- {results: [...]} <--+
 | |
 +--> Execute Scrape -> MCP Tools
 | /tools/call (scrape) (fetches URL content)
 <-- {content} <--+
 | |
 +--> Index Documents -> MCP Tools
 | /tools/call (index) (builds search index)
 <-- {indexed_id, chunks} <--+
 | |
 +--> Query Index -> MCP Tools
 | /tools/call (query) (searches index)
 <-- {results: [...]} <--+
 | |
 +--> Execute Python -> MCP Tools
 /tools/call (exec) (sandboxed execution)
```

---

## Test Dependency Matrix

```
+----------------------------------------------------------------+
| TEST DEPENDENCY HIERARCHY |
+----------------------------------------------------------------+

Level 0: Health Checks
+- Verify all services are running

Level 1: Authentication
+- Guest Token Generation
+- Keycloak Integration
+- JWT Token Generation
+- API Key Management

Level 2: Resource Discovery & Setup
+- List Available Models
+- Create Projects
+- Create Conversations
+- Initialize Test Data

Level 3: Functional Tests (Can run in parallel)
+- Conversation Operations
+- Chat Completions
+- Tool Calling & Orchestration
+- Media File Operations
+- Response Generation

Level 4: Integration Tests
+- Multi-step Workflows
+- Cross-service Interactions
+- Conversation Continuity
+- Tool Chaining

Level 5: Cleanup
+- Delete All Test Resources
```

---

## Service Communication Map

```
 +----------------------+
 | TEST RUNNER |
 | (jan-cli api-test) |
 +----------------------+
 |
 +---------------+---------------+
 | | |
 v v v
 +------------+ +----------+ +----------+
 | Kong | |Keycloak | |SearXNG |
 | (Gateway) | |(Auth) | |(Search) |
 +------+-----+ +----+-----+ +----+-----+
 | | |
 +----------+--------------+--------------+
 | | |
 v v v
+----------+ +----------+ +------------+
| LLM API |<--> MCP Tools | (external) |
|:8080 | |:8091 | |
+----+-----+ +-----+----+ +------------+
 | |
 | +--------+
 | v
 +-> Media API:8285
 |
 +-> Response API:8082
 |
 +-> PostgreSQL (persistent storage)
```

---

## Test Data Flow

```
+-----------------------------------------------------------------+
| POSTMAN COLLECTION VARIABLES |
| |
| SETUP PHASE
| +- guest_access_token <- /auth/guest-login
| +- test_user_id <- /admin/realms/jan/users
| +- kc_admin_access_token <- /realms/master/token
| +- model_id <- /v1/models (first item)
| |
| LLM API PHASE
| +- project_id_1,2,3 <- /v1/projects (POST)
| +- conversation_id <- /v1/conversations (POST)
| +- conversation_title <- GET /v1/conversations/{id}
| |
| RESPONSE API PHASE
| +- response_id <- /responses (POST with tools)
| +- tool_result <- MCP /tools/call
| +- response_content <- GET /responses/{id}
| |
| MEDIA API PHASE
| +- presigned_url <- /media/presign
| +- media_id <- /media/ingest
| +- resolved_content <- /media/resolve
| |
| CLEANUP PHASE
| +- DELETE /v1/conversations/{id}
| +- DELETE /v1/projects/{id}
| +- DELETE /users/{test_user_id}
| |
+-----------------------------------------------------------------+
```

---

## Error Handling Architecture

```
+------------------------+--------------+-------------------------+
| Error Scenario | HTTP Status | Test Assertion |
+------------------------+--------------+-------------------------+
| Service Unavailable | 503 | Retry or Fail Fast |
| Invalid Token | 401 | Verify Rejection |
| Insufficient Perms | 403 | Verify Denial |
| Resource Not Found | 404 | Expected for cleanup |
| Invalid Input | 400 or 422 | Validate Error Message |
| Resource Conflict | 409 | Handle Duplicate Create |
| Rate Limited | 429 | Implement Backoff |
| Internal Error | 500 | Retry or Report |
| Timeout | - | Extend Timeout |
| Connection Refused | - | Ensure Service Running |
+------------------------+--------------+-------------------------+
```

---

## Workflow State Machine

```
 START
 |
 v
+--------------+
|Health Check |--NO---> FAIL (Service Down)
+------+-------+
 |YES
 v
+------------------+
|Authenticate |--NO---> FAIL (Auth Error)
|(Get Token) |
+------+-----------+
 |YES
 v
+------------------+
|Setup Resources |--NO---> FAIL (Setup Error)
|(Projects, Docs) |
+------+-----------+
 |YES
 v
+------------------------------------------+
|Execute Main Tests (Parallel/Serial) |
|- Conversations |
|- Tool Calls |
|- Media Operations |
|- Error Scenarios |
+------+-----------------------------------+
 |
 +----+------+
 |YES NO |
 v v
+--------+ +--------+
|CLEANUP | |CLEANUP |
|SUCCESS | |& FAIL |
| Tests | | Tests |
+----+---+ +----+---+
 | |
 +----+-----+
 v
+------------------+
|Generate Report |
|- Pass/Fail |
|- Assertions |
|- Timing |
|- Coverage |
+------+-----------+
 v
 +-----+
 | END |
 +-----+
```

---

## Test Execution Timeline

```
Timeline: 0s 5s 10s 15s 20s

Auth Flow: [Health ]->[Auth ]->[Models ]->[Setup ]
 v
Conversations: +-----------------[Project Mgmt ]
 | v
 | [Conversation Create->Get->Chat->Continue ]
 v
Media API: | [Ingest->Dedup->Resolve->Download ]
 | v
Response API: | [Basic Response][Tool Call][Multi-step]
 | v
MCP Tools: | [Search][Scrape][IndexQuery][Python]
 |
Cleanup: +---------------[Delete Resources]

Total: ~15-25 seconds (depends on service latency)
```

---

## Test Coverage Summary

| Component       | Tests    | Coverage                                        |
| --------------- | -------- | ----------------------------------------------- |
| Authentication  | 8        | Guest, JWT, API Keys                            |
| Conversations   | 14       | CRUD, Pagination, Validation                    |
| Projects        | 8        | CRUD, State Management                          |
| Chat Completion | 3        | Basic Usage, Conversation                       |
| Models          | 2        | Listing, Details                                |
| Tool Calling    | 8        | Search, Scrape, Index, Exec                     |
| Media Upload    | 3        | URL, DataURL, Dedup                             |
| Media Download  | 2        | Streaming, Error Cases                          |
| Error Handling  | 5        | Invalid Input, Missing Auth                     |
| **TOTAL**       | **100+** | **Comprehensive (run via `make test-all`)**     |

---

## Integration Points

### With System Design

See [System Design](system-design.md) for:

- Architecture layers
- Service responsibilities
- Data flow patterns
- Deployment strategies

### With Services

See [Services](services.md) for:

- LLM API details
- Response API details
- Media API details
- MCP Tools details

### With Security

See [Security](security.md) for:

- Authentication mechanisms
- Authorization patterns
- API key management
- Token validation

### With Data Flow

See [Data Flow](data-flow.md) for:

- Request/response patterns
- Data transformation
- Persistence strategies

---

## Related Documentation

- **Main Architecture Index**: See [README.md](README.md)
