package authhandler

import (
	"github.com/gin-gonic/gin"

	"jan-server/mono/apps/backend/internal/domain/user"
	middleware "jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/responses"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
	"jan-server/mono/apps/backend/internal/utils/ptr"
)

// GetUserFromContext returns the ensured application user from the request context.
func GetUserFromContext(c *gin.Context) (*user.User, bool) {
	val, ok := c.Get(appUserContextKey)
	if !ok || val == nil {
		return nil, false
	}
	usr, ok := val.(*user.User)
	return usr, ok && usr != nil
}

func (h *AuthHandler) ensureAppUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.userService == nil {
			c.Next()
			return
		}

		if _, ok := GetUserFromContext(c); ok {
			c.Next()
			return
		}

		principal, ok := middleware.PrincipalFromContext(c)
		if !ok {
			responses.HandleNewError(c, platformerrors.ErrorTypeUnauthorized, "authentication required", "5e1d3524-929e-4c7a-9bb7-0a8b74fa6f10")
			c.Abort()
			return
		}

		issuer := principal.Issuer
		if issuer == "" {
			issuer = principal.Credentials["issuer"]
		}

		identity := user.Identity{
			Provider: string(principal.AuthMethod),
			Issuer:   issuer,
			Subject:  principal.Subject,
		}
		if identity.Issuer == "" || identity.Subject == "" {
			responses.HandleNewError(c, platformerrors.ErrorTypeUnauthorized, "invalid user identity", "a6c6d3d0-5ca3-4235-9d54-8c4af3b04d62")
			c.Abort()
			return
		}

		if principal.Username != "" {
			identity.Username = ptr.ToString(principal.Username)
		}
		if principal.Email != "" {
			identity.Email = ptr.ToString(principal.Email)
		}
		if principal.Name != "" {
			identity.Name = ptr.ToString(principal.Name)
		}
		if picture := principal.Credentials["picture"]; picture != "" {
			identity.Picture = ptr.ToString(picture)
		}

		usr, err := h.userService.EnsureUser(c.Request.Context(), identity)
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to ensure user from principal")
			responses.HandleNewError(c, platformerrors.ErrorTypeInternal, "unable to resolve user identity", "7f6b30e8-6dc0-4af9-b42f-6fd717fe5a0c")
			c.Abort()
			return
		}

		c.Set(appUserContextKey, usr)
		c.Next()
	}
}
