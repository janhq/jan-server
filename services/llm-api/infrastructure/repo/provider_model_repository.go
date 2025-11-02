package repo

import (
	"context"
	"errors"

	"jan-server/services/llm-api/domain/model"
	"jan-server/services/llm-api/domain/query"
	"jan-server/services/llm-api/infrastructure/db/dbschema"

	"gorm.io/gorm"
)

type ProviderModelRepository struct {
	db *gorm.DB
}

func NewProviderModelRepository(db *gorm.DB) *ProviderModelRepository {
	return &ProviderModelRepository{db: db}
}

func (r *ProviderModelRepository) Create(ctx context.Context, m *model.ProviderModel) error {
	dbModel, err := dbschema.NewSchemaProviderModel(m)
	if err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Create(dbModel).Error; err != nil {
		return err
	}
	converted, err := dbModel.EtoD()
	if err != nil {
		return err
	}
	*m = *converted
	return nil
}

func (r *ProviderModelRepository) Update(ctx context.Context, m *model.ProviderModel) error {
	dbModel, err := dbschema.NewSchemaProviderModel(m)
	if err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Save(dbModel).Error; err != nil {
		return err
	}
	converted, err := dbModel.EtoD()
	if err != nil {
		return err
	}
	*m = *converted
	return nil
}

func (r *ProviderModelRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&dbschema.ProviderModel{}, id).Error
}

func (r *ProviderModelRepository) FindByID(ctx context.Context, id uint) (*model.ProviderModel, error) {
	var dbModel dbschema.ProviderModel
	if err := r.db.WithContext(ctx).First(&dbModel, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return dbModel.EtoD()
}

func (r *ProviderModelRepository) FindByPublicID(ctx context.Context, publicID string) (*model.ProviderModel, error) {
	var dbModel dbschema.ProviderModel
	if err := r.db.WithContext(ctx).Where("public_id = ?", publicID).First(&dbModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return dbModel.EtoD()
}

func (r *ProviderModelRepository) FindByFilter(ctx context.Context, filter model.ProviderModelFilter, p *query.Pagination) ([]*model.ProviderModel, error) {
	var dbModels []dbschema.ProviderModel
	q := r.db.WithContext(ctx)

	if filter.IDs != nil {
		q = q.Where("id IN ?", *filter.IDs)
	}
	if filter.PublicID != nil {
		q = q.Where("public_id = ?", *filter.PublicID)
	}
	if filter.ProviderIDs != nil {
		q = q.Where("provider_id IN ?", *filter.ProviderIDs)
	}
	if filter.ProviderID != nil {
		q = q.Where("provider_id = ?", *filter.ProviderID)
	}
	if filter.ModelCatalogID != nil {
		q = q.Where("model_catalog_id = ?", *filter.ModelCatalogID)
	}
	if filter.ModelPublicID != nil {
		q = q.Where("model_public_id = ?", *filter.ModelPublicID)
	}
	if filter.ModelPublicIDs != nil {
		q = q.Where("model_public_id IN ?", *filter.ModelPublicIDs)
	}
	if filter.Active != nil {
		q = q.Where("active = ?", *filter.Active)
	}
	if filter.SupportsImages != nil {
		q = q.Where("supports_images = ?", *filter.SupportsImages)
	}
	if filter.SupportsEmbeddings != nil {
		q = q.Where("supports_embeddings = ?", *filter.SupportsEmbeddings)
	}
	if filter.SupportsReasoning != nil {
		q = q.Where("supports_reasoning = ?", *filter.SupportsReasoning)
	}
	if filter.SupportsAudio != nil {
		q = q.Where("supports_audio = ?", *filter.SupportsAudio)
	}
	if filter.SupportsVideo != nil {
		q = q.Where("supports_video = ?", *filter.SupportsVideo)
	}

	if p != nil {
		q = q.Limit(p.Limit).Offset(p.Offset)
	}

	if err := q.Find(&dbModels).Error; err != nil {
		return nil, err
	}

	models := make([]*model.ProviderModel, 0, len(dbModels))
	for _, dbModel := range dbModels {
		m, err := dbModel.EtoD()
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}

func (r *ProviderModelRepository) Count(ctx context.Context, filter model.ProviderModelFilter) (int64, error) {
	var count int64
	q := r.db.WithContext(ctx).Model(&dbschema.ProviderModel{})

	if filter.IDs != nil {
		q = q.Where("id IN ?", *filter.IDs)
	}
	if filter.ProviderID != nil {
		q = q.Where("provider_id = ?", *filter.ProviderID)
	}
	if filter.Active != nil {
		q = q.Where("active = ?", *filter.Active)
	}

	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *ProviderModelRepository) BatchUpdateActive(ctx context.Context, filter model.ProviderModelFilter, active bool) (int64, error) {
	q := r.db.WithContext(ctx).Model(&dbschema.ProviderModel{})

	if filter.ProviderID != nil {
		q = q.Where("provider_id = ?", *filter.ProviderID)
	}
	if filter.ModelPublicIDs != nil {
		q = q.Where("model_public_id IN ?", *filter.ModelPublicIDs)
	}

	result := q.Update("active", active)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
