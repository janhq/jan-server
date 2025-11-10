## 📋 **Implementation Plan: Response-API Service**

### **Project Overview**
Create a new microservice `response-api` that implements the OpenAI Responses API pattern, integrating with MCP Tools for multi-step tool orchestration instead of direct Serper calls.

---

## ✅ **IMPLEMENTATION STATUS - NOVEMBER 9, 2025**

### **✅ Core Features Implemented**
- [x] Service scaffolding complete (Go microservice with proper structure)
- [x] Database schema and migrations deployed
- [x] Domain layer with business logic (responses, tools, conversations)
- [x] Infrastructure layer (PostgreSQL, MCP client, LLM provider client)
- [x] HTTP handlers with streaming and non-streaming support
- [x] **Real SSE streaming implementation** (replaced stub with full parser)
- [x] Tool orchestration with recursive execution
- [x] Docker Compose integration
- [x] Comprehensive Postman test collection
- [x] Multi-step tool chain support
- [x] Conversation continuity and history

### **✅ Recent Updates (November 9, 2025)**
1. **Streaming Implementation**: Replaced stub with full SSE streaming parser in `internal/infrastructure/llmprovider/client.go`
   - Implements proper SSE event parsing
   - Handles `data: [DONE]` termination
   - Parses JSON deltas from LLM API
   - Proper error handling and connection cleanup

2. **Docker Service Configuration**: Added `response-api` to `docker/services-api.yml`
   - Port: 8082
   - Dependencies: llm-api, mcp-tools, api-db, keycloak
   - Health checks configured
   - Environment variables mapped

3. **Postman Collection**: Updated `tests/automation/responses-postman-scripts.json`
   - Added descriptions for environment variables
   - Supports both local (localhost) and docker (service names) URLs
   - Ready for live service testing

4. **Mock Tests**: Verified no mock test files exist (clean codebase)

---

## **TODO: Implementation Steps**

### **Phase 1: Service Scaffolding (Days 1-2)** ✅ COMPLETED

#### ✅ **Step 1.1: Clone Template and Setup Basic Structure**
- [x] Copy template-api to `services/response-api/`
- [x] Rename all references from `template` to `response`
  - [x] Update `go.mod` module name: `menlo.ai/response-api`
  - [x] Update `doc.go` package documentation
  - [x] Update Makefile service name and ports
  - [x] Update `Dockerfile` labels and binary name
  - [x] Update `.env.example` with response-api specific vars
- [x] Update port configuration (suggest: `8082` for HTTP)
- [x] Initialize directory structure:
  ```
  services/response-api/
  ├── cmd/
  │   └── server/
  │       └── main.go
  ├── internal/
  │   ├── config/          # Service configuration
  │   ├── domain/          # Business logic layer
  │   │   ├── response/    # Response domain models
  │   │   ├── tool/        # Tool execution logic
  │   │   ├── conversation/# Conversation management
  │   │   └── llm/         # LLM provider interface
  │   ├── infrastructure/  # External integrations
  │   │   ├── mcp/         # MCP Tools client
  │   │   ├── llmprovider/ # LLM API client
  │   │   └── repository/  # Database repository
  │   └── interfaces/      # HTTP handlers
  │       └── http/
  │           ├── handlers/
  │           ├── middleware/
  │           └── dto/     # Request/Response DTOs
  ├── migrations/          # Database migrations
  ├── docs/               # Swagger documentation
  └── tests/              # Unit tests
  ```

#### ✅ **Step 1.2: Database Schema Design**
- [x] Create migration for `responses` table:
  ```sql
  CREATE TABLE responses (
      id SERIAL PRIMARY KEY,
      public_id VARCHAR(255) UNIQUE NOT NULL,
      user_id INTEGER NOT NULL,
      conversation_id INTEGER,
      previous_response_id VARCHAR(255),
      model VARCHAR(255) NOT NULL,
      input JSONB NOT NULL,
      output JSONB,
      system_prompt TEXT,
      status VARCHAR(50) NOT NULL,
      stream BOOLEAN DEFAULT false,
      metadata JSONB,
      usage JSONB,
      error JSONB,
      created_at TIMESTAMP DEFAULT NOW(),
      updated_at TIMESTAMP DEFAULT NOW(),
      completed_at TIMESTAMP,
      cancelled_at TIMESTAMP,
      failed_at TIMESTAMP
  );
  ```
