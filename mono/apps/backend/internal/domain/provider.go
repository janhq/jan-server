package domain

import (
	"github.com/google/wire"
	"github.com/rs/zerolog"

	"jan-server/mono/apps/backend/internal/config"
	"jan-server/mono/apps/backend/internal/domain/apikey"
	"jan-server/mono/apps/backend/internal/domain/conversation"
	"jan-server/mono/apps/backend/internal/domain/document"
	"jan-server/mono/apps/backend/internal/domain/mcptool"
	"jan-server/mono/apps/backend/internal/domain/model"
	"jan-server/mono/apps/backend/internal/domain/modelprompttemplate"
	"jan-server/mono/apps/backend/internal/domain/project"
	"jan-server/mono/apps/backend/internal/domain/prompt"
	"jan-server/mono/apps/backend/internal/domain/prompttemplate"
	"jan-server/mono/apps/backend/internal/domain/share"
	"jan-server/mono/apps/backend/internal/domain/tokenusage"
	"jan-server/mono/apps/backend/internal/domain/user"
	"jan-server/mono/apps/backend/internal/domain/usersettings"
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
	ProvideUserServiceConfig,
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

	// Document services
	document.NewDocumentService,
	document.NewProjectFileService,

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

// ProvideUserServiceConfig creates the configuration for the local user service.
func ProvideUserServiceConfig(cfg *config.Config) user.ServiceConfig {
	return user.ServiceConfig{
		JWTSecret:        cfg.LocalJWTSecret,
		JWTIssuer:        cfg.LocalJWTIssuer,
		JWTExpiration:    cfg.LocalJWTExpiration,
		RefreshTokenTTL:  cfg.LocalRefreshTokenTTL,
		BcryptCost:       cfg.LocalBcryptCost,
		APIKeyPrefix:     cfg.APIKeyPrefix,
		APIKeyMaxPerUser: cfg.APIKeyMaxPerUser,
		APIKeyDefaultTTL: cfg.APIKeyDefaultTTL,
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
	projectFileService *document.ProjectFileService,
) *prompt.ProcessorImpl {
	processor := prompt.NewProcessorWithServices(config, log, templateService, modelPromptService)

	// Register Project Files module if prompt orchestration is enabled
	if config.Enabled && projectFileService != nil {
		processor.RegisterModule(prompt.NewProjectFilesModule(projectFileService))
		log.Info().Msg("registered Project Files prompt module")
	}

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
