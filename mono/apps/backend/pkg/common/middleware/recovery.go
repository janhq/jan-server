package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"jan-server/mono/apps/backend/pkg/common/logger"
)

// Recovery is a middleware that recovers from panics
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				stack := debug.Stack()
				requestID := GetRequestID(c)

				logger.Error().
					Str("request_id", requestID).
					Str("method", c.Request.Method).
					Str("path", c.Request.URL.Path).
					Interface("error", err).
					Str("stack", string(stack)).
					Msg("Panic recovered")

				// Return 500 error
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"message":    "An internal error occurred",
						"type":       "internal_error",
						"request_id": requestID,
					},
				})
			}
		}()

		c.Next()
	}
}
