package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/domain/query"
	"jan-server/services/llm-api/internal/infrastructure/logger"
	"jan-server/services/llm-api/internal/utils/crypto"
	"jan-server/services/llm-api/internal/utils/httpclients/chat"
	"jan-server/services/llm-api/internal/utils/idgen"
	"jan-server/services/llm-api/internal/utils/platformerrors"
	"jan-server/services/llm-api/internal/utils/ptr"
)

type ProviderService struct {
	providerRepo         ProviderRepository
	providerModelService *ProviderModelService
	modelCatalogService  *ModelCatalogService
	modelProviderSecret  string // Encryption secret for provider API keys
}

func NewProviderService(
	providerRepo ProviderRepository,
	providerModelService *ProviderModelService,
	modelCatalogService *ModelCatalogService,
) *ProviderService {
	return &ProviderService{
		providerRepo:         providerRepo,
		providerModelService: providerModelService,
		modelCatalogService:  modelCatalogService,
	}
}

type RegisterProviderInput struct {
	Name                 string
	Vendor               string
	Category             ProviderCategory // "llm" or "image"; defaults to "llm"
	BaseURL              string
	Endpoints            EndpointList
	APIKey               string
	Metadata             map[string]string
	Active               bool
	DefaultImageGenerate bool
	DefaultImageEdit     bool
}

type UpdateProviderInput struct {
	Name                 *string
	BaseURL              *string
	Endpoints            *EndpointList
	APIKey               *string
	Metadata             *map[string]string
	Active               *bool
	Category             *ProviderCategory // Optional category update
	DefaultImageGenerate *bool
	DefaultImageEdit     *bool
}

type UpsertProviderInput struct {
	Name                 string
	Vendor               string
	Category             ProviderCategory // "llm" or "image"; defaults to "llm"
	BaseURL              string
	Endpoints            EndpointList
	APIKey               string
	Metadata             map[string]string
	Active               bool
	DefaultImageGenerate bool
	DefaultImageEdit     bool
}