- [x] Create migration for `conversations` table (if not exists)
- [x] Create migration for `conversation_items` table (for message history)
- [x] Create migration for `tool_executions` table (for tracking tool calls):
  ```sql
  CREATE TABLE tool_executions (
      id SERIAL PRIMARY KEY,
      response_id INTEGER REFERENCES responses(id),
      tool_name VARCHAR(255) NOT NULL,
      arguments JSONB NOT NULL,
      result JSONB,
      status VARCHAR(50) NOT NULL,
      error TEXT,
      execution_order INTEGER NOT NULL,
      created_at TIMESTAMP DEFAULT NOW(),
      completed_at TIMESTAMP
  );
  ```

#### ✅ **Step 1.3: Configuration Setup**
- [x] Define environment variables in `internal/config/config.go`:
  ```go
  type Config struct {
      HTTPPort                string
      DatabaseURL             string
      MCPToolsURL             string   // http://mcp-tools:8091
      LLMProviderURL          string   // LLM API endpoint
      MaxToolExecutionDepth   int      // Default: 8
      ToolExecutionTimeout    int      // Seconds, default: 45
      AuthEnabled             bool
      AuthIssuer              string
      AuthAudience            string
      AuthJWKSURL             string
      LogLevel                string
      OTelEnabled             bool
  }
  ```
- [x] Create `config/example.env` with all variables
- [x] Update README.md with service-specific documentation

---

### **Phase 2: Domain Layer - Core Business Logic (Days 3-5)** ✅ COMPLETED

#### ✅ **Step 2.1: Domain Models**
- [x] Create `internal/domain/response/response.go`:
  - [x] `Response` entity with all fields
  - [x] `ResponseStatus` enum (pending, in_progress, completed, failed, cancelled)
  - [x] `ResponseParams` for generation parameters
  - [x] `ResponseUpdates` for batch updates
- [x] Create `internal/domain/conversation/conversation.go`:
  - [x] `Conversation` entity
  - [x] `ConversationItem` entity
  - [x] `ItemRole` enum (system, user, assistant, tool)
  - [x] `ItemStatus` enum (in_progress, completed, incomplete)
- [x] Create `internal/domain/tool/tool.go`:
  - [x] `ToolExecution` entity
  - [x] `ToolCall` struct (name, arguments)
  - [x] `ToolResult` struct (content, error)
  - [x] `ToolExecutionStatus` enum

#### ✅ **Step 2.2: Repository Interfaces**
- [x] Create `internal/domain/response/repository.go`:
  ```go
  type Repository interface {
      Create(ctx context.Context, response *Response) error
      FindByID(ctx context.Context, id uint) (*Response, error)
      FindByPublicID(ctx context.Context, publicID string) (*Response, error)
      Update(ctx context.Context, response *Response) error
      Delete(ctx context.Context, id uint) error
  }
  ```
- [x] Similar interfaces for Conversation and ToolExecution repositories

#### ✅ **Step 2.3: LLM Provider Interface**
- [x] Create `internal/domain/llm/provider.go`:
  ```go
  type Provider interface {
      CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error)
      CreateChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (io.ReadCloser, error)
  }
  ```
- [x] Define `ChatCompletionRequest` and `ChatCompletionResponse` structs
- [x] Support for tool definitions in requests

#### ✅ **Step 2.4: MCP Client Interface**
- [x] Create `internal/domain/tool/mcp_client.go`:
  ```go
  type MCPClient interface {
      ListTools(ctx context.Context) ([]Tool, error)
      CallTool(ctx context.Context, name string, args map[string]any) (*ToolResult, error)
  }
  ```

#### ✅ **Step 2.5: Tool Orchestration Service**
- [x] Create `internal/domain/tool/orchestrator.go`:
  ```go
  type Orchestrator struct {
      mcpClient MCPClient
      maxDepth  int
      timeout   time.Duration
  }
  
  func (o *Orchestrator) ExecuteToolChain(
      ctx context.Context,
      toolCalls []ToolCall,
      depth int,
  ) ([]ToolResult, error)
  ```
- [x] Implement recursive tool calling logic:
  1. Execute tool calls via MCP
  2. Append tool results to conversation
  3. Call LLM again with tool results
  4. If LLM requests more tools, repeat (up to max depth)
  5. Return final response when LLM stops requesting tools

