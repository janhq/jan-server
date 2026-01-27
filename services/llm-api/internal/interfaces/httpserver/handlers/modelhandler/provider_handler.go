package modelhandler

import (
	"context"
	"strings"

	domainmodel "jan-server/services/llm-api/internal/domain/model"
	"jan-server/services/llm-api/internal/infrastructure/inference"
	requestmodels "jan-server/services/llm-api/internal/interfaces/httpserver/requests/models"
	modelresponses "jan-server/services/llm-api/internal/interfaces/httpserver/responses/model"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

type ProviderHandler struct {
	providerService      *domainmodel.ProviderService
	providerModelService *domainmodel.ProviderModelService
	inferenceProvider    *inference.InferenceProvider
}

func NewProviderHandler(
	providerService *domainmodel.ProviderService,
	providerModelService *domainmodel.ProviderModelService,
	inferenceProvider *inference.InferenceProvider,
) *ProviderHandler {
	return &ProviderHandler{
		providerService:      providerService,
		providerModelService: providerModelService,
		inferenceProvider:    inferenceProvider,
	}
}

func (providerHandler *ProviderHandler) RegisterProvider(addProviderRequest requestmodels.AddProviderRequest, ctx context.Context) (*modelresponses.ProviderWithModelsResponse, error) {

	// Check if provider with the same vendor already exists if vendor != "custom"
	provider, err := providerHandler.providerService.FindProviderByVendor(ctx, addProviderRequest.Vendor)
	if err == nil && provider != nil && provider.Kind != domainmodel.ProviderCustom {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeConflict, "provider with vendor already exists", nil, "30c583bd-82f0-41c7-83ca-c2bf071cb018")
	}

	endpoints, err := addProviderRequest.GetEndpointList()
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "invalid endpoints")
	}

	active := true
	if addProviderRequest.Active != nil {
		active = *addProviderRequest.Active
	}
	defaultImageGenerate := false
	if addProviderRequest.DefaultProviderImageGenerate != nil {
		defaultImageGenerate = *addProviderRequest.DefaultProviderImageGenerate
	}
	defaultImageEdit := false
	if addProviderRequest.DefaultProviderImageEdit != nil {
		defaultImageEdit = *addProviderRequest.DefaultProviderImageEdit
	}

	result, err := providerHandler.providerService.RegisterProvider(ctx, domainmodel.RegisterProviderInput{
		Name:      addProviderRequest.Name,
		Vendor:    addProviderRequest.Vendor,
		BaseURL:   addProviderRequest.BaseURL,
		Endpoints: endpoints,
		APIKey:    addProviderRequest.APIKey,
		Metadata:  addProviderRequest.Metadata,
		Active:    active,
		Category:  domainmodel.ProviderCategory(addProviderRequest.Category),
		DefaultImageGenerate: defaultImageGenerate,
		DefaultImageEdit:     defaultImageEdit,
	})
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "register provider failed")
	}

	// Only LLM providers expose /v1/models; skip sync for OCR/image/custom non-LLM categories.
	if result.Category != domainmodel.ProviderCategoryLLM {
		return modelresponses.BuildProviderResponseWithModels(result, nil), nil
	}

	models, err := providerHandler.inferenceProvider.ListModels(ctx, result)
	if err != nil {
		return nil, err
	}
	syncModels, syncErr := providerHandler.providerService.SyncProviderModelsWithOptions(ctx, result, models, true)
	if syncErr != nil {
		return nil, syncErr
	}

	return modelresponses.BuildProviderResponseWithModels(result, syncModels), nil
}