func (s *ProviderService) RegisterProvider(ctx context.Context, input RegisterProviderInput) (*Provider, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "provider name is required", nil, "c86f2bc3-5ea3-41d3-b450-e86adb33352c")
	}

	endpoints, err := normalizeAndValidateEndpoints(input.Endpoints)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "invalid endpoints")
	}
	baseURL := strings.TrimSpace(input.BaseURL)
	if len(endpoints) == 0 {
		if baseURL == "" {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "base_url is required", nil, "c80a4867-6c8b-4adb-878d-41fe1b5e96ae")
		}
		if _, err := url.ParseRequestURI(baseURL); err != nil {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, fmt.Sprintf("invalid base_url format: %v", err), nil, "9e944ba1-c849-4959-957f-cb3de40e2eb1")
		}
		baseURL = normalizeURL(baseURL)
		endpoints = EndpointList{{URL: baseURL, Weight: 1, Healthy: true}}
	} else {
		baseURL = endpoints[0].URL
	}

	kind := ProviderKindFromVendor(input.Vendor)

	// Allow multiple providers with the same kind.

	publicID, err := idgen.GenerateSecureID("prov", 16)
	if err != nil {
		return nil, err
	}

	plainAPIKey := strings.TrimSpace(input.APIKey)
	apiKeyHint := apiKeyHint(plainAPIKey)
	var encryptedAPIKey string
	if plainAPIKey != "" {
		secret := strings.TrimSpace(config.GetGlobal().ModelProviderSecret)
		if secret == "" {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeInternal, "model provider secret is not configured", nil, "9fd675bb-1471-4dd4-9160-16df36500595")
		}
		cipher, err := crypto.EncryptString(secret, plainAPIKey)
		if err != nil {
			return nil, err
		}
		encryptedAPIKey = cipher
	}

	metadata := sanitizeMetadata(input.Metadata)
	metadata = setDefaultCapabilities(kind, metadata)

	// Default category to LLM if not specified.
	// Auto-detect category for known image/OCR providers.
	category := input.Category
	if category == "" {
		switch kind {
		case ProviderZImage:
			category = ProviderCategoryImage
		case ProviderOCR:
			category = ProviderCategoryOCR
		default:
			category = ProviderCategoryLLM
		}
	}

	provider := &Provider{
		PublicID:             publicID,
		DisplayName:          name,
		Kind:                 kind,
		Category:             category,
		EncryptedAPIKey:      encryptedAPIKey,
		APIKeyHint:           apiKeyHint,
		IsModerated:          false,
		Active:               input.Active,
		DefaultImageGenerate: input.DefaultImageGenerate,
		DefaultImageEdit:     input.DefaultImageEdit,
		Metadata:             metadata,
	}
	// SetEndpoints updates both Endpoints and BaseURL (for backward compat)
	provider.SetEndpoints(endpoints)

	if err := s.providerRepo.Create(ctx, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

func (s *ProviderService) FindProviderByVendor(ctx context.Context, vendor string) (*Provider, error) {
	kind := ProviderKindFromVendor(vendor)
	filter := ProviderFilter{Kind: &kind}
	result, err := s.providerRepo.FindByFilter(ctx, filter, &query.Pagination{Limit: ptr.ToInt(1)})
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result[0], nil
}

func ProviderKindFromVendor(vendor string) ProviderKind {
	switch strings.ToLower(strings.TrimSpace(vendor)) {
	case "jan":
		return ProviderJan
	case "openrouter":
		return ProviderOpenRouter
	case "openai":
		return ProviderOpenAI
	case "anthropic":
		return ProviderAnthropic
	case "gemini", "google", "googleai":
		return ProviderGoogle
	case "mistral":
		return ProviderMistral
	case "groq":
		return ProviderGroq
	case "cohere":
		return ProviderCohere
	case "ollama":
		return ProviderOllama
	case "replicate":
		return ProviderReplicate
	case "azure_openai", "azure-openai":
		return ProviderAzureOpenAI
	case "aws_bedrock", "bedrock":
		return ProviderAWSBedrock
	case "perplexity":
		return ProviderPerplexity
	case "togetherai", "together":
		return ProviderTogetherAI
	case "huggingface":
		return ProviderHuggingFace
	case "vercel_ai", "vercel-ai", "vercel":
		return ProviderVercelAI
	case "deepinfra":
		return ProviderDeepInfra
	case "z-image", "zimage":
		return ProviderZImage
	case "ocr", "docling":
		return ProviderOCR
	default:
		return ProviderCustom
	}
}

func apiKeyHint(apiKey string) *string {
	key := strings.TrimSpace(apiKey)
	if len(key) < 4 {
		return nil
	}
	hint := key[len(key)-4:]
	return ptr.ToString(hint)
}

func (s *ProviderService) GetByID(ctx context.Context, providerId uint) (*Provider, error) {
	provider, err := s.providerRepo.FindByID(ctx, providerId)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func (s *ProviderService) FindByPublicID(ctx context.Context, publicID string) (*Provider, error) {
	return s.providerRepo.FindByPublicID(ctx, publicID)
}

func (s *ProviderService) GetByIDs(ctx context.Context, ids []uint) (map[uint]*Provider, error) {
	if len(ids) == 0 {
		return make(map[uint]*Provider), nil
	}

	providers, err := s.providerRepo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	result := make(map[uint]*Provider, len(providers))
	for _, provider := range providers {
		result[provider.ID] = provider
	}

	return result, nil
}

func (s *ProviderService) FindAllProviders(ctx context.Context) ([]*Provider, error) {
	filter := ProviderFilter{}
	return s.providerRepo.FindByFilter(ctx, filter, nil)
}

func (s *ProviderService) FindProviders(ctx context.Context, filter ProviderFilter) ([]*Provider, error) {
	return s.providerRepo.FindByFilter(ctx, filter, nil)
}

func (s *ProviderService) FindAllActiveProviders(ctx context.Context) ([]*Provider, error) {
	filter := ProviderFilter{Active: ptr.ToBool(true)}
	return s.providerRepo.FindByFilter(ctx, filter, nil)
}

func (s *ProviderService) DeleteProviderByPublicID(ctx context.Context, publicID string) error {
	if strings.TrimSpace(publicID) == "" {
		return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "provider public ID is required", nil, "35b02081-5c0d-4d65-9841-c1d7a5300829")
	}

	provider, err := s.providerRepo.FindByPublicID(ctx, publicID)
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to find provider")
	}
	if provider == nil {
		return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeNotFound, "provider not found", nil, "9f6ec5ff-5ce1-4df6-9871-7dd89da8c548")
	}

	if err := s.providerModelService.DeleteByProviderID(ctx, provider.ID); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to delete provider models")
	}

	if err := s.providerRepo.DeleteByID(ctx, provider.ID); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to delete provider")
	}

	return nil
}

