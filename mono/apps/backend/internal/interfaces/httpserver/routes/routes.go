package routes

import (
	"net/http"

	"jan-server/mono/apps/backend/internal/infrastructure/config"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthCheck returns server health status.
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"service": "jan-server",
		"version": config.Version,
	})
}

// ReadyCheck returns server readiness status (includes DB check).
func ReadyCheck(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "error",
				"error": "database connection unavailable",
			})
			return
		}

		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "error",
				"error": "database ping failed",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	}
}

// ============================================
// Auth Routes (/auth/*)
// ============================================

// RegisterAuthRoutes registers authentication endpoints (public - no auth required).
func RegisterAuthRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB) {
	auth := rg.Group("/auth")
	{
		// Local auth endpoints (if enabled)
		if cfg.LocalAuthEnabled {
			auth.POST("/local/register", localRegisterHandler(cfg, db))
			auth.POST("/local/login", localLoginHandler(cfg, db))
			auth.POST("/local/refresh", localRefreshHandler(cfg, db))
		}

		// Keycloak auth endpoints (if enabled)
		if cfg.KeycloakEnabled {
			auth.GET("/login", keycloakLoginHandler(cfg))
			auth.GET("/callback", keycloakCallbackHandler(cfg, db))
			auth.POST("/guest-login", guestLoginHandler(cfg, db))
		}

		// Public auth endpoints
		auth.POST("/refresh-token", refreshTokenHandler(cfg, db))
		auth.POST("/logout", logoutHandler(cfg))
		auth.POST("/validate", validateTokenHandler(cfg, db))

		// Kong plugin endpoint (called by Kong, not users)
		if cfg.KongEnabled {
			auth.POST("/validate-api-key", validateAPIKeyForKongHandler(cfg, db))
		}
	}
}

// RegisterProtectedAuthRoutes registers auth endpoints that require authentication.
func RegisterProtectedAuthRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB) {
	auth := rg.Group("/auth")
	{
		// Get current user
		auth.GET("/me", meHandler(cfg, db))

		// Change password (requires auth)
		if cfg.LocalAuthEnabled {
			auth.POST("/local/change-password", localChangePasswordHandler(cfg, db))
		}

		// Upgrade guest (requires auth)
		if cfg.KeycloakEnabled {
			auth.POST("/upgrade", upgradeGuestHandler(cfg, db))
		}

		// API key management (requires auth)
		auth.POST("/api-keys", createAPIKeyHandler(cfg, db))
		auth.GET("/api-keys", listAPIKeysHandler(cfg, db))
		auth.DELETE("/api-keys/:id", deleteAPIKeyHandler(cfg, db))
	}
}

// ============================================
// LLM API Routes
// ============================================

// RegisterChatRoutes registers chat completion endpoints.
func RegisterChatRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB) {
	rg.POST("/chat/completions", chatCompletionsHandler(cfg, db))
}

// RegisterConversationRoutes registers conversation management endpoints.
func RegisterConversationRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB) {
	conv := rg.Group("/conversations")
	{
		conv.GET("", listConversationsHandler(cfg, db))
		conv.POST("", createConversationHandler(cfg, db))
		conv.GET("/:id", getConversationHandler(cfg, db))
		conv.PUT("/:id", updateConversationHandler(cfg, db))
		conv.DELETE("/:id", deleteConversationHandler(cfg, db))
		conv.POST("/:id/branch", branchConversationHandler(cfg, db))
	}
}

// RegisterModelRoutes registers model management endpoints.
func RegisterModelRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB) {
	rg.GET("/models", listModelsHandler(cfg, db))
	rg.GET("/models/:id", getModelHandler(cfg, db))
}

// RegisterProviderRoutes registers provider management endpoints.
func RegisterProviderRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB) {
	providers := rg.Group("/providers")
	{
		providers.GET("", listProvidersHandler(cfg, db))
		providers.GET("/:id", getProviderHandler(cfg, db))
	}
}

