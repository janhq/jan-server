package repository

import (
	"context"
	"encoding/json"
	"errors"

	"jan-server/mono/apps/backend/internal/domain/artifact"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"

	"gorm.io/gorm"
)

// ArtifactRepository implements artifact.Repository using GORM.
type ArtifactRepository struct {
	db *gorm.DB
}

// NewArtifactRepository creates a new artifact repository.
func NewArtifactRepository(db *gorm.DB) *ArtifactRepository {
	return &ArtifactRepository{db: db}
}

var _ artifact.Repository = (*ArtifactRepository)(nil)

func (r *ArtifactRepository) Create(ctx context.Context, a *artifact.Artifact) error {
	schema := toArtifactSchema(a)
	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		return err
	}
	a.ID = schema.ID
	a.CreatedAt = schema.CreatedAt
	a.UpdatedAt = schema.UpdatedAt
	return nil
}

func (r *ArtifactRepository) GetByID(ctx context.Context, id string) (*artifact.Artifact, error) {
	var schema dbschema.Artifact
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, artifact.ErrArtifactNotFound
		}
		return nil, err
	}
	return toArtifactDomain(&schema), nil
}

func (r *ArtifactRepository) GetByShareToken(ctx context.Context, token string) (*artifact.Artifact, error) {
	var schema dbschema.Artifact
	if err := r.db.WithContext(ctx).Where("share_token = ?", token).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, artifact.ErrArtifactNotFound
		}
		return nil, err
	}
	return toArtifactDomain(&schema), nil
}

func (r *ArtifactRepository) Update(ctx context.Context, a *artifact.Artifact) error {
	schema := toArtifactSchema(a)
	return r.db.WithContext(ctx).Model(&dbschema.Artifact{}).Where("id = ?", a.ID).Updates(schema).Error
}

func (r *ArtifactRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&dbschema.Artifact{}, "id = ?", id).Error
}

func (r *ArtifactRepository) List(ctx context.Context, filter artifact.ListArtifactsFilter) ([]*artifact.Artifact, int64, error) {
	var schemas []dbschema.Artifact
	var total int64

	query := r.db.WithContext(ctx).Model(&dbschema.Artifact{}).Where("user_id = ?", filter.UserID)

	if filter.ConversationID != nil {
		query = query.Where("conversation_id = ?", *filter.ConversationID)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Search != "" {
		query = query.Where("title ILIKE ?", "%"+filter.Search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("updated_at DESC").Offset(filter.Offset).Limit(filter.Limit).Find(&schemas).Error; err != nil {
		return nil, 0, err
	}

	artifacts := make([]*artifact.Artifact, len(schemas))
	for i, s := range schemas {
		artifacts[i] = toArtifactDomain(&s)
	}

	return artifacts, total, nil
}

func (r *ArtifactRepository) CreateVersion(ctx context.Context, v *artifact.ArtifactVersion) error {
	schema := &dbschema.ArtifactVersion{
		ArtifactID: v.ArtifactID,
		Version:    v.Version,
		Content:    v.Content,
	}
	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		return err
	}
	v.ID = schema.ID
	v.CreatedAt = schema.CreatedAt
	return nil
}

func (r *ArtifactRepository) ListVersions(ctx context.Context, artifactID string) ([]*artifact.ArtifactVersion, error) {
	var schemas []dbschema.ArtifactVersion
	if err := r.db.WithContext(ctx).Where("artifact_id = ?", artifactID).Order("version DESC").Find(&schemas).Error; err != nil {
		return nil, err
	}

	versions := make([]*artifact.ArtifactVersion, len(schemas))
	for i, s := range schemas {
		versions[i] = &artifact.ArtifactVersion{
			ID:         s.ID,
			ArtifactID: s.ArtifactID,
			Version:    s.Version,
			Content:    s.Content,
			CreatedAt:  s.CreatedAt,
		}
	}

	return versions, nil
}

func (r *ArtifactRepository) GetVersion(ctx context.Context, artifactID string, version int) (*artifact.ArtifactVersion, error) {
	var schema dbschema.ArtifactVersion
	if err := r.db.WithContext(ctx).Where("artifact_id = ? AND version = ?", artifactID, version).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, artifact.ErrArtifactNotFound
		}
		return nil, err
	}
	return &artifact.ArtifactVersion{
		ID:         schema.ID,
		ArtifactID: schema.ArtifactID,
		Version:    schema.Version,
		Content:    schema.Content,
		CreatedAt:  schema.CreatedAt,
	}, nil
}

func toArtifactSchema(a *artifact.Artifact) *dbschema.Artifact {
	metadataJSON, _ := json.Marshal(a.Metadata)
	isPublic := a.IsPublic
	return &dbschema.Artifact{
		ID:             a.ID,
		UserID:         a.UserID,
		ConversationID: a.ConversationID,
		ResponseID:     a.ResponseID,
		Title:          a.Title,
		Description:    a.Description,
		Type:           a.Type,
		Language:       a.Language,
		Content:        a.Content,
		Version:        a.Version,
		Metadata:       metadataJSON,
		IsPublic:       &isPublic,
		ShareToken:     a.ShareToken,
	}
}

func toArtifactDomain(s *dbschema.Artifact) *artifact.Artifact {
	var metadata map[string]any
	_ = json.Unmarshal(s.Metadata, &metadata)

	isPublic := false
	if s.IsPublic != nil {
		isPublic = *s.IsPublic
	}

	return &artifact.Artifact{
		ID:             s.ID,
		UserID:         s.UserID,
		ConversationID: s.ConversationID,
		ResponseID:     s.ResponseID,
		Title:          s.Title,
		Description:    s.Description,
		Type:           s.Type,
		Language:       s.Language,
		Content:        s.Content,
		Version:        s.Version,
		Metadata:       metadata,
		IsPublic:       isPublic,
		ShareToken:     s.ShareToken,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}
