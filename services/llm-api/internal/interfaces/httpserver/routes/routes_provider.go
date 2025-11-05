package routes

import (
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/chathandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/conversationhandler"
	guestauth "jan-server/services/llm-api/internal/interfaces/httpserver/handlers/guesthandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/modelhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/auth"
	v1 "jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/admin"
	adminModel "jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/admin/model"
	adminProvider "jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/admin/provider"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/chat"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/conversation"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/model"
	modelProvider "jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/model/provider"

	"github.com/google/wire"
)

var RouteProvider = wire.NewSet(
	// Handlers
	authhandler.NewAuthHandler,
	authhandler.NewTokenHandler,
	chathandler.NewChatHandler,
	conversationhandler.NewConversationHandler,
	guestauth.NewGuestHandler,
	guestauth.NewUpgradeHandler,
	modelhandler.NewProviderHandler,
	modelhandler.NewModelHandler,
	modelhandler.NewModelCatalogHandler,
	modelhandler.NewProviderModelHandler,

	// Routes
	auth.NewAuthRoute,
	v1.NewV1Route,
	admin.NewAdminRoute,
	adminModel.NewAdminModelRoute,
	adminProvider.NewAdminProviderRoute,
	chat.NewChatRoute,
	chat.NewChatCompletionRoute,
	conversation.NewConversationRoute,
	model.NewModelRoute,
	modelProvider.NewModelProviderRoute,
)
