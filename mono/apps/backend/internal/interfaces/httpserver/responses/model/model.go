package modelresponses

import (
	"sort"
	"strings"

	domainmodel "jan-server/mono/apps/backend/internal/domain/model"
	"jan-server/mono/apps/backend/internal/utils/ptr"
)

// getModelDisplayName returns ModelDisplayName if set, otherwise falls back to ModelPublicID
func getModelDisplayName(pm *domainmodel.ProviderModel) string {
	if pm.ModelDisplayName != "" {
		return pm.ModelDisplayName
	}
	return pm.ModelPublicID
}

type ModelResponse struct {
	ID                  string `json:"id"`
	Object              string `json:"object"`
	Created             int64  `json:"created"`
	OwnedBy             string `json:"owned_by"`
	ModelDisplayName    string `json:"model_display_name"`
	Category            string `json:"category"`
	CategoryOrderNumber int    `json:"category_order_number"`
	ModelOrderNumber    int    `json:"model_order_number"`
}

type ModelResponseList struct {
	Object string          `json:"object"`
	Data   []ModelResponse `json:"data"`
}

type ModelResponseWithProvider struct {
	ID                  string `json:"id"`
	Object              string `json:"object"`
	Created             int64  `json:"created"`
	OwnedBy             string `json:"owned_by"`
	ModelDisplayName    string `json:"model_display_name"`
	Category            string `json:"category"`
	CategoryOrderNumber int    `json:"category_order_number"`
	ModelOrderNumber    int    `json:"model_order_number"`
	ProviderID          string `json:"provider_id"`
	ProviderVendor      string `json:"provider_vendor"`
	ProviderName        string `json:"provider_name"`
}

type ModelWithProviderResponseList struct {
	Object string                      `json:"object"`
	Data   []ModelResponseWithProvider `json:"data"`
}

type ProviderResponse struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Vendor    string             `json:"vendor"`
	BaseURL   string             `json:"base_url"`
	Endpoints []EndpointResponse `json:"endpoints,omitempty"`
	Active    bool               `json:"active"`
	Category  string             `json:"category"`
	DefaultProviderImageGenerate bool `json:"default_provider_image_generate"`
	DefaultProviderImageEdit     bool `json:"default_provider_image_edit"`
	Metadata  map[string]string  `json:"metadata,omitempty"`
}

type ProviderWithModelCountResponse struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Vendor           string             `json:"vendor"`
	BaseURL          string             `json:"base_url"`
	Endpoints        []EndpointResponse `json:"endpoints,omitempty"`
	Active           bool               `json:"active"`
	Category         string             `json:"category"`
	DefaultProviderImageGenerate bool `json:"default_provider_image_generate"`
	DefaultProviderImageEdit     bool `json:"default_provider_image_edit"`
	ModelCount       int64              `json:"model_count"`
	ModelActiveCount int64              `json:"model_active_count"`
	Metadata         map[string]string  `json:"metadata,omitempty"`
}

type ProviderWithModelsResponse struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Vendor    string             `json:"vendor"`
	BaseURL   string             `json:"base_url"`
	Endpoints []EndpointResponse `json:"endpoints,omitempty"`
	Models    []ModelResponse    `json:"models"`
	Active    bool               `json:"active"`
	Category  string             `json:"category"`
	DefaultProviderImageGenerate bool `json:"default_provider_image_generate"`
	DefaultProviderImageEdit     bool `json:"default_provider_image_edit"`
	Metadata  map[string]string  `json:"metadata,omitempty"`
}

type ProviderResponseList struct {
	Object string             `json:"object"`
	Data   []ProviderResponse `json:"data"`
}

