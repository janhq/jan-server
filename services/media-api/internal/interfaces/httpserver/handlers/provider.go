package handlers

import (
	"github.com/rs/zerolog"

	"jan-server/services/media-api/internal/config"
	domain "jan-server/services/media-api/internal/domain/media"
)

// Provider wires HTTP handlers.
type Provider struct {
	Media *MediaHandler
}

func NewProvider(cfg *config.Config, service *domain.Service, log zerolog.Logger) *Provider {
	return &Provider{
		Media: NewMediaHandler(cfg, service, log),
	}
}
