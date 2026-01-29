package handlers

import (
	"github.com/google/wire"

	"jan-server/mono/apps/backend/internal/config"
	"jan-server/mono/apps/backend/internal/domain/usersettings"
	"jan-server/mono/apps/backend/internal/infrastructure/memory"
	adminhandler "jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/admin"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/apikeyhandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/chathandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/conversationhandler"
	guestauth "jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/guesthandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/modelhandler"
)

// ProvideMemoryHandler creates a memory handler with application config
func ProvideMemoryHandler(
	memoryClient *memory.Client,
	cfg *config.Config,
	userSettingsService *usersettings.Service,
) *chathandler.MemoryHandler {
	return chathandler.NewMemoryHandler(memoryClient, cfg.MemoryEnabled, userSettingsService)
}

var HandlerProvider = wire.NewSet(
	authhandler.NewAuthHandler,
	authhandler.NewTokenHandler,
	apikeyhandler.NewHandler,
	guestauth.NewGuestHandler,
	guestauth.NewUpgradeHandler,
	ProvideMemoryHandler,
	chathandler.NewChatHandler,
	conversationhandler.NewConversationHandler,
	modelhandler.NewModelHandler,
	modelhandler.NewProviderHandler,
	modelhandler.NewModelCatalogHandler,
	modelhandler.NewProviderModelHandler,
	adminhandler.NewAdminUserHandler,
	adminhandler.NewAdminGroupHandler,
	adminhandler.NewFeatureFlagHandler,
)
