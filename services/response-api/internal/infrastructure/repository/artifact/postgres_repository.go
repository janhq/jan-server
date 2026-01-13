package artifact

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	domain "jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/infrastructure/database/entities"
	"jan-server/services/response-api/internal/utils/platformerrors"
)

// PostgresRepository provides persistence for artifacts.
type PostgresRepository struct {
	db *gorm.DB
}

// NewPostgresRepository constructs the repository.
func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create inserts a new artifact record.
func (r *PostgresRepository) Create(ctx context.Context, artifact *domain.Artifact) error {
	responseID, err := r.resolveResponseID(ctx, artifact.ResponseID)
	if err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeNotFound,
			"response not found for artifact creation",
			err,
			"artifact-create-response-001",
		)
	}

	planID, err := r.resolvePlanID(ctx, artifact.PlanID)
	if err != nil {
		return err
	}

	parentID, err := r.resolveArtifactID(ctx, artifact.ParentID)
	if err != nil {
		return err
	}

	entity := mapArtifactToEntity(artifact, responseID, planID, parentID)
	if entity.PublicID == "" {
		entity.PublicID = uuid.New().String()
	}

	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to create artifact",
			err,
			"artifact-create-db-001",
		)
	}

	artifact.ID = entity.PublicID
	return nil
}

// Update persists changes to an artifact.
func (r *PostgresRepository) Update(ctx context.Context, artifact *domain.Artifact) error {
	responseID, err := r.resolveResponseID(ctx, artifact.ResponseID)
	if err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeNotFound,
			"response not found for artifact update",
			err,
			"artifact-update-response-001",
		)
	}

	planID, err := r.resolvePlanID(ctx, artifact.PlanID)
	if err != nil {
		return err
	}

	parentID, err := r.resolveArtifactID(ctx, artifact.ParentID)
	if err != nil {
		return err
	}

	entity := mapArtifactToEntity(artifact, responseID, planID, parentID)

	updates := map[string]interface{}{
		"response_id":       entity.ResponseID,
		"plan_id":           entity.PlanID,
		"content_type":      entity.ContentType,
		"mime_type":         entity.MimeType,
		"title":             entity.Title,
		"content":           entity.Content,
		"storage_path":      entity.StoragePath,
		"size_bytes":        entity.SizeBytes,
		"version":           entity.Version,
		"parent_id":         entity.ParentID,
		"is_latest":         entity.IsLatest,
		"retention_policy":  entity.RetentionPolicy,
		"metadata":          entity.Metadata,
		"updated_at":        entity.UpdatedAt,
		"expires_at":        entity.ExpiresAt,
	}

	if err := r.db.WithContext(ctx).
		Model(&entities.Artifact{}).
		Where("public_id = ?", artifact.ID).
		Updates(updates).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to update artifact",
			err,
			"artifact-update-db-001",
		)
	}
	return nil
}

// FindByID retrieves an artifact by public ID.
func (r *PostgresRepository) FindByID(ctx context.Context, id string) (*domain.Artifact, error) {
	var entity entities.Artifact
	if err := r.db.WithContext(ctx).
		Preload("Response").
		Preload("Plan").
		Preload("Parent").
		Where("public_id = ?", id).
		First(&entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				"artifact not found",
				err,
				"artifact-find-notfound-001",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find artifact",
			err,
			"artifact-find-db-001",
		)
	}

	artifact := mapArtifactFromEntity(&entity)
	if err := r.hydrateArtifactRefs(ctx, artifact, &entity); err != nil {
		return nil, err
	}
	return artifact, nil
}

// FindLatestByResponseID finds the latest artifact for a response.
func (r *PostgresRepository) FindLatestByResponseID(ctx context.Context, responseID string) (*domain.Artifact, error) {
	var entity entities.Artifact
	if err := r.db.WithContext(ctx).
		Preload("Response").
		Preload("Plan").
		Preload("Parent").
		Joins("JOIN responses ON responses.id = artifacts.response_id").
		Where("responses.public_id = ?", responseID).
		Where("artifacts.is_latest = ?", true).
		First(&entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				"artifact not found for response",
				err,
				"artifact-find-response-notfound-001",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find artifact by response",
			err,
			"artifact-find-response-db-001",
		)
	}

	artifact := mapArtifactFromEntity(&entity)
	if err := r.hydrateArtifactRefs(ctx, artifact, &entity); err != nil {
		return nil, err
	}
	return artifact, nil
}