func BuildModelResponseListWithProvider(
	providerModels []*domainmodel.ProviderModel,
	providerByID map[uint]*domainmodel.Provider,
) []ModelResponseWithProvider {
	items := make([]ModelResponseWithProvider, 0, len(providerModels))

	for _, pm := range providerModels {
		if pm == nil {
			continue
		}
		provider := providerByID[pm.ProviderID]
		if provider == nil {
			continue
		}
		items = append(items, ModelResponseWithProvider{
			ID:                  pm.ModelPublicID,
			Object:              "model",
			Created:             pm.CreatedAt.Unix(),
			OwnedBy:             provider.DisplayName,
			ModelDisplayName:    getModelDisplayName(pm),
			Category:            pm.Category,
			CategoryOrderNumber: pm.CategoryOrderNumber,
			ModelOrderNumber:    pm.ModelOrderNumber,
			ProviderID:          provider.PublicID,
			ProviderVendor:      strings.ToLower(string(provider.Kind)),
			ProviderName:        provider.DisplayName,
		})
	}

	// Sort by category_order_number (ASC), then model_order_number (ASC)
	// Treat 0 values as undefined and put them at the end
	sort.Slice(items, func(i, j int) bool {
		catI := items[i].CategoryOrderNumber
		catJ := items[j].CategoryOrderNumber
		modelI := items[i].ModelOrderNumber
		modelJ := items[j].ModelOrderNumber

		// If both category order numbers are 0, treat as undefined - compare by model order
		if catI == 0 && catJ == 0 {
			// If both model orders are also 0, keep original order (stable)
			if modelI == 0 && modelJ == 0 {
				return false
			}
			// Put 0 model order at the end
			if modelI == 0 {
				return false
			}
			if modelJ == 0 {
				return true
			}
			return modelI < modelJ
		}

		// Put category order 0 at the end
		if catI == 0 {
			return false
		}
		if catJ == 0 {
			return true
		}

		// Normal comparison: category first, then model order
		if catI != catJ {
			return catI < catJ
		}

		// Within same category, put model order 0 at the end
		if modelI == 0 {
			return false
		}
		if modelJ == 0 {
			return true
		}
		return modelI < modelJ
	})

	return items
}

func BuildModelResponseList(
	providerModels []*domainmodel.ProviderModel,
	providerByID map[uint]*domainmodel.Provider,
) []ModelResponse {
	items := make([]ModelResponse, 0, len(providerModels))

	for _, pm := range providerModels {
		if pm == nil {
			continue
		}
		provider := providerByID[pm.ProviderID]
		if provider == nil {
			continue
		}
		items = append(items, ModelResponse{
			ID:                  pm.ModelPublicID,
			Object:              "model",
			Created:             pm.CreatedAt.Unix(),
			OwnedBy:             provider.DisplayName,
			ModelDisplayName:    getModelDisplayName(pm),
			Category:            pm.Category,
			CategoryOrderNumber: pm.CategoryOrderNumber,
			ModelOrderNumber:    pm.ModelOrderNumber,
		})
	}

	// Sort by category_order_number (ASC), then model_order_number (ASC)
	// Treat 0 values as undefined and put them at the end
	sort.Slice(items, func(i, j int) bool {
		catI := items[i].CategoryOrderNumber
		catJ := items[j].CategoryOrderNumber
		modelI := items[i].ModelOrderNumber
		modelJ := items[j].ModelOrderNumber

		// If both category order numbers are 0, treat as undefined - compare by model order
		if catI == 0 && catJ == 0 {
			// If both model orders are also 0, keep original order (stable)
			if modelI == 0 && modelJ == 0 {
				return false
			}
			// Put 0 model order at the end
			if modelI == 0 {
				return false
			}
			if modelJ == 0 {
				return true
			}
			return modelI < modelJ
		}

		// Put category order 0 at the end
		if catI == 0 {
			return false
		}
		if catJ == 0 {
			return true
		}

		// Normal comparison: category first, then model order
		if catI != catJ {
			return catI < catJ
		}

		// Within same category, put model order 0 at the end
		if modelI == 0 {
			return false
		}
		if modelJ == 0 {
			return true
		}
		return modelI < modelJ
	})

	return items
}

