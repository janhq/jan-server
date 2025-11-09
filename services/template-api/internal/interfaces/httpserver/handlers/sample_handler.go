package handlers

import (
	"context"

	domain "jan-server/services/template-api/internal/domain/sample"
)

// SampleHandler invokes domain logic for sample use cases.
type SampleHandler struct {
	service domain.Service
}

// NewSampleHandler wires dependencies for sample routes.
func NewSampleHandler(service domain.Service) *SampleHandler {
	return &SampleHandler{
		service: service,
	}
}

// GetSample executes the domain use case and returns the response.
func (h *SampleHandler) GetSample(ctx context.Context) (domain.Sample, error) {
	return h.service.GetSample(ctx)
}