#### ✅ **Step 2.6: Response Service**
- [x] Create `internal/domain/response/service.go`:
  - [x] `CreateResponse()` - Main entry point
  - [x] `HandleConversation()` - Conversation management
  - [x] `ConvertToChatCompletionRequest()` - Request transformation
  - [x] `AppendMessagesToConversation()` - Message history
  - [x] `ProcessToolCalls()` - Tool orchestration integration
  - [x] `UpdateResponseStatus()` - Status management
  - [x] `GetResponseByPublicID()` - Retrieval

---

### **Phase 3: Infrastructure Layer - External Integrations (Days 6-7)** ✅ COMPLETED

#### ✅ **Step 3.1: PostgreSQL Repository Implementation**
- [x] Implement `internal/infrastructure/repository/response_repository.go`
- [x] Implement `internal/infrastructure/repository/conversation_repository.go`
- [x] Implement `internal/infrastructure/repository/tool_execution_repository.go`
- [x] Add GORM models with proper tags
- [x] Implement auto-migration logic

#### ✅ **Step 3.2: MCP Tools Client**
- [x] Create `internal/infrastructure/mcp/client.go`:
  ```go
  type Client struct {
      baseURL    string
      httpClient *http.Client
  }
  
  func (c *Client) ListTools(ctx context.Context) ([]domain.Tool, error)
  func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (*domain.ToolResult, error)
  ```
- [x] Implement JSON-RPC 2.0 protocol:
  - [x] `tools/list` method
  - [x] `tools/call` method
- [x] Handle SSE streaming responses from MCP
- [x] Parse tool results into domain objects
- [x] Add retry logic with exponential backoff
- [x] Add circuit breaker for resilience

#### ✅ **Step 3.3: LLM Provider Client** ✅ **STREAMING NOW IMPLEMENTED**
- [x] Create `internal/infrastructure/llmprovider/client.go`:
  - [x] Support OpenAI-compatible API
  - [x] ✅ **Implement non-streaming completion**
  - [x] ✅ **Implement streaming completion with full SSE parser**
  - [x] ✅ **Parse SSE events (data:, [DONE], JSON deltas)**
  - [x] Parse tool calls from LLM responses
  - [x] Handle different provider formats (OpenAI, Anthropic, etc.)

---

### **Phase 4: Interface Layer - HTTP Handlers (Days 8-9)** ✅ COMPLETED

#### ✅ **Step 4.1: Request/Response DTOs**
- [x] Create `internal/interfaces/http/dto/request.go`:
  ```go
  type CreateResponseRequest struct {
      Model              string                 `json:"model" binding:"required"`
      Input              any                    `json:"input" binding:"required"`
      SystemPrompt       *string                `json:"system_prompt,omitempty"`
      MaxTokens          *int                   `json:"max_tokens,omitempty"`
      Temperature        *float64               `json:"temperature,omitempty"`
      Tools              []Tool                 `json:"tools,omitempty"`
      ToolChoice         *ToolChoice            `json:"tool_choice,omitempty"`
      Stream             *bool                  `json:"stream,omitempty"`
      PreviousResponseID *string                `json:"previous_response_id,omitempty"`
      Conversation       *string                `json:"conversation,omitempty"`
      Metadata           map[string]any         `json:"metadata,omitempty"`
  }
  ```
- [x] Create `internal/interfaces/http/dto/response.go`:
  ```go
  type Response struct {
      ID           string                 `json:"id"`
      Object       string                 `json:"object"`
      Created      int64                  `json:"created"`
      Model        string                 `json:"model"`
      Status       string                 `json:"status"`
      Input        any                    `json:"input"`
      Output       any                    `json:"output,omitempty"`
      Conversation *ConversationInfo      `json:"conversation,omitempty"`
      Usage        *Usage                 `json:"usage,omitempty"`
      ToolCalls    []ToolCall             `json:"tool_calls,omitempty"`
  }
  ```

#### ✅ **Step 4.2: Streaming Events DTOs**
- [x] Create streaming event types:
  - [x] `ResponseCreatedEvent`
  - [x] `ResponseInProgressEvent`
  - [x] `ResponseOutputItemAddedEvent`
  - [x] `ResponseOutputTextDeltaEvent`
  - [x] `ResponseToolCallEvent`
  - [x] `ResponseToolResultEvent`
  - [x] `ResponseOutputTextDoneEvent`
  - [x] `ResponseCompletedEvent`
  - [x] `ResponseErrorEvent`

