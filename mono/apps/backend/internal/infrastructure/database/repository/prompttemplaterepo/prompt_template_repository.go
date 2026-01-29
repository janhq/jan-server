package prompttemplaterepo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"jan-server/mono/apps/backend/internal/domain/prompttemplate"
	"jan-server/mono/apps/backend/internal/domain/query"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"
	"jan-server/mono/apps/backend/internal/infrastructure/database/transaction"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// PromptTemplateGormRepository implements PromptTemplateRepository using GORM
type PromptTemplateGormRepository struct {
	db *transaction.Database
}

var _ prompttemplate.PromptTemplateRepository = (*PromptTemplateGormRepository)(nil)

// NewPromptTemplateGormRepository creates a new GORM-based prompt template repository
func NewPromptTemplateGormRepository(db *transaction.Database) prompttemplate.PromptTemplateRepository {
	return &PromptTemplateGormRepository{db: db}
}

// Create creates a new prompt template
func (r *PromptTemplateGormRepository) Create(ctx context.Context, template *prompttemplate.PromptTemplate) error {
	schema, err := dbschema.NewSchemaPromptTemplate(template)
	if err != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeValidation, "failed to convert template to schema", err, "a1b2c3d4-5678-9abc-def0-123456789abc")
	}

	tx := r.db.GetTx(ctx)
	if err := tx.Create(schema).Error; err != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to create prompt template", err, "b2c3d4e5-6789-abcd-ef01-23456789abcd")
	}

	template.ID = schema.ID
	template.CreatedAt = schema.CreatedAt
	template.UpdatedAt = schema.UpdatedAt

	return nil
}

// Update updates an existing prompt template
func (r *PromptTemplateGormRepository) Update(ctx context.Context, template *prompttemplate.PromptTemplate) error {
	schema, err := dbschema.NewSchemaPromptTemplate(template)
	if err != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeValidation, "failed to convert template to schema", err, "c3d4e5f6-789a-bcde-f012-3456789abcde")
	}

	schema.UpdatedAt = time.Now()

	tx := r.db.GetTx(ctx)
	result := tx.Model(&dbschema.PromptTemplate{}).
		Where("id = ?", schema.ID).
		Updates(map[string]interface{}{
			"name":        schema.Name,
			"description": schema.Description,
			"category":    schema.Category,
			"content":     schema.Content,
			"variables":   schema.Variables,
			"metadata":    schema.Metadata,
			"is_active":   schema.IsActive,
			"version":     schema.Version,
			"updated_at":  schema.UpdatedAt,
			"updated_by":  schema.UpdatedBy,
		})

	if result.Error != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to update prompt template", result.Error, "d4e5f6a7-89ab-cdef-0123-456789abcdef")
	}

	template.UpdatedAt = schema.UpdatedAt

	return nil
}

// Delete deletes a prompt template by ID
func (r *PromptTemplateGormRepository) Delete(ctx context.Context, id string) error {
	tx := r.db.GetTx(ctx)
	result := tx.Delete(&dbschema.PromptTemplate{}, "id = ?", id)
	if result.Error != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to delete prompt template", result.Error, "e5f6a7b8-9abc-def0-1234-56789abcdef0")
	}
	return nil
}

// FindByID finds a prompt template by its internal ID
func (r *PromptTemplateGormRepository) FindByID(ctx context.Context, id string) (*prompttemplate.PromptTemplate, error) {
	var schema dbschema.PromptTemplate
	tx := r.db.GetTx(ctx)
	if err := tx.Where("id = ?", id).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound, "prompt template not found", err, "f6a7b8c9-abcd-ef01-2345-6789abcdef01")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to find prompt template", err, "a7b8c9d0-bcde-f012-3456-789abcdef012")
	}

	return schema.ToDomain()
}

// FindByPublicID finds a prompt template by its public ID
func (r *PromptTemplateGormRepository) FindByPublicID(ctx context.Context, publicID string) (*prompttemplate.PromptTemplate, error) {
	var schema dbschema.PromptTemplate
	tx := r.db.GetTx(ctx)
	if err := tx.Where("public_id = ?", publicID).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound, "prompt template not found", err, "b8c9d0e1-cdef-0123-4567-89abcdef0123")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to find prompt template", err, "c9d0e1f2-def0-1234-5678-9abcdef01234")
	}

	return schema.ToDomain()
}

