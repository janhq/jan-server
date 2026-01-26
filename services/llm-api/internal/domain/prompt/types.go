package prompt

import (
	"context"

	openai "github.com/sashabaranov/go-openai"

	"jan-server/services/llm-api/internal/domain/usersettings"
)

// ProcessorConfig contains configuration for the prompt orchestration processor
type ProcessorConfig struct {
	Enabled         bool
	EnableMemory    bool
	EnableTemplates bool
	EnableTools     bool
}

// Context contains contextual information for prompt processing
type Context struct {
	UserID             uint
	ConversationID     string
	Language           string
	Preferences        map[string]interface{}
	Memory             []string
	ProjectInstruction string
	ProjectID          *uint // Internal project ID for fetching project files
	AppliedModules     []string
	Profile            *usersettings.ProfileSettings

	// Model context for model-specific template resolution
	ModelCatalogID *string

	// Tools available in the request
	Tools []openai.Tool
}

// Module represents a prompt module that can be applied
type Module interface {
	// Name returns the module identifier
	Name() string

	// ShouldApply determines if this module should be applied based on context
	ShouldApply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) bool

	// Apply modifies the messages array by adding or modifying prompts
	Apply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error)
}

// Processor orchestrates prompt composition by applying conditional modules
type Processor interface {
	// Process takes a request and applies all relevant modules
	Process(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error)
}
