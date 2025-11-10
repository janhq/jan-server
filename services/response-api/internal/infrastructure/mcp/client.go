package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"

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

// ListTools fetches the tools via JSON-RPC call tools/list.
func (c *Client) ListTools(ctx context.Context) ([]tool.MCPTool, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/list",
		"params":  map[string]interface{}{},
		"id":      1,
	}

	var rpcResp rpcResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetBody(payload).
		SetResult(&rpcResp).
		Post("/v1/mcp")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("mcp list tools error: %s", resp.String())
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
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*tool.Result, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      name,
			"arguments": args,
		},
		"id": name,
	}

	var rpcResp rpcResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetBody(payload).
		SetResult(&rpcResp).
		Post("/v1/mcp")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("mcp call error: %s", resp.String())
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
		ToolName: name,
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
