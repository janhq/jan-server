package middlewares

import (
	"github.com/gin-gonic/gin"
)

// CORS middleware for handling cross-origin requests
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RequestLogger logs incoming requests
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