// RegisterMessageRoutes registers message endpoints.
func RegisterMessageRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB) {
	messages := rg.Group("/messages")
	{
		messages.GET("", listMessagesHandler(cfg, db))
		messages.GET("/:id", getMessageHandler(cfg, db))
	}
}

// RegisterConnectorRoutes registers OAuth connector endpoints.
func RegisterConnectorRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB) {
	connectors := rg.Group("/connectors")
	{
		connectors.GET("", listConnectorsHandler(cfg, db))
		connectors.GET("/:provider/auth", connectorAuthHandler(cfg, db))
		connectors.GET("/:provider/callback", connectorCallbackHandler(cfg, db))
		connectors.DELETE("/:provider", disconnectConnectorHandler(cfg, db))
	}
}

// ============================================
// Response API Routes
// ============================================

// RegisterResponseRoutes registers multi-step response endpoints.
func RegisterResponseRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB) {
	responses := rg.Group("/responses")
	{
		responses.POST("", createResponseHandler(cfg, db))
		responses.GET("/:id", getResponseHandler(cfg, db))
		responses.GET("/:id/full", getResponseFullHandler(cfg, db))
		responses.DELETE("/:id", deleteResponseHandler(cfg, db))
		responses.POST("/:id/cancel", cancelResponseHandler(cfg, db))
		responses.POST("/:id/retry", retryResponseHandler(cfg, db))
		responses.GET("/:id/input_items", getResponseInputItemsHandler(cfg, db))

		// Plan endpoints
		responses.GET("/:id/plan", getPlanHandler(cfg, db))
		responses.GET("/:id/plan/progress", getPlanProgressHandler(cfg, db))
		responses.POST("/:id/plan/cancel", cancelPlanHandler(cfg, db))
	}
}

// RegisterArtifactRoutes registers artifact management endpoints.
func RegisterArtifactRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB) {
	artifacts := rg.Group("/artifacts")
	{
		artifacts.GET("", listArtifactsHandler(cfg, db))
		artifacts.POST("", createArtifactHandler(cfg, db))
		artifacts.GET("/:id", getArtifactHandler(cfg, db))
		artifacts.PUT("/:id", updateArtifactHandler(cfg, db))
		artifacts.DELETE("/:id", deleteArtifactHandler(cfg, db))
		artifacts.GET("/:id/versions", listArtifactVersionsHandler(cfg, db))
		artifacts.GET("/:id/download", downloadArtifactHandler(cfg, db))
	}
}

// RegisterAgentRoutes registers agent discovery endpoints.
func RegisterAgentRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB) {
	agents := rg.Group("/agents")
	{
		agents.GET("", listAgentsHandler(cfg, db))
		agents.GET("/:id", getAgentHandler(cfg, db))
		agents.GET("/:id/capabilities", getAgentCapabilitiesHandler(cfg, db))
		agents.GET("/:id/schema", getAgentSchemaHandler(cfg, db))
	}
}

// ============================================
// Media API Routes
// ============================================

// RegisterMediaRoutes registers file upload/download endpoints.
func RegisterMediaRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB) {
	// /v1/media endpoints
	media := rg.Group("/media")
	{
		media.POST("", uploadMediaHandler(cfg, db))
		media.POST("/upload", getPresignedUploadURLHandler(cfg, db))
		media.GET("/:id", getMediaHandler(cfg, db))
		media.GET("/:id/metadata", getMediaMetadataHandler(cfg, db))
	}

	// /v1/files endpoints (alias)
	files := rg.Group("/files")
	{
		files.POST("", uploadMediaHandler(cfg, db))
		files.POST("/upload", getPresignedUploadURLHandler(cfg, db))
		files.GET("/:id", getMediaHandler(cfg, db))
		files.GET("/:id/metadata", getMediaMetadataHandler(cfg, db))
	}
}

// ============================================
// Memory Routes
// ============================================

