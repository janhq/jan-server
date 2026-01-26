package domain

import (
	"github.com/google/wire"
	"github.com/rs/zerolog"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/domain/apikey"
	"jan-server/services/llm-api/internal/domain/conversation"
	"jan-server/services/llm-api/internal/domain/mcptool"
	"jan-server/services/llm-api/internal/domain/model"
	"jan-server/services/llm-api/internal/domain/modelprompttemplate"
	"jan-server/services/llm-api/internal/domain/project"
	"jan-server/services/llm-api/internal/domain/prompt"
	"jan-server/services/llm-api/internal/domain/prompttemplate"
	"jan-server/services/llm-api/internal/domain/share"
	"jan-server/services/llm-api/internal/domain/tokenusage"
	"jan-server/services/llm-api/internal/domain/user"
	"jan-server/services/llm-api/internal/domain/usersettings"
)

// ServiceProvider provides all domain services
var ServiceProvider = wire.NewSet(
	// Conversation domain
	conversation.NewConversationService,
	conversation.NewMessageActionService,

	// Project domain
	project.NewProjectService,

	// Model domain
	model.NewProviderModelService,
	model.NewModelCatalogService,
	model.NewProviderService,

	// User domain
	user.NewService,

	// User settings
	usersettings.NewService,

	// API keys
	ProvideAPIKeyConfig,
	apikey.NewService,

	// Prompt templates
	prompttemplate.NewService,

	// Model prompt templates
	modelprompttemplate.NewService,

	// MCP tools
	mcptool.NewService,

	// Prompt orchestration
	ProvidePromptProcessorConfig,
	ProvidePromptProcessor,

	// Share domain
	share.NewShareService,

	// Token usage
	tokenusage.NewService,
)

func ProvideAPIKeyConfig(cfg *config.Config) apikey.Config {
	return apikey.Config{
		DefaultTTL: cfg.APIKeyDefaultTTL,
		MaxTTL:     cfg.APIKeyMaxTTL,
		MaxPerUser: cfg.APIKeyMaxPerUser,
		KeyPrefix:  cfg.APIKeyPrefix,
	}
}

func ProvidePromptProcessorConfig(cfg *config.Config, log zerolog.Logger) prompt.ProcessorConfig {
	return prompt.ProcessorConfig{
		Enabled:         cfg.PromptOrchestrationEnabled,
		EnableMemory:    cfg.PromptOrchestrationEnableMemory,
		EnableTemplates: cfg.PromptOrchestrationEnableTemplates,
		EnableTools:     cfg.PromptOrchestrationEnableTools,
	}
}

// ProvidePromptProcessor creates the prompt processor with all modules including Deep Research
func ProvidePromptProcessor(
	config prompt.ProcessorConfig,
	log zerolog.Logger,
	templateService *prompttemplate.Service,
	modelPromptService *modelprompttemplate.Service,
) *prompt.ProcessorImpl {
	processor := prompt.NewProcessorWithServices(config, log, templateService, modelPromptService)

	// Register Deep Research module if prompt orchestration is enabled
	if config.Enabled && templateService != nil {
		// Use model-aware Deep Research module if model prompt service is available
		if modelPromptService != nil {
			processor.RegisterModule(prompt.NewDeepResearchModuleWithModelPrompts(templateService, modelPromptService))
			log.Info().Msg("registered Deep Research prompt module with model-specific template support")
		} else {
			processor.RegisterModule(prompt.NewDeepResearchModule(templateService))
			log.Info().Msg("registered Deep Research prompt module")
		}
	}

	return processor
}
