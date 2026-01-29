package generation

import (
	"context"
)

// GenerationClient defines the generation operations required by the domain layer.
type GenerationClient interface {
	GenerateSlides(ctx context.Context, req SlideGenerationRequest) (*SlideGenerationResponse, error)
	DeepResearch(ctx context.Context, req DeepResearchRequest) (*DeepResearchResponse, error)
}

// GenerationService orchestrates content generation operations.
type GenerationService struct {
	client GenerationClient
}

// NewGenerationService creates a new generation service.
func NewGenerationService(client GenerationClient) *GenerationService {
	return &GenerationService{
		client: client,
	}
}

// GenerateSlides creates a slide presentation based on the request.
func (s *GenerationService) GenerateSlides(ctx context.Context, req SlideGenerationRequest) (*SlideGenerationResponse, error) {
	// Apply defaults
	if req.SlideCount == nil {
		defaultCount := 10
		req.SlideCount = &defaultCount
	}
	if req.Theme == nil {
		defaultTheme := "professional"
		req.Theme = &defaultTheme
	}
	if req.AspectRatio == nil {
		defaultRatio := "16:9"
		req.AspectRatio = &defaultRatio
	}
	if req.IncludeNotes == nil {
		defaultNotes := true
		req.IncludeNotes = &defaultNotes
	}
	if req.Language == nil {
		defaultLang := "en"
		req.Language = &defaultLang
	}

	return s.client.GenerateSlides(ctx, req)
}

// DeepResearch performs deep research on a topic.
func (s *GenerationService) DeepResearch(ctx context.Context, req DeepResearchRequest) (*DeepResearchResponse, error) {
	// Apply defaults
	if req.MaxSources == nil {
		defaultMax := 10
		req.MaxSources = &defaultMax
	}
	if req.SearchDepth == nil {
		defaultDepth := 2
		req.SearchDepth = &defaultDepth
	}
	if req.IncludeCitations == nil {
		defaultCitations := true
		req.IncludeCitations = &defaultCitations
	}
	if req.OutputFormat == nil {
		defaultFormat := "detailed"
		req.OutputFormat = &defaultFormat
	}
	if req.Language == nil {
		defaultLang := "en"
		req.Language = &defaultLang
	}

	return s.client.DeepResearch(ctx, req)
}
