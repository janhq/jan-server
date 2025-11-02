package conversation

import (
	"context"
	"fmt"
	"time"

	"jan-server/services/llm-api/domain/query"
)

// ===============================================
// Conversation Types
// ===============================================

type ConversationStatus string

const (
	ConversationStatusActive   ConversationStatus = "active"
	ConversationStatusArchived ConversationStatus = "archived"
	ConversationStatusDeleted  ConversationStatus = "deleted"
)

// ConversationBranch represents a specific flow/path in a conversation
const (
	BranchMain = "MAIN" // Default main conversation flow
)

// ===============================================
// Conversation Structure
// ===============================================

type Conversation struct {
	ID             uint                      `json:"-"`
	PublicID       string                    `json:"id"`
	Object         string                    `json:"object"`
	Title          *string                   `json:"title,omitempty"`
	UserID         uint                      `json:"-"`
	Status         ConversationStatus        `json:"status"`
	Items          []Item                    `json:"items,omitempty"`
	Branches       map[string][]Item         `json:"branches,omitempty"`
	ActiveBranch   string                    `json:"active_branch,omitempty"`
	BranchMetadata map[string]BranchMetadata `json:"branch_metadata,omitempty"`
	Metadata       map[string]string         `json:"metadata,omitempty"`
	Referrer       *string                   `json:"referrer,omitempty"`
	IsPrivate      bool                      `json:"is_private"`
	CreatedAt      time.Time                 `json:"created_at"`
	UpdatedAt      time.Time                 `json:"updated_at"`
}