// RegisterMemoryRoutes registers semantic memory endpoints.
func RegisterMemoryRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB) {
	memory := rg.Group("/memory")
	{
		memory.POST("/store", storeMemoryHandler(cfg, db))
		memory.POST("/search", searchMemoryHandler(cfg, db))
		memory.GET("", listMemoriesHandler(cfg, db))
		memory.DELETE("/:id", deleteMemoryHandler(cfg, db))
	}
}

// ============================================
// Realtime Routes
// ============================================

// RegisterRealtimeRoutes registers WebRTC session endpoints.
func RegisterRealtimeRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB) {
	realtime := rg.Group("/realtime/sessions")
	{
		realtime.POST("", createRealtimeSessionHandler(cfg, db))
		realtime.GET("", listRealtimeSessionsHandler(cfg, db))
		realtime.GET("/:id", getRealtimeSessionHandler(cfg, db))
		realtime.DELETE("/:id", deleteRealtimeSessionHandler(cfg, db))
	}
}

// ============================================
// MCP Routes
// ============================================

// RegisterMCPRoutes registers MCP JSON-RPC endpoint.
func RegisterMCPRoutes(router *gin.Engine, cfg *config.Config, db *gorm.DB) {
	router.POST("/mcp", mcpHandler(cfg, db))
	router.GET("/mcp", mcpHandler(cfg, db)) // For WebSocket upgrade
}

// ============================================
// Admin Routes
// ============================================

// RegisterAdminRoutes registers admin management endpoints.
func RegisterAdminRoutes(rg *gin.RouterGroup, cfg *config.Config, db *gorm.DB) {
	// Model management
	models := rg.Group("/models")
	{
		models.POST("", adminCreateModelHandler(cfg, db))
		models.PUT("/:id", adminUpdateModelHandler(cfg, db))
		models.DELETE("/:id", adminDeleteModelHandler(cfg, db))
	}

	// Provider management
	providers := rg.Group("/providers")
	{
		providers.POST("", adminCreateProviderHandler(cfg, db))
		providers.PUT("/:id", adminUpdateProviderHandler(cfg, db))
		providers.DELETE("/:id", adminDeleteProviderHandler(cfg, db))
	}

	// Prompt templates
	templates := rg.Group("/prompt-templates")
	{
		templates.GET("", adminListPromptTemplatesHandler(cfg, db))
		templates.POST("", adminCreatePromptTemplateHandler(cfg, db))
		templates.PUT("/:id", adminUpdatePromptTemplateHandler(cfg, db))
		templates.DELETE("/:id", adminDeletePromptTemplateHandler(cfg, db))
	}

	// User management
	users := rg.Group("/users")
	{
		users.GET("", adminListUsersHandler(cfg, db))
		users.GET("/:id", adminGetUserHandler(cfg, db))
		users.PUT("/:id", adminUpdateUserHandler(cfg, db))
		users.DELETE("/:id", adminDeleteUserHandler(cfg, db))
	}

	// MCP tools management
	mcpTools := rg.Group("/mcp-tools")
	{
		mcpTools.GET("", adminListMCPToolsHandler(cfg, db))
		mcpTools.PUT("/:id", adminUpdateMCPToolHandler(cfg, db))
	}
}

// ============================================
// Share Routes (Public)
// ============================================

// RegisterShareRoutes registers public conversation share endpoints.
func RegisterShareRoutes(router *gin.Engine, cfg *config.Config, db *gorm.DB) {
	// Public share access (no auth required)
	router.GET("/share/:token", getSharedConversationHandler(cfg, db))

	// Share management (requires auth) - registered in protected routes
}

// ============================================
// Swagger Routes
// ============================================

// RegisterSwaggerRoutes registers Swagger documentation endpoints.
func RegisterSwaggerRoutes(router *gin.Engine) {
	// TODO: Implement swagger routes using swaggo
	router.GET("/swagger/*any", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "swagger documentation coming soon",
		})
	})
}
