package modelhandler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/llm-api/infrastructure/provider"
	"jan-server/services/llm-api/infrastructure/repo"
	"jan-server/services/llm-api/interfaces/httpserver/responses"
)

func RequestIDFromContext(c *gin.Context) string {
	if val, ok := c.Get("X-Request-Id"); ok {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// ModelsHandler exposes model catalogue endpoints.
type ModelsHandler struct {
	registry *provider.Registry
	repo     *repo.ModelRepository
	logger   zerolog.Logger
}

// NewModelsHandler creates a new handler instance.
func NewModelsHandler(registry *provider.Registry, repo *repo.ModelRepository, logger zerolog.Logger) *ModelsHandler {
	return &ModelsHandler{registry: registry, repo: repo, logger: logger}
}

// List handles GET /v1/models
// @Summary List models
// @Tags models
// @Security BearerJWT
// @Security ApiKey
// @Produce json
// @Success 200 {object} gin.H
// @Failure 401 {object} responses.ErrorResponse
// @Router /v1/models [get]
func (h *ModelsHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	models, err := h.repo.ListActive(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("list models")
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "list_models_failed",
			Message:   "unable to list models",
			RequestID: RequestIDFromContext(c),
		})
		return
	}

	if len(models) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": h.modelsFromRegistry()})
		return
	}

	responseModels := make([]gin.H, 0, len(models))
	for _, model := range models {
		responseModels = append(responseModels, gin.H{
			"id":           model.ID,
			"provider":     model.Provider,
			"display_name": model.DisplayName,
			"family":       model.Family,
			"capabilities": model.Capabilities,
			"active":       model.Active,
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": responseModels})
}

func (h *ModelsHandler) modelsFromRegistry() []gin.H {
	configs := h.registry.Models()
	models := make([]gin.H, 0, len(configs))
	for _, cfg := range configs {
		route, err := h.registry.Resolve(cfg.ID)
		providerName := ""
		if err == nil {
			providerName = route.Provider.Name()
		}
		models = append(models, gin.H{
			"id":           cfg.ID,
			"provider":     providerName,
			"display_name": cfg.ID,
			"family":       providerName,
			"capabilities": cfg.Capabilities,
			"active":       true,
		})
	}
	return models
}

// Get handles GET /v1/models/:model_id
// @Summary Get model metadata
// @Tags models
// @Security BearerJWT
// @Security ApiKey
// @Param model_id path string true "Model ID"
// @Produce json
// @Success 200 {object} gin.H
// @Failure 404 {object} responses.ErrorResponse
// @Router /v1/models/{model_id} [get]
func (h *ModelsHandler) Get(c *gin.Context) {
	modelID := strings.TrimPrefix(c.Param("model_id"), "/")
	ctx := c.Request.Context()
	model, err := h.repo.Get(ctx, modelID)
	if err != nil {
		h.logger.Error().Err(err).Str("model_id", modelID).Msg("get model")
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "model_lookup_failed",
			Message:   "unable to retrieve model",
			RequestID: RequestIDFromContext(c),
		})
		return
	}
	if model == nil {
		// fallback to config registry
		route, err := h.registry.Resolve(modelID)
		if err != nil {
			c.JSON(http.StatusNotFound, responses.ErrorResponse{
				Type:      responses.ErrorTypeInvalidRequest,
				Code:      "model_not_found",
				Message:   "model not found",
				RequestID: RequestIDFromContext(c),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"id":           route.Model.ID,
			"provider":     route.Provider.Name(),
			"display_name": route.Model.ID,
			"capabilities": route.Model.Capabilities,
			"active":       true,
		}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"id":           model.ID,
		"provider":     model.Provider,
		"display_name": model.DisplayName,
		"family":       model.Family,
		"capabilities": model.Capabilities,
		"active":       model.Active,
	}})
}
