package routes

import (
	"github.com/google/wire"
	"github.com/rs/zerolog/log"

	"jan-server/services/mcp-tools/internal/infrastructure/aio"
	"jan-server/services/mcp-tools/internal/infrastructure/config"
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
	ProvideAIOMCP,
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

// ProvideAIOMCP creates an AIOMCP handler if configured
func ProvideAIOMCP(cfg *config.Config) *mcp.AIOMCP {
	if !cfg.AIOEnabled {
		log.Info().Msg("AIO Sandbox integration disabled (AIO_ENABLED=false)")
		return nil
	}
	aioClient := aio.NewClient(aio.ClientConfig{
		BaseURL: cfg.AIOURL,
		Timeout: cfg.AIOTimeout,
		Enabled: cfg.AIOEnabled,
	})
	if !aioClient.IsEnabled() {
		log.Warn().Msg("AIO_ENABLED=true but client failed to initialize")
		return nil
	}
	log.Info().
		Str("aio_url", cfg.AIOURL).
		Dur("aio_timeout", cfg.AIOTimeout).
		Msg("AIO Sandbox integration enabled")
	return mcp.NewAIOMCP(aioClient, true)
}

// ProvideMCPRoute creates a MCPRoute with all dependencies
func ProvideMCPRoute(
	searchMCP *mcp.SearchMCP,
	providerMCP *mcp.ProviderMCP,
	sandboxMCP *mcp.SandboxFusionMCP,
	memoryMCP *mcp.MemoryMCP,
	imageMCP *mcp.ImageGenerateMCP,
	imageEditMCP *mcp.ImageEditMCP,
	aioMCP *mcp.AIOMCP,
	llmClient *llmapi.Client,
	toolConfigCache *toolconfig.Cache,
) *mcp.MCPRoute {
	// Set tool config cache on searchMCP for dynamic descriptions
	if toolConfigCache != nil {
		searchMCP.SetToolConfigCache(toolConfigCache)
	}
	return mcp.NewMCPRoute(searchMCP, providerMCP, sandboxMCP, memoryMCP, imageMCP, imageEditMCP, aioMCP, llmClient, toolConfigCache)
}
