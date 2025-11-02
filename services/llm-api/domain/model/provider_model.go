package model

import (
	"context"
	"time"

	"jan-server/services/llm-api/domain/query"
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

// PriceLine is a single line item (e.g., prompt token price)
type PriceLine struct {
	Unit     PriceUnit `json:"unit"`
	Amount   MicroUSD  `json:"amount_micro_usd"`
	Currency string    `json:"currency"`
}

// Pricing groups price lines for a model
type Pricing struct {
	Lines []PriceLine `json:"lines"`
}

// TokenLimits for context and completion
type TokenLimits struct {
	ContextLength       int `json:"context_length"`
	MaxCompletionTokens int `json:"max_completion_tokens"`
}

// ProviderModel describes a specific model under a provider
type ProviderModel struct {
	ID                      uint         `json:"id"`
	PublicID                string       `json:"public_id"`
	ProviderID              uint         `json:"provider_id"`
	Kind                    ProviderKind `json:"kind"`
	ModelCatalogID          *uint        `json:"model_catalog_id"`
	ModelPublicID           string       `json:"model_public_id"`
	ProviderOriginalModelID string       `json:"provider_original_model_id"`
	DisplayName             string       `json:"display_name"`
	Pricing                 Pricing      `json:"pricing"`
	TokenLimits             *TokenLimits `json:"token_limits,omitempty"`
	Family                  *string      `json:"family,omitempty"`
	SupportsImages          bool         `json:"supports_images"`
	SupportsEmbeddings      bool         `json:"supports_embeddings"`
	SupportsReasoning       bool         `json:"supports_reasoning"`
	SupportsAudio           bool         `json:"supports_audio"`
	SupportsVideo           bool         `json:"supports_video"`
	Active                  bool         `json:"active"`
	CreatedAt               time.Time    `json:"created_at"`
	UpdatedAt               time.Time    `json:"updated_at"`
}

// ProviderModelFilter defines optional conditions for querying provider models
type ProviderModelFilter struct {
	IDs                *[]uint
	PublicID           *string
	ProviderIDs        *[]uint
	ProviderID         *uint
	ModelCatalogID     *uint
	ModelPublicID      *string
	ModelPublicIDs     *[]string
	Active             *bool
	SupportsImages     *bool
	SupportsEmbeddings *bool
	SupportsReasoning  *bool
	SupportsAudio      *bool
	SupportsVideo      *bool
}

// ProviderModelRepository abstracts persistence for provider models
type ProviderModelRepository interface {
	Create(ctx context.Context, model *ProviderModel) error
	Update(ctx context.Context, model *ProviderModel) error
	DeleteByID(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*ProviderModel, error)
	FindByPublicID(ctx context.Context, publicID string) (*ProviderModel, error)
	FindByFilter(ctx context.Context, filter ProviderModelFilter, p *query.Pagination) ([]*ProviderModel, error)
	Count(ctx context.Context, filter ProviderModelFilter) (int64, error)
	BatchUpdateActive(ctx context.Context, filter ProviderModelFilter, active bool) (int64, error)
}
