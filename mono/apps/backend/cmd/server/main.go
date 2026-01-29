package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jan-server/mono/apps/backend/internal/infrastructure/config"
	"jan-server/mono/apps/backend/internal/infrastructure/database"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	_ "github.com/joho/godotenv/autoload"
	_ "net/http/pprof"
)

// @title Jan Server Unified API
// @version 3.0
// @description Unified backend API combining LLM, Response, Media, MCP, Memory, and Realtime services.
// @description Supports both Keycloak OIDC and local password authentication.
// @contact.name Jan Server Team
// @contact.url https://github.com/janhq/jan-server
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token (Keycloak or local).

// @securityDefinitions.apikey APIKeyAuth
// @in header
// @name X-API-Key
// @description API Key starting with "sk_"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize logger
	initLogger()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// Reconfigure logger with config values
	configureLogger(cfg)

	log.Info().
		Str("version", config.Version).
		Str("environment", cfg.Environment).
		Bool("keycloak_enabled", cfg.KeycloakEnabled).
		Bool("local_auth_enabled", cfg.LocalAuthEnabled).
		Bool("kong_enabled", cfg.KongEnabled).
		Msg("starting jan-server unified backend")

	// Initialize database
	db, err := database.NewConnection(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	// Run migrations if enabled
	if cfg.AutoMigrate {
		if err := database.RunMigrations(db, cfg); err != nil {
			log.Fatal().Err(err).Msg("failed to run migrations")
		}
	}

	// Create HTTP server
	server := httpserver.NewServer(cfg, db)

	// Start server with graceful shutdown
	var eg errgroup.Group

	// pprof server for debugging
	if cfg.EnablePprof {
		eg.Go(func() error {
			log.Info().Int("port", 6060).Msg("starting pprof server")
			return http.ListenAndServe("0.0.0.0:6060", nil)
		})
	}

	// Main HTTP server
	eg.Go(func() error {
		return server.Run()
	})

	// Wait for shutdown signal
	<-ctx.Done()
	log.Info().Msg("shutdown signal received")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("failed to shutdown server gracefully")
	}

	log.Info().Msg("server stopped")
}

func initLogger() {
	// Default console logger for startup
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		With().
		Timestamp().
		Caller().
		Logger()
}

func configureLogger(cfg *config.Config) {
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.LogFormat == "json" {
		log.Logger = zerolog.New(os.Stderr).
			With().
			Timestamp().
			Str("service", cfg.ServiceName).
			Str("env", cfg.Environment).
			Logger()
	} else {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
			With().
			Timestamp().
			Str("service", cfg.ServiceName).
			Logger()
	}
}