// BranchMetadata contains information about a conversation branch
type BranchMetadata struct {
	Name             string     `json:"name"`
	Description      *string    `json:"description,omitempty"`
	ParentBranch     *string    `json:"parent_branch,omitempty"`
	ForkedAt         *time.Time `json:"forked_at,omitempty"`
	ForkedFromItemID *string    `json:"forked_from_item_id,omitempty"`
	ItemCount        int        `json:"item_count"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ===============================================
// Conversation Repository
// ===============================================

type ConversationFilter struct {
	ID       *uint
	PublicID *string
	UserID   *uint
	Referrer *string
}

type ConversationRepository interface {
	Create(ctx context.Context, conversation *Conversation) error
	FindByFilter(ctx context.Context, filter ConversationFilter, pagination *query.Pagination) ([]*Conversation, error)
	Count(ctx context.Context, filter ConversationFilter) (int64, error)
	FindByID(ctx context.Context, id uint) (*Conversation, error)
	FindByPublicID(ctx context.Context, publicID string) (*Conversation, error)
	Update(ctx context.Context, conversation *Conversation) error
	Delete(ctx context.Context, id uint) error

	// Item operations (legacy - assumes MAIN branch)
	AddItem(ctx context.Context, conversationID uint, item *Item) error
	SearchItems(ctx context.Context, conversationID uint, query string) ([]*Item, error)
	BulkAddItems(ctx context.Context, conversationID uint, items []*Item) error
	GetItemByID(ctx context.Context, conversationID uint, itemID uint) (*Item, error)
	GetItemByPublicID(ctx context.Context, conversationID uint, publicID string) (*Item, error)
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
	GetBranchItems(ctx context.Context, conversationID uint, branchName string, pagination *query.Pagination) ([]*Item, error)
	BulkAddItemsToBranch(ctx context.Context, conversationID uint, branchName string, items []*Item) error

	// Fork operation
	ForkBranch(ctx context.Context, conversationID uint, sourceBranch, newBranch string, fromItemID string, description *string) error

	// Item rating operations
	RateItem(ctx context.Context, conversationID uint, itemID string, rating ItemRating, comment *string) error
	GetItemRating(ctx context.Context, conversationID uint, itemID string) (*ItemRating, error)
	RemoveItemRating(ctx context.Context, conversationID uint, itemID string) error
}

// ===============================================
// Conversation Factory Functions
// ===============================================

// NewConversation creates a new conversation with the given parameters
func NewConversation(publicID string, userID uint, title *string, metadata map[string]string) *Conversation {
	now := time.Now()

	if metadata == nil {
		metadata = make(map[string]string)
	}

	conv := &Conversation{
		PublicID:       publicID,
		Object:         "conversation",
		Title:          title,
		UserID:         userID,
		Status:         ConversationStatusActive,
		ActiveBranch:   BranchMain,
		Branches:       make(map[string][]Item),
		BranchMetadata: make(map[string]BranchMetadata),
		Metadata:       metadata,
		IsPrivate:      false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Initialize MAIN branch metadata
	conv.BranchMetadata[BranchMain] = BranchMetadata{
		Name:             BranchMain,
		Description:      nil,
		ParentBranch:     nil,
		ForkedAt:         nil,
		ForkedFromItemID: nil,
		ItemCount:        0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	return conv
}

// GetActiveBranchItems returns items from the currently active branch
func (c *Conversation) GetActiveBranchItems() []Item {
	if c.Branches != nil {
		if items, exists := c.Branches[c.ActiveBranch]; exists {
			return items
		}
	}
	return c.Items
}

// GetBranchItems returns items from a specific branch
func (c *Conversation) GetBranchItems(branchName string) []Item {
	if c.Branches != nil {
		if items, exists := c.Branches[branchName]; exists {
			return items
		}
	}
	if branchName == BranchMain {
		return c.Items
	}
	return []Item{}
}

// AddItemToActiveBranch adds an item to the currently active branch
func (c *Conversation) AddItemToActiveBranch(item Item) {
	if c.Branches == nil {
		c.Branches = make(map[string][]Item)
	}

	item.Branch = c.ActiveBranch
	item.SequenceNumber = len(c.Branches[c.ActiveBranch])

	c.Branches[c.ActiveBranch] = append(c.Branches[c.ActiveBranch], item)

	if c.BranchMetadata != nil {
		if meta, exists := c.BranchMetadata[c.ActiveBranch]; exists {
			meta.ItemCount++
			meta.UpdatedAt = time.Now()
			c.BranchMetadata[c.ActiveBranch] = meta
		}
	}
}

// SwitchBranch changes the active branch
func (c *Conversation) SwitchBranch(branchName string) error {
	if c.BranchMetadata != nil {
		if _, exists := c.BranchMetadata[branchName]; !exists {
			return fmt.Errorf("branch not found: %s", branchName)
		}
	}
	c.ActiveBranch = branchName
	return nil
}

// CreateBranch creates a new branch (fork) from an existing branch
func (c *Conversation) CreateBranch(newBranchName, sourceBranch, fromItemID string, description *string) error {
	if c.Branches == nil {
		c.Branches = make(map[string][]Item)
	}
	if c.BranchMetadata == nil {
		c.BranchMetadata = make(map[string]BranchMetadata)
	}

	if _, exists := c.BranchMetadata[newBranchName]; exists {
		return fmt.Errorf("branch already exists: %s", newBranchName)
	}

	sourceItems := c.GetBranchItems(sourceBranch)

	forkIndex := -1
	for i, item := range sourceItems {
		if item.PublicID == fromItemID {
			forkIndex = i
			break
		}
	}

	if forkIndex == -1 && fromItemID != "" {
		return fmt.Errorf("item not found: %s", fromItemID)
	}

	var newBranchItems []Item
	if forkIndex >= 0 {
		newBranchItems = make([]Item, forkIndex+1)
		for i := 0; i <= forkIndex; i++ {
			item := sourceItems[i]
			item.Branch = newBranchName
			item.SequenceNumber = i
			newBranchItems[i] = item
		}
	}

	c.Branches[newBranchName] = newBranchItems

	now := time.Now()
	c.BranchMetadata[newBranchName] = BranchMetadata{
		Name:             newBranchName,
		Description:      description,
		ParentBranch:     &sourceBranch,
		ForkedAt:         &now,
		ForkedFromItemID: &fromItemID,
		ItemCount:        len(newBranchItems),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	return nil
}

// GenerateEditBranchName generates a unique branch name for conversation edits
func GenerateEditBranchName(conversationID uint) string {
	return fmt.Sprintf("EDIT_%d_%d", conversationID, time.Now().Unix())
}
