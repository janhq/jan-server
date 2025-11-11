package domain

import (
	"github.com/google/wire"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/domain/apikey"
	"jan-server/services/llm-api/internal/domain/conversation"
	"jan-server/services/llm-api/internal/domain/model"
	"jan-server/services/llm-api/internal/domain/project"
	"jan-server/services/llm-api/internal/domain/user"
)

// ServiceProvider provides all domain services
var ServiceProvider = wire.NewSet(
	// Conversation domain
	conversation.NewConversationService,

	// Project domain
	project.NewProjectService,

	// Model domain
	model.NewProviderModelService,
	model.NewModelCatalogService,
	model.NewProviderService,

	// User domain
	user.NewService,

	// API keys
	ProvideAPIKeyConfig,
	apikey.NewService,
)

func ProvideAPIKeyConfig(cfg *config.Config) apikey.Config {
	return apikey.Config{
		DefaultTTL: cfg.APIKeyDefaultTTL,
		MaxTTL:     cfg.APIKeyMaxTTL,
		MaxPerUser: cfg.APIKeyMaxPerUser,
		KeyPrefix:  cfg.APIKeyPrefix,
	}
}
