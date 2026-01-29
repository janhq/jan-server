package session

import "context"

// Store defines the interface for session storage.
// This interface is storage-only - no lifecycle methods.
// Sync/cleanup logic lives in the Syncer component.
type Store interface {
	// Create stores a new session.
	Create(ctx context.Context, session *Session) error

	// Get retrieves a session by ID.
	Get(ctx context.Context, id string) (*Session, error)

	// GetByUser retrieves all sessions for a user.
	GetByUser(ctx context.Context, userID string) ([]*Session, error)

	// GetByRoom retrieves a session by LiveKit room name.
	GetByRoom(ctx context.Context, room string) (*Session, error)

	// Delete removes a session by ID.
	Delete(ctx context.Context, id string) error

	// List returns all sessions (for sync/cleanup iteration).
	List(ctx context.Context) ([]*Session, error)

	// UpdateState updates the state of a session.
	UpdateState(ctx context.Context, id string, state SessionState) error
}