func (s *ProviderService) UpsertProvider(ctx context.Context, input UpsertProviderInput) (*Provider, error) {
	// Check if provider exists by display name (since Name field doesn't exist in filter)
	filter := ProviderFilter{}
	allProviders, err := s.providerRepo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return nil, err
	}

	// Find existing provider by name
	var existing *Provider
	for _, p := range allProviders {
		if p.DisplayName == input.Name {
			existing = p
			break
		}
	}

	if existing != nil {
		// Update existing provider
		updateInput := UpdateProviderInput{
			BaseURL:              &input.BaseURL,
			Endpoints:            &input.Endpoints,
			APIKey:               &input.APIKey,
			Metadata:             &input.Metadata,
			Active:               &input.Active,
			DefaultImageGenerate: &input.DefaultImageGenerate,
			DefaultImageEdit:     &input.DefaultImageEdit,
		}
		// Only update category if explicitly provided
		if input.Category != "" {
			category := input.Category
			updateInput.Category = &category
		}
		return s.UpdateProvider(ctx, existing, updateInput)
	}

	// Register new provider
	registerInput := RegisterProviderInput{
		Name:                 input.Name,
		Vendor:               input.Vendor,
		Category:             input.Category,
		BaseURL:              input.BaseURL,
		Endpoints:            input.Endpoints,
		APIKey:               input.APIKey,
		Metadata:             input.Metadata,
		Active:               input.Active,
		DefaultImageGenerate: input.DefaultImageGenerate,
		DefaultImageEdit:     input.DefaultImageEdit,
	}
	return s.RegisterProvider(ctx, registerInput)
}

