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

	"jan-server/services/media-api/internal/config"
	domain "jan-server/services/media-api/internal/domain/media"
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
	httpServer *httpserver.HttpServer
	log        zerolog.Logger
}

func NewApplication(httpServer *httpserver.HttpServer, log zerolog.Logger) *Application {
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
		DSN:             cfg.DatabaseURL,
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

	httpServer := httpserver.New(cfg, log, mediaService)
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
