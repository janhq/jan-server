# Testing Guide

This guide covers all testing approaches for the jan-server project.

## Table of Contents

1. [Test Types](#test-types)
2. [Running Tests](#running-tests)
3. [Test Suites](#test-suites)
4. [Writing Tests](#writing-tests)
5. [CI/CD Integration](#cicd-integration)

## Test Types

### 1. Unit Tests (Go)

Fast, isolated tests for individual functions and methods.

```bash
# Run all unit tests
make test

# Run tests for specific service
make test-api      # LLM API tests
make test-mcp      # MCP Tools tests

# With coverage
make test-coverage  # Generates coverage.html
```

### 2. Integration Tests (Newman/Postman)

End-to-end API testing using Newman (Postman CLI).

```bash
# Run all integration tests
make test-all

# Run specific test suites
make test-auth              # Authentication & authorization
make test-conversations     # Conversation API
make test-mcp-integration   # MCP tools integration
```

## Running Tests

### Quick Test Run

```bash
# 1. Start services
make up-full

# 2. Wait for services to be ready
make health-check

# 3. Run tests
make test-all
```

### Complete Test Workflow

```bash
# 1. Setup test environment
make test-setup        # Switches to testing.env, starts services

# 2. Run tests
make test-all

# 3. Teardown
make test-teardown     # Stops services

# 4. Clean artifacts
make test-clean        # Removes newman.json, coverage files
```

## Test Suites

### Authentication Tests (`test-auth`)

**File**: `tests/automation/auth-postman-scripts.json`

Tests OAuth2/OIDC authentication flow with Keycloak:

- Client credentials grant
- Token generation
- Token validation
- Protected endpoint access

**Environment Variables**:
```bash
kong_url=http://localhost:8000
llm_api_url=http://localhost:8080
keycloak_base_url=http://localhost:8085
keycloak_admin=admin
keycloak_admin_password=admin
realm=jan
client_id_public=llm-api
```

**Run**:
```bash
make test-auth
```

### Conversation API Tests (`test-conversations`)

**File**: `tests/automation/conversations-postman-scripts.json`

Tests conversation management API:

- Create conversation
- List conversations
- Get conversation by ID
- Update conversation
- Delete conversation

**Environment Variables**:
```bash
kong_url=http://localhost:8000
llm_api_url=http://localhost:8000/llm
keycloak_base_url=http://localhost:8085
keycloak_admin=admin
keycloak_admin_password=admin
realm=jan
client_id_public=llm-api
```

**Run**:
```bash
make test-conversations
```

### Response API Tests (`test-response`)

**File**: `tests/automation/responses-postman-scripts.json`

Tests response API functionality:

- Response creation
- Response retrieval
- Response streaming
- Error handling

**Environment Variables**:
```bash
response_api_url=http://localhost:8000/responses
llm_api_url=http://localhost:8000/llm
mcp_tools_url=http://localhost:8000/mcp
```

**Run**:
```bash
make test-response
```

### Media API Tests (`test-media`)

**File**: `tests/automation/media-postman-scripts.json`

Tests media upload and management:

- File upload
- File retrieval
- File deletion
- Presigned URLs
- Size limits

**Environment Variables**:
```bash
media_api_url=http://localhost:8000/media
media_service_key=changeme-media-key
```

**Run**:
```bash
make test-media
```

### MCP Integration Tests (`test-mcp-integration`)

**File**: `tests/automation/mcp-postman-scripts.json`

Tests MCP (Model Context Protocol) tools:

**SearXNG (Web Search)**:
- List tools
- Web search queries
- Result formatting

**Vector Store (File Search)**:
- File upload
- Vector indexing
- Semantic search
- File deletion

**SandboxFusion (Code Execution)**:
- Python code execution
- Output capture
- Error handling

**Environment Variables**:
```bash
kong_url=http://localhost:8000
llm_api_url=http://localhost:8000/llm
mcp_tools_url=http://localhost:8000/mcp
searxng_url=http://localhost:8086
```

**Run**:
```bash
make test-mcp-integration
```

### Gateway End-to-End Tests (`test-e2e`)

**File**: `tests/automation/test-all.postman.json`

Tests complete flows through Kong Gateway:

- Gateway routing
- Service integration
- Authentication flow
- Cross-service communication

**Environment Variables**:
```bash
gateway_url=http://localhost:8000
llm_api_url=http://localhost:8000/llm
media_api_url=http://localhost:8000/media
response_api_url=http://localhost:8000/responses
mcp_tools_url=http://localhost:8000/mcp
media_service_key=changeme-media-key
```

**Run**:
```bash
make test-e2e
```

**Run**:
```bash
make test-mcp-integration
```

## Test Debugging

### Newman Debug Mode

Run tests with verbose output:

```bash
make newman-debug
```

This shows:
- Full HTTP requests/responses
- Headers
- Body content
- Timing information

### Manual API Testing

```bash
# Test health endpoints
make curl-health

# Test MCP tools list
make curl-mcp

# Test chat completion (requires TOKEN)
TOKEN=your_token_here make curl-chat
```

### View Service Logs

```bash
# All logs
make logs

# Specific service
make logs-api
make logs-mcp

# Error logs only
make logs-error

# Tail last 100 lines
make logs-api-tail
make logs-mcp-tail
```

## Writing Tests

### Adding Newman Tests

1. **Open Postman** and create your requests
2. **Export collection** to `tests/automation/`
3. **Add test scripts** in Postman:

```javascript
// Example: Test successful response
pm.test("Status code is 200", function () {
    pm.response.to.have.status(200);
});

pm.test("Response has data", function () {
    var jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property('data');
});

// Save values for later requests
pm.environment.set("conversation_id", jsonData.data.id);
```

4. **Add Makefile target**:

```makefile
## test-myfeature: Run my feature tests
test-myfeature:
	@echo "Running my feature tests..."
	@$(NEWMAN) run tests/automation/myfeature-postman-scripts.json \
		--env-var "api_url=http://localhost:8080" \
		--reporters cli
	@echo " My feature tests passed"
```

### Adding Go Unit Tests

```go
// services/llm-api/internal/domain/conversation_test.go
package domain_test

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestConversationCreation(t *testing.T) {
	conv := &Conversation{
		Title: "Test Conversation",
	}
	
	assert.NotNil(t, conv)
	assert.Equal(t, "Test Conversation", conv.Title)
}
```

Run with:
```bash
make test-api
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Setup
        run: make setup
      
      - name: Run unit tests
        run: make test
      
      - name: Start services
        run: make up-full
      
      - name: Wait for services
        run: sleep 30
      
      - name: Health check
        run: make health-check
      
      - name: Run integration tests
        run: make test-all
      
      - name: Cleanup
        if: always()
        run: make down
```

### CI Make Targets

```bash
# Run all CI checks
make ci-test    # Unit + integration tests
make ci-lint    # Code linting
make ci-build   # Build verification
```

## Test Environment Configuration

Tests use `config/testing.env`:

```bash
# API URLs (localhost for tests)
LLM_API_URL=http://localhost:8080
MCP_TOOLS_URL=http://localhost:8091

# Database
DB_DSN=postgres://jan_user:jan_password@localhost:5432/jan_llm_api

# Keycloak
KEYCLOAK_BASE_URL=http://localhost:8085
KEYCLOAK_ADMIN=admin
KEYCLOAK_ADMIN_PASSWORD=admin

# Logging (info level for tests)
LOG_LEVEL=info
LOG_FORMAT=json
```

Switch to test environment:
```bash
make env-switch ENV=testing
```

## Troubleshooting Tests

### Tests Fail with "Connection Refused"

Services might not be ready:

```bash
# Check service health
make health-check

# Wait longer
sleep 10 && make test-all
```

### Authentication Tests Fail

Keycloak might not be initialized:

```bash
# Restart Keycloak
make restart-keycloak

# Wait for it to be ready
sleep 15

# Try again
make test-auth
```

### MCP Tests Fail

Check MCP services are running:

```bash
# Check MCP health
make health-mcp

# View MCP logs
make logs-mcp

# Restart MCP services
make restart-mcp
```

### Database Tests Fail

Reset database:

```bash
# Reset database
make db-reset

# Restart API
make restart-api

# Run tests
make test-all
```

## Best Practices

1. **Always run health checks** before tests
2. **Use test environment** (`config/testing.env`)
3. **Clean up after tests** (`make test-clean`)
4. **Run tests locally** before pushing
5. **Check logs** if tests fail
6. **Use newman-debug** for troubleshooting

## Test Coverage Goals

- **Unit Tests**: >80% coverage
- **Integration Tests**: All critical paths
- **API Endpoints**: 100% coverage
- **MCP Tools**: All tools tested

Check coverage:
```bash
make test-coverage
# Opens coverage.html
```

---

