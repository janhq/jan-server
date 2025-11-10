package handlers

import (
	"github.com/rs/zerolog"

	domain "jan-server/services/response-api/internal/domain/response"
)

// Provider wires all HTTP handlers for dependency injection.
type Provider struct {
	Response *ResponseHandler
}

// NewProvider constructs the handler provider with domain services.
func NewProvider(responseService domain.Service, log zerolog.Logger) *Provider {
	return &Provider{
		Response: NewResponseHandler(responseService, log),
	}
}
