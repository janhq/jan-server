package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"

	"jan-server/services/mcp-tools/internal/infrastructure/llmapi"
	"jan-server/services/mcp-tools/internal/infrastructure/toolconfig"
	"jan-server/services/mcp-tools/internal/interfaces/httpserver/responses"
	"jan-server/services/mcp-tools/utils/platformerrors"
)

var allowedMCPMethods = map[string]bool{
	// Initialization / handshake
	"initialize":                true,
	"notifications/initialized": true,
	"ping":                      true,

	// Tools
	"tools/list": true,
	"tools/call": true,

	// Prompts
	"prompts/list": true,
	"prompts/call": true,

	// Resources
	"resources/list":           true,
	"resources/templates/list": true,
	"resources/read":           true,
	"resources/subscribe":      true,
}

type MCPRoute struct {
	searchMCP       *SearchMCP
	providerMCP     *ProviderMCP
	sandboxMCP      *SandboxFusionMCP
	memoryMCP       *MemoryMCP
	imageMCP        *ImageGenerateMCP
	imageEditMCP    *ImageEditMCP
	aioMCP          *AIOMCP
	llmClient       *llmapi.Client    // LLM-API client for tool call tracking
	toolConfigCache *toolconfig.Cache // Cache for dynamic tool descriptions
	mcpServer       *mcp.Server
	httpHandler     http.Handler
}

func NewMCPRoute(
	searchMCP *SearchMCP,
	providerMCP *ProviderMCP,
	sandboxMCP *SandboxFusionMCP,
	memoryMCP *MemoryMCP,
	imageMCP *ImageGenerateMCP,
	imageEditMCP *ImageEditMCP,
	aioMCP *AIOMCP,
	llmClient *llmapi.Client,
	toolConfigCache *toolconfig.Cache,
) *MCPRoute {
	impl := &mcp.Implementation{
		Name:    "menlo-platform",
		Version: "1.0.0",
	}
	server := mcp.NewServer(impl, nil)

	// Pass LLM client to tool handlers for tracking
	searchMCP.SetLLMClient(llmClient)

	if sandboxMCP != nil {
		sandboxMCP.SetLLMClient(llmClient)
	}

	// Register memory tools
	if memoryMCP != nil {
		memoryMCP.SetLLMClient(llmClient)
	}

	searchMCP.RegisterTools(server)
	if imageMCP != nil {
		imageMCP.RegisterTools(server)
	}
	if imageEditMCP != nil {
		imageEditMCP.RegisterTools(server)
	}

	if sandboxMCP != nil {
		sandboxMCP.RegisterTools(server)
	}

	// Register memory tools
	if memoryMCP != nil {
		memoryMCP.RegisterTools(server)
	}

	// Register AIO Sandbox tools
	if aioMCP != nil {
		aioMCP.RegisterTools(server)
	}

	// Register tools from external MCP providers
	if providerMCP != nil {
		if err := providerMCP.RegisterTools(server); err != nil {
			// Log error but continue
			// (error already logged in RegisterTools)
		}
	}

	return &MCPRoute{
		searchMCP:       searchMCP,
		providerMCP:     providerMCP,
		sandboxMCP:      sandboxMCP,
		memoryMCP:       memoryMCP,
		imageMCP:        imageMCP,
		imageEditMCP:    imageEditMCP,
		aioMCP:          aioMCP,
		llmClient:       llmClient,
		toolConfigCache: toolConfigCache,
		mcpServer:       server,
		httpHandler: mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
			return server
		}, &mcp.StreamableHTTPOptions{Stateless: true}),
	}
}

func (route *MCPRoute) RegisterRouter(router *gin.RouterGroup) {
	router.POST("/mcp",
		MCPMethodGuard(allowedMCPMethods),
		InjectUserContext(),
		ExtractToolTracking(), // Extract tracking headers for tool call tracking
		route.serveMCP,
	)
}

