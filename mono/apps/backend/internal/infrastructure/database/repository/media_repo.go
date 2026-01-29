package repository

import (
	"context"
	"encoding/json"
	"errors"

	"jan-server/mono/apps/backend/internal/domain/media"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"

	"gorm.io/gorm"
)

// MediaRepository implements media.Repository using GORM.
type MediaRepository struct {
	db *gorm.DB
}

// NewMediaRepository creates a new media repository.
func NewMediaRepository(db *gorm.DB) *MediaRepository {
	return &MediaRepository{db: db}
}

var _ media.Repository = (*MediaRepository)(nil)

func (r *MediaRepository) Create(ctx context.Context, m *media.Media) error {
	schema := toMediaSchema(m)
	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		return err
	}
	m.ID = schema.ID
	m.CreatedAt = schema.CreatedAt
	return nil
}

func (r *MediaRepository) GetByID(ctx context.Context, id string) (*media.Media, error) {
	var schema dbschema.Media
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, media.ErrMediaNotFound
		}
		return nil, err
	}
	return toMediaDomain(&schema), nil
}

func (r *MediaRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&dbschema.Media{}, "id = ?", id).Error
}

func (r *MediaRepository) List(ctx context.Context, userID string, limit, offset int) ([]*media.Media, int64, error) {
	var schemas []dbschema.Media
	var total int64

	query := r.db.WithContext(ctx).Model(&dbschema.Media{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&schemas).Error; err != nil {
		return nil, 0, err
	}

	files := make([]*media.Media, len(schemas))
	for i, s := range schemas {
		files[i] = toMediaDomain(&s)
	}

	return files, total, nil
}

func toMediaSchema(m *media.Media) *dbschema.Media {
	metadataJSON, _ := json.Marshal(m.Metadata)
	return &dbschema.Media{
		ID:           m.ID,
		UserID:       m.UserID,
		Filename:     m.Filename,
		OriginalName: m.OriginalName,
		MimeType:     m.MimeType,
		Size:         m.Size,
		StorageKey:   m.StorageKey,
		Bucket:       m.Bucket,
		ContentHash:  m.ContentHash,
		Metadata:     metadataJSON,
		Purpose:      m.Purpose,
		ExpiresAt:    m.ExpiresAt,
	}
}

func toMediaDomain(s *dbschema.Media) *media.Media {
	var metadata map[string]any
	_ = json.Unmarshal(s.Metadata, &metadata)

	return &media.Media{
		ID:           s.ID,
		UserID:       s.UserID,
		Filename:     s.Filename,
		OriginalName: s.OriginalName,
		MimeType:     s.MimeType,
		Size:         s.Size,
		StorageKey:   s.StorageKey,
		Bucket:       s.Bucket,
		ContentHash:  s.ContentHash,
		Metadata:     metadata,
		Purpose:      s.Purpose,
		ExpiresAt:    s.ExpiresAt,
		CreatedAt:    s.CreatedAt,
	}
}
