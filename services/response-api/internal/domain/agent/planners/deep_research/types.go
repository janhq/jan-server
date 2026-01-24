// Package deepresearch contains the deep research planner and executor.
package deepresearch

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

// Constants for retry limits.
const (
	// MaxInstallRetries is the maximum number of package install retry attempts.
	MaxInstallRetries = 3

	// MaxCodeFixRetries is the maximum number of LLM code fix retry attempts.
	MaxCodeFixRetries = 5
)

// codeExecutionState tracks the state of code execution retries.
type codeExecutionState struct {
	originalCode      string
	currentCode       string
	installedPackages []string
	installRetryCount int
	codeFixRetryCount int
	executionErrors   []string
}
