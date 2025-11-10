package conversation

import "context"

// Repository exposes CRUD operations for conversation metadata.
type Repository interface {
	Create(ctx context.Context, conversation *Conversation) error
	FindByPublicID(ctx context.Context, publicID string) (*Conversation, error)
}

// ItemRepository persists individual conversation messages.
type ItemRepository interface {
	BulkInsert(ctx context.Context, items []Item) error
	ListByConversationID(ctx context.Context, conversationID uint) ([]Item, error)
}
