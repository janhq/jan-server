package handlers

import (
	"context"

	"jan-server/mono/apps/backend/internal/domain/session"
)

// SessionHandler handles session-related HTTP requests.
type SessionHandler struct {
	service session.Service
}

// NewSessionHandler creates a new session handler.
func NewSessionHandler(service session.Service) *SessionHandler {
	return &SessionHandler{service: service}
}

// CreateSession creates a new realtime session.
func (h *SessionHandler) CreateSession(ctx context.Context, req *session.CreateSessionRequest, userID string) (*session.Session, error) {
	return h.service.CreateSession(ctx, req, userID)
}

// GetSession retrieves a session by ID.
func (h *SessionHandler) GetSession(ctx context.Context, id string) (*session.Session, error) {
	return h.service.GetSession(ctx, id)
}

// ListUserSessions retrieves all sessions for a user.
func (h *SessionHandler) ListUserSessions(ctx context.Context, userID string) ([]*session.Session, error) {
	return h.service.ListUserSessions(ctx, userID)
}

// DeleteSession removes a session.
func (h *SessionHandler) DeleteSession(ctx context.Context, id string) error {
	return h.service.DeleteSession(ctx, id)
}
