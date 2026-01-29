package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"jan-server/mono/apps/backend/internal/domain/query"
	"jan-server/mono/apps/backend/internal/infrastructure/logger"
	"jan-server/mono/apps/backend/internal/utils/httpclients/chat"
	"jan-server/mono/apps/backend/internal/utils/idgen"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
	"jan-server/mono/apps/backend/internal/utils/ptr"
)

type ProviderModelService struct {
	providerModelRepo ProviderModelRepository
	modelCatalogRepo  ModelCatalogRepository
}

func NewProviderModelService(
	providerModelRepo ProviderModelRepository,
	modelCatalogRepo ModelCatalogRepository,
) *ProviderModelService {
	return &ProviderModelService{
		providerModelRepo: providerModelRepo,
		modelCatalogRepo:  modelCatalogRepo,
	}
}

func (s *ProviderModelService) ListActiveProviderByIDs(ctx context.Context, providerIDs []uint) ([]*ProviderModel, error) {
	if len(providerIDs) == 0 {
		return nil, nil
	}
	ids := providerIDs
	return s.providerModelRepo.FindByFilter(ctx, ProviderModelFilter{
		ProviderIDs: &ids,
		Active:      ptr.ToBool(true),
	}, nil)
}

func (s *ProviderModelService) FindActiveByModelKey(ctx context.Context, modelPublicID string) ([]*ProviderModel, error) {
	if strings.TrimSpace(modelPublicID) == "" {
		return nil, nil
	}
	key := strings.TrimSpace(modelPublicID)
	active := ptr.ToBool(true)
	return s.providerModelRepo.FindByFilter(ctx, ProviderModelFilter{
		ModelPublicID: &key,
		Active:        active,
	}, nil)
}

func (s *ProviderModelService) FindActiveByProviderOriginalModelID(ctx context.Context, originalModelID string) ([]*ProviderModel, error) {
	if strings.TrimSpace(originalModelID) == "" {
		return nil, nil
	}
	key := strings.TrimSpace(originalModelID)
	active := ptr.ToBool(true)
	return s.providerModelRepo.FindByFilter(ctx, ProviderModelFilter{
		ProviderOriginalModelID: &key,
		Active:                  active,
	}, nil)
}

func (s *ProviderModelService) FindByModelKey(ctx context.Context, modelPublicID string) ([]*ProviderModel, error) {
	if strings.TrimSpace(modelPublicID) == "" {
		return nil, nil
	}
	key := strings.TrimSpace(modelPublicID)
	return s.providerModelRepo.FindByFilter(ctx, ProviderModelFilter{
		ModelPublicID: &key,
	}, nil)
}

func (s *ProviderModelService) FindByProviderOriginalModelID(ctx context.Context, originalModelID string) ([]*ProviderModel, error) {
	if strings.TrimSpace(originalModelID) == "" {
		return nil, nil
	}
	key := strings.TrimSpace(originalModelID)
	return s.providerModelRepo.FindByFilter(ctx, ProviderModelFilter{
		ProviderOriginalModelID: &key,
	}, nil)
}

func (s *ProviderModelService) FindActiveByProviderIDsAndKey(ctx context.Context, providerIDs []uint, modelPublicID string) ([]*ProviderModel, error) {
	if strings.TrimSpace(modelPublicID) == "" {
		return nil, nil
	}
	ids := providerIDs
	key := strings.TrimSpace(modelPublicID)
	active := ptr.ToBool(true)
	return s.providerModelRepo.FindByFilter(ctx, ProviderModelFilter{
		ProviderIDs:   &ids,
		ModelPublicID: &key,
		Active:        active,
	}, nil)
}

func (s *ProviderModelService) UpsertProviderModel(ctx context.Context, provider *Provider, catalog *ModelCatalog, model chat.Model) (*ProviderModel, error) {
	return s.UpsertProviderModelWithOptions(ctx, provider, catalog, model, false)
}

