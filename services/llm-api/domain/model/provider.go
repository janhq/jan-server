package model

import (
	"context"
	"time"

	"jan-server/services/llm-api/domain/query"
)

type ProviderKind string

const (
	ProviderMenlo       ProviderKind = "menlo"
	ProviderOpenAI      ProviderKind = "openai"
	ProviderOpenRouter  ProviderKind = "openrouter"
	ProviderAnthropic   ProviderKind = "anthropic"
	ProviderGoogle      ProviderKind = "google"
	ProviderMistral     ProviderKind = "mistral"
	ProviderGroq        ProviderKind = "groq"
	ProviderCohere      ProviderKind = "cohere"
	ProviderOllama      ProviderKind = "ollama"
	ProviderReplicate   ProviderKind = "replicate"
	ProviderAzureOpenAI ProviderKind = "azure_openai"
	ProviderAWSBedrock  ProviderKind = "aws_bedrock"
	ProviderPerplexity  ProviderKind = "perplexity"
	ProviderTogetherAI  ProviderKind = "togetherai"
	ProviderHuggingFace ProviderKind = "huggingface"
	ProviderVercelAI    ProviderKind = "vercel_ai"
	ProviderDeepInfra   ProviderKind = "deepinfra"
	ProviderVLLM        ProviderKind = "vllm"
	ProviderCustom      ProviderKind = "custom"
)

type Provider struct {
	ID              uint              `json:"id"`
	PublicID        string            `json:"public_id"`
	DisplayName     string            `json:"display_name"`
	Kind            ProviderKind      `json:"kind"`
	BaseURL         string            `json:"base_url"`
	EncryptedAPIKey string            `json:"-"`
	APIKeyHint      *string           `json:"api_key_hint,omitempty"`
	IsModerated     bool              `json:"is_moderated"`
	Active          bool              `json:"active"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	LastSyncedAt    *time.Time        `json:"last_synced_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// ProviderFilter defines optional conditions for querying providers
type ProviderFilter struct {
	IDs              *[]uint
	PublicID         *string
	Kind             *ProviderKind
	Active           *bool
	IsModerated      *bool
	LastSyncedAfter  *time.Time
	LastSyncedBefore *time.Time
}

type AccessibleModels struct {
	Providers      []*Provider      `json:"providers"`
	ProviderModels []*ProviderModel `json:"provider_models"`
}

// ProviderRepository abstracts persistence for provider aggregate roots
type ProviderRepository interface {
	Create(ctx context.Context, provider *Provider) error
	Update(ctx context.Context, provider *Provider) error
	DeleteByID(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*Provider, error)
	FindByPublicID(ctx context.Context, publicID string) (*Provider, error)
	FindByFilter(ctx context.Context, filter ProviderFilter, p *query.Pagination) ([]*Provider, error)
	Count(ctx context.Context, filter ProviderFilter) (int64, error)
	FindByIDs(ctx context.Context, ids []uint) ([]*Provider, error)
}
