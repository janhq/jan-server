package middlewares

import (
	"github.com/gin-gonic/gin"
)

// CORSMiddleware returns a middleware that handles CORS
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// List of allowed origins
		allowedOrigins := map[string]bool{
			"http://localhost":      true,
			"http://localhost:3000": true,
			"http://localhost:8080": true,
			"http://127.0.0.1":      true,
		}

		// Check if origin is allowed
		if allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key, Idempotency-Key, X-Request-Id, Mcp-Session-Id")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "X-Request-Id")
		c.Writer.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
