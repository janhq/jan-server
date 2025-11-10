package conversation

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	domain "jan-server/services/response-api/internal/domain/conversation"
	"jan-server/services/response-api/internal/infrastructure/database/entities"
)

// Repository persists conversation metadata.
type Repository struct {
	db *gorm.DB
}

// NewRepository builds a conversation repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts the conversation record.
func (r *Repository) Create(ctx context.Context, conv *domain.Conversation) error {
	metadata, err := marshalJSON(conv.Metadata)
	if err != nil {
		return err
	}

	entity := &entities.Conversation{
		PublicID: conv.PublicID,
		UserID:   conv.UserID,
		Metadata: metadata,
	}

	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		return err
	}

	conv.ID = entity.ID
	conv.CreatedAt = entity.CreatedAt
	conv.UpdatedAt = entity.UpdatedAt
	return nil
}

// FindByPublicID fetches a conversation.
func (r *Repository) FindByPublicID(ctx context.Context, publicID string) (*domain.Conversation, error) {
	var entity entities.Conversation
	if err := r.db.WithContext(ctx).Where("public_id = ?", publicID).First(&entity).Error; err != nil {
		return nil, err
	}

	var metadata map[string]interface{}
	if len(entity.Metadata) > 0 {
		if err := json.Unmarshal(entity.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("unmarshal conversation metadata: %w", err)
		}
	}

	return &domain.Conversation{
		ID:        entity.ID,
		PublicID:  entity.PublicID,
		UserID:    entity.UserID,
		Metadata:  metadata,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}, nil
}
