package dbschema

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"

	"jan-server/mono/apps/backend/internal/domain/prompttemplate"
	"jan-server/mono/apps/backend/internal/infrastructure/database/registry"
)

func init() {
	registry.RegisterSchemaForAutoMigrate(PromptTemplate{})
}

// PromptTemplate represents the database schema for prompt templates
type PromptTemplate struct {
	ID          string         `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	PublicID    string         `gorm:"column:public_id;size:50;not null;uniqueIndex"`
	Name        string         `gorm:"column:name;size:255;not null"`
	Description *string        `gorm:"column:description;type:text"`
	Category    string         `gorm:"column:category;size:100;not null;index"`
	TemplateKey string         `gorm:"column:template_key;size:100;not null;uniqueIndex"`
	Content     string         `gorm:"column:content;type:text;not null"`
	Variables   datatypes.JSON `gorm:"column:variables;type:jsonb"`
	Metadata    datatypes.JSON `gorm:"column:metadata;type:jsonb"`
	IsActive    bool           `gorm:"column:is_active;default:true;index"`
	IsSystem    bool           `gorm:"column:is_system;default:false;index"`
	Version     int            `gorm:"column:version;default:1"`
	CreatedAt   time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;not null;default:now()"`
	CreatedBy   *string        `gorm:"column:created_by;type:uuid"`
	UpdatedBy   *string        `gorm:"column:updated_by;type:uuid"`
}

// TableName returns the table name for GORM
func (PromptTemplate) TableName() string {
	return "llm_api.prompt_templates"
}

// NewSchemaPromptTemplate converts a domain PromptTemplate to a database schema
func NewSchemaPromptTemplate(pt *prompttemplate.PromptTemplate) (*PromptTemplate, error) {
	var variablesJSON datatypes.JSON
	if len(pt.Variables) > 0 {
		data, err := json.Marshal(pt.Variables)
		if err != nil {
			return nil, err
		}
		variablesJSON = datatypes.JSON(data)
	}

	var metadataJSON datatypes.JSON
	if len(pt.Metadata) > 0 {
		data, err := json.Marshal(pt.Metadata)
		if err != nil {
			return nil, err
		}
		metadataJSON = datatypes.JSON(data)
	}

	return &PromptTemplate{
		ID:          pt.ID,
		PublicID:    pt.PublicID,
		Name:        pt.Name,
		Description: pt.Description,
		Category:    pt.Category,
		TemplateKey: pt.TemplateKey,
		Content:     pt.Content,
		Variables:   variablesJSON,
		Metadata:    metadataJSON,
		IsActive:    pt.IsActive,
		IsSystem:    pt.IsSystem,
		Version:     pt.Version,
		CreatedAt:   pt.CreatedAt,
		UpdatedAt:   pt.UpdatedAt,
		CreatedBy:   pt.CreatedBy,
		UpdatedBy:   pt.UpdatedBy,
	}, nil
}

// ToDomain converts a database schema PromptTemplate to a domain model
func (pt *PromptTemplate) ToDomain() (*prompttemplate.PromptTemplate, error) {
	var variables []string
	if len(pt.Variables) > 0 {
		if err := json.Unmarshal(pt.Variables, &variables); err != nil {
			return nil, err
		}
	}

	var metadata map[string]any
	if len(pt.Metadata) > 0 {
		if err := json.Unmarshal(pt.Metadata, &metadata); err != nil {
			return nil, err
		}
	}

	return &prompttemplate.PromptTemplate{
		ID:          pt.ID,
		PublicID:    pt.PublicID,
		Name:        pt.Name,
		Description: pt.Description,
		Category:    pt.Category,
		TemplateKey: pt.TemplateKey,
		Content:     pt.Content,
		Variables:   variables,
		Metadata:    metadata,
		IsActive:    pt.IsActive,
		IsSystem:    pt.IsSystem,
		Version:     pt.Version,
		CreatedAt:   pt.CreatedAt,
		UpdatedAt:   pt.UpdatedAt,
		CreatedBy:   pt.CreatedBy,
		UpdatedBy:   pt.UpdatedBy,
	}, nil
}
