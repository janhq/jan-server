package projectreq

// CreateProjectRequest represents the request to create a project
type CreateProjectRequest struct {
	Name        string  `json:"name" binding:"required"`
	Instruction *string `json:"instruction,omitempty"`
}

// UpdateProjectRequest represents the request to update a project
type UpdateProjectRequest struct {
	Name        *string `json:"name,omitempty"`
	Instruction *string `json:"instruction,omitempty"`
	Archived    *bool   `json:"is_archived,omitempty"`
	Favorite    *bool   `json:"is_favorite,omitempty"`
}
