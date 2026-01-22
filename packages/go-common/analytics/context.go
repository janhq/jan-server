package analytics

import (
	"context"
)

type contextKey string

const (
	trackerContextKey   contextKey = "analytics_tracker"
	distinctIDContextKey contextKey = "analytics_distinct_id"
	sessionIDContextKey  contextKey = "analytics_session_id"
	platformContextKey   contextKey = "analytics_platform"
	userStatusContextKey contextKey = "analytics_user_status"
)

// ContextWithTracker adds a tracker to the context
func ContextWithTracker(ctx context.Context, tracker Tracker) context.Context {
	return context.WithValue(ctx, trackerContextKey, tracker)
}

// TrackerFromContext extracts the tracker from context
// Returns a NoopTracker if no tracker is found
func TrackerFromContext(ctx context.Context) Tracker {
	if t, ok := ctx.Value(trackerContextKey).(Tracker); ok {
		return t
	}
	return NewNoopTracker()
}

// ContextWithDistinctID adds a distinct ID to the context
func ContextWithDistinctID(ctx context.Context, distinctID string) context.Context {
	return context.WithValue(ctx, distinctIDContextKey, distinctID)
}

// DistinctIDFromContext extracts the distinct ID from context
func DistinctIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(distinctIDContextKey).(string); ok {
		return id
	}
	return ""
}

// ContextWithSessionID adds a session ID to the context
func ContextWithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDContextKey, sessionID)
}

// SessionIDFromContext extracts the session ID from context
func SessionIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(sessionIDContextKey).(string); ok {
		return id
	}
	return ""
}

// ContextWithPlatform adds a platform to the context
func ContextWithPlatform(ctx context.Context, platform string) context.Context {
	return context.WithValue(ctx, platformContextKey, platform)
}

// PlatformFromContext extracts the platform from context
func PlatformFromContext(ctx context.Context) string {
	if p, ok := ctx.Value(platformContextKey).(string); ok {
		return p
	}
	return "web" // Default to web
}

// ContextWithUserStatus adds a user status to the context
func ContextWithUserStatus(ctx context.Context, status string) context.Context {
	return context.WithValue(ctx, userStatusContextKey, status)
}

// UserStatusFromContext extracts the user status from context
func UserStatusFromContext(ctx context.Context) string {
	if s, ok := ctx.Value(userStatusContextKey).(string); ok {
		return s
	}
	return "guest" // Default to guest
}

// TrackFromContext is a convenience function to track an event using the context tracker
func TrackFromContext(ctx context.Context, event Event) error {
	return TrackerFromContext(ctx).Track(ctx, event)
}

// ContextValues holds all analytics context values for easy extraction
type ContextValues struct {
	DistinctID string
	SessionID  string
	Platform   string
	UserStatus string
}

// ValuesFromContext extracts all analytics values from context
func ValuesFromContext(ctx context.Context) ContextValues {
	return ContextValues{
		DistinctID: DistinctIDFromContext(ctx),
		SessionID:  SessionIDFromContext(ctx),
		Platform:   PlatformFromContext(ctx),
		UserStatus: UserStatusFromContext(ctx),
	}
}

// EnrichEventFromContext adds context values to an event's properties
func EnrichEventFromContext(ctx context.Context, event Event) Event {
	values := ValuesFromContext(ctx)

	if event.DistinctID == "" && values.DistinctID != "" {
		event.DistinctID = values.DistinctID
	}

	if event.Properties == nil {
		event.Properties = make(map[string]interface{})
	}

	if _, ok := event.Properties[PropSessionID]; !ok && values.SessionID != "" {
		event.Properties[PropSessionID] = values.SessionID
	}
	if _, ok := event.Properties[PropPlatform]; !ok && values.Platform != "" {
		event.Properties[PropPlatform] = values.Platform
	}
	if _, ok := event.Properties[PropUserStatus]; !ok && values.UserStatus != "" {
		event.Properties[PropUserStatus] = values.UserStatus
	}

	return event
}