func (s *ProviderModelService) UpsertProviderModelWithOptions(ctx context.Context, provider *Provider, catalog *ModelCatalog, model chat.Model, autoEnableNewModels bool) (*ProviderModel, error) {
	originalModelID := strings.TrimSpace(model.ID)
	if originalModelID == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "model identifier missing", nil, "aa85b5ad-cc4a-4b24-ae35-f260163768ff")
	}

	// Generate ModelPublicID using NormalizeModelKey which returns vendor/model format
	kind := ProviderKind(provider.Kind)
	modelPublicID := NormalizeModelKey(kind, originalModelID)

	filter := ProviderModelFilter{
		ProviderID:    ptr.ToUint(provider.ID),
		ModelPublicID: &modelPublicID,
	}
	existing, err := s.providerModelRepo.FindByFilter(ctx, filter, &query.Pagination{Limit: ptr.ToInt(1)})
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to find existing provider model")
	}

	var catalogID *uint
	if catalog != nil {
		catalogID = &catalog.ID
	}

	if len(existing) > 0 {
		pm := existing[0]
		updateProviderModelFromRaw(pm, provider, catalog, model)
		if err := pm.Validate(); err != nil {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, err.Error(), nil, "validation-failed")
		}
		if err := s.providerModelRepo.Update(ctx, pm); err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to update provider model")
		}
		return pm, nil
	}

	publicID, err := idgen.GenerateSecureID("pmdl", 16)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to generate provider model ID")
	}

	pm := buildProviderModelFromRaw(provider, catalogID, model)
	pm.PublicID = publicID

	pm.Active = autoEnableNewModels

	// Validate before creating
	if err := pm.Validate(); err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, err.Error(), nil, "validation-failed")
	}

	if err := s.providerModelRepo.Create(ctx, pm); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to create provider model")
	}
	return pm, nil
}

func (s *ProviderModelService) FindByPublicID(ctx context.Context, publicID string) (*ProviderModel, error) {
	if publicID == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "provider model public ID is required", nil, "f7cdce27-bfed-48c2-a966-14549a666f6a")
	}

	return s.providerModelRepo.FindByPublicID(ctx, publicID)
}

// FindByID finds a provider model by its internal ID
func (s *ProviderModelService) FindByID(ctx context.Context, id uint) (*ProviderModel, error) {
	if id == 0 {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "provider model ID is required", nil, "8a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d")
	}

	return s.providerModelRepo.FindByID(ctx, id)
}

// FindByIDs finds provider models by their internal IDs
func (s *ProviderModelService) FindByIDs(ctx context.Context, ids []uint) ([]*ProviderModel, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	return s.providerModelRepo.FindByFilter(ctx, ProviderModelFilter{
		IDs: &ids,
	}, nil)
}

func (s *ProviderModelService) Update(ctx context.Context, providerModel *ProviderModel) (*ProviderModel, error) {
	if providerModel == nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "provider model cannot be nil", nil, "45c19f50-e0d1-4745-b6c4-be6de6ce0ec0")
	}

	// Validate before updating
	if err := providerModel.Validate(); err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, err.Error(), nil, "validation-failed")
	}

	if providerModel.Active && providerModel.ModelCatalogID != nil {
		catalog, err := s.modelCatalogRepo.FindByID(ctx, *providerModel.ModelCatalogID)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to check catalog status")
		}

		if catalog != nil && catalog.Active != nil && !*catalog.Active {
			providerModel.Active = false
		}
	}

	if err := s.providerModelRepo.Update(ctx, providerModel); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to update provider model")
	}

	return providerModel, nil
}

func (s *ProviderModelService) FindByFilter(ctx context.Context, filter ProviderModelFilter) ([]*ProviderModel, error) {
	models, err := s.providerModelRepo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to find provider models")
	}
	return models, nil
}

func (s *ProviderModelService) FindByFilterWithPagination(ctx context.Context, filter ProviderModelFilter, pagination *query.Pagination) ([]*ProviderModel, error) {
	models, err := s.providerModelRepo.FindByFilter(ctx, filter, pagination)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to find provider models")
	}
	return models, nil
}

func (s *ProviderModelService) Count(ctx context.Context, filter ProviderModelFilter) (int64, error) {
	count, err := s.providerModelRepo.Count(ctx, filter)
	if err != nil {
		return 0, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to count provider models")
	}
	return count, nil
}

