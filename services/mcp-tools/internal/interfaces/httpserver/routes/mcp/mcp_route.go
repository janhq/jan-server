package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"

	sandboxdomain "jan-server/services/mcp-tools/internal/domain/sandbox"
	"jan-server/services/mcp-tools/internal/infrastructure/llmapi"
	"jan-server/services/mcp-tools/internal/infrastructure/metrics"
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

	// Logging
	"logging/setLevel": true,
}

type MCPRoute struct {
	searchMCP         *SearchMCP
	providerMCP       *ProviderMCP
	sandboxFusionMCP  *SandboxFusionMCP
	memoryMCP         *MemoryMCP
	imageMCP          *ImageGenerateMCP
	imageEditMCP      *ImageEditMCP
	sandboxMCP        *SandboxMCP        // Unified sandbox provider (AIO or E2B)
	sandboxManagement *SandboxManagement // Sandbox lifecycle management (E2B only)
	agentProxyMCP     *AgentProxyMCP
	llmClient         *llmapi.Client        // LLM-API client for tool call tracking
	toolConfigCache   *toolconfig.Cache     // Cache for dynamic tool descriptions
	sandboxManager    sandboxdomain.Manager // For E2B sandbox state checks
	sandboxProvider   string                // "aio", "e2b", or ""
	mcpServer         *mcp.Server
	httpHandler       http.Handler
}

type toolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

func NewMCPRoute(
	searchMCP *SearchMCP,
	providerMCP *ProviderMCP,
	sandboxFusionMCP *SandboxFusionMCP,
	memoryMCP *MemoryMCP,
	imageMCP *ImageGenerateMCP,
	imageEditMCP *ImageEditMCP,
	sandboxMCP *SandboxMCP,
	sandboxManagement *SandboxManagement,
	agentProxyMCP *AgentProxyMCP,
	llmClient *llmapi.Client,
	toolConfigCache *toolconfig.Cache,
	sandboxManager sandboxdomain.Manager,
	sandboxProvider string,
) *MCPRoute {
	impl := &mcp.Implementation{
		Name:    "menlo-platform",
		Version: "1.0.0",
	}
	server := mcp.NewServer(impl, nil)

	// Pass LLM client to tool handlers for tracking
	searchMCP.SetLLMClient(llmClient)

	if sandboxFusionMCP != nil {
		sandboxFusionMCP.SetLLMClient(llmClient)
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

	if sandboxFusionMCP != nil {
		sandboxFusionMCP.RegisterTools(server)
	}

	// Register memory tools
	if memoryMCP != nil {
		memoryMCP.RegisterTools(server)
	}

	// Register unified sandbox tools (AIO or E2B provider)
	if sandboxMCP != nil {
		sandboxMCP.SetLLMClient(llmClient)
		sandboxMCP.RegisterTools(server)
	}

	// Register sandbox management tools (E2B only)
	if sandboxManagement != nil && sandboxManagement.IsEnabled() {
		sandboxManagement.RegisterTools(server)
	}

	// Register unified run_agent tool for agent execution
	if agentProxyMCP != nil {
		agentProxyMCP.SetLLMClient(llmClient)
		agentProxyMCP.RegisterTools(server)
	}

	// Register tools from external MCP providers
	if providerMCP != nil {
		if err := providerMCP.RegisterTools(server); err != nil {
			// Log error but continue
			// (error already logged in RegisterTools)
		}
	}

	return &MCPRoute{
		searchMCP:         searchMCP,
		providerMCP:       providerMCP,
		sandboxFusionMCP:  sandboxFusionMCP,
		memoryMCP:         memoryMCP,
		imageMCP:          imageMCP,
		imageEditMCP:      imageEditMCP,
		sandboxMCP:        sandboxMCP,
		sandboxManagement: sandboxManagement,
		agentProxyMCP:     agentProxyMCP,
		llmClient:         llmClient,
		toolConfigCache:   toolConfigCache,
		sandboxManager:    sandboxManager,
		sandboxProvider:   sandboxProvider,
		mcpServer:         server,
		httpHandler: mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
			return server
		}, &mcp.StreamableHTTPOptions{
			Stateless:    true,
			JSONResponse: true, // Return plain JSON instead of SSE for simpler client parsing
		}),
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
	// Read and parse the request body to check the method
	bodyBytes, err := io.ReadAll(reqCtx.Request.Body)
	if err == nil && len(bodyBytes) > 0 {
		// Restore body for potential re-use
		reqCtx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var payload struct {
			Method string                 `json:"method"`
			ID     interface{}            `json:"id"`
			Params map[string]interface{} `json:"params,omitempty"`
		}
		if json.Unmarshal(bodyBytes, &payload) == nil {
			// Handle tools/list with dynamic filtering
			if payload.Method == "tools/list" {
				route.handleToolsListWithDynamicDescriptions(reqCtx, payload.ID)
				return
			}

			// Handle tools/call for sandbox_browser_* tools (dynamic tools from E2B sandbox)
			if payload.Method == "tools/call" && payload.Params != nil {
				if toolName, ok := payload.Params["name"].(string); ok {
					if toolName == "sandbox_browser" || strings.HasPrefix(toolName, "sandbox_browser_") {
						route.handleDynamicBrowserToolCall(reqCtx, payload.ID, toolName, payload.Params)
						return
					}
				}
			}
		}
	}

	// Force acceptable content types for go-sdk streamable handler even if client omits Accept.
	reqCtx.Request.Header.Set("Accept", "application/json, text/event-stream")
	route.httpHandler.ServeHTTP(reqCtx.Writer, reqCtx.Request)
}

