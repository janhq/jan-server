package modelhandler

import (
	"context"

	domainmodel "jan-server/mono/apps/backend/internal/domain/model"
	"jan-server/mono/apps/backend/internal/domain/query"
	requestmodels "jan-server/mono/apps/backend/internal/interfaces/httpserver/requests/models"
	modelresponses "jan-server/mono/apps/backend/internal/interfaces/httpserver/responses/model"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

type ProviderModelHandler struct {
	providerModelService *domainmodel.ProviderModelService
	providerService      *domainmodel.ProviderService
	modelCatalogService  *domainmodel.ModelCatalogService
}

func NewProviderModelHandler(
	providerModelService *domainmodel.ProviderModelService,
	providerService *domainmodel.ProviderService,
	modelCatalogService *domainmodel.ModelCatalogService,
) *ProviderModelHandler {
	return &ProviderModelHandler{
		providerModelService: providerModelService,
		providerService:      providerService,
		modelCatalogService:  modelCatalogService,
	}
}

func (h *ProviderModelHandler) GetProviderModel(ctx context.Context, publicID string) (*modelresponses.ProviderModelResponse, error) {
	if publicID == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "provider model public ID is required", nil, "14c0d733-6143-4eac-b09f-f5475895fec1")
	}

	providerModel, err := h.providerModelService.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get provider model")
	}

	if providerModel == nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "provider model not found", nil, "9820aaa1-2f72-4a92-ba9f-e84d4bb103ca")
	}

	provider, err := h.providerService.GetByID(ctx, providerModel.ProviderID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get provider")
	}

	var modelCatalog *domainmodel.ModelCatalog
	if providerModel.ModelCatalogID != nil {
		modelCatalog, _ = h.modelCatalogService.FindByID(ctx, *providerModel.ModelCatalogID)
	}

	// Resolve instruct model public ID if set
	var instructModelPublicID *string
	if providerModel.InstructModelID != nil {
		instructModel, err := h.providerModelService.FindByID(ctx, *providerModel.InstructModelID)
		if err == nil && instructModel != nil {
			instructModelPublicID = &instructModel.PublicID
		}
	}

	response := modelresponses.BuildProviderModelResponse(providerModel, provider, modelCatalog, instructModelPublicID)
	return &response, nil
}

func (h *ProviderModelHandler) ListProviderModels(
	ctx context.Context,
	filterParams requestmodels.ProviderModelFilterParams,
	pagination *query.Pagination,
) ([]modelresponses.ProviderModelResponse, int64, error) {
	filter := domainmodel.ProviderModelFilter{}

	if filterParams.ModelKey != nil {
		filter.ModelPublicID = filterParams.ModelKey
	}

	if filterParams.Active != nil {
		filter.Active = filterParams.Active
	}

	if filterParams.SearchText != nil {
		filter.SearchText = filterParams.SearchText
	}

	if filterParams.SupportsImages != nil {
		filter.SupportsImages = filterParams.SupportsImages
	}

	if filterParams.ProviderPublicID != nil {
		provider, err := h.providerService.FindByPublicID(ctx, *filterParams.ProviderPublicID)
		if err != nil {
			return nil, 0, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to find provider")
		}
		if provider != nil {
			filter.ProviderID = &provider.ID
		}
	}

	total, err := h.providerModelService.Count(ctx, filter)
	if err != nil {
		return nil, 0, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to count provider models")
	}

	providerModels, err := h.providerModelService.FindByFilterWithPagination(ctx, filter, pagination)
	if err != nil {
		return nil, 0, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to list provider models")
	}

	providerIDs := make(map[uint]bool)
	catalogIDs := make(map[uint]bool)
	for _, pm := range providerModels {
		providerIDs[pm.ProviderID] = true
		if pm.ModelCatalogID != nil {
			catalogIDs[*pm.ModelCatalogID] = true
		}
	}

	// Convert maps to slices for batch lookup
	providerIDSlice := make([]uint, 0, len(providerIDs))
	for id := range providerIDs {
		providerIDSlice = append(providerIDSlice, id)
	}

	catalogIDSlice := make([]uint, 0, len(catalogIDs))
	for id := range catalogIDs {
		catalogIDSlice = append(catalogIDSlice, id)
	}

	// Batch fetch providers and catalogs
	providerMap, err := h.providerService.GetByIDs(ctx, providerIDSlice)
	if err != nil {
		return nil, 0, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to fetch providers")
	}

	catalogMap, err := h.modelCatalogService.FindByIDs(ctx, catalogIDSlice)
	if err != nil {
		return nil, 0, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to fetch catalogs")
	}

	// Collect instruct model IDs for batch lookup
	instructModelIDs := make(map[uint]bool)
	for _, pm := range providerModels {
		if pm.InstructModelID != nil {
			instructModelIDs[*pm.InstructModelID] = true
		}
	}

	// Batch fetch instruct models
	instructModelMap := make(map[uint]*domainmodel.ProviderModel)
	if len(instructModelIDs) > 0 {
		instructModelIDSlice := make([]uint, 0, len(instructModelIDs))
		for id := range instructModelIDs {
			instructModelIDSlice = append(instructModelIDSlice, id)
		}
		instructModels, err := h.providerModelService.FindByIDs(ctx, instructModelIDSlice)
		if err == nil {
			for _, im := range instructModels {
				instructModelMap[im.ID] = im
			}
		}
	}

	result := make([]modelresponses.ProviderModelResponse, 0, len(providerModels))
	for _, pm := range providerModels {
		provider := providerMap[pm.ProviderID]
		if provider == nil {
			continue
		}

		var catalog *domainmodel.ModelCatalog
		if pm.ModelCatalogID != nil {
			catalog = catalogMap[*pm.ModelCatalogID]
		}

		// Resolve instruct model public ID
		var instructModelPublicID *string
		if pm.InstructModelID != nil {
			if instructModel := instructModelMap[*pm.InstructModelID]; instructModel != nil {
				instructModelPublicID = &instructModel.PublicID
			}
		}

		result = append(result, modelresponses.BuildProviderModelResponse(pm, provider, catalog, instructModelPublicID))
	}

	return result, total, nil
}

