package mcp

import (
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func extractArguments(req *mcp.CallToolRequest) map[string]any {
	if req == nil || req.Params == nil {
		return nil
	}

	raw := req.Params.Arguments
	if len(raw) == 0 {
		return nil
	}

	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil
	}
	return args
}

func extractContextString(req *mcp.CallToolRequest, key string) string {
	args := extractArguments(req)
	if args == nil {
		return ""
	}
	if val, ok := args[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func extractAllContext(req *mcp.CallToolRequest) map[string]string {
	return map[string]string{
		"tool_call_id":    extractContextString(req, "tool_call_id"),
		"request_id":      extractContextString(req, "request_id"),
		"conversation_id": extractContextString(req, "conversation_id"),
		"user_id":         extractContextString(req, "user_id"),
	}
}