// FindByTemplateKey finds a prompt template by its unique template key
func (r *PromptTemplateGormRepository) FindByTemplateKey(ctx context.Context, templateKey string) (*prompttemplate.PromptTemplate, error) {
	var schema dbschema.PromptTemplate
	tx := r.db.GetTx(ctx)
	if err := tx.Where("template_key = ?", templateKey).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound, "prompt template not found", err, "d0e1f2a3-ef01-2345-6789-abcdef012345")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to find prompt template", err, "e1f2a3b4-f012-3456-789a-bcdef0123456")
	}

	return schema.ToDomain()
}

// FindByFilter finds prompt templates matching the given filter
func (r *PromptTemplateGormRepository) FindByFilter(ctx context.Context, filter prompttemplate.PromptTemplateFilter, p *query.Pagination) ([]*prompttemplate.PromptTemplate, error) {
	tx := r.db.GetTx(ctx)
	q := tx.Model(&dbschema.PromptTemplate{})

	// Apply filters
	if filter.ID != nil {
		q = q.Where("id = ?", *filter.ID)
	}
	if filter.PublicID != nil {
		q = q.Where("public_id = ?", *filter.PublicID)
	}
	if filter.TemplateKey != nil {
		q = q.Where("template_key = ?", *filter.TemplateKey)
	}
	if filter.Category != nil {
		q = q.Where("category = ?", *filter.Category)
	}
	if filter.IsActive != nil {
		q = q.Where("is_active = ?", *filter.IsActive)
	}
	if filter.IsSystem != nil {
		q = q.Where("is_system = ?", *filter.IsSystem)
	}
	if filter.Search != nil && *filter.Search != "" {
		searchPattern := "%" + *filter.Search + "%"
		q = q.Where("name ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	// Apply pagination
	if p != nil {
		if p.Limit != nil && *p.Limit > 0 {
			q = q.Limit(*p.Limit)
		}
		if p.Offset != nil && *p.Offset > 0 {
			q = q.Offset(*p.Offset)
		}
	}

	// Order by name
	q = q.Order("name ASC")

	var schemas []dbschema.PromptTemplate
	if err := q.Find(&schemas).Error; err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to find prompt templates", err, "f2a3b4c5-0123-4567-89ab-cdef01234567")
	}

	templates := make([]*prompttemplate.PromptTemplate, 0, len(schemas))
	for _, schema := range schemas {
		template, err := schema.ToDomain()
		if err != nil {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal, "failed to convert schema to domain", err, "a3b4c5d6-1234-5678-9abc-def012345678")
		}
		templates = append(templates, template)
	}

	return templates, nil
}

// Count returns the count of prompt templates matching the given filter
func (r *PromptTemplateGormRepository) Count(ctx context.Context, filter prompttemplate.PromptTemplateFilter) (int64, error) {
	tx := r.db.GetTx(ctx)
	q := tx.Model(&dbschema.PromptTemplate{})

	// Apply filters
	if filter.ID != nil {
		q = q.Where("id = ?", *filter.ID)
	}
	if filter.PublicID != nil {
		q = q.Where("public_id = ?", *filter.PublicID)
	}
	if filter.TemplateKey != nil {
		q = q.Where("template_key = ?", *filter.TemplateKey)
	}
	if filter.Category != nil {
		q = q.Where("category = ?", *filter.Category)
	}
	if filter.IsActive != nil {
		q = q.Where("is_active = ?", *filter.IsActive)
	}
	if filter.IsSystem != nil {
		q = q.Where("is_system = ?", *filter.IsSystem)
	}
	if filter.Search != nil && *filter.Search != "" {
		searchPattern := "%" + *filter.Search + "%"
		q = q.Where("name ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to count prompt templates", err, "b4c5d6e7-2345-6789-abcd-ef0123456789")
	}

	return count, nil
}