func (s *ProviderService) UpdateProvider(ctx context.Context, provider *Provider, input UpdateProviderInput) (*Provider, error) {
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "provider name is required", nil, "a5df830c-8084-4238-9e17-f44950764ca5")
		}
		provider.DisplayName = name
	}
	if input.Endpoints != nil {
		endpoints, err := normalizeAndValidateEndpoints(*input.Endpoints)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "invalid endpoints")
		}
		// Always delegate to SetEndpoints to keep BaseURL in sync
		provider.SetEndpoints(endpoints)
	}
	if input.BaseURL != nil && (input.Endpoints == nil || len(*input.Endpoints) == 0) {
		baseURL := strings.TrimSpace(*input.BaseURL)
		if baseURL == "" {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "base_url is required", nil, "302ffb5e-243f-4112-99ec-e4f9bfbc331a")
		}
		if _, err := url.ParseRequestURI(baseURL); err != nil {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, fmt.Sprintf("invalid base_url format: %v", err), nil, "0037a0ec-1342-49e9-8479-cda7db9d1ce8")
		}
		normalized := normalizeURL(baseURL)
		provider.SetEndpoints(EndpointList{{URL: normalized, Weight: 1, Healthy: true}})
	}
	if input.APIKey != nil {
		key := strings.TrimSpace(*input.APIKey)
		if key == "" {
			provider.EncryptedAPIKey = ""
			provider.APIKeyHint = nil
		} else {
			secret := strings.TrimSpace(config.GetGlobal().ModelProviderSecret)
			if secret == "" {
				return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeInternal, "model provider secret is not configured", nil, "b31c3083-4a15-4e86-baf9-35fc557cfa0a")
			}
			cipher, err := crypto.EncryptString(secret, key)
			if err != nil {
				return nil, err
			}
			provider.EncryptedAPIKey = cipher
			provider.APIKeyHint = apiKeyHint(key)
		}
	}
	if input.Metadata != nil {
		sanitized := sanitizeMetadata(*input.Metadata)
		// Apply default capabilities for missing keys (don't override user-provided values)
		provider.Metadata = setDefaultCapabilities(provider.Kind, sanitized)
	}
	if input.Category != nil {
		provider.Category = *input.Category
	}
	shouldDisableProviderModels := false
	if input.Active != nil {
		shouldDisableProviderModels = provider.Active && !*input.Active
	}
	if input.Active != nil {
		provider.Active = *input.Active
	}
	if input.DefaultImageGenerate != nil {
		provider.DefaultImageGenerate = *input.DefaultImageGenerate
	}
	if input.DefaultImageEdit != nil {
		provider.DefaultImageEdit = *input.DefaultImageEdit
	}
	if err := s.providerRepo.Update(ctx, provider); err != nil {
		return nil, err
	}

	if shouldDisableProviderModels {
		// Disable all provider models when the provider is disabled to keep routing consistent
		filter := ProviderModelFilter{
			ProviderID: ptr.ToUint(provider.ID),
		}
		if _, err := s.providerModelService.BatchUpdateActive(ctx, filter, false); err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to disable provider models")
		}
	}

	return provider, nil
}

func normalizeAndValidateEndpoints(endpoints EndpointList) (EndpointList, error) {
	if len(endpoints) == 0 {
		return endpoints, nil
	}

	normalized := make(EndpointList, 0, len(endpoints))
	for _, ep := range endpoints {
		urlStr := strings.TrimSpace(ep.URL)
		if urlStr == "" {
			continue
		}
		if _, err := url.ParseRequestURI(urlStr); err != nil {
			return nil, fmt.Errorf("invalid endpoint url %q: %w", urlStr, err)
		}
		if ep.Weight <= 0 {
			ep.Weight = 1
		}
		ep.URL = normalizeURL(urlStr)
		ep.Healthy = true
		normalized = append(normalized, ep)
	}
	return normalized, nil
}

