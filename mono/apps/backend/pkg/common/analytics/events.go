package analytics

import "time"

// EventName defines known event types
type EventName string

const (
	// LLM Events
	EventChatResponse EventName = "chat_response"
	EventChatStarted  EventName = "chat_started"
	EventChatStopped  EventName = "chat_stopped"

	// User/Session Events
	EventAppOpened      EventName = "app_opened"
	EventSessionStarted EventName = "session_started"
	EventUserLoggedIn   EventName = "user_logged_in"
	EventUserLoggedOut  EventName = "user_logged_out"
	EventUserIdentified EventName = "user_identified"

	// Messaging Events
	EventMessageSent    EventName = "message_sent"
	EventMessageStopped EventName = "message_stopped"

	// Conversation Events
	EventConversationCreated EventName = "conversation_created"
	EventConversationDeleted EventName = "conversation_deleted"

	// Settings Events
	EventModePreferenceUpdated EventName = "mode_preference_updated"
	EventModeSelected          EventName = "mode_selected"
	EventModelSelected         EventName = "model_selected"

	// Share Events
	EventShareCreated EventName = "share_created"

	// Tool Events
	EventToolCalled    EventName = "tool_called"
	EventToolCompleted EventName = "tool_completed"
	EventToolFailed    EventName = "tool_failed"

	// Plan Events (response-api)
	EventPlanCreated      EventName = "plan_created"
	EventPlanCompleted    EventName = "plan_completed"
	EventPlanStepExecuted EventName = "plan_step_executed"

	// Session Events (realtime-api)
	EventRealtimeSessionCreated EventName = "realtime_session_created"
	EventRealtimeSessionDeleted EventName = "realtime_session_deleted"

	// Media Events
	EventMediaUploaded   EventName = "media_uploaded"
	EventMediaDownloaded EventName = "media_downloaded"

	// System Events
	EventHTTPRequest   EventName = "http_request"
	EventProviderError EventName = "provider_error"
	EventRateLimited   EventName = "rate_limited"
)

// Event represents a tracked event
type Event struct {
	Name       EventName              `json:"event"`
	DistinctID string                 `json:"distinct_id"`
	Timestamp  time.Time              `json:"timestamp"`
	Properties map[string]interface{} `json:"properties"`
}

// NewEvent creates a new event with the current timestamp
func NewEvent(name EventName, distinctID string) Event {
	return Event{
		Name:       name,
		DistinctID: distinctID,
		Timestamp:  time.Now().UTC(),
		Properties: make(map[string]interface{}),
	}
}

// WithProperty adds a property to the event and returns it for chaining
func (e Event) WithProperty(key string, value interface{}) Event {
	if e.Properties == nil {
		e.Properties = make(map[string]interface{})
	}
	e.Properties[key] = value
	return e
}

// WithProperties adds multiple properties to the event
func (e Event) WithProperties(props map[string]interface{}) Event {
	if e.Properties == nil {
		e.Properties = make(map[string]interface{})
	}
	for k, v := range props {
		e.Properties[k] = v
	}
	return e
}

// Validate checks if the event is valid
func (e Event) Validate() error {
	if e.Name == "" {
		return ErrInvalidEvent
	}
	if e.DistinctID == "" {
		return ErrMissingDistinctID
	}
	return nil
}

// UserProperties for identify calls
type UserProperties struct {
	Email     string            `json:"email,omitempty"`
	Name      string            `json:"name,omitempty"`
	CreatedAt time.Time         `json:"created_at,omitempty"`
	Plan      string            `json:"plan,omitempty"`
	Platform  string            `json:"platform,omitempty"`
	Custom    map[string]string `json:"custom,omitempty"`
}

// ToMap converts UserProperties to a map
func (u UserProperties) ToMap() map[string]interface{} {
	m := make(map[string]interface{})

	if u.Email != "" {
		m["email"] = u.Email
	}
	if u.Name != "" {
		m["name"] = u.Name
	}
	if !u.CreatedAt.IsZero() {
		m["created_at"] = u.CreatedAt
	}
	if u.Plan != "" {
		m["plan"] = u.Plan
	}
	if u.Platform != "" {
		m["platform"] = u.Platform
	}
	for k, v := range u.Custom {
		m[k] = v
	}

	return m
}
