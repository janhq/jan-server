package session

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"jan-server/mono/apps/backend/internal/utils/idgen"
)

// TokenGenerator defines the interface for generating LiveKit tokens.
type TokenGenerator interface {
	Generate(room, identity string, ttl time.Duration) (token string, err error)
}

// Service defines the business operations for session management.
type Service interface {
	CreateSession(ctx context.Context, req *CreateSessionRequest, userID string) (*Session, error)
	GetSession(ctx context.Context, id string) (*Session, error)
	ListUserSessions(ctx context.Context, userID string) ([]*Session, error)
	DeleteSession(ctx context.Context, id string) error
}

type service struct {
	store    Store
	tokenGen TokenGenerator
	wsURL    string
	tokenTTL time.Duration
	log      zerolog.Logger
}

// NewService creates a new session service.
func NewService(store Store, tokenGen TokenGenerator, wsURL string, tokenTTL time.Duration, log zerolog.Logger) Service {
	return &service{
		store:    store,
		tokenGen: tokenGen,
		wsURL:    wsURL,
		tokenTTL: tokenTTL,
		log:      log.With().Str("component", "session-service").Logger(),
	}
}

func (s *service) CreateSession(ctx context.Context, req *CreateSessionRequest, userID string) (*Session, error) {
	sessionID, err := idgen.GenerateSecureID("sess", 24)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to generate session ID")
		return nil, err
	}

	roomID, err := idgen.GenerateSecureID("room", 24)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to generate room ID")
		return nil, err
	}

	identity := userID
	if identity == "" {
		identity, err = idgen.GenerateSecureID("user", 24)
		if err != nil {
			s.log.Error().Err(err).Msg("failed to generate user ID")
			return nil, err
		}
	}

	// Generate LiveKit token
	token, err := s.tokenGen.Generate(roomID, identity, s.tokenTTL)
	if err != nil {
		s.log.Error().Err(err).Str("session_id", sessionID).Msg("failed to generate token")
		return nil, err
	}

	now := time.Now()
	tokenExpiresAt := now.Add(s.tokenTTL)

	// Build session
	session := &Session{
		ID:     sessionID,
		Object: "realtime.session",
		ClientSecret: &ClientSecret{
			Value:     token,
			ExpiresAt: tokenExpiresAt.Unix(),
		},
		WsURL:     s.wsURL,
		RoomID:    roomID,
		UserID:    userID,
		Room:      roomID, // internal tracking
		State:     StateCreated,
		CreatedAt: now,
	}

	if err := s.store.Create(ctx, session); err != nil {
		s.log.Error().Err(err).Str("session_id", sessionID).Msg("failed to store session")
		return nil, err
	}

	s.log.Info().
		Str("session_id", sessionID).
		Str("user_id", userID).
		Str("room_id", roomID).
		Str("state", string(StateCreated)).
		Msg("session created")

	return session, nil
}

func (s *service) GetSession(ctx context.Context, id string) (*Session, error) {
	return s.store.Get(ctx, id)
}

func (s *service) ListUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	return s.store.GetByUser(ctx, userID)
}

func (s *service) DeleteSession(ctx context.Context, id string) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}
	s.log.Info().Str("session_id", id).Msg("session deleted")
	return nil
}
