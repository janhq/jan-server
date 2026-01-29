package sharerepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"jan-server/mono/apps/backend/internal/domain/query"
	"jan-server/mono/apps/backend/internal/domain/share"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"
	"jan-server/mono/apps/backend/internal/infrastructure/database/transaction"
	"jan-server/mono/apps/backend/internal/utils/functional"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// ShareGormRepository implements share.ShareRepository using GORM
type ShareGormRepository struct {
	db *transaction.Database
}

var _ share.ShareRepository = (*ShareGormRepository)(nil)

// NewShareGormRepository creates a new share repository
func NewShareGormRepository(db *transaction.Database) share.ShareRepository {
	return &ShareGormRepository{db: db}
}

// Create implements share.ShareRepository.
func (repo *ShareGormRepository) Create(ctx context.Context, s *share.Share) error {
	model := dbschema.NewSchemaConversationShare(s)
	if err := repo.getDB(ctx).Create(model).Error; err != nil {
		return platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerRepository, err, "failed to create share", "4a5b6c7d-8e9f-4a0b-1c2d-3e4f5a6b7c8d")
	}
	// Update the domain object with generated ID and timestamps
	s.ID = model.ID
	s.CreatedAt = model.CreatedAt
	s.UpdatedAt = model.UpdatedAt
	return nil
}

// FindByFilter implements share.ShareRepository.
func (repo *ShareGormRepository) FindByFilter(ctx context.Context, filter share.ShareFilter, pagination *query.Pagination) ([]*share.Share, error) {
	db := repo.applyFilter(repo.getDB(ctx), filter)
	db = repo.applyPagination(db, pagination)

	var rows []dbschema.ConversationShare
	if err := db.Find(&rows).Error; err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerRepository, err, "failed to find shares", "5b6c7d8e-9f0a-4b1c-2d3e-4f5a6b7c8d9e")
	}

	result := functional.Map(rows, func(item dbschema.ConversationShare) *share.Share {
		return item.EtoD()
	})
	return result, nil
}

// Count implements share.ShareRepository.
func (repo *ShareGormRepository) Count(ctx context.Context, filter share.ShareFilter) (int64, error) {
	db := repo.applyFilter(repo.getDB(ctx).Model(&dbschema.ConversationShare{}), filter)
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerRepository, err, "failed to count shares", "6c7d8e9f-0a1b-4c2d-3e4f-5a6b7c8d9e0f")
	}
	return count, nil
}

// FindByID implements share.ShareRepository.
func (repo *ShareGormRepository) FindByID(ctx context.Context, id uint) (*share.Share, error) {
	var model dbschema.ConversationShare
	if err := repo.getDB(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerRepository, err, "failed to find share by ID", "7d8e9f0a-1b2c-4d3e-4f5a-6b7c8d9e0f1a")
	}
	return model.EtoD(), nil
}

// FindByPublicID implements share.ShareRepository.
func (repo *ShareGormRepository) FindByPublicID(ctx context.Context, publicID string) (*share.Share, error) {
	var model dbschema.ConversationShare
	if err := repo.getDB(ctx).Where("public_id = ?", publicID).First(&model).Error; err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerRepository, err, "failed to find share by public ID", "8e9f0a1b-2c3d-4e4f-5a6b-7c8d9e0f1a2b")
	}
	return model.EtoD(), nil
}

// FindBySlug implements share.ShareRepository.
func (repo *ShareGormRepository) FindBySlug(ctx context.Context, slug string) (*share.Share, error) {
	var model dbschema.ConversationShare
	if err := repo.getDB(ctx).Where("slug = ?", slug).First(&model).Error; err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerRepository, err, "failed to find share by slug", "9f0a1b2c-3d4e-4f5a-6b7c-8d9e0f1a2b3c")
	}
	return model.EtoD(), nil
}

// Update implements share.ShareRepository.
func (repo *ShareGormRepository) Update(ctx context.Context, s *share.Share) error {
	model := dbschema.NewSchemaConversationShare(s)
	if err := repo.getDB(ctx).Save(model).Error; err != nil {
		return platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerRepository, err, "failed to update share", "0a1b2c3d-4e5f-4a6b-7c8d-9e0f1a2b3c4d")
	}
	s.UpdatedAt = model.UpdatedAt
	return nil
}

// Delete implements share.ShareRepository.
func (repo *ShareGormRepository) Delete(ctx context.Context, id uint) error {
	if err := repo.getDB(ctx).Delete(&dbschema.ConversationShare{}, id).Error; err != nil {
		return platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerRepository, err, "failed to delete share", "1b2c3d4e-5f6a-4b7c-8d9e-0f1a2b3c4d5e")
	}
	return nil
}

