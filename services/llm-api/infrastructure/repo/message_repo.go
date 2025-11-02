package repo

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"jan-server/services/llm-api/domain"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Append stores a new message for a conversation.
func (r *MessageRepository) Append(ctx context.Context, msg *domain.Message) error {
	if err := r.db.WithContext(ctx).Create(msg).Error; err != nil {
		return errors.Wrap(err, "append message")
	}
	return nil
}

// List returns paginated messages sorted ascending by creation time.
func (r *MessageRepository) List(ctx context.Context, conversationID string, limit int, after string) ([]domain.Message, string, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC, id ASC").
		Limit(limit + 1)

	if after != "" {
		if cursorTime, err := time.Parse(time.RFC3339Nano, after); err == nil {
			query = query.Where("created_at > ?", cursorTime)
		}
	}

	var messages []domain.Message
	if err := query.Find(&messages).Error; err != nil {
		return nil, "", errors.Wrap(err, "list messages")
	}

	var nextCursor string
	if len(messages) > limit {
		nextCursor = messages[limit].CreatedAt.Format(time.RFC3339Nano)
		messages = messages[:limit]
	}

	return messages, nextCursor, nil
}
