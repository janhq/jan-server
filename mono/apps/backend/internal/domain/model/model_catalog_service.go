package model

import (
	"context"
	"time"

	"jan-server/mono/apps/backend/internal/domain/query"
	"jan-server/mono/apps/backend/internal/utils/httpclients/chat"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
	"jan-server/mono/apps/backend/internal/utils/ptr"

	decimal "github.com/shopspring/decimal"
)

type ModelCatalogService struct {
	modelCatalogRepo ModelCatalogRepository
}

func NewModelCatalogService(modelCatalogRepo ModelCatalogRepository) *ModelCatalogService {
	return &ModelCatalogService{
		modelCatalogRepo: modelCatalogRepo,
	}
}

func (s *ModelCatalogService) UpsertCatalog(ctx context.Context, provider *Provider, model chat.Model) (*ModelCatalog, bool, error) {
	kind := ProviderCustom
	if provider != nil {
		kind = provider.Kind
	}
	publicID := catalogPublicID(kind, model.ID, model.CanonicalSlug)
	if publicID == "" {
		return nil, false, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "model identifier missing", nil, "3934616c-8447-4ba8-809e-9b3c3924c32d")
	}
	existing, err := s.modelCatalogRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		// NotFound is expected for new catalogs - only treat other errors as fatal
		if !platformerrors.IsErrorType(err, platformerrors.ErrorTypeNotFound) {
			return nil, false, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to find existing model catalog")
		}
		// Not found - proceed to create new catalog
		existing = nil
	}

	catalog := buildModelCatalogFromModel(provider, model)
	catalog.PublicID = publicID
	now := time.Now().UTC()
	catalog.LastSyncedAt = &now

	if existing != nil {
		catalog.ID = existing.ID
		catalog.CreatedAt = existing.CreatedAt
		catalog.Active = existing.Active // Preserve Active status - don't override manually disabled catalogs

		// Only skip update if manually edited (status = "updated")
		if existing.Status == ModelCatalogStatusUpdated {
			return existing, false, nil
		}

		// Update if status is "init" or "filled"
		if err := s.modelCatalogRepo.Update(ctx, catalog); err != nil {
			return nil, false, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to update model catalog")
		}
		return catalog, false, nil
	}

	if err := s.modelCatalogRepo.Create(ctx, catalog); err != nil {
		return nil, false, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to create model catalog")
	}
	return catalog, true, nil
}

// BatchUpsertCatalogs performs batch upsertion of model catalogs with pre-fetched existing catalogs.
// This eliminates N+1 queries by fetching all existing catalogs at once.
// Returns a map of publicID -> (catalog, wasCreated, error)
func (s *ModelCatalogService) BatchUpsertCatalogs(ctx context.Context, provider *Provider, models []chat.Model) (map[string]*ModelCatalog, map[string]bool, error) {
	if len(models) == 0 {
		return make(map[string]*ModelCatalog), make(map[string]bool), nil
	}

	kind := ProviderCustom
	if provider != nil {
		kind = provider.Kind
	}

	// Step 1: Build public IDs for all models
	publicIDs := make([]string, 0, len(models))
	modelsByPublicID := make(map[string]chat.Model, len(models))
	for _, model := range models {
		publicID := catalogPublicID(kind, model.ID, model.CanonicalSlug)
		if publicID == "" {
			continue // Skip models with invalid IDs
		}
		publicIDs = append(publicIDs, publicID)
		modelsByPublicID[publicID] = model
	}

	// Step 2: Batch fetch existing catalogs (eliminates N+1 queries)
	existingCatalogs, err := s.FindByPublicIDs(ctx, publicIDs)
	if err != nil {
		return nil, nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to batch fetch existing catalogs")
	}

	// Step 3: Process each model
	now := time.Now().UTC()
	results := make(map[string]*ModelCatalog, len(models))
	createdFlags := make(map[string]bool, len(models))

	for publicID, model := range modelsByPublicID {
		existing := existingCatalogs[publicID]

		catalog := buildModelCatalogFromModel(provider, model)
		catalog.PublicID = publicID
		catalog.LastSyncedAt = &now

		if existing != nil {
			catalog.ID = existing.ID
			catalog.CreatedAt = existing.CreatedAt
			catalog.Active = existing.Active

			// Skip update if manually edited (status = "updated")
			if existing.Status == ModelCatalogStatusUpdated {
				results[publicID] = existing
				createdFlags[publicID] = false
				continue
			}

			// Update if status is "init" or "filled"
			if err := s.modelCatalogRepo.Update(ctx, catalog); err != nil {
				return nil, nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to update model catalog")
			}
			results[publicID] = catalog
			createdFlags[publicID] = false
		} else {
			// Create new catalog
			if err := s.modelCatalogRepo.Create(ctx, catalog); err != nil {
				return nil, nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to create model catalog")
			}
			results[publicID] = catalog
			createdFlags[publicID] = true
		}
	}

	return results, createdFlags, nil
}

