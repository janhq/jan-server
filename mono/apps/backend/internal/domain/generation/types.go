// Package generation provides types for content generation tools.
package generation

import "encoding/json"

// SlideGenerationRequest represents a request to generate slides.
type SlideGenerationRequest struct {
	Topic           string   `json:"topic"`
	SlideCount      *int     `json:"slide_count,omitempty"`       // Target number of slides (default: 10)
	Theme           *string  `json:"theme,omitempty"`             // Theme/style: "professional", "creative", "minimal"
	AspectRatio     *string  `json:"aspect_ratio,omitempty"`      // "16:9", "4:3"
	IncludeNotes    *bool    `json:"include_notes,omitempty"`     // Include speaker notes
	Outline         []string `json:"outline,omitempty"`           // Optional outline/structure
	SourceMaterial  *string  `json:"source_material,omitempty"`   // Reference content
	Audience        *string  `json:"audience,omitempty"`          // Target audience description
	Language        *string  `json:"language,omitempty"`          // Language (default: "en")
	MaxContentDepth *int     `json:"max_content_depth,omitempty"` // Level of detail (1-3)
}

// SlideGenerationResponse represents the result of slide generation.
type SlideGenerationResponse struct {
	Slides     []Slide         `json:"slides"`
	Outline    []string        `json:"outline"`
	Theme      string          `json:"theme"`
	Title      string          `json:"title"`
	Subtitle   *string         `json:"subtitle,omitempty"`
	ArtifactID *string         `json:"artifact_id,omitempty"` // If artifact was created
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// Slide represents a single slide in a presentation.
type Slide struct {
	Index        int          `json:"index"`
	Title        string       `json:"title"`
	Content      string       `json:"content"`
	Layout       SlideLayout  `json:"layout"`
	SpeakerNotes *string      `json:"speaker_notes,omitempty"`
	Images       []SlideImage `json:"images,omitempty"`
	BulletPoints []string     `json:"bullet_points,omitempty"`
}

// SlideLayout defines the layout type for a slide.
type SlideLayout string

const (
	LayoutTitle      SlideLayout = "title"
	LayoutContent    SlideLayout = "content"
	LayoutTwoColumn  SlideLayout = "two_column"
	LayoutImageLeft  SlideLayout = "image_left"
	LayoutImageRight SlideLayout = "image_right"
	LayoutImageFull  SlideLayout = "image_full"
	LayoutQuote      SlideLayout = "quote"
	LayoutBullets    SlideLayout = "bullets"
	LayoutConclusion SlideLayout = "conclusion"
)

// SlideImage represents an image reference in a slide.
type SlideImage struct {
	URL         string  `json:"url,omitempty"`
	Caption     *string `json:"caption,omitempty"`
	AltText     string  `json:"alt_text"`
	Placeholder bool    `json:"placeholder"` // True if image needs to be sourced
}

// DeepResearchRequest represents a request for deep research.
type DeepResearchRequest struct {
	Query            string   `json:"query"`
	MaxSources       *int     `json:"max_sources,omitempty"`       // Maximum sources to analyze (default: 10)
	SearchDepth      *int     `json:"search_depth,omitempty"`      // Depth of research (1-3)
	IncludeCitations *bool    `json:"include_citations,omitempty"` // Include source citations
	OutputFormat     *string  `json:"output_format,omitempty"`     // "summary", "detailed", "outline"
	FocusAreas       []string `json:"focus_areas,omitempty"`       // Specific areas to focus on
	ExcludeDomains   []string `json:"exclude_domains,omitempty"`   // Domains to exclude
	IncludeDomains   []string `json:"include_domains,omitempty"`   // Domains to prefer
	TimeRange        *string  `json:"time_range,omitempty"`        // "day", "week", "month", "year"
	Language         *string  `json:"language,omitempty"`          // Language preference
}

// DeepResearchResponse represents the result of deep research.
type DeepResearchResponse struct {
	Summary       string            `json:"summary"`
	Sections      []ResearchSection `json:"sections"`
	Sources       []ResearchSource  `json:"sources"`
	KeyFindings   []string          `json:"key_findings"`
	RelatedTopics []string          `json:"related_topics,omitempty"`
	WordCount     int               `json:"word_count"`
	SourceCount   int               `json:"source_count"`
	ArtifactID    *string           `json:"artifact_id,omitempty"`
	Metadata      json.RawMessage   `json:"metadata,omitempty"`
}

// ResearchSection represents a section in a research report.
type ResearchSection struct {
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	Sources     []int             `json:"sources"` // Indices into Sources array
	Subsections []ResearchSection `json:"subsections,omitempty"`
}

// ResearchSource represents a source used in research.
type ResearchSource struct {
	Index       int     `json:"index"`
	URL         string  `json:"url"`
	Title       string  `json:"title"`
	Domain      string  `json:"domain"`
	Snippet     *string `json:"snippet,omitempty"`
	PublishedAt *string `json:"published_at,omitempty"`
	Reliability *string `json:"reliability,omitempty"` // "high", "medium", "low"
}
