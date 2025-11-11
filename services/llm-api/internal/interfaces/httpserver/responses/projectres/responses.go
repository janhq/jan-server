package projectres

import (
	"jan-server/services/llm-api/internal/domain/project"
)

// ProjectResponse represents a single project response
type ProjectResponse struct {
	ID          string  `json:"id"`
	Object      string  `json:"object"`
	Name        string  `json:"name"`
	Instruction *string `json:"instruction,omitempty"`
	Favorite    bool    `json:"is_favorite"`
	IsArchived  bool    `json:"is_archived"`
	ArchivedAt  *int64  `json:"archived_at,omitempty"`
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   int64   `json:"updated_at"`
}

// ProjectListResponse represents a paginated list of projects
type ProjectListResponse struct {
	Object  string            `json:"object"`
	Data    []ProjectResponse `json:"data"`
	FirstID string            `json:"first_id,omitempty"`
	LastID  string            `json:"last_id,omitempty"`
	NextCursor *string        `json:"next_cursor,omitempty"`
	HasMore bool              `json:"has_more"`
	Total   int64             `json:"total"`
}

// ProjectDeletedResponse represents the delete confirmation response
type ProjectDeletedResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

// NewProjectResponse creates a response from a domain project
func NewProjectResponse(proj *project.Project) *ProjectResponse {
	resp := &ProjectResponse{
		ID:          proj.PublicID,
		Object:      "project",
		Name:        proj.Name,
		Instruction: proj.Instruction,
		Favorite:    proj.Favorite,
		IsArchived:  proj.ArchivedAt != nil,
		CreatedAt:   proj.CreatedAt.Unix(),
		UpdatedAt:   proj.UpdatedAt.Unix(),
	}

	if proj.ArchivedAt != nil {
		archivedUnix := proj.ArchivedAt.Unix()
		resp.ArchivedAt = &archivedUnix
	}

	return resp
}

// NewProjectListResponse creates a list response from domain projects
func NewProjectListResponse(projects []*project.Project, hasMore bool, nextCursor *string, total int64) *ProjectListResponse {
	data := make([]ProjectResponse, len(projects))
	for i, proj := range projects {
		data[i] = *NewProjectResponse(proj)
	}

	resp := &ProjectListResponse{
		Object:  "list",
		Data:    data,
		HasMore: hasMore,
		Total:   total,
		NextCursor: nextCursor,
	}

	if len(data) > 0 {
		resp.FirstID = data[0].ID
		resp.LastID = data[len(data)-1].ID
	}

	return resp
}

// NewProjectDeletedResponse creates a delete response
func NewProjectDeletedResponse(publicID string) *ProjectDeletedResponse {
	return &ProjectDeletedResponse{
		ID:      publicID,
		Object:  "project",
		Deleted: true,
	}
}
