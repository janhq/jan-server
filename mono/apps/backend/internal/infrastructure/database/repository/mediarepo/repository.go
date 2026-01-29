package media

import (
	"context"

	"gorm.io/gorm"

	domain "jan-server/mono/apps/backend/internal/domain/media"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// Repository handles media object persistence.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByHash(ctx context.Context, hash string) (*domain.MediaObject, error) {
	var entity dbschema.MediaObject
	err := r.db.WithContext(ctx).Where("sha256 = ?", hash).First(&entity).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find media by hash",
			err,
			"7a8f3d2e-4b1c-4a9e-8f7d-2c3e4f5a6b7c",
		)
	}
	obj := mapEntity(entity)
	return &obj, nil
}

func (r *Repository) Create(ctx context.Context, obj *domain.MediaObject) error {
	entity := dbschema.MediaObject{
		ID:              obj.ID,
		StorageProvider: obj.StorageProvider,
		StorageKey:      obj.StorageKey,
		MimeType:        obj.MimeType,
		Bytes:           obj.Bytes,
		Sha256:          obj.Sha256,
		CreatedBy:       obj.CreatedBy,
		RetentionUntil:  obj.RetentionUntil,
	}
	err := r.db.WithContext(ctx).Create(&entity).Error
	if err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to create media object",
			err,
			"9b2e4f5a-6c7d-4e8f-9a0b-1c2d3e4f5a6b",
		)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.MediaObject, error) {
	var entity dbschema.MediaObject
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&entity).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				"media object not found",
				err,
				"1c2d3e4f-5a6b-4c7d-8e9f-0a1b2c3d4e5f",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to get media by id",
			err,
			"2d3e4f5a-6b7c-4d8e-9f0a-1b2c3d4e5f6a",
		)
	}
	obj := mapEntity(entity)
	return &obj, nil
}

func mapEntity(entity dbschema.MediaObject) domain.MediaObject {
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