// serveMCP streams Model Context Protocol responses using the underlying MCP server.
// @Summary MCP endpoint for tool execution
// @Description Handles Model Context Protocol (MCP) requests over HTTP. Supports MCP methods: initialize, ping, tools/list, tools/call, prompts/list, prompts/call, resources/list, resources/read.
// @Description
// @Description **Available Tools:**
// @Description - `google_search`: Web search via pluggable engines (Serper/SearXNG) with params: q, gl, hl, location, num, tbs, page, autocorrect, domain_allow_list, location_hint, offline_mode. Returns structured citations.
// @Description - `scrape`: Web page scraping (params: url, includeMarkdown) returning text, preview, cache_status, and metadata.
// @Description - `file_search_index` / `file_search_query`: Index arbitrary text and run similarity queries against the lightweight vector store.
// @Description - `python_exec`: Execute trusted code through SandboxFusion (params: code, language, session_id, approved) to retrieve stdout/stderr/artifacts.
// @Description - `memory_retrieve`: Retrieve relevant user preferences, project context, or conversation history (params: query, user_id, project_id, max_user_items, max_project_items, min_similarity). Returns personalized context.
// @Description - `generate_image`: Generate images from a text prompt via LLM API /v1/images/generations (params: prompt, size, n, num_inference_steps, cfg_scale).
// @Description - `edit_image`: Edit images with a prompt + input image via LLM API /v1/images/edits (params: prompt, image, mask, size, strength, steps, seed, cfg_scale).
// @Description
// @Description **MCP Protocol:**
// @Description - Request format: JSON-RPC 2.0 with method and params
// @Description - Response format: Server-Sent Events (SSE) stream
// @Description - Stateless mode (no session management)
// @Tags MCP API
// @Accept json
// @Produce text/event-stream
// @Param request body object true "MCP JSON-RPC request payload (e.g., {\"jsonrpc\":\"2.0\",\"method\":\"tools/list\",\"id\":1})"
// @Success 200 {string} string "Streamed MCP response in SSE format"
// @Failure 400 {object} responses.ErrorResponse "Invalid MCP request payload or unsupported method"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/mcp [post]
func (route *MCPRoute) serveMCP(reqCtx *gin.Context) {
	// Check if this is a tools/list request and intercept it to provide dynamic descriptions
	if route.toolConfigCache != nil {
		// Read body to check method
		bodyBytes, err := io.ReadAll(reqCtx.Request.Body)
		if err == nil && len(bodyBytes) > 0 {
			// Restore body for potential re-use
			reqCtx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			var payload struct {
				Method string      `json:"method"`
				ID     interface{} `json:"id"`
			}
			if json.Unmarshal(bodyBytes, &payload) == nil && payload.Method == "tools/list" {
				route.handleToolsListWithDynamicDescriptions(reqCtx, payload.ID)
				return
			}
		}
	}

	// Force acceptable content types for go-sdk streamable handler even if client omits Accept.
	reqCtx.Request.Header.Set("Accept", "application/json, text/event-stream")
	route.httpHandler.ServeHTTP(reqCtx.Writer, reqCtx.Request)
}

// handleToolsListWithDynamicDescriptions handles tools/list with descriptions from the cache
func (route *MCPRoute) handleToolsListWithDynamicDescriptions(reqCtx *gin.Context, requestID interface{}) {
	ctx := reqCtx.Request.Context()

	// Get descriptions from cache
	descriptionMap := make(map[string]string)
	if route.toolConfigCache != nil {
		tools, err := route.toolConfigCache.GetAllTools(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to get tools from config cache")
		} else {
			log.Debug().Int("tool_count", len(tools)).Msg("Fetched tools from cache for description override")
			for _, tool := range tools {
				if tool.Config.Description != "" {
					descriptionMap[tool.Config.ToolKey] = tool.Config.Description
					log.Debug().
						Str("tool_key", tool.Config.ToolKey).
						Str("description", tool.Config.Description).
						Msg("Loaded description from cache")
				}
			}
		}
	} else {
		log.Debug().Msg("Tool config cache is nil, skipping description override")
	}

	// Get the base response from the MCP server by calling it directly
	// We need to use a custom response writer to capture the response
	captureWriter := &responseCapture{header: make(http.Header)}
	captureReq := reqCtx.Request.Clone(ctx)

	// Restore body for the SDK handler
	bodyBytes, _ := io.ReadAll(reqCtx.Request.Body)
	captureReq.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	reqCtx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	captureReq.Header.Set("Accept", "application/json, text/event-stream")
	route.httpHandler.ServeHTTP(captureWriter, captureReq)

	// Parse and modify the response
	responseBody := captureWriter.body.Bytes()

	// The response might be in SSE format (event: message\ndata: {...}\n\n)
	// or plain JSON. Try to extract JSON from SSE format first.
	jsonData := extractJSONFromSSE(responseBody)
	if jsonData == nil {
		jsonData = responseBody // Assume it's plain JSON
	}

	// For JSON-RPC over HTTP, the response format is: {"jsonrpc":"2.0","id":...,"result":{"tools":[...]}}
	var rpcResponse struct {
		Jsonrpc string      `json:"jsonrpc"`
		ID      interface{} `json:"id"`
		Result  struct {
			Tools []struct {
				Name        string                 `json:"name"`
				Description string                 `json:"description"`
				InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
			} `json:"tools"`
			NextCursor string `json:"nextCursor,omitempty"`
		} `json:"result"`
		Error interface{} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(jsonData, &rpcResponse); err != nil {
		// If parsing fails, just forward the original response
		log.Warn().Err(err).Str("response_preview", string(responseBody[:min(200, len(responseBody))])).Msg("Failed to parse tools/list response for description override")
		for k, v := range captureWriter.header {
			reqCtx.Writer.Header()[k] = v
		}
		reqCtx.Writer.WriteHeader(captureWriter.statusCode)
		reqCtx.Writer.Write(responseBody)
		return
	}

	// Override descriptions from cache
	for i := range rpcResponse.Result.Tools {
		toolName := rpcResponse.Result.Tools[i].Name
		if desc, ok := descriptionMap[toolName]; ok && desc != "" {
			log.Debug().
				Str("tool_name", toolName).
				Str("old_desc", rpcResponse.Result.Tools[i].Description[:min(50, len(rpcResponse.Result.Tools[i].Description))]).
				Str("new_desc", desc[:min(50, len(desc))]).
				Msg("Overriding tool description from cache")
			rpcResponse.Result.Tools[i].Description = desc
		} else {
			log.Debug().
				Str("tool_name", toolName).
				Bool("found_in_map", ok).
				Msg("No description override for tool")
		}
	}

	// Send modified response
	modifiedBody, err := json.Marshal(rpcResponse)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to marshal modified tools/list response")
		for k, v := range captureWriter.header {
			reqCtx.Writer.Header()[k] = v
		}
		reqCtx.Writer.WriteHeader(captureWriter.statusCode)
		reqCtx.Writer.Write(responseBody)
		return
	}

	reqCtx.Writer.Header().Set("Content-Type", "application/json")
	reqCtx.Writer.WriteHeader(http.StatusOK)
	reqCtx.Writer.Write(modifiedBody)
}