func (s *ProviderModelService) BatchUpdateActive(ctx context.Context, filter ProviderModelFilter, active bool) (int64, error) {
	rowsAffected, err := s.providerModelRepo.BatchUpdateActive(ctx, filter, active)
	if err != nil {
		return 0, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to batch update active status")
	}

	affectedModels, err := s.providerModelRepo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return rowsAffected, nil
	}

	modelKeys := make([]string, 0, len(affectedModels))
	providerIDs := make(map[uint]bool)

	for _, model := range affectedModels {
		modelKeys = append(modelKeys, model.ModelPublicID)
		providerIDs[model.ProviderID] = true
	}

	return rowsAffected, nil
}

// FindCatalogByID returns a model catalog by ID (used for applying defaults).
func (s *ProviderModelService) FindCatalogByID(ctx context.Context, id uint) (*ModelCatalog, error) {
	if id == 0 {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "model catalog ID is required", nil, "c9bde6a4-6bd1-4f8e-97df-e7b03c3f6f73")
	}

	catalog, err := s.modelCatalogRepo.FindByID(ctx, id)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to find model catalog by ID")
	}
	return catalog, nil
}

func (s *ProviderModelService) BatchUpdateModelDisplayName(ctx context.Context, filter ProviderModelFilter, modelDisplayName string) (int64, error) {
	rowsAffected, err := s.providerModelRepo.BatchUpdateModelDisplayName(ctx, filter, modelDisplayName)
	if err != nil {
		return 0, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to batch update model display name")
	}
	return rowsAffected, nil
}

func (s *ProviderModelService) DeleteByProviderID(ctx context.Context, providerID uint) error {
	if providerID == 0 {
		return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "provider_id is required", nil, "1d0c2ebe-1585-45e6-bfb4-0e6ec98e7e2b")
	}

	filter := ProviderModelFilter{
		ProviderID: &providerID,
	}
	models, err := s.providerModelRepo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to find provider models for deletion")
	}

	for _, model := range models {
		if err := s.providerModelRepo.DeleteByID(ctx, model.ID); err != nil {
			return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, fmt.Sprintf("failed to delete provider model %d", model.ID))
		}
	}

	return nil
}

func buildProviderModelFromRaw(provider *Provider, catalogID *uint, model chat.Model) *ProviderModel {
	log := logger.GetLogger()

	pricing := extractPricing(model.Raw["pricing"])
	tokenLimits := extractTokenLimits(model.Raw)
	reasoningMeta := extractReasoningMetadata(model)
	providerFlags := extractProviderFlags(model)

	supportsAuto := false
	if reasoningMeta.SupportsAutoMode != nil {
		supportsAuto = *reasoningMeta.SupportsAutoMode
	}
	supportsThinking := false
	if reasoningMeta.SupportsThinkingMode != nil {
		supportsThinking = *reasoningMeta.SupportsThinkingMode
	}

	modelDisplayName := getModelDisplayName(model)

	// Extract category and ordering
	category := extractCategoryFromModel(model)
	categoryOrder := extractCategoryOrder(model)
	modelOrder := extractModelOrder(model)

	// Log missing critical fields
	if model.DisplayName == "" && model.ID != "" {
		log.Warn().
			Str("model_id", model.ID).
			Str("provider", string(provider.Kind)).
			Msg("Model missing display_name")
	}
	if len(pricing.Lines) == 0 {
		log.Debug().
			Str("model_id", model.ID).
			Msg("Model missing pricing data")
	}

	// Generate ModelPublicID using NormalizeModelKey which returns canonical vendor/model format
	kind := ProviderKind(provider.Kind)
	modelPublicID := NormalizeModelKey(kind, model.ID)
	// if modelPublicID contains string "jan", set category to "jan"
	if strings.Contains(modelPublicID, "jan") {
		category = "jan"
	}

	pm := &ProviderModel{
		ProviderID:              provider.ID,
		Kind:                    kind,
		ModelCatalogID:          catalogID,
		ModelPublicID:           modelPublicID,
		ProviderOriginalModelID: model.ID,
		ModelDisplayName:        modelDisplayName,
		Category:                category,
		CategoryOrderNumber:     categoryOrder,
		ModelOrderNumber:        modelOrder,
		Pricing:                 pricing,
		TokenLimits:             tokenLimits,
		SupportsAutoMode:        supportsAuto,
		SupportsThinkingMode:    supportsThinking,
		Active:                  false,
	}
	applyReasoningMetadata(pm, reasoningMeta, false)
	pm.ProviderFlags = providerFlags
	return pm
}

