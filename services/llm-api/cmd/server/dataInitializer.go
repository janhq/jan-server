package main

import (
	"context"
	"fmt"
	"strings"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/domain/model"
	"jan-server/services/llm-api/internal/domain/prompttemplate"
	"jan-server/services/llm-api/internal/infrastructure/inference"
	"jan-server/services/llm-api/internal/infrastructure/logger"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

type DataInitializer struct {
	provider              *model.ProviderService
	modelCatalogService   *model.ModelCatalogService
	inferenceProvider     *inference.InferenceProvider
	promptTemplateService *prompttemplate.Service
}

func (d *DataInitializer) Install(ctx context.Context) error {
	log := logger.GetLogger()
	cfg := config.GetGlobal()

	// Ensure default prompt templates exist
	if err := d.promptTemplateService.EnsureDefaultTemplates(ctx); err != nil {
		log.Warn().Err(err).Msg("failed to ensure default prompt templates, continuing with fallback")
	}

	if entries := cfg.ProviderBootstrapEntries(); len(entries) > 0 {
		// If we already have active providers in the database, skip bootstrapping from providers.yml.
		activeProviders, err := d.provider.FindAllActiveProviders(ctx)
		if err != nil {
			return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to check existing providers before bootstrap")
		}
		if len(activeProviders) > 0 {
			log.Info().
				Int("active_providers", len(activeProviders)).
				Msg("skipping providers.yml bootstrap because active providers already exist")
			return nil
		}

		if err := d.setupConfiguredProviders(ctx, entries); err != nil {
			return err
		}
		return nil
	}

	return nil
}

func (d *DataInitializer) setupConfiguredProviders(ctx context.Context, entries []config.ProviderBootstrapEntry) error {
	for i := range entries {
		entry := entries[i]
		if err := d.bootstrapProvider(ctx, entry); err != nil {
			return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, fmt.Sprintf("failed to bootstrap provider %q", entry.Name))
		}
	}
	return nil
}

func (d *DataInitializer) bootstrapProvider(ctx context.Context, entry config.ProviderBootstrapEntry) error {
	provider, err := d.ensureProvider(ctx, entry)
	if err != nil {
		return err
	}

	if !entry.SyncModels {
		return nil
	}
	if provider == nil || provider.Category != model.ProviderCategoryLLM {
		return nil
	}

	models, err := d.inferenceProvider.ListModels(ctx, provider)
	if err != nil {
		return err
	}

	_, err = d.provider.SyncProviderModelsWithOptions(ctx, provider, models, entry.AutoEnableNewModels)
	return err
}

func (d *DataInitializer) ensureProvider(ctx context.Context, entry config.ProviderBootstrapEntry) (*model.Provider, error) {
	kind := model.ProviderKindFromVendor(entry.Vendor)
	metadata := cloneMetadata(entry.Metadata)
	endpoints := toDomainEndpoints(entry.Endpoints)
	category := model.ProviderCategory(entry.Category)
	if category == "" {
		switch kind {
		case model.ProviderOCR:
			category = model.ProviderCategoryOCR
		case model.ProviderZImage:
			category = model.ProviderCategoryImage
		default:
			category = model.ProviderCategoryLLM
		}
	}

	if kind == model.ProviderCustom {
		return d.provider.UpsertProvider(ctx, model.UpsertProviderInput{
			Name:      entry.Name,
			Vendor:    entry.Vendor,
			Category:  category,
			BaseURL:   entry.BaseURL,
			Endpoints: endpoints,
			APIKey:    entry.APIKey,
			Metadata:  metadata,
			Active:    entry.Active,
		})
	}

	existing, err := d.provider.FindProviderByVendor(ctx, entry.Vendor)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		return d.provider.RegisterProvider(ctx, model.RegisterProviderInput{
			Name:      entry.Name,
			Vendor:    entry.Vendor,
			Category:  category,
			BaseURL:   entry.BaseURL,
			Endpoints: endpoints,
			APIKey:    entry.APIKey,
			Metadata:  metadata,
			Active:    entry.Active,
		})
	}

	updateMetadata := metadata
	updateInput := model.UpdateProviderInput{
		BaseURL:   &entry.BaseURL,
		Endpoints: &endpoints,
		APIKey:    &entry.APIKey,
		Metadata:  &updateMetadata,
		Active:    &entry.Active,
	}
	if entry.Name != "" && entry.Name != existing.DisplayName {
		updateInput.Name = &entry.Name
	}

	updated, err := d.provider.UpdateProvider(ctx, existing, updateInput)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func cloneMetadata(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func toDomainEndpoints(cfgs []config.EndpointConfig) model.EndpointList {
	if len(cfgs) == 0 {
		return nil
	}
	endpoints := make(model.EndpointList, 0, len(cfgs))
	for _, ep := range cfgs {
		urlStr := strings.TrimSpace(ep.URL)
		if urlStr == "" {
			continue
		}
		weight := ep.Weight
		if weight <= 0 {
			weight = 1
		}
		endpoints = append(endpoints, model.Endpoint{
			URL:      urlStr,
			Weight:   weight,
			Priority: ep.Priority,
			Healthy:  true,
		})
	}
	return endpoints
}
