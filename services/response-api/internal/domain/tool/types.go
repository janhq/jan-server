package tool

import (
	"context"
	"encoding/json"
	"time"

	"jan-server/services/response-api/internal/domain/llm"
)

// ExecutionStatus represents the lifecycle of a tool execution attempt.
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
)

// Call encapsulates one tool call requested by the LLM.
type Call struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// Result captures the outcome returned by the MCP tool runner.
type Result struct {
	ToolName string       `json:"tool_name"`
	Content  []MCPContent `json:"content"`
	IsError  bool         `json:"is_error"`
	Error    string       `json:"error,omitempty"`
}

// MCPContent represents values inside the MCP streaming payload.
type MCPContent struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text,omitempty"`
	Resource map[string]interface{} `json:"resource,omitempty"`
}

// Execution links a requested tool call to its persisted record.
type Execution struct {
	ID             uint            `json:"id"`
	ResponseID     uint            `json:"response_id"`
	CallID         string          `json:"call_id"`
	ToolName       string          `json:"tool_name"`
	Arguments      map[string]any  `json:"arguments"`
	Result         *Result         `json:"result,omitempty"`
	Status         ExecutionStatus `json:"status"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	ExecutionOrder int             `json:"execution_order"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// MCPClient abstracts calls to mcp-tools /v1/mcp endpoint.
type MCPClient interface {
	ListTools(ctx context.Context) ([]MCPTool, error)
	CallTool(ctx context.Context, name string, args map[string]interface{}) (*Result, error)
}

// StreamObserver receives live updates during orchestration.
type StreamObserver interface {
	OnDelta(delta llm.ChatCompletionDelta)
	OnToolCall(call Call)
	OnToolResult(callID string, result *Result)
}

// MCPTool describes the tool metadata returned by mcp-tools.
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToLLMTool converts MCP metadata into OpenAI-compatible tool definition.
func (t MCPTool) ToLLMTool() llm.ToolDefinition {
	return llm.ToolDefinition{
		Type: "function",
		Function: llm.ToolFunctionSchema{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.InputSchema,
		},
	}
}

// ParseToolCall converts an LLM provided tool call into the domain Call struct.
func ParseToolCall(call llm.ToolCall) (Call, error) {
	var args map[string]interface{}
	if len(call.Function.Arguments) > 0 {
		if err := json.Unmarshal(call.Function.Arguments, &args); err != nil {
			return Call{}, err
		}
	}
	return Call{
		ID:        call.ID,
		Name:      call.Function.Name,
		Arguments: args,
	}, nil
}
