package modelprompttemplaterepo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"jan-server/mono/apps/backend/internal/domain/modelprompttemplate"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"
	"jan-server/mono/apps/backend/internal/infrastructure/database/transaction"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// ModelPromptTemplateGormRepository implements ModelPromptTemplateRepository using GORM
type ModelPromptTemplateGormRepository struct {
	db *transaction.Database
}

var _ modelprompttemplate.ModelPromptTemplateRepository = (*ModelPromptTemplateGormRepository)(nil)

// NewModelPromptTemplateGormRepository creates a new GORM-based model prompt template repository
func NewModelPromptTemplateGormRepository(db *transaction.Database) modelprompttemplate.ModelPromptTemplateRepository {
	return &ModelPromptTemplateGormRepository{db: db}
}

// Create creates a new model prompt template assignment
func (r *ModelPromptTemplateGormRepository) Create(ctx context.Context, mpt *modelprompttemplate.ModelPromptTemplate) error {
	schema := dbschema.NewSchemaModelPromptTemplate(mpt)

	tx := r.db.GetTx(ctx)
	if err := tx.Create(schema).Error; err != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError,
			"failed to create model prompt template", err, "mpt-create-001")
	}

	mpt.ID = schema.ID
	mpt.CreatedAt = schema.CreatedAt
	mpt.UpdatedAt = schema.UpdatedAt

	return nil
}

// Update updates an existing model prompt template assignment
func (r *ModelPromptTemplateGormRepository) Update(ctx context.Context, mpt *modelprompttemplate.ModelPromptTemplate) error {
	schema := dbschema.NewSchemaModelPromptTemplate(mpt)
	schema.UpdatedAt = time.Now()

	tx := r.db.GetTx(ctx)
	result := tx.Model(&dbschema.ModelPromptTemplate{}).
		Where("id = ?", schema.ID).
		Updates(map[string]interface{}{
			"prompt_template_id": schema.PromptTemplateID,
			"priority":           schema.Priority,
			"is_active":          schema.IsActive,
			"updated_at":         schema.UpdatedAt,
			"updated_by":         schema.UpdatedBy,
		})

	if result.Error != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError,
			"failed to update model prompt template", result.Error, "mpt-update-001")
	}

	mpt.UpdatedAt = schema.UpdatedAt
	return nil
}

// Delete deletes a model prompt template assignment by model catalog ID and template key
func (r *ModelPromptTemplateGormRepository) Delete(ctx context.Context, modelCatalogID, templateKey string) error {
	tx := r.db.GetTx(ctx)
	result := tx.Where("model_catalog_public_id = ? AND template_key = ?", modelCatalogID, templateKey).
		Delete(&dbschema.ModelPromptTemplate{})

	if result.Error != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError,
			"failed to delete model prompt template", result.Error, "mpt-delete-001")
	}

	if result.RowsAffected == 0 {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound,
			"model prompt template not found", nil, "mpt-delete-002")
	}

	return nil
}

// DeleteAllForModel deletes all prompt template assignments for a model catalog
func (r *ModelPromptTemplateGormRepository) DeleteAllForModel(ctx context.Context, modelCatalogID string) error {
	tx := r.db.GetTx(ctx)
	result := tx.Where("model_catalog_public_id = ?", modelCatalogID).Delete(&dbschema.ModelPromptTemplate{})

	if result.Error != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError,
			"failed to delete model prompt templates", result.Error, "mpt-delete-all-001")
	}

	return nil
}

// FindByID finds a model prompt template by its ID
func (r *ModelPromptTemplateGormRepository) FindByID(ctx context.Context, id string) (*modelprompttemplate.ModelPromptTemplate, error) {
	var schema dbschema.ModelPromptTemplate
	tx := r.db.GetTx(ctx)

	if err := tx.Where("id = ?", id).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound,
				"model prompt template not found", err, "mpt-find-id-001")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError,
			"failed to find model prompt template", err, "mpt-find-id-002")
	}

	return schema.ToDomain(), nil
}