func BuildProviderResponse(provider *domainmodel.Provider) ProviderResponse {
	return ProviderResponse{
		ID:        provider.PublicID,
		Name:      provider.DisplayName,
		Vendor:    strings.ToLower(string(provider.Kind)),
		BaseURL:   provider.BaseURL,
		Endpoints: buildEndpointResponses(provider.GetEndpoints()),
		Active:    provider.Active,
		Category:  string(provider.Category),
		DefaultProviderImageGenerate: provider.DefaultImageGenerate,
		DefaultProviderImageEdit:     provider.DefaultImageEdit,
		Metadata:  provider.Metadata,
	}
}

func BuildProviderWithModelCountResponse(
	provider *domainmodel.Provider,
	modelCount int64,
	activeCount int64,
) ProviderWithModelCountResponse {
	return ProviderWithModelCountResponse{
		ID:               provider.PublicID,
		Name:             provider.DisplayName,
		Vendor:           strings.ToLower(string(provider.Kind)),
		BaseURL:          provider.BaseURL,
		Endpoints:        buildEndpointResponses(provider.GetEndpoints()),
		Active:           provider.Active,
		Category:         string(provider.Category),
		DefaultProviderImageGenerate: provider.DefaultImageGenerate,
		DefaultProviderImageEdit:     provider.DefaultImageEdit,
		ModelCount:       modelCount,
		ModelActiveCount: activeCount,
		Metadata:         provider.Metadata,
	}
}

func BuildProviderWithModelsResponse(
	provider *domainmodel.Provider,
	models []*domainmodel.ProviderModel,
) *ProviderWithModelsResponse {
	if provider == nil {
		return nil
	}

	modelResponses := make([]ModelResponse, 0, len(models))
	for _, model := range models {
		if model == nil {
			continue
		}
		modelResponses = append(modelResponses, ModelResponse{
			ID:                  model.ModelPublicID,
			Object:              "model",
			Created:             model.CreatedAt.Unix(),
			OwnedBy:             provider.DisplayName,
			ModelDisplayName:    getModelDisplayName(model),
			Category:            model.Category,
			CategoryOrderNumber: model.CategoryOrderNumber,
			ModelOrderNumber:    model.ModelOrderNumber,
		})
	}
	return &ProviderWithModelsResponse{
		ID:        provider.PublicID,
		Name:      provider.DisplayName,
		Vendor:    strings.ToLower(string(provider.Kind)),
		BaseURL:   provider.BaseURL,
		Endpoints: buildEndpointResponses(provider.GetEndpoints()),
		Models:    modelResponses,
		Active:    provider.Active,
		Category:  string(provider.Category),
		DefaultProviderImageGenerate: provider.DefaultImageGenerate,
		DefaultProviderImageEdit:     provider.DefaultImageEdit,
		Metadata:  provider.Metadata,
	}
}

func BuildProviderResponseWithModels(
	provider *domainmodel.Provider,
	models []*domainmodel.ProviderModel,
) *ProviderWithModelsResponse {
	return BuildProviderWithModelsResponse(provider, models)
}

func BuildProviderResponseList(providers []*domainmodel.Provider) []ProviderResponse {
	items := make([]ProviderResponse, 0, len(providers))

	for _, provider := range providers {
		if provider == nil {
			continue
		}
		items = append(items, BuildProviderResponse(provider))
	}

	return items
}

type EndpointResponse struct {
	URL      string `json:"url"`
	Weight   int    `json:"weight"`
	Priority int    `json:"priority"`
	Healthy  bool   `json:"healthy"`
}

func buildEndpointResponses(endpoints domainmodel.EndpointList) []EndpointResponse {
	if len(endpoints) == 0 {
		return nil
	}
	result := make([]EndpointResponse, 0, len(endpoints))
	for _, ep := range endpoints {
		result = append(result, EndpointResponse{
			URL:      ep.URL,
			Weight:   ep.Weight,
			Priority: ep.Priority,
			Healthy:  ep.Healthy,
		})
	}
	return result
}

