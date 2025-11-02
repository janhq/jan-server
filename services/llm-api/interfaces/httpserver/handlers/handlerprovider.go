package handlers

import (
	"github.com/google/wire"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/handlers/accesspolicyhandler"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/handlers/authhandler"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/handlers/chathandler"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/handlers/conversationhandler"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/handlers/management/apikeyhandler"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/handlers/modelhandler"
)

var HandlerProvider = wire.NewSet(
	authhandler.NewAuthHandler,
	apikeyhandler.NewAPIKeyHandler,
	chathandler.NewChatHandler,
	conversationhandler.NewConversationHandler,
	modelhandler.NewModelHandler,
	modelhandler.NewProviderHandler,
	modelhandler.NewModelCatalogHandler,
	modelhandler.NewProviderModelHandler,
	accesspolicyhandler.NewAccessPolicyHandler,
)
