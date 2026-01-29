package prompttemplate

import (
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/prompttemplatehandler"
	middleware "jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"

	"github.com/gin-gonic/gin"
)

// PromptTemplateRoute handles routing for prompt template endpoints
type PromptTemplateRoute struct {
	handler *prompttemplatehandler.PromptTemplateHandler
}

// NewPromptTemplateRoute creates a new prompt template route
func NewPromptTemplateRoute(handler *prompttemplatehandler.PromptTemplateHandler) *PromptTemplateRoute {
	return &PromptTemplateRoute{handler: handler}
}

// RegisterAdminRouter registers admin routes for prompt templates under /admin/prompt-templates
func (r *PromptTemplateRoute) RegisterAdminRouter(router gin.IRouter) {
	promptTemplatesGroup := router.Group("/prompt-templates")
	promptTemplatesGroup.Use(middleware.RequireAdmin(), middleware.RateLimitMiddleware(100))
	{
		promptTemplatesGroup.GET("", r.handler.List)
		promptTemplatesGroup.POST("", r.handler.Create)
		promptTemplatesGroup.GET("/:id", r.handler.Get)
		promptTemplatesGroup.PATCH("/:id", r.handler.Update)
		promptTemplatesGroup.DELETE("/:id", r.handler.Delete)
		promptTemplatesGroup.POST("/:id/duplicate", r.handler.Duplicate)
	}
}

// RegisterPublicRouter registers public routes for prompt templates under /prompt-templates
func (r *PromptTemplateRoute) RegisterPublicRouter(router gin.IRouter) {
	promptTemplatesGroup := router.Group("/prompt-templates")
	{
		promptTemplatesGroup.GET("/:key", r.handler.GetByKey)
	}
}
