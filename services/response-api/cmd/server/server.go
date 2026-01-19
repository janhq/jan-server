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
	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/agent/planners"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/llm"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/response"
	"jan-server/services/response-api/internal/domain/skill"
	"jan-server/services/response-api/internal/domain/tool"
	"jan-server/services/response-api/internal/infrastructure/apikey"
	"jan-server/services/response-api/internal/infrastructure/auth"
	"jan-server/services/response-api/internal/infrastructure/database"
	"jan-server/services/response-api/internal/infrastructure/llmprovider"
	"jan-server/services/response-api/internal/infrastructure/logger"
	"jan-server/services/response-api/internal/infrastructure/mcp"
	"jan-server/services/response-api/internal/infrastructure/media"
	"jan-server/services/response-api/internal/infrastructure/observability"
	"jan-server/services/response-api/internal/infrastructure/queue"
	artifactrepo "jan-server/services/response-api/internal/infrastructure/repository/artifact"
	conversationrepo "jan-server/services/response-api/internal/infrastructure/repository/conversation"
	planrepo "jan-server/services/response-api/internal/infrastructure/repository/plan"
	respRepo "jan-server/services/response-api/internal/infrastructure/repository/response"
	skillinfra "jan-server/services/response-api/internal/infrastructure/skill"
	"jan-server/services/response-api/internal/interfaces/httpserver"
	"jan-server/services/response-api/internal/webhook"
	"jan-server/services/response-api/internal/worker"
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

	authValidator, err := auth.NewValidator(ctx, cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("initialize auth validator")
	}

	responseRepository := respRepo.NewPostgresRepository(db)
	conversationRepository := conversationrepo.NewRepository(db)
	conversationItemRepository := conversationrepo.NewItemRepository(db)
	planRepository := planrepo.NewPostgresRepository(db)
	artifactRepository := artifactrepo.NewPostgresRepository(db)
	llmClient := llmprovider.NewClient(cfg.LLMAPIURL)
	mcpClient := mcp.NewClient(cfg.MCPToolsURL)
	mediaClient := media.NewClient(cfg.MediaAPIURL)
	orchestrator := tool.NewOrchestrator(llmClient, mcpClient, cfg.MaxToolDepth, cfg.ToolTimeout)

	// Initialize webhook service
	webhookService := webhook.NewHTTPService(log)

	// Initialize plan service first (needed for agent registry)
	planService := plan.NewService(planRepository)

	// Initialize artifact service
	artifactService := artifact.NewService(artifactRepository)

	// Initialize skill service
	skillService, err := skillinfra.NewService()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize skill service")
	}

	// Initialize agent registry with planners and executors
	agentRegistry := agent.NewRegistry()
	deepResearchPlanner := planners.NewDeepResearchPlanner(planService)
	if err := agentRegistry.RegisterPlanner(deepResearchPlanner); err != nil {
		log.Warn().Err(err).Msg("failed to register deep research planner")
	}

	// Register slide generator planner
	slideGeneratorPlanner := planners.NewSlideGeneratorPlanner(planService, artifactService)
	if err := agentRegistry.RegisterPlanner(slideGeneratorPlanner); err != nil {
		log.Warn().Err(err).Msg("failed to register slide generator planner")
	}

	docGeneratorPlanner := planners.NewDocGeneratorPlanner(planService, artifactService)
	if err := agentRegistry.RegisterPlanner(docGeneratorPlanner); err != nil {
		log.Warn().Err(err).Msg("failed to register doc generator planner")
	}

	pdfGeneratorPlanner := planners.NewPDFGeneratorPlanner(planService, artifactService)
	if err := agentRegistry.RegisterPlanner(pdfGeneratorPlanner); err != nil {
		log.Warn().Err(err).Msg("failed to register pdf generator planner")
	}

	spreadsheetGeneratorPlanner := planners.NewSpreadsheetGeneratorPlanner(planService, artifactService)
	if err := agentRegistry.RegisterPlanner(spreadsheetGeneratorPlanner); err != nil {
		log.Warn().Err(err).Msg("failed to register spreadsheet generator planner")
	}

	// Create code fixer for LLM-based code fix retry
	codeFixer := llm.NewCodeFixer(llmClient, cfg.CodeFixModel)

	// Register executors for tool calls and LLM calls
	deepResearchExecutor := planners.NewDeepResearchExecutor(mcpClient, codeFixer)

	// Register slide generator executor for artifact creation
	skillExecutor := planners.NewSkillExecutor(
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
	slideGeneratorExecutor := planners.NewSlideGeneratorExecutor(mcpClient, codeFixer, artifactService, mediaClient, skillExecutor, cfg)
	routingExecutor := planners.NewRoutingExecutor(deepResearchExecutor, slideGeneratorExecutor)
	if err := agentRegistry.RegisterExecutor(plan.ActionTypeToolCall, routingExecutor); err != nil {
		log.Warn().Err(err).Msg("failed to register tool call executor")
	}
	if err := agentRegistry.RegisterExecutor(plan.ActionTypeLLMCall, routingExecutor); err != nil {
		log.Warn().Err(err).Msg("failed to register llm call executor")
	}
	if err := agentRegistry.RegisterExecutor(plan.ActionTypeArtifactCreate, slideGeneratorExecutor); err != nil {
		log.Warn().Err(err).Msg("failed to register artifact create executor")
	}
	if err := agentRegistry.RegisterExecutor(plan.ActionTypeSkillExecute, skillExecutor); err != nil {
		log.Warn().Err(err).Msg("failed to register skill execute executor")
	}

	agentOrchestrator := agent.NewOrchestrator(agentRegistry, planService)

	// Initialize response service with webhook support
	responseService := response.NewService(
		responseRepository,
		conversationRepository,
		conversationItemRepository,
		responseRepository,
		orchestrator,
		agentOrchestrator,
		mcpClient,
		mediaClient,
		llmClient, // Also implements ModelInfoProvider
		webhookService,
		agentRegistry,
		planService,
		log,
	)

	// Initialize background task infrastructure
	taskQueue := queue.NewPostgresQueue(db, log)
	workerPool := worker.NewPool(
		taskQueue,
		responseService,
		worker.Config{
			WorkerCount: cfg.BackgroundWorkerCount,
			TaskTimeout: cfg.BackgroundTaskTimeout,
		},
		log,
	)

	// Start worker pool
	workerPool.Start(ctx)
	defer func() {
		log.Info().Msg("stopping worker pool")
		workerPool.Stop()
	}()

	apiKeyProvider := apikey.NewClient(cfg.LLMAPIURL)
	httpServer := httpserver.New(cfg, log, responseService, planService, artifactService, apiKeyProvider, authValidator)
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
