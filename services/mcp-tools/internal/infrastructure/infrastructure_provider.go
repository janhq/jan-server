package infrastructure

import (
	"context"

	"github.com/google/wire"
	"github.com/rs/zerolog/log"

	"github.com/janhq/jan-server/packages/go-common/analytics"

	"jan-server/services/mcp-tools/internal/domain/search"
	"jan-server/services/mcp-tools/internal/infrastructure/auth"
	"jan-server/services/mcp-tools/internal/infrastructure/config"
	"jan-server/services/mcp-tools/internal/infrastructure/llmapi"
	"jan-server/services/mcp-tools/internal/infrastructure/mcpprovider"
	sandboxfusionclient "jan-server/services/mcp-tools/internal/infrastructure/sandboxfusion"
	searchclient "jan-server/services/mcp-tools/internal/infrastructure/search"
	vectorstoreclient "jan-server/services/mcp-tools/internal/infrastructure/vectorstore"
)

// InfrastructureProvider provides all infrastructure dependencies
var InfrastructureProvider = wire.NewSet(
	// Config
	ProvideConfig,

	// Search client
	ProvideSearchClient,

	// Vector store client
	ProvideVectorStoreClient,

	// Sandbox Fusion client
	ProvideSandboxFusionClient,

	// MCP Provider config
	ProvideMCPProviderConfig,

	// Auth validator
	ProvideAuthValidator,

	// LLM-API client for tool tracking
	ProvideLLMAPIClient,

	// Analytics
	ProvideAnalyticsTracker,
)

// ProvideConfig loads and provides the application configuration
func ProvideConfig() (*config.Config, error) {
	return config.LoadConfig()
}

// ProvideSearchClient provides the search client
func ProvideSearchClient(cfg *config.Config) search.SearchClient {
	return searchclient.NewSearchClient(searchclient.ClientConfig{
		Engine:        searchclient.Engine(cfg.SearchEngine),
		SerperAPIKey:  cfg.SerperAPIKey,
		SearxngURL:    cfg.SearxngURL,
		DomainFilters: cfg.SerperDomainFilter,
		LocationHint:  cfg.SerperLocationHint,
		OfflineMode:   cfg.SerperOfflineMode,
		CBEnabled:     cfg.SearchCBEnabled,
	})
}

// ProvideVectorStoreClient provides the vector store client
func ProvideVectorStoreClient(cfg *config.Config) *vectorstoreclient.Client {
	if cfg.VectorStoreURL == "" {
		return nil
	}
	return vectorstoreclient.NewClient(cfg.VectorStoreURL)
}

// ProvideSandboxFusionClient provides the sandbox fusion client
func ProvideSandboxFusionClient(cfg *config.Config) *sandboxfusionclient.Client {
	if cfg.SandboxFusionURL == "" {
		return nil
	}
	return sandboxfusionclient.NewClient(cfg.SandboxFusionURL)
}

// ProvideMCPProviderConfig loads the MCP provider configuration
func ProvideMCPProviderConfig() *mcpprovider.Config {
	providerConfig, err := mcpprovider.LoadConfig("configs/mcp-providers.yml")
	if err != nil {
		// Return empty config if file not found
		return &mcpprovider.Config{}
	}
	return providerConfig
}

// ProvideAuthValidator provides the auth validator
func ProvideAuthValidator(ctx context.Context, cfg *config.Config) (*auth.Validator, error) {
	// Get global logger from zerolog
	logger := log.Logger
	return auth.NewValidator(ctx, cfg, logger)
}

// ProvideLLMAPIClient provides the LLM-API client for tool call tracking
func ProvideLLMAPIClient(cfg *config.Config) *llmapi.Client {
	if !cfg.MCPTrackingEnabled || cfg.LLMAPIBaseURL == "" {
		log.Info().
			Bool("tracking_enabled", cfg.MCPTrackingEnabled).
			Str("llm_api_url", cfg.LLMAPIBaseURL).
			Msg("LLM-API client disabled or not configured")
		return nil
	}
	log.Info().
		Str("llm_api_url", cfg.LLMAPIBaseURL).
		Msg("LLM-API client initialized for tool tracking")
	return llmapi.NewClient(cfg.LLMAPIBaseURL)
}

// ProvideAnalyticsTracker creates the analytics tracker from config
func ProvideAnalyticsTracker(cfg *config.Config) analytics.Tracker {
	logger := log.Logger

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
		logger.Warn().Err(err).Msg("Failed to create analytics tracker, using no-op")
		return analytics.NewNoopTracker()
	}

	logger.Info().
		Bool("enabled", analyticsCfg.Enabled).
		Bool("posthog", analyticsCfg.PostHog.Enabled).
		Bool("otel", analyticsCfg.OTel.Enabled).
		Str("environment", analyticsCfg.Environment).
		Msg("Analytics tracker initialized")

	return tracker
}
