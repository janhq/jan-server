package v1

import (
	"net/http"

	"jan-server/services/llm-api/interfaces/httpserver/routes/v1/admin"
	"jan-server/services/llm-api/interfaces/httpserver/routes/v1/chat"
	"jan-server/services/llm-api/interfaces/httpserver/routes/v1/conversation"
	"jan-server/services/llm-api/interfaces/httpserver/routes/v1/model"

	"github.com/gin-gonic/gin"
)

// V1Route aggregates all v1 sub-routes
type V1Route struct {
	model        *model.ModelRoute
	chat         *chat.ChatRoute
	conversation *conversation.ConversationRoute
	admin        *admin.AdminRoute
}

// NewV1Route creates a new V1 route with all sub-routes
func NewV1Route(
	modelRoute *model.ModelRoute,
	chatRoute *chat.ChatRoute,
	conversationRoute *conversation.ConversationRoute,
	adminRoute *admin.AdminRoute,
) *V1Route {
	return &V1Route{
		model:        modelRoute,
		chat:         chatRoute,
		conversation: conversationRoute,
		admin:        adminRoute,
	}
}

// RegisterRouter registers all v1 routes
func (v1 *V1Route) RegisterRouter(router gin.IRouter) {
	v1Router := router.Group("/v1")

	// Health check endpoints
	v1Router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1Router.GET("/readyz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Register sub-routes
	v1.model.RegisterRouter(v1Router)
	v1.chat.RegisterRouter(v1Router)
	v1.conversation.RegisterRouter(v1Router)
	v1.admin.RegisterRouter(v1Router)
}
