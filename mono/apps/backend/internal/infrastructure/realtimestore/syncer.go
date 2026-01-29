package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"jan-server/mono/apps/backend/internal/domain/session"
	"jan-server/mono/apps/backend/internal/infrastructure/livekit"
)

// Syncer handles session synchronization with LiveKit.
// It polls LiveKit for active rooms and updates session state:
// - created → connected when room has participants
// - delete session when room is empty or removed
// - delete stale sessions that never connected (after staleTTL)
type Syncer struct {
	store      session.Store
	roomClient *livekit.RoomClient
	staleTTL   time.Duration
	interval   time.Duration
	log        zerolog.Logger
	done       chan struct{}
	wg         sync.WaitGroup
	startOnce  sync.Once
	stopOnce   sync.Once
}

// NewSyncer creates a new session syncer.
func NewSyncer(
	store session.Store,
	roomClient *livekit.RoomClient,
	staleTTL time.Duration,
	interval time.Duration,
	log zerolog.Logger,
) *Syncer {
	return &Syncer{
		store:      store,
		roomClient: roomClient,
		staleTTL:   staleTTL,
		interval:   interval,
		log:        log.With().Str("component", "session-syncer").Logger(),
		done:       make(chan struct{}),
	}
}

// Start begins the sync loop in background.
// Safe to call multiple times - only the first call starts the syncer.
func (s *Syncer) Start(ctx context.Context) {
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go s.run(ctx)
		s.log.Info().Msg("session syncer started")
	})
}

// Stop gracefully shuts down the syncer.
// Safe to call multiple times - only the first call stops the syncer.
func (s *Syncer) Stop() {
	s.stopOnce.Do(func() {
		close(s.done)
		s.wg.Wait()
		s.log.Info().Msg("session syncer stopped")
	})
}

func (s *Syncer) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.Debug().Msg("context cancelled, shutting down syncer")
			return
		case <-s.done:
			s.log.Debug().Msg("done signal received, shutting down syncer")
			return
		case <-ticker.C:
			s.sync(ctx)
		}
	}
}

// sync polls LiveKit and syncs session state.
func (s *Syncer) sync(ctx context.Context) {
	// Get all active rooms from LiveKit
	activeRooms, err := s.roomClient.ListActiveRooms(ctx)
	if err != nil {
		s.log.Warn().Err(err).Msg("failed to list rooms from LiveKit, falling back to TTL cleanup")
		s.cleanupByTTL(ctx)
		return
	}

	// Get all sessions from store
	sessions, err := s.store.List(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list sessions from store")
		return
	}

	// Build room lists for logging
	livekitRooms := make([]string, 0, len(activeRooms))
	for name, info := range activeRooms {
		livekitRooms = append(livekitRooms, fmt.Sprintf("%s(%d)", name, info.NumParticipants))
	}
	ourRooms := make([]string, 0, len(sessions))
	for _, sess := range sessions {
		ourRooms = append(ourRooms, fmt.Sprintf("%s(%s)", sess.Room, sess.State))
	}

	s.log.Info().
		Strs("livekit_rooms", livekitRooms).
		Strs("our_sessions", ourRooms).
		Msg("sync cycle")

	now := time.Now()

	// Update status and cleanup sessions
	for _, sess := range sessions {
		roomInfo, roomExists := activeRooms[sess.Room]

		switch {
		case !roomExists || roomInfo.NumParticipants == 0:
			// Room doesn't exist or is empty
			if sess.State == session.StateConnected {
				// Was connected, now room is gone → delete
				if err := s.store.Delete(ctx, sess.ID); err == nil {
					s.log.Info().
						Str("action", "deleted").
						Str("room", sess.Room).
						Str("reason", "room_empty").
						Msg("session cleanup")
				}
			} else if sess.State == session.StateCreated && now.Sub(sess.CreatedAt) > s.staleTTL {
				// Never connected and stale → delete
				if err := s.store.Delete(ctx, sess.ID); err == nil {
					s.log.Info().
						Str("action", "deleted").
						Str("room", sess.Room).
						Str("reason", "stale").
						Dur("age", now.Sub(sess.CreatedAt)).
						Msg("session cleanup")
				}
			}

		case roomInfo.NumParticipants > 0 && sess.State == session.StateCreated:
			// Room has participants, update state to connected
			if err := s.store.UpdateState(ctx, sess.ID, session.StateConnected); err == nil {
				s.log.Info().
					Str("action", "connected").
					Str("room", sess.Room).
					Int("participants", roomInfo.NumParticipants).
					Msg("session updated")
			}
		}
	}
}

// cleanupByTTL is a fallback when LiveKit is unreachable.
// Only cleans up stale sessions that never connected.
func (s *Syncer) cleanupByTTL(ctx context.Context) {
	sessions, err := s.store.List(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list sessions for TTL cleanup")
		return
	}

	now := time.Now()
	stale := 0

	for _, sess := range sessions {
		if sess.State == session.StateCreated && now.Sub(sess.CreatedAt) > s.staleTTL {
			if err := s.store.Delete(ctx, sess.ID); err == nil {
				stale++
			}
		}
	}

	if stale > 0 {
		s.log.Info().
			Int("stale_deleted", stale).
			Msg("TTL fallback cleanup completed")
	}
}
