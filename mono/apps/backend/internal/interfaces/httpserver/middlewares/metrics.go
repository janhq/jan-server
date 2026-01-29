package middlewares

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"jan-server/mono/apps/backend/internal/infrastructure/metrics"
)

// MetricsMiddleware records HTTP request metrics
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics after request completes
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = c.Request.URL.Path
		}

		// Get model from context if available (set by chat handler)
		model := c.GetString("model")
		if model == "" {
			model = "unknown"
		}

		// Get stream flag from context if available
		stream := c.GetBool("stream")

		metrics.RecordRequest(method, endpoint, status, model, stream, duration)

		// User agent metrics
		metrics.RecordUserAgent(c.Request.UserAgent())
	}
}