// handleToolsListWithDynamicDescriptions handles tools/list with descriptions from the cache
// and filters tools based on context (x-conversation-id, sandbox state)
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
				}
			}
		}
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
			Tools      []toolInfo `json:"tools"`
			NextCursor string     `json:"nextCursor,omitempty"`
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

	// --- DYNAMIC TOOL FILTERING ---
	// Filter and modify tools based on context
	filteredTools := route.filterToolsForContext(ctx, rpcResponse.Result.Tools)

	// Override descriptions from cache for remaining tools
	for i := range filteredTools {
		toolName := filteredTools[i].Name
		if desc, ok := descriptionMap[toolName]; ok && desc != "" {
			filteredTools[i].Description = desc
		}
	}

	// Update response with filtered tools
	rpcResponse.Result.Tools = filteredTools

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

// sandboxToolPrefixes are prefixes that identify sandbox-related tools
var sandboxToolPrefixes = []string{
	"sandbox_", // sandbox_shell_exec, sandbox_file_read, sandbox_start, sandbox_browser_*, etc.
	"shell_",   // shell_exec (legacy)
	"file_",    // file_read, file_write, file_list (legacy)
	"code_",    // code_execute (legacy)
	"install_", // install_packages (legacy)
}

// isSandboxTool returns true if the tool name is a sandbox-related tool
func isSandboxTool(name string) bool {
	for _, prefix := range sandboxToolPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// filterToolsForContext filters tools based on context:
// - No x-conversation-id: Remove ALL sandbox tools
// - AIO + x-conversation-id: Include all sandbox execution tools (no management)
// - E2B + sandbox NOT running: Include only management tools
// - E2B + sandbox running: Include all tools + dynamic browser tools
func (route *MCPRoute) filterToolsForContext(ctx context.Context, tools []toolInfo) []toolInfo {
	// Get conversation_id from X-Conversation-ID header
	// Sandbox tool visibility only requires X-Conversation-ID, not X-Tool-Call-ID
	conversationID := GetConversationID(ctx)

	userID := ""
	if val := ctx.Value("user_id"); val != nil {
		if str, ok := val.(string); ok {
			userID = str
		}
	}

	log.Debug().
		Str("conversation_id", conversationID).
		Str("user_id", userID).
		Str("sandbox_provider", route.sandboxProvider).
		Int("total_tools", len(tools)).
		Msg("Filtering tools for context")

	// If no conversation_id, filter out ALL sandbox tools
	if conversationID == "" {
		var filtered []toolInfo
		for _, tool := range tools {
			if !isSandboxTool(tool.Name) {
				filtered = append(filtered, tool)
			}
		}
		log.Debug().Int("filtered_count", len(filtered)).Msg("No conversation_id - removed sandbox tools")
		return dedupeTools(filtered)
	}

	// Handle based on sandbox provider
	switch route.sandboxProvider {
	case "aio":
		// AIO: Include all sandbox execution tools (no management, no dynamic browser tools)
		var filtered []toolInfo
		for _, tool := range tools {
			// Skip management tools (AIO doesn't support them)
			if IsSandboxManagementTool(tool.Name) {
				continue
			}
			// Skip sandbox_browser tools (AIO doesn't have dynamic browser tools from search-mcp-server)
			if tool.Name == "sandbox_browser" || strings.HasPrefix(tool.Name, "sandbox_browser_") {
				continue
			}
			filtered = append(filtered, tool)
		}
		log.Debug().Int("filtered_count", len(filtered)).Msg("AIO - included execution tools")
		return dedupeTools(filtered)

	case "e2b":
		// E2B: Check if sandbox is running
		sandboxRunning := false
		if route.sandboxManager != nil && userID != "" {
			running, err := route.sandboxManager.IsRunning(ctx, userID)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to check sandbox state, assuming not running")
			} else {
				sandboxRunning = running
			}
		}

		if !sandboxRunning {
			// Sandbox NOT running: Include management tools + non-sandbox tools
			var filtered []toolInfo

			// Add non-sandbox tools from the server response
			for _, tool := range tools {
				if !isSandboxTool(tool.Name) {
					filtered = append(filtered, tool)
				}
			}

			// Explicitly add management tools (they may not be in the MCP server response)
			if route.sandboxManagement != nil && route.sandboxManagement.IsEnabled() {
				for _, mgmtTool := range route.sandboxManagement.GetManagementToolDefinitions() {
					filtered = append(filtered, toolInfo{
						Name:        mgmtTool.Name,
						Description: mgmtTool.Description,
						InputSchema: mgmtTool.InputSchema,
					})
				}
			}

			log.Debug().Int("filtered_count", len(filtered)).Msg("E2B - sandbox not running, management tools + non-sandbox tools")
			return dedupeTools(filtered)
		}

		// Sandbox IS running: Include all tools + sandbox tools + management tools + dynamic browser tools
		var allTools []toolInfo

		// Add all tools from the MCP server response
		allTools = append(allTools, tools...)

		// Explicitly add management tools (they may not be in the MCP server response)
		if route.sandboxManagement != nil && route.sandboxManagement.IsEnabled() {
			for _, mgmtTool := range route.sandboxManagement.GetManagementToolDefinitions() {
				allTools = append(allTools, toolInfo{
					Name:        mgmtTool.Name,
					Description: mgmtTool.Description,
					InputSchema: mgmtTool.InputSchema,
				})
			}
		}

		// Explicitly add sandbox execution tools (they may not be in the MCP server response)
		if route.sandboxMCP != nil {
			for _, sandboxTool := range route.sandboxMCP.GetToolDefinitions() {
				allTools = append(allTools, toolInfo{
					Name:        sandboxTool.Name,
					Description: sandboxTool.Description,
					InputSchema: sandboxTool.InputSchema,
				})
			}
		}

		// Fetch dynamic browser tools from sandbox (from search-mcp-server inside E2B)
		if route.sandboxManager != nil && userID != "" {
			dynamicTools, err := route.sandboxManager.GetDynamicTools(ctx, userID)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to fetch dynamic tools from sandbox")
			} else if len(dynamicTools) > 0 {
				log.Debug().Int("dynamic_count", len(dynamicTools)).Msg("Fetched dynamic tools from sandbox")
				for _, dt := range dynamicTools {
					// Normalize browser tool names to sandbox_browser_*
					normalizedName := toSandboxBrowserToolName(dt.Name)
					allTools = append(allTools, toolInfo{
						Name:        normalizedName,
						Description: dt.Description,
						InputSchema: dt.InputSchema,
					})
				}
			}
		}

		log.Debug().Int("total_count", len(allTools)).Msg("E2B - sandbox running, all tools included")
		return dedupeTools(allTools)

	default:
		// No sandbox provider: Remove all sandbox tools
		var filtered []toolInfo
		for _, tool := range tools {
			if !isSandboxTool(tool.Name) {
				filtered = append(filtered, tool)
			}
		}
		log.Debug().Int("filtered_count", len(filtered)).Msg("No sandbox provider - removed sandbox tools")
		return dedupeTools(filtered)
	}
}

