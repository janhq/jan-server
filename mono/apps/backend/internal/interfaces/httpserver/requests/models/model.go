package requestmodels

import (
	"fmt"
	"strings"

	domainmodel "jan-server/mono/apps/backend/internal/domain/model"
)

type AddProviderRequest struct {
	Name      string            `json:"name" binding:"required"`
	Vendor    string            `json:"vendor" binding:"required"`
	BaseURL   string            `json:"base_url"`
	URL       string            `json:"url"`
	Endpoints []EndpointDTO     `json:"endpoints"`
	APIKey    string            `json:"api_key"`
	Metadata  map[string]string `json:"metadata"`
	Active    *bool             `json:"active"`
	Category  string            `json:"category"` // "llm" or "image", defaults to "llm"
	DefaultProviderImageGenerate *bool `json:"default_provider_image_generate"`
	DefaultProviderImageEdit     *bool `json:"default_provider_image_edit"`
}

type UpdateProviderRequest struct {
	Name      *string            `json:"name"`
	BaseURL   *string            `json:"base_url"`
	URL       *string            `json:"url"`
	Endpoints []EndpointDTO      `json:"endpoints"`
	APIKey    *string            `json:"api_key"`
	Metadata  *map[string]string `json:"metadata"`
	Active    *bool              `json:"active"`
	DefaultProviderImageGenerate *bool `json:"default_provider_image_generate"`
	DefaultProviderImageEdit     *bool `json:"default_provider_image_edit"`
}

type EndpointDTO struct {
	URL      string `json:"url"`
	Weight   int    `json:"weight,omitempty"`
	Priority int    `json:"priority,omitempty"`
}

// GetEndpointList resolves endpoints with precedence: explicit endpoints array > URL/base_url fields.
func (r *AddProviderRequest) GetEndpointList() (domainmodel.EndpointList, error) {
	if len(r.Endpoints) > 0 {
		return buildEndpointList(r.Endpoints)
	}
	source := strings.TrimSpace(firstNonEmpty(r.URL, r.BaseURL))
	if source == "" {
		return nil, fmt.Errorf("either endpoints or base_url/url is required")
	}
	return domainmodel.ParseEndpoints(source)
}

// GetEndpointList resolves endpoints for update request. The second return indicates whether endpoints were provided.
func (r *UpdateProviderRequest) GetEndpointList() (domainmodel.EndpointList, bool, error) {
	if r.Endpoints != nil {
		endpoints, err := buildEndpointList(r.Endpoints)
		return endpoints, true, err
	}
	if r.BaseURL != nil || r.URL != nil {
		source := ""
		if r.URL != nil {
			source = strings.TrimSpace(*r.URL)
		}
		if source == "" && r.BaseURL != nil {
			source = strings.TrimSpace(*r.BaseURL)
		}
		if source == "" {
			return nil, true, nil
		}
		endpoints, err := domainmodel.ParseEndpoints(source)
		return endpoints, true, err
	}
	return nil, false, nil
}

func buildEndpointList(dtos []EndpointDTO) (domainmodel.EndpointList, error) {
	if len(dtos) == 0 {
		return domainmodel.EndpointList{}, nil
	}
	result := make(domainmodel.EndpointList, 0, len(dtos))
	for idx, dto := range dtos {
		urlStr := strings.TrimSpace(dto.URL)
		if urlStr == "" {
			continue
		}
		parsed, err := domainmodel.ParseEndpoints(urlStr)
		if err != nil {
			return nil, fmt.Errorf("endpoints[%d]: %w", idx, err)
		}
		for _, ep := range parsed {
			weight := dto.Weight
			if weight <= 0 {
				weight = ep.Weight
			}
			result = append(result, domainmodel.Endpoint{
				URL:      ep.URL,
				Weight:   weight,
				Priority: dto.Priority,
				Healthy:  true,
			})
		}
	}
	return result, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

type UpdateModelCatalogRequest struct {
	SupportedParameters *domainmodel.SupportedParameters `json:"supported_parameters"`
	Architecture        *domainmodel.Architecture        `json:"architecture"`
	Tags                *[]string                        `json:"tags"`
	Notes               *string                          `json:"notes"`
	Description         *string                          `json:"description"`
	IsModerated         *bool                            `json:"is_moderated"`
	Extras              *map[string]any                  `json:"extras"`
	Active              *bool                            `json:"active"`
	Experimental        *bool                            `json:"experimental"`
	RequiresFeatureFlag *string                          `json:"requires_feature_flag"`
	SupportsImages      *bool                            `json:"supports_images"`
	SupportsEmbeddings  *bool                            `json:"supports_embeddings"`
	SupportsReasoning   *bool                            `json:"supports_reasoning"`
	SupportsInstruct    *bool                            `json:"supports_instruct"`
	SupportsAudio       *bool                            `json:"supports_audio"`
	SupportsVideo       *bool                            `json:"supports_video"`
	SupportsTools       *bool                            `json:"supports_tools"`
	SupportsBrowser     *bool                            `json:"supports_browser"`
	Family              *string                          `json:"family"`
	ModelDisplayName    *string                          `json:"model_display_name"`
	ContextLength       *float64                         `json:"context_length"`
}

type UpdateProviderModelRequest struct {
	ModelDisplayName      *string                  `json:"model_display_name"`
	Category              *string                  `json:"category"`
	CategoryOrderNumber   *int                     `json:"category_order_number"`
	ModelOrderNumber      *int                     `json:"model_order_number"`
	Pricing               *domainmodel.Pricing     `json:"pricing"`
	TokenLimits           *domainmodel.TokenLimits `json:"token_limits"`
	Family                *string                  `json:"family"`
	SupportsImages        *bool                    `json:"supports_images"`
	SupportsEmbeddings    *bool                    `json:"supports_embeddings"`
	SupportsReasoning     *bool                    `json:"supports_reasoning"`
	SupportsAudio         *bool                    `json:"supports_audio"`
	SupportsVideo         *bool                    `json:"supports_video"`
	Active                *bool                    `json:"active"`
	InstructModelPublicID *string                  `json:"instruct_model_public_id"` // Public ID of the instruct model to use when enable_thinking=false
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
func trimWhitespace(input string) string {
	// Trim leading and trailing whitespace
	result := ""
	start := 0
	end := len(input) - 1

	// Find first non-space character
	for start <= end && (input[start] == ' ' || input[start] == '\t' || input[start] == '\n' || input[start] == '\r') {
		start++
	}

	// Find last non-space character
	for end >= start && (input[end] == ' ' || input[end] == '\t' || input[end] == '\n' || input[end] == '\r') {
		end--
	}

	if start <= end {
		result = input[start : end+1]
	}

	return result
}
