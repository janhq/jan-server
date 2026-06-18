package handlers

import (
	"github.com/google/wire"

	adminhandler "jan-server/services/llm-api/internal/interfaces/httpserver/handlers/admin"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/apikeyhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/chathandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/conversationhandler"
	guestauth "jan-server/services/llm-api/internal/interfaces/httpserver/handlers/guesthandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/modelhandler"
)

var HandlerProvider = wire.NewSet(
	authhandler.NewAuthHandler,
	authhandler.NewTokenHandler,
	apikeyhandler.NewHandler,
	guestauth.NewGuestHandler,
	guestauth.NewUpgradeHandler,
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
