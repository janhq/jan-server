package artifact

import (
	"time"
)

// Artifact represents a code artifact.
type Artifact struct {
	ID             string
	UserID         string
	ConversationID *string
	ResponseID     *string
	Title          string
	Description    string
	Type           string // code, html, svg, markdown, mermaid
	Language       string
	Content        string
	Version        int
	Metadata       map[string]any
	IsPublic       bool
	ShareToken     *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ArtifactVersion stores a previous version of an artifact.
type ArtifactVersion struct {
	ID         string
	ArtifactID string
	Version    int
	Content    string
	CreatedAt  time.Time
}

// CreateArtifactRequest contains data for creating an artifact.
type CreateArtifactRequest struct {
	ConversationID *string
	ResponseID     *string
	Title          string
	Description    string
	Type           string
	Language       string
	Content        string
	Metadata       map[string]any
	IsPublic       bool
}

// UpdateArtifactRequest contains data for updating an artifact.
type UpdateArtifactRequest struct {
	Title       *string
	Description *string
	Content     *string
	Language    *string
	IsPublic    *bool
	Metadata    map[string]any
}

// ListArtifactsFilter contains filters for listing artifacts.
type ListArtifactsFilter struct {
	UserID         string
	ConversationID *string
	Type           string
	Search         string
	Limit          int
	Offset         int
}

// ArtifactResponse is the API response for an artifact.
type ArtifactResponse struct {
	ID             string         `json:"id"`
	Title          string         `json:"title"`
	Description    string         `json:"description,omitempty"`
	Type           string         `json:"type"`
	Language       string         `json:"language,omitempty"`
	Content        string         `json:"content"`
	Version        int            `json:"version"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	IsPublic       bool           `json:"is_public"`
	ShareToken     *string        `json:"share_token,omitempty"`
	ConversationID *string        `json:"conversation_id,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// ArtifactVersionResponse is the API response for an artifact version.
type ArtifactVersionResponse struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse converts an Artifact to ArtifactResponse.
func (a *Artifact) ToResponse() ArtifactResponse {
	return ArtifactResponse{
		ID:             a.ID,
		Title:          a.Title,
		Description:    a.Description,
		Type:           a.Type,
		Language:       a.Language,
		Content:        a.Content,
		Version:        a.Version,
		Metadata:       a.Metadata,
		IsPublic:       a.IsPublic,
		ShareToken:     a.ShareToken,
		ConversationID: a.ConversationID,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}

// ToResponse converts an ArtifactVersion to ArtifactVersionResponse.
func (v *ArtifactVersion) ToResponse() ArtifactVersionResponse {
	return ArtifactVersionResponse{
		ID:        v.ID,
		Version:   v.Version,
		Content:   v.Content,
		CreatedAt: v.CreatedAt,
	}
}