// handleDynamicBrowserToolCall handles sandbox_browser_* tool calls by proxying to e2b-service
func (route *MCPRoute) handleDynamicBrowserToolCall(reqCtx *gin.Context, requestID interface{}, toolName string, params map[string]interface{}) {
	ctx := reqCtx.Request.Context()

	// Extract user_id from context
	userID := ""
	if val := ctx.Value("user_id"); val != nil {
		if str, ok := val.(string); ok {
			userID = str
		}
	}

	if userID == "" {
		route.sendMCPError(reqCtx, requestID, -32600, "user_id is required (from JWT)")
		return
	}

	// Only E2B supports dynamic browser tools
	if route.sandboxProvider != "e2b" || route.sandboxManager == nil {
		route.sendMCPError(reqCtx, requestID, -32601, "sandbox_browser_* tools are only available with E2B sandbox provider")
		return
	}

	// Map sandbox_browser_* to actual browser_* tool name inside sandbox
	actualToolName := fromSandboxBrowserToolName(toolName)

	// Get tool arguments
	args, _ := params["arguments"].(map[string]interface{})
	if args == nil {
		args = make(map[string]interface{})
	}

	log.Info().
		Str("tool", toolName).
		Str("actual_tool", actualToolName).
		Str("user_id", userID).
		Msg("Proxying browser tool call to E2B sandbox")

	// Call e2b-service MCP endpoint directly
	startTime := time.Now()
	result, err := route.callE2BMCPTool(ctx, userID, actualToolName, args)
	status := "success"
	if err != nil {
		status = "error"
	}
	metrics.RecordToolCall(toolName, "e2b", status, time.Since(startTime).Seconds())
	if err != nil {
		log.Error().Err(err).Str("tool", toolName).Msg("Browser tool execution failed")
		route.sendMCPError(reqCtx, requestID, -32603, "tool execution failed: "+err.Error())
		return
	}

	// Return successful result
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      requestID,
		"result":  result,
	}

	reqCtx.Header("Content-Type", "application/json")
	reqCtx.JSON(http.StatusOK, response)
}