// FindLatestByPlanID finds the latest artifact for a plan.
func (r *PostgresRepository) FindLatestByPlanID(ctx context.Context, planID string) (*domain.Artifact, error) {
	var entity entities.Artifact
	if err := r.db.WithContext(ctx).
		Preload("Response").
		Preload("Plan").
		Preload("Parent").
		Joins("JOIN plans ON plans.id = artifacts.plan_id").
		Where("plans.public_id = ?", planID).
		Where("artifacts.is_latest = ?", true).
		First(&entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				"artifact not found for plan",
				err,
				"artifact-find-plan-notfound-001",
			)
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find artifact by plan",
			err,
			"artifact-find-plan-db-001",
		)
	}

	artifact := mapArtifactFromEntity(&entity)
	if err := r.hydrateArtifactRefs(ctx, artifact, &entity); err != nil {
		return nil, err
	}
	return artifact, nil
}

// List retrieves artifacts matching the filter.
func (r *PostgresRepository) List(ctx context.Context, filter *domain.Filter) ([]*domain.Artifact, int64, error) {
	query := r.db.WithContext(ctx).Model(&entities.Artifact{}).
		Preload("Response").
		Preload("Plan").
		Preload("Parent")

	if filter.ResponseID != nil {
		query = query.Joins("JOIN responses ON responses.id = artifacts.response_id").
			Where("responses.public_id = ?", *filter.ResponseID)
	}
	if filter.PlanID != nil {
		query = query.Joins("JOIN plans ON plans.id = artifacts.plan_id").
			Where("plans.public_id = ?", *filter.PlanID)
	}
	if filter.ContentType != nil {
		query = query.Where("artifacts.content_type = ?", string(*filter.ContentType))
	}
	if filter.IsLatest != nil {
		query = query.Where("artifacts.is_latest = ?", *filter.IsLatest)
	}
	if filter.RetentionPolicy != nil {
		query = query.Where("artifacts.retention_policy = ?", string(*filter.RetentionPolicy))
	}
	if filter.ExcludeExpired {
		query = query.Where("artifacts.expires_at IS NULL OR artifacts.expires_at > ?", time.Now())
	}
	if filter.CreatedAfter != nil {
		query = query.Where("artifacts.created_at > ?", *filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		query = query.Where("artifacts.created_at < ?", *filter.CreatedBefore)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to count artifacts",
			err,
			"artifact-list-count-001",
		)
	}

	var entities []entities.Artifact
	if err := query.
		Order("artifacts.created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&entities).Error; err != nil {
		return nil, 0, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to list artifacts",
			err,
			"artifact-list-db-001",
		)
	}

	artifacts := make([]*domain.Artifact, 0, len(entities))
	for _, e := range entities {
		artifact := mapArtifactFromEntity(&e)
		if err := r.hydrateArtifactRefs(ctx, artifact, &e); err != nil {
			return nil, 0, err
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, total, nil
}

// ListVersions retrieves all versions of an artifact.
func (r *PostgresRepository) ListVersions(ctx context.Context, artifactID string) ([]*domain.Artifact, error) {
	// First find the artifact to get its root
	artifact, err := r.FindByID(ctx, artifactID)
	if err != nil {
		return nil, err
	}

	// Find the root artifact ID
	rootID := artifactID
	if artifact.ParentID != nil {
		rootID = *artifact.ParentID
	}

	// Get internal ID of root
	var rootEntity entities.Artifact
	if err := r.db.WithContext(ctx).
		Select("id").
		Where("public_id = ?", rootID).
		First(&rootEntity).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeNotFound,
			"root artifact not found",
			err,
			"artifact-versions-root-001",
		)
	}

	// Find all versions
	var entities []entities.Artifact
	if err := r.db.WithContext(ctx).
		Preload("Response").
		Preload("Plan").
		Preload("Parent").
		Where("id = ? OR parent_id = ?", rootEntity.ID, rootEntity.ID).
		Order("version ASC").
		Find(&entities).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to list artifact versions",
			err,
			"artifact-versions-db-001",
		)
	}

	artifacts := make([]*domain.Artifact, 0, len(entities))
	for _, e := range entities {
		artifact := mapArtifactFromEntity(&e)
		if err := r.hydrateArtifactRefs(ctx, artifact, &e); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

// Delete removes an artifact.
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).
		Where("public_id = ?", id).
		Delete(&entities.Artifact{}).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to delete artifact",
			err,
			"artifact-delete-db-001",
		)
	}
	return nil
}

// DeleteExpired removes all expired artifacts.
func (r *PostgresRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Delete(&entities.Artifact{})

	if result.Error != nil {
		return 0, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to delete expired artifacts",
			result.Error,
			"artifact-delete-expired-001",
		)
	}

	return result.RowsAffected, nil
}