type ModelCatalogResponse struct {
	ID                  string                          `json:"id"`
	PublicID            string                          `json:"public_id"`
	ModelDisplayName    string                          `json:"model_display_name,omitempty"`
	Description         *string                         `json:"description,omitempty"`
	SupportedParameters domainmodel.SupportedParameters `json:"supported_parameters"`
	Architecture        domainmodel.Architecture        `json:"architecture"`
	Tags                []string                        `json:"tags,omitempty"`
	Notes               *string                         `json:"notes,omitempty"`
	ContextLength       *int                            `json:"context_length,omitempty"`
	IsModerated         *bool                           `json:"is_moderated,omitempty"`
	Active              *bool                           `json:"active,omitempty"`
	Extras              map[string]any                  `json:"extras,omitempty"`
	Status              domainmodel.ModelCatalogStatus  `json:"status"`
	Family              *string                         `json:"family,omitempty"`
	Experimental        bool                            `json:"experimental"`
	RequiresFeatureFlag *string                         `json:"requires_feature_flag,omitempty"`
	SupportsImages      bool                            `json:"supports_images"`
	SupportsEmbeddings  bool                            `json:"supports_embeddings"`
	SupportsReasoning   bool                            `json:"supports_reasoning"`
	SupportsInstruct    bool                            `json:"supports_instruct"`
	SupportsAudio       bool                            `json:"supports_audio"`
	SupportsVideo       bool                            `json:"supports_video"`
	SupportsTools       bool                            `json:"supports_tools"`
	SupportsBrowser     bool                            `json:"supports_browser"`
	LastSyncedAt        *int64                          `json:"last_synced_at,omitempty"`
	CreatedAt           int64                           `json:"created_at"`
	UpdatedAt           int64                           `json:"updated_at"`
}

type ProviderModelResponse struct {
	ID                      string                   `json:"id"`
	ProviderID              string                   `json:"provider_id"`
	ProviderVendor          string                   `json:"provider_vendor"`
	ModelCatalogID          *string                  `json:"model_catalog_id,omitempty"`
	ModelPublicID           string                   `json:"model_public_id"`
	ProviderOriginalModelID string                   `json:"provider_original_model_id"`
	ModelDisplayName        string                   `json:"model_display_name"`
	Category                string                   `json:"category"`
	CategoryOrderNumber     int                      `json:"category_order_number"`
	ModelOrderNumber        int                      `json:"model_order_number"`
	Pricing                 domainmodel.Pricing      `json:"pricing"`
	TokenLimits             *domainmodel.TokenLimits `json:"token_limits,omitempty"`
	Family                  *string                  `json:"family,omitempty"`
	SupportsImages          bool                     `json:"supports_images"`
	SupportsEmbeddings      bool                     `json:"supports_embeddings"`
	SupportsReasoning       bool                     `json:"supports_reasoning"`
	SupportsInstruct        bool                     `json:"supports_instruct"`
	SupportsAudio           bool                     `json:"supports_audio"`
	SupportsVideo           bool                     `json:"supports_video"`
	Active                  bool                     `json:"active"`
	InstructModelPublicID   *string                  `json:"instruct_model_public_id,omitempty"` // Public ID of the instruct model to use when enable_thinking=false
	CreatedAt               int64                    `json:"created_at"`
	UpdatedAt               int64                    `json:"updated_at"`
}

