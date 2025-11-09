package handlers

import (
	domain "jan-server/services/template-api/internal/domain/sample"
)

// Provider wires all HTTP handlers for dependency injection.
type Provider struct {
	Sample *SampleHandler
}

// NewProvider constructs the handler provider with domain services.
func NewProvider(sampleService domain.Service) *Provider {
	return &Provider{
		Sample: NewSampleHandler(sampleService),
	}
}