func (providerHandler *ProviderHandler) GetAllProviders(ctx context.Context, filter domainmodel.ProviderFilter) ([]modelresponses.ProviderWithModelCountResponse, error) {
	providers, err := providerHandler.providerService.FindProviders(ctx, filter)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get providers")
	}

	if len(providers) == 0 {
		return []modelresponses.ProviderWithModelCountResponse{}, nil
	}

	providerIDs := make([]uint, len(providers))
	for i, provider := range providers {
		providerIDs[i] = provider.ID
	}

	modelCounts, err := providerHandler.providerModelService.FindModelCountsByProviderIDs(ctx, providerIDs)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get model counts")
	}

	activeModelCounts, err := providerHandler.providerModelService.FindActiveModelCountsByProviderIDs(ctx, providerIDs)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get active model counts")
	}

	result := make([]modelresponses.ProviderWithModelCountResponse, 0, len(providers))
	for _, provider := range providers {
		modelCount := modelCounts[provider.ID]
		activeModelCount := activeModelCounts[provider.ID]
		result = append(result, modelresponses.BuildProviderWithModelCountResponse(provider, modelCount, activeModelCount))
	}

	return result, nil
}

func (providerHandler *ProviderHandler) GetProviderByPublicID(ctx context.Context, publicID string) (*modelresponses.ProviderResponse, error) {
	if strings.TrimSpace(publicID) == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "provider public ID is required", nil, "d5d9dc70-4dd1-4063-b0be-990d69ee7086")
	}

	provider, err := providerHandler.providerService.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to find provider")
	}
	if provider == nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "provider not found", nil, "97728060-34f7-4451-8b9a-5df38ac3364f")
	}

	response := modelresponses.BuildProviderResponse(provider)
	return &response, nil
}

func (providerHandler *ProviderHandler) SelectProviderModelForModelPublicID(ctx context.Context, modelPublicID string) (*domainmodel.ProviderModel, *domainmodel.Provider, error) {
	if strings.TrimSpace(modelPublicID) == "" {
		return nil, nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "model key is required", nil, "abeb247f-ef80-44bf-921b-6e2c92ffca73")
	}
	var providerModels []*domainmodel.ProviderModel

	providerModels, err := providerHandler.providerModelService.FindActiveByModelKey(ctx, modelPublicID)
	if err != nil {
		return nil, nil, err
	}

	if len(providerModels) == 0 {
		return nil, nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "model not found in accessible providers", nil, "caa8476d-1b95-42a7-a96b-18b0c11b2f64")
	}

	selectedProviderModel := providerHandler.selectBestProvider(providerModels)
	if selectedProviderModel == nil {
		return nil, nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "no valid provider found for model", nil, "265747b1-0aee-4a99-863e-99a7af8ada5e")
	}

	selectedProvider, err := providerHandler.providerService.GetByID(ctx, selectedProviderModel.ProviderID)
	if err != nil {
		return nil, nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get provider details")
	}
	return selectedProviderModel, selectedProvider, nil
}

func (providerHandler *ProviderHandler) SelectProviderModelForProviderOriginalModelID(ctx context.Context, modelID string) (*domainmodel.ProviderModel, *domainmodel.Provider, error) {
	if strings.TrimSpace(modelID) == "" {
		return nil, nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "provider original model id is required", nil, "d4d3a5a9-9cb2-4c28-8d1c-5ce0b5b0e02c")
	}

	providerModels, err := providerHandler.providerModelService.FindActiveByProviderOriginalModelID(ctx, modelID)
	if err != nil {
		return nil, nil, err
	}

	if len(providerModels) == 0 {
		return nil, nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "model not found in accessible providers", nil, "7ad98cc6-2d1d-4ed6-9e1b-7b74855958ee")
	}

	selectedProviderModel := providerHandler.selectBestProvider(providerModels)
	if selectedProviderModel == nil {
		return nil, nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "no valid provider found for model", nil, "ca2e22b9-7c5b-42f6-a8b7-22b9c6a533a2")
	}

	selectedProvider, err := providerHandler.providerService.GetByID(ctx, selectedProviderModel.ProviderID)
	if err != nil {
		return nil, nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get provider details")
	}
	return selectedProviderModel, selectedProvider, nil
}