// callE2BMCPTool calls a tool via e2b-service MCP endpoint
func (route *MCPRoute) callE2BMCPTool(ctx context.Context, userID, toolName string, args map[string]interface{}) (map[string]interface{}, error) {
	if route.sandboxManager == nil {
		return nil, fmt.Errorf("sandbox manager not available")
	}

	// Use the Manager's CallTool method to execute the tool
	return route.sandboxManager.CallTool(ctx, userID, toolName, args)
}

// sendMCPError sends an MCP JSON-RPC error response
func (route *MCPRoute) sendMCPError(reqCtx *gin.Context, requestID interface{}, code int, message string) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      requestID,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	reqCtx.Header("Content-Type", "application/json")
	reqCtx.JSON(http.StatusOK, response)
}

func dedupeTools(tools []toolInfo) []toolInfo {
	if len(tools) == 0 {
		return tools
	}
	seen := make(map[string]struct{}, len(tools))
	result := make([]toolInfo, 0, len(tools))
	for _, tool := range tools {
		if tool.Name == "" {
			continue
		}
		if _, exists := seen[tool.Name]; exists {
			continue
		}
		seen[tool.Name] = struct{}{}
		result = append(result, tool)
	}
	return result
}

func toSandboxBrowserToolName(actual string) string {
	if strings.HasPrefix(actual, "sandbox_browser_") {
		return actual
	}
	if strings.HasPrefix(actual, "browser_") {
		actual = strings.TrimPrefix(actual, "browser_")
	} else if actual == "browser" {
		actual = ""
	}
	if actual == "" {
		return "sandbox_browser"
	}
	return "sandbox_browser_" + actual
}

func fromSandboxBrowserToolName(display string) string {
	if !strings.HasPrefix(display, "sandbox_browser") {
		return display
	}
	suffix := strings.TrimPrefix(display, "sandbox_browser")
	suffix = strings.TrimPrefix(suffix, "_")
	if suffix == "" {
		return "browser"
	}
	if suffix == "browser" || strings.HasPrefix(suffix, "browser_") {
		return suffix
	}
	return "browser_" + suffix
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

// InjectUserContext extracts user_id from JWT token or API key and injects it into request context
func InjectUserContext() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		var userID string

		// First, check if user_id was already set by API key validation (in gin context)
		if uid, exists := reqCtx.Get("user_id"); exists {
			if str, ok := uid.(string); ok && str != "" {
				userID = str
			}
		}

		// If not set by API key, try to extract from JWT token
		if userID == "" {
			if tokenVal, exists := reqCtx.Get("auth_token"); exists {
				if token, ok := tokenVal.(*jwt.Token); ok && token.Valid {
					if claims, ok := token.Claims.(jwt.MapClaims); ok {
						// Try to extract user_id from various claim fields
						if sub, ok := claims["sub"].(string); ok && sub != "" {
							userID = sub
						} else if uid, ok := claims["user_id"].(string); ok && uid != "" {
							userID = uid
						} else if uid, ok := claims["uid"].(string); ok && uid != "" {
							userID = uid
						}
					}
				}
			}
		}

		// Inject user_id into request context (required by sandbox tools)
		if userID != "" {
			ctx := context.WithValue(reqCtx.Request.Context(), "user_id", userID)
			reqCtx.Request = reqCtx.Request.WithContext(ctx)
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
