package projectrepo

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"jan-server/services/llm-api/internal/domain/project"
	"jan-server/services/llm-api/internal/domain/query"
	"jan-server/services/llm-api/internal/infrastructure/database/dbschema"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

type ProjectGormRepository struct {
	db *gorm.DB
}

var _ project.ProjectRepository = (*ProjectGormRepository)(nil)

func NewProjectGormRepository(db *gorm.DB) project.ProjectRepository {
	return &ProjectGormRepository{db: db}
}

// Create implements project.ProjectRepository.
func (repo *ProjectGormRepository) Create(ctx context.Context, proj *project.Project) error {
	dbProject := dbschema.NewSchemaProject(proj)
	if err := repo.db.WithContext(ctx).Create(dbProject).Error; err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to create project")
	}
	proj.ID = dbProject.ID
	proj.CreatedAt = dbProject.CreatedAt
	proj.UpdatedAt = dbProject.UpdatedAt
	return nil
}

// GetByPublicID implements project.ProjectRepository.
func (repo *ProjectGormRepository) GetByPublicID(ctx context.Context, publicID string) (*project.Project, error) {
	var dbProject dbschema.Project
	err := repo.db.WithContext(ctx).
		Where("public_id = ? AND deleted_at IS NULL", publicID).
		First(&dbProject).Error

	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to find project by public ID")
	}
	return dbProject.EtoD(), nil
}

// GetByPublicIDAndUserID implements project.ProjectRepository.
func (repo *ProjectGormRepository) GetByPublicIDAndUserID(ctx context.Context, publicID string, userID uint) (*project.Project, error) {
	var dbProject dbschema.Project
	err := repo.db.WithContext(ctx).
		Where("public_id = ? AND user_id = ? AND deleted_at IS NULL", publicID, userID).
		First(&dbProject).Error

	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to find project by public ID and user ID")
	}
	return dbProject.EtoD(), nil
}

// ListByUserID implements project.ProjectRepository.
func (repo *ProjectGormRepository) ListByUserID(ctx context.Context, userID uint, pagination *query.Pagination) ([]*project.Project, int64, error) {
	// Build base query
	baseQuery := repo.db.WithContext(ctx).
		Model(&dbschema.Project{}).
		Where("user_id = ? AND deleted_at IS NULL", userID)

	// Count total
	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to count projects")
	}

	// Apply pagination
	query := baseQuery
	if pagination != nil {
		if pagination.After != nil {
			if pagination.Order == "desc" {
				query = query.Where("id < ?", *pagination.After)
			} else {
				query = query.Where("id > ?", *pagination.After)
			}
		}

		if pagination.Order == "desc" {
			query = query.Order("updated_at DESC")
		} else {
			query = query.Order("updated_at ASC")
		}

		if pagination.Limit != nil && *pagination.Limit > 0 {
			query = query.Limit(*pagination.Limit)
		}
	} else {
		// Default ordering
		query = query.Order("updated_at DESC")
	}

	// Execute query
	var rows []dbschema.Project
	if err := query.Find(&rows).Error; err != nil {
		return nil, 0, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to list projects")
	}

	// Convert to domain
	result := make([]*project.Project, len(rows))
	for i, row := range rows {
		result[i] = row.EtoD()
	}

	return result, total, nil
}

// Update implements project.ProjectRepository.
func (repo *ProjectGormRepository) Update(ctx context.Context, proj *project.Project) error {
	dbProject := dbschema.ProjectDtoE(proj)
	dbProject.UpdatedAt = time.Now()

	// Update only specified fields
	err := repo.db.WithContext(ctx).Model(&dbschema.Project{}).
		Where("public_id = ?", proj.PublicID).
		Updates(map[string]interface{}{
			"name":         dbProject.Name,
			"instruction":  dbProject.Instruction,
			"favorite":     dbProject.Favorite,
			"archived_at":  dbProject.ArchivedAt,
			"last_used_at": dbProject.LastUsedAt,
			"updated_at":   dbProject.UpdatedAt,
		}).Error

	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to update project")
	}

	proj.UpdatedAt = dbProject.UpdatedAt
	return nil
}

// Delete implements project.ProjectRepository.
func (repo *ProjectGormRepository) Delete(ctx context.Context, publicID string) error {
	now := time.Now()

	result := repo.db.WithContext(ctx).Model(&dbschema.Project{}).
		Where("public_id = ? AND deleted_at IS NULL", publicID).
		Update("deleted_at", now)

	if result.Error != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, result.Error, "failed to delete project")
	}

	if result.RowsAffected == 0 {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound, fmt.Sprintf("project %s not found", publicID), nil, "")
	}

	return nil
}
