package media

import (
	"context"

	"gorm.io/gorm"

	domain "jan-server/services/media-api/internal/domain/media"
	"jan-server/services/media-api/internal/infrastructure/database/entities"
)

// Repository handles media object persistence.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByHash(ctx context.Context, hash string) (*domain.MediaObject, error) {
	var entity entities.MediaObject
	err := r.db.WithContext(ctx).Where("sha256 = ?", hash).First(&entity).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	obj := mapEntity(entity)
	return &obj, nil
}

func (r *Repository) Create(ctx context.Context, obj *domain.MediaObject) error {
	entity := entities.MediaObject{
		ID:              obj.ID,
		StorageProvider: obj.StorageProvider,
		StorageKey:      obj.StorageKey,
		MimeType:        obj.MimeType,
		Bytes:           obj.Bytes,
		Sha256:          obj.Sha256,
		CreatedBy:       obj.CreatedBy,
		RetentionUntil:  obj.RetentionUntil,
	}
	return r.db.WithContext(ctx).Create(&entity).Error
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.MediaObject, error) {
	var entity entities.MediaObject
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&entity).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	obj := mapEntity(entity)
	return &obj, nil
}

func mapEntity(entity entities.MediaObject) domain.MediaObject {
	return domain.MediaObject{
		ID:              entity.ID,
		StorageProvider: entity.StorageProvider,
		StorageKey:      entity.StorageKey,
		MimeType:        entity.MimeType,
		Bytes:           entity.Bytes,
		Sha256:          entity.Sha256,
		CreatedBy:       entity.CreatedBy,
		RetentionUntil:  entity.RetentionUntil,
		CreatedAt:       entity.CreatedAt,
		UpdatedAt:       entity.UpdatedAt,
	}
}