func updateProviderModelFromRaw(pm *ProviderModel, provider *Provider, catalog *ModelCatalog, model chat.Model) {
	pm.Kind = ProviderKind(provider.Kind) // Update Kind field to match provider

	// Update catalog ID
	if catalog != nil {
		pm.ModelCatalogID = &catalog.ID
	} else {
		pm.ModelCatalogID = nil
	}

	// Only update ModelDisplayName if catalog hasn't been manually updated
	// If catalog.Status == "updated", preserve existing ModelDisplayName to respect admin changes
	if catalog == nil || catalog.Status != ModelCatalogStatusUpdated {
		pm.ModelDisplayName = getModelDisplayName(model)
	}
	// else: preserve existing pm.ModelDisplayName

	// Only update pricing if new data is available (preserve existing pricing)
	newPricing := extractPricing(model.Raw["pricing"])
	if len(newPricing.Lines) > 0 {
		pm.Pricing = newPricing
	}

	pm.TokenLimits = extractTokenLimits(model.Raw)
	reasoningMeta := extractReasoningMetadata(model)
	applyReasoningMetadata(pm, reasoningMeta, true)

	// Update category and ordering
	pm.Category = extractCategoryFromModel(model)
	pm.CategoryOrderNumber = extractCategoryOrder(model)
	pm.ModelOrderNumber = extractModelOrder(model)

	flags := extractProviderFlags(model)
	if len(flags) > 0 {
		pm.ProviderFlags = flags
	}

	// Don't update Active field - keep existing value for already-synced models
	pm.UpdatedAt = time.Now().UTC()
}

func extractPricing(value any) Pricing {
	pricing := Pricing{}
	pricingMap, ok := value.(map[string]any)
	if !ok {
		return pricing
	}

	if lines, ok := pricingMap["lines"].([]any); ok {
		for _, line := range lines {
			lineMap, ok := line.(map[string]any)
			if !ok {
				continue
			}
			unitStr, _ := getString(lineMap, "unit")
			amount, ok := floatFromAny(lineMap["amount"])
			if !ok {
				continue
			}
			pricing.Lines = append(pricing.Lines, PriceLine{
				Unit:     PriceUnit(strings.ToLower(strings.TrimSpace(unitStr))),
				Amount:   MicroUSD(int64(amount * 1_000_000)),
				Currency: "USD",
			})
		}
	}

	return pricing
}

func extractTokenLimits(raw map[string]any) *TokenLimits {
	if raw == nil {
		return nil
	}

	limitsMap := raw
	if nested, ok := raw["token_limits"].(map[string]any); ok {
		limitsMap = nested
	}

	hasContextLength := false
	hasMaxCompletion := false
	limits := TokenLimits{}

	if contextLen, ok := floatFromAny(limitsMap["context_length"]); ok {
		limits.ContextLength = int(contextLen)
		hasContextLength = true
	} else if contextLen, ok := floatFromAny(raw["context_length"]); ok {
		limits.ContextLength = int(contextLen)
		hasContextLength = true
	}
	if maxCompletion, ok := floatFromAny(limitsMap["max_completion_tokens"]); ok {
		limits.MaxCompletionTokens = int(maxCompletion)
		hasMaxCompletion = true
	} else if maxCompletion, ok := floatFromAny(raw["max_completion_tokens"]); ok {
		limits.MaxCompletionTokens = int(maxCompletion)
		hasMaxCompletion = true
	}

	// Return nil only if no token limit data exists at all
	if !hasContextLength && !hasMaxCompletion {
		return nil
	}
	return &limits
}

