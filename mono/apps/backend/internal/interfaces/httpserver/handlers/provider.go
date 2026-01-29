package handlers

import (
	"github.com/google/wire"
	"github.com/rs/zerolog"

	"jan-server/mono/apps/backend/internal/config"
	"jan-server/mono/apps/backend/internal/domain/media"
	"jan-server/mono/apps/backend/internal/domain/session"
)

// Provider holds all HTTP handlers.
type Provider struct {
	Session *SessionHandler
	Media   *MediaHandler
}

// NewProvider creates a new handler provider.
func NewProvider(
	sessionService session.Service,
	cfg *config.Config,
	mediaService *media.Service,
	logger zerolog.Logger,
) *Provider {
	return &Provider{
		Session: NewSessionHandler(sessionService),
		Media:   NewMediaHandler(cfg, mediaService, logger),
	}
}

// SessionHandlerProvider provides session handlers for wire.
var SessionHandlerProvider = wire.NewSet(
	NewSessionHandler,
	NewMediaHandler,
	NewProvider,
)
