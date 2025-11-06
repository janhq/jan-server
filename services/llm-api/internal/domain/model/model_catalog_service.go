package model

import (
	"context"
	"time"

	decimal "github.com/shopspring/decimal"
	"jan-server/services/llm-api/internal/domain/query"
	"jan-server/services/llm-api/internal/utils/httpclients/chat"
	"jan-server/services/llm-api/internal/utils/platformerrors"
	"jan-server/services/llm-api/internal/utils/ptr"
)

type ModelCatalogService struct {
	modelCatalogRepo ModelCatalogRepository
}

func NewModelCatalogService(modelCatalogRepo ModelCatalogRepository) *ModelCatalogService {
	return &ModelCatalogService{
		modelCatalogRepo: modelCatalogRepo,
	}
}

func (s *ModelCatalogService) UpsertCatalog(ctx context.Context, kind ProviderKind, model chat.Model) (*ModelCatalog, bool, error) {
	publicID := catalogPublicID(kind, model)
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

	catalog := buildModelCatalogFromModel(kind, model)
	catalog.PublicID = publicID
	now := time.Now().UTC()
	catalog.LastSyncedAt = &now

	if existing != nil {
		catalog.ID = existing.ID
		catalog.CreatedAt = existing.CreatedAt
		catalog.Active = existing.Active // Preserve Active status - don't override manually disabled catalogs
		if existing.Status == ModelCatalogStatusFilled {
			return existing, false, nil
		}
		if catalog.Status == ModelCatalogStatusFilled {
			if err := s.modelCatalogRepo.Update(ctx, catalog); err != nil {
				return nil, false, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to update model catalog")
			}
			return catalog, false, nil
		}
		return existing, false, nil
	}

	if err := s.modelCatalogRepo.Create(ctx, catalog); err != nil {
		return nil, false, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to create model catalog")
	}
	return catalog, true, nil
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

func catalogPublicID(kind ProviderKind, model chat.Model) string {
	// Use CanonicalSlug if available, otherwise use model ID
	rawModelKey := model.CanonicalSlug
	if rawModelKey == "" {
		rawModelKey = model.ID
	}
	// Return the canonical vendor/model format
	return NormalizeModelKey(kind, rawModelKey)
}

func buildModelCatalogFromModel(kind ProviderKind, model chat.Model) *ModelCatalog {
	status := ModelCatalogStatusInit
	if kind == ProviderOpenRouter {
		status = ModelCatalogStatusFilled
	}

	var notes *string
	if desc, ok := getString(model.Raw, "description"); ok && desc != "" {
		notes = ptr.ToString(desc)
	}

	defaultParameterNames := []string{
		"temperature",
		"max_tokens",
		"top_p",
		"frequency_penalty",
		"presence_penalty",
		"stop",
		"stream",
		"n",
		"tools",
		"tool_choice",
		"response_format",
	}

	supportedNames := extractStringSlice(model.Raw["supported_parameters"])
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

	return &ModelCatalog{
		SupportedParameters: supportedParameters,
		Architecture:        architecture,
		Notes:               notes,
		IsModerated:         isModerated,
		Extras:              extras,
		Status:              status,
	}
}
