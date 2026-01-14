package v1

import (
	"github.com/gin-gonic/gin"

	"jan-server/services/response-api/internal/interfaces/httpserver/handlers"
)

func registerAgentRoutes(router gin.IRoutes, handler *handlers.AgentHandler) {
	// Agent discovery routes
	// Note: /agents/capabilities must come before /agents/:type to avoid route conflicts
	router.GET("/agents/capabilities", handler.GetCapabilities)
	router.GET("/agents", handler.List)
	router.GET("/agents/:type", handler.Get)
	router.GET("/agents/:type/schema", handler.GetSchema)
}
