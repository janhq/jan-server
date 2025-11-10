package llm

import (
	"context"
	"encoding/json"
)

// Provider defines the contract for calling the LLM API /v1/chat/completions endpoint.
type Provider interface {
	CreateChatCompletion(reqCtx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error)
	CreateChatCompletionStream(reqCtx context.Context, req ChatCompletionRequest) (Stream, error)
}

// Stream abstracts an SSE or chunked response from the LLM API.
type Stream interface {
	Recv() (*ChatCompletionDelta, error)
	Close() error
}

// ChatCompletionRequest mirrors the OpenAI-compatible request shape exposed by llm-api.
type ChatCompletionRequest struct {
	Model       string           `json:"model"`
	Messages    []ChatMessage    `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	ToolChoice  *ToolChoice      `json:"tool_choice,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
	MaxTokens   *int             `json:"max_tokens,omitempty"`
	Stream      bool             `json:"stream"`
}

// ChatMessage represents a single message in the conversation history.
type ChatMessage struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID *string     `json:"tool_call_id,omitempty"`
}

// ToolCall mirrors the OpenAI tool call format.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction contains the function name and JSON arguments.
type ToolFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolDefinition is the OpenAI compatible representation of an MCP tool.
type ToolDefinition struct {
	Type     string             `json:"type"`
	Function ToolFunctionSchema `json:"function"`
}

// ToolFunctionSchema declares the function contract passed to the LLM.
type ToolFunctionSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolChoice allows forcing a specific tool or auto mode.
type ToolChoice struct {
	Type     string `json:"type"`
	Function struct {
		Name string `json:"name"`
	} `json:"function"`
}

// ChatCompletionResponse captures the non-streaming completion payload.
type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   *Usage                 `json:"usage,omitempty"`
}

// ChatCompletionChoice represents one completion choice.
type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// Usage contains token accounting metadata.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletionDelta represents a streaming chunk.
type ChatCompletionDelta struct {
	Choices []ChatCompletionDeltaChoice `json:"choices"`
}

// ChatCompletionDeltaChoice mirrors OpenAI streaming deltas.
type ChatCompletionDeltaChoice struct {
	Delta        ChatMessage `json:"delta"`
	FinishReason string      `json:"finish_reason"`
	Index        int         `json:"index"`
}
