package store

import (
	"context"
	"errors"
	"sync"

	"github.com/rs/zerolog"

	"jan-server/mono/apps/backend/internal/domain/session"
)

var (
	// ErrSessionNotFound is returned when a session is not found.
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionAlreadyExists is returned when trying to create a session that already exists.
	ErrSessionAlreadyExists = errors.New("session already exists")
	// ErrRoomAlreadyExists is returned when trying to create a session with a room that already exists.
	ErrRoomAlreadyExists = errors.New("room already exists")
)

// MemoryStore is a mutex-based in-memory session store.
// Thread-safe via sync.RWMutex.
type MemoryStore struct {
	mu        sync.RWMutex
	sessions  map[string]*session.Session
	roomIndex map[string]string // room -> session ID
	log       zerolog.Logger
}

// NewMemoryStore creates a new in-memory session store.
func NewMemoryStore(log zerolog.Logger) *MemoryStore {
	return &MemoryStore{
		sessions:  make(map[string]*session.Session),
		roomIndex: make(map[string]string),
		log:       log.With().Str("component", "session-store").Logger(),
	}
}

// Create stores a new session.
func (s *MemoryStore) Create(ctx context.Context, sess *session.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[sess.ID]; exists {
		return ErrSessionAlreadyExists
	}
	if _, exists := s.roomIndex[sess.Room]; exists {
		return ErrRoomAlreadyExists
	}

	s.sessions[sess.ID] = sess
	s.roomIndex[sess.Room] = sess.ID
	return nil
}

// Get retrieves a session by ID.
func (s *MemoryStore) Get(ctx context.Context, id string) (*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return sess, nil
}

// GetByUser retrieves all sessions for a user.
func (s *MemoryStore) GetByUser(ctx context.Context, userID string) ([]*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*session.Session
	for _, sess := range s.sessions {
		if sess.UserID == userID {
			result = append(result, sess)
		}
	}
	return result, nil
}

// GetByRoom retrieves a session by LiveKit room name.
func (s *MemoryStore) GetByRoom(ctx context.Context, room string) (*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionID, ok := s.roomIndex[room]
	if !ok {
		return nil, ErrSessionNotFound
	}
	sess, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return sess, nil
}

// Delete removes a session by ID.
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[id]
	if !ok {
		return ErrSessionNotFound
	}

	delete(s.roomIndex, sess.Room)
	delete(s.sessions, id)
	return nil
}

// List returns all sessions.
func (s *MemoryStore) List(ctx context.Context) ([]*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*session.Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		result = append(result, sess)
	}
	return result, nil
}

// UpdateState updates the state of a session.
func (s *MemoryStore) UpdateState(ctx context.Context, id string, state session.SessionState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[id]
	if !ok {
		return ErrSessionNotFound
	}
	sess.State = state
	return nil
}
