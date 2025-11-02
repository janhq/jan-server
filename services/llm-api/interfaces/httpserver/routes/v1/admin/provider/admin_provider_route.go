package provider

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminProviderRoute handles admin provider management endpoints
type AdminProviderRoute struct {
	// Handlers will be added here
}

// NewAdminProviderRoute creates a new AdminProviderRoute
func NewAdminProviderRoute() *AdminProviderRoute {
	return &AdminProviderRoute{}
}

// RegisterRouter registers admin provider routes
func (r *AdminProviderRoute) RegisterRouter(router *gin.RouterGroup) {
	providerRoute := router.Group("/providers")
	{
		providerRoute.GET("", r.ListProviders)
		providerRoute.POST("", r.RegisterProvider)
		providerRoute.GET("/:provider_public_id", r.GetProvider)
		providerRoute.PATCH("/:provider_public_id", r.UpdateProvider)
		providerRoute.DELETE("/:provider_public_id", r.DeleteProvider)
	}
}

// ListProviders lists all providers
// @Summary List providers
// @Description Retrieves all providers with their model counts (admin)
// @Tags Admin - Providers
// @Security BearerAuth
// @Produce json
// @Success 200 {array} object "List of providers"
// @Failure 500 {object} object "Internal server error"
// @Router /v1/admin/providers [get]
func (r *AdminProviderRoute) ListProviders(c *gin.Context) {
	// TODO: Implement list providers
	c.JSON(http.StatusOK, []interface{}{})
}

// RegisterProvider registers a new provider
// @Summary Register provider
// @Description Registers a new provider and synchronizes its models (admin)
// @Tags Admin - Providers
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param payload body object true "Provider registration payload"
// @Success 201 {object} object "Registered provider"
// @Failure 400 {object} object "Invalid request"
// @Failure 500 {object} object "Internal server error"
// @Router /v1/admin/providers [post]
func (r *AdminProviderRoute) RegisterProvider(c *gin.Context) {
	// TODO: Implement register provider
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// GetProvider retrieves a specific provider
// @Summary Get provider
// @Description Retrieves detailed information about a provider (admin)
// @Tags Admin - Providers
// @Security BearerAuth
// @Produce json
// @Param provider_public_id path string true "Provider Public ID"
// @Success 200 {object} object "Provider details"
// @Failure 404 {object} object "Provider not found"
// @Failure 500 {object} object "Internal server error"
// @Router /v1/admin/providers/{provider_public_id} [get]
func (r *AdminProviderRoute) GetProvider(c *gin.Context) {
	publicID := c.Param("provider_public_id")
	// TODO: Implement get provider
	c.JSON(http.StatusNotFound, gin.H{"error": "not implemented", "provider_id": publicID})
}

// UpdateProvider updates a provider
// @Summary Update provider
// @Description Updates an existing provider's configuration (admin)
// @Tags Admin - Providers
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param provider_public_id path string true "Provider Public ID"
// @Param payload body object true "Provider update payload"
// @Success 200 {object} object "Updated provider"
// @Failure 400 {object} object "Invalid request"
// @Failure 404 {object} object "Provider not found"
// @Failure 500 {object} object "Internal server error"
// @Router /v1/admin/providers/{provider_public_id} [patch]
func (r *AdminProviderRoute) UpdateProvider(c *gin.Context) {
	publicID := c.Param("provider_public_id")
	// TODO: Implement update provider
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented", "provider_id": publicID})
}

// DeleteProvider deletes a provider
// @Summary Delete provider
// @Description Soft deletes a provider (admin)
// @Tags Admin - Providers
// @Security BearerAuth
// @Produce json
// @Param provider_public_id path string true "Provider Public ID"
// @Success 204 "Provider deleted"
// @Failure 404 {object} object "Provider not found"
// @Failure 500 {object} object "Internal server error"
// @Router /v1/admin/providers/{provider_public_id} [delete]
func (r *AdminProviderRoute) DeleteProvider(c *gin.Context) {
	publicID := c.Param("provider_public_id")
	// TODO: Implement delete provider
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented", "provider_id": publicID})
}
