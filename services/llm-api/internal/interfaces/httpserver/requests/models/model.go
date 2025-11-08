package requestmodels

import (
	domainmodel "jan-server/services/llm-api/internal/domain/model"
)

type AddProviderRequest struct {
	Name     string            `json:"name" binding:"required"`
	Vendor   string            `json:"vendor" binding:"required"`
	BaseURL  string            `json:"base_url" binding:"required"`
	APIKey   string            `json:"api_key"`
	Metadata map[string]string `json:"metadata"`
	Active   *bool             `json:"active"`
}

type UpdateProviderRequest struct {
	Name     *string            `json:"name"`
	BaseURL  *string            `json:"base_url"`
	APIKey   *string            `json:"api_key"`
	Metadata *map[string]string `json:"metadata"`
	Active   *bool              `json:"active"`
}

type UpdateModelCatalogRequest struct {
	SupportedParameters *domainmodel.SupportedParameters `json:"supported_parameters"`
	Architecture        *domainmodel.Architecture        `json:"architecture"`
	Tags                *[]string                        `json:"tags"`
	Notes               *string                          `json:"notes"`
	IsModerated         *bool                            `json:"is_moderated"`
	Extras              *map[string]any                  `json:"extras"`
}

type UpdateProviderModelRequest struct {
	DisplayName        *string                  `json:"display_name"`
	Pricing            *domainmodel.Pricing     `json:"pricing"`
	TokenLimits        *domainmodel.TokenLimits `json:"token_limits"`
	Family             *string                  `json:"family"`
	SupportsImages     *bool                    `json:"supports_images"`
	SupportsEmbeddings *bool                    `json:"supports_embeddings"`
	SupportsReasoning  *bool                    `json:"supports_reasoning"`
	SupportsAudio      *bool                    `json:"supports_audio"`
	SupportsVideo      *bool                    `json:"supports_video"`
	Active             *bool                    `json:"active"`
}

type BulkEnableModelsRequest struct {
	Enable       *bool    `json:"enable" binding:"required"`             // Required: true to enable, false to disable
	ExceptModels []string `json:"except_models"`                         // List of model keys to exclude
	ProviderID   *string  `json:"provider_id" binding:"omitempty,min=1"` // Optional: filter by provider
}

// Normalize removes duplicates and trims whitespace from model keys
func (r *BulkEnableModelsRequest) Normalize() {
	if len(r.ExceptModels) == 0 {
		return
	}

	seen := make(map[string]bool)
	normalized := make([]string, 0, len(r.ExceptModels))
	for _, key := range r.ExceptModels {
		trimmed := trimWhitespace(key)
		if trimmed == "" {
			continue
		}
		if !seen[trimmed] {
			seen[trimmed] = true
			normalized = append(normalized, trimmed)
		}
	}
	r.ExceptModels = normalized
}

type BulkToggleCatalogsRequest struct {
	Enable       *bool    `json:"enable" binding:"required"`        // Required: true to enable, false to disable
	CatalogIDs   []string `json:"catalog_ids"  binding:"omitempty"` // Optional: specific catalog public IDs. If empty, applies to all catalogs
	ExceptModels []string `json:"except_models"`                    // List of model keys to exclude from the operation
}

// Normalize removes duplicates and trims whitespace from catalog IDs and model keys
func (r *BulkToggleCatalogsRequest) Normalize() {
	// Normalize catalog IDs
	if len(r.CatalogIDs) > 0 {
		seen := make(map[string]bool)
		normalized := make([]string, 0, len(r.CatalogIDs))
		for _, id := range r.CatalogIDs {
			trimmed := trimWhitespace(id)
			if trimmed == "" {
				continue
			}
			if !seen[trimmed] {
				seen[trimmed] = true
				normalized = append(normalized, trimmed)
			}
		}
		r.CatalogIDs = normalized
	}

	// Normalize except models
	if len(r.ExceptModels) > 0 {
		seen := make(map[string]bool)
		normalized := make([]string, 0, len(r.ExceptModels))
		for _, key := range r.ExceptModels {
			trimmed := trimWhitespace(key)
			if trimmed == "" {
				continue
			}
			if !seen[trimmed] {
				seen[trimmed] = true
				normalized = append(normalized, trimmed)
			}
		}
		r.ExceptModels = normalized
	}
}

// trimWhitespace is a helper function to trim whitespace
func trimWhitespace(s string) string {
	// Trim leading and trailing whitespace
	result := ""
	start := 0
	end := len(s) - 1

	// Find first non-space character
	for start <= end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	// Find last non-space character
	for end >= start && (s[end] == ' ' || s[end] == '\t' || s[end] == '\n' || s[end] == '\r') {
		end--
	}

	if start <= end {
		result = s[start : end+1]
	}

	return result
}
