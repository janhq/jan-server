package conversation

import "context"

// ConversationFilter for filtering conversations
type ConversationFilter struct {
	ID        *uint
	PublicID  *string
	UserID    *uint
	ProjectID *uint
	Referrer  *string
	Status    *ConversationStatus
}

// ItemFilter for filtering items
type ItemFilter struct {
	ID             *uint
	PublicID       *string
	CallID         *string
	ConversationID *uint
	Role           *ItemRole
	ResponseID     *uint
	Branch         *string
	Type           *ItemType
}

// Pagination for paginated queries
type Pagination struct {
	Page     int
	PageSize int
}

// Repository exposes CRUD operations for conversation metadata.
type Repository interface {
	Create(ctx context.Context, conversation *Conversation) error
	FindByPublicID(ctx context.Context, publicID string) (*Conversation, error)
	FindByID(ctx context.Context, id uint) (*Conversation, error)
	FindByFilter(ctx context.Context, filter ConversationFilter, pagination *Pagination) ([]*Conversation, error)
	Count(ctx context.Context, filter ConversationFilter) (int64, error)
	Update(ctx context.Context, conversation *Conversation) error
	Delete(ctx context.Context, id uint) error
	DeleteAllByUserID(ctx context.Context, userID uint) (int64, error)

	// Item operations (legacy - assumes MAIN branch)
	AddItem(ctx context.Context, conversationID uint, item *Item) error
	BulkAddItems(ctx context.Context, conversationID uint, items []*Item) error
	GetItemByID(ctx context.Context, conversationID uint, itemID uint) (*Item, error)
	GetItemByPublicID(ctx context.Context, conversationID uint, publicID string) (*Item, error)
	GetItemByCallID(ctx context.Context, conversationID uint, callID string) (*Item, error)
	GetItemByCallIDAndType(ctx context.Context, conversationID uint, callID string, itemType ItemType) (*Item, error)
	UpdateItem(ctx context.Context, conversationID uint, item *Item) error
	DeleteItem(ctx context.Context, conversationID uint, itemID uint) error
	CountItems(ctx context.Context, conversationID uint, branchName string) (int, error)

	// Branch operations
	CreateBranch(ctx context.Context, conversationID uint, branchName string, metadata *BranchMetadata) error
	GetBranch(ctx context.Context, conversationID uint, branchName string) (*BranchMetadata, error)
	ListBranches(ctx context.Context, conversationID uint) ([]*BranchMetadata, error)
	DeleteBranch(ctx context.Context, conversationID uint, branchName string) error
	SetActiveBranch(ctx context.Context, conversationID uint, branchName string) error

	// Branch item operations
	AddItemToBranch(ctx context.Context, conversationID uint, branchName string, item *Item) error
	GetBranchItems(ctx context.Context, conversationID uint, branchName string, pagination *Pagination) ([]*Item, error)
	BulkAddItemsToBranch(ctx context.Context, conversationID uint, branchName string, items []*Item) error

	// Fork operation
	ForkBranch(ctx context.Context, conversationID uint, sourceBranch, newBranch string, fromItemID string, description *string) error

	// SwapBranchToMain swaps a branch with MAIN
	SwapBranchToMain(ctx context.Context, conversationID uint, branchToPromote string) (oldMainBackupName string, err error)

	// Item rating operations
	RateItem(ctx context.Context, conversationID uint, itemID string, rating ItemRating, comment *string) error
	GetItemRating(ctx context.Context, conversationID uint, itemID string) (*ItemRating, error)
	RemoveItemRating(ctx context.Context, conversationID uint, itemID string) error
}

// ItemRepository persists individual conversation messages.
type ItemRepository interface {
	Create(ctx context.Context, item *Item) error
	BulkInsert(ctx context.Context, items []Item) error
	FindByID(ctx context.Context, id uint) (*Item, error)
	FindByPublicID(ctx context.Context, publicID string) (*Item, error)
	FindByConversationID(ctx context.Context, conversationID uint) ([]*Item, error)
	FindByFilter(ctx context.Context, filter ItemFilter, pagination *Pagination) ([]*Item, error)
	Count(ctx context.Context, filter ItemFilter) (int64, error)
	ListByConversationID(ctx context.Context, conversationID uint) ([]Item, error)
	Update(ctx context.Context, item *Item) error
	Delete(ctx context.Context, id uint) error
	BulkCreate(ctx context.Context, items []*Item) error
	CountByConversation(ctx context.Context, conversationID uint) (int64, error)
	ExistsByIDAndConversation(ctx context.Context, itemID uint, conversationID uint) (bool, error)
}
