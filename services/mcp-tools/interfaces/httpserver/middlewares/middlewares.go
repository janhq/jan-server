package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// RequestLogger logs HTTP requests
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("client_ip", c.ClientIP()).
			Msg("incoming request")

		c.Next()

		// Log errors if any
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				log.Error().
					Str("method", c.Request.Method).
					Str("path", c.Request.URL.Path).
					Int("status", c.Writer.Status()).
					Err(e.Err).
					Msg("request error")
			}
		}

		logEvent := log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status())

		if c.Writer.Status() >= 400 {
			logEvent = log.Warn().
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Int("status", c.Writer.Status())
		}

		logEvent.Msg("request completed")
	}
}

// CORS adds CORS headers
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key, Idempotency-Key, X-Request-Id, Mcp-Session-Id, mcp-protocol-version")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "X-Request-Id")
		c.Writer.Header().Set("Access-Control-Max-Age", "3600")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
