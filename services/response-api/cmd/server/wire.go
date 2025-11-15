//go:build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"jan-server/services/response-api/internal/config"
	"jan-server/services/response-api/internal/domain/conversation"
	"jan-server/services/response-api/internal/domain/llm"
	responseDomain "jan-server/services/response-api/internal/domain/response"
	"jan-server/services/response-api/internal/domain/tool"
	"jan-server/services/response-api/internal/infrastructure/auth"
	"jan-server/services/response-api/internal/infrastructure/database"
	"jan-server/services/response-api/internal/infrastructure/llmprovider"
	"jan-server/services/response-api/internal/infrastructure/logger"
	"jan-server/services/response-api/internal/infrastructure/mcp"
	conversationrepo "jan-server/services/response-api/internal/infrastructure/repository/conversation"
	responseRepo "jan-server/services/response-api/internal/infrastructure/repository/response"
	"jan-server/services/response-api/internal/interfaces/httpserver"
)

var responseSet = wire.NewSet(
	responseRepo.NewPostgresRepository,
	wire.Bind(new(responseDomain.Repository), new(*responseRepo.PostgresRepository)),
	wire.Bind(new(responseDomain.ToolExecutionRepository), new(*responseRepo.PostgresRepository)),
	conversationrepo.NewRepository,
	wire.Bind(new(conversation.Repository), new(*conversationrepo.Repository)),
	conversationrepo.NewItemRepository,
	wire.Bind(new(conversation.ItemRepository), new(*conversationrepo.ItemRepository)),
	newLLMProvider,
	wire.Bind(new(llm.Provider), new(*llmprovider.Client)),
	newMCPClient,
	wire.Bind(new(tool.MCPClient), new(*mcp.Client)),
	newOrchestrator,
	newResponseService,
)

// BuildApplication demonstrates how to assemble the response service with Wire.
func BuildApplication(ctx context.Context) (*Application, error) {
	wire.Build(
		config.Load,
		logger.New,
		newDatabaseConfig,
		newGormDB,
		newAuthValidator,
		responseSet,
		httpserver.New,
		NewApplication,
	)
	return nil, nil
}

func newDatabaseConfig(cfg *config.Config) database.Config {
	return database.Config{
		DSN:             cfg.GetDatabaseWriteDSN(),
		MaxIdleConns:    cfg.DBMaxIdleConns,
		MaxOpenConns:    cfg.DBMaxOpenConns,
		ConnMaxLifetime: cfg.DBConnLifetime,
		LogLevel:        gormlogger.Warn,
	}
}

func newGormDB(ctx context.Context, cfg database.Config, log zerolog.Logger) (*gorm.DB, error) {
	db, err := database.Connect(cfg)
	if err != nil {
		return nil, err
	}
	if err := database.AutoMigrate(ctx, db, log); err != nil {
		return nil, err
	}
	return db, nil
}

func newAuthValidator(ctx context.Context, cfg *config.Config, log zerolog.Logger) (*auth.Validator, error) {
	return auth.NewValidator(ctx, cfg, log)
}

func newLLMProvider(cfg *config.Config) *llmprovider.Client {
	return llmprovider.NewClient(cfg.LLMAPIURL)
}

func newMCPClient(cfg *config.Config) *mcp.Client {
	return mcp.NewClient(cfg.MCPToolsURL)
}

func newOrchestrator(cfg *config.Config, provider llm.Provider, mcpClient tool.MCPClient) *tool.Orchestrator {
	return tool.NewOrchestrator(provider, mcpClient, cfg.MaxToolDepth, cfg.ToolTimeout)
}

func newResponseService(
	repo responseDomain.Repository,
	conversations conversation.Repository,
	conversationItems conversation.ItemRepository,
	toolRepo responseDomain.ToolExecutionRepository,
	orchestrator *tool.Orchestrator,
	mcpClient tool.MCPClient,
	log zerolog.Logger,
) responseDomain.Service {
	return responseDomain.NewService(repo, conversations, conversationItems, toolRepo, orchestrator, mcpClient, log)
}
