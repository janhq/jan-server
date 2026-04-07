package model

import (
	"context"
	"time"

	"jan-server/services/llm-api/internal/domain/query"
)

type MicroUSD int64

type PriceUnit string

const (
	Per1KPromptTokens     PriceUnit = "per_1k_prompt_tokens"
	Per1KCompletionTokens PriceUnit = "per_1k_completion_tokens"
	PerRequest            PriceUnit = "per_request"
	PerImage              PriceUnit = "per_image"
	PerWebSearch          PriceUnit = "per_web_search"
	PerInternalReasoning  PriceUnit = "per_internal_reasoning"
)

// PriceLine is a single line item (e.g., prompt token price).
type PriceLine struct {
	Unit     PriceUnit `json:"unit"`
	Amount   MicroUSD  `json:"amount_micro_usd"` // e.g., 15000 -> $0.0150
	Currency string    `json:"currency"`         // "USD" (fixed if you only bill in USD)
}

// Pricing groups price lines for a model.
type Pricing struct {
	Lines []PriceLine `json:"lines"` // flexible: add/remove units without schema churn
}

// TokenLimits for context and completion.
type TokenLimits struct {
	ContextLength       int `json:"context_length"`        // e.g., 400000
	MaxCompletionTokens int `json:"max_completion_tokens"` // e.g., 128000
}

// ReasoningModeOption describes a selectable reasoning/auto mode for UI hints.
type ReasoningModeOption struct {
	Name                string   `json:"name"`
	DisplayName         string   `json:"display_name"`
	Description         string   `json:"description,omitempty"`
	LatencyHintMs       *int     `json:"latency_hint_ms,omitempty"`
	PriceHintMultiplier *float64 `json:"price_hint_multiplier,omitempty"`
}

// ReasoningConfig contains all reasoning-related configuration for a model.
type ReasoningConfig struct {
	EffortLevels    []string              `json:"effort_levels,omitempty"`    // e.g., ["low","medium","high"]
	DefaultEffort   string                `json:"default_effort,omitempty"`   // Default effort level
	MaxTokens       *int                  `json:"max_tokens,omitempty"`       // Max tokens for reasoning output
	PriceMultiplier *float64              `json:"price_multiplier,omitempty"` // Price multiplier vs standard mode
	LatencyHintMs   *int                  `json:"latency_hint_ms,omitempty"`  // Estimated added latency
	ModeDisplay     []ReasoningModeOption `json:"mode_display,omitempty"`     // UI display options
}

// ProviderModel describes a specific model under a provider.
type ProviderModel struct {
	ID                      uint             `json:"id"`
	PublicID                string           `json:"public_id"`
	ProviderID              uint             `json:"provider_id"`
	Kind                    ProviderKind     `json:"kind"`
	ModelCatalogID          *uint            `json:"model_catalog_id"`
	ModelPublicID           string           `json:"model_public_id"`            // Matches model_catalog.PublicID (canonical vendor/model format)
	ModelDisplayName        string           `json:"model_display_name"`         // User-friendly display name for the model (defaults to ModelPublicID)
	ProviderOriginalModelID string           `json:"provider_original_model_id"` // Original model ID from provider API (chat.Model.ID)
	Category                string           `json:"category"`                   // Category for grouping models (e.g., "Chat", "Embedding", "Image")
	CategoryOrderNumber     int              `json:"category_order_number"`      // Order number for sorting categories
	ModelOrderNumber        int              `json:"model_order_number"`         // Order number for sorting models within a category
	Pricing                 Pricing          `json:"pricing"`
	TokenLimits             *TokenLimits     `json:"token_limits,omitempty"` // override provider top caps
	SupportsAutoMode        bool             `json:"supports_auto_mode"`
	SupportsThinkingMode    bool             `json:"supports_thinking_mode"`
	DefaultConversationMode string           `json:"default_conversation_mode,omitempty"`
	ReasoningConfig         *ReasoningConfig `json:"reasoning_config,omitempty"`
	ProviderFlags           map[string]any   `json:"provider_flags,omitempty"`
	Active                  bool             `json:"active"`
	InstructModelID         *uint            `json:"instruct_model_id,omitempty"` // Self-referencing FK to the instruct model variant (used when enable_thinking=false)
	CreatedAt               time.Time        `json:"created_at"`
	UpdatedAt               time.Time        `json:"updated_at"`
}

// Validate checks if the ProviderModel has all required fields populated
func (pm *ProviderModel) Validate() error {
	if pm.ProviderID == 0 {
		return &ValidationError{Field: "provider_id", Message: "provider_id is required"}
	}
	if pm.ModelPublicID == "" {
		return &ValidationError{Field: "model_public_id", Message: "model_public_id is required"}
	}
	if pm.ModelDisplayName == "" {
		return &ValidationError{Field: "model_display_name", Message: "model_display_name is required"}
	}
	if pm.ProviderOriginalModelID == "" {
		return &ValidationError{Field: "provider_original_model_id", Message: "provider_original_model_id is required"}
	}
	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ProviderModelFilter defines optional conditions for querying provider models.
type ProviderModelFilter struct {
	IDs                     *[]uint
	PublicID                *string
	ProviderIDs             *[]uint
	ProviderID              *uint
	ModelCatalogID          *uint
	ModelPublicID           *string
	ModelPublicIDs          *[]string
	ProviderOriginalModelID *string
	Active                  *bool
	SupportsImages          *bool
	SearchText              *string
}

// ProviderModelRepository abstracts persistence for provider models.
type ProviderModelRepository interface {
	Create(ctx context.Context, model *ProviderModel) error
	Update(ctx context.Context, model *ProviderModel) error
	DeleteByID(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*ProviderModel, error)
	FindByPublicID(ctx context.Context, publicID string) (*ProviderModel, error)
	FindByFilter(ctx context.Context, filter ProviderModelFilter, p *query.Pagination) ([]*ProviderModel, error)
	Count(ctx context.Context, filter ProviderModelFilter) (int64, error)
	BatchUpdateActive(ctx context.Context, filter ProviderModelFilter, active bool) (int64, error)
	BatchUpdateModelDisplayName(ctx context.Context, filter ProviderModelFilter, modelDisplayName string) (int64, error)
}
