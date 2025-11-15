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

	"jan-server/services/response-api/internal/config"
	"jan-server/services/response-api/internal/domain/response"
	"jan-server/services/response-api/internal/domain/tool"
	"jan-server/services/response-api/internal/infrastructure/auth"
	"jan-server/services/response-api/internal/infrastructure/database"
	"jan-server/services/response-api/internal/infrastructure/llmprovider"
	"jan-server/services/response-api/internal/infrastructure/logger"
	"jan-server/services/response-api/internal/infrastructure/mcp"
	"jan-server/services/response-api/internal/infrastructure/observability"
	conversationrepo "jan-server/services/response-api/internal/infrastructure/repository/conversation"
	respRepo "jan-server/services/response-api/internal/infrastructure/repository/response"
	"jan-server/services/response-api/internal/interfaces/httpserver"
)

// @title Response API
// @version 1.0
// @description Orchestrates LLM responses with MCP tool integration, conversation context, and streaming support.
// @contact.name Jan Server Team
// @contact.url https://github.com/janhq/jan-server
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
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

	authValidator, err := auth.NewValidator(ctx, cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("initialize auth validator")
	}

	responseRepository := respRepo.NewPostgresRepository(db)
	conversationRepository := conversationrepo.NewRepository(db)
	conversationItemRepository := conversationrepo.NewItemRepository(db)
	llmClient := llmprovider.NewClient(cfg.LLMAPIURL)
	mcpClient := mcp.NewClient(cfg.MCPToolsURL)
	orchestrator := tool.NewOrchestrator(llmClient, mcpClient, cfg.MaxToolDepth, cfg.ToolTimeout)

	responseService := response.NewService(
		responseRepository,
		conversationRepository,
		conversationItemRepository,
		responseRepository,
		orchestrator,
		mcpClient,
		log,
	)

	httpServer := httpserver.New(cfg, log, responseService, authValidator)
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
