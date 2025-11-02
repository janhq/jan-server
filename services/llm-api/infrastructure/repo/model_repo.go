package repo

import (
	"context"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"jan-server/services/llm-api/domain"
)

type ModelRepository struct {
	db *gorm.DB
}

func NewModelRepository(db *gorm.DB) *ModelRepository {
	return &ModelRepository{db: db}
}

// ListActive returns all active models.
func (r *ModelRepository) ListActive(ctx context.Context) ([]domain.Model, error) {
	var models []domain.Model
	if err := r.db.WithContext(ctx).Where("active = ?", true).Order("display_name ASC").Find(&models).Error; err != nil {
		return nil, errors.Wrap(err, "list models")
	}
	return models, nil
}

// Upsert ensures a model record exists.
func (r *ModelRepository) Upsert(ctx context.Context, m *domain.Model) error {
	// Ensure capabilities is stored as JSONB, not a record/array literal.
	assignments := map[string]interface{}{
		"provider":     m.Provider,
		"display_name": m.DisplayName,
		"family":       m.Family,
		// Let GORM marshal the []string into JSON for JSONB column
		"capabilities": m.Capabilities,
		"active":       m.Active,
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.Assignments(assignments),
	}).Create(&domain.Model{
		ID:           m.ID,
		Provider:     m.Provider,
		DisplayName:  m.DisplayName,
		Family:       m.Family,
		Capabilities: m.Capabilities,
		Active:       m.Active,
	}).Error; err != nil {
		return errors.Wrap(err, "upsert model")
	}
	return nil
}

// Get returns metadata for a single model.
func (r *ModelRepository) Get(ctx context.Context, id string) (*domain.Model, error) {
	var model domain.Model
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "get model")
	}
	return &model, nil
}
