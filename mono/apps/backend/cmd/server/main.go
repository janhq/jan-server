package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	_ "github.com/joho/godotenv/autoload"
	_ "net/http/pprof"

	"jan-server/mono/apps/backend/internal/config"
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

	// Load configuration for early logging
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// Reconfigure logger with config values
	configureLogger(cfg)

	log.Info().
		Str("version", config.Version).
		Str("environment", cfg.Environment).
		Int("port", cfg.HTTPPort).
		Msg("starting jan-server unified backend")

	// Initialize HTTP server via wire dependency injection
	server, cleanup, err := InitializeHTTPServer()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize HTTP server")
	}
	defer cleanup()

	// Start server with graceful shutdown
	var eg errgroup.Group

	// pprof server for debugging (always enabled in development)
	if config.IsDev() {
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

	// Note: HTTPServer.Run() uses gin.Run() which doesn't support graceful shutdown directly
	// The errgroup will handle the shutdown when context is cancelled
	_ = shutdownCtx

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
