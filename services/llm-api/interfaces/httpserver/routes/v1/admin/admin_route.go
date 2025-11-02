package admin

import (
	adminmodel "jan-server/services/llm-api/interfaces/httpserver/routes/v1/admin/model"
	adminprovider "jan-server/services/llm-api/interfaces/httpserver/routes/v1/admin/provider"

	"github.com/gin-gonic/gin"
)

// AdminRoute aggregates all admin sub-routes
type AdminRoute struct {
	adminModelRoute    *adminmodel.AdminModelRoute
	adminProviderRoute *adminprovider.AdminProviderRoute
}

// NewAdminRoute creates a new AdminRoute
func NewAdminRoute(
	adminModelRoute *adminmodel.AdminModelRoute,
	adminProviderRoute *adminprovider.AdminProviderRoute,
) *AdminRoute {
	return &AdminRoute{
		adminModelRoute:    adminModelRoute,
		adminProviderRoute: adminProviderRoute,
	}
}

// RegisterRouter registers admin routes under /admin prefix
func (r *AdminRoute) RegisterRouter(router gin.IRouter) {
	adminGroup := router.Group("/admin")
	{
		r.adminModelRoute.RegisterRouter(adminGroup)
		r.adminProviderRoute.RegisterRouter(adminGroup)
	}
}
