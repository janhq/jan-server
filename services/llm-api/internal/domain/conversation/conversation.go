package conversation

import (
	"context"
	"fmt"
	"time"

	"jan-server/services/llm-api/internal/domain/query"
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
// Used to support editing items while maintaining conversation history
const (
	BranchMain = "MAIN" // Default main conversation flow
)

// Branch names for edited conversations follow pattern: "EDIT_1", "EDIT_2", etc.
// Or custom names for specific purposes

// ===============================================
// Conversation Structure
// ===============================================

type Conversation struct {
	ID             uint                      `json:"-"`
	PublicID       string                    `json:"id"`     // OpenAI-compatible string ID like "conv_abc123"
	Object         string                    `json:"object"` // Always "conversation" for OpenAI compatibility
	Title          *string                   `json:"title,omitempty"`
	UserID         uint                      `json:"-"`
	ProjectID      *uint                     `json:"-"` // Optional project grouping
	Status         ConversationStatus        `json:"status"`
	Items          []Item                    `json:"items,omitempty"`           // Legacy: items without branch (defaults to MAIN)
	Branches       map[string][]Item         `json:"branches,omitempty"`        // Branched items organized by branch name
	ActiveBranch   string                    `json:"active_branch,omitempty"`   // Currently active branch (default: "MAIN")
	BranchMetadata map[string]BranchMetadata `json:"branch_metadata,omitempty"` // Metadata about each branch
	Metadata       map[string]string         `json:"metadata,omitempty"`
	Referrer       *string                   `json:"referrer,omitempty"`
	IsPrivate      bool                      `json:"is_private"`

	// Project instruction inheritance
	InstructionVersion           int     `json:"instruction_version"`                      // Version of project instruction when conversation was created
	EffectiveInstructionSnapshot *string `json:"effective_instruction_snapshot,omitempty"` // Snapshot of merged instruction for reproducibility

	CreatedAt time.Time `json:"created_at"` // Unix timestamp for OpenAI compatibility
	UpdatedAt time.Time `json:"updated_at"` // Unix timestamp for OpenAI compatibility
}

// BranchMetadata contains information about a conversation branch
type BranchMetadata struct {
	Name             string     `json:"name"`                          // Branch identifier (MAIN, EDIT_1, etc.)
	Description      *string    `json:"description,omitempty"`         // Optional description of this branch
	ParentBranch     *string    `json:"parent_branch,omitempty"`       // Branch this was forked from
	ForkedAt         *time.Time `json:"forked_at,omitempty"`           // When this branch was created
	ForkedFromItemID *string    `json:"forked_from_item_id,omitempty"` // Item ID where fork occurred
	ItemCount        int        `json:"item_count"`                    // Number of items in this branch
	CreatedAt        time.Time  `json:"created_at"`                    // Branch creation time
	UpdatedAt        time.Time  `json:"updated_at"`                    // Last update time
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
	SearchItems(ctx context.Context, conversationID uint, query string) ([]*Item, error) // TODO: Implement search functionality
	BulkAddItems(ctx context.Context, conversationID uint, items []*Item) error
	GetItemByID(ctx context.Context, conversationID uint, itemID uint) (*Item, error)
	GetItemByPublicID(ctx context.Context, conversationID uint, publicID string) (*Item, error)
	DeleteItem(ctx context.Context, conversationID uint, itemID uint) error
	CountItems(ctx context.Context, conversationID uint, branchName string) (int, error)

	// Branch operations - TODO: Implement branching UI and endpoints
	CreateBranch(ctx context.Context, conversationID uint, branchName string, metadata *BranchMetadata) error
	GetBranch(ctx context.Context, conversationID uint, branchName string) (*BranchMetadata, error)
	ListBranches(ctx context.Context, conversationID uint) ([]*BranchMetadata, error)
	DeleteBranch(ctx context.Context, conversationID uint, branchName string) error
	SetActiveBranch(ctx context.Context, conversationID uint, branchName string) error

	// Branch item operations
	AddItemToBranch(ctx context.Context, conversationID uint, branchName string, item *Item) error
	GetBranchItems(ctx context.Context, conversationID uint, branchName string, pagination *query.Pagination) ([]*Item, error)
	BulkAddItemsToBranch(ctx context.Context, conversationID uint, branchName string, items []*Item) error

	// Fork operation - creates a new branch from an existing branch at a specific item
	// TODO: Implement forking functionality for conversation editing
	ForkBranch(ctx context.Context, conversationID uint, sourceBranch, newBranch string, fromItemID string, description *string) error

	// Item rating operations - TODO: Implement item rating/feedback system
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

	// Initialize metadata if nil
	if metadata == nil {
		metadata = make(map[string]string)
	}

	conv := &Conversation{
		PublicID:                     publicID,
		Object:                       "conversation",
		Title:                        title,
		UserID:                       userID,
		ProjectID:                    nil,
		Status:                       ConversationStatusActive,
		ActiveBranch:                 BranchMain,
		Branches:                     make(map[string][]Item),
		BranchMetadata:               make(map[string]BranchMetadata),
		Metadata:                     metadata,
		IsPrivate:                    false,
		InstructionVersion:           1,
		EffectiveInstructionSnapshot: nil,
		CreatedAt:                    now,
		UpdatedAt:                    now,
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
// TODO: Currently unused - will be needed when implementing conversation branching UI
func (c *Conversation) GetActiveBranchItems() []Item {
	if c.Branches != nil {
		if items, exists := c.Branches[c.ActiveBranch]; exists {
			return items
		}
	}
	// Fallback to legacy Items field
	return c.Items
}

// GetBranchItems returns items from a specific branch
func (c *Conversation) GetBranchItems(branchName string) []Item {
	if c.Branches != nil {
		if items, exists := c.Branches[branchName]; exists {
			return items
		}
	}
	// If requesting MAIN and Branches is empty, return legacy Items
	if branchName == BranchMain {
		return c.Items
	}
	return []Item{}
}

// AddItemToActiveBranch adds an item to the currently active branch
// TODO: Currently unused - will be needed when implementing conversation branching UI
func (c *Conversation) AddItemToActiveBranch(item Item) {
	if c.Branches == nil {
		c.Branches = make(map[string][]Item)
	}

	// Set branch on item
	item.Branch = c.ActiveBranch
	item.SequenceNumber = len(c.Branches[c.ActiveBranch])

	c.Branches[c.ActiveBranch] = append(c.Branches[c.ActiveBranch], item)

	// Update branch metadata
	if c.BranchMetadata != nil {
		if meta, exists := c.BranchMetadata[c.ActiveBranch]; exists {
			meta.ItemCount++
			meta.UpdatedAt = time.Now()
			c.BranchMetadata[c.ActiveBranch] = meta
		}
	}
}

// SwitchBranch changes the active branch
// TODO: Currently unused - will be needed when implementing conversation branching UI
func (c *Conversation) SwitchBranch(branchName string) error {
	// Check if branch exists
	if c.BranchMetadata != nil {
		if _, exists := c.BranchMetadata[branchName]; !exists {
			return fmt.Errorf("branch not found: %s", branchName)
		}
	}
	c.ActiveBranch = branchName
	return nil
}

// CreateBranch creates a new branch (fork) from an existing branch
// TODO: Currently unused - will be needed when implementing conversation branching UI
func (c *Conversation) CreateBranch(newBranchName, sourceBranch, fromItemID string, description *string) error {
	if c.Branches == nil {
		c.Branches = make(map[string][]Item)
	}
	if c.BranchMetadata == nil {
		c.BranchMetadata = make(map[string]BranchMetadata)
	}

	// Check if branch already exists
	if _, exists := c.BranchMetadata[newBranchName]; exists {
		return fmt.Errorf("branch already exists: %s", newBranchName)
	}

	// Get source branch items
	sourceItems := c.GetBranchItems(sourceBranch)

	// Find the fork point
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

	// Copy items up to fork point
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

	// Create branch metadata
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
// TODO: Currently unused - will be needed when implementing conversation branching UI
func GenerateEditBranchName(conversationID uint) string {
	return fmt.Sprintf("EDIT_%d_%d", conversationID, time.Now().Unix())
}
