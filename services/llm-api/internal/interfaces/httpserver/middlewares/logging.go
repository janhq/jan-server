package middlewares

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// LoggingMiddleware logs HTTP requests with OpenTelemetry trace context
func LoggingMiddleware(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// Build log event
		logEvent := logger.Info()
		if statusCode >= 500 {
			logEvent = logger.Error()
		} else if statusCode >= 400 {
			logEvent = logger.Warn()
		}

		// Add OpenTelemetry trace context if available
		span := trace.SpanFromContext(c.Request.Context())
		if span.SpanContext().IsValid() {
			logEvent = logEvent.
				Str("trace_id", span.SpanContext().TraceID().String()).
				Str("span_id", span.SpanContext().SpanID().String())
		}

		// Add request ID if available
		if requestID := c.GetString("request_id"); requestID != "" {
			logEvent = logEvent.Str("request_id", requestID)
		}

		// Log the request
		logEvent.
			Str("client_ip", clientIP).
			Str("method", method).
			Str("path", path).
			Str("query", raw).
			Int("status", statusCode).
			Dur("latency", latency).
			Str("user_agent", c.Request.UserAgent()).
			Msg(errorMessage)
	}
}
