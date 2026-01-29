package authhandler

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/mono/apps/backend/internal/domain/user"
)

const appUserContextKey = "app_user"

// AuthHandler coordinates per-request authentication helpers.
type AuthHandler struct {
	userService *user.Service
	logger      zerolog.Logger
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(userService *user.Service, logger zerolog.Logger) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		logger:      logger,
	}
}

// WithAppUserAuthChain ensures the authenticated app user exists before executing handlers.
func (h *AuthHandler) WithAppUserAuthChain(handlers ...gin.HandlerFunc) []gin.HandlerFunc {
	chain := []gin.HandlerFunc{h.ensureAppUser()}
	return append(chain, handlers...)
}

// RequireAuth currently delegates to WithAppUserAuthChain for backwards compatibility.
func (h *AuthHandler) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		h.ensureAppUser()(c)
	}
}