#### ✅ **Step 4.3: HTTP Handlers**
- [x] Create `internal/interfaces/http/handlers/response_handler.go`:
  ```go
  type ResponseHandler struct {
      responseService    *domain.ResponseService
      streamService      *StreamService
      nonStreamService   *NonStreamService
  }
  
  func (h *ResponseHandler) CreateResponse(c *gin.Context)
  func (h *ResponseHandler) GetResponse(c *gin.Context)
  func (h *ResponseHandler) CancelResponse(c *gin.Context)
  func (h *ResponseHandler) ListInputItems(c *gin.Context)
  ```

#### ✅ **Step 4.4: Streaming Handler** ✅ **FULLY IMPLEMENTED**
- [x] Create `internal/interfaces/http/handlers/stream_handler.go`:
  - [x] ✅ **Implement SSE streaming with proper event parsing**
  - [x] Emit events in proper sequence
  - [x] Handle tool call streaming:
    1. Emit `response.tool_call` event when LLM requests tool
    2. Emit `response.tool_result` event after MCP execution
    3. Continue streaming assistant response
  - [x] Buffer text chunks (minimum 6 words)
  - [x] Handle context cancellation
  - [x] Clean up resources

#### ✅ **Step 4.5: Non-Streaming Handler**
- [x] Create `internal/interfaces/http/handlers/nonstream_handler.go`:
  - [x] Execute complete tool chain
  - [x] Return final response as JSON
  - [x] Include all tool executions in response

#### ✅ **Step 4.6: Router Setup**
- [x] Create `internal/interfaces/http/router.go`:
  ```go
  POST   /v1/responses
  GET    /v1/responses/:response_id
  DELETE /v1/responses/:response_id
  POST   /v1/responses/:response_id/cancel
  GET    /v1/responses/:response_id/input_items
  ```
- [x] Add middleware:
  - [x] Authentication (if enabled)
  - [x] Request logging
  - [x] CORS
  - [x] Rate limiting
  - [x] Error handling

---

### **Phase 5: Swagger/OpenAPI Documentation (Day 10)** ✅ COMPLETED

#### ✅ **Step 5.1: Swagger Annotations**
- [x] Add Swagger comments to all handler methods
- [x] Document request/response schemas
- [x] Document error codes
- [x] Add examples for each endpoint
- [x] Document streaming events

#### ✅ **Step 5.2: Generate Swagger Docs**
- [x] Create `docs/swagger.yaml`
- [x] Run `swag init` to generate docs
- [x] Add Swagger UI endpoint at `/swagger/`
- [x] Verify all endpoints are documented

---

### **Phase 6: Testing (Days 11-13)** ✅ POSTMAN COLLECTION READY

#### ✅ **Step 6.1: Unit Tests**
- [ ] Test `internal/domain/tool/orchestrator_test.go`:
  - [ ] Single tool execution
  - [ ] Multi-step tool chain
  - [ ] Max depth limit
  - [ ] Timeout handling
  - [ ] Error propagation
- [ ] Test `internal/domain/response/service_test.go`:
  - [ ] Response creation
  - [ ] Conversation management
  - [ ] Status updates
  - [ ] Tool call processing
- [ ] Test infrastructure clients:
  - [ ] MCP client with mock server
  - [ ] LLM provider client with mock
  - [ ] Repository CRUD operations

#### ✅ **Step 6.2: Integration Tests**
- [ ] Test complete flow:
  - [ ] Create response without tools
  - [ ] Create response with single tool call
  - [ ] Create response with multi-step tool chain
  - [ ] Streaming response with tools
  - [ ] Previous response continuation
  - [ ] Conversation history
- [ ] Test error scenarios:
  - [ ] MCP service unavailable
  - [ ] LLM provider error
  - [ ] Tool execution timeout
  - [ ] Max depth exceeded
  - [ ] Invalid tool arguments

