package memoryrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"jan-server/mono/apps/backend/internal/domain/memory"
)

// MemoryConversationItem is the database schema for memory conversation items.
// This is separate from the main conversation.Item schema used by LLM-API.
type MemoryConversationItem struct {
	ID             string    `gorm:"primaryKey;type:varchar(36)"`
	ConversationID string    `gorm:"type:varchar(255);index;not null"`
	Role           string    `gorm:"type:varchar(50);not null"`
	Content        string    `gorm:"type:text"`
	ToolCalls      string    `gorm:"type:text"`
	CreatedAt      time.Time `gorm:"not null"`
}

func (MemoryConversationItem) TableName() string {
	return "memory_conversation_items"
}

func (r *Repository) CreateConversationItem(ctx context.Context, item *memory.ConversationItem) error {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}

	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}

	schema := &MemoryConversationItem{
		ID:             item.ID,
		ConversationID: item.ConversationID,
		Role:           item.Role,
		Content:        item.Content,
		ToolCalls:      item.ToolCalls,
		CreatedAt:      item.CreatedAt,
	}

	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		return fmt.Errorf("create conversation item: %w", err)
	}

	return nil
}

func (r *Repository) GetConversationItems(ctx context.Context, conversationID string) ([]memory.ConversationItem, error) {
	var rows []MemoryConversationItem
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query conversation items: %w", err)
	}

	items := make([]memory.ConversationItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, memory.ConversationItem{
			ID:             row.ID,
			ConversationID: row.ConversationID,
			Role:           row.Role,
			Content:        row.Content,
			ToolCalls:      row.ToolCalls,
			CreatedAt:      row.CreatedAt,
		})
	}

	return items, nil
}
