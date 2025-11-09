package sample

import (
	"context"
	"errors"

	"gorm.io/gorm"

	domain "jan-server/services/template-api/internal/domain/sample"
	"jan-server/services/template-api/internal/infrastructure/database/entities"
)

// PostgresRepository persists samples via PostgreSQL using GORM.
type PostgresRepository struct {
	db *gorm.DB
}

// NewPostgresRepository creates a repository backed by the provided DB.
func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// FetchLatest returns the most recently created sample row.
func (r *PostgresRepository) FetchLatest(ctx context.Context) (domain.Sample, error) {
	var record entities.Sample
	err := r.db.WithContext(ctx).Order("created_at DESC").First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Sample{}, err
		}
		return domain.Sample{}, err
	}

	return domain.Sample{
		ID:      record.ID,
		Message: record.Message,
	}, nil
}
