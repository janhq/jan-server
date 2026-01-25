// Package skill contains the skill executor for generating documents, presentations, etc.
package skill

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

// ExecuteParams contains the parameters for skill execution.
type ExecuteParams struct {
	SkillType  string                 `json:"skill_type"`
	OutputPath string                 `json:"output_path,omitempty"`
	Options    map[string]interface{} `json:"options,omitempty"`
}

// ExecuteOutput contains the result of skill execution.
// Note: file_content_base64 is intentionally omitted from outputs to avoid large payloads.
// Artifacts are uploaded from sandbox files in the artifact creation step.
type ExecuteOutput struct {
	Success           bool   `json:"success"`
	SkillType         string `json:"skill_type"`
	OutputPath        string `json:"output_path"`
	FileContentBase64 string `json:"file_content_base64,omitempty"` // Sanitized in API responses
	FileName          string `json:"file_name"`
	MimeType          string `json:"mime_type"`
	FileSize          int64  `json:"file_size,omitempty"` // Size in bytes
}
