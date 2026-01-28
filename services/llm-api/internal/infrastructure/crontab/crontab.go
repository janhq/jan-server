package crontab

import (
	"context"
	"fmt"
	"sync"
	"time"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/domain/model"
	"jan-server/services/llm-api/internal/infrastructure/inference"
	"jan-server/services/llm-api/internal/infrastructure/logger"
	"jan-server/services/llm-api/internal/utils/platformerrors"

	"github.com/mileusna/crontab"
)

const (
	MetadataAutoEnableNewModels = "auto_enable_new_models" // "true" or "false"
	DefaultModelSyncInterval    = 1                        // in minutes
	CronJobTimeout              = 10 * time.Minute         // Timeout for each cron job execution
)

type Crontab struct {
	ctab              *crontab.Crontab
	providerService   *model.ProviderService
	inferenceProvider *inference.InferenceProvider
}

func NewCrontab(
	providerService *model.ProviderService,
	inferenceProvider *inference.InferenceProvider,
) *Crontab {
	return &Crontab{
		ctab:              crontab.New(),
		providerService:   providerService,
		inferenceProvider: inferenceProvider,
	}
}

func (c *Crontab) Run(ctx context.Context) error {
	log := logger.GetLogger()
	// execute once on server start
	c.syncAllProviderModels(ctx)

	// Schedule model sync job if enabled
	cfg := config.GetGlobal()
	if cfg != nil && cfg.ModelSyncEnabled {
		syncInterval := cfg.ModelSyncIntervalMinutes
		if syncInterval <= 0 {
			syncInterval = DefaultModelSyncInterval
		}

		cronExpr := fmt.Sprintf("*/%d * * * *", syncInterval)
		if err := c.ctab.AddJob(cronExpr, func() {
			jobCtx, cancel := context.WithTimeout(context.Background(), CronJobTimeout)
			defer cancel()
			c.syncAllProviderModels(jobCtx)
		}); err != nil {
			return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to add model sync job")
		}
		log.Warn().Msgf("Model sync scheduled: every %d minute(s)", syncInterval)
	}

	// Schedule environment reload job
	if err := c.ctab.AddJob("* * * * *", func() {
		// Reload config
		config.Load()
	}); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to add env reload job")
	}

	<-ctx.Done()
	c.ctab.Shutdown()
	return nil
}

func (c *Crontab) syncAllProviderModels(ctx context.Context) {
	log := logger.GetLogger()
	providers, err := c.providerService.FindAllActiveProviders(ctx)

	if err != nil {
		log.Error().Err(err).Msg("Failed to list providers for sync")
		return
	}

	if len(providers) == 0 {
		return
	}

	const maxConcurrentSyncs = 10
	sem := make(chan struct{}, maxConcurrentSyncs)
	var wg sync.WaitGroup

	for _, provider := range providers {
		wg.Add(1)
		go func(p *model.Provider) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			c.syncProviderModels(ctx, p)
		}(provider)
	}
	wg.Wait()
}

func (c *Crontab) syncProviderModels(ctx context.Context, provider *model.Provider) {
	log := logger.GetLogger()

	if provider == nil || provider.Category != model.ProviderCategoryLLM {
		return
	}

	models, err := c.inferenceProvider.ListModels(ctx, provider)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch models from provider")
		return
	}

	if len(models) == 0 {
		return
	}

	autoEnable := provider.Metadata != nil && provider.Metadata[MetadataAutoEnableNewModels] == "true"

	if _, err := c.providerService.SyncProviderModelsWithOptions(ctx, provider, models, autoEnable); err != nil {
		log.Error().Err(err).Msg("Failed to sync provider models")
		return
	}

	log.Info().Msgf("Synced %d models", len(models))
}
