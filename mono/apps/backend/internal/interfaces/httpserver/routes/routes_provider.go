package routes

import (
	"github.com/google/wire"

	"jan-server/mono/apps/backend/internal/domain/usersettings"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers"
	adminhandler "jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/admin"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/apikeyhandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/chathandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/connectorhandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/conversationhandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/documenthandler"
	guestauth "jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/guesthandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/imagehandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/mcptoolhandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/messageshandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/modelhandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/modelprompthandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/projecthandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/prompttemplatehandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/sharehandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/usagehandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/usersettingshandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/auth"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/public"
	v1 "jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1/admin"
	adminModel "jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1/admin/model"
	adminProvider "jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1/admin/provider"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1/chat"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1/connectors"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1/conversation"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1/image"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1/llm/documents"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1/llm/projects"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1/messages"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1/model"
	modelProvider "jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1/model/provider"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1/share"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1/usage"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes/v1/users"
)

var RouteProvider = wire.NewSet(
	// Handlers
	authhandler.NewAuthHandler,
	authhandler.NewTokenHandler,
	authhandler.ProvideKeycloakOAuthHandler,
	authhandler.NewRegisterHandler,
	apikeyhandler.NewHandler,
	handlers.ProvideMemoryHandler,
	chathandler.NewChatHandler,
	conversationhandler.NewConversationHandler,
	conversationhandler.NewBranchHandler,
	guestauth.NewGuestHandler,
	guestauth.NewUpgradeHandler,
	modelhandler.NewProviderHandler,
	modelhandler.NewModelHandler,
	modelhandler.NewModelCatalogHandler,
	modelhandler.NewProviderModelHandler,
	adminhandler.NewAdminUserHandler,
	adminhandler.NewAdminGroupHandler,
	adminhandler.NewFeatureFlagHandler,
	projecthandler.NewProjectHandler,
	usersettingshandler.NewUserSettingsHandler,
	prompttemplatehandler.NewPromptTemplateHandler,
	modelprompthandler.NewModelPromptTemplateHandler,
	sharehandler.NewShareHandler,
	mcptoolhandler.NewMCPToolHandler,
	imagehandler.NewImageHandler,
	messageshandler.NewMessagesHandler,
	usagehandler.NewUsageHandler,
	documenthandler.NewDocumentHandler,
	connectorhandler.NewConnectorHandler,

	// Bind ModelHandler to ModelProvider interface for usersettings
	wire.Bind(new(usersettings.ModelProvider), new(*modelhandler.ModelHandler)),

	// Routes
	auth.NewAuthRoute,
	v1.NewV1Route,
	admin.NewAdminRoute,
	adminModel.NewAdminModelRoute,
	adminProvider.NewAdminProviderRoute,
	chat.NewChatRoute,
	chat.NewChatCompletionRoute,
	conversation.NewConversationRoute,
	conversation.NewBranchRoute,
	projects.NewProjectRoute,
	model.NewModelRoute,
	modelProvider.NewModelProviderRoute,
	users.NewUsersRoute,
	share.NewShareRoute,
	public.NewPublicShareRoute,
	image.NewImageRoute,
	messages.NewMessagesRoute,
	usage.NewUsageRoute,
	documents.NewDocumentRoute,
	connectors.NewConnectorRoute,
)
