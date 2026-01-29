package modelhandler

import (
	"context"

	domainmodel "jan-server/mono/apps/backend/internal/domain/model"
	"jan-server/mono/apps/backend/internal/domain/query"
	requestmodels "jan-server/mono/apps/backend/internal/interfaces/httpserver/requests/models"
	modelresponses "jan-server/mono/apps/backend/internal/interfaces/httpserver/responses/model"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

type ModelCatalogHandler struct {
	modelCatalogService  *domainmodel.ModelCatalogService
	providerModelService *domainmodel.ProviderModelService
}

func NewModelCatalogHandler(
	modelCatalogService *domainmodel.ModelCatalogService,
	providerModelService *domainmodel.ProviderModelService,
) *ModelCatalogHandler {
	return &ModelCatalogHandler{
		modelCatalogService:  modelCatalogService,
		providerModelService: providerModelService,
	}
}

func (h *ModelCatalogHandler) GetCatalog(ctx context.Context, publicID string) (*modelresponses.ModelCatalogResponse, error) {
	if publicID == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "model catalog public ID is required", nil, "c9076125-ba1b-496d-b55f-c1711af98eaa")
	}

	catalog, err := h.modelCatalogService.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get model catalog")
	}

	response := modelresponses.BuildModelCatalogResponse(catalog)
	return &response, nil
}

func (h *ModelCatalogHandler) UpdateCatalog(
	ctx context.Context,
	publicID string,
	req requestmodels.UpdateModelCatalogRequest,
) (*modelresponses.ModelCatalogResponse, error) {
	if publicID == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "model catalog public ID is required", nil, "b371fe0d-7c3e-41b2-b98f-fb7b2b6cf54a")
	}

	catalog, err := h.modelCatalogService.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get model catalog")
	}

	// Update fields if provided
	if req.SupportedParameters != nil {
		catalog.SupportedParameters = *req.SupportedParameters
	}
	if req.Architecture != nil {
		catalog.Architecture = *req.Architecture
	}
	if req.Tags != nil {
		catalog.Tags = *req.Tags
	}
	if req.Notes != nil {
		catalog.Notes = req.Notes
	}
	if req.Description != nil {
		catalog.Description = req.Description
	}
	if req.IsModerated != nil {
		catalog.IsModerated = req.IsModerated
	}
	if req.Extras != nil {
		catalog.Extras = *req.Extras
	}
	if req.Active != nil {
		catalog.Active = req.Active
	}
	if req.Experimental != nil {
		catalog.Experimental = *req.Experimental
	}
	if req.RequiresFeatureFlag != nil {
		catalog.RequiresFeatureFlag = req.RequiresFeatureFlag
	}
	if req.SupportsImages != nil {
		catalog.SupportsImages = *req.SupportsImages
	}
	if req.SupportsEmbeddings != nil {
		catalog.SupportsEmbeddings = *req.SupportsEmbeddings
	}
	if req.SupportsReasoning != nil {
		catalog.SupportsReasoning = *req.SupportsReasoning
	}
	if req.SupportsInstruct != nil {
		catalog.SupportsInstruct = *req.SupportsInstruct
	}
	if req.SupportsAudio != nil {
		catalog.SupportsAudio = *req.SupportsAudio
	}
	if req.SupportsVideo != nil {
		catalog.SupportsVideo = *req.SupportsVideo
	}
	if req.SupportsTools != nil {
		catalog.SupportsTools = *req.SupportsTools
	}
	if req.SupportsBrowser != nil {
		catalog.SupportsBrowser = *req.SupportsBrowser
	}
	if req.Family != nil {
		catalog.Family = *req.Family
	}
	if req.ModelDisplayName != nil {
		catalog.ModelDisplayName = *req.ModelDisplayName
		// Update model_display_name in all provider_models that reference this catalog
		if catalog.ID != 0 {
			filter := domainmodel.ProviderModelFilter{
				ModelCatalogID: &catalog.ID,
			}
			_, err := h.providerModelService.BatchUpdateModelDisplayName(ctx, filter, *req.ModelDisplayName)
			if err != nil {
				return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to update provider models display name")
			}
		}
	}
	if req.ContextLength != nil {
		val := int(*req.ContextLength)
		catalog.ContextLength = &val
	}

	// Mark as updated by admin (prevents auto-sync from overwriting)
	catalog.Status = domainmodel.ModelCatalogStatusUpdated

	updatedCatalog, err := h.modelCatalogService.Update(ctx, catalog)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to update model catalog")
	}

	response := modelresponses.BuildModelCatalogResponse(updatedCatalog)
	return &response, nil
}