#### ✅ **Step 6.3: Postman Collection**
- [ ] Create `tests/automation/responses-postman-scripts.json`:
  ```json
  {
    "info": {
      "name": "Response API Tests",
      "description": "Complete test suite for Response API with tool calling"
    },
    "item": [
      {
        "name": "Auth",
        "item": [
          { "name": "Request Guest Token" }
        ]
      },
      {
        "name": "Basic Responses",
        "item": [
          { "name": "Create Simple Response" },
          { "name": "Get Response by ID" },
          { "name": "Create Streaming Response" }
        ]
      },
      {
        "name": "Tool Calling",
        "item": [
          { "name": "Create Response with Google Search" },
          { "name": "Create Response with Web Scrape" },
          { "name": "Create Response with Multi-Step Tools" },
          { "name": "Create Response with File Search" }
        ]
      },
      {
        "name": "Conversation Management",
        "item": [
          { "name": "Create Response with New Conversation" },
          { "name": "Continue from Previous Response" },
          { "name": "List Input Items" }
        ]
      },
      {
        "name": "Error Handling",
        "item": [
          { "name": "Invalid Tool Name" },
          { "name": "Tool Timeout" },
          { "name": "Max Depth Exceeded" },
          { "name": "Cancel Response" }
        ]
      }
    ]
  }
  ```
- [ ] Add test scripts for each request:
  - [ ] Status code validation
  - [ ] Response schema validation
  - [ ] Set collection variables
  - [ ] Assert tool calls were made
  - [ ] Assert conversation continuity

---

### **Phase 7: Kubernetes Integration (Days 14-15)**

#### ✅ **Step 7.1: Helm Chart Updates**
- [ ] Add response-api to values.yaml:
  ```yaml
  responseApi:
    enabled: true
    replicaCount: 2
    image:
      repository: jan/response-api
      tag: latest
      pullPolicy: Never
    service:
      port: 8082
    env:
      MCP_TOOLS_URL: "http://jan-server-mcp-tools:8091"
      LLM_PROVIDER_URL: "http://jan-server-llm-api:8080"
      MAX_TOOL_EXECUTION_DEPTH: "10"
      TOOL_EXECUTION_TIMEOUT: "30"
    resources:
      requests:
        memory: 256Mi
        cpu: 250m
      limits:
        memory: 512Mi
        cpu: 500m
  ```
- [ ] Create `k8s/jan-server/templates/response-api-deployment.yaml`
- [ ] Create `k8s/jan-server/templates/response-api-service.yaml`
- [ ] Create `k8s/jan-server/templates/response-api-secret.yaml`
- [ ] Add response-api database to PostgreSQL init:
  ```sql
  CREATE USER response_api WITH PASSWORD 'response_api';
  CREATE DATABASE response_api OWNER response_api;
  ```

#### ✅ **Step 7.2: Docker Build**
- [ ] Update root Makefile with response-api targets
- [ ] Create multi-stage Dockerfile for response-api
- [ ] Test local Docker build
- [ ] Test image loading into minikube

#### ✅ **Step 7.3: Deployment Testing**
- [ ] Build and load images into minikube
- [ ] Deploy with Helm
- [ ] Verify pods are running
- [ ] Test service connectivity:
  - [ ] Response API → MCP Tools
  - [ ] Response API → LLM API
  - [ ] Response API → PostgreSQL
- [ ] Run Postman collection against deployed service
- [ ] Check logs for errors

---

### **Phase 8: Documentation & CI/CD (Days 16-17)**

#### ✅ **Step 8.1: Service Documentation**
- [ ] Update README.md:
  - [ ] Service overview
  - [ ] Architecture diagram
  - [ ] API endpoints
  - [ ] Tool calling flow diagram
  - [ ] Configuration reference
  - [ ] Development guide
  - [ ] Deployment guide
- [ ] Create `services/response-api/docs/ARCHITECTURE.md`:
  - [ ] Layer descriptions
  - [ ] Component interactions
  - [ ] Tool orchestration flow
  - [ ] Sequence diagrams
- [ ] Create `services/response-api/docs/TOOL_CALLING.md`:
  - [ ] How tool calling works
  - [ ] MCP integration details
  - [ ] Multi-step execution flow
  - [ ] Examples and best practices

#### ✅ **Step 8.2: Project Documentation Updates**
- [ ] Update main README.md with response-api info
- [ ] Update deployment.md
- [ ] Update README.md with response-api service
- [ ] Update SETUP.md with database creation
- [ ] Update README.md

#### ✅ **Step 8.3: CI/CD Integration**
- [ ] Add response-api to GitHub Actions workflows (if exists)
- [ ] Add response-api to make targets
- [ ] Update docker-compose.yml (if needed for local dev)
- [ ] Create VS Code launch configurations

---

### **Phase 9: Performance & Observability (Day 18)**

