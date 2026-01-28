package middlewares

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/domain"
)

// RequireAdmin ensures the authenticated principal carries an admin role or is_admin attribute.
// Admin authentication is ALWAYS enforced in production. Bypass is only allowed in development mode.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if bypass is allowed (only in development mode)
		if adminBypassEnabled() {
			log.Warn().
				Str("path", c.Request.URL.Path).
				Msg("Admin auth bypassed - this should only happen in development")
			c.Next()
			return
		}

		principal, ok := PrincipalFromContext(c)
		if !ok || principal.ID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Authentication required",
			})
			c.Abort()
			return
		}

		if isAdminPrincipal(principal) {
			c.Next()
			return
		}

		log.Warn().
			Str("user_id", principal.ID).
			Str("username", principal.Username).
			Str("path", c.Request.URL.Path).
			Strs("roles", principal.Roles).
			Msg("Non-admin user attempted to access admin endpoint")

		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Forbidden",
			"message": "Admin access required",
		})
		c.Abort()
	}
}

// adminBypassEnabled returns true only if bypass is explicitly enabled AND we're in development mode.
// In production, admin bypass is NEVER allowed regardless of environment variables.
func adminBypassEnabled() bool {
	// Only allow bypass in development mode
	if !config.IsDev() {
		return false
	}

	// Even in dev, bypass must be explicitly enabled
	if val := os.Getenv("ADMIN_BYPASS"); strings.EqualFold(val, "true") || val == "1" {
		return true
	}
	if val := os.Getenv("DISABLE_ADMIN_AUTH"); strings.EqualFold(val, "true") || val == "1" {
		return true
	}
	return false
}

// isAdminPrincipal checks if the principal has admin privileges.
// Admin status is determined by:
// 1. Having "admin" role in the Roles slice (from JWT realm_access.roles)
// 2. Having is_admin attribute set to true in JWT custom claims
func isAdminPrincipal(p domain.Principal) bool {
	// Check for admin role (case-insensitive)
	for _, role := range p.Roles {
		if strings.EqualFold(role, "admin") {
			return true
		}
	}

	// Check for is_admin attribute in JWT custom claims
	if p.Attributes != nil {
		if flag, ok := p.Attributes["is_admin"].(bool); ok && flag {
			return true
		}
	}

	return false
}