func (s *ModelCatalogService) FindByID(ctx context.Context, id uint) (*ModelCatalog, error) {
	if id == 0 {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "model catalog ID is required", nil, "bfa98c70-387e-445c-a541-d1d07f722f67")
	}

	catalog, err := s.modelCatalogRepo.FindByID(ctx, id)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to find model catalog by ID")
	}

	return catalog, nil
}

func (s *ModelCatalogService) FindByPublicID(ctx context.Context, publicID string) (*ModelCatalog, error) {
	if publicID == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "model catalog public ID is required", nil, "c7539cbf-157d-49c3-8b04-adc572a496f7")
	}

	catalog, err := s.modelCatalogRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to find model catalog by public ID")
	}

	return catalog, nil
}

func (s *ModelCatalogService) FindByPublicIDs(ctx context.Context, publicIDs []string) (map[string]*ModelCatalog, error) {
	if len(publicIDs) == 0 {
		return make(map[string]*ModelCatalog), nil
	}

	catalogs, err := s.modelCatalogRepo.FindByPublicIDs(ctx, publicIDs)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to find model catalogs by public IDs")
	}

	result := make(map[string]*ModelCatalog, len(catalogs))
	for _, catalog := range catalogs {
		result[catalog.PublicID] = catalog
	}

	return result, nil
}

func (s *ModelCatalogService) FindByIDs(ctx context.Context, ids []uint) (map[uint]*ModelCatalog, error) {
	if len(ids) == 0 {
		return make(map[uint]*ModelCatalog), nil
	}

	catalogs, err := s.modelCatalogRepo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to find model catalogs by IDs")
	}

	result := make(map[uint]*ModelCatalog, len(catalogs))
	for _, catalog := range catalogs {
		result[catalog.ID] = catalog
	}

	return result, nil
}

func (s *ModelCatalogService) Update(ctx context.Context, catalog *ModelCatalog) (*ModelCatalog, error) {
	if catalog == nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "catalog cannot be nil", nil, "d2305a92-e294-4429-838f-963438264abe")
	}

	if err := s.modelCatalogRepo.Update(ctx, catalog); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to update model catalog")
	}

	return catalog, nil
}

func (s *ModelCatalogService) FindByFilter(ctx context.Context, filter ModelCatalogFilter, pagination *query.Pagination) ([]*ModelCatalog, error) {
	catalogs, err := s.modelCatalogRepo.FindByFilter(ctx, filter, pagination)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to find model catalogs")
	}
	return catalogs, nil
}

func (s *ModelCatalogService) Count(ctx context.Context, filter ModelCatalogFilter) (int64, error) {
	count, err := s.modelCatalogRepo.Count(ctx, filter)
	if err != nil {
		return 0, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to count model catalogs")
	}
	return count, nil
}

func (s *ModelCatalogService) BatchUpdateActive(ctx context.Context, filter ModelCatalogFilter, active bool) (int64, error) {
	rowsAffected, err := s.modelCatalogRepo.BatchUpdateActive(ctx, filter, active)
	if err != nil {
		return 0, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to batch update active status")
	}

	affectedCatalogs, err := s.modelCatalogRepo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return rowsAffected, nil
	}

	if len(affectedCatalogs) > 0 {
		modelKeys := make([]string, 0, len(affectedCatalogs))
		for _, catalog := range affectedCatalogs {
			modelKeys = append(modelKeys, catalog.PublicID)
		}
	}

	return rowsAffected, nil
}

