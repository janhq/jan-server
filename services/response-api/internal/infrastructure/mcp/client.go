package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"

	"jan-server/services/response-api/internal/domain/llm"
	"jan-server/services/response-api/internal/domain/tool"
)

// Client implements tool.MCPClient.
type Client struct {
	httpClient *resty.Client
}

// NewClient constructs the MCP client.
func NewClient(baseURL string) *Client {
	return &Client{
		httpClient: resty.New().
			SetBaseURL(baseURL).
			SetHeader("Content-Type", "application/json"),
	}
}

// setAuthHeader adds X-API-Key header to the request if a token is available in context.
// Also sets X-JAN-SRC header to identify requests from response-api.
func setAuthHeader(ctx context.Context, req *resty.Request) {
	// Mark request as coming from response-api so downstream services can skip tracking
	req.SetHeader("X-JAN-SRC", "RESPONSE")

	token := strings.TrimSpace(llm.AuthTokenFromContext(ctx))
	if token == "" {
		return
	}
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		req.SetHeader("Authorization", token)
		return
	}
	if looksLikeJWT(token) {
		req.SetHeader("Authorization", "Bearer "+token)
		return
	}
	req.SetHeader("X-API-Key", token)
}

func looksLikeJWT(token string) bool {
	if strings.HasPrefix(token, "sk_") {
		return false
	}
	return strings.Count(token, ".") == 2
}

// parseSSEorJSON extracts JSON from SSE format or returns body as-is if already JSON.
// Supports multiple SSE formats:
//   - Plain JSON: {"jsonrpc":...}
//   - SSE with data prefix: "event: message\ndata: {...}\n\n"
//   - SSE data-only: "data: {...}\n\n"
//
// Returns the first valid JSON object found.
func parseSSEorJSON(body []byte) ([]byte, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	bodyStr := string(body)
	trimmed := strings.TrimSpace(bodyStr)

	// If it starts with '{' or '[', it's already JSON
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return body, nil
	}

	// Parse SSE format - look for "data: " lines and collect all JSON data
	// Some SSE implementations may have multiple data lines for a single event
	scanner := bufio.NewScanner(strings.NewReader(bodyStr))

	// Increase scanner buffer for large payloads (e.g., scrape responses)
	const maxTokenSize = 10 * 1024 * 1024 // 10MB
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxTokenSize)

	var jsonData strings.Builder
	var foundData bool

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data != "" {
				jsonData.WriteString(data)
				foundData = true
			}
		} else if strings.HasPrefix(line, "data:") {
			// Handle "data:" without space
			data := strings.TrimPrefix(line, "data:")
			if data != "" {
				jsonData.WriteString(data)
				foundData = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning SSE response: %w", err)
	}

	if !foundData {
		// Log first 500 chars of response for debugging
		preview := bodyStr
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		log.Debug().
			Str("response_preview", preview).
			Int("response_length", len(bodyStr)).
			Msg("SSE parsing failed - no data field found")

		return nil, fmt.Errorf("no JSON data found in SSE response (len=%d)", len(bodyStr))
	}

	return []byte(jsonData.String()), nil
}

// ListTools fetches the tools via JSON-RPC call tools/list.
func (c *Client) ListTools(ctx context.Context) ([]tool.MCPTool, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/list",
		"params":  map[string]interface{}{},
		"id":      1,
	}

	req := c.httpClient.R().
		SetContext(ctx).
		SetBody(payload)

	// Forward auth token from context to MCP service
	setAuthHeader(ctx, req)

	resp, err := req.Post("/v1/mcp")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("mcp list tools error: %s", resp.String())
	}

	// Parse SSE or JSON response
	jsonBody, err := parseSSEorJSON(resp.Body())
	if err != nil {
		return nil, fmt.Errorf("failed to parse MCP response: %w", err)
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(jsonBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}

	var result struct {
		Tools []tool.MCPTool `json:"tools"`
	}
	if err := json.Unmarshal(rpcResp.Result, &result); err != nil {
		return nil, err
	}
	return result.Tools, nil
}

// CallTool triggers a tool execution via JSON-RPC tools/call.
func (c *Client) CallTool(ctx context.Context, req tool.CallRequest) (*tool.Result, error) {
	rpcID := req.ToolCallID
	if strings.TrimSpace(rpcID) == "" {
		rpcID = req.Name
	}

	log.Info().
		Str("tool", req.Name).
		Str("tool_call_id", req.ToolCallID).
		Str("request_id", req.RequestID).
		Str("user_id", req.UserID).
		Msg("Calling MCP tool")

	// Build params with arguments and _meta for context (MCP standard way to pass metadata)
	params := map[string]interface{}{
		"name":      req.Name,
		"arguments": req.Arguments,
	}

	// Add context IDs to _meta field (not merged into arguments)
	// Note: conversation_id is NOT included - it's internal to response-api
	meta := buildMetaContext(req.RequestID, req.UserID, req.ToolCallID)
	if len(meta) > 0 {
		params["_meta"] = meta
	}

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/call",
		"params":  params,
		"id":      rpcID,
	}

	httpReq := c.httpClient.R().
		SetContext(ctx).
		SetBody(payload)

	// Forward auth token from context to MCP service
	setAuthHeader(ctx, httpReq)

	resp, err := httpReq.Post("/v1/mcp")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("mcp call error: %s", resp.String())
	}

	// Parse SSE or JSON response
	jsonBody, err := parseSSEorJSON(resp.Body())
	if err != nil {
		return nil, fmt.Errorf("failed to parse MCP response: %w", err)
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(jsonBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}

	var result struct {
		Content []tool.MCPContent `json:"content"`
		IsError bool              `json:"isError"`
		Error   string            `json:"error"`
	}
	if err := json.Unmarshal(rpcResp.Result, &result); err != nil {
		return nil, err
	}

	return &tool.Result{
		ToolName: req.Name,
		Content:  result.Content,
		IsError:  result.IsError,
		Error:    result.Error,
	}, nil
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
	ID      interface{}     `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (r *rpcError) Error() string {
	return fmt.Sprintf("mcp error (%d): %s", r.Code, r.Message)
}

// buildMetaContext creates metadata for MCP tool calls.
// Note: conversation_id is intentionally NOT included as it's internal to response-api.
// Downstream services should not track or store conversation context from response-api calls.
func buildMetaContext(requestID, userID, toolCallID string) map[string]interface{} {
	meta := make(map[string]interface{})

	if strings.TrimSpace(requestID) != "" {
		meta["request_id"] = requestID
	}
	// conversation_id is NOT sent - it's internal to response-api
	if strings.TrimSpace(userID) != "" {
		meta["user_id"] = userID
	}
	if strings.TrimSpace(toolCallID) != "" {
		meta["tool_call_id"] = toolCallID
	}

	return meta
}
