package infrastructure

import (
	"context"

	"github.com/google/wire"
	"github.com/rs/zerolog"

	"github.com/janhq/jan-server/packages/go-common/analytics"

	"jan-server/services/realtime-api/internal/config"
	"jan-server/services/realtime-api/internal/domain/session"
	"jan-server/services/realtime-api/internal/infrastructure/auth"
	"jan-server/services/realtime-api/internal/infrastructure/livekit"
	"jan-server/services/realtime-api/internal/infrastructure/logger"
	"jan-server/services/realtime-api/internal/infrastructure/store"
)

// ProvideConfig loads and provides the application configuration.
func ProvideConfig() (*config.Config, error) {
	return config.Load()
}

// ProvideLogger provides a configured logger.
func ProvideLogger(cfg *config.Config) zerolog.Logger {
	return logger.New(cfg)
}

// ProvideTokenGenerator provides a LiveKit token generator.
func ProvideTokenGenerator(cfg *config.Config) session.TokenGenerator {
	return livekit.NewTokenGenerator(cfg)
}

// ProvideRoomClient provides a LiveKit room client.
func ProvideRoomClient(cfg *config.Config) *livekit.RoomClient {
	return livekit.NewRoomClient(cfg)
}

// ProvideSessionStore provides a session store.
func ProvideSessionStore(log zerolog.Logger) session.Store {
	return store.NewMemoryStore(log)
}

// ProvideSyncer provides a session syncer.
func ProvideSyncer(
	sessionStore session.Store,
	roomClient *livekit.RoomClient,
	cfg *config.Config,
	log zerolog.Logger,
) *store.Syncer {
	return store.NewSyncer(sessionStore, roomClient, cfg.SessionStaleTTL, cfg.SessionCleanupInterval, log)
}

// ProvideAuthValidator provides an auth validator.
func ProvideAuthValidator(ctx context.Context, cfg *config.Config, log zerolog.Logger) (*auth.Validator, error) {
	return auth.NewValidator(ctx, cfg, log)
}

// ProvideAnalyticsTracker creates the analytics tracker from config.
func ProvideAnalyticsTracker(cfg *config.Config, log zerolog.Logger) analytics.Tracker {
	analyticsCfg := analytics.Config{
		Enabled:     cfg.AnalyticsEnabled,
		Environment: cfg.AnalyticsEnvironment,
		PIILevel:    cfg.AnalyticsPIILevel,
		PostHog: analytics.PostHogConfig{
			Enabled:       cfg.PostHogEnabled,
			APIKey:        cfg.PostHogAPIKey,
			Host:          cfg.PostHogHost,
			Debug:         cfg.PostHogDebug,
			BatchSize:     cfg.PostHogBatchSize,
			FlushInterval: cfg.PostHogFlushInterval,
		},
		OTel: analytics.OTelConfig{
			Enabled:  cfg.OTelAnalyticsEnabled,
			Endpoint: cfg.OTLPEndpoint,
		},
	}

	// Create sanitizer for PII protection
	sanitizer := analytics.NewSanitizer(analytics.PIILevel(cfg.AnalyticsPIILevel), cfg.ServiceName)

	tracker, err := analytics.NewTracker(analyticsCfg, sanitizer)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create analytics tracker, using no-op")
		return analytics.NewNoopTracker()
	}

	log.Info().
		Bool("enabled", analyticsCfg.Enabled).
		Bool("posthog", analyticsCfg.PostHog.Enabled).
		Bool("otel", analyticsCfg.OTel.Enabled).
		Str("environment", analyticsCfg.Environment).
		Msg("Analytics tracker initialized")

	return tracker
}

// InfrastructureProvider provides all infrastructure dependencies.
var InfrastructureProvider = wire.NewSet(
	// Config
	ProvideConfig,

	// Logger
	ProvideLogger,

	// LiveKit
	ProvideTokenGenerator,
	ProvideRoomClient,

	// Store
	ProvideSessionStore,
	ProvideSyncer,

	// Auth
	ProvideAuthValidator,

	// Analytics
	ProvideAnalyticsTracker,
)
