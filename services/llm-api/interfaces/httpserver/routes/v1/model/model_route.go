package model

import (
	"strings"

	"jan-server/services/llm-api/interfaces/httpserver/handlers/modelhandler"

	"github.com/gin-gonic/gin"
)

// ModelRoute handles public model endpoints
type ModelRoute struct {
	modelsHandler *modelhandler.ModelsHandler
}

// NewModelRoute creates a new ModelRoute
func NewModelRoute(modelsHandler *modelhandler.ModelsHandler) *ModelRoute {
	return &ModelRoute{
		modelsHandler: modelsHandler,
	}
}

// RegisterRouter registers model routes
func (r *ModelRoute) RegisterRouter(router gin.IRouter) {
	modelsGroup := router.Group("/models")
	{
		modelsGroup.GET("", r.ListModels)
		modelsGroup.GET("/*model_id", r.GetModel)
	}
}

// ListModels lists all available models
// @Summary List available models
// @Description Retrieves a list of available models
// @Tags Models
// @Security BearerAuth
// @Produce json
// @Success 200 {object} object "List of models"
// @Failure 500 {object} object "Internal server error"
// @Router /v1/models [get]
func (r *ModelRoute) ListModels(c *gin.Context) {
	r.modelsHandler.List(c)
}

// GetModel retrieves a specific model by ID
// @Summary Get model by ID
// @Description Retrieves detailed information about a specific model
// @Tags Models
// @Security BearerAuth
// @Produce json
// @Param model_id path string true "Model ID (can contain slashes)"
// @Success 200 {object} object "Model details"
// @Failure 404 {object} object "Model not found"
// @Failure 500 {object} object "Internal server error"
// @Router /v1/models/{model_id} [get]
func (r *ModelRoute) GetModel(c *gin.Context) {
	// Trim leading slash from wildcard param
	modelID := strings.TrimPrefix(c.Param("model_id"), "/")
	c.Set("model_id", modelID)
	r.modelsHandler.Get(c)
}
