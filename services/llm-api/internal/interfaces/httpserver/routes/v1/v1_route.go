package v1

import (
	"net/http"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/admin"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/chat"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/conversation"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/llm/projects"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/model"

	"github.com/gin-gonic/gin"
)

type V1Route struct {
	model        *model.ModelRoute
	chat         *chat.ChatRoute
	conversation *conversation.ConversationRoute
	project      *projects.ProjectRoute
	adminRoute   *admin.AdminRoute
}

func NewV1Route(
	model *model.ModelRoute,
	chat *chat.ChatRoute,
	conversation *conversation.ConversationRoute,
	project *projects.ProjectRoute,
	adminRoute *admin.AdminRoute) *V1Route {
	return &V1Route{
		model,
		chat,
		conversation,
		project,
		adminRoute,
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
	v1Route.conversation.RegisterRouter(v1Router)

	// LLM routes group
	llmRouter := v1Router.Group("/llm")
	v1Route.project.RegisterRoutes(llmRouter)

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
