package auth

import (
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	guestauth "jan-server/services/llm-api/internal/interfaces/httpserver/handlers/guesthandler"

	"github.com/gin-gonic/gin"
)

// AuthRoute handles authentication routes
type AuthRoute struct {
	guestHandler   *guestauth.GuestHandler
	upgradeHandler *guestauth.UpgradeHandler
	tokenHandler   *authhandler.TokenHandler
}

// NewAuthRoute creates a new auth route
func NewAuthRoute(
	guestHandler *guestauth.GuestHandler,
	upgradeHandler *guestauth.UpgradeHandler,
	tokenHandler *authhandler.TokenHandler,
) *AuthRoute {
	return &AuthRoute{
		guestHandler:   guestHandler,
		upgradeHandler: upgradeHandler,
		tokenHandler:   tokenHandler,
	}
}

// RegisterRouter registers auth routes
func (a *AuthRoute) RegisterRouter(router gin.IRouter, protectedRouter gin.IRouter) {
	// Public routes
	router.POST("/auth/guest-login", a.guestHandler.CreateGuest)
	router.GET("/auth/refresh-token", a.tokenHandler.RefreshToken)
	router.GET("/auth/logout", a.tokenHandler.Logout)

	// Protected routes (require authentication)
	protectedRouter.POST("/auth/upgrade", a.upgradeHandler.Upgrade)
	protectedRouter.GET("/auth/me", a.tokenHandler.GetMe)
}
