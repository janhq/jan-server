// Package sessionres contains HTTP response DTOs for session endpoints.
package sessionres

import (
	domainsession "jan-server/mono/apps/backend/internal/domain/session"
)

// SessionResponse represents a session in API responses.
type SessionResponse struct {
	ID           string              `json:"id"`
	Object       string              `json:"object"`
	ClientSecret *ClientSecretDetail `json:"client_secret,omitempty"`
	WsURL        string              `json:"ws_url,omitempty"`
	RoomID       string              `json:"room_id,omitempty"`
	UserID       string              `json:"user_id,omitempty"`
	Status       string              `json:"status,omitempty"`
}

// ClientSecretDetail contains the client secret for a session.
type ClientSecretDetail struct {
	Value     string `json:"value"`
	ExpiresAt int64  `json:"expires_at"`
}

// ListSessionsResponse represents the response for listing sessions.
type ListSessionsResponse struct {
	Object string             `json:"object"`
	Data   []*SessionResponse `json:"data"`
}

// DeleteSessionResponse represents the response for deleting a session.
type DeleteSessionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

// NewSessionResponse creates a SessionResponse from a domain Session.
// Use this for POST responses that include client_secret.
func NewSessionResponse(sess *domainsession.Session) *SessionResponse {
	resp := &SessionResponse{
		ID:     sess.ID,
		Object: sess.Object,
		WsURL:  sess.WsURL,
		RoomID: sess.RoomID,
		UserID: sess.UserID,
	}

	if sess.ClientSecret != nil {
		resp.ClientSecret = &ClientSecretDetail{
			Value:     sess.ClientSecret.Value,
			ExpiresAt: sess.ClientSecret.ExpiresAt,
		}
	}

	return resp
}

// NewSessionResponseForGet creates a SessionResponse for GET responses.
// Excludes client_secret and includes status.
func NewSessionResponseForGet(sess *domainsession.Session) *SessionResponse {
	return &SessionResponse{
		ID:     sess.ID,
		Object: sess.Object,
		WsURL:  sess.WsURL,
		RoomID: sess.Room,
		UserID: sess.UserID,
		Status: string(sess.State),
	}
}

// NewListSessionsResponse creates a ListSessionsResponse from domain Sessions.
func NewListSessionsResponse(sessions []*domainsession.Session) *ListSessionsResponse {
	data := make([]*SessionResponse, len(sessions))
	for i, s := range sessions {
		data[i] = NewSessionResponseForGet(s)
	}

	return &ListSessionsResponse{
		Object: "list",
		Data:   data,
	}
}

// NewDeleteSessionResponse creates a DeleteSessionResponse.
func NewDeleteSessionResponse(id string) *DeleteSessionResponse {
	return &DeleteSessionResponse{
		ID:      id,
		Object:  "realtime.session.deleted",
		Deleted: true,
	}
}
