package apikeyrepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"jan-server/services/llm-api/internal/domain/apikey"
	"jan-server/services/llm-api/internal/infrastructure/database/dbschema"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

type Repository struct {
	db *gorm.DB
}

func NewAPIKeyRepository(db *gorm.DB) apikey.Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, key *apikey.APIKey) (*apikey.APIKey, error) {
	model := dbschema.FromDomain(key)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to create api key")
	}
	return model.EtoD(), nil
}

func (r *Repository) ListByUser(ctx context.Context, userID uint) ([]apikey.APIKey, error) {
	var models []dbschema.APIKey
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to list api keys")
	}
	result := make([]apikey.APIKey, 0, len(models))
	for _, m := range models {
		if domain := m.EtoD(); domain != nil {
			result = append(result, *domain)
		}
	}
	return result, nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*apikey.APIKey, error) {
	var model dbschema.APIKey
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to fetch api key")
	}
	return model.EtoD(), nil
}

func (r *Repository) CountActiveByUser(ctx context.Context, userID uint) (int64, error) {
	var count int64
	now := time.Now()
	err := r.db.WithContext(ctx).
		Model(&dbschema.APIKey{}).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, now).
		Count(&count).Error
	if err != nil {
		return 0, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to count api keys")
	}
	return count, nil
}

func (r *Repository) FindByHash(ctx context.Context, hash string) (*apikey.APIKey, error) {
	var model dbschema.APIKey
	if err := r.db.WithContext(ctx).Where("hash = ?", hash).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to fetch api key by hash")
	}
	return model.EtoD(), nil
}

func (r *Repository) MarkRevoked(ctx context.Context, id string, revokedAt time.Time) error {
	updateErr := r.db.WithContext(ctx).Model(&dbschema.APIKey{}).
		Where("id = ?", id).
		Update("revoked_at", revokedAt).Error
	if updateErr != nil {
		return platformerrors.AsError(
			ctx,
			platformerrors.LayerRepository,
			updateErr,
			"failed to revoke api key",
		)
	}
	return nil
}
