package llm

import "context"

// ModelInfo contains model metadata including context length limits.
type ModelInfo struct {
	ID            string `json:"id"`
	ContextLength *int   `json:"context_length,omitempty"`
	MaxTokens     *int   `json:"max_tokens,omitempty"`
}

// ModelInfoProvider fetches model metadata from llm-api.
type ModelInfoProvider interface {
	GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error)
}