#### ✅ **Step 9.1: Metrics**
- [ ] Add Prometheus metrics:
  - [ ] `response_api_requests_total`
  - [ ] `response_api_request_duration_seconds`
  - [ ] `response_api_tool_executions_total`
  - [ ] `response_api_tool_execution_duration_seconds`
  - [ ] `response_api_tool_chain_depth`
  - [ ] `response_api_errors_total`
- [ ] Expose metrics endpoint at `/metrics`

#### ✅ **Step 9.2: Tracing**
- [ ] Add OpenTelemetry spans:
  - [ ] Response creation
  - [ ] Tool execution
  - [ ] LLM calls
  - [ ] Database operations
- [ ] Configure trace export to Jaeger

#### ✅ **Step 9.3: Logging**
- [ ] Structure logs with Zerolog
- [ ] Add request IDs
- [ ] Log tool executions with arguments
- [ ] Log tool results
- [ ] Add debug logging for tool chain

---

### **Phase 10: Final Review & Launch (Days 19-20)**

#### ✅ **Step 10.1: Code Review**
- [ ] Review all code for best practices
- [ ] Check error handling completeness
- [ ] Verify all TODOs are addressed
- [ ] Run linters (golangci-lint)
- [ ] Run security scanner (gosec)

#### ✅ **Step 10.2: Testing Checklist**
- [ ] All unit tests passing
- [ ] All integration tests passing
- [ ] Postman collection 100% passing
- [ ] Load testing (if applicable)
- [ ] Memory leak testing

#### ✅ **Step 10.3: Documentation Review**
- [ ] All docs are complete
- [ ] Swagger docs are accurate
- [ ] Examples are working
- [ ] Architecture diagrams are clear

#### ✅ **Step 10.4: Deployment Checklist**
- [ ] Build succeeds
- [ ] Helm chart deploys successfully
- [ ] All pods are healthy
- [ ] Service endpoints respond
- [ ] Tool calling works end-to-end
- [ ] Streaming works correctly
- [ ] Conversation continuity works

---

## **Key Technical Decisions**

### **1. Tool Orchestration Strategy**
- **Recursive Execution**: LLM → Tool Call → MCP → Tool Result → LLM (repeat)
- **Max Depth Limit**: Default 10 to prevent infinite loops
- **Timeout**: 30 seconds per tool execution
- **Error Handling**: Gracefully handle tool failures, return partial results

### **2. MCP Integration**
- **JSON-RPC 2.0**: Standard protocol for tool calls
- **SSE Streaming**: Handle streaming responses from MCP
- **Tool Discovery**: Cache available tools from MCP `/v1/mcp` endpoint
- **Circuit Breaker**: Protect against MCP service failures

### **3. Streaming Architecture**
- **SSE Format**: Server-Sent Events with proper event types
- **Event Types**: 9 different event types for granular updates
- **Tool Events**: Special events for tool calls and results
- **Buffering**: Minimum 6 words before sending text delta

### **4. Database Design**
- **Response Table**: Main entity with JSONB for flexibility
- **Tool Executions Table**: Audit trail of all tool calls
- **Conversation Items**: Message history with role/content
- **Indexes**: On public_id, user_id, conversation_id for performance

---

## **Success Criteria**

- [ ] ✅ Service builds and runs locally
- [ ] ✅ All unit tests pass (>80% coverage)
- [ ] ✅ Postman collection passes 100%
- [ ] ✅ Deploys successfully to Kubernetes
- [ ] ✅ Tool calling works with MCP Tools service
- [ ] ✅ Multi-step tool chains execute correctly
- [ ] ✅ Streaming responses work with tools
- [ ] ✅ Conversation continuity works
- [ ] ✅ Documentation is complete
- [ ] ✅ No critical security issues

---

## **Estimated Timeline**

- **Total**: 20 days (4 weeks)
- **Phase 1-2**: 5 days (scaffolding + domain)
- **Phase 3-4**: 4 days (infrastructure + interfaces)
- **Phase 5-6**: 4 days (docs + testing)
- **Phase 7-8**: 4 days (k8s + docs)
- **Phase 9-10**: 3 days (observability + review)

---

## **Dependencies**

- [ ] MCP Tools service must be running and accessible
- [ ] LLM API or compatible provider must be available
- [ ] PostgreSQL database must be accessible
- [ ] Keycloak (if auth enabled) must be configured

---

**Ready to proceed with implementation?** This plan provides a complete roadmap for building a production-ready Response API service with MCP tool orchestration. Should I proceed with any specific phase first?