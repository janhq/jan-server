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

type DocumentContentGormRepository struct {
	db *gorm.DB
}

var _ document.DocumentContentRepository = (*DocumentContentGormRepository)(nil)

func NewDocumentContentGormRepository(db *gorm.DB) document.DocumentContentRepository {
	return &DocumentContentGormRepository{db: db}
}

// Create implements document.DocumentContentRepository.
func (repo *DocumentContentGormRepository) Create(ctx context.Context, doc *document.DocumentContent) error {
	dbDoc := dbschema.NewSchemaDocumentContent(doc)
	if err := repo.db.WithContext(ctx).Create(dbDoc).Error; err != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to create document content", err, "doc-content-create-001")
	}
	doc.ID = dbDoc.ID
	doc.CreatedAt = dbDoc.CreatedAt
	doc.UpdatedAt = dbDoc.UpdatedAt
	return nil
}

// GetByPublicID implements document.DocumentContentRepository.
func (repo *DocumentContentGormRepository) GetByPublicID(ctx context.Context, publicID string) (*document.DocumentContent, error) {
	var dbDoc dbschema.DocumentContent
	err := repo.db.WithContext(ctx).
		Where("public_id = ?", publicID).
		First(&dbDoc).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound,
				fmt.Sprintf("document content %s not found", publicID), err, "doc-content-notfound-001")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to find document content", err, "doc-content-get-001")
	}
	return dbDoc.EtoD(), nil
}

// GetByMediaObjectID implements document.DocumentContentRepository.
func (repo *DocumentContentGormRepository) GetByMediaObjectID(ctx context.Context, mediaObjectID string, userID uint) (*document.DocumentContent, error) {
	var dbDoc dbschema.DocumentContent
	err := repo.db.WithContext(ctx).
		Where("media_object_id = ? AND user_id = ?", mediaObjectID, userID).
		Order("CASE WHEN processing_status = 'completed' THEN 1 ELSE 0 END DESC").
		Order("updated_at DESC").
		First(&dbDoc).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound,
				fmt.Sprintf("document content for media %s not found", mediaObjectID), err, "doc-content-notfound-002")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to find document content by media object ID", err, "doc-content-get-002")
	}
	return dbDoc.EtoD(), nil
}

// Update implements document.DocumentContentRepository.
func (repo *DocumentContentGormRepository) Update(ctx context.Context, doc *document.DocumentContent) error {
	dbDoc := dbschema.NewSchemaDocumentContent(doc)
	dbDoc.UpdatedAt = time.Now()

	err := repo.db.WithContext(ctx).Model(&dbschema.DocumentContent{}).
		Where("public_id = ?", doc.PublicID).
		Updates(map[string]interface{}{
			"processing_status": dbDoc.ProcessingStatus,
			"extracted_text":    dbDoc.ExtractedText,
			"extraction_model":  dbDoc.ExtractionModel,
			"page_count":        dbDoc.PageCount,
			"word_count":        dbDoc.WordCount,
			"error_message":     dbDoc.ErrorMessage,
			"updated_at":        dbDoc.UpdatedAt,
		}).Error

	if err != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to update document content", err, "doc-content-update-001")
	}

	doc.UpdatedAt = dbDoc.UpdatedAt
	return nil
}

// Delete implements document.DocumentContentRepository.
func (repo *DocumentContentGormRepository) Delete(ctx context.Context, publicID string) error {
	result := repo.db.WithContext(ctx).
		Where("public_id = ?", publicID).
		Delete(&dbschema.DocumentContent{})

	if result.Error != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to delete document content", result.Error, "doc-content-delete-001")
	}

	if result.RowsAffected == 0 {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound,
			fmt.Sprintf("document content %s not found", publicID), nil, "doc-content-notfound-003")
	}

	return nil
}

// List implements document.DocumentContentRepository.
func (repo *DocumentContentGormRepository) List(ctx context.Context, filter document.DocumentContentFilter, pagination *query.Pagination) ([]*document.DocumentContent, int64, error) {
	baseQuery := repo.db.WithContext(ctx).Model(&dbschema.DocumentContent{})

	// Apply filters
	if filter.UserID != nil {
		baseQuery = baseQuery.Where("user_id = ?", *filter.UserID)
	}
	if filter.ProcessingStatus != nil {
		baseQuery = baseQuery.Where("processing_status = ?", string(*filter.ProcessingStatus))
	}
	if filter.MediaObjectID != nil {
		baseQuery = baseQuery.Where("media_object_id = ?", *filter.MediaObjectID)
	}

	// Count total
	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to count document contents", err, "doc-content-list-001")
	}

	// Apply pagination
	q := baseQuery
	if pagination != nil {
		if pagination.After != nil {
			if pagination.Order == "desc" {
				q = q.Where("id < ?", *pagination.After)
			} else {
				q = q.Where("id > ?", *pagination.After)
			}
		}

		if pagination.Order == "desc" {
			q = q.Order("created_at DESC")
		} else {
			q = q.Order("created_at ASC")
		}

		if pagination.Limit != nil && *pagination.Limit > 0 {
			q = q.Limit(*pagination.Limit)
		}
	} else {
		q = q.Order("created_at DESC")
	}

	// Execute query
	var rows []dbschema.DocumentContent
	if err := q.Find(&rows).Error; err != nil {
		return nil, 0, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeInternal,
			"failed to list document contents", err, "doc-content-list-002")
	}

	// Convert to domain
	result := make([]*document.DocumentContent, len(rows))
	for i, row := range rows {
		result[i] = row.EtoD()
	}

	return result, total, nil
}
