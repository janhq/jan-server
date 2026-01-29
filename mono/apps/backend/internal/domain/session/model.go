package session

import "time"

// SessionState represents the state of a realtime session.
type SessionState string

const (
	// StateCreated indicates the session token was created, waiting for connection.
	StateCreated SessionState = "created"
	// StateConnected indicates the participant has joined the LiveKit room.
	StateConnected SessionState = "connected"
)

// Session represents a realtime session.
type Session struct {
	ID           string        `json:"id"`
	Object       string        `json:"object"` // "realtime.session"
	ClientSecret *ClientSecret `json:"client_secret,omitempty"`
	WsURL        string        `json:"ws_url,omitempty"` // LiveKit WebSocket URL
	RoomID       string        `json:"room_id,omitempty"`
	UserID       string        `json:"user_id,omitempty"`
	Status       SessionState  `json:"status,omitempty"` // connection status for GET responses

	// Internal tracking (not serialized to JSON response)
	Room      string    `json:"-"` // internal room name (same as RoomID)
	State     SessionState `json:"-"` // internal state tracking
	CreatedAt time.Time `json:"-"`
}

// ClientSecret contains the ephemeral token for client authentication.
type ClientSecret struct {
	Value     string `json:"value"`
	ExpiresAt int64  `json:"expires_at"` // actual token expiry timestamp
}

// CreateSessionRequest is the request body for creating a session.
// Currently empty - no parameters required.
type CreateSessionRequest struct {
	// No parameters - session creation uses server defaults
}

// ListSessionsResponse is the response for listing sessions.
type ListSessionsResponse struct {
	Object string     `json:"object"` // "list"
	Data   []*Session `json:"data"`
}

// DeleteSessionResponse is the response for deleting a session.
type DeleteSessionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"` // "realtime.session.deleted"
	Deleted bool   `json:"deleted"`
}
