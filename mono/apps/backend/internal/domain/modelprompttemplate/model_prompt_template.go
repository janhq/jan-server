package modelprompttemplate

import (
	"context"
	"time"

	"jan-server/mono/apps/backend/internal/domain/prompttemplate"
)

// ModelPromptTemplate represents a model-specific prompt template assignment
// This allows each model catalog to have its own prompt templates that override global defaults
type ModelPromptTemplate struct {
	ID               string    `json:"id"`
	ModelCatalogID   string    `json:"model_catalog_id"`
	TemplateKey      string    `json:"template_key"`
	PromptTemplateID string    `json:"prompt_template_id"`
	Priority         int       `json:"priority"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	CreatedBy        *string   `json:"created_by,omitempty"`
	UpdatedBy        *string   `json:"updated_by,omitempty"`

	// Joined data - populated when fetching with template details
	PromptTemplate *prompttemplate.PromptTemplate `json:"prompt_template,omitempty"`
}

// AssignTemplateRequest contains fields for assigning a prompt template to a model
type AssignTemplateRequest struct {
	TemplateKey      string `json:"template_key" binding:"required"`
	PromptTemplateID string `json:"prompt_template_id" binding:"required"`
	Priority         *int   `json:"priority,omitempty"`
	IsActive         *bool  `json:"is_active,omitempty"`
}

// UpdateAssignmentRequest contains fields for updating an existing assignment
type UpdateAssignmentRequest struct {
	PromptTemplateID *string `json:"prompt_template_id,omitempty"`
	Priority         *int    `json:"priority,omitempty"`
	IsActive         *bool   `json:"is_active,omitempty"`
}

// ModelPromptTemplateFilter contains filter options for querying model prompt templates
type ModelPromptTemplateFilter struct {
	ModelCatalogID   *string
	TemplateKey      *string
	PromptTemplateID *string
	IsActive         *bool
}

// EffectiveTemplate represents a resolved template with its source information
type EffectiveTemplate struct {
	Template *prompttemplate.PromptTemplate `json:"template"`
	Source   string                         `json:"source"` // "model_specific", "global_default", "hardcoded"
}

// EffectiveTemplatesResponse contains resolved templates for a model
type EffectiveTemplatesResponse struct {
	Templates map[string]EffectiveTemplate `json:"templates"`
}

// ModelPromptTemplateRepository defines the interface for model prompt template data access
type ModelPromptTemplateRepository interface {
	// Create creates a new model prompt template assignment
	Create(ctx context.Context, mpt *ModelPromptTemplate) error

	// Update updates an existing model prompt template assignment
	Update(ctx context.Context, mpt *ModelPromptTemplate) error

	// Delete deletes a model prompt template assignment by model catalog ID and template key
	Delete(ctx context.Context, modelCatalogID, templateKey string) error

	// DeleteAllForModel deletes all prompt template assignments for a model catalog
	DeleteAllForModel(ctx context.Context, modelCatalogID string) error

	// FindByID finds a model prompt template by its ID
	FindByID(ctx context.Context, id string) (*ModelPromptTemplate, error)

	// FindByModelAndKey finds a model prompt template by model catalog ID and template key
	FindByModelAndKey(ctx context.Context, modelCatalogID, templateKey string) (*ModelPromptTemplate, error)

	// FindByModel finds all model prompt templates for a model catalog
	FindByModel(ctx context.Context, modelCatalogID string) ([]*ModelPromptTemplate, error)

	// FindByModelWithTemplates finds all model prompt templates for a model with joined template data
	FindByModelWithTemplates(ctx context.Context, modelCatalogID string) ([]*ModelPromptTemplate, error)

	// FindByFilter finds model prompt templates matching the given filter
	FindByFilter(ctx context.Context, filter ModelPromptTemplateFilter) ([]*ModelPromptTemplate, error)

	// Count returns the count of model prompt templates matching the given filter
	Count(ctx context.Context, filter ModelPromptTemplateFilter) (int64, error)
}
