package users

import (
	"github.com/gin-gonic/gin"

	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/usersettingshandler"
)

// UsersRoute handles /v1/users routes
type UsersRoute struct {
	settingsHandler *usersettingshandler.UserSettingsHandler
	authHandler     *authhandler.AuthHandler
}

// NewUsersRoute constructs a new users route handler
func NewUsersRoute(
	settingsHandler *usersettingshandler.UserSettingsHandler,
	authHandler *authhandler.AuthHandler,
) *UsersRoute {
	return &UsersRoute{
		settingsHandler: settingsHandler,
		authHandler:     authHandler,
	}
}

// RegisterRouter registers user-related routes
func (r *UsersRoute) RegisterRouter(router gin.IRouter) {
	usersGroup := router.Group("/users")
	{
		// /v1/users/me/settings - User settings endpoints
		meGroup := usersGroup.Group("/me")
		{
			meGroup.GET("/settings", r.authHandler.WithAppUserAuthChain(r.settingsHandler.GetSettings)...)
			meGroup.PATCH("/settings", r.authHandler.WithAppUserAuthChain(r.settingsHandler.UpdateSettings)...)

			// /v1/users/me/settings/preferences - User preferences endpoints
			meGroup.GET("/settings/preferences", r.authHandler.WithAppUserAuthChain(r.settingsHandler.GetPreferences)...)
			meGroup.PATCH("/settings/preferences", r.authHandler.WithAppUserAuthChain(r.settingsHandler.UpdatePreferences)...)
		}
	}
}
