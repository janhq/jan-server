package dbschema

import (
	"time"

	"jan-server/services/llm-api/internal/domain/project"
	"jan-server/services/llm-api/internal/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(Project{})
}

// ===============================================
// Project Schema
// ===============================================

// Project represents the database schema for projects
type Project struct {
	BaseModel
	PublicID    string     `gorm:"uniqueIndex;size:64;not null"`
	UserID      uint       `gorm:"index:idx_projects_user;not null"`
	Name        string     `gorm:"size:255;not null"`
	Instruction *string    `gorm:"type:text"`
	Favorite    bool       `gorm:"not null;default:false"`
	ArchivedAt  *time.Time `gorm:"index"`
	DeletedAt   *time.Time `gorm:"index"`
	LastUsedAt  *time.Time
}

// TableName specifies the table name for Project
func (Project) TableName() string {
	return "llm_api.projects"
}

// ===============================================
// Conversion Methods
// ===============================================

// EtoD converts database schema to domain project (Entity to Domain)
func (p *Project) EtoD() *project.Project {
	return &project.Project{
		ID:          p.ID,
		PublicID:    p.PublicID,
		Object:      "project",
		UserID:      p.UserID,
		Name:        p.Name,
		Instruction: p.Instruction,
		Favorite:    p.Favorite,
		ArchivedAt:  p.ArchivedAt,
		DeletedAt:   p.DeletedAt,
		LastUsedAt:  p.LastUsedAt,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// DtoE converts domain project to database schema (Domain to Entity)
func ProjectDtoE(p *project.Project) *Project {
	return &Project{
		BaseModel: BaseModel{
			ID:        p.ID,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
		PublicID:    p.PublicID,
		UserID:      p.UserID,
		Name:        p.Name,
		Instruction: p.Instruction,
		Favorite:    p.Favorite,
		ArchivedAt:  p.ArchivedAt,
		DeletedAt:   p.DeletedAt,
		LastUsedAt:  p.LastUsedAt,
	}
}

// NewSchemaProject creates a database schema from domain project
func NewSchemaProject(p *project.Project) *Project {
	return &Project{
		BaseModel: BaseModel{
			ID:        p.ID,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
		PublicID:    p.PublicID,
		UserID:      p.UserID,
		Name:        p.Name,
		Instruction: p.Instruction,
		Favorite:    p.Favorite,
		ArchivedAt:  p.ArchivedAt,
		DeletedAt:   p.DeletedAt,
		LastUsedAt:  p.LastUsedAt,
	}
}