// MarkOldVersionsNotLatest marks old versions as not latest.
func (r *PostgresRepository) MarkOldVersionsNotLatest(ctx context.Context, newVersionID string, parentID string) error {
	// Get internal IDs
	var parentEntity entities.Artifact
	if err := r.db.WithContext(ctx).
		Select("id").
		Where("public_id = ?", parentID).
		First(&parentEntity).Error; err != nil {
		return nil // Parent not found, nothing to update
	}

	isLatest := false
	if err := r.db.WithContext(ctx).
		Model(&entities.Artifact{}).
		Where("id = ? OR parent_id = ?", parentEntity.ID, parentEntity.ID).
		Where("public_id != ?", newVersionID).
		Update("is_latest", &isLatest).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to mark old versions as not latest",
			err,
			"artifact-versions-update-001",
		)
	}

	return nil
}

// Mapping functions

func mapArtifactToEntity(artifact *domain.Artifact, responseID uint, planID, parentID *uint) *entities.Artifact {
	isLatest := artifact.IsLatest

	return &entities.Artifact{
		PublicID:        artifact.ID,
		ResponseID:      responseID,
		PlanID:          planID,
		ContentType:     string(artifact.ContentType),
		MimeType:        artifact.MimeType,
		Title:           artifact.Title,
		Content:         artifact.Content,
		StoragePath:     artifact.StoragePath,
		SizeBytes:       artifact.SizeBytes,
		Version:         artifact.Version,
		ParentID:        parentID,
		IsLatest:        &isLatest,
		RetentionPolicy: string(artifact.RetentionPolicy),
		Metadata:        datatypes.JSON(artifact.Metadata),
		CreatedAt:       artifact.CreatedAt,
		UpdatedAt:       artifact.UpdatedAt,
		ExpiresAt:       artifact.ExpiresAt,
	}
}

func mapArtifactFromEntity(entity *entities.Artifact) *domain.Artifact {
	artifact := &domain.Artifact{
		ID:              entity.PublicID,
		ContentType:     domain.ContentType(entity.ContentType),
		MimeType:        entity.MimeType,
		Title:           entity.Title,
		Content:         entity.Content,
		StoragePath:     entity.StoragePath,
		SizeBytes:       entity.SizeBytes,
		Version:         entity.Version,
		RetentionPolicy: domain.RetentionPolicy(entity.RetentionPolicy),
		Metadata:        json.RawMessage(entity.Metadata),
		CreatedAt:       entity.CreatedAt,
		UpdatedAt:       entity.UpdatedAt,
		ExpiresAt:       entity.ExpiresAt,
	}

	if entity.IsLatest != nil {
		artifact.IsLatest = *entity.IsLatest
	}

	return artifact
}

func (r *PostgresRepository) resolveResponseID(ctx context.Context, publicID string) (uint, error) {
	if publicID == "" {
		return 0, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeValidation,
			"response_id is required",
			nil,
			"artifact-response-missing-001",
		)
	}
	var response entities.Response
	if err := r.db.WithContext(ctx).Select("id").Where("public_id = ?", publicID).First(&response).Error; err != nil {
		return 0, err
	}
	return response.ID, nil
}

func (r *PostgresRepository) resolvePlanID(ctx context.Context, publicID *string) (*uint, error) {
	if publicID == nil || *publicID == "" {
		return nil, nil
	}
	var plan entities.Plan
	if err := r.db.WithContext(ctx).Select("id").Where("public_id = ?", *publicID).First(&plan).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				"plan not found",
				err,
				"artifact-plan-001",
			)
		}
		return nil, err
	}
	return &plan.ID, nil
}

func (r *PostgresRepository) resolveArtifactID(ctx context.Context, publicID *string) (*uint, error) {
	if publicID == nil || *publicID == "" {
		return nil, nil
	}
	var artifact entities.Artifact
	if err := r.db.WithContext(ctx).Select("id").Where("public_id = ?", *publicID).First(&artifact).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(
				ctx,
				platformerrors.LayerRepository,
				platformerrors.ErrorTypeNotFound,
				"artifact not found",
				err,
				"artifact-parent-001",
			)
		}
		return nil, err
	}
	return &artifact.ID, nil
}

func (r *PostgresRepository) hydrateArtifactRefs(ctx context.Context, artifact *domain.Artifact, entity *entities.Artifact) error {
	if artifact.ResponseID == "" && entity.ResponseID != 0 {
		var response entities.Response
		if err := r.db.WithContext(ctx).Select("public_id").Where("id = ?", entity.ResponseID).First(&response).Error; err != nil {
			return err
		}
		artifact.ResponseID = response.PublicID
	}

	if entity.PlanID != nil {
		var plan entities.Plan
		if err := r.db.WithContext(ctx).Select("public_id").Where("id = ?", *entity.PlanID).First(&plan).Error; err != nil {
			return err
		}
		artifact.PlanID = &plan.PublicID
	}

	if entity.ParentID != nil {
		var parent entities.Artifact
		if err := r.db.WithContext(ctx).Select("public_id").Where("id = ?", *entity.ParentID).First(&parent).Error; err != nil {
			return err
		}
		artifact.ParentID = &parent.PublicID
	}

	return nil
}
