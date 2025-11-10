package conversation

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	domain "jan-server/services/response-api/internal/domain/conversation"
	"jan-server/services/response-api/internal/infrastructure/database/entities"
)

// ItemRepository persists conversation items.
type ItemRepository struct {
	db *gorm.DB
}

// NewItemRepository constructs the item repository.
func NewItemRepository(db *gorm.DB) *ItemRepository {
	return &ItemRepository{db: db}
}

// BulkInsert stores multiple conversation items in sequence order.
func (r *ItemRepository) BulkInsert(ctx context.Context, items []domain.Item) error {
	if len(items) == 0 {
		return nil
	}

	rows := make([]entities.ConversationItem, 0, len(items))
	for _, item := range items {
		content, err := marshalJSON(item.Content)
		if err != nil {
			return fmt.Errorf("marshal conversation item: %w", err)
		}
		rows = append(rows, entities.ConversationItem{
			ConversationID: item.ConversationID,
			Role:           string(item.Role),
			Status:         string(item.Status),
			Content:        content,
			Sequence:       item.Sequence,
			CreatedAt:      item.CreatedAt,
		})
	}

	return r.db.WithContext(ctx).Create(&rows).Error
}

// ListByConversationID returns items ordered by sequence.
func (r *ItemRepository) ListByConversationID(ctx context.Context, conversationID uint) ([]domain.Item, error) {
	var rows []entities.ConversationItem
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("sequence ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]domain.Item, 0, len(rows))
	for _, row := range rows {
		var content map[string]interface{}
		if len(row.Content) > 0 {
			_ = json.Unmarshal(row.Content, &content)
		}
		items = append(items, domain.Item{
			ID:             row.ID,
			ConversationID: row.ConversationID,
			Role:           domain.ItemRole(row.Role),
			Status:         domain.ItemStatus(row.Status),
			Content:        content,
			Sequence:       row.Sequence,
			CreatedAt:      row.CreatedAt,
		})
	}
	return items, nil
}