func (h *ProviderModelHandler) UpdateProviderModel(
	ctx context.Context,
	publicID string,
	req requestmodels.UpdateProviderModelRequest,
) (*modelresponses.ProviderModelResponse, error) {
	if publicID == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "provider model public ID is required", nil, "794588fc-4a61-4f1f-bec7-7041091da4d3")
	}

	providerModel, err := h.providerModelService.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get provider model")
	}

	if providerModel == nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "provider model not found", nil, "bef76423-07b2-438a-9899-c7f8ecac3e09")
	}

	if req.ModelDisplayName != nil {
		providerModel.ModelDisplayName = *req.ModelDisplayName
	}
	if req.Category != nil {
		providerModel.Category = *req.Category
	}
	if req.CategoryOrderNumber != nil {
		providerModel.CategoryOrderNumber = *req.CategoryOrderNumber
	}
	if req.ModelOrderNumber != nil {
		providerModel.ModelOrderNumber = *req.ModelOrderNumber
	}
	if req.Pricing != nil {
		providerModel.Pricing = *req.Pricing
	}
	if req.TokenLimits != nil {
		providerModel.TokenLimits = req.TokenLimits
	}
	if req.Active != nil {
		providerModel.Active = *req.Active
	}

	// Handle instruct model assignment
	if req.InstructModelPublicID != nil {
		if *req.InstructModelPublicID == "" {
			// Clear the instruct model
			providerModel.InstructModelID = nil
		} else {
			// Look up the instruct model by public ID
			instructModel, err := h.providerModelService.FindByPublicID(ctx, *req.InstructModelPublicID)
			if err != nil {
				return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to find instruct model")
			}
			if instructModel == nil {
				return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "instruct model not found", nil, "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d")
			}
			providerModel.InstructModelID = &instructModel.ID
		}
	}

	updatedModel, err := h.providerModelService.Update(ctx, providerModel)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to update provider model")
	}

	provider, err := h.providerService.GetByID(ctx, updatedModel.ProviderID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get provider")
	}

	var modelCatalog *domainmodel.ModelCatalog
	if updatedModel.ModelCatalogID != nil {
		modelCatalog, _ = h.modelCatalogService.FindByID(ctx, *updatedModel.ModelCatalogID)
	}

	// Resolve instruct model public ID if set
	var instructModelPublicID *string
	if updatedModel.InstructModelID != nil {
		instructModel, err := h.providerModelService.FindByID(ctx, *updatedModel.InstructModelID)
		if err == nil && instructModel != nil {
			instructModelPublicID = &instructModel.PublicID
		}
	}

	response := modelresponses.BuildProviderModelResponse(updatedModel, provider, modelCatalog, instructModelPublicID)
	return &response, nil
}

// Performs bulk enable/disable operations on provider models.
// Example use cases:
//   - "Disable all models except production whitelist"
//   - "Enable all OpenAI models except experimental ones"
//   - "Disable all models from a specific provider"
func (h *ProviderModelHandler) BulkEnableDisableProviderModels(
	ctx context.Context,
	req requestmodels.BulkEnableModelsRequest,
) (*modelresponses.BulkOperationResponse, error) {
	// Validate and normalize request
	if req.Enable == nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "enable field is required", nil, "3219f31b-1585-4e81-8976-06861cbf4358")
	}
	req.Normalize()

	filter := domainmodel.ProviderModelFilter{}

	if req.ProviderID != nil && *req.ProviderID != "" {
		provider, err := h.providerService.FindByPublicID(ctx, *req.ProviderID)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to find provider")
		}
		if provider == nil {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "provider not found", nil, "03a1267a-1daf-499e-b926-343c9429b5f5")
		}
		filter.ProviderID = &provider.ID
	}

	enableValue := *req.Enable
	exceptModelKeys := make(map[string]bool)
	for _, key := range req.ExceptModels {
		exceptModelKeys[key] = true
	}

	totalCount, err := h.providerModelService.Count(ctx, filter)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to count provider models")
	}

	var modelsUpdated int64
	var skippedCount int64

	if len(exceptModelKeys) > 0 {
		allModels, err := h.providerModelService.FindByFilter(ctx, filter)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to list provider models")
		}

		idsToUpdate := make([]uint, 0)
		for _, model := range allModels {
			if exceptModelKeys[model.ModelPublicID] {
				skippedCount++
				continue
			}
			if model.Active == enableValue {
				skippedCount++
				continue
			}
			idsToUpdate = append(idsToUpdate, model.ID)
		}

		if len(idsToUpdate) > 0 {
			updateFilter := domainmodel.ProviderModelFilter{
				IDs: &idsToUpdate,
			}
			updated, err := h.providerModelService.BatchUpdateActive(ctx, updateFilter, enableValue)
			if err != nil {
				return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to batch update provider models")
			}
			modelsUpdated = updated
		}
	} else {
		modelsUpdated, err = h.providerModelService.BatchUpdateActive(ctx, filter, enableValue)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to batch update provider models")
		}
		skippedCount = totalCount - modelsUpdated
	}

	return &modelresponses.BulkOperationResponse{
		UpdatedCount: int(modelsUpdated),
		SkippedCount: int(skippedCount),
		FailedCount:  0,
		TotalChecked: int(totalCount),
	}, nil
}