func (s *ProviderModelService) FindModelCountsByProviderIDs(ctx context.Context, providerIDs []uint) (map[uint]int64, error) {
	counts := make(map[uint]int64)

	for _, providerID := range providerIDs {
		filter := ProviderModelFilter{
			ProviderID: &providerID,
		}
		count, err := s.providerModelRepo.Count(ctx, filter)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, fmt.Sprintf("failed to count models for provider %d", providerID))
		}
		counts[providerID] = count
	}

	return counts, nil
}

func (s *ProviderModelService) FindActiveModelCountsByProviderIDs(ctx context.Context, providerIDs []uint) (map[uint]int64, error) {
	counts := make(map[uint]int64)

	for _, providerID := range providerIDs {
		active := true
		filter := ProviderModelFilter{
			ProviderID: &providerID,
			Active:     &active,
		}
		count, err := s.providerModelRepo.Count(ctx, filter)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, fmt.Sprintf("failed to count active models for provider %d", providerID))
		}
		counts[providerID] = count
	}

	return counts, nil
}

// getModelDisplayName returns the best available display name for a model
func getModelDisplayName(model chat.Model) string {
	if model.DisplayName != "" {
		return model.DisplayName
	}
	if model.Name != "" {
		return model.Name
	}
	if model.ID != "" {
		return model.ID
	}
	// Fallback to canonical slug or generate unique name
	if model.CanonicalSlug != "" {
		return model.CanonicalSlug
	}
	return "Unnamed Model"
}

// extractCategoryFromModel extracts category from provider API with legacy fallback
func extractCategoryFromModel(model chat.Model) string {
	// Priority 1: Use category from provider API if available
	if cat, ok := getString(model.Raw, "category"); ok && cat != "" {
		return cat
	}

	// Priority 2: Check for "top_provider" field (OpenRouter-specific)
	if topProvider, ok := model.Raw["top_provider"].(map[string]any); ok {
		if cat, ok := getString(topProvider, "category"); ok && cat != "" {
			return cat
		}
	}

	// Priority 3: Legacy fallback - infer from model properties
	return inferLegacyCategory(model)
}

// inferLegacyCategory infers category from model capabilities when provider doesn't supply it
func inferLegacyCategory(model chat.Model) string {

	// Check for reasoning models
	if containsString(extractStringSlice(model.Raw["supported_parameters"]), "include_reasoning") {
		return "Reasoning"
	}

	// Check for embedding models
	if detectEmbeddingSupport(model.ID, model.Raw) {
		return "Embedding"
	}

	inputModalities := extractStringSliceFromMap(model.Raw, "architecture", "input_modalities")
	outputModalities := extractStringSliceFromMap(model.Raw, "architecture", "output_modalities")

	// Check for image generation models
	if containsString(outputModalities, "image") {
		return "Image Generation"
	}

	// Check for vision models (input images but no output)
	if containsString(inputModalities, "image") && !containsString(outputModalities, "image") {
		return "Vision"
	}

	// Check for audio models
	if containsString(inputModalities, "audio") || containsString(outputModalities, "audio") {
		return "Audio"
	}

	// Check for video models
	if containsString(inputModalities, "video") || containsString(outputModalities, "video") {
		return "Video"
	}

	// Default to legacy for text-based models
	return "legacy"
}

// extractCategoryOrder extracts category ordering from provider API
func extractCategoryOrder(model chat.Model) int {
	// Try to get from provider API
	if order, ok := floatFromAny(model.Raw["category_order"]); ok {
		return int(order)
	}
	if order, ok := floatFromAny(model.Raw["category_order_number"]); ok {
		return int(order)
	}
	return 0
}

// extractModelOrder extracts model ordering from provider API
func extractModelOrder(model chat.Model) int {
	// Try to get from provider API
	if order, ok := floatFromAny(model.Raw["model_order"]); ok {
		return int(order)
	}
	if order, ok := floatFromAny(model.Raw["model_order_number"]); ok {
		return int(order)
	}
	return 0
}

type ReasoningMetadata struct {
	SupportsThinkingMode    *bool
	SupportsAutoMode        *bool
	DefaultConversationMode string
	ReasoningConfig         *ReasoningConfig
}

