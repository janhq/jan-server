//go:build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/janhq/jan-server/packages/go-common/analytics"

	"jan-server/services/response-api/internal/config"
	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners"
	deepresearch "jan-server/services/response-api/internal/domain/agent/planners/deep_research"
	docplanner "jan-server/services/response-api/internal/domain/agent/planners/doc"
	pdfplanner "jan-server/services/response-api/internal/domain/agent/planners/pdf"
	skillexec "jan-server/services/response-api/internal/domain/agent/planners/skill"
	slide_creator "jan-server/services/response-api/internal/domain/agent/planners/slide_creator"
	spreadsheetplanner "jan-server/services/response-api/internal/domain/agent/planners/spreadsheet"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/conversation"
	"jan-server/services/response-api/internal/domain/llm"
	"jan-server/services/response-api/internal/domain/plan"
	responseDomain "jan-server/services/response-api/internal/domain/response"
	"jan-server/services/response-api/internal/domain/skill"
	"jan-server/services/response-api/internal/domain/tool"
	"jan-server/services/response-api/internal/infrastructure/apikey"
	"jan-server/services/response-api/internal/infrastructure/auth"
	"jan-server/services/response-api/internal/infrastructure/database"
	"jan-server/services/response-api/internal/infrastructure/llmprovider"
	"jan-server/services/response-api/internal/infrastructure/logger"
	"jan-server/services/response-api/internal/infrastructure/mcp"
	"jan-server/services/response-api/internal/infrastructure/media"
	artifactrepo "jan-server/services/response-api/internal/infrastructure/repository/artifact"
	conversationrepo "jan-server/services/response-api/internal/infrastructure/repository/conversation"
	planrepo "jan-server/services/response-api/internal/infrastructure/repository/plan"
	responseRepo "jan-server/services/response-api/internal/infrastructure/repository/response"
	skillinfra "jan-server/services/response-api/internal/infrastructure/skill"
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
	newMediaClient,
	newSkillService,
	wire.Bind(new(skill.Service), new(*skillinfra.Service)),
	newOrchestrator,
	newAgentOrchestrator,
	newWebhookService,
	wire.Bind(new(webhook.Service), new(*webhook.HTTPService)),
	newAPIKeyProvider,
	wire.Bind(new(apikey.Provider), new(*apikey.Client)),
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
		newAnalyticsTracker,
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

func newAPIKeyProvider(cfg *config.Config) *apikey.Client {
	return apikey.NewClient(cfg.LLMAPIURL)
}

func newMediaClient(cfg *config.Config) *media.Client {
	return media.NewClient(cfg.MediaAPIURL)
}

func newSkillService() (*skillinfra.Service, error) {
	return skillinfra.NewService()
}

func newOrchestrator(cfg *config.Config, provider llm.Provider, mcpClient tool.MCPClient) *tool.Orchestrator {
	return tool.NewOrchestrator(provider, mcpClient, cfg.MaxToolDepth, cfg.ToolTimeout, cfg.LLMStreamMode)
}

func newAgentOrchestrator(registry agent.Registry, planService plan.Service, mcpClient *mcp.Client) agent.Orchestrator {
	return agent.NewOrchestrator(registry, planService, mcpClient)
}

func newWebhookService(log zerolog.Logger) *webhook.HTTPService {
	return webhook.NewHTTPService(log)
}

func newAgentRegistry(planService plan.Service, mcpClient tool.MCPClient, llmProvider llm.Provider, artifactService artifact.Service, cfg *config.Config, mediaClient *media.Client, skillService skill.Service) agent.Registry {
	registry := agent.NewRegistry()

	// Register the deep research planner
	deepResearchPlanner := deepresearch.NewPlanner(planService)
	if err := registry.RegisterPlanner(deepResearchPlanner); err != nil {
		// Log but don't fail - planner registration is not critical
		_ = err
	}

	// Register the slide creator planner
	slideCreatorPlanner := slide_creator.NewSlideCreatorPlanner(planService, artifactService)
	if err := registry.RegisterPlanner(slideCreatorPlanner); err != nil {
		_ = err
	}

	docGeneratorPlanner := docplanner.NewPlanner(planService, artifactService)
	if err := registry.RegisterPlanner(docGeneratorPlanner); err != nil {
		_ = err
	}

	pdfGeneratorPlanner := pdfplanner.NewPlanner(planService, artifactService)
	if err := registry.RegisterPlanner(pdfGeneratorPlanner); err != nil {
		_ = err
	}

	spreadsheetGeneratorPlanner := spreadsheetplanner.NewPlanner(planService, artifactService)
	if err := registry.RegisterPlanner(spreadsheetGeneratorPlanner); err != nil {
		_ = err
	}

	// Create code fixer for LLM-based code fix retry
	codeFixer := llm.NewCodeFixer(llmProvider, cfg.CodeFixModel)
	codeFixer.SetDisableCustomTemperature(cfg.LLMDisableCustomTemperature)

	// Register the deep research executor for tool calls and LLM calls
	deepResearchExecutor := deepresearch.NewExecutor(mcpClient, codeFixer)

	// Register the skill executor for artifact creation
	skillExecutor := skillexec.NewExecutor(
		mcpClient,
		codeFixer,
		skillService,
		cfg.SkillExecutionEnabled,
		cfg.SkillMaxInstallRetries,
		cfg.SkillMaxCodeFixRetries,
		cfg.SkillMaxFileSize,
		cfg.SkillExecutionTimeout,
		map[skill.SkillType]bool{
			skill.SkillTypeSlides:       cfg.SkillSlidesEnabled,
			skill.SkillTypeDocs:         cfg.SkillDocsEnabled,
			skill.SkillTypePDFs:         cfg.SkillPDFsEnabled,
			skill.SkillTypeSpreadsheets: cfg.SkillSpreadsheetsEnabled,
		},
	)
	slideCreatorExecutor := slide_creator.NewSlideCreatorExecutor(mcpClient, codeFixer, artifactService, mediaClient, cfg)
	routingExecutor := planners.NewRoutingExecutor(deepResearchExecutor, slideCreatorExecutor)
	artifactRoutingExecutor := planners.NewRoutingExecutor(slideCreatorExecutor, slideCreatorExecutor)
	_ = registry.RegisterExecutor(plan.ActionTypeToolCall, routingExecutor)
	_ = registry.RegisterExecutor(plan.ActionTypeLLMCall, routingExecutor)
	_ = registry.RegisterExecutor(plan.ActionTypeArtifactCreate, artifactRoutingExecutor)
	_ = registry.RegisterExecutor(plan.ActionTypeTransform, slideCreatorExecutor)
	_ = registry.RegisterExecutor(plan.ActionTypeSkillExecute, skillExecutor)

	return registry
}

func newResponseService(
	repo responseDomain.Repository,
	conversations conversation.Repository,
	conversationItems conversation.ItemRepository,
	toolRepo responseDomain.ToolExecutionRepository,
	orchestrator *tool.Orchestrator,
	agentOrchestrator agent.Orchestrator,
	mcpClient tool.MCPClient,
	mediaClient *media.Client,
	modelInfoProvider llm.ModelInfoProvider,
	webhookService webhook.Service,
	agentRegistry agent.Registry,
	planService plan.Service,
	log zerolog.Logger,
) responseDomain.Service {
	return responseDomain.NewService(repo, conversations, conversationItems, toolRepo, orchestrator, agentOrchestrator, mcpClient, mediaClient, modelInfoProvider, webhookService, agentRegistry, planService, log)
}

// newAnalyticsTracker creates the analytics tracker from config
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
