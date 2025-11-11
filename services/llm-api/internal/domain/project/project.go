package project

import (
	"context"
	"time"

	"jan-server/services/llm-api/internal/domain/query"
)

// ===============================================
// Project Types
// ===============================================

// Project represents a user's project that groups conversations and inherits instructions
type Project struct {
	ID          uint       `json:"-"`
	PublicID    string     `json:"id"`     // OpenAI-compatible string ID like "proj_abc123"
	Object      string     `json:"object"` // Always "project" for OpenAI compatibility
	UserID      uint       `json:"-"`      // Internal user ID
	Name        string     `json:"name"`
	Instruction *string    `json:"instruction,omitempty"` // Optional persona/context text
	Favorite    bool       `json:"favorite"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ===============================================
// Project Repository
// ===============================================

type ProjectFilter struct {
	ID       *uint
	PublicID *string
	UserID   *uint
	Archived *bool
	Search   *string
}

type ProjectRepository interface {
	Create(ctx context.Context, project *Project) error
	GetByPublicID(ctx context.Context, publicID string) (*Project, error)
	GetByPublicIDAndUserID(ctx context.Context, publicID string, userID uint) (*Project, error)
	ListByUserID(ctx context.Context, userID uint, pagination *query.Pagination) ([]*Project, int64, error)
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, publicID string) error
}

// ===============================================
// Project Factory
// ===============================================

// NewProject creates a new project with the given parameters
func NewProject(publicID string, userID uint, name string, instruction *string) *Project {
	now := time.Now()

	return &Project{
		PublicID:    publicID,
		Object:      "project",
		UserID:      userID,
		Name:        name,
		Instruction: instruction,
		Favorite:    false,
		ArchivedAt:  nil,
		DeletedAt:   nil,
		LastUsedAt:  nil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
