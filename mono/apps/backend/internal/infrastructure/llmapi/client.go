package llmapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// Client handles communication with LLM-API for tool tracking
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new LLM-API client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UpdateItemRequest represents the PATCH request body for updating mcp_call items
type UpdateItemRequest struct {
	// Required
	Status string `json:"status"`

	// Result fields
	Output *string `json:"output,omitempty"`
	Error  *string `json:"error,omitempty"`

	// Tool info fields (for updating mcp_call item)
	Name        *string `json:"name,omitempty"`
	Arguments   *string `json:"arguments,omitempty"`
	ServerLabel *string `json:"server_label,omitempty"`
}

// ItemResponse represents the response from the PATCH endpoint
type ItemResponse struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Type      string `json:"type"`
	CallID    string `json:"call_id"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

// PatchResult represents the result of a PATCH operation
type PatchResult struct {
	Success    bool
	StatusCode int
	Item       *ItemResponse
	Error      error
}

// UpdateToolCallResult updates an existing in_progress mcp_call item to completed/failed
// This is the main method called by MCP tool handlers after tool execution
//
// Security: LLM-API will validate:
// 1. The call_id exists in the specified conversation_id
// 2. The conversation belongs to the authenticated user (from JWT)
// 3. The item status is "in_progress" (not already completed)
func (c *Client) UpdateToolCallResult(
	ctx context.Context,
	authToken string,
	conversationID string,
	toolCallID string,
	toolName string,
	arguments string,
	serverLabel string,
	output string,
	toolError *string,
) *PatchResult {
	// Use call_id to find and update the in_progress item
	endpoint := fmt.Sprintf("%s/v1/conversations/%s/items/by-call-id/%s", c.baseURL, conversationID, toolCallID)

	status := "completed"
	if toolError != nil && *toolError != "" {
		status = "failed"
	}

	reqBody := UpdateItemRequest{
		Status:      status,
		Output:      &output,
		Error:       toolError,
		Name:        &toolName,
		Arguments:   &arguments,
		ServerLabel: &serverLabel,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return &PatchResult{
			Success: false,
			Error:   fmt.Errorf("failed to marshal request: %w", err),
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return &PatchResult{
			Success: false,
			Error:   fmt.Errorf("failed to create request: %w", err),
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authToken)

	log.Info().
		Str("conv_id", conversationID).
		Str("call_id", toolCallID).
		Str("tool_name", toolName).
		Str("status", status).
		Msg("Updating mcp_call item in LLM-API")

	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	patchDuration := time.Since(startTime)

	if err != nil {
		log.Error().
			Err(err).
			Str("conv_id", conversationID).
			Str("call_id", toolCallID).
			Str("tool_name", toolName).
			Int64("patch_duration_ms", patchDuration.Milliseconds()).
			Msg("Failed to call LLM-API PATCH endpoint")
		return &PatchResult{
			Success: false,
			Error:   fmt.Errorf("failed to call LLM-API: %w", err),
		}
	}
	defer resp.Body.Close()

	result := &PatchResult{
		StatusCode: resp.StatusCode,
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		result.Success = true
		// Parse response body
		var itemResp ItemResponse
		if err := json.NewDecoder(resp.Body).Decode(&itemResp); err == nil {
			result.Item = &itemResp
		}
		log.Info().
			Str("conv_id", conversationID).
			Str("call_id", toolCallID).
			Str("tool_name", toolName).
			Str("status", status).
			Int64("patch_duration_ms", patchDuration.Milliseconds()).
			Msg("Tool result saved to LLM-API")

	case http.StatusConflict:
		// Item already processed - this is idempotent, log as info
		result.Success = true // Idempotent success
		log.Info().
			Str("conv_id", conversationID).
			Str("call_id", toolCallID).
			Str("tool_name", toolName).
			Int("status_code", resp.StatusCode).
			Msg("PATCH idempotent - item already processed")

	case http.StatusNotFound:
		body, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Errorf("call_id not found in conversation: %s", string(body))
		log.Warn().
			Str("conv_id", conversationID).
			Str("call_id", toolCallID).
			Str("tool_name", toolName).
			Int("status_code", resp.StatusCode).
			Msg("PATCH failed - call_id not found")

	case http.StatusForbidden:
		body, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Errorf("access denied: %s", string(body))
		log.Error().
			Str("conv_id", conversationID).
			Str("call_id", toolCallID).
			Str("tool_name", toolName).
			Int("status_code", resp.StatusCode).
			Msg("PATCH failed - access denied (security event)")

	default:
		body, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Errorf("LLM-API returned status %d: %s", resp.StatusCode, string(body))
		log.Error().
			Str("conv_id", conversationID).
			Str("call_id", toolCallID).
			Str("tool_name", toolName).
			Int("status_code", resp.StatusCode).
			Str("response_body", string(body)).
			Int64("patch_duration_ms", patchDuration.Milliseconds()).
			Msg("Failed to save tool result to LLM-API")
	}

	return result
}

// MCPToolConfig represents the configuration of an MCP tool from LLM-API
type MCPToolConfig struct {
	ID                 string   `json:"id"`
	PublicID           string   `json:"public_id"`
	ToolKey            string   `json:"tool_key"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	Category           string   `json:"category"`
	IsActive           bool     `json:"is_active"`
	DisallowedKeywords []string `json:"disallowed_keywords,omitempty"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

// MCPToolsResponse represents the response from the ListActive endpoint
type MCPToolsResponse struct {
	Data []MCPToolConfig `json:"data"`
}

// SingleMCPToolResponse represents the response for a single tool
type SingleMCPToolResponse struct {
	Data MCPToolConfig `json:"data"`
}

// GetActiveMCPTools fetches all active MCP tool configurations from LLM-API
// This is used by mcp-tools to populate tool descriptions dynamically
func (c *Client) GetActiveMCPTools(ctx context.Context) ([]MCPToolConfig, error) {
	endpoint := fmt.Sprintf("%s/v1/mcp-tools", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	log.Debug().
		Str("endpoint", endpoint).
		Msg("Fetching active MCP tools from LLM-API")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM-API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM-API returned status %d: %s", resp.StatusCode, string(body))
	}

	var toolsResp MCPToolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&toolsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Debug().
		Int("count", len(toolsResp.Data)).
		Msg("Fetched active MCP tools")

	return toolsResp.Data, nil
}

// GetMCPToolByKey fetches a single MCP tool configuration by its key
func (c *Client) GetMCPToolByKey(ctx context.Context, toolKey string) (*MCPToolConfig, error) {
	endpoint := fmt.Sprintf("%s/v1/mcp-tools/%s", c.baseURL, toolKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM-API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Tool not found or inactive
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM-API returned status %d: %s", resp.StatusCode, string(body))
	}

	var toolResp SingleMCPToolResponse
	if err := json.NewDecoder(resp.Body).Decode(&toolResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &toolResp.Data, nil
}
