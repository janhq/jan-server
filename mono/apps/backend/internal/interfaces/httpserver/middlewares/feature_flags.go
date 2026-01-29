package middlewares

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// FeatureEnabled checks whether the current principal has the given feature flag.
func FeatureEnabled(c *gin.Context, key string) bool {
	if key == "" {
		return false
	}

	principal, ok := PrincipalFromContext(c)
	if !ok {
		return false
	}

	for _, flag := range principal.FeatureFlags {
		if strings.EqualFold(flag, key) {
			return true
		}
	}
	return false
}
