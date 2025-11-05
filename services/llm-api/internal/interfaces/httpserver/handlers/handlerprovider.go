package handlers

import (
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/chathandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/conversationhandler"
	guestauth "jan-server/services/llm-api/internal/interfaces/httpserver/handlers/guesthandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/modelhandler"

	"github.com/google/wire"
)

var HandlerProvider = wire.NewSet(
	authhandler.NewAuthHandler,
	authhandler.NewTokenHandler,
	guestauth.NewGuestHandler,
	guestauth.NewUpgradeHandler,
	chathandler.NewChatHandler,
	conversationhandler.NewConversationHandler,
	modelhandler.NewModelHandler,
	modelhandler.NewProviderHandler,
	modelhandler.NewModelCatalogHandler,
	modelhandler.NewProviderModelHandler,
)