func BuildModelCatalogResponse(catalog *domainmodel.ModelCatalog) ModelCatalogResponse {
	var lastSyncedAt *int64
	if catalog.LastSyncedAt != nil {
		ts := catalog.LastSyncedAt.Unix()
		lastSyncedAt = &ts
	}
	var family *string
	if catalog.Family != "" {
		family = ptr.ToString(catalog.Family)
	}

	return ModelCatalogResponse{
		ID:                  catalog.PublicID,
		PublicID:            catalog.PublicID,
		ModelDisplayName:    catalog.ModelDisplayName,
		Description:         catalog.Description,
		SupportedParameters: catalog.SupportedParameters,
		Architecture:        catalog.Architecture,
		Tags:                catalog.Tags,
		Notes:               catalog.Notes,
		ContextLength:       catalog.ContextLength,
		IsModerated:         catalog.IsModerated,
		Active:              catalog.Active,
		Extras:              catalog.Extras,
		Status:              catalog.Status,
		Family:              family,
		Experimental:        catalog.Experimental,
		RequiresFeatureFlag: catalog.RequiresFeatureFlag,
		SupportsImages:      catalog.SupportsImages,
		SupportsEmbeddings:  catalog.SupportsEmbeddings,
		SupportsReasoning:   catalog.SupportsReasoning,
		SupportsInstruct:    catalog.SupportsInstruct,
		SupportsAudio:       catalog.SupportsAudio,
		SupportsVideo:       catalog.SupportsVideo,
		SupportsTools:       catalog.SupportsTools,
		SupportsBrowser:     catalog.SupportsBrowser,
		LastSyncedAt:        lastSyncedAt,
		CreatedAt:           catalog.CreatedAt.Unix(),
		UpdatedAt:           catalog.UpdatedAt.Unix(),
	}
}

func BuildProviderModelResponse(
	providerModel *domainmodel.ProviderModel,
	provider *domainmodel.Provider,
	modelCatalog *domainmodel.ModelCatalog,
	instructModelPublicID *string,
) ProviderModelResponse {
	var modelCatalogID *string
	if modelCatalog != nil {
		modelCatalogID = &modelCatalog.PublicID
	}

	// Get capabilities from model catalog (canonical source)
	var family *string
	var supportsImages, supportsEmbeddings, supportsReasoning, supportsInstruct, supportsAudio, supportsVideo bool
	if modelCatalog != nil {
		if modelCatalog.Family != "" {
			family = ptr.ToString(modelCatalog.Family)
		}
		supportsImages = modelCatalog.SupportsImages
		supportsEmbeddings = modelCatalog.SupportsEmbeddings
		supportsReasoning = modelCatalog.SupportsReasoning
		supportsInstruct = modelCatalog.SupportsInstruct
		supportsAudio = modelCatalog.SupportsAudio
		supportsVideo = modelCatalog.SupportsVideo
	}

	return ProviderModelResponse{
		ID:                      providerModel.PublicID,
		ProviderID:              provider.PublicID,
		ProviderVendor:          strings.ToLower(string(provider.Kind)),
		ModelCatalogID:          modelCatalogID,
		ModelPublicID:           providerModel.ModelPublicID,
		ProviderOriginalModelID: providerModel.ProviderOriginalModelID,
		ModelDisplayName:        providerModel.ModelDisplayName,
		Category:                providerModel.Category,
		CategoryOrderNumber:     providerModel.CategoryOrderNumber,
		ModelOrderNumber:        providerModel.ModelOrderNumber,
		Pricing:                 providerModel.Pricing,
		TokenLimits:             providerModel.TokenLimits,
		Family:                  family,
		SupportsImages:          supportsImages,
		SupportsEmbeddings:      supportsEmbeddings,
		SupportsReasoning:       supportsReasoning,
		SupportsInstruct:        supportsInstruct,
		SupportsAudio:           supportsAudio,
		SupportsVideo:           supportsVideo,
		Active:                  providerModel.Active,
		InstructModelPublicID:   instructModelPublicID,
		CreatedAt:               providerModel.CreatedAt.Unix(),
		UpdatedAt:               providerModel.UpdatedAt.Unix(),
	}
}

type BulkOperationResponse struct {
	UpdatedCount int      `json:"updated_count"`
	SkippedCount int      `json:"skipped_count,omitempty"`
	FailedCount  int      `json:"failed_count,omitempty"`
	TotalChecked int      `json:"total_checked,omitempty"`
	FailedModels []string `json:"failed_models,omitempty"`
}
