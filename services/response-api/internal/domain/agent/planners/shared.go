// Package planners contains agent planner implementations.
package planners

import (
	"context"

	"jan-server/services/response-api/internal/domain/tool"
)

// MCPClient interface for tool execution - matches tool.MCPClient.
type MCPClient interface {
	CallTool(ctx context.Context, req tool.CallRequest) (*tool.Result, error)
}

// LLMProvider interface for LLM calls to fix code and generate content.
type LLMProvider interface {
	FixCode(ctx context.Context, code string, errorMsg string, language string) (string, error)
	Generate(ctx context.Context, prompt string) (string, error)
	GenerateWithModel(ctx context.Context, prompt string, model string) (string, error)
	GenerateWithModelWithMaxTokens(ctx context.Context, prompt string, model string, maxTokens *int) (string, error)
	GenerateWithSystemPrompt(ctx context.Context, systemPrompt string, userPrompt string, model string) (string, error)
	GenerateWithSystemPromptWithMaxTokens(ctx context.Context, systemPrompt string, userPrompt string, model string, maxTokens *int) (string, error)
	GenerateWithStructuredOutput(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any) (string, error)
	GenerateWithStructuredOutputWithMaxTokens(ctx context.Context, systemPrompt string, userPrompt string, model string, schema map[string]any, maxTokens *int) (string, error)
}

// strPtr returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}
