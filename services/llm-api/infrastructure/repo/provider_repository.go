package repo

import (
	"context"
	"errors"

	"jan-server/services/llm-api/domain/model"
	"jan-server/services/llm-api/domain/query"
	"jan-server/services/llm-api/infrastructure/db/dbschema"

	"gorm.io/gorm"
)

type ProviderRepository struct {
	db *gorm.DB
}

func NewProviderRepository(db *gorm.DB) *ProviderRepository {
	return &ProviderRepository{db: db}
}

func (r *ProviderRepository) Create(ctx context.Context, provider *model.Provider) error {
	dbProvider := dbschema.NewSchemaProvider(provider)
	if err := r.db.WithContext(ctx).Create(dbProvider).Error; err != nil {
		return err
	}
	*provider = *dbProvider.EtoD()
	return nil
}

func (r *ProviderRepository) Update(ctx context.Context, provider *model.Provider) error {
	dbProvider := dbschema.NewSchemaProvider(provider)
	if err := r.db.WithContext(ctx).Save(dbProvider).Error; err != nil {
		return err
	}
	*provider = *dbProvider.EtoD()
	return nil
}

func (r *ProviderRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&dbschema.Provider{}, id).Error
}

func (r *ProviderRepository) FindByID(ctx context.Context, id uint) (*model.Provider, error) {
	var dbProvider dbschema.Provider
	if err := r.db.WithContext(ctx).First(&dbProvider, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return dbProvider.EtoD(), nil
}

func (r *ProviderRepository) FindByPublicID(ctx context.Context, publicID string) (*model.Provider, error) {
	var dbProvider dbschema.Provider
	if err := r.db.WithContext(ctx).Where("public_id = ?", publicID).First(&dbProvider).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return dbProvider.EtoD(), nil
}

func (r *ProviderRepository) FindByFilter(ctx context.Context, filter model.ProviderFilter, p *query.Pagination) ([]*model.Provider, error) {
	var dbProviders []dbschema.Provider
	q := r.db.WithContext(ctx)

	if filter.IDs != nil {
		q = q.Where("id IN ?", *filter.IDs)
	}
	if filter.PublicID != nil {
		q = q.Where("public_id = ?", *filter.PublicID)
	}
	if filter.Kind != nil {
		q = q.Where("kind = ?", string(*filter.Kind))
	}
	if filter.Active != nil {
		q = q.Where("active = ?", *filter.Active)
	}
	if filter.IsModerated != nil {
		q = q.Where("is_moderated = ?", *filter.IsModerated)
	}
	if filter.LastSyncedAfter != nil {
		q = q.Where("last_synced_at > ?", *filter.LastSyncedAfter)
	}
	if filter.LastSyncedBefore != nil {
		q = q.Where("last_synced_at < ?", *filter.LastSyncedBefore)
	}

	if p != nil {
		q = q.Limit(p.Limit).Offset(p.Offset)
	}

	if err := q.Find(&dbProviders).Error; err != nil {
		return nil, err
	}

	providers := make([]*model.Provider, len(dbProviders))
	for i, dbProvider := range dbProviders {
		providers[i] = dbProvider.EtoD()
	}
	return providers, nil
}

func (r *ProviderRepository) Count(ctx context.Context, filter model.ProviderFilter) (int64, error) {
	var count int64
	q := r.db.WithContext(ctx).Model(&dbschema.Provider{})

	if filter.IDs != nil {
		q = q.Where("id IN ?", *filter.IDs)
	}
	if filter.PublicID != nil {
		q = q.Where("public_id = ?", *filter.PublicID)
	}
	if filter.Kind != nil {
		q = q.Where("kind = ?", string(*filter.Kind))
	}
	if filter.Active != nil {
		q = q.Where("active = ?", *filter.Active)
	}

	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *ProviderRepository) FindByIDs(ctx context.Context, ids []uint) ([]*model.Provider, error) {
	var dbProviders []dbschema.Provider
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&dbProviders).Error; err != nil {
		return nil, err
	}

	providers := make([]*model.Provider, len(dbProviders))
	for i, dbProvider := range dbProviders {
		providers[i] = dbProvider.EtoD()
	}
	return providers, nil
}
