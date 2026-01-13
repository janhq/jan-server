package handlers

import (
	"github.com/rs/zerolog"

	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/plan"
	domain "jan-server/services/response-api/internal/domain/response"
	"jan-server/services/response-api/internal/infrastructure/apikey"
)

// Provider wires all HTTP handlers for dependency injection.
type Provider struct {
	Response *ResponseHandler
	Plan     *PlanHandler
	Artifact *ArtifactHandler
}

// NewProvider constructs the handler provider with domain services.
func NewProvider(responseService domain.Service, apiKeyProvider apikey.Provider, log zerolog.Logger) *Provider {
	return &Provider{
		Response: NewResponseHandler(responseService, apiKeyProvider, log),
	}
}

// NewProviderWithPlanAndArtifact constructs the handler provider with all services.
func NewProviderWithPlanAndArtifact(
	responseService domain.Service,
	planService plan.Service,
	artifactService artifact.Service,
	apiKeyProvider apikey.Provider,
	log zerolog.Logger,
) *Provider {
	return &Provider{
		Response: NewResponseHandler(responseService, apiKeyProvider, log),
		Plan:     NewPlanHandler(planService, log),
		Artifact: NewArtifactHandler(artifactService, log),
	}
}
