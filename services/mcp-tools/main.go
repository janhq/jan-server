package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"

	sandboxdomain "jan-server/services/mcp-tools/internal/domain/sandbox"
	domainsearch "jan-server/services/mcp-tools/internal/domain/search"
	"jan-server/services/mcp-tools/internal/infrastructure/aio"
	"jan-server/services/mcp-tools/internal/infrastructure/auth"
	"jan-server/services/mcp-tools/internal/infrastructure/config"
	"jan-server/services/mcp-tools/internal/infrastructure/e2b"
	"jan-server/services/mcp-tools/internal/infrastructure/llmapi"
	"jan-server/services/mcp-tools/internal/infrastructure/logger"
	"jan-server/services/mcp-tools/internal/infrastructure/mcpprovider"
	"jan-server/services/mcp-tools/internal/infrastructure/metrics"
	sandboxfusionclient "jan-server/services/mcp-tools/internal/infrastructure/sandboxfusion"
	searchclient "jan-server/services/mcp-tools/internal/infrastructure/search"
	"jan-server/services/mcp-tools/internal/infrastructure/toolconfig"
	vectorstoreclient "jan-server/services/mcp-tools/internal/infrastructure/vectorstore"
	"jan-server/services/mcp-tools/internal/interfaces/httpserver/middlewares"
	"jan-server/services/mcp-tools/internal/interfaces/httpserver/routes/mcp"
)

func init() {
	// Initialize MCP metrics with a startup marker
	// This ensures metrics appear in Prometheus immediately
	metrics.SetCircuitBreakerState("serper", "closed")
	metrics.SetCircuitBreakerState("exa", "closed")
	metrics.SetCircuitBreakerState("tavily", "closed")
	metrics.SetCircuitBreakerState("searxng", "closed")
	metrics.SetCircuitBreakerState("sandboxfusion", "closed")
	metrics.SetCircuitBreakerState("memory-tools", "closed")
}

