package provider

import (
	"net/http"
	"strings"

	domainmodel "jan-server/mono/apps/backend/internal/domain/model"
	modelHandler "jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/modelhandler"
	requestmodels "jan-server/mono/apps/backend/internal/interfaces/httpserver/requests/models"

	"jan-server/mono/apps/backend/internal/interfaces/httpserver/responses"

	"github.com/gin-gonic/gin"
)

type AdminProviderRoute struct {
	providerHandler *modelHandler.ProviderHandler
}

func NewAdminProviderRoute(
	providerHandler *modelHandler.ProviderHandler,
) *AdminProviderRoute {
	return &AdminProviderRoute{
		providerHandler: providerHandler,
	}
}

func (AdminProviderRoute *AdminProviderRoute) RegisterRouter(router *gin.RouterGroup) {
	providerRoute := router.Group("providers")

	providerRoute.GET("", AdminProviderRoute.GetAllProviders)
	providerRoute.POST("", AdminProviderRoute.RegisterProvider)
	providerRoute.GET("/:provider_public_id", AdminProviderRoute.GetProvider)
	providerRoute.PATCH("/:provider_public_id", AdminProviderRoute.UpdateProvider)
	providerRoute.DELETE("/:provider_public_id", AdminProviderRoute.DeleteProvider)

}

// GetAllProviders
// @Summary Get all providers
// @Description Retrieves all providers with their model counts
// @Tags Admin Provider API
// @Security BearerAuth
// @Produce json
// @Success 200 {array} modelresponses.ProviderWithModelCountResponse "List of providers with model counts"
// @Failure 500 {object} responses.ErrorResponse "Failed to retrieve providers"
// @Router /v1/admin/providers [get]
func (route *AdminProviderRoute) GetAllProviders(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	search := strings.TrimSpace(reqCtx.Query("search"))
	filter := domainmodel.ProviderFilter{}
	if search != "" {
		filter.SearchText = &search
	}

	providersWithCounts, err := route.providerHandler.GetAllProviders(ctx, filter)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to retrieve providers")
		return
	}

	reqCtx.JSON(http.StatusOK, providersWithCounts)
}

// RegisterProvider
// @Summary Register a provider
// @Description Registers a new provider and synchronizes its available models.
// @Tags Admin Provider API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param payload body requestmodels.AddProviderRequest true "Provider registration payload"
// @Success 200 {object} modelresponses.ProviderWithModelsResponse "Registered provider with synced models"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload"
// @Failure 500 {object} responses.ErrorResponse "Failed to register provider"
// @Router /v1/admin/providers [post]
func (route *AdminProviderRoute) RegisterProvider(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	var request requestmodels.AddProviderRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		responses.HandleError(reqCtx, err, "Invalid request body")
		return
	}

	providerWithModels, err := route.providerHandler.RegisterProvider(request, ctx)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to register provider")
		return
	}

	reqCtx.JSON(http.StatusOK, providerWithModels)
}

// GetProvider
// @Summary Get a provider
// @Description Retrieves a provider by its public ID
// @Tags Admin Provider API
// @Security BearerAuth
// @Produce json
// @Param provider_public_id path string true "Provider public ID"
// @Success 200 {object} modelresponses.ProviderResponse "Provider details"
// @Failure 404 {object} responses.ErrorResponse "Provider not found"
// @Failure 500 {object} responses.ErrorResponse "Failed to retrieve provider"
// @Router /v1/admin/providers/{provider_public_id} [get]
func (route *AdminProviderRoute) GetProvider(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	publicID := reqCtx.Param("provider_public_id")

	providerResponse, err := route.providerHandler.GetProviderByPublicID(ctx, publicID)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to retrieve provider")
		return
	}

	reqCtx.JSON(http.StatusOK, providerResponse)
}

// UpdateProvider
// @Summary Update a provider
// @Description Updates an existing provider's configuration
// @Tags Admin Provider API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param provider_public_id path string true "Provider public ID"
// @Param payload body requestmodels.UpdateProviderRequest true "Provider update payload"
// @Success 200 {object} modelresponses.ProviderResponse "Updated provider"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload"
// @Failure 404 {object} responses.ErrorResponse "Provider not found"
// @Failure 500 {object} responses.ErrorResponse "Failed to update provider"
// @Router /v1/admin/providers/{provider_public_id} [patch]
func (route *AdminProviderRoute) UpdateProvider(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	publicID := reqCtx.Param("provider_public_id")

	var request requestmodels.UpdateProviderRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		responses.HandleError(reqCtx, err, "Invalid request body")
		return
	}

	providerResponse, err := route.providerHandler.UpdateProvider(ctx, publicID, request)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to update provider")
		return
	}

	reqCtx.JSON(http.StatusOK, providerResponse)
}

// DeleteProvider
// @Summary Delete a provider
// @Description Deletes a provider by its public ID along with its provider models
// @Tags Admin Provider API
// @Security BearerAuth
// @Produce json
// @Param provider_public_id path string true "Provider public ID"
// @Success 204 "Provider deleted"
// @Failure 404 {object} responses.ErrorResponse "Provider not found"
// @Failure 500 {object} responses.ErrorResponse "Failed to delete provider"
// @Router /v1/admin/providers/{provider_public_id} [delete]
func (route *AdminProviderRoute) DeleteProvider(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	publicID := reqCtx.Param("provider_public_id")

	if err := route.providerHandler.DeleteProvider(ctx, publicID); err != nil {
		responses.HandleError(reqCtx, err, "Failed to delete provider")
		return
	}

	reqCtx.Status(http.StatusNoContent)
}