func (providerHandler *ProviderHandler) SelectProviderModelForModelPublicIDIncludingInactive(ctx context.Context, modelPublicID string) (*domainmodel.ProviderModel, *domainmodel.Provider, error) {
	if strings.TrimSpace(modelPublicID) == "" {
		return nil, nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "model key is required", nil, "0eb0c36b-5c68-4a03-9ab5-acde7c967713")
	}

	providerModels, err := providerHandler.providerModelService.FindByModelKey(ctx, modelPublicID)
	if err != nil {
		return nil, nil, err
	}

	if len(providerModels) == 0 {
		return nil, nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "model not found in accessible providers", nil, "1c8f485c-78b0-4a79-9605-213e96c4ef7a")
	}

	selectedProviderModel := providerHandler.selectBestProvider(providerModels)
	if selectedProviderModel == nil {
		return nil, nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "no valid provider found for model", nil, "2d66c2d4-49ef-4f7d-9136-c27f2a1f6d5f")
	}

	selectedProvider, err := providerHandler.providerService.GetByID(ctx, selectedProviderModel.ProviderID)
	if err != nil {
		return nil, nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get provider details")
	}
	return selectedProviderModel, selectedProvider, nil
}

func (providerHandler *ProviderHandler) SelectProviderModelForProviderOriginalModelIDIncludingInactive(ctx context.Context, modelID string) (*domainmodel.ProviderModel, *domainmodel.Provider, error) {
	if strings.TrimSpace(modelID) == "" {
		return nil, nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "provider original model id is required", nil, "19c1e63f-0b61-45d5-900b-5dbf2a4950aa")
	}

	providerModels, err := providerHandler.providerModelService.FindByProviderOriginalModelID(ctx, modelID)
	if err != nil {
		return nil, nil, err
	}

	if len(providerModels) == 0 {
		return nil, nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "model not found in accessible providers", nil, "83c1c7d6-c9a1-43b0-85b4-e21ef46e33b5")
	}

	selectedProviderModel := providerHandler.selectBestProvider(providerModels)
	if selectedProviderModel == nil {
		return nil, nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "no valid provider found for model", nil, "98af855f-2d3f-47ba-bc8f-5e4d1d45d0a6")
	}

	selectedProvider, err := providerHandler.providerService.GetByID(ctx, selectedProviderModel.ProviderID)
	if err != nil {
		return nil, nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get provider details")
	}
	return selectedProviderModel, selectedProvider, nil
}

// selectBestProvider selects the best provider for a model based on:
// 1. LOWEST PRICING (if pricing data exists)
// 2. MENLO PROVIDER (if prices are equal or no pricing)
// 3. FIRST PROVIDER (if all criteria equal)
func (providerHandler *ProviderHandler) selectBestProvider(
	providerModels []*domainmodel.ProviderModel,
) *domainmodel.ProviderModel {
	if len(providerModels) == 0 {
		return nil
	}

	type providerCandidate struct {
		providerModel *domainmodel.ProviderModel
		hasPricing    bool
		lowestPrice   domainmodel.MicroUSD
		isJan         bool
	}

	candidates := make([]providerCandidate, 0, len(providerModels))

	for _, providerModel := range providerModels {
		if providerModel == nil {
			continue
		}

		lowestPrice, hasPricing := calculateLowestPrice(providerModel.Pricing)
		isJan := providerModel.Kind == domainmodel.ProviderJan

		candidates = append(candidates, providerCandidate{
			providerModel: providerModel,
			hasPricing:    hasPricing,
			lowestPrice:   lowestPrice,
			isJan:         isJan,
		})
	}

	if len(candidates) == 0 {
		return nil
	}

	// Find the best candidate
	best := candidates[0]
	for i := 1; i < len(candidates); i++ {
		candidate := candidates[i]

		// Compare pricing first (if both have pricing)
		if candidate.hasPricing && best.hasPricing {
			if candidate.lowestPrice < best.lowestPrice {
				best = candidate
				continue
			} else if candidate.lowestPrice > best.lowestPrice {
				continue
			}
			// Prices are equal, continue to next criteria
		}

		// Prefer candidate with pricing over one without
		if candidate.hasPricing && !best.hasPricing {
			best = candidate
			continue
		}
		if !candidate.hasPricing && best.hasPricing {
			continue
		}

		// Prefer Jan provider
		if candidate.isJan && !best.isJan {
			best = candidate
			continue
		}
		if !candidate.isJan && best.isJan {
			continue
		}

	}

	return best.providerModel
}