// @title Jan Server MCP Tools Service
// @version 1.0
// @description Model Context Protocol (MCP) tools service providing search and scraping capabilities.
// @contact.name Jan Server Team
// @contact.url https://github.com/janhq/jan-server
// @BasePath /

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Initialize logger
	logger.Init(cfg.LogLevel, cfg.LogFormat)
	log.Info().
		Str("http_port", cfg.HTTPPort).
		Str("log_level", cfg.LogLevel).
		Msg("Starting MCP Tools service")

	// Initialize infrastructure
	searchClient := searchclient.NewSearchClient(searchclient.ClientConfig{
		Engine:             searchclient.Engine(cfg.SearchEngine),
		SerperAPIKey:       cfg.SerperAPIKey,
		SerperEnabled:      cfg.SerperEnabled,
		SearxngURL:         cfg.SearxngURL,
		SearxngEnabled:     cfg.SearxngEnabled,
		DomainFilters:      cfg.SerperDomainFilter,
		LocationHint:       cfg.SerperLocationHint,
		OfflineMode:        cfg.SerperOfflineMode,
		ExaAPIKey:          cfg.ExaAPIKey,
		ExaEnabled:         cfg.ExaEnabled,
		ExaEndpoint:        cfg.ExaSearchEndpoint,
		ExaTimeout:         cfg.ExaTimeout,
		TavilyAPIKey:       cfg.TavilyAPIKey,
		TavilyEnabled:      cfg.TavilyEnabled,
		TavilyEndpoint:     cfg.TavilySearchEndpoint,
		TavilyTimeout:      cfg.TavilyTimeout,
		CBEnabled:          cfg.SearchCBEnabled,
		CBFailureThreshold: cfg.SerperCBFailureThreshold,
		CBSuccessThreshold: cfg.SerperCBSuccessThreshold,
		CBTimeout:          time.Duration(cfg.SerperCBTimeout) * time.Second,
		CBMaxHalfOpen:      cfg.SerperCBMaxHalfOpen,
		HTTPTimeout:        time.Duration(cfg.SerperHTTPTimeout) * time.Second,
		ScrapeTimeout:      time.Duration(cfg.SerperScrapeTimeout) * time.Second,
		MaxConnsPerHost:    cfg.SerperMaxConnsPerHost,
		MaxIdleConns:       cfg.SerperMaxIdleConns,
		IdleConnTimeout:    time.Duration(cfg.SerperIdleConnTimeout) * time.Second,
		RetryMaxAttempts:   cfg.SerperRetryMaxAttempts,
		RetryInitialDelay:  time.Duration(cfg.SerperRetryInitialDelay) * time.Millisecond,
		RetryMaxDelay:      time.Duration(cfg.SerperRetryMaxDelay) * time.Millisecond,
		RetryBackoffFactor: cfg.SerperRetryBackoffFactor,
	})
	searchService := domainsearch.NewSearchService(searchClient)

	var vectorClient *vectorstoreclient.Client
	if cfg.VectorStoreURL != "" {
		vectorClient = vectorstoreclient.NewClient(cfg.VectorStoreURL)
	}
	var sandboxMCP *mcp.SandboxFusionMCP
	switch {
	case !cfg.EnablePythonExec:
		log.Warn().Msg("SandboxFusion python_exec tool disabled via config")
	case cfg.SandboxFusionURL != "":
		sandboxClient := sandboxfusionclient.NewClient(cfg.SandboxFusionURL)
		sandboxMCP = mcp.NewSandboxFusionMCP(sandboxClient, cfg.SandboxFusionRequireApproval, cfg.EnablePythonExec)
	default:
		log.Warn().Msg("SandboxFusion URL not configured, python_exec tool will not be available")
	}

	// Load MCP provider configuration
	providerConfig, err := mcpprovider.LoadConfig("configs/mcp-providers.yml")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load MCP provider config, continuing without external providers")
		providerConfig = &mcpprovider.Config{} // Empty config
	}

	// Initialize MCP routes
	searchMCP := mcp.NewSearchMCP(searchService, vectorClient, mcp.SearchMCPConfig{
		MaxSnippetChars:       cfg.MaxSnippetChars,
		MaxScrapePreviewChars: cfg.MaxScrapePreviewChars,
		MaxScrapeTextChars:    cfg.MaxScrapeTextChars,
		EnableFileSearch:      cfg.EnableFileSearch,
	})

	// Initialize memory MCP
	var memoryMCP *mcp.MemoryMCP
	switch {
	case !cfg.EnableMemoryRetrieve:
		log.Warn().Msg("memory_retrieve MCP tool disabled via config")
	case cfg.MemoryToolsURL != "":
		memoryMCP = mcp.NewMemoryMCP(cfg.MemoryToolsURL, cfg.EnableMemoryRetrieve)
		log.Info().Str("url", cfg.MemoryToolsURL).Msg("Memory tools integration enabled")
	default:
		log.Warn().Msg("Memory tools URL not configured, memory_retrieve tool will not be available")
	}

	// Initialize image generation MCP
	var imageMCP *mcp.ImageGenerateMCP
	switch {
	case !cfg.EnableImageGenerate:
		log.Warn().Msg("generate_image MCP tool disabled via config")
	case cfg.LLMAPIBaseURL != "":
		imageMCP = mcp.NewImageGenerateMCP(cfg.LLMAPIBaseURL, cfg.EnableImageGenerate)
		log.Info().Str("llm_api_url", cfg.LLMAPIBaseURL).Msg("Image generation MCP tool enabled")
	default:
		log.Warn().Msg("LLM_API_BASE_URL not configured, generate_image tool will not be available")
	}

	// Initialize image edit MCP
	var imageEditMCP *mcp.ImageEditMCP
	switch {
	case !cfg.EnableImageEdit:
		log.Warn().Msg("edit_image MCP tool disabled via config")
	case cfg.LLMAPIBaseURL != "":
		imageEditMCP = mcp.NewImageEditMCP(cfg.LLMAPIBaseURL, cfg.EnableImageEdit)
		log.Info().Str("llm_api_url", cfg.LLMAPIBaseURL).Msg("Image edit MCP tool enabled")
	default:
		log.Warn().Msg("LLM_API_BASE_URL not configured, edit_image tool will not be available")
	}

	// Initialize external MCP providers
	ctx := context.Background()
	providerMCP := mcp.NewProviderMCP(providerConfig)
	if err := providerMCP.Initialize(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to initialize MCP providers")
	}

	// Initialize LLM-API client for tool tracking
	var llmClient *llmapi.Client
	if cfg.MCPTrackingEnabled && cfg.LLMAPIBaseURL != "" {
		llmClient = llmapi.NewClient(cfg.LLMAPIBaseURL)
		log.Info().
			Str("llm_api_url", cfg.LLMAPIBaseURL).
			Msg("LLM-API client initialized for tool tracking")
	} else {
		log.Info().
			Bool("tracking_enabled", cfg.MCPTrackingEnabled).
			Str("llm_api_url", cfg.LLMAPIBaseURL).
			Msg("LLM-API tool tracking disabled or not configured")
	}

	// Initialize tool config cache for dynamic descriptions
	var toolConfigCache *toolconfig.Cache
	if cfg.LLMAPIBaseURL != "" {
		// Create a client specifically for tool config if not already created
		configClient := llmClient
		if configClient == nil {
			configClient = llmapi.NewClient(cfg.LLMAPIBaseURL)
		}
		toolConfigCache = toolconfig.NewCache(configClient)
		log.Info().Msg("Tool config cache initialized for dynamic descriptions")
	}

	// Set tool config cache on searchMCP
	if toolConfigCache != nil {
		searchMCP.SetToolConfigCache(toolConfigCache)
	}

	// Initialize sandbox provider (mutually exclusive: AIO or E2B)
	var sandboxProviderImpl sandboxdomain.Provider
	var sandboxManager sandboxdomain.Manager
	var unifiedSandboxMCP *mcp.SandboxMCP
	var sandboxManagement *mcp.SandboxManagement

	switch cfg.SandboxProvider {
	case "aio":
		aioClient := aio.NewClient(aio.ClientConfig{
			BaseURL: cfg.AIOURL,
			Timeout: cfg.AIOTimeout,
			Enabled: true,
		})
		if aioClient.IsEnabled() {
			sandboxProviderImpl = aioClient
			// AIO does not support Manager interface (no lifecycle management)
			log.Info().
				Str("provider", "aio").
				Str("url", cfg.AIOURL).
				Dur("timeout", cfg.AIOTimeout).
				Msg("Sandbox provider initialized")
		} else {
			log.Warn().Msg("SANDBOX_PROVIDER=aio but client failed to initialize")
		}

	case "e2b":
		e2bClient := e2b.NewClient(e2b.ClientConfig{
			BaseURL: cfg.E2BServiceURL,
			Timeout: cfg.E2BTimeout,
			Enabled: true,
		})
		if e2bClient.IsEnabled() {
			sandboxProviderImpl = e2bClient
			sandboxManager = e2bClient // E2B client implements Manager interface
			log.Info().
				Str("provider", "e2b").
				Str("url", cfg.E2BServiceURL).
				Dur("timeout", cfg.E2BTimeout).
				Msg("Sandbox provider initialized (with lifecycle management)")
		} else {
			log.Warn().Msg("SANDBOX_PROVIDER=e2b but client failed to initialize")
		}

	case "":
		log.Info().Msg("Sandbox provider disabled (SANDBOX_PROVIDER not set)")

	default:
		log.Warn().Str("provider", cfg.SandboxProvider).Msg("Unknown sandbox provider")
	}

	if sandboxProviderImpl != nil {
		unifiedSandboxMCP = mcp.NewSandboxMCP(sandboxProviderImpl)
	}

	// Create sandbox management (E2B only)
	if sandboxManager != nil {
		sandboxManagement = mcp.NewSandboxManagement(sandboxManager)
		log.Info().Msg("Sandbox management tools enabled (E2B)")
	}

	// Initialize Agent Proxy MCP handler
	var agentProxyMCP *mcp.AgentProxyMCP
	if cfg.AgentProxyEnabled {
		agentProxyMCP = mcp.NewAgentProxyMCP(cfg.ResponseAPIURL, cfg.LLMAPIBaseURL, true)
		if agentProxyMCP != nil {
			log.Info().
				Str("response_api_url", cfg.ResponseAPIURL).
				Str("llm_api_url", cfg.LLMAPIBaseURL).
				Msg("Agent proxy integration enabled")
		} else {
			log.Warn().Msg("MCP_AGENT_PROXY_ENABLED=true but client failed to initialize")
		}
	} else {
		log.Info().Msg("Agent proxy integration disabled (MCP_AGENT_PROXY_ENABLED=false)")
	}

	mcpRoute := mcp.NewMCPRoute(
		searchMCP,
		providerMCP,
		sandboxMCP, // SandboxFusionMCP for python_exec
		memoryMCP,
		imageMCP,
		imageEditMCP,
		unifiedSandboxMCP, // Unified sandbox provider (AIO or E2B)
		sandboxManagement, // Sandbox lifecycle management (E2B only)
		agentProxyMCP,
		llmClient,
		toolConfigCache,
		sandboxManager,      // Sandbox manager for state checks (E2B only)
		cfg.SandboxProvider, // "aio", "e2b", or ""
	)

	authValidator, err := auth.NewValidator(ctx, cfg, log.Logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize auth validator")
	}

	// Setup HTTP server
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middlewares.RequestLogger())
	router.Use(middlewares.CORS())
	router.Use(middlewares.MetricsRecorder())

	// Apply auth middleware (will skip health checks internally)
	if authValidator != nil {
		router.Use(authValidator.Middleware())
	}

	// Health check endpoints
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "mcp-tools"})
	})

	router.GET("/readyz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ready", "service": "mcp-tools"})
	})

	router.GET("/health/auth", func(c *gin.Context) {
		if authValidator == nil || authValidator.Ready() {
			c.JSON(200, gin.H{"status": "ready"})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "initializing"})
	})

	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Register MCP routes
	v1 := router.Group("/v1")
	mcpRoute.RegisterRouter(v1) // Start server
	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	log.Info().Str("address", addr).Msg("Server listening")

	if err := router.Run(addr); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
