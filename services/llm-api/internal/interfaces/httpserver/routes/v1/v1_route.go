package v1

import (
	"net/http"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/mcptoolhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/prompttemplatehandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/public"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/admin"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/chat"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/conversation"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/image"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/llm/projects"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/messages"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/model"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/share"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/users"

	"github.com/gin-gonic/gin"
)

type V1Route struct {
	model                 *model.ModelRoute
	chat                  *chat.ChatRoute
	image                 *image.ImageRoute
	conversation          *conversation.ConversationRoute
	branch                *conversation.BranchRoute
	project               *projects.ProjectRoute
	adminRoute            *admin.AdminRoute
	users                 *users.UsersRoute
	promptTemplateHandler *prompttemplatehandler.PromptTemplateHandler
	mcpToolHandler        *mcptoolhandler.MCPToolHandler
	share                 *share.ShareRoute
	publicShare           *public.PublicShareRoute
	messages              *messages.MessagesRoute
}

func NewV1Route(
	model *model.ModelRoute,
	chat *chat.ChatRoute,
	image *image.ImageRoute,
	conversation *conversation.ConversationRoute,
	branch *conversation.BranchRoute,
	project *projects.ProjectRoute,
	adminRoute *admin.AdminRoute,
	users *users.UsersRoute,
	promptTemplateHandler *prompttemplatehandler.PromptTemplateHandler,
	mcpToolHandler *mcptoolhandler.MCPToolHandler,
	share *share.ShareRoute,
	publicShare *public.PublicShareRoute,
	messagesRoute *messages.MessagesRoute,
) *V1Route {
	return &V1Route{
		model,
		chat,
		image,
		conversation,
		branch,
		project,
		adminRoute,
		users,
		promptTemplateHandler,
		mcpToolHandler,
		share,
		publicShare,
		messagesRoute,
	}
}

func (v1Route *V1Route) RegisterRouter(router gin.IRouter) {
	v1Router := router.Group("/v1")
	v1Router.GET("/version", GetVersion)
	v1Router.GET("/healthz", GetHealthz)
	v1Router.GET("/readyz", GetReadyz)

	v1Route.adminRoute.RegisterRouter(v1Router)
	v1Route.model.RegisterRouter(v1Router)
	v1Route.chat.RegisterRouter(v1Router)
	v1Route.image.RegisterRouter(v1Router)
	v1Route.conversation.RegisterRouter(v1Router)
	v1Route.branch.RegisterRouter(v1Router)
	v1Route.project.RegisterRoutes(v1Router)
	v1Route.users.RegisterRouter(v1Router)
	v1Route.messages.RegisterRouter(v1Router)

	// Share routes (authenticated, under /conversations)
	conversations := v1Router.Group("/conversations")
	v1Route.share.RegisterConversationShareRoutes(conversations)

	// User share routes (authenticated, under /shares)
	shares := v1Router.Group("/shares")
	v1Route.share.RegisterUserShareRoutes(shares)
}

// RegisterPublicRouter registers endpoints that do not require authentication
func (v1Route *V1Route) RegisterPublicRouter(router gin.IRouter) {
	v1Router := router.Group("/v1")

	// Public prompt template endpoints
	v1Router.GET("/prompt-templates/:key", v1Route.promptTemplateHandler.GetByKey)

	// Public MCP tool endpoints (for mcp-tools service)
	v1Router.GET("/mcp-tools", v1Route.mcpToolHandler.ListActive)
	v1Router.GET("/mcp-tools/:key", v1Route.mcpToolHandler.GetByKey)

	// Public share routes (no auth required)
	v1Route.publicShare.RegisterRouter(v1Router)
}

// GetVersion godoc
// @Summary Get API build version
// @Description Returns the current build version of the API server and environment reload timestamp.
// @Tags Server API
// @Produce json
// @Success 200 {object} map[string]string "Version information including version number and environment reload timestamp"
// @Router /v1/version [get]
func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":         config.Version,
		"env_reloaded_at": config.GetEnvReloadedAt().Format("2006-01-02T15:04:05Z07:00"),
	})
}

// GetHealthz godoc
// @Summary Health check endpoint
// @Description Returns the health status of the API server. Used by orchestrators and monitoring systems.
// @Tags Server API
// @Produce json
// @Success 200 {object} map[string]string "Health status OK"
// @Router /v1/healthz [get]
func GetHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetReadyz godoc
// @Summary Readiness check endpoint
// @Description Returns the readiness status of the API server. Indicates if the service is ready to accept traffic.
// @Tags Server API
// @Produce json
// @Success 200 {object} map[string]string "Readiness status ready"
// @Router /v1/readyz [get]
func GetReadyz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