func (h *ModelCatalogHandler) ListCatalogs(
	ctx context.Context,
	filterParams requestmodels.ModelCatalogFilterParams,
	pagination *query.Pagination,
) ([]modelresponses.ModelCatalogResponse, int64, error) {
	filter := domainmodel.ModelCatalogFilter{}

	if filterParams.Status != nil {
		status := domainmodel.ModelCatalogStatus(*filterParams.Status)
		filter.Status = &status
	}

	if filterParams.IsModerated != nil {
		filter.IsModerated = filterParams.IsModerated
	}

	if filterParams.Active != nil {
		filter.Active = filterParams.Active
	}

	if filterParams.Experimental != nil {
		filter.Experimental = filterParams.Experimental
	}

	if filterParams.RequiresFeatureFlag != nil {
		filter.RequiresFeatureFlag = filterParams.RequiresFeatureFlag
	}

	if filterParams.SupportsImages != nil {
		filter.SupportsImages = filterParams.SupportsImages
	}

	if filterParams.SupportsEmbeddings != nil {
		filter.SupportsEmbeddings = filterParams.SupportsEmbeddings
	}

	if filterParams.SupportsReasoning != nil {
		filter.SupportsReasoning = filterParams.SupportsReasoning
	}

	if filterParams.SupportsAudio != nil {
		filter.SupportsAudio = filterParams.SupportsAudio
	}

	if filterParams.SupportsVideo != nil {
		filter.SupportsVideo = filterParams.SupportsVideo
	}

	if filterParams.SupportsTools != nil {
		filter.SupportsTools = filterParams.SupportsTools
	}

	if filterParams.Family != nil {
		filter.Family = filterParams.Family
	}

	total, err := h.modelCatalogService.Count(ctx, filter)
	if err != nil {
		return nil, 0, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to count model catalogs")
	}

	catalogs, err := h.modelCatalogService.FindByFilter(ctx, filter, pagination)
	if err != nil {
		return nil, 0, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to list model catalogs")
	}

	result := make([]modelresponses.ModelCatalogResponse, 0, len(catalogs))
	for _, catalog := range catalogs {
		result = append(result, modelresponses.BuildModelCatalogResponse(catalog))
	}

	return result, total, nil
}

func (h *ModelCatalogHandler) BulkToggleCatalogs(ctx context.Context, req requestmodels.BulkToggleCatalogsRequest) (*modelresponses.BulkOperationResponse, error) {
	// Validate and normalize request
	if req.Enable == nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "enable field is required", nil, "6080cd28-097d-4ba1-b740-9e50a1345461")
	}
	req.Normalize()

	var catalogIDs []uint
	exceptModelKeys := make(map[string]bool)
	for _, key := range req.ExceptModels {
		exceptModelKeys[key] = true
	}

	if len(req.CatalogIDs) > 0 {
		// Use batch method to fetch all catalogs in a single query
		catalogsMap, err := h.modelCatalogService.FindByPublicIDs(ctx, req.CatalogIDs)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to find catalogs")
		}

		// Validate all requested catalog IDs were found
		if len(catalogsMap) != len(req.CatalogIDs) {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound, "some catalog(s) not found", nil, "e8b4a9b6-dc74-4b53-9ca5-9652bbd96b80")
		} // Extract catalog IDs
		catalogIDs = make([]uint, 0, len(catalogsMap))
		for _, catalog := range catalogsMap {
			catalogIDs = append(catalogIDs, catalog.ID)
		}
	} else {
		allCatalogs, err := h.modelCatalogService.FindByFilter(ctx, domainmodel.ModelCatalogFilter{}, nil)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to retrieve all catalogs")
		}
		catalogIDs = make([]uint, 0, len(allCatalogs))
		for _, catalog := range allCatalogs {
			catalogIDs = append(catalogIDs, catalog.ID)
		}
	}

	if len(catalogIDs) == 0 {
		return &modelresponses.BulkOperationResponse{
			UpdatedCount: 0,
			SkippedCount: 0,
			TotalChecked: 0,
		}, nil
	}

	enableValue := *req.Enable

	catalogFilter := domainmodel.ModelCatalogFilter{
		IDs: &catalogIDs,
	}
	catalogsUpdated, err := h.modelCatalogService.BatchUpdateActive(ctx, catalogFilter, enableValue)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to batch update catalogs")
	}

	var modelsUpdated int64
	var totalModelsChecked int64
	var skippedCount int64

	if !enableValue {
		for _, catalogID := range catalogIDs {
			filter := domainmodel.ProviderModelFilter{
				ModelCatalogID: &catalogID,
			}

			count, err := h.providerModelService.Count(ctx, filter)
			if err != nil {
				return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to count provider models for catalog")
			}
			totalModelsChecked += count

			if len(exceptModelKeys) > 0 {
				allModels, err := h.providerModelService.FindByFilter(ctx, filter)
				if err != nil {
					return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to retrieve provider models for catalog")
				}

				idsToUpdate := make([]uint, 0)
				for _, pm := range allModels {
					if exceptModelKeys[pm.ModelPublicID] {
						skippedCount++
						continue
					}
					idsToUpdate = append(idsToUpdate, pm.ID)
				}

				if len(idsToUpdate) > 0 {
					updateFilter := domainmodel.ProviderModelFilter{
						IDs: &idsToUpdate,
					}
					updated, err := h.providerModelService.BatchUpdateActive(ctx, updateFilter, false)
					if err != nil {
						return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to batch update provider models")
					}
					modelsUpdated += updated
				}
			} else {
				updated, err := h.providerModelService.BatchUpdateActive(ctx, filter, false)
				if err != nil {
					return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to batch update provider models")
				}
				modelsUpdated += updated
			}
		}
	} else {
		for _, catalogID := range catalogIDs {
			filter := domainmodel.ProviderModelFilter{
				ModelCatalogID: &catalogID,
			}
			count, err := h.providerModelService.Count(ctx, filter)
			if err != nil {
				return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to count provider models")
			}
			totalModelsChecked += count
			skippedCount += count
		}
	}

	return &modelresponses.BulkOperationResponse{
		UpdatedCount: int(catalogsUpdated + modelsUpdated),
		SkippedCount: int(skippedCount),
		FailedCount:  0,
		TotalChecked: int(totalModelsChecked),
	}, nil
}
