package routes

import (
	"github.com/google/wire"
	"github.com/rs/zerolog/log"

	sandboxdomain "jan-server/services/mcp-tools/internal/domain/sandbox"
	"jan-server/services/mcp-tools/internal/infrastructure/aio"
	"jan-server/services/mcp-tools/internal/infrastructure/config"
	"jan-server/services/mcp-tools/internal/infrastructure/e2b"
	"jan-server/services/mcp-tools/internal/infrastructure/llmapi"
	sandboxfusionclient "jan-server/services/mcp-tools/internal/infrastructure/sandboxfusion"
	"jan-server/services/mcp-tools/internal/infrastructure/toolconfig"
	"jan-server/services/mcp-tools/internal/interfaces/httpserver/routes/mcp"
)

// RoutesProvider provides all route dependencies
var RoutesProvider = wire.NewSet(
	mcp.NewSearchMCP,
	mcp.NewProviderMCP,
	ProvideSandboxFusionMCP,
	ProvideMemoryMCP,
	ProvideImageGenerateMCP,
	ProvideImageEditMCP,
	ProvideE2BClient, // Shared E2B client (used by both Provider and Manager)
	ProvideSandboxMCP,
	ProvideSandboxManagement,
	ProvideSandboxManager,
	ProvideAgentProxyMCP,
	ProvideToolConfigCache,
	ProvideMCPRoute,
	ProvideSearchMCPConfig,
)

// ProvideSearchMCPConfig creates a SearchMCPConfig from the main config
func ProvideSearchMCPConfig(cfg *config.Config) mcp.SearchMCPConfig {
	return mcp.SearchMCPConfig{
		EnableFileSearch: cfg.EnableFileSearch,
	}
}

// ProvideSandboxFusionMCP creates a SandboxFusionMCP if configured
func ProvideSandboxFusionMCP(
	client *sandboxfusionclient.Client,
	cfg *config.Config,
) *mcp.SandboxFusionMCP {
	if !cfg.EnablePythonExec {
		log.Warn().Msg("SandboxFusion python_exec tool disabled via config")
		return nil
	}
	if client == nil {
		return nil
	}
	return mcp.NewSandboxFusionMCP(client, cfg.SandboxFusionRequireApproval, cfg.EnablePythonExec)
}

// ProvideMemoryMCP creates a MemoryMCP if configured
func ProvideMemoryMCP(cfg *config.Config) *mcp.MemoryMCP {
	if !cfg.EnableMemoryRetrieve {
		log.Warn().Msg("memory_retrieve MCP tool disabled via config")
		return nil
	}
	if cfg.MemoryToolsURL == "" {
		return nil
	}
	return mcp.NewMemoryMCP(cfg.MemoryToolsURL, cfg.EnableMemoryRetrieve)
}

// ProvideImageGenerateMCP creates an ImageGenerateMCP if configured
func ProvideImageGenerateMCP(cfg *config.Config) *mcp.ImageGenerateMCP {
	if !cfg.EnableImageGenerate {
		log.Warn().Msg("generate_image MCP tool disabled via config")
		return nil
	}
	if cfg.LLMAPIBaseURL == "" {
		log.Warn().Msg("LLM_API_BASE_URL not configured; skipping generate_image tool registration")
		return nil
	}
	return mcp.NewImageGenerateMCP(cfg.LLMAPIBaseURL, cfg.EnableImageGenerate)
}

// ProvideImageEditMCP creates an ImageEditMCP if configured
func ProvideImageEditMCP(cfg *config.Config) *mcp.ImageEditMCP {
	if !cfg.EnableImageEdit {
		log.Warn().Msg("edit_image MCP tool disabled via config")
		return nil
	}
	if cfg.LLMAPIBaseURL == "" {
		log.Warn().Msg("LLM_API_BASE_URL not configured; skipping edit_image tool registration")
		return nil
	}
	return mcp.NewImageEditMCP(cfg.LLMAPIBaseURL, cfg.EnableImageEdit)
}

// ProvideToolConfigCache creates a tool config cache if LLM-API is configured
func ProvideToolConfigCache(cfg *config.Config, llmClient *llmapi.Client) *toolconfig.Cache {
	if cfg.LLMAPIBaseURL == "" {
		log.Warn().Msg("Tool config cache disabled - LLM-API base URL not configured")
		return nil
	}
	if llmClient == nil {
		llmClient = llmapi.NewClient(cfg.LLMAPIBaseURL)
	}
	log.Info().Msg("Tool config cache initialized for dynamic descriptions")
	return toolconfig.NewCache(llmClient)
}

// ProvideE2BClient creates a shared E2B client when SANDBOX_PROVIDER=e2b.
// This client is used by both SandboxMCP (Provider) and SandboxManagement (Manager).
func ProvideE2BClient(cfg *config.Config) *e2b.Client {
	if cfg.SandboxProvider != "e2b" {
		return nil
	}

	client := e2b.NewClient(e2b.ClientConfig{
		BaseURL: cfg.E2BServiceURL,
		Timeout: cfg.E2BTimeout,
		Enabled: true,
	})
	if !client.IsEnabled() {
		log.Warn().Msg("E2B client failed to initialize")
		return nil
	}
	log.Info().
		Str("provider", "e2b").
		Str("url", cfg.E2BServiceURL).
		Dur("timeout", cfg.E2BTimeout).
		Msg("E2B client initialized (shared)")
	return client
}

