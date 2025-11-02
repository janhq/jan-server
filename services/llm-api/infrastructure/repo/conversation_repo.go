package repo

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"jan-server/services/llm-api/domain"
)

type ConversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

// Create inserts a new conversation record.
func (r *ConversationRepository) Create(ctx context.Context, conversation *domain.Conversation) error {
	if err := r.db.WithContext(ctx).Create(conversation).Error; err != nil {
		return errors.Wrap(err, "create conversation")
	}
	return nil
}

// Get fetches a single conversation.
func (r *ConversationRepository) Get(ctx context.Context, id string) (*domain.Conversation, error) {
	var conversation domain.Conversation
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&conversation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "get conversation")
	}
	return &conversation, nil
}

// List returns a page of conversations ordered by creation date descending.
func (r *ConversationRepository) List(ctx context.Context, ownerID string, limit int, after string) ([]domain.Conversation, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	query := r.db.WithContext(ctx).
		Where("owner_principal_id = ?", ownerID).
		Order("created_at DESC, id DESC").
		Limit(limit + 1)

	if after != "" {
		if cursorTime, err := time.Parse(time.RFC3339Nano, after); err == nil {
			query = query.Where("created_at < ?", cursorTime)
		}
	}

	var conversations []domain.Conversation
	if err := query.Find(&conversations).Error; err != nil {
		return nil, "", errors.Wrap(err, "list conversations")
	}

	var nextCursor string
	if len(conversations) > limit {
		nextCursor = conversations[limit].CreatedAt.Format(time.RFC3339Nano)
		conversations = conversations[:limit]
	}

	return conversations, nextCursor, nil
}
