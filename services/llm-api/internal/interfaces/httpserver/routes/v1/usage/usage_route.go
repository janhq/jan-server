package usage

import (
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/usagehandler"

	"github.com/gin-gonic/gin"
)

// UsageRoute handles usage-related routes
type UsageRoute struct {
	handler     *usagehandler.UsageHandler
	authHandler *authhandler.AuthHandler
}

// NewUsageRoute creates a new UsageRoute
func NewUsageRoute(handler *usagehandler.UsageHandler, authHandler *authhandler.AuthHandler) *UsageRoute {
	return &UsageRoute{
		handler:     handler,
		authHandler: authHandler,
	}
}

// RegisterRouter registers usage routes on the given router
func (r *UsageRoute) RegisterRouter(router gin.IRouter) {
	usageGroup := router.Group("/usage")
	{
		// User's own usage - requires app user auth to get internal user ID
		usageGroup.GET("/me", r.authHandler.WithAppUserAuthChain(r.handler.GetMyUsage)...)
		usageGroup.GET("/me/daily", r.authHandler.WithAppUserAuthChain(r.handler.GetMyDailyUsage)...)
		usageGroup.GET("/me/activity", r.authHandler.WithAppUserAuthChain(r.handler.GetMyActivityUsage)...)

		// Project usage
		usageGroup.GET("/projects/:id", r.authHandler.WithAppUserAuthChain(r.handler.GetProjectUsage)...)
	}
}

// RegisterAdminRouter registers admin usage routes
func (r *UsageRoute) RegisterAdminRouter(router gin.IRouter) {
	router.GET("/usage", r.handler.GetPlatformUsage)
}