func (s *ProviderService) SyncProviderModelsWithOptions(ctx context.Context, provider *Provider, models []chat.Model, autoEnableNewModels bool) ([]*ProviderModel, error) {
	log := logger.GetLogger()

	// Batch upsert catalogs (eliminates N+1 queries)
	catalogs, createdFlags, err := s.modelCatalogService.BatchUpsertCatalogs(ctx, provider, models)
	if err != nil {
		log.Error().
			Str("provider", provider.DisplayName).
			Err(err).
			Msg("failed to batch upsert catalogs")
		return nil, err
	}

	// Batch fetch existing provider models to check what already exists
	existingModels, err := s.providerModelService.FindByFilter(ctx, ProviderModelFilter{
		ProviderID: ptr.ToUint(provider.ID),
	})
	if err != nil {
		log.Error().
			Str("provider", provider.DisplayName).
			Err(err).
			Msg("failed to fetch existing provider models")
		return nil, err
	}

	// Build a map of existing modelPublicIDs for fast lookup
	existingModelPublicIDs := make(map[string]bool)
	for _, existingModel := range existingModels {
		existingModelPublicIDs[existingModel.ModelPublicID] = true
	}

	// Process provider models - only insert new ones
	results := make([]*ProviderModel, 0, len(models))
	skippedCount := 0

	for _, model := range models {
		// Get catalog from batch results
		publicID := catalogPublicID(provider.Kind, model.ID, model.CanonicalSlug)
		catalog, exists := catalogs[publicID]
		if !exists || catalog == nil {
			log.Warn().
				Str("model_id", model.ID).
				Str("public_id", publicID).
				Str("provider", provider.DisplayName).
				Msg("catalog not found in batch results")
			continue
		}

		// Generate ModelPublicID to check if it already exists
		kind := ProviderKind(provider.Kind)
		modelPublicID := NormalizeModelKey(kind, model.ID)

		// Skip if model already exists in provider_models
		if existingModelPublicIDs[modelPublicID] {
			skippedCount++
			continue
		}

		// Only create new models
		shouldAutoEnable := autoEnableNewModels && createdFlags[publicID]
		providerModel, err := s.providerModelService.UpsertProviderModelWithOptions(ctx, provider, catalog, model, shouldAutoEnable)
		if err != nil {
			log.Error().
				Str("model_id", model.ID).
				Str("provider", provider.DisplayName).
				Err(err).
				Msgf("failed to upsert provider model for '%s' from provider '%s'", model.ID, provider.DisplayName)
			continue
		}
		results = append(results, providerModel)
	}

	if skippedCount > 0 {
		log.Info().
			Str("provider", provider.DisplayName).
			Int("skipped", skippedCount).
			Int("created", len(results)).
			Msg("skipped existing models during sync")
	}

	now := time.Now().UTC()
	provider.LastSyncedAt = &now
	if err := s.providerRepo.Update(ctx, provider); err != nil {
		return nil, err
	}

	return results, nil
}

func sanitizeMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	result := make(map[string]string, len(metadata))
	for key, value := range metadata {
		k := strings.TrimSpace(key)
		v := strings.TrimSpace(value)
		if k == "" || v == "" {
			continue
		}
		result[k] = v
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// setDefaultCapabilities sets default capability metadata based on provider kind
// if not already present in the metadata map
func setDefaultCapabilities(kind ProviderKind, metadata map[string]string) map[string]string {
	if metadata == nil {
		metadata = make(map[string]string)
	}

	// Get default capabilities from the loaded defaults (providers_metadata_default.yml)
	defaults := GetDefaultCapabilities(kind)

	// Set image_input capability if not already configured
	if _, exists := metadata[MetadataKeyImageInput]; !exists {
		if imageInputJSON, err := json.Marshal(defaults.ImageInput); err == nil {
			metadata[MetadataKeyImageInput] = string(imageInputJSON)
		}
	}

	// Set file_attachment capability if not already configured
	if _, exists := metadata[MetadataKeyFileAttachment]; !exists {
		if fileAttachmentJSON, err := json.Marshal(defaults.FileAttachment); err == nil {
			metadata[MetadataKeyFileAttachment] = string(fileAttachmentJSON)
		}
	}

	return metadata
}

// FindActiveImageProvider returns the first active image provider
func (s *ProviderService) FindActiveImageProvider(ctx context.Context) (*Provider, error) {
	category := ProviderCategoryImage
	filter := ProviderFilter{
		Active:   ptr.ToBool(true),
		Category: &category,
	}
	providers, err := s.providerRepo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	if len(providers) == 0 {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
			platformerrors.ErrorTypeNotFound,
			"no active image provider configured", nil,
			"image-provider-not-found")
	}
	return providers[0], nil // Return first active image provider
}

// FindDefaultImageGenerateProvider returns the configured default image provider for generation.
func (s *ProviderService) FindDefaultImageGenerateProvider(ctx context.Context) (*Provider, error) {
	return s.findDefaultImageProvider(ctx, func(p *Provider) bool {
		return p.DefaultImageGenerate
	}, "image-generate")
}

