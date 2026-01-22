package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/janhq/jan-server/packages/go-common/analytics"
)

const (
	// Header names for analytics context propagation
	HeaderDistinctID = "X-PostHog-Distinct-Id"
	HeaderSessionID  = "X-Request-Session-Id"
	HeaderPlatform   = "X-Platform"
	HeaderRequestID  = "X-Request-Id"
)

// Config holds configuration for the analytics middleware
type Config struct {
	// Tracker is the analytics tracker to use
	Tracker analytics.Tracker

	// TrackHTTPRequests enables automatic tracking of HTTP requests
	TrackHTTPRequests bool

	// ExcludePaths are paths to exclude from HTTP request tracking
	ExcludePaths []string

	// ExtractDistinctID is a custom function to extract distinct ID
	// If nil, uses X-PostHog-Distinct-Id header or user ID from auth
	ExtractDistinctID func(*gin.Context) string

	// ExtractUserStatus is a custom function to extract user status
	// If nil, defaults to "guest"
	ExtractUserStatus func(*gin.Context) string
}

// DefaultConfig returns a default middleware configuration
func DefaultConfig(tracker analytics.Tracker) Config {
	return Config{
		Tracker:           tracker,
		TrackHTTPRequests: false,
		ExcludePaths: []string{
			"/health",
			"/healthz",
			"/ready",
			"/readyz",
			"/metrics",
			"/favicon.ico",
		},
	}
}

// isPathExcluded checks if a path should be excluded from tracking
// Supports both exact matches and prefix matches (paths ending with *)
func isPathExcluded(path string, excludePaths []string) bool {
	for _, excludePath := range excludePaths {
		if strings.HasSuffix(excludePath, "*") {
			// Prefix match
			prefix := strings.TrimSuffix(excludePath, "*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		} else {
			// Exact match
			if path == excludePath {
				return true
			}
		}
	}
	return false
}

// Analytics returns a Gin middleware that adds analytics context to requests
func Analytics(cfg Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Extract analytics context from headers
		distinctID := extractDistinctID(c, cfg.ExtractDistinctID)
		sessionID := c.GetHeader(HeaderSessionID)
		platform := c.GetHeader(HeaderPlatform)
		if platform == "" {
			platform = "web"
		}

		userStatus := "guest"
		if cfg.ExtractUserStatus != nil {
			userStatus = cfg.ExtractUserStatus(c)
		}

		// Add analytics context to request context
		ctx := c.Request.Context()
		ctx = analytics.ContextWithTracker(ctx, cfg.Tracker)
		ctx = analytics.ContextWithDistinctID(ctx, distinctID)
		ctx = analytics.ContextWithSessionID(ctx, sessionID)
		ctx = analytics.ContextWithPlatform(ctx, platform)
		ctx = analytics.ContextWithUserStatus(ctx, userStatus)
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// Track HTTP request if enabled
		if cfg.TrackHTTPRequests && !isPathExcluded(c.Request.URL.Path, cfg.ExcludePaths) {
			trackHTTPRequest(c, cfg.Tracker, distinctID, userStatus, platform, start)
		}
	}
}

// extractDistinctID extracts the distinct ID from the request
func extractDistinctID(c *gin.Context, customExtractor func(*gin.Context) string) string {
	// Try custom extractor first
	if customExtractor != nil {
		if id := customExtractor(c); id != "" {
			return id
		}
	}

	// Try PostHog header
	if id := c.GetHeader(HeaderDistinctID); id != "" {
		return id
	}

	// Try to get from auth context (common pattern)
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok && id != "" {
			return id
		}
	}

	// Try subject from JWT claims
	if subject, exists := c.Get("subject"); exists {
		if id, ok := subject.(string); ok && id != "" {
			return id
		}
	}

	return ""
}

// trackHTTPRequest tracks an HTTP request event
func trackHTTPRequest(c *gin.Context, tracker analytics.Tracker, distinctID, userStatus, platform string, start time.Time) {
	if distinctID == "" {
		return
	}

	event := analytics.NewEvent(analytics.EventHTTPRequest, distinctID).
		WithProperties(map[string]interface{}{
			"method":                 c.Request.Method,
			"path":                   c.Request.URL.Path,
			"status":                 c.Writer.Status(),
			analytics.PropDurationMs: time.Since(start).Milliseconds(),
			analytics.PropUserStatus: userStatus,
			analytics.PropPlatform:   platform,
		})

	if requestID := c.GetHeader(HeaderRequestID); requestID != "" {
		event = event.WithProperty(analytics.PropRequestID, requestID)
	}

	// Fire and forget - don't block on analytics
	go func() {
		_ = tracker.Track(c.Request.Context(), event)
	}()
}

// TrackerFromGin is a helper to get the tracker from a Gin context
func TrackerFromGin(c *gin.Context) analytics.Tracker {
	return analytics.TrackerFromContext(c.Request.Context())
}

// TrackEvent is a helper to track an event from a Gin handler
func TrackEvent(c *gin.Context, name analytics.EventName, props map[string]interface{}) error {
	ctx := c.Request.Context()
	values := analytics.ValuesFromContext(ctx)

	event := analytics.NewEvent(name, values.DistinctID).
		WithProperties(props)

	// Enrich with context values
	event = analytics.EnrichEventFromContext(ctx, event)

	return analytics.TrackerFromContext(ctx).Track(ctx, event)
}
