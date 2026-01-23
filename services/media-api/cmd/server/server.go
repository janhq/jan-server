package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	gormlogger "gorm.io/gorm/logger"

	"github.com/janhq/jan-server/packages/go-common/analytics"

	"jan-server/services/media-api/internal/config"
	domain "jan-server/services/media-api/internal/domain/media"
	"jan-server/services/media-api/internal/infrastructure/auth"
	"jan-server/services/media-api/internal/infrastructure/database"
	"jan-server/services/media-api/internal/infrastructure/logger"
	"jan-server/services/media-api/internal/infrastructure/observability"
	repo "jan-server/services/media-api/internal/infrastructure/repository/media"
	"jan-server/services/media-api/internal/infrastructure/storage"
	"jan-server/services/media-api/internal/interfaces/httpserver"
)

// @title Media API
// @version 1.0
// @description Secure media ingestion and resolution service
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-Media-Service-Key
type Application struct {
	httpServer *httpserver.HTTPServer
	log        zerolog.Logger
}

func NewApplication(httpServer *httpserver.HTTPServer, log zerolog.Logger) *Application {
	return &Application{
		httpServer: httpServer,
		log:        log,
	}
}

func (a *Application) Start(ctx context.Context) error {
	return a.httpServer.Run(ctx)
}

func main() {
	loadEnvFiles()

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log := logger.New(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTelemetry, err := observability.Setup(ctx, cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("initialize observability")
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("shutdown telemetry")
		}
	}()

	db, err := database.Connect(database.Config{
		DSN:             cfg.GetDatabaseWriteDSN(),
		MaxIdleConns:    cfg.DBMaxIdleConns,
		MaxOpenConns:    cfg.DBMaxOpenConns,
		ConnMaxLifetime: cfg.DBConnLifetime,
		LogLevel:        gormlogger.Warn,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("connect database")
	}

	if err := database.AutoMigrate(ctx, db, log); err != nil {
		log.Fatal().Err(err).Msg("migrate database")
	}

	storageClient, err := storage.NewS3Storage(ctx, cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("initialize storage")
	}

	mediaRepository := repo.NewRepository(db)
	mediaService := domain.NewService(cfg, mediaRepository, storageClient, log)

	authValidator, err := auth.NewValidator(ctx, cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize auth validator")
	}

	// Initialize analytics tracker
	tracker := newAnalyticsTracker(cfg, log)

	httpServer := httpserver.New(cfg, log, mediaService, authValidator, tracker)
	app := NewApplication(httpServer, log)

	if err := app.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("application stopped with error")
	}

	log.Info().Msg("application exited cleanly")
}

func loadEnvFiles() {
	paths := []string{".env", "../.env"}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			if err := godotenv.Overload(path); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to load %s: %v\n", path, err)
			}
		}
	}
}

// newAnalyticsTracker creates the analytics tracker from config.
func newAnalyticsTracker(cfg *config.Config, log zerolog.Logger) analytics.Tracker {
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