func (h *ProviderHandler) UpdateProvider(
	ctx context.Context,
	publicID string,
	req requestmodels.UpdateProviderRequest,
) (*modelresponses.ProviderResponse, error) {
	if publicID == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "provider public ID is required", nil, "140e34cf-ed9f-4008-9d9b-c3e7b9d183b8")
	}

	provider, err := h.providerService.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to find provider")
	}
	if provider == nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "provider not found", nil, "0d77a312-f914-492d-8dbc-7f1ba9d14da9")
	}

	endpoints, provided, err := req.GetEndpointList()
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "invalid endpoints")
	}

	updateInput := domainmodel.UpdateProviderInput{
		Name:    req.Name,
		BaseURL: req.BaseURL,
		Endpoints: func() *domainmodel.EndpointList {
			if provided {
				return &endpoints
			}
			return nil
		}(),
		APIKey:   req.APIKey,
		Metadata: req.Metadata,
		Active:   req.Active,
		DefaultImageGenerate: req.DefaultProviderImageGenerate,
		DefaultImageEdit:     req.DefaultProviderImageEdit,
	}

	updatedProvider, err := h.providerService.UpdateProvider(ctx, provider, updateInput)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to update provider")
	}

	response := modelresponses.BuildProviderResponse(updatedProvider)
	return &response, nil
}

// GetModelCatalogByID returns catalog details (used to apply default parameters).
func (providerHandler *ProviderHandler) GetModelCatalogByID(ctx context.Context, id uint) (*domainmodel.ModelCatalog, error) {
	return providerHandler.providerModelService.FindCatalogByID(ctx, id)
}

// GetProviderModelByID returns a provider model and its provider by internal ID.
// This is used for instruct model fallback when enable_thinking=false.
func (providerHandler *ProviderHandler) GetProviderModelByID(ctx context.Context, id uint) (*domainmodel.ProviderModel, *domainmodel.Provider, error) {
	providerModel, err := providerHandler.providerModelService.FindByID(ctx, id)
	if err != nil {
		return nil, nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to find provider model")
	}
	if providerModel == nil {
		return nil, nil, nil
	}

	provider, err := providerHandler.providerService.GetByID(ctx, providerModel.ProviderID)
	if err != nil {
		return nil, nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get provider")
	}

	return providerModel, provider, nil
}

func (h *ProviderHandler) DeleteProvider(ctx context.Context, publicID string) error {
	if strings.TrimSpace(publicID) == "" {
		return platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "provider public ID is required", nil, "0c3f68da-0aa4-4a7c-9cec-c22d47c86f8b")
	}

	if err := h.providerService.DeleteProviderByPublicID(ctx, publicID); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to delete provider")
	}

	return nil
}

// TODO(pricing): Remove pricing calculation from model handler
// This function calculates the lowest price for a provider model, but pricing logic
// should be handled by a dedicated billing domain, not in the model management layer.
// Consider removing this once pricing is moved to the billing domain.
// Related: See TODO in internal/domain/model/provider_model.go
func calculateLowestPrice(pricing domainmodel.Pricing) (domainmodel.MicroUSD, bool) {
	if len(pricing.Lines) == 0 {
		return 0, false
	}

	lowest := pricing.Lines[0].Amount
	for _, line := range pricing.Lines[1:] {
		if line.Amount < lowest {
			lowest = line.Amount
		}
	}

	return lowest, true
}