// responseCapture captures HTTP response for modification
type responseCapture struct {
	header     http.Header
	body       bytes.Buffer
	statusCode int
}

func (r *responseCapture) Header() http.Header {
	return r.header
}

func (r *responseCapture) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *responseCapture) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

// extractJSONFromSSE extracts JSON data from SSE (Server-Sent Events) format.
// SSE format is: "event: message\ndata: {...}\n\n"
// Returns nil if the input is not in SSE format.
func extractJSONFromSSE(data []byte) []byte {
	str := string(data)

	// Check if this looks like SSE format
	if !strings.HasPrefix(str, "event:") && !strings.HasPrefix(str, "data:") {
		return nil
	}

	// Split by newlines and find the data line
	lines := strings.Split(str, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			jsonStr := strings.TrimPrefix(line, "data:")
			jsonStr = strings.TrimSpace(jsonStr)
			if jsonStr != "" {
				return []byte(jsonStr)
			}
		}
	}

	return nil
}

// InjectUserContext extracts user_id from JWT token and injects it into request context
func InjectUserContext() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		// Try to get auth token from gin context (set by auth middleware)
		if tokenVal, exists := reqCtx.Get("auth_token"); exists {
			if token, ok := tokenVal.(*jwt.Token); ok && token.Valid {
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					// Try to extract user_id from various claim fields
					var userID string
					if sub, ok := claims["sub"].(string); ok && sub != "" {
						userID = sub
					} else if uid, ok := claims["user_id"].(string); ok && uid != "" {
						userID = uid
					} else if uid, ok := claims["uid"].(string); ok && uid != "" {
						userID = uid
					}

					if userID != "" {
						// Inject user_id into request context
						ctx := context.WithValue(reqCtx.Request.Context(), "user_id", userID)
						reqCtx.Request = reqCtx.Request.WithContext(ctx)
					}
				}
			}
		}
		reqCtx.Next()
	}
}

func MCPMethodGuard(allowedMethods map[string]bool) gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		bodyBytes, err := io.ReadAll(reqCtx.Request.Body)
		if err != nil {
			responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "failed to read MCP request body", "f10df80f-1651-4faa-8a75-3d91814d7990")
			return
		}
		_ = reqCtx.Request.Body.Close()

		if len(bodyBytes) == 0 {
			responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "empty MCP request body", "abf862e2-f2a8-4bd7-b1b7-56fc16647759")
			return
		}

		reqCtx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var payload struct {
			Method string `json:"method"`
		}

		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "invalid MCP request payload", "81f2eaae-8aa1-4569-95ec-c7a611fda0d0")
			return
		}

		if payload.Method == "" {
			responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "missing method field in MCP request", "7b3c9e5a-2f4d-4a1e-9c8b-1d5f3e7a9b2c")
			return
		}

		if !allowedMethods[payload.Method] {
			responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "unsupported MCP method: "+payload.Method, "6e5f62bb-a0fb-4146-969b-7d6dd1bbe8d6")
			return
		}

		reqCtx.Next()
	}
}
