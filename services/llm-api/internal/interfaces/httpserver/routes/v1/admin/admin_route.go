package admin

import (
	adminhandler "jan-server/services/llm-api/internal/interfaces/httpserver/handlers/admin"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/mcptoolhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/prompttemplatehandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/usagehandler"
	middleware "jan-server/services/llm-api/internal/interfaces/httpserver/middlewares"
	adminmodel "jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/admin/model"
	adminprovider "jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/admin/provider"

	"github.com/gin-gonic/gin"
)

// AdminRoute aggregates all admin sub-routes
type AdminRoute struct {
	adminModelRoute       *adminmodel.AdminModelRoute
	adminProviderRoute    *adminprovider.AdminProviderRoute
	userHandler           *adminhandler.AdminUserHandler
	groupHandler          *adminhandler.AdminGroupHandler
	featureFlagHandler    *adminhandler.FeatureFlagHandler
	promptTemplateHandler *prompttemplatehandler.PromptTemplateHandler
	mcpToolHandler        *mcptoolhandler.MCPToolHandler
	usageHandler          *usagehandler.UsageHandler
}

// NewAdminRoute creates a new AdminRoute
func NewAdminRoute(
	adminModelRoute *adminmodel.AdminModelRoute,
	adminProviderRoute *adminprovider.AdminProviderRoute,
	userHandler *adminhandler.AdminUserHandler,
	groupHandler *adminhandler.AdminGroupHandler,
	featureFlagHandler *adminhandler.FeatureFlagHandler,
	promptTemplateHandler *prompttemplatehandler.PromptTemplateHandler,
	mcpToolHandler *mcptoolhandler.MCPToolHandler,
	usageHandler *usagehandler.UsageHandler,
) *AdminRoute {
	return &AdminRoute{
		adminModelRoute:       adminModelRoute,
		adminProviderRoute:    adminProviderRoute,
		userHandler:           userHandler,
		groupHandler:          groupHandler,
		featureFlagHandler:    featureFlagHandler,
		promptTemplateHandler: promptTemplateHandler,
		mcpToolHandler:        mcpToolHandler,
		usageHandler:          usageHandler,
	}
}

// RegisterRouter registers admin routes under /admin prefix
func (r *AdminRoute) RegisterRouter(router gin.IRouter) {
	adminGroup := router.Group("/admin")
	adminGroup.Use(middleware.RequireAdmin(), middleware.RateLimitMiddleware(100))
	{
		r.adminModelRoute.RegisterRouter(adminGroup)
		r.adminProviderRoute.RegisterRouter(adminGroup)

		// User management
		adminGroup.GET("/users", r.userHandler.ListUsers)
		adminGroup.POST("/users", r.userHandler.CreateUser)
		adminGroup.GET("/users/:id", r.userHandler.GetUser)
		adminGroup.PATCH("/users/:id", r.userHandler.UpdateUser)
		adminGroup.DELETE("/users/:id", r.userHandler.DeleteUser)
		adminGroup.POST("/users/:id/activate", r.userHandler.ActivateUser)
		adminGroup.POST("/users/:id/deactivate", r.userHandler.DeactivateUser)
		adminGroup.POST("/users/:id/roles/:role", r.userHandler.AssignRole)
		adminGroup.DELETE("/users/:id/roles/:role", r.userHandler.RemoveRole)

		// Group management
		adminGroup.GET("/groups", r.groupHandler.ListGroups)
		adminGroup.POST("/groups", r.groupHandler.CreateGroup)
		adminGroup.GET("/groups/:id", r.groupHandler.GetGroup)
		adminGroup.PATCH("/groups/:id", r.groupHandler.UpdateGroup)
		adminGroup.DELETE("/groups/:id", r.groupHandler.DeleteGroup)
		adminGroup.GET("/groups/:id/members", r.groupHandler.GetGroupMembers)
		adminGroup.POST("/users/:id/groups/:groupId", r.groupHandler.AddUserToGroup)
		adminGroup.DELETE("/users/:id/groups/:groupId", r.groupHandler.RemoveUserFromGroup)
		adminGroup.GET("/groups/:id/feature-flags", r.groupHandler.GetGroupFeatureFlags)
		adminGroup.PATCH("/groups/:id/feature-flags", r.groupHandler.SetGroupFeatureFlags)
		adminGroup.POST("/groups/:id/feature-flags/:flagKey", r.groupHandler.EnableGroupFeatureFlag)
		adminGroup.DELETE("/groups/:id/feature-flags/:flagKey", r.groupHandler.DisableGroupFeatureFlag)

		// Feature flag definitions
		adminGroup.GET("/feature-flags", r.featureFlagHandler.ListFeatureFlags)
		adminGroup.POST("/feature-flags", r.featureFlagHandler.CreateFeatureFlag)
		adminGroup.PATCH("/feature-flags/:id", r.featureFlagHandler.UpdateFeatureFlag)
		adminGroup.DELETE("/feature-flags/:id", r.featureFlagHandler.DeleteFeatureFlag)

		// Prompt template management
		adminGroup.GET("/prompt-templates", r.promptTemplateHandler.List)
		adminGroup.POST("/prompt-templates", r.promptTemplateHandler.Create)
		adminGroup.GET("/prompt-templates/:id", r.promptTemplateHandler.Get)
		adminGroup.PATCH("/prompt-templates/:id", r.promptTemplateHandler.Update)
		adminGroup.DELETE("/prompt-templates/:id", r.promptTemplateHandler.Delete)
		adminGroup.POST("/prompt-templates/:id/duplicate", r.promptTemplateHandler.Duplicate)

		// MCP tool management
		adminGroup.GET("/mcp-tools", r.mcpToolHandler.List)
		adminGroup.GET("/mcp-tools/:id", r.mcpToolHandler.Get)
		adminGroup.PATCH("/mcp-tools/:id", r.mcpToolHandler.Update)

		// Usage analytics (admin)
		adminGroup.GET("/usage", r.usageHandler.GetPlatformUsage)
	}
}
