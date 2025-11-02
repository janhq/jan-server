package crontab

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mileusna/crontab"
	"github.com/sirupsen/logrus"
	"menlo.ai/menlo-platform/config/envs"
	"menlo.ai/menlo-platform/internal/domain/accesspolicy"
	"menlo.ai/menlo-platform/internal/domain/model"
	"menlo.ai/menlo-platform/internal/infrastructure/inference"
	"menlo.ai/menlo-platform/internal/utils/logger"
	"menlo.ai/menlo-platform/internal/utils/platformerrors"
)

const (
	MetadataAutoEnableNewModels = "auto_enable_new_models" // "true" or "false"
	DefaultModelSyncInterval    = 1                        // in minutes
	CronJobTimeout              = 10 * time.Minute         // Timeout for each cron job execution
)

type Crontab struct {
	ctab                *crontab.Crontab
	providerService     *model.ProviderService
	inferenceProvider   *inference.InferenceProvider
	accesspolicyService *accesspolicy.AccessPolicyService
}

func NewCrontab(
	providerService *model.ProviderService,
	inferenceProvider *inference.InferenceProvider,
	accesspolicyService *accesspolicy.AccessPolicyService,
) *Crontab {
	return &Crontab{
		ctab:                crontab.New(),
		providerService:     providerService,
		inferenceProvider:   inferenceProvider,
		accesspolicyService: accesspolicyService,
	}
}

func (c *Crontab) Run(ctx context.Context) error {
	// execute once on server start
	c.syncAllProviderModels(ctx)

	// Schedule model sync job if enabled
	if envs.ENV.MODEL_SYNC_ENABLED {
		syncInterval := envs.ENV.MODEL_SYNC_INTERVAL_MINUTES
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
		logger.GetLogger().Infof("Model sync scheduled: every %d minute(s)", syncInterval)
	}

	// Schedule environment reload job
	if err := c.ctab.AddJob("* * * * *", func() {
		envs.ENV.LoadFromEnv()
	}); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to add env reload job")
	}

	<-ctx.Done()
	c.ctab.Shutdown()
	return nil
}

func (c *Crontab) syncAllProviderModels(ctx context.Context) {
	providers, err := c.providerService.FindAllActiveProviders(ctx)

	if err != nil {
		logger.GetLogger().WithError(err).Error("Failed to list providers for sync")
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
	c.accesspolicyService.LoadModelAccessPolicy(ctx)
}

func (c *Crontab) syncProviderModels(ctx context.Context, provider *model.Provider) {
	log := logger.GetLogger().WithFields(logrus.Fields{
		"provider_id":   provider.PublicID,
		"provider_name": provider.DisplayName,
	})

	models, err := c.inferenceProvider.ListModels(ctx, provider)
	if err != nil {
		log.WithError(err).Error("Failed to fetch models from provider")
		return
	}

	if len(models) == 0 {
		return
	}

	autoEnable := provider.Metadata != nil && provider.Metadata[MetadataAutoEnableNewModels] == "true"

	if _, err := c.providerService.SyncProviderModelsWithOptions(ctx, provider, models, autoEnable); err != nil {
		log.WithError(err).Error("Failed to sync provider models")
		return
	}

	log.Infof("Synced %d models", len(models))
}
