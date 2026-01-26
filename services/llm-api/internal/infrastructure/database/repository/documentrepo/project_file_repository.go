package documentrepo

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"jan-server/services/llm-api/internal/domain/document"
	"jan-server/services/llm-api/internal/domain/query"
	"jan-server/services/llm-api/internal/infrastructure/database/dbschema"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

type ProjectFileGormRepository struct {
	db *gorm.DB
}

var _ document.ProjectFileRepository = (*ProjectFileGormRepository)(nil)

func NewProjectFileGormRepository(db *gorm.DB) document.ProjectFileRepository {
	return &ProjectFileGormRepository{db: db}
}

// Create implements document.ProjectFileRepository.
func (repo *ProjectFileGormRepository) Create(ctx context.Context, file *document.ProjectFile) error {
	dbFile := dbschema.NewSchemaProjectFile(file)
	if err := repo.db.WithContext(ctx).Create(dbFile).Error; err != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to create project file", err, "proj-file-create-001")
	}
	file.ID = dbFile.ID
	file.CreatedAt = dbFile.CreatedAt
	file.UpdatedAt = dbFile.UpdatedAt
	return nil
}

// GetByPublicID implements document.ProjectFileRepository.
func (repo *ProjectFileGormRepository) GetByPublicID(ctx context.Context, publicID string) (*document.ProjectFile, error) {
	var dbFile dbschema.ProjectFile
	err := repo.db.WithContext(ctx).
		Preload("DocumentContent").
		Where("public_id = ? AND deleted_at IS NULL", publicID).
		First(&dbFile).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound,
				fmt.Sprintf("project file %s not found", publicID), err, "proj-file-notfound-001")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to find project file", err, "proj-file-get-001")
	}
	return dbFile.EtoD(), nil
}

// GetByPublicIDAndProjectID implements document.ProjectFileRepository.
func (repo *ProjectFileGormRepository) GetByPublicIDAndProjectID(ctx context.Context, publicID string, projectID uint) (*document.ProjectFile, error) {
	var dbFile dbschema.ProjectFile
	err := repo.db.WithContext(ctx).
		Preload("DocumentContent").
		Where("public_id = ? AND project_id = ? AND deleted_at IS NULL", publicID, projectID).
		First(&dbFile).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound,
				fmt.Sprintf("project file %s not found in project", publicID), err, "proj-file-notfound-002")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to find project file", err, "proj-file-get-002")
	}
	return dbFile.EtoD(), nil
}

// Update implements document.ProjectFileRepository.
func (repo *ProjectFileGormRepository) Update(ctx context.Context, file *document.ProjectFile) error {
	dbFile := dbschema.NewSchemaProjectFile(file)
	dbFile.UpdatedAt = time.Now()

	err := repo.db.WithContext(ctx).Model(&dbschema.ProjectFile{}).
		Where("public_id = ? AND deleted_at IS NULL", file.PublicID).
		Updates(map[string]interface{}{
			"display_order": dbFile.DisplayOrder,
			"updated_at":    dbFile.UpdatedAt,
		}).Error

	if err != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to update project file", err, "proj-file-update-001")
	}

	file.UpdatedAt = dbFile.UpdatedAt
	return nil
}

// Delete implements document.ProjectFileRepository (soft delete).
func (repo *ProjectFileGormRepository) Delete(ctx context.Context, publicID string) error {
	now := time.Now()

	result := repo.db.WithContext(ctx).Model(&dbschema.ProjectFile{}).
		Where("public_id = ? AND deleted_at IS NULL", publicID).
		Update("deleted_at", now)

	if result.Error != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to delete project file", result.Error, "proj-file-delete-001")
	}

	if result.RowsAffected == 0 {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound,
			fmt.Sprintf("project file %s not found", publicID), nil, "proj-file-notfound-003")
	}

	return nil
}

// ListByProjectID implements document.ProjectFileRepository.
func (repo *ProjectFileGormRepository) ListByProjectID(ctx context.Context, projectID uint, pagination *query.Pagination) ([]*document.ProjectFile, int64, error) {
	baseQuery := repo.db.WithContext(ctx).
		Model(&dbschema.ProjectFile{}).
		Where("project_id = ? AND deleted_at IS NULL", projectID)

	// Count total
	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to count project files", err, "proj-file-list-001")
	}

	// Apply pagination and ordering
	q := repo.db.WithContext(ctx).
		Preload("DocumentContent").
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Order("display_order ASC, created_at ASC")

	if pagination != nil {
		if pagination.After != nil {
			q = q.Where("id > ?", *pagination.After)
		}
		if pagination.Limit != nil && *pagination.Limit > 0 {
			q = q.Limit(*pagination.Limit)
		}
	}

	// Execute query
	var rows []dbschema.ProjectFile
	if err := q.Find(&rows).Error; err != nil {
		return nil, 0, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to list project files", err, "proj-file-list-002")
	}

	// Convert to domain
	result := make([]*document.ProjectFile, len(rows))
	for i, row := range rows {
		result[i] = row.EtoD()
	}

	return result, total, nil
}

// UpdateDisplayOrders implements document.ProjectFileRepository.
func (repo *ProjectFileGormRepository) UpdateDisplayOrders(ctx context.Context, projectID uint, fileOrders map[string]int) error {
	tx := repo.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to begin transaction", tx.Error, "proj-file-reorder-001")
	}

	now := time.Now()
	for publicID, order := range fileOrders {
		result := tx.Model(&dbschema.ProjectFile{}).
			Where("public_id = ? AND project_id = ? AND deleted_at IS NULL", publicID, projectID).
			Updates(map[string]interface{}{
				"display_order": order,
				"updated_at":    now,
			})

		if result.Error != nil {
			tx.Rollback()
			return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
				"failed to update display order", result.Error, "proj-file-reorder-002")
		}
	}

	if err := tx.Commit().Error; err != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to commit transaction", err, "proj-file-reorder-003")
	}

	return nil
}
