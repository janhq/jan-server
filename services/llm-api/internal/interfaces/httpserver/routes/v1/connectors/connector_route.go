package connectors

import (
	"github.com/gin-gonic/gin"

	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/connectorhandler"
)

// ConnectorRoute handles connector routes.
type ConnectorRoute struct {
	handler     *connectorhandler.ConnectorHandler
	authHandler *authhandler.AuthHandler
}

// NewConnectorRoute creates a new connector route.
func NewConnectorRoute(handler *connectorhandler.ConnectorHandler, authHandler *authhandler.AuthHandler) *ConnectorRoute {
	return &ConnectorRoute{
		handler:     handler,
		authHandler: authHandler,
	}
}

// RegisterRouter registers connector routes on the authenticated router.
func (r *ConnectorRoute) RegisterRouter(router gin.IRouter) {
	connectors := router.Group("/connectors")
	{
		connectors.GET("", r.authHandler.WithAppUserAuthChain(r.handler.ListConnectors)...)
		connectors.GET("/:type", r.authHandler.WithAppUserAuthChain(r.handler.GetConnector)...)
		connectors.GET("/:type/auth-url", r.authHandler.WithAppUserAuthChain(r.handler.GetAuthURL)...)
		connectors.POST("/:type/connect", r.authHandler.WithAppUserAuthChain(r.handler.Connect)...)
		connectors.DELETE("/:type/disconnect", r.authHandler.WithAppUserAuthChain(r.handler.Disconnect)...)
		connectors.GET("/:type/status", r.authHandler.WithAppUserAuthChain(r.handler.GetStatus)...)
		connectors.GET("/:type/token", r.authHandler.WithAppUserAuthChain(r.handler.GetToken)...)
		connectors.POST("/:type/refresh", r.authHandler.WithAppUserAuthChain(r.handler.RefreshTokens)...)
	}
}

// RegisterPublicRouter registers public connector routes (OAuth callbacks).
func (r *ConnectorRoute) RegisterPublicRouter(router gin.IRouter) {
	connectors := router.Group("/connectors")
	{
		// OAuth callback must be public (no auth header in redirect)
		connectors.GET("/:type/callback", r.handler.HandleCallback)
	}
}
