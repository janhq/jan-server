package routes

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"jan-server/services/mcp-tools/interfaces/httpserver/responses"
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
	serperMCP   *SerperMCP
	mcpServer   *mcpserver.MCPServer
	httpHandler http.Handler
}

func NewMCPRoute(
	serperMCP *SerperMCP,
) *MCPRoute {
	server := mcpserver.NewMCPServer("menlo-platform", "1.0.0",
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithRecovery(),
	)

	serperMCP.RegisterTools(server)

	return &MCPRoute{
		serperMCP:   serperMCP,
		mcpServer:   server,
		httpHandler: mcpserver.NewStreamableHTTPServer(server, mcpserver.WithStateLess(true)),
	}
}

func (route *MCPRoute) RegisterRouter(router *gin.RouterGroup) {
	router.POST("/mcp",
		MCPMethodGuard(allowedMCPMethods),
		route.serveMCP,
	)
}

// serveMCP streams Model Context Protocol responses using the underlying MCP server.
// @Summary MCP streamable endpoint
// @Description Handles Model Context Protocol (MCP) requests over an HTTP stream. The response is sent as a continuous stream of data.
// @Tags Chat Completions API
// @Security BearerAuth
// @Accept json
// @Produce text/event-stream
// @Param request body any true "MCP request payload"
// @Success 200 {string} string "Streamed response (SSE or chunked transfer)"
// @Failure 400 {object} responses.ErrorResponse "Invalid MCP request payload"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Router /v1/mcp [post]
func (route *MCPRoute) serveMCP(reqCtx *gin.Context) {
	route.httpHandler.ServeHTTP(reqCtx.Writer, reqCtx.Request)
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