func buildModelCatalogFromModel(provider *Provider, model chat.Model) *ModelCatalog {
	kind := ProviderCustom
	if provider != nil {
		kind = provider.Kind
	}

	status := ModelCatalogStatusInit
	if kind == ProviderOpenRouter {
		status = ModelCatalogStatusFilled
	}

	modelDisplayName := ""
	if displayName, ok := getString(model.Raw, "display_name"); ok && displayName != "" {
		modelDisplayName = displayName
	}

	var notes *string
	if rawNotes, ok := getString(model.Raw, "notes"); ok && rawNotes != "" {
		notes = ptr.ToString(rawNotes)
	}

	var description *string
	if desc, ok := getString(model.Raw, "description"); ok && desc != "" {
		description = ptr.ToString(desc)
	}

	defaultParameterNames := []string{
		"temperature",
		"max_tokens",
		"top_p",
		"top_k",
		"frequency_penalty",
		"presence_penalty",
		"repetition_penalty",
		"stop",
		"stream",
		"n",
		"response_format",
	}

	toolSupport := provider != nil && provider.SupportsTools()
	if toolSupport {
		defaultParameterNames = append(defaultParameterNames, "tools", "tool_choice")
	}

	supportedNames := extractStringSlice(model.Raw["supported_parameters"])
	if !toolSupport && len(supportedNames) > 0 {
		filtered := make([]string, 0, len(supportedNames))
		for _, name := range supportedNames {
			if name == "tools" || name == "tool_choice" {
				continue
			}
			filtered = append(filtered, name)
		}
		supportedNames = filtered
	}
	nameSet := make(map[string]struct{}, len(supportedNames)+len(defaultParameterNames))
	for _, name := range supportedNames {
		nameSet[name] = struct{}{}
	}
	for _, name := range defaultParameterNames {
		if _, exists := nameSet[name]; !exists {
			supportedNames = append(supportedNames, name)
			nameSet[name] = struct{}{}
		}
	}

	defaultParameters := extractDefaultParameters(model.Raw["default_parameters"])
	if _, exists := defaultParameters["top_p"]; !exists {
		if val, err := decimal.NewFromString("1"); err == nil {
			defaultParameters["top_p"] = &val
		}
	}
	if _, exists := defaultParameters["temperature"]; !exists {
		if val, err := decimal.NewFromString("0.7"); err == nil {
			defaultParameters["temperature"] = &val
		}
	}

	supportedParameters := SupportedParameters{
		Names:   supportedNames,
		Default: defaultParameters,
	}

	architecture := Architecture{}
	if archMap, ok := model.Raw["architecture"].(map[string]any); ok {
		architecture.Modality, _ = getString(archMap, "modality")
		architecture.InputModalities = extractStringSlice(archMap["input_modalities"])
		architecture.OutputModalities = extractStringSlice(archMap["output_modalities"])
		architecture.Tokenizer, _ = getString(archMap, "tokenizer")
		if instructType, ok := getString(archMap, "instruct_type"); ok && instructType != "" {
			architecture.InstructType = ptr.ToString(instructType)
		}
	}

	var isModerated *bool
	if topProvider, ok := model.Raw["top_provider"].(map[string]any); ok {
		if moderated, ok := topProvider["is_moderated"].(bool); ok {
			isModerated = ptr.ToBool(moderated)
		}
	}

	extras := copyMap(model.Raw)
	experimental := false
	if val, ok := model.Raw["experimental"].(bool); ok {
		experimental = val
	}

	var contextLength *int
	if rawCtx, ok := model.Raw["context_length"]; ok {
		if ctxLen, ok := floatFromAny(rawCtx); ok {
			val := int(ctxLen)
			contextLength = &val
		}
	}

	// Extract capabilities (moved from provider_model)
	inputModalities := extractStringSliceFromMap(model.Raw, "architecture", "input_modalities")
	outputModalities := extractStringSliceFromMap(model.Raw, "architecture", "output_modalities")

	supportsImages := containsString(inputModalities, "image")
	supportsReasoning := containsString(extractStringSlice(model.Raw["supported_parameters"]), "include_reasoning")
	supportsAudio := containsString(inputModalities, "audio") || containsString(outputModalities, "audio")
	supportsVideo := containsString(inputModalities, "video") || containsString(outputModalities, "video")
	supportsEmbeddings := detectEmbeddingSupport(model.ID, model.Raw)

	// Extract family
	family := extractFamily(model.ID)

	return &ModelCatalog{
		ModelDisplayName:    modelDisplayName,
		Description:         description,
		SupportedParameters: supportedParameters,
		Architecture:        architecture,
		Notes:               notes,
		ContextLength:       contextLength,
		IsModerated:         isModerated,
		Extras:              extras,
		Status:              status,
		Experimental:        experimental,
		SupportsImages:      supportsImages,
		SupportsEmbeddings:  supportsEmbeddings,
		SupportsReasoning:   supportsReasoning,
		SupportsAudio:       supportsAudio,
		SupportsVideo:       supportsVideo,
		Family:              family,
	}
}