func extractReasoningMetadata(model chat.Model) ReasoningMetadata {
	meta := ReasoningMetadata{}
	rawReasoning, _ := model.Raw["reasoning"].(map[string]any)
	rawThinking, _ := model.Raw["thinking"].(map[string]any)
	supportedParams := extractStringSlice(model.Raw["supported_parameters"])

	setBool := func(target **bool, value any) {
		if b, ok := boolFromAny(value); ok {
			*target = ptr.ToBool(b)
		}
	}

	// Initialize reasoning config
	config := &ReasoningConfig{}
	hasReasoningConfig := false

	if rawReasoning != nil {
		setBool(&meta.SupportsAutoMode, rawReasoning["auto"])
		setBool(&meta.SupportsAutoMode, rawReasoning["supports_auto"])
		setBool(&meta.SupportsThinkingMode, rawReasoning["thinking"])
		setBool(&meta.SupportsThinkingMode, rawReasoning["supports_thinking"])

		if defaultMode, ok := getString(rawReasoning, "default_mode"); ok {
			meta.DefaultConversationMode = defaultMode
		} else if defaultMode, ok := getString(rawReasoning, "default_conversation_mode"); ok {
			meta.DefaultConversationMode = defaultMode
		}

		// Extract default effort
		if defEffort, ok := getString(rawReasoning, "default_effort"); ok {
			config.DefaultEffort = defEffort
			hasReasoningConfig = true
		}
		if defEffort, ok := getString(rawReasoning, "reasoning_default_effort"); ok && config.DefaultEffort == "" {
			config.DefaultEffort = defEffort
			hasReasoningConfig = true
		}

		// Extract effort levels
		if effortLevels := extractStringSlice(rawReasoning["effort_levels"]); len(effortLevels) > 0 {
			config.EffortLevels = effortLevels
			hasReasoningConfig = true
		}
		if len(config.EffortLevels) == 0 {
			if effortLevels := extractStringSlice(rawReasoning["reasoning_effort_levels"]); len(effortLevels) > 0 {
				config.EffortLevels = effortLevels
				hasReasoningConfig = true
			}
		}

		// Extract max tokens
		if maxTokens, ok := floatFromAny(rawReasoning["max_tokens"]); ok {
			mt := int(maxTokens)
			config.MaxTokens = &mt
			hasReasoningConfig = true
		}
		if maxTokens, ok := floatFromAny(rawReasoning["reasoning_max_tokens"]); ok && config.MaxTokens == nil {
			mt := int(maxTokens)
			config.MaxTokens = &mt
			hasReasoningConfig = true
		}

		// Extract latency hint
		if latency, ok := floatFromAny(rawReasoning["latency_hint_ms"]); ok {
			l := int(latency)
			config.LatencyHintMs = &l
			hasReasoningConfig = true
		}
		if latency, ok := floatFromAny(rawReasoning["latency_ms"]); ok && config.LatencyHintMs == nil {
			l := int(latency)
			config.LatencyHintMs = &l
			hasReasoningConfig = true
		}

		// Extract price multiplier
		if multiplier, ok := floatFromAny(rawReasoning["price_multiplier"]); ok {
			config.PriceMultiplier = &multiplier
			hasReasoningConfig = true
		} else if multiplier, ok := floatFromAny(rawReasoning["reasoning_price_multiplier"]); ok {
			config.PriceMultiplier = &multiplier
			hasReasoningConfig = true
		}

		// Extract mode display
		if modeDisplay := extractReasoningModeDisplay(rawReasoning["modes"]); len(modeDisplay) > 0 {
			config.ModeDisplay = modeDisplay
			hasReasoningConfig = true
		}
		if len(config.ModeDisplay) == 0 {
			if modeDisplay := extractReasoningModeDisplay(rawReasoning["mode_display"]); len(modeDisplay) > 0 {
				config.ModeDisplay = modeDisplay
				hasReasoningConfig = true
			}
		}
	}

	if rawThinking != nil {
		setBool(&meta.SupportsThinkingMode, rawThinking["enabled"])

		if maxTokens, ok := floatFromAny(rawThinking["max_tokens"]); ok && config.MaxTokens == nil {
			mt := int(maxTokens)
			config.MaxTokens = &mt
			hasReasoningConfig = true
		}
		if latency, ok := floatFromAny(rawThinking["latency_ms"]); ok && config.LatencyHintMs == nil {
			l := int(latency)
			config.LatencyHintMs = &l
			hasReasoningConfig = true
		}
	}

	if meta.SupportsThinkingMode == nil && containsString(supportedParams, "reasoning_effort") {
		meta.SupportsThinkingMode = ptr.ToBool(true)
	}
	if meta.SupportsAutoMode == nil && containsString(supportedParams, "auto") {
		meta.SupportsAutoMode = ptr.ToBool(true)
	}

	// Only set reasoning config if we extracted any data
	if hasReasoningConfig {
		meta.ReasoningConfig = config
	}

	return meta
}

