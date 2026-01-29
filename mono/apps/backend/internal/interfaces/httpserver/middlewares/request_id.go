package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-Id"

// RequestID injects an X-Request-Id header when missing and makes it available via gin context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(requestIDHeader)
		if requestID == "" {
			requestID = uuid.NewString()
			c.Request.Header.Set(requestIDHeader, requestID)
		}
		c.Writer.Header().Set(requestIDHeader, requestID)
		c.Set(requestIDHeader, requestID)
		c.Next()
	}
}

// RequestIDFromContext returns the request id stored in the gin context.
func RequestIDFromContext(c *gin.Context) string {
	if val, ok := c.Get(requestIDHeader); ok {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}
