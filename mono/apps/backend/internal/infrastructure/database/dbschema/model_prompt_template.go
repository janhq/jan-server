package dbschema

import (
	"time"

	"jan-server/mono/apps/backend/internal/domain/modelprompttemplate"
	"jan-server/mono/apps/backend/internal/infrastructure/database/registry"
)

func init() {
	registry.RegisterSchemaForAutoMigrate(ModelPromptTemplate{})
}

// ModelPromptTemplate represents the database schema for model-specific prompt template assignments
type ModelPromptTemplate struct {
	ID                   string    `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	ModelCatalogPublicID string    `gorm:"column:model_catalog_public_id;type:varchar(64);not null;index:idx_model_prompt_templates_model"`
	TemplateKey          string    `gorm:"column:template_key;size:100;not null;index:idx_model_prompt_templates_key"`
	PromptTemplateID     string    `gorm:"column:prompt_template_id;type:uuid;not null;index:idx_model_prompt_templates_template"`
	Priority             int       `gorm:"column:priority;default:0"`
	IsActive             bool      `gorm:"column:is_active;default:true;index:idx_model_prompt_templates_active"`
	CreatedAt            time.Time `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt            time.Time `gorm:"column:updated_at;not null;default:now()"`
	CreatedBy            *string   `gorm:"column:created_by;type:uuid"`
	UpdatedBy            *string   `gorm:"column:updated_by;type:uuid"`

	// Relations
	ModelCatalog   *ModelCatalog   `gorm:"foreignKey:ModelCatalogPublicID;references:PublicID"`
	PromptTemplate *PromptTemplate `gorm:"foreignKey:PromptTemplateID;references:ID"`
}

// TableName returns the table name for GORM
func (ModelPromptTemplate) TableName() string {
	return "llm_api.model_prompt_templates"
}

// NewSchemaModelPromptTemplate converts a domain ModelPromptTemplate to a database schema
func NewSchemaModelPromptTemplate(mpt *modelprompttemplate.ModelPromptTemplate) *ModelPromptTemplate {
	return &ModelPromptTemplate{
		ID:                   mpt.ID,
		ModelCatalogPublicID: mpt.ModelCatalogID,
		TemplateKey:          mpt.TemplateKey,
		PromptTemplateID:     mpt.PromptTemplateID,
		Priority:             mpt.Priority,
		IsActive:             mpt.IsActive,
		CreatedAt:            mpt.CreatedAt,
		UpdatedAt:            mpt.UpdatedAt,
		CreatedBy:            mpt.CreatedBy,
		UpdatedBy:            mpt.UpdatedBy,
	}
}

// ToDomain converts a database schema ModelPromptTemplate to a domain model
func (mpt *ModelPromptTemplate) ToDomain() *modelprompttemplate.ModelPromptTemplate {
	result := &modelprompttemplate.ModelPromptTemplate{
		ID:               mpt.ID,
		ModelCatalogID:   mpt.ModelCatalogPublicID,
		TemplateKey:      mpt.TemplateKey,
		PromptTemplateID: mpt.PromptTemplateID,
		Priority:         mpt.Priority,
		IsActive:         mpt.IsActive,
		CreatedAt:        mpt.CreatedAt,
		UpdatedAt:        mpt.UpdatedAt,
		CreatedBy:        mpt.CreatedBy,
		UpdatedBy:        mpt.UpdatedBy,
	}

	// Convert joined PromptTemplate if available
	if mpt.PromptTemplate != nil {
		pt, err := mpt.PromptTemplate.ToDomain()
		if err == nil {
			result.PromptTemplate = pt
		}
	}

	return result
}
