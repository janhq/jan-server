package mcpprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// MCPRequest represents a generic MCP JSON-RPC request
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

// MCPResponse represents a generic MCP JSON-RPC response
type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Bridge handles communication with external MCP providers
type Bridge struct {
	provider   Provider
	httpClient *http.Client
	sessionID  string // MCP session ID for stateful connections
}

// NewBridge creates a new MCP provider bridge
func NewBridge(provider Provider) *Bridge {
	timeout := provider.TimeoutDuration()

	return &Bridge{
		provider: provider,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ListTools retrieves the list of tools from an MCP provider
func (b *Bridge) ListTools(ctx context.Context) (json.RawMessage, error) {
	req := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
	}

	resp, err := b.sendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools from %s: %w", b.provider.Name, err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("MCP error from %s: %s", b.provider.Name, resp.Error.Message)
	}

	return resp.Result, nil
}

// CallTool forwards a tool call to an MCP provider
func (b *Bridge) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (json.RawMessage, error) {
	call := func() (json.RawMessage, error) {
		params := map[string]interface{}{
			"name":      toolName,
			"arguments": arguments,
		}

		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tool call params: %w", err)
		}

		req := MCPRequest{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params:  paramsJSON,
			ID:      time.Now().UnixNano(), // Use timestamp as unique ID
		}

		resp, err := b.sendRequest(ctx, req)
		if err != nil {
			return nil, err
		}

		if resp.Error != nil {
			return nil, fmt.Errorf("MCP error calling %s on %s: %s", toolName, b.provider.Name, resp.Error.Message)
		}

		return resp.Result, nil
	}

	result, err := call()
	if err == nil {
		return result, nil
	}

	if b.shouldReinitialize(err) {
		log.Warn().
			Err(err).
			Str("provider", b.provider.Name).
			Msg("Provider session invalid, reinitializing MCP bridge")
		if initErr := b.Initialize(ctx); initErr == nil {
			return call()
		}
	}

	return nil, fmt.Errorf("failed to call tool %s on %s: %w", toolName, b.provider.Name, err)
}

func (b *Bridge) shouldReinitialize(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if strings.Contains(strings.ToLower(msg), "session not found") {
		return true
	}
	if strings.Contains(msg, "HTTP 404") {
		return true
	}
	return false
}

// CallTool forwards a tool call to an MCP provider

// Initialize sends an initialize request to the MCP provider
func (b *Bridge) Initialize(ctx context.Context) error {
	req := MCPRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"mcp-tools","version":"1.0.0"}}`),
		ID:      0,
	}

	resp, sessionID, err := b.sendRequestWithSession(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to initialize %s: %w", b.provider.Name, err)
	}

	if resp.Error != nil {
		return fmt.Errorf("MCP initialization error from %s: %s", b.provider.Name, resp.Error.Message)
	}

	// Store session ID for subsequent requests
	if sessionID != "" {
		b.sessionID = sessionID
		log.Debug().
			Str("provider", b.provider.Name).
			Str("sessionID", sessionID).
			Msg("Stored MCP session ID")
	}

	log.Info().
		Str("provider", b.provider.Name).
		Str("endpoint", b.provider.Endpoint).
		Msg("MCP provider initialized successfully")

	return nil
}

// sendRequest sends an MCP JSON-RPC request to the provider
// Returns the response and session ID (if present in response headers)
func (b *Bridge) sendRequest(ctx context.Context, mcpReq MCPRequest) (*MCPResponse, error) {
	resp, _, err := b.sendRequestWithSession(ctx, mcpReq)
	return resp, err
}

// sendRequestWithSession sends an MCP JSON-RPC request and returns session ID
func (b *Bridge) sendRequestWithSession(ctx context.Context, mcpReq MCPRequest) (*MCPResponse, string, error) {
	bodyBytes, err := json.Marshal(mcpReq)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal MCP request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.provider.Endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	// Support both JSON and SSE (Server-Sent Events) for MCP protocol
	httpReq.Header.Set("Accept", "application/json, text/event-stream")
	// Set Host header to localhost while preserving provider port for services with host restrictions
	hostHeader := "localhost:3000"
	if parsed, err := url.Parse(b.provider.Endpoint); err == nil {
		if port := parsed.Port(); port != "" {
			hostHeader = fmt.Sprintf("localhost:%s", port)
		}
	}
	httpReq.Host = hostHeader

	// Include session ID if we have one (for stateful MCP servers)
	if b.sessionID != "" {
		httpReq.Header.Set("mcp-session-id", b.sessionID)
	}

	log.Debug().
		Str("provider", b.provider.Name).
		Str("method", mcpReq.Method).
		Str("endpoint", b.provider.Endpoint).
		Str("sessionID", b.sessionID).
		Msg("Sending MCP request to provider")

	httpResp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Extract session ID from response headers (for new sessions)
	sessionID := httpResp.Header.Get("mcp-session-id")

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, sessionID, fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(body))
	}

	respBodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, sessionID, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle SSE (Server-Sent Events) format if present
	var jsonData []byte
	respStr := string(respBodyBytes)

	// Check if response is SSE format (starts with "event:" or "data:")
	if bytes.HasPrefix(respBodyBytes, []byte("event:")) || bytes.HasPrefix(respBodyBytes, []byte("data:")) {
		// Parse SSE format to extract JSON from "data:" field
		lines := bytes.Split(respBodyBytes, []byte("\n"))
		for _, line := range lines {
			line = bytes.TrimSpace(line)
			if bytes.HasPrefix(line, []byte("data: ")) {
				jsonData = bytes.TrimPrefix(line, []byte("data: "))
				break
			}
		}

		if len(jsonData) == 0 {
			return nil, sessionID, fmt.Errorf("no data field found in SSE response: %s", respStr)
		}
	} else {
		// Plain JSON response
		jsonData = respBodyBytes
	}

	var mcpResp MCPResponse
	if err := json.Unmarshal(jsonData, &mcpResp); err != nil {
		return nil, sessionID, fmt.Errorf("failed to unmarshal MCP response: %w (data: %s)", err, string(jsonData))
	}

	return &mcpResp, sessionID, nil
}

// Ping sends a ping request to check provider health
func (b *Bridge) Ping(ctx context.Context) error {
	req := MCPRequest{
		JSONRPC: "2.0",
		Method:  "ping",
		ID:      time.Now().UnixNano(),
	}

	_, err := b.sendRequest(ctx, req)
	return err
}