func applyReasoningMetadata(pm *ProviderModel, meta ReasoningMetadata, preserveExisting bool) {
	if preserveExisting {
		if meta.SupportsAutoMode != nil {
			pm.SupportsAutoMode = *meta.SupportsAutoMode
		}
		if meta.SupportsThinkingMode != nil {
			pm.SupportsThinkingMode = *meta.SupportsThinkingMode
		}
	} else {
		pm.SupportsAutoMode = boolOrDefault(meta.SupportsAutoMode, pm.SupportsAutoMode)
		pm.SupportsThinkingMode = boolOrDefault(meta.SupportsThinkingMode, pm.SupportsThinkingMode)
	}

	if meta.DefaultConversationMode != "" {
		pm.DefaultConversationMode = normalizeConversationMode(meta.DefaultConversationMode, pm.DefaultConversationMode)
	} else if !preserveExisting && pm.DefaultConversationMode == "" {
		pm.DefaultConversationMode = "standard"
	}

	// Apply reasoning config
	if meta.ReasoningConfig != nil {
		pm.ReasoningConfig = meta.ReasoningConfig
	} else if !preserveExisting {
		pm.ReasoningConfig = nil
	}

	if pm.DefaultConversationMode == "" {
		pm.DefaultConversationMode = normalizeConversationMode("", pm.DefaultConversationMode)
	}
}

func normalizeConversationMode(mode string, fallback string) string {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "auto", "automatic":
		return "auto"
	case "thinking", "reasoning", "long_thought":
		return "thinking"
	case "fast", "standard", "default", "":
		if fallback != "" {
			return fallback
		}
		return "standard"
	default:
		return m
	}
}

func extractReasoningModeDisplay(value any) []ReasoningModeOption {
	result := []ReasoningModeOption{}
	items, ok := value.([]any)
	if !ok {
		return result
	}
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := getString(entry, "name")
		displayName, _ := getString(entry, "display_name")
		description, _ := getString(entry, "description")
		var latency *int
		if v, ok := floatFromAny(entry["latency_hint_ms"]); ok {
			val := int(v)
			latency = &val
		}
		var priceMultiplier *float64
		if v, ok := floatFromAny(entry["price_hint_multiplier"]); ok {
			val := v
			priceMultiplier = &val
		}
		if name == "" && displayName == "" {
			continue
		}
		result = append(result, ReasoningModeOption{
			Name:                name,
			DisplayName:         displayName,
			Description:         description,
			LatencyHintMs:       latency,
			PriceHintMultiplier: priceMultiplier,
		})
	}
	return result
}

func extractProviderFlags(model chat.Model) map[string]any {
	if flags, ok := model.Raw["provider_flags"].(map[string]any); ok {
		return flags
	}
	return nil
}

func boolOrDefault(value *bool, fallback bool) bool {
	if value != nil {
		return *value
	}
	return fallback
}

func boolFromAny(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		l := strings.ToLower(strings.TrimSpace(v))
		if l == "true" || l == "1" || l == "yes" {
			return true, true
		}
		if l == "false" || l == "0" || l == "no" {
			return false, true
		}
	case float64:
		return v != 0, true
	case int:
		return v != 0, true
	case int64:
		return v != 0, true
	}
	return false, false
}
