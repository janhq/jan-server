//go:build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"jan-server/services/response-api/internal/config"
	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/conversation"
	"jan-server/services/response-api/internal/domain/llm"
	"jan-server/services/response-api/internal/domain/plan"
	responseDomain "jan-server/services/response-api/internal/domain/response"
	"jan-server/services/response-api/internal/domain/tool"
	"jan-server/services/response-api/internal/infrastructure/auth"
	"jan-server/services/response-api/internal/infrastructure/database"
	"jan-server/services/response-api/internal/infrastructure/llmprovider"
	"jan-server/services/response-api/internal/infrastructure/logger"
	"jan-server/services/response-api/internal/infrastructure/media"
	"jan-server/services/response-api/internal/infrastructure/mcp"
	artifactrepo "jan-server/services/response-api/internal/infrastructure/repository/artifact"
	conversationrepo "jan-server/services/response-api/internal/infrastructure/repository/conversation"
	planrepo "jan-server/services/response-api/internal/infrastructure/repository/plan"
	responseRepo "jan-server/services/response-api/internal/infrastructure/repository/response"
	"jan-server/services/response-api/internal/interfaces/httpserver"
	"jan-server/services/response-api/internal/webhook"
)

var responseSet = wire.NewSet(
	responseRepo.NewPostgresRepository,
	wire.Bind(new(responseDomain.Repository), new(*responseRepo.PostgresRepository)),
	wire.Bind(new(responseDomain.ToolExecutionRepository), new(*responseRepo.PostgresRepository)),
	planrepo.NewPostgresRepository,
	wire.Bind(new(plan.Repository), new(*planrepo.PostgresRepository)),
	artifactrepo.NewPostgresRepository,
	wire.Bind(new(artifact.Repository), new(*artifactrepo.PostgresRepository)),
	conversationrepo.NewRepository,
	wire.Bind(new(conversation.Repository), new(*conversationrepo.Repository)),
	conversationrepo.NewItemRepository,
	wire.Bind(new(conversation.ItemRepository), new(*conversationrepo.ItemRepository)),
	newLLMProvider,
	wire.Bind(new(llm.Provider), new(*llmprovider.Client)),
	wire.Bind(new(llm.ModelInfoProvider), new(*llmprovider.Client)),
	newMCPClient,
	wire.Bind(new(tool.MCPClient), new(*mcp.Client)),
	wire.Bind(new(planners.MCPClient), new(*mcp.Client)),
	newMediaClient,
	newOrchestrator,
	newWebhookService,
	wire.Bind(new(webhook.Service), new(*webhook.HTTPService)),
	plan.NewService,
	newAgentRegistry,
	newResponseService,
	artifact.NewService,
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

func newMediaClient(cfg *config.Config) *media.Client {
	return media.NewClient(cfg.MediaAPIURL)
}

func newOrchestrator(cfg *config.Config, provider llm.Provider, mcpClient tool.MCPClient) *tool.Orchestrator {
	return tool.NewOrchestrator(provider, mcpClient, cfg.MaxToolDepth, cfg.ToolTimeout)
}

func newWebhookService(log zerolog.Logger) *webhook.HTTPService {
	return webhook.NewHTTPService(log)
}

func newAgentRegistry(planService plan.Service, mcpClient tool.MCPClient, llmProvider llm.Provider, artifactService artifact.Service, cfg *config.Config) agent.Registry {
	registry := agent.NewRegistry()

	// Register the deep research planner
	deepResearchPlanner := planners.NewDeepResearchPlanner(planService)
	if err := registry.RegisterPlanner(deepResearchPlanner); err != nil {
		// Log but don't fail - planner registration is not critical
		_ = err
	}

	// Register the slide generator planner
	slideGeneratorPlanner := planners.NewSlideGeneratorPlanner(planService, artifactService)
	if err := registry.RegisterPlanner(slideGeneratorPlanner); err != nil {
		_ = err
	}

	// Create code fixer for LLM-based code fix retry
	codeFixer := llm.NewCodeFixer(llmProvider, cfg.CodeFixModel)

	// Register the deep research executor for tool calls and LLM calls
	deepResearchExecutor := planners.NewDeepResearchExecutor(mcpClient, codeFixer)
	_ = registry.RegisterExecutor(plan.ActionTypeToolCall, deepResearchExecutor)
	_ = registry.RegisterExecutor(plan.ActionTypeLLMCall, deepResearchExecutor)

	// Register the slide generator executor for artifact creation
	slideGeneratorExecutor := planners.NewSlideGeneratorExecutor(mcpClient, codeFixer, artifactService)
	_ = registry.RegisterExecutor(plan.ActionTypeArtifactCreate, slideGeneratorExecutor)

	return registry
}

func newResponseService(
	repo responseDomain.Repository,
	conversations conversation.Repository,
	conversationItems conversation.ItemRepository,
	toolRepo responseDomain.ToolExecutionRepository,
	orchestrator *tool.Orchestrator,
	mcpClient tool.MCPClient,
	mediaClient *media.Client,
	modelInfoProvider llm.ModelInfoProvider,
	webhookService webhook.Service,
	agentRegistry agent.Registry,
	planService plan.Service,
	log zerolog.Logger,
) responseDomain.Service {
	return responseDomain.NewService(repo, conversations, conversationItems, toolRepo, orchestrator, mcpClient, mediaClient, modelInfoProvider, webhookService, agentRegistry, planService, log)
}
