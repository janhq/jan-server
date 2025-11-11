# Testing Guide# Testing Guide



**Generated**: November 11, 2025This guide covers all testing approaches for the jan-server project.



Comprehensive guide for all automated API tests in jan-server, including setup, execution, and best practices.## Table of Contents



---1. [Test Types](#test-types)

2. [Running Tests](#running-tests)

## Table of Contents3. [Test Suites](#test-suites)

4. [Writing Tests](#writing-tests)

1. [Quick Start](#quick-start)5. [CI/CD Integration](#cicd-integration)

2. [Test Collections Overview](#test-collections-overview)

3. [Detailed Test Flows](#detailed-test-flows)## Test Types

4. [Running Tests](#running-tests)

5. [Viewing Results](#viewing-results)### 1. Unit Tests (Go)

6. [CI/CD Integration](#cicd-integration)

7. [Troubleshooting](#troubleshooting)Fast, isolated tests for individual functions and methods.

8. [Best Practices](#best-practices)

```bash

---# Run all unit tests

make test

## Quick Start

# Run tests for specific service

### Prerequisitesmake test-api      # LLM API tests

make test-mcp      # MCP Tools tests

```bash

# Install Newman (Postman CLI)# With coverage

npm install -g newmanmake test-coverage  # Generates coverage.html

```

# Or via brew (macOS)

brew install newman### 2. Integration Tests (Newman/Postman)



# Verify installationEnd-to-end API testing using Newman (Postman CLI).

newman --version

``````bash

# Run all integration tests

### Start Servicesmake test-all



```bash# Run specific test suites

# Terminal 1: Start Docker stackmake test-auth              # Authentication & authorization

docker compose up -dmake test-conversations     # Conversation API

make test-mcp-integration   # MCP tools integration

# Wait for services to be healthy (~30-60 seconds)```

docker compose ps

## Running Tests

# Verify all services are running

docker compose logs -f### Quick Test Run

```

```bash

### Run All Tests# 1. Start services

make up-full

**Via VS Code Tasks**:

1. Open Command Palette: `Ctrl+Shift+P`# 2. Wait for services to be ready

2. Select `Tasks: Run Task`make health-check

3. Choose one of:

   - `Test: Run Auth Postman Scripts`# 3. Run tests

   - `Test: Run Conversation Postman Scripts`make test-all

   - `Test: Run Model Postman Scripts````



**Via Terminal** (Sequential):### Complete Test Workflow

```bash

cd tests/automation```bash

newman run auth-postman-scripts.json --env-var kong_url=http://localhost:8000 --env-var llm_api_url=http://localhost:8000 --env-var keycloak_base_url=http://localhost:8085# 1. Setup test environment

newman run conversations-postman-scripts.json --env-var llm_api_url=http://localhost:8000make test-setup        # Switches to testing.env, starts services

newman run mcp-postman-scripts.json

newman run media-postman-scripts.json --env-var llm_api_url=http://localhost:8000 --env-var media_api_url=http://localhost:8081# 2. Run tests

newman run responses-postman-scripts.jsonmake test-all

```

# 3. Teardown

**Via Terminal** (Parallel):make test-teardown     # Stops services

```bash

cd tests/automation# 4. Clean artifacts

make test-clean        # Removes newman.json, coverage files

newman run auth-postman-scripts.json --env-var kong_url=http://localhost:8000 --env-var llm_api_url=http://localhost:8000 --env-var keycloak_base_url=http://localhost:8085 &```

newman run conversations-postman-scripts.json --env-var llm_api_url=http://localhost:8000 &

newman run mcp-postman-scripts.json &## Test Suites

newman run media-postman-scripts.json --env-var llm_api_url=http://localhost:8000 --env-var media_api_url=http://localhost:8081 &

newman run responses-postman-scripts.json### Authentication Tests (`test-auth`)



wait**File**: `tests/automation/auth-postman-scripts.json`

```

Tests OAuth2/OIDC authentication flow with Keycloak:

---

- Client credentials grant

## Test Collections Overview- Token generation

- Token validation

| Collection | File | Focus | Tests | Status |- Protected endpoint access

|-----------|------|-------|-------|--------|

| **Auth & LLM** | auth-postman-scripts.json | Authentication flows | 8 flows, 20+ | ✅ Ready |**Environment Variables**:

| **Conversations** | conversations-postman-scripts.json | Conversation mgmt | 3 flows, 30+ | ✅ Ready |```bash

| **MCP Tools** | mcp-postman-scripts.json | Tool orchestration | 2 flows, 8+ | ✅ Ready |kong_url=http://localhost:8000

| **Media API** | media-postman-scripts.json | Media operations | 11 tests | ✅ Ready |llm_api_url=http://localhost:8080

| **Response API** | responses-postman-scripts.json | Response generation | 9 flows, 25+ | ✅ Ready |keycloak_base_url=http://localhost:8085

keycloak_admin=admin

**Total**: 5 collections, 27+ flows, 100+ individual test caseskeycloak_admin_password=admin

realm=jan

---client_id_public=llm-api

```

## Detailed Test Flows

**Run**:

### 1. Authentication Tests (`auth-postman-scripts.json`)```bash

make test-auth

**Purpose**: Comprehensive testing of authentication flows including JWT tokens (guest/registered users), API keys, and Kong gateway validation.```



**What it Tests**:### Conversation API Tests (`test-conversations`)

- ✓ Guest token issuance

- ✓ Keycloak user management**File**: `tests/automation/conversations-postman-scripts.json`

- ✓ JWT token generation

- ✓ API key creation/usage/revocationTests conversation management API:

- ✓ Kong gateway validation

- Create conversation

#### Flow Diagram- List conversations

- Get conversation by ID

```- Update conversation

Health Checks- Delete conversation

    ↓

Setup [Guest Token] → [Keycloak Admin] → [Create User] → [Set Password]**Environment Variables**:

    ↓```bash

Main Tests (Parallel)kong_url=http://localhost:8000

├─ LLM API - Guest Token     [List Models, Get Details, Chat]llm_api_url=http://localhost:8000/llm

├─ LLM API - User Token      [List Models]keycloak_base_url=http://localhost:8085

├─ Guest Login Flow          [Request Token, Upgrade Account]keycloak_admin=admin

├─ JWT Login Flow            [Keycloak Auth, User Management]keycloak_admin_password=admin

└─ API Key Flow              [Create, List, Use, Revoke]realm=jan

    ↓client_id_public=llm-api

Cleanup [Delete User]```

```

**Run**:

#### Test Folders & Flows```bash

make test-conversations

**📁 Health Checks**```

- **✓ LLM API Health Check**: Verifies LLM API is running and responding with status `ok`

### Response API Tests (`test-response`)

**📁 Setup**

- **✓ Seed Guest Token**: Creates a guest token for use in subsequent tests**File**: `tests/automation/responses-postman-scripts.json`

- **✓ Seed Obtain Keycloak Admin Token**: Retrieves Keycloak admin credentials

- **✓ Seed Create Test User**: Provisions a test user in the jan realmTests response API functionality:

- **✓ Seed Set Test User Password**: Sets password for the test user

- **✓ Seed Obtain Registered User Token**: Retrieves JWT token for registered user- Response creation

- Response retrieval

**📁 LLM API - Guest Token**- Response streaming

- **✓ List Models (Guest Token)**: Retrieves available models with guest credentials- Error handling

- **✓ Get Model Details (Guest Token)**: Fetches model details as guest

- **✓ Create Chat Completion (Guest Token)**: Initiates chat completion using guest token**Environment Variables**:

```bash

**📁 LLM API - User Token**response_api_url=http://localhost:8000/responses

- **✓ List Models (Registered User)**: Retrieves models with registered user credentialsllm_api_url=http://localhost:8000/llm

mcp_tools_url=http://localhost:8000/mcp

**📁 Guest Login Flow**```

- **✓ Request Guest Token**: Creates new guest token via `/auth/guest-login`

- **✓ Upgrade Guest Account**: Upgrades guest account to registered**Run**:

```bash

**📁 JWT Login Flow**make test-response

- **✓ Obtain Keycloak Admin Token**: Retrieves master realm admin token```

- **✓ Create Test User**: Provisions user in Keycloak

- **✓ Set Test User Password**: Sets user password via Keycloak### Media API Tests (`test-media`)

- **✓ Obtain Registered User Token**: Fetches JWT token for registered user

**File**: `tests/automation/media-postman-scripts.json`

**📁 API Key Flow**

- **✓ Create API Key**: Generates a new API keyTests media upload and management:

- **✓ List API Keys**: Retrieves all API keys for user

- **✓ Use API Key - List Models**: Tests model listing with API key- File upload

- **✓ Use API Key - Chat Completion**: Tests chat completion with API key- File retrieval

- **✓ Test Invalid API Key**: Verifies rejection of invalid API key- File deletion

- **✓ Test No Authentication**: Verifies rejection when no auth provided- Presigned URLs

- **✓ Delete API Key**: Revokes an API key- Size limits

- **✓ Verify Revoked Key Rejected**: Confirms revoked key is rejected

**Environment Variables**:

**📁 Teardown**```bash

- **✓ Delete Test User**: Removes the test user from Keycloakmedia_api_url=http://localhost:8000/media

media_service_key=changeme-media-key

#### Run Command```



```bash**Run**:

newman run auth-postman-scripts.json \```bash

  --env-var kong_url=http://localhost:8000 \make test-media

  --env-var llm_api_url=http://localhost:8000 \```

  --env-var keycloak_base_url=http://localhost:8085 \

  --env-var keycloak_admin=admin \### MCP Integration Tests (`test-mcp-integration`)

  --env-var keycloak_admin_password=admin \

  --env-var realm=jan \**File**: `tests/automation/mcp-postman-scripts.json`

  --reporters cli,json \

  --reporter-json-export auth-results.jsonTests MCP (Model Context Protocol) tools:

```

**SearXNG (Web Search)**:

**Expected Duration**: 10-15 seconds  - List tools

**Expected Pass Rate**: 95%+- Web search queries

- Result formatting

---

**Vector Store (File Search)**:

### 2. Conversations API Tests (`conversations-postman-scripts.json`)- File upload

- Vector indexing

**Purpose**: Comprehensive testing of conversation management features including creation, project association, and multi-turn chat.- Semantic search

- File deletion

**What it Tests**:

- ✓ Project creation and management**SandboxFusion (Code Execution)**:

- ✓ Conversation lifecycle- Python code execution

- ✓ Multi-turn chat- Output capture

- ✓ Pagination and listing- Error handling

- ✓ Input validation

**Environment Variables**:

#### Flow Diagram```bash

kong_url=http://localhost:8000

```llm_api_url=http://localhost:8000/llm

Health & Auth Setupmcp_tools_url=http://localhost:8000/mcp

    ↓searxng_url=http://localhost:8086

Model Discovery [List Available Models]```

    ↓

Project Management (Parallel)**Run**:

├─ Create Projects (3 types)```bash

├─ CRUD Operationsmake test-mcp-integration

├─ List & Pagination```

├─ Update (Name, Instructions, Favorite, Archive)

└─ Validation Tests### Gateway End-to-End Tests (`test-e2e`)

    ↓

Conversation Flow**File**: `tests/automation/test-all.postman.json`

├─ Create Conversation

├─ Verify TitleTests complete flows through Kong Gateway:

├─ Start Chat (First Message)

├─ Continue Chat (Follow-ups)- Gateway routing

├─ Get Details- Service integration

└─ List Conversations- Authentication flow

    ↓- Cross-service communication

Cleanup [Delete All Resources]

```**Environment Variables**:

```bash

#### Test Folders & Flowsgateway_url=http://localhost:8000

llm_api_url=http://localhost:8000/llm

**📁 Health Check**media_api_url=http://localhost:8000/media

- **✓ LLM API Health Check**: Verifies service availabilityresponse_api_url=http://localhost:8000/responses

mcp_tools_url=http://localhost:8000/mcp

**📁 Authentication**media_service_key=changeme-media-key

- **✓ Request Guest Token**: Acquires guest token for conversation tests```



**📁 Model Catalogue****Run**:

- **✓ List Available Models**: Fetches available models for chat operations```bash

make test-e2e

**📁 Project Management**```

- **✓ Create Project - Marketing Campaign**: Creates project with marketing context

- **✓ Create Project - Technical Support**: Creates support workflow project**Run**:

- **✓ Create Project - Personal Assistant**: Creates personal assistant project```bash

- **✓ Get Single Project**: Retrieves specific project detailsmake test-mcp-integration

- **✓ List All Projects - Page 1**: Retrieves first page of projects```

- **✓ List Projects - Page 2 (with cursor)**: Tests pagination with cursor

- **✓ Update Project - Name**: Updates project name## Test Debugging

- **✓ Update Project - Instruction**: Updates project instructions

- **✓ Update Project - Mark as Favorite**: Sets project as favorite### Newman Debug Mode

- **✓ Update Project - Archive**: Archives a project

- **✓ Update Project - Unarchive**: Restores an archived projectRun tests with verbose output:

- **✓ Validation - Create Project with Long Name**: Tests name length validation

- **✓ Validation - Create Project with Empty Name**: Tests required field validation```bash

make newman-debug

**📁 Basic Conversation Flow**```

- **✓ Step 3: Create Conversation**: Creates new conversation with title

- **✓ Step 4: Verify Conversation Title**: Retrieves and verifies metadataThis shows:

- **✓ Step 5: Start Chat with Conversation**: Initiates first message- Full HTTP requests/responses

- **✓ Step 6: Continue Conversation**: Adds follow-up messages- Headers

- **✓ Step 7: Get Conversation Details**: Retrieves full conversation data- Body content

- **✓ Step 8: List User Conversations**: Fetches all conversations for user- Timing information



**📁 Cleanup**### Manual API Testing

- **✓ Delete Conversation**: Removes test conversation

- **✓ Delete Project 1-3**: Removes test projects```bash

- **✓ Verify Deleted Project Not Found**: Confirms deletion# Test health endpoints

make curl-health

#### Run Command

# Test MCP tools list

```bashmake curl-mcp

newman run conversations-postman-scripts.json \

  --env-var kong_url=http://localhost:8000 \# Test chat completion (requires TOKEN)

  --env-var llm_api_url=http://localhost:8000 \TOKEN=your_token_here make curl-chat

  --reporters cli,html \```

  --reporter-html-export conversation-report.html

```### View Service Logs



**Expected Duration**: 15-20 seconds  ```bash

**Expected Pass Rate**: 90%+# All logs

make logs

---

# Specific service

### 3. MCP Tools Tests (`mcp-postman-scripts.json`)make logs-api

make logs-mcp

**Purpose**: Tests Model Context Protocol tool integration including search, scraping, indexing, and code execution.

# Error logs only

**What it Tests**:make logs-error

- ✓ Tool discovery and listing

- ✓ Google Search integration# Tail last 100 lines

- ✓ Web scrapingmake logs-api-tail

- ✓ File indexing and searchmake logs-mcp-tail

- ✓ Python code execution```

- ✓ SearXNG integration

## Writing Tests

#### Flow Diagram

### Adding Newman Tests

```

Guest Authentication1. **Open Postman** and create your requests

    ↓2. **Export collection** to `tests/automation/`

Tool Discovery [List Available Tools]3. **Add test scripts** in Postman:

    ↓

Individual Tool Tests```javascript

├─ Serper Search          [Query with domain filters]// Example: Test successful response

├─ Web Scraping           [Scrape URLs]pm.test("Status code is 200", function () {

├─ File Search Index      [Index & Query documents]    pm.response.to.have.status(200);

├─ Python Execution       [Sandboxed code execution]});

└─ SearXNG Direct         [Meta-search integration]

```pm.test("Response has data", function () {

    var jsonData = pm.response.json();

#### Test Folders & Flows    pm.expect(jsonData).to.have.property('data');

});

**📁 Guest Auth**

- **✓ Request Guest Token**: Acquires token for MCP operations// Save values for later requests

- **✓ MCP Search Domain Filter**: Tests domain filtering in searchpm.environment.set("conversation_id", jsonData.data.id);

- **✓ MCP Search Offline Mode**: Tests offline search functionality```



**📁 MCP Tools**4. **Add Makefile target**:

- **✓ List MCP Tools**: Discovers available MCP tools

- **✓ Serper Search via MCP**: Executes search using Serper```makefile

- **✓ Serper Scrape via MCP**: Scrapes web content via Serper## test-myfeature: Run my feature tests

- **✓ File Search Index**: Indexes documents for searchtest-myfeature:

- **✓ File Search Query**: Queries indexed documents	@echo "Running my feature tests..."

- **✓ SandboxFusion Python Exec**: Executes Python in sandboxed environment	@$(NEWMAN) run tests/automation/myfeature-postman-scripts.json \

		--env-var "api_url=http://localhost:8080" \

**📁 SearXNG**		--reporters cli

- **✓ SearXNG HTML Search**: Tests HTML search format	@echo " My feature tests passed"

- **✓ SearXNG Text Scrape**: Tests text extraction/scraping```



#### Run Command### Adding Go Unit Tests



```bash```go

newman run mcp-postman-scripts.json \// services/llm-api/internal/domain/conversation_test.go

  --timeout-request 30000 \package domain_test

  --reporters cli,json \

  --reporter-json-export mcp-results.jsonimport (

```	"testing"

	"github.com/stretchr/testify/assert"

**Expected Duration**: 20-30 seconds  )

**Expected Pass Rate**: 85%+

func TestConversationCreation(t *testing.T) {

---	conv := &Conversation{

		Title: "Test Conversation",

### 4. Media API Tests (`media-postman-scripts.json`)	}

	

**Purpose**: Comprehensive testing of media service including uploads, deduplication, resolution, and streaming.	assert.NotNil(t, conv)

	assert.Equal(t, "Test Conversation", conv.Title)

**What it Tests**:}

- ✓ Presigned URL generation```

- ✓ Remote URL ingestion

- ✓ Data URL ingestionRun with:

- ✓ Content deduplication```bash

- ✓ Payload resolutionmake test-api

- ✓ Media streaming```

- ✓ Error handling

## CI/CD Integration

#### Flow Diagram

### GitHub Actions Example

```

Authentication```yaml

    ↓name: Tests

Upload Operations

├─ Presigned URL Generationon: [push, pull_request]

├─ Remote URL Ingestion

├─ Data URL Ingestionjobs:

└─ Deduplication Testing  test:

    ↓    runs-on: ubuntu-latest

Resolution & Download    steps:

├─ Payload Resolution (with jan_* placeholders)      - uses: actions/checkout@v2

├─ Direct Stream Download      

└─ Error Cases (404, 400, 401)      - name: Setup

```        run: make setup

      

#### Test Cases      - name: Run unit tests

        run: make test

- **✓ Authentication**: Request guest token for media operations      

- **✓ Health Check**: Verify service availability      - name: Start services

- **✓ Prepare Upload (Get Presigned URL)**: Obtain presigned URLs        run: make up-full

- **✓ Ingest Media (Remote URL)**: Ingest from remote URLs      

- **✓ Ingest Media (Data URL)**: Ingest embedded content      - name: Wait for services

- **✓ Test Deduplication (Upload Same Data URL)**: Verify content deduplication        run: sleep 30

- **✓ Resolve Payload with jan_* Placeholder**: Test placeholder resolution      

- **✓ Proxy Download (Direct Stream)**: Stream media files      - name: Health check

- **✓ Get Nonexistent Media (404 Test)**: Verify 404 handling        run: make health-check

- **✓ Ingest Invalid Source Type (Error Test)**: Validate source type checking      

- **✓ Ingest Without Auth Key (401 Test)**: Verify authentication enforcement      - name: Run integration tests

        run: make test-all

#### Run Command      

      - name: Cleanup

```bash        if: always()

newman run media-postman-scripts.json \        run: make down

  --env-var media_api_url=http://localhost:8081 \```

  --env-var llm_api_url=http://localhost:8000 \

  --env-var media_service_key=dev-key \### CI Make Targets

  --reporters cli,json \

  --reporter-json-export media-results.json```bash

```# Run all CI checks

make ci-test    # Unit + integration tests

**Expected Duration**: 10-15 seconds  make ci-lint    # Code linting

**Expected Pass Rate**: 90%+make ci-build   # Build verification

```

---

## Test Environment Configuration

### 5. Response API Tests (`responses-postman-scripts.json`)

Tests use `config/testing.env`:

**Purpose**: Comprehensive testing of Response API with focus on tool orchestration and multi-step workflows.

```bash

**What it Tests**:# API URLs (localhost for tests)

- ✓ Basic text responsesLLM_API_URL=http://localhost:8080

- ✓ Single-tool integrationMCP_TOOLS_URL=http://localhost:8091

- ✓ Multi-step tool chains

- ✓ Conversation continuity# Database

- ✓ Error handlingDB_DSN=postgres://jan_user:jan_password@localhost:5432/jan_llm_api

- ✓ Response cancellation

# Keycloak

#### Flow DiagramKEYCLOAK_BASE_URL=http://localhost:8085

KEYCLOAK_ADMIN=admin

```KEYCLOAK_ADMIN_PASSWORD=admin

Authentication & Setup

    ↓# Logging (info level for tests)

Health & Service ChecksLOG_LEVEL=info

├─ Response API HealthLOG_FORMAT=json

├─ MCP Tools Availability```

└─ LLM API Smoke Test

    ↓Switch to test environment:

Response Generation (Parallel)```bash

├─ Basic Text Responses     [No tools]make env-switch ENV=testing

├─ Single Tool Calling      [Search integration]```

├─ Multi-Step Tool Chains   [Search + Scrape]

├─ File Search Workflows    [Index + Query]## Troubleshooting Tests

├─ Conversation Continuity  [Multi-turn with context]

├─ Error Handling           [Invalid tools, missing params]### Tests Fail with "Connection Refused"

└─ Complex Scenarios        [Search + Scrape + Analyze]

```Services might not be ready:



#### Test Folders & Flows```bash

# Check service health

**📁 1. Authentication**make health-check

- **✓ Request Guest Token**: Acquires token for response operations

# Wait longer

**📁 2. Model Catalogue**sleep 10 && make test-all

- **✓ List Available Models**: Fetches available LLM models```



**📁 2. Health & Service Checks**### Authentication Tests Fail

- **✓ Response API Health**: Checks Response API status

- **✓ MCP Tools Available**: Verifies MCP tools integrationKeycloak might not be initialized:

- **✓ LLM API Chat Completion Smoke**: Basic LLM functionality test

```bash

**📁 3. Basic Responses (No Tools)**# Restart Keycloak

- **✓ Create Simple Text Response**: Generates basic text responsemake restart-keycloak

- **✓ Get Response by ID**: Retrieves response by identifier

# Wait for it to be ready

**📁 4. Tool Calling - Google Search**sleep 15

- **✓ Create Response with Search Tool**: Generates response using web search

# Try again

**📁 5. Tool Calling - Multi-Step**make test-auth

- **✓ Search and Scrape Chain**: Performs search followed by scraping```



**📁 6. Tool Calling - File Search**### MCP Tests Fail

- **✓ Index Document**: Indexes document for retrieval

- **✓ Query Indexed Documents**: Queries indexed contentCheck MCP services are running:



**📁 7. Conversation Continuity**```bash

- **✓ Create Response with Conversation**: Creates response within conversation# Check MCP health

- **✓ Continue from Previous Response**: Builds on previous responsesmake health-mcp

- **✓ List Input Items**: Lists input items/references

# View MCP logs

**📁 8. Error Handling**make logs-mcp

- **✓ Invalid Tool Name**: Tests rejection of non-existent tools

- **✓ Missing Required Parameters**: Tests parameter validation# Restart MCP services

- **✓ Cancel Response**: Tests response cancellationmake restart-mcp

```

**📁 9. Complex Scenario**

- **✓ Search, Scrape, Analyze Chain**: Complex multi-tool workflow### Database Tests Fail



#### Run CommandReset database:



```bash```bash

newman run responses-postman-scripts.json \# Reset database

  --env-var response_api_url=http://localhost:8082 \make db-reset

  --env-var llm_api_url=http://localhost:8000 \

  --env-var mcp_tools_url=http://localhost:8091 \# Restart API

  --timeout-request 30000 \make restart-api

  --reporters cli,json,html \

  --reporter-html-export response-report.html# Run tests

```make test-all

```

**Expected Duration**: 25-35 seconds  

**Expected Pass Rate**: 85%+## Best Practices



---1. **Always run health checks** before tests

2. **Use test environment** (`config/testing.env`)

## Running Tests3. **Clean up after tests** (`make test-clean`)

4. **Run tests locally** before pushing

### Via Newman CLI5. **Check logs** if tests fail

6. **Use newman-debug** for troubleshooting

**Single Collection**:

```bash## Test Coverage Goals

newman run tests/automation/auth-postman-scripts.json

```- **Unit Tests**: >80% coverage

- **Integration Tests**: All critical paths

**With Environment Variables**:- **API Endpoints**: 100% coverage

```bash- **MCP Tools**: All tools tested

newman run tests/automation/conversations-postman-scripts.json \

  --env-var llm_api_url=http://localhost:8000 \Check coverage:

  --env-var kong_url=http://localhost:8000```bash

```make test-coverage

# Opens coverage.html

**With Postman Environment File**:```

```bash

newman run tests/automation/auth-postman-scripts.json \---

  --environment config/postman-env.json

```

**Multiple Collections Sequential**:
```bash
for file in tests/automation/*-postman-scripts.json; do
  echo "Running: $file"
  newman run "$file"
done
```

**Multiple Collections Parallel**:
```bash
cd tests/automation
newman run auth-postman-scripts.json &
newman run conversations-postman-scripts.json &
newman run mcp-postman-scripts.json &
wait
```

### Environment Configuration

**Collection Variables**:

| Variable | Default | Purpose |
|----------|---------|---------|
| `kong_url` | http://localhost:8000 | Kong gateway endpoint |
| `llm_api_url` | http://localhost:8000 | LLM API endpoint |
| `keycloak_base_url` | http://localhost:8085 | Keycloak server |
| `media_api_url` | http://localhost:8081 | Media API endpoint |
| `response_api_url` | http://localhost:8082 | Response API endpoint |
| `mcp_tools_url` | http://localhost:8091 | MCP Tools endpoint |
| `searxng_url` | http://localhost:8086 | SearXNG endpoint |

**Common Configuration**:
```bash
# Override multiple variables
newman run collection.json \
  --env-var kong_url=http://gateway.example.com:8000 \
  --env-var llm_api_url=http://llm.example.com:8080 \
  --env-var keycloak_base_url=http://keycloak.example.com
```

---

## Viewing Results

### Console Output

```bash
# Basic CLI reporter
newman run collection.json --reporters cli

# Verbose output
newman run collection.json --reporters cli --verbose
```

### HTML Report

```bash
# Generate interactive HTML report
newman run collection.json \
  --reporters html \
  --reporter-html-export report.html

# Open in browser
open report.html  # macOS
start report.html # Windows
```

### JSON Report

```bash
# Export structured JSON results
newman run collection.json \
  --reporters json \
  --reporter-json-export results.json

# Parse with jq/PowerShell
jq '.run.stats' results.json
jq '.run.failures | length' results.json
```

### JUnit XML (for CI/CD)

```bash
# Install reporter
npm install -g newman-reporter-junitfull

# Generate JUnit report
newman run collection.json \
  --reporters cli,junitfull \
  --reporter-junitfull-export test-results.xml
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: API Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_USER: jan_user
          POSTGRES_PASSWORD: jan_password
          POSTGRES_DB: jan_llm_api
      
      keycloak:
        image: keycloak:latest
        env:
          KEYCLOAK_ADMIN: admin
          KEYCLOAK_ADMIN_PASSWORD: admin
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Install Newman
        run: npm install -g newman
      
      - name: Start Services
        run: docker compose up -d
      
      - name: Wait for Services
        run: sleep 30
      
      - name: Run Auth Tests
        run: |
          newman run tests/automation/auth-postman-scripts.json \
            --reporters junitfull \
            --reporter-junitfull-export auth-results.xml
      
      - name: Run Conversation Tests
        run: |
          newman run tests/automation/conversations-postman-scripts.json \
            --reporters junitfull \
            --reporter-junitfull-export conv-results.xml
      
      - name: Upload Results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: '*-results.xml'
```

### GitLab CI Example

```yaml
test:api:
  image: node:18
  services:
    - docker:dind
  before_script:
    - npm install -g newman
    - docker compose up -d
    - sleep 30
  script:
    - cd tests/automation
    - newman run auth-postman-scripts.json --reporters junitfull --reporter-junitfull-export ../auth-results.xml
    - newman run conversations-postman-scripts.json --reporters junitfull --reporter-junitfull-export ../conv-results.xml
  artifacts:
    reports:
      junit: "*-results.xml"
```

---

## Troubleshooting

### Service Health Check

```bash
# Check all services
docker compose ps

# View service logs
docker compose logs llm-api -f
docker compose logs mcp-tools -f
docker compose logs media-api -f

# Check database connectivity
docker compose exec postgres psql -U jan_user -d jan_llm_api -c "SELECT version();"

# Verify Keycloak
curl -s http://localhost:8085/realms/jan | jq '.realm'
```

### Network Connectivity

```bash
# Test service from container
docker compose exec newman curl -s http://llm-api:8080/healthz

# Check DNS resolution
docker compose exec newman nslookup llm-api

# Verify port forwarding
netstat -an | grep 8080  # Linux/macOS
netstat -an | grep 8080  # Windows
```

### Test Failures

| Issue | Solution |
|-------|----------|
| Service Unavailable | Verify services are running: `docker compose ps` |
| Invalid Token | Ensure auth setup step completed successfully |
| Model Not Found | Verify models are available: `curl http://localhost:8000/v1/models` |
| Connection Refused | Check service is running and port is correct |
| Timeout | Increase timeout: `--timeout-request 60000` |
| Database Error | Check PostgreSQL is running: `docker compose logs postgres` |

### Data Consistency

```bash
# Check database state
docker compose exec postgres psql -U jan_user -d jan_llm_api -c "\dt"

# Clear test data between runs
docker compose exec postgres psql -U jan_user -d jan_llm_api -c "TRUNCATE conversations, projects, responses CASCADE;"

# Reset to clean state
docker compose down -v
docker compose up -d
```

---

## Best Practices

### Test Structure

✅ **DO**:
- Each test should be **independent** and reusable
- Use **collection variables** to share data between tests
- Include **assertions** for both positive and negative cases
- Add **descriptive names** and comments
- **Validate** response structure AND content
- **Clean up** resources in teardown steps
- Document **expected values** and **error conditions**

❌ **DON'T**:
- Hardcode values in requests
- Rely on test execution order (except setup/cleanup)
- Skip error validation
- Leave orphaned resources
- Use flaky timings/assertions
- Test UI behavior with API tests

### Common Postman Patterns

**Extract and Store Value**:
```javascript
const response = pm.response.json();
pm.collectionVariables.set('user_id', response.id);
```

**Conditional Logic**:
```javascript
if (pm.response.code === 201) {
    pm.collectionVariables.set('resource_id', pm.response.json().id);
}
```

**Validate Response Structure**:
```javascript
pm.test('Response contains expected fields', function () {
    const data = pm.response.json();
    pm.expect(data).to.have.all.keys('id', 'name', 'created_at');
});
```

**Error Handling**:
```javascript
pm.test('Error response is properly formatted', function () {
    if (pm.response.code >= 400) {
        const error = pm.response.json();
        pm.expect(error).to.have.property('error');
        pm.expect(error).to.have.property('message');
    }
});
```

### Adding New Tests

1. **Open Postman**: Import a collection
2. **Create Request**: Add HTTP request with headers/body
3. **Add Assertions**: Write test scripts
4. **Test Variable**: Use collection variables for data
5. **Export Collection**: File → Export → Format: JSON (v2.1)
6. **Save to Repository**: `tests/automation/`
7. **Update Documentation**: Add flow to this guide
8. **Add VS Code Task** (if needed)

### Test Maintenance

**Regular Updates**:
- Review test coverage quarterly
- Update endpoints when APIs change
- Verify test data is still valid
- Check for flaky/slow tests
- Update documentation

**Performance Baseline**:

| Endpoint | Typical Time |
|----------|--------------|
| `/healthz` | < 50ms |
| `/auth/guest-login` | 100-200ms |
| `/v1/models` | 50-100ms |
| `/v1/chat/completions` | 1-10s |
| `/v1/conversations` | 100-300ms |
| `/media/ingest` | 500ms-2s |
| `/tools/call` | 1-5s |

---

## Related Documentation

- **Architecture & Diagrams**: See `/docs/architecture/test-flows.md`
- **System Design**: See `/docs/architecture/system-design.md`
- **Services Reference**: See `/docs/architecture/services.md`
- **API Reference**: See `/docs/api/README.md`
- **Development Guide**: See `/docs/guides/development.md`

---

**Last Updated**: November 11, 2025  
**Document Type**: Testing Guide  
**Target Audience**: QA Engineers, Developers, DevOps  
**Maintainer**: Jan-Server Team