// FindByModelAndKey finds a model prompt template by model catalog ID and template key
func (r *ModelPromptTemplateGormRepository) FindByModelAndKey(ctx context.Context, modelCatalogID, templateKey string) (*modelprompttemplate.ModelPromptTemplate, error) {
	var schema dbschema.ModelPromptTemplate
	tx := r.db.GetTx(ctx)

	if err := tx.Where("model_catalog_public_id = ? AND template_key = ?", modelCatalogID, templateKey).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound,
				"model prompt template not found", err, "mpt-find-mk-001")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError,
			"failed to find model prompt template", err, "mpt-find-mk-002")
	}

	return schema.ToDomain(), nil
}

// FindByModel finds all model prompt templates for a model catalog
func (r *ModelPromptTemplateGormRepository) FindByModel(ctx context.Context, modelCatalogID string) ([]*modelprompttemplate.ModelPromptTemplate, error) {
	var schemas []dbschema.ModelPromptTemplate
	tx := r.db.GetTx(ctx)

	if err := tx.Where("model_catalog_public_id = ?", modelCatalogID).Order("template_key ASC").Find(&schemas).Error; err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError,
			"failed to find model prompt templates", err, "mpt-find-model-001")
	}

	result := make([]*modelprompttemplate.ModelPromptTemplate, len(schemas))
	for i, schema := range schemas {
		result[i] = schema.ToDomain()
	}

	return result, nil
}

// FindByModelWithTemplates finds all model prompt templates for a model with joined template data
func (r *ModelPromptTemplateGormRepository) FindByModelWithTemplates(ctx context.Context, modelCatalogID string) ([]*modelprompttemplate.ModelPromptTemplate, error) {
	var schemas []dbschema.ModelPromptTemplate
	tx := r.db.GetTx(ctx)

	if err := tx.Preload("PromptTemplate").
		Where("model_catalog_public_id = ?", modelCatalogID).
		Order("template_key ASC").
		Find(&schemas).Error; err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError,
			"failed to find model prompt templates with templates", err, "mpt-find-model-tmpl-001")
	}

	result := make([]*modelprompttemplate.ModelPromptTemplate, len(schemas))
	for i, schema := range schemas {
		result[i] = schema.ToDomain()
	}

	return result, nil
}

// FindByFilter finds model prompt templates matching the given filter
func (r *ModelPromptTemplateGormRepository) FindByFilter(ctx context.Context, filter modelprompttemplate.ModelPromptTemplateFilter) ([]*modelprompttemplate.ModelPromptTemplate, error) {
	tx := r.db.GetTx(ctx)
	q := tx.Model(&dbschema.ModelPromptTemplate{})

	if filter.ModelCatalogID != nil {
		q = q.Where("model_catalog_public_id = ?", *filter.ModelCatalogID)
	}
	if filter.TemplateKey != nil {
		q = q.Where("template_key = ?", *filter.TemplateKey)
	}
	if filter.PromptTemplateID != nil {
		q = q.Where("prompt_template_id = ?", *filter.PromptTemplateID)
	}
	if filter.IsActive != nil {
		q = q.Where("is_active = ?", *filter.IsActive)
	}

	q = q.Order("template_key ASC")

	var schemas []dbschema.ModelPromptTemplate
	if err := q.Find(&schemas).Error; err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError,
			"failed to find model prompt templates by filter", err, "mpt-find-filter-001")
	}

	result := make([]*modelprompttemplate.ModelPromptTemplate, len(schemas))
	for i, schema := range schemas {
		result[i] = schema.ToDomain()
	}

	return result, nil
}

// Count returns the count of model prompt templates matching the given filter
func (r *ModelPromptTemplateGormRepository) Count(ctx context.Context, filter modelprompttemplate.ModelPromptTemplateFilter) (int64, error) {
	tx := r.db.GetTx(ctx)
	q := tx.Model(&dbschema.ModelPromptTemplate{})

	if filter.ModelCatalogID != nil {
		q = q.Where("model_catalog_public_id = ?", *filter.ModelCatalogID)
	}
	if filter.TemplateKey != nil {
		q = q.Where("template_key = ?", *filter.TemplateKey)
	}
	if filter.PromptTemplateID != nil {
		q = q.Where("prompt_template_id = ?", *filter.PromptTemplateID)
	}
	if filter.IsActive != nil {
		q = q.Where("is_active = ?", *filter.IsActive)
	}

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError,
			"failed to count model prompt templates", err, "mpt-count-001")
	}

	return count, nil
}
