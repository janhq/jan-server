package model

import (
	"net/http"
	"strings"

	domainmodel "jan-server/services/llm-api/internal/domain/model"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	modelHandler "jan-server/services/llm-api/internal/interfaces/httpserver/handlers/modelhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/responses"
	modelresponses "jan-server/services/llm-api/internal/interfaces/httpserver/responses/model"
	modelProvider "jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/model/provider"
	"jan-server/services/llm-api/internal/utils/platformerrors"

	"github.com/gin-gonic/gin"
)

const HeaderIncludeProviderData = "X-PROVIDER-DATA"

type ModelRoute struct {
	modelHandler        *modelHandler.ModelHandler
	modelCatalogHandler *modelHandler.ModelCatalogHandler
	modelProvider       *modelProvider.ModelProviderRoute
	authHandler         *authhandler.AuthHandler
}

func NewModelRoute(
	modelHandler *modelHandler.ModelHandler,
	modelCatalogHandler *modelHandler.ModelCatalogHandler,
	modelProvider *modelProvider.ModelProviderRoute,
	authHandler *authhandler.AuthHandler,
) *ModelRoute {
	return &ModelRoute{
		modelHandler:        modelHandler,
		modelCatalogHandler: modelCatalogHandler,
		modelProvider:       modelProvider,
		authHandler:         authHandler,
	}
}

func (ModelRoute *ModelRoute) RegisterRouter(router *gin.RouterGroup) {
	modelsRoute := router.Group("models")
	modelsRoute.GET(
		"",
		ModelRoute.authHandler.WithAppUserAuthChain(ModelRoute.GetModels)...,
	)
	modelsRoute.GET("/catalogs/*model_public_id", ModelRoute.GetModelCatalog)

	ModelRoute.modelProvider.RegisterRouter(modelsRoute)

}

// ListModels
// @Summary List available models
// @Description Retrieves a list of available models that can be used for chat completions or other tasks.
// @Tags Chat Completions API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param X-PROVIDER-DATA header string false "Set to true to include provider metadata" Enums(true, false)
// @Success 200 {object} modelresponses.ModelResponseList "List of models"
// @Success 200 {object} modelresponses.ModelWithProviderResponseList "List of models with provider metadata"
// @Failure 404 {object} responses.ErrorResponse "Models or providers not found"
// @Failure 500 {object} responses.ErrorResponse "Failed to retrieve models"
func (ModelRoute *ModelRoute) GetModels(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	includeProviderData := strings.EqualFold(reqCtx.GetHeader(HeaderIncludeProviderData), "true")

	accessibleModels, err := ModelRoute.modelHandler.BuildAccessibleProviderModels(ctx)
	if err != nil || accessibleModels == nil {
		responses.HandleError(reqCtx, err, "Failed to retrieve accessible models")
		return
	}

	if len(accessibleModels.ProviderModels) == 0 || len(accessibleModels.Providers) == 0 {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeNotFound, "no models or providers found", "92597a6f-3846-451e-b2de-f41bf1fbff68")
		return
	}

	providerByID := make(map[uint]*domainmodel.Provider, len(accessibleModels.Providers))
	for _, provider := range accessibleModels.Providers {
		if provider == nil {
			continue
		}
		providerByID[provider.ID] = provider
	}

	if includeProviderData {
		models := modelresponses.BuildModelResponseListWithProvider(accessibleModels.ProviderModels, providerByID)
		reqCtx.JSON(http.StatusOK, modelresponses.ModelWithProviderResponseList{
			Object: "list",
			Data:   models,
		})

	} else {
		mergedProviderModels := ModelRoute.modelHandler.MergeModels(accessibleModels.ProviderModels, providerByID)
		mergedModels := modelresponses.BuildModelResponseList(mergedProviderModels, providerByID)
		reqCtx.JSON(http.StatusOK, modelresponses.ModelResponseList{
			Object: "list",
			Data:   mergedModels,
		})
	}

}

// GetModelCatalog
// @Summary Get a model catalog entry
// @Description Retrieves detailed information about a model catalog entry by its public ID (supports IDs with slashes like openrouter/nova-lite-v1)
// @Tags Model API
// @Security BearerAuth
// @Produce json
// @Param model_public_id path string true "Model Catalog Public ID (can contain slashes)"
// @Success 200 {object} modelresponses.ModelCatalogResponse "Model catalog details"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 404 {object} responses.ErrorResponse "Model catalog not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/models/{model_public_id} [get]
func (route *ModelRoute) GetModelCatalog(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	// Wildcard param includes leading slash, so trim it
	publicID := strings.TrimPrefix(reqCtx.Param("model_public_id"), "/")

	catalog, err := route.modelCatalogHandler.GetCatalog(ctx, publicID)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to retrieve model catalog")
		return
	}

	reqCtx.JSON(http.StatusOK, catalog)
}
