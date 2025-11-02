package modelresponses

import (
	"strings"

	domainmodel "menlo.ai/menlo-platform/internal/domain/model"
)

type ModelResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ModelResponseList struct {
	Object string          `json:"object"`
	Data   []ModelResponse `json:"data"`
}

type ModelResponseWithProvider struct {
	ID             string `json:"id"`
	Object         string `json:"object"`
	Created        int64  `json:"created"`
	OwnedBy        string `json:"owned_by"`
	ProviderID     string `json:"provider_id"`
	ProviderVendor string `json:"provider_vendor"`
	ProviderName   string `json:"provider_name"`
}

type ModelWithProviderResponseList struct {
	Object string                      `json:"object"`
	Data   []ModelResponseWithProvider `json:"data"`
}

type ProviderResponse struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Vendor   string            `json:"vendor"`
	BaseURL  string            `json:"base_url"`
	Active   bool              `json:"active"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type ProviderWithModelCountResponse struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Vendor           string            `json:"vendor"`
	BaseURL          string            `json:"base_url"`
	Active           bool              `json:"active"`
	ModelCount       int64             `json:"model_count"`
	ModelActiveCount int64             `json:"model_active_count"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type ProviderWithModelsResponse struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Vendor   string            `json:"vendor"`
	BaseURL  string            `json:"base_url"`
	Models   []ModelResponse   `json:"models"`
	Active   bool              `json:"active"`
	Metadata map[string]string `json:"metadata,omitempty"`
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
			ID:             pm.ModelPublicID,
			Object:         "model",
			Created:        pm.CreatedAt.Unix(),
			OwnedBy:        provider.DisplayName,
			ProviderID:     provider.PublicID,
			ProviderVendor: strings.ToLower(string(provider.Kind)),
			ProviderName:   provider.DisplayName,
		})
	}

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
			ID:      pm.ModelPublicID,
			Object:  "model",
			Created: pm.CreatedAt.Unix(),
			OwnedBy: provider.DisplayName,
		})
	}

	return items
}

func BuildProviderResponse(provider *domainmodel.Provider) ProviderResponse {
	return ProviderResponse{
		ID:       provider.PublicID,
		Name:     provider.DisplayName,
		Vendor:   strings.ToLower(string(provider.Kind)),
		BaseURL:  provider.BaseURL,
		Active:   provider.Active,
		Metadata: provider.Metadata,
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
		Active:           provider.Active,
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
			ID:      model.ModelPublicID,
			Object:  "model",
			Created: model.CreatedAt.Unix(),
			OwnedBy: provider.DisplayName,
		})
	}
	return &ProviderWithModelsResponse{
		ID:       provider.PublicID,
		Name:     provider.DisplayName,
		Vendor:   strings.ToLower(string(provider.Kind)),
		BaseURL:  provider.BaseURL,
		Models:   modelResponses,
		Active:   provider.Active,
		Metadata: provider.Metadata,
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

type ModelCatalogResponse struct {
	ID                  string                          `json:"id"`
	SupportedParameters domainmodel.SupportedParameters `json:"supported_parameters"`
	Architecture        domainmodel.Architecture        `json:"architecture"`
	Tags                []string                        `json:"tags,omitempty"`
	Notes               *string                         `json:"notes,omitempty"`
	IsModerated         *bool                           `json:"is_moderated,omitempty"`
	Active              *bool                           `json:"active,omitempty"`
	Extras              map[string]any                  `json:"extras,omitempty"`
	Status              domainmodel.ModelCatalogStatus  `json:"status"`
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
	DisplayName             string                   `json:"display_name"`
	Pricing                 domainmodel.Pricing      `json:"pricing"`
	TokenLimits             *domainmodel.TokenLimits `json:"token_limits,omitempty"`
	Family                  *string                  `json:"family,omitempty"`
	SupportsImages          bool                     `json:"supports_images"`
	SupportsEmbeddings      bool                     `json:"supports_embeddings"`
	SupportsReasoning       bool                     `json:"supports_reasoning"`
	SupportsAudio           bool                     `json:"supports_audio"`
	SupportsVideo           bool                     `json:"supports_video"`
	Active                  bool                     `json:"active"`
	CreatedAt               int64                    `json:"created_at"`
	UpdatedAt               int64                    `json:"updated_at"`
}

func BuildModelCatalogResponse(catalog *domainmodel.ModelCatalog) ModelCatalogResponse {
	var lastSyncedAt *int64
	if catalog.LastSyncedAt != nil {
		ts := catalog.LastSyncedAt.Unix()
		lastSyncedAt = &ts
	}

	return ModelCatalogResponse{
		ID:                  catalog.PublicID,
		SupportedParameters: catalog.SupportedParameters,
		Architecture:        catalog.Architecture,
		Tags:                catalog.Tags,
		Notes:               catalog.Notes,
		IsModerated:         catalog.IsModerated,
		Active:              catalog.Active,
		Extras:              catalog.Extras,
		Status:              catalog.Status,
		LastSyncedAt:        lastSyncedAt,
		CreatedAt:           catalog.CreatedAt.Unix(),
		UpdatedAt:           catalog.UpdatedAt.Unix(),
	}
}

func BuildProviderModelResponse(
	providerModel *domainmodel.ProviderModel,
	provider *domainmodel.Provider,
	modelCatalog *domainmodel.ModelCatalog,
) ProviderModelResponse {
	var modelCatalogID *string
	if modelCatalog != nil {
		modelCatalogID = &modelCatalog.PublicID
	}

	return ProviderModelResponse{
		ID:                      providerModel.PublicID,
		ProviderID:              provider.PublicID,
		ProviderVendor:          strings.ToLower(string(provider.Kind)),
		ModelCatalogID:          modelCatalogID,
		ModelPublicID:           providerModel.ModelPublicID,
		ProviderOriginalModelID: providerModel.ProviderOriginalModelID,
		DisplayName:             providerModel.DisplayName,
		Pricing:                 providerModel.Pricing,
		TokenLimits:             providerModel.TokenLimits,
		Family:                  providerModel.Family,
		SupportsImages:          providerModel.SupportsImages,
		SupportsEmbeddings:      providerModel.SupportsEmbeddings,
		SupportsReasoning:       providerModel.SupportsReasoning,
		SupportsAudio:           providerModel.SupportsAudio,
		SupportsVideo:           providerModel.SupportsVideo,
		Active:                  providerModel.Active,
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
