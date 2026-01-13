// Package artifact defines artifact-related domain entities and services.
package artifact

import (
	"encoding/json"
	"time"
)

// Artifact represents a generated content item (slides, documents, code, etc.).
type Artifact struct {
	ID              string          `json:"id"`
	ResponseID      string          `json:"response_id"`
	PlanID          *string         `json:"plan_id,omitempty"`
	ContentType     ContentType     `json:"content_type"`
	MimeType        string          `json:"mime_type"`
	Title           string          `json:"title"`
	Content         *string         `json:"content,omitempty"`      // For inline content
	StoragePath     *string         `json:"storage_path,omitempty"` // For file-based content
	SizeBytes       int64           `json:"size_bytes"`
	Version         int             `json:"version"`
	ParentID        *string         `json:"parent_id,omitempty"` // For version chain
	IsLatest        bool            `json:"is_latest"`
	RetentionPolicy RetentionPolicy `json:"retention_policy"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`
}

// ContentType identifies the type of artifact content.
type ContentType string

const (
	ContentTypeSlides   ContentType = "slides"   // Slide deck (PPTX, Google Slides, etc.)
	ContentTypeDocument ContentType = "document" // Document (DOCX, PDF, Markdown)
	ContentTypeCode     ContentType = "code"     // Source code
	ContentTypeImage    ContentType = "image"    // Generated image
	ContentTypeChart    ContentType = "chart"    // Chart/graph
	ContentTypeTable    ContentType = "table"    // Tabular data
	ContentTypeResearch ContentType = "research" // Research report
	ContentTypeJSON     ContentType = "json"     // Structured JSON data
	ContentTypeHTML     ContentType = "html"     // HTML content
	ContentTypeMarkdown ContentType = "markdown" // Markdown content
)

// String returns the string representation of the content type.
func (c ContentType) String() string {
	return string(c)
}

// MimeTypeFor returns the default MIME type for the content type.
func (c ContentType) MimeTypeFor() string {
	switch c {
	case ContentTypeSlides:
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ContentTypeDocument:
		return "application/pdf"
	case ContentTypeCode:
		return "text/plain"
	case ContentTypeImage:
		return "image/png"
	case ContentTypeChart:
		return "image/svg+xml"
	case ContentTypeTable:
		return "text/csv"
	case ContentTypeResearch:
		return "text/markdown"
	case ContentTypeJSON:
		return "application/json"
	case ContentTypeHTML:
		return "text/html"
	case ContentTypeMarkdown:
		return "text/markdown"
	default:
		return "application/octet-stream"
	}
}

// RetentionPolicy determines how long an artifact is retained.
type RetentionPolicy string

const (
	RetentionEphemeral RetentionPolicy = "ephemeral"  // Deleted after session/request
	RetentionSession   RetentionPolicy = "session"    // Retained for session duration
	RetentionPermanent RetentionPolicy = "permanent"  // Retained indefinitely
	RetentionTimeBased RetentionPolicy = "time_based" // Retained until expires_at
)

// String returns the string representation of the retention policy.
func (r RetentionPolicy) String() string {
	return string(r)
}

// IsEphemeral returns true if the artifact should be auto-deleted.
func (r RetentionPolicy) IsEphemeral() bool {
	return r == RetentionEphemeral
}

// ArtifactMetadata contains common metadata fields.
type ArtifactMetadata struct {
	Author      *string  `json:"author,omitempty"`
	Description *string  `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	SourceURL   *string  `json:"source_url,omitempty"`
	ToolUsed    *string  `json:"tool_used,omitempty"`
	ModelUsed   *string  `json:"model_used,omitempty"`
	Checksum    *string  `json:"checksum,omitempty"` // SHA256
}

// SlidesMetadata contains metadata specific to slide artifacts.
type SlidesMetadata struct {
	ArtifactMetadata
	SlideCount      int     `json:"slide_count"`
	Theme           *string `json:"theme,omitempty"`
	AspectRatio     *string `json:"aspect_ratio,omitempty"` // "16:9", "4:3"
	HasSpeakerNotes bool    `json:"has_speaker_notes"`
}

// ResearchMetadata contains metadata specific to research artifacts.
type ResearchMetadata struct {
	ArtifactMetadata
	Sources     []ResearchSource `json:"sources,omitempty"`
	QueryUsed   *string          `json:"query_used,omitempty"`
	SearchDepth int              `json:"search_depth"`
	WordCount   int              `json:"word_count"`
}

// ResearchSource represents a source used in research.
type ResearchSource struct {
	URL        string    `json:"url"`
	Title      string    `json:"title"`
	Snippet    *string   `json:"snippet,omitempty"`
	AccessedAt time.Time `json:"accessed_at"`
}

// HasStoredContent returns true if the artifact has file-based content.
func (a *Artifact) HasStoredContent() bool {
	return a.StoragePath != nil && *a.StoragePath != ""
}

// HasInlineContent returns true if the artifact has inline content.
func (a *Artifact) HasInlineContent() bool {
	return a.Content != nil && *a.Content != ""
}

// IsExpired returns true if the artifact has expired.
func (a *Artifact) IsExpired() bool {
	if a.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*a.ExpiresAt)
}

// CreateNextVersion creates a new version of this artifact.
func (a *Artifact) CreateNextVersion() *Artifact {
	return &Artifact{
		ResponseID:      a.ResponseID,
		PlanID:          a.PlanID,
		ContentType:     a.ContentType,
		MimeType:        a.MimeType,
		Title:           a.Title,
		Version:         a.Version + 1,
		ParentID:        &a.ID,
		IsLatest:        true,
		RetentionPolicy: a.RetentionPolicy,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
}
