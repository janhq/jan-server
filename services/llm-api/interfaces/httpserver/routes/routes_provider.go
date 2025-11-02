package routes

import (
	"github.com/google/wire"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/handlers"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/middlewares"
	v1 "menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1"
	admin "menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1/admin"
	adminModel "menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1/admin/model"
	adminProvider "menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1/admin/provider"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1/auth"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1/auth/google"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1/chat"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1/conversation"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1/management"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1/management/apikeys"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1/management/billing"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1/mcp"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1/model"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1/model/provider"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1/referrer"
)

var RouteProvider = wire.NewSet(
	handlers.HandlerProvider,
	middlewares.MiddlewareProvider,
	v1.NewV1Route,

	// Admin routes
	admin.NewAdminRoute,
	adminModel.NewAdminModelRoute,
	adminProvider.NewAdminProviderRoute,

	auth.NewAuthRoute,
	google.NewGoogleRoute,
	management.NewManagementRoute,
	apikeys.NewAPIKeyRoute,
	billing.NewBillingRoute,
	billing.NewBillingEventRoute,
	billing.NewBillingHistoryRoute,
	model.NewModelRoute,
	provider.NewModelProviderRoute,
	chat.NewChatRoute,
	chat.NewChatCompletionRoute,
	conversation.NewConversationRoute,
	mcp.NewSerperMCP,
	mcp.NewMCPRoute,

	// Referrer routes
	referrer.NewReferrerRoute,
	referrer.NewReferrerCompletionRoute,
	referrer.NewReferrerModelRoute,
)