// ProvideSandboxMCP creates a SandboxMCP handler with the appropriate provider.
// Uses the shared E2B client when provider is e2b.
func ProvideSandboxMCP(cfg *config.Config, e2bClient *e2b.Client) *mcp.SandboxMCP {
	var sandboxProvider sandboxdomain.Provider

	switch cfg.SandboxProvider {
	case "aio":
		aioClient := aio.NewClient(aio.ClientConfig{
			BaseURL: cfg.AIOURL,
			Timeout: cfg.AIOTimeout,
			Enabled: true,
		})
		if aioClient.IsEnabled() {
			sandboxProvider = aioClient
			log.Info().
				Str("provider", "aio").
				Str("url", cfg.AIOURL).
				Dur("timeout", cfg.AIOTimeout).
				Msg("Sandbox provider initialized")
		} else {
			log.Warn().Msg("SANDBOX_PROVIDER=aio but client failed to initialize")
			return nil
		}

	case "e2b":
		if e2bClient == nil || !e2bClient.IsEnabled() {
			log.Warn().Msg("SANDBOX_PROVIDER=e2b but E2B client not available")
			return nil
		}
		sandboxProvider = e2bClient

	case "":
		log.Info().Msg("Sandbox provider disabled (SANDBOX_PROVIDER not set)")
		return nil

	default:
		log.Warn().Str("provider", cfg.SandboxProvider).Msg("Unknown sandbox provider")
		return nil
	}

	return mcp.NewSandboxMCP(sandboxProvider)
}

// ProvideSandboxManagement creates a SandboxManagement handler (E2B only)
func ProvideSandboxManagement(cfg *config.Config, manager sandboxdomain.Manager) *mcp.SandboxManagement {
	if cfg.SandboxProvider != "e2b" {
		log.Debug().Msg("Sandbox management disabled (only available for E2B provider)")
		return nil
	}
	if manager == nil {
		log.Warn().Msg("Sandbox management disabled (manager not initialized)")
		return nil
	}
	log.Info().Msg("Sandbox management tools enabled (E2B)")
	return mcp.NewSandboxManagement(manager)
}

// ProvideSandboxManager returns the E2B client as Manager when SANDBOX_PROVIDER=e2b.
// Uses the shared E2B client.
func ProvideSandboxManager(cfg *config.Config, e2bClient *e2b.Client) sandboxdomain.Manager {
	if cfg.SandboxProvider != "e2b" {
		return nil
	}
	if e2bClient == nil || !e2bClient.IsEnabled() {
		log.Warn().Msg("Sandbox manager disabled (E2B client not available)")
		return nil
	}
	return e2bClient
}

// ProvideAgentProxyMCP creates an AgentProxyMCP with dependencies
func ProvideAgentProxyMCP(cfg *config.Config) *mcp.AgentProxyMCP {
	if !cfg.AgentProxyEnabled {
		log.Info().Msg("Agent proxy integration disabled (MCP_AGENT_PROXY_ENABLED=false)")
		return nil
	}
	agentProxy := mcp.NewAgentProxyMCP(cfg.ResponseAPIURL, cfg.LLMAPIBaseURL, true)
	if agentProxy == nil {
		log.Warn().Msg("MCP_AGENT_PROXY_ENABLED=true but client failed to initialize")
		return nil
	}
	log.Info().
		Str("response_api_url", cfg.ResponseAPIURL).
		Str("llm_api_url", cfg.LLMAPIBaseURL).
		Msg("Agent proxy integration enabled")
	return agentProxy
}

// ProvideMCPRoute creates a MCPRoute with all dependencies
func ProvideMCPRoute(
	cfg *config.Config,
	searchMCP *mcp.SearchMCP,
	providerMCP *mcp.ProviderMCP,
	sandboxFusionMCP *mcp.SandboxFusionMCP,
	memoryMCP *mcp.MemoryMCP,
	imageMCP *mcp.ImageGenerateMCP,
	imageEditMCP *mcp.ImageEditMCP,
	sandboxMCP *mcp.SandboxMCP,
	sandboxManagement *mcp.SandboxManagement,
	agentProxyMCP *mcp.AgentProxyMCP,
	llmClient *llmapi.Client,
	toolConfigCache *toolconfig.Cache,
	sandboxManager sandboxdomain.Manager,
) *mcp.MCPRoute {
	// Set tool config cache on searchMCP for dynamic descriptions
	if toolConfigCache != nil {
		searchMCP.SetToolConfigCache(toolConfigCache)
	}
	return mcp.NewMCPRoute(
		searchMCP,
		providerMCP,
		sandboxFusionMCP,
		memoryMCP,
		imageMCP,
		imageEditMCP,
		sandboxMCP,
		sandboxManagement,
		agentProxyMCP,
		llmClient,
		toolConfigCache,
		sandboxManager,
		cfg.SandboxProvider,
	)
}