// FindActiveByConversationID implements share.ShareRepository.
func (repo *ShareGormRepository) FindActiveByConversationID(ctx context.Context, conversationID uint) ([]*share.Share, error) {
	var rows []dbschema.ConversationShare
	if err := repo.getDB(ctx).
		Where("conversation_id = ?", conversationID).
		Where("revoked_at IS NULL").
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerRepository, err, "failed to find active shares", "2c3d4e5f-6a7b-4c8d-9e0f-1a2b3c4d5e6f")
	}

	result := functional.Map(rows, func(item dbschema.ConversationShare) *share.Share {
		return item.EtoD()
	})
	return result, nil
}

// IncrementViewCount implements share.ShareRepository.
func (repo *ShareGormRepository) IncrementViewCount(ctx context.Context, id uint) error {
	now := time.Now()
	if err := repo.getDB(ctx).
		Model(&dbschema.ConversationShare{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"view_count":     gorm.Expr("view_count + 1"),
			"last_viewed_at": now,
		}).Error; err != nil {
		return platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerRepository, err, "failed to increment view count", "3d4e5f6a-7b8c-4d9e-0f1a-2b3c4d5e6f7a")
	}
	return nil
}

// Revoke implements share.ShareRepository.
func (repo *ShareGormRepository) Revoke(ctx context.Context, id uint) error {
	now := time.Now()
	if err := repo.getDB(ctx).
		Model(&dbschema.ConversationShare{}).
		Where("id = ?", id).
		Update("revoked_at", now).Error; err != nil {
		return platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerRepository, err, "failed to revoke share", "4e5f6a7b-8c9d-4e0f-1a2b-3c4d5e6f7a8b")
	}
	return nil
}

// RevokeAllByConversationID implements share.ShareRepository.
func (repo *ShareGormRepository) RevokeAllByConversationID(ctx context.Context, conversationID uint) error {
	now := time.Now()
	if err := repo.getDB(ctx).
		Model(&dbschema.ConversationShare{}).
		Where("conversation_id = ?", conversationID).
		Where("revoked_at IS NULL").
		Update("revoked_at", now).Error; err != nil {
		return platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerRepository, err, "failed to revoke all shares for conversation", "5f6a7b8c-9d0e-4f1a-2b3c-4d5e6f7a8b9c")
	}
	return nil
}

// SlugExists implements share.ShareRepository.
func (repo *ShareGormRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	var count int64
	if err := repo.getDB(ctx).
		Model(&dbschema.ConversationShare{}).
		Where("slug = ?", slug).
		Count(&count).Error; err != nil {
		return false, platformerrors.AsErrorWithUUID(ctx, platformerrors.LayerRepository, err, "failed to check slug existence", "6a7b8c9d-0e1f-4a2b-3c4d-5e6f7a8b9c0d")
	}
	return count > 0, nil
}

// getDB returns the database connection, checking for transaction context
func (repo *ShareGormRepository) getDB(ctx context.Context) *gorm.DB {
	return repo.db.GetTx(ctx)
}

// applyFilter applies filter criteria to the query
func (repo *ShareGormRepository) applyFilter(db *gorm.DB, filter share.ShareFilter) *gorm.DB {
	if filter.ID != nil {
		db = db.Where("id = ?", *filter.ID)
	}
	if filter.PublicID != nil {
		db = db.Where("public_id = ?", *filter.PublicID)
	}
	if filter.Slug != nil {
		db = db.Where("slug = ?", *filter.Slug)
	}
	if filter.ConversationID != nil {
		db = db.Where("conversation_id = ?", *filter.ConversationID)
	}
	if filter.OwnerUserID != nil {
		db = db.Where("owner_user_id = ?", *filter.OwnerUserID)
	}
	if !filter.IncludeRevoked {
		db = db.Where("revoked_at IS NULL")
	}
	return db
}

// applyPagination applies pagination to the query
func (repo *ShareGormRepository) applyPagination(db *gorm.DB, pagination *query.Pagination) *gorm.DB {
	if pagination == nil {
		return db.Order("created_at DESC").Limit(20)
	}

	if pagination.After != nil {
		db = db.Where("id > ?", *pagination.After)
	}

	limit := 20
	if pagination.Limit != nil && *pagination.Limit > 0 {
		limit = *pagination.Limit
	}
	db = db.Limit(limit + 1) // Fetch one more to check for hasMore

	if pagination.Order == "asc" {
		db = db.Order("created_at ASC")
	} else {
		db = db.Order("created_at DESC")
	}

	return db
}