// FindDefaultImageEditProvider returns the configured default image provider for edits.
func (s *ProviderService) FindDefaultImageEditProvider(ctx context.Context) (*Provider, error) {
	provider, err := s.findDefaultImageProvider(ctx, func(p *Provider) bool {
		return p.DefaultImageEdit
	}, "image-edit")
	if err == nil {
		return provider, nil
	}

	var platformErr *platformerrors.PlatformError
	if errors.As(err, &platformErr) && platformErr.UUID == "default-image-provider-not-found" {
		// Allow default edit providers that live outside the image category.
		if fallback, fallbackErr := s.findDefaultImageProviderAnyCategory(ctx, func(p *Provider) bool {
			return p.DefaultImageEdit
		}, "image-edit"); fallbackErr == nil {
			return fallback, nil
		}
		return s.findDefaultImageProvider(ctx, func(p *Provider) bool {
			return p.DefaultImageGenerate
		}, "image-generate")
	}

	return nil, err
}

func (s *ProviderService) findDefaultImageProvider(
	ctx context.Context,
	isDefault func(*Provider) bool,
	label string,
) (*Provider, error) {
	category := ProviderCategoryImage
	filter := ProviderFilter{
		Active:   ptr.ToBool(true),
		Category: &category,
	}
	providers, err := s.providerRepo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	if len(providers) == 0 {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
			platformerrors.ErrorTypeNotFound,
			"no active image provider configured", nil,
			"image-provider-not-found")
	}

	defaults := make([]*Provider, 0, 1)
	for _, provider := range providers {
		if provider != nil && isDefault(provider) {
			defaults = append(defaults, provider)
		}
	}

	if len(defaults) == 0 {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
			platformerrors.ErrorTypeNotFound,
			fmt.Sprintf("no default image provider configured for %s", label), nil,
			"default-image-provider-not-found")
	}
	if len(defaults) > 1 {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
			platformerrors.ErrorTypeValidation,
			fmt.Sprintf("multiple default image providers configured for %s", label),
			nil, "default-image-provider-multiple")
	}
	return defaults[0], nil
}

func (s *ProviderService) findDefaultImageProviderAnyCategory(
	ctx context.Context,
	isDefault func(*Provider) bool,
	label string,
) (*Provider, error) {
	filter := ProviderFilter{
		Active: ptr.ToBool(true),
	}
	providers, err := s.providerRepo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	if len(providers) == 0 {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
			platformerrors.ErrorTypeNotFound,
			"no active provider configured", nil,
			"default-image-provider-not-found")
	}

	defaults := make([]*Provider, 0, 1)
	for _, provider := range providers {
		if provider != nil && isDefault(provider) {
			defaults = append(defaults, provider)
		}
	}

	if len(defaults) == 0 {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
			platformerrors.ErrorTypeNotFound,
			fmt.Sprintf("no default provider configured for %s", label), nil,
			"default-image-provider-not-found")
	}
	if len(defaults) > 1 {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain,
			platformerrors.ErrorTypeValidation,
			fmt.Sprintf("multiple default providers configured for %s", label),
			nil, "default-image-provider-multiple")
	}
	return defaults[0], nil
}

// FindAllActiveProvidersByCategory returns all active providers of a specific category
func (s *ProviderService) FindAllActiveProvidersByCategory(ctx context.Context, category ProviderCategory) ([]*Provider, error) {
	filter := ProviderFilter{
		Active:   ptr.ToBool(true),
		Category: &category,
	}
	return s.providerRepo.FindByFilter(ctx, filter, nil)
}

// GetProvidersByCategory returns providers by category with count
func (s *ProviderService) GetProvidersByCategory(ctx context.Context, category ProviderCategory) ([]*Provider, int, error) {
	filter := ProviderFilter{
		Category: &category,
	}
	providers, err := s.providerRepo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return nil, 0, err
	}
	return providers, len(providers), nil
}
