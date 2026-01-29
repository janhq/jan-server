package provider

import (
	"net/http"

	modelHandler "jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/modelhandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/responses"
	modelresponses "jan-server/mono/apps/backend/internal/interfaces/httpserver/responses/model"

	"github.com/gin-gonic/gin"
)

type ModelProviderRoute struct {
	modelHandler *modelHandler.ModelHandler
}

func NewModelProviderRoute(modelHandler *modelHandler.ModelHandler) *ModelProviderRoute {
	return &ModelProviderRoute{
		modelHandler: modelHandler,
	}
}

func (modelProviderRoute *ModelProviderRoute) RegisterRouter(router *gin.RouterGroup) {
	group := router.Group("providers")
	group.GET("", modelProviderRoute.listProviders)
}

// listProviders godoc
// @Summary List model providers
// @Description Retrieves a list of available model providers that can be used for inference.
// @Tags Model API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} modelresponses.ProviderResponseList "List of providers"
// @Failure 500 {object} responses.ErrorResponse "Failed to retrieve providers"
// @Router /v1/models/providers [get]
func (modelProviderRoute *ModelProviderRoute) listProviders(reqCtx *gin.Context) {
	accessibleModels, err := modelProviderRoute.modelHandler.BuildAccessibleProviderModels(reqCtx)
	if err != nil || accessibleModels == nil {
		responses.HandleError(reqCtx, err, "Failed to retrieve providers")
		return
	}
	reqCtx.JSON(http.StatusOK, modelresponses.BuildProviderResponseList(accessibleModels.Providers))
}
