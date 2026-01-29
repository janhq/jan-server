package responses

import (
	"jan-server/mono/apps/backend/internal/domain/plan"
	"jan-server/mono/apps/backend/internal/domain/response"
)

// ArtifactPayload represents an artifact in API responses.
type ArtifactPayload struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Filename    string `json:"filename"`
	DownloadURL string `json:"download_url"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

// CitationPayload represents a citation in API responses.
type CitationPayload struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Snippet   string `json:"snippet,omitempty"`
	FetchedAt string `json:"fetched_at,omitempty"`
}

// ResponsePayload is returned to clients.
type ResponsePayload struct {
	ID                 string                    `json:"id"`
	Object             string                    `json:"object"`
	Created            int64                     `json:"created"`
	CreatedAt          int64                     `json:"created_at"` // Same as Created, for compatibility
	Model              string                    `json:"model"`
	Status             string                    `json:"status"`
	Input              interface{}               `json:"input"`
	Output             interface{}               `json:"output,omitempty"`
	Usage              interface{}               `json:"usage,omitempty"`
	Metadata           map[string]interface{}    `json:"metadata,omitempty"`
	ConversationID     *string                   `json:"conversation_id,omitempty"`
	PreviousResponseID *string                   `json:"previous_response_id,omitempty"`
	SystemPrompt       *string                   `json:"system_prompt,omitempty"`
	Stream             bool                      `json:"stream"`
	Background         bool                      `json:"background"`
	Store              bool                      `json:"store"`
	Error              interface{}               `json:"error,omitempty"`
	Artifacts          []ArtifactPayload         `json:"artifacts,omitempty"`
	Citations          []CitationPayload         `json:"citations,omitempty"`
	Data               []ConversationItemPayload `json:"data,omitempty"` // All conversation items when data=true
}

// ConversationItemPayload represents a conversation item in the response.
type ConversationItemPayload struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
	Status  string      `json:"status,omitempty"`
}

// FromDomain maps the domain response to DTO.
func FromDomain(r *response.Response) ResponsePayload {
	createdUnix := r.CreatedAt.Unix()
	payload := ResponsePayload{
		ID:                 r.PublicID,
		Object:             r.Object,
		Created:            createdUnix,
		CreatedAt:          createdUnix, // Duplicate for compatibility
		Model:              r.Model,
		Status:             string(r.Status),
		Input:              r.Input,
		Output:             r.Output,
		Usage:              r.Usage,
		Metadata:           r.Metadata,
		ConversationID:     r.ConversationPublicID,
		PreviousResponseID: r.PreviousResponseID,
		SystemPrompt:       r.SystemPrompt,
		Stream:             r.Stream,
		Background:         r.Background,
		Store:              r.Store,
		Error:              r.Error,
	}

	// Map artifacts
	if len(r.Artifacts) > 0 {
		payload.Artifacts = MapArtifactsToPayload(r.Artifacts)
	}

	// Map citations
	if len(r.Citations) > 0 {
		payload.Citations = MapCitationsToPayload(r.Citations)
	}

	return payload
}

// MapArtifactsToPayload converts domain artifacts to payload format.
func MapArtifactsToPayload(artifacts []plan.MediaArtifact) []ArtifactPayload {
	result := make([]ArtifactPayload, 0, len(artifacts))
	for _, a := range artifacts {
		result = append(result, ArtifactPayload{
			ID:          a.ID,
			Type:        a.Type,
			Filename:    a.Filename,
			DownloadURL: a.DownloadURL,
			Size:        a.Size,
			ContentType: a.ContentType,
		})
	}
	return result
}

// MapCitationsToPayload converts domain citations to payload format.
func MapCitationsToPayload(citations []plan.Citation) []CitationPayload {
	result := make([]CitationPayload, 0, len(citations))
	for _, c := range citations {
		result = append(result, CitationPayload{
			Title:     c.Title,
			URL:       c.URL,
			Snippet:   c.Snippet,
			FetchedAt: c.FetchedAt,
		})
	}
	return result
}

// ConversationItemsResponse wraps conversation input items for consistent responses.
type ConversationItemsResponse struct {
	Data []response.ConversationItem `json:"data"`
}

// FullResponsePayload combines response with plan details in one payload.
type FullResponsePayload struct {
	ResponsePayload
	Plan *PlanDetailResponse `json:"plan,omitempty"`
}

// FromDomainWithPlan maps the domain response with plan to DTO.
func FromDomainWithPlan(rwp *response.ResponseWithPlan, planMapper func(interface{}) *PlanDetailResponse) FullResponsePayload {
	payload := FullResponsePayload{
		ResponsePayload: FromDomain(rwp.Response),
	}
	if rwp.PlanDetails != nil && planMapper != nil {
		payload.Plan = planMapper(rwp.PlanDetails)
	}
	return payload
}
