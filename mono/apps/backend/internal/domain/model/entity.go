package model

import (
	"time"
)

// Provider represents an LLM provider.
type Provider struct {
	ID          string
	Name        string
	DisplayName string
	BaseURL     string
	APIKey      string // Encrypted
	IsEnabled   bool
	Config      map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Model represents an LLM model.
type Model struct {
	ID            string
	ProviderID    string
	Name          string
	DisplayName   string
	Description   string
	ContextWindow int
	MaxTokens     int
	IsEnabled     bool
	Capabilities  ModelCapabilities
	Pricing       ModelPricing
	CreatedAt     time.Time
	UpdatedAt     time.Time

	Provider *Provider
}

// ModelCapabilities describes what the model can do.
type ModelCapabilities struct {
	Vision          bool `json:"vision"`
	FunctionCalling bool `json:"function_calling"`
	Streaming       bool `json:"streaming"`
	JSON            bool `json:"json"`
}

// ModelPricing describes the model's pricing.
type ModelPricing struct {
	InputPerMillion  float64 `json:"input_per_million"`
	OutputPerMillion float64 `json:"output_per_million"`
	Currency         string  `json:"currency"`
}

// ProviderResponse is the API response for a provider.
type ProviderResponse struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name"`
	BaseURL     string         `json:"base_url,omitempty"`
	IsEnabled   bool           `json:"is_enabled"`
	Config      map[string]any `json:"config,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

// ModelResponse is the API response for a model.
type ModelResponse struct {
	ID            string            `json:"id"`
	ProviderID    string            `json:"provider_id"`
	Name          string            `json:"name"`
	DisplayName   string            `json:"display_name"`
	Description   string            `json:"description,omitempty"`
	ContextWindow int               `json:"context_window"`
	MaxTokens     int               `json:"max_tokens"`
	IsEnabled     bool              `json:"is_enabled"`
	Capabilities  ModelCapabilities `json:"capabilities"`
	Pricing       ModelPricing      `json:"pricing,omitempty"`
	Provider      *ProviderResponse `json:"provider,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}

// ListModelsFilter contains filters for listing models.
type ListModelsFilter struct {
	ProviderID string
	IsEnabled  *bool
	Search     string
	Limit      int
	Offset     int
}

// CreateProviderRequest contains data for creating a provider.
type CreateProviderRequest struct {
	Name        string
	DisplayName string
	BaseURL     string
	APIKey      string
	Config      map[string]any
}

// CreateModelRequest contains data for creating a model.
type CreateModelRequest struct {
	ProviderID    string
	Name          string
	DisplayName   string
	Description   string
	ContextWindow int
	MaxTokens     int
	Capabilities  ModelCapabilities
	Pricing       ModelPricing
}

// ToResponse converts a Provider to ProviderResponse.
func (p *Provider) ToResponse() ProviderResponse {
	return ProviderResponse{
		ID:          p.ID,
		Name:        p.Name,
		DisplayName: p.DisplayName,
		BaseURL:     p.BaseURL,
		IsEnabled:   p.IsEnabled,
		Config:      p.Config,
		CreatedAt:   p.CreatedAt,
	}
}

// ToResponse converts a Model to ModelResponse.
func (m *Model) ToResponse() ModelResponse {
	resp := ModelResponse{
		ID:            m.ID,
		ProviderID:    m.ProviderID,
		Name:          m.Name,
		DisplayName:   m.DisplayName,
		Description:   m.Description,
		ContextWindow: m.ContextWindow,
		MaxTokens:     m.MaxTokens,
		IsEnabled:     m.IsEnabled,
		Capabilities:  m.Capabilities,
		Pricing:       m.Pricing,
		CreatedAt:     m.CreatedAt,
	}
	if m.Provider != nil {
		pr := m.Provider.ToResponse()
		resp.Provider = &pr
	}
	return resp
}
