package conversation

import (
	"context"
	"fmt"
	"time"

	"jan-server/mono/apps/backend/internal/domain/query"
	"jan-server/mono/apps/backend/internal/utils/idgen"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// ConversationService handles business logic for conversations
type ConversationService struct {
	repo      ConversationRepository
	validator *ConversationValidator
}

// NewConversationService creates a new conversation service
func NewConversationService(repo ConversationRepository) *ConversationService {
	return &ConversationService{
		repo:      repo,
		validator: NewConversationValidator(nil), // Use default config
	}
}

// ===============================================
// Core CRUD Operations
// ===============================================

// CreateConversation creates a conversation (core function - direct repository call)
func (s *ConversationService) CreateConversation(ctx context.Context, conv *Conversation) (*Conversation, error) {
	// Validate conversation
	if err := s.validator.ValidateConversation(conv); err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "conversation validation failed", err, "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d")
	}

	// Persist conversation
	if err := s.repo.Create(ctx, conv); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to create conversation")
	}

	return conv, nil
}

// GetConversationByPublicIDAndUserID retrieves a conversation by public ID and validates ownership (core function)
func (s *ConversationService) GetConversationByPublicIDAndUserID(ctx context.Context, publicID string, userID string) (*Conversation, error) {
	// Validate conversation ID format
	if err := s.validator.ValidateConversationID(publicID); err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "invalid conversation ID", err, "b2c3d4e5-f6a7-4b8c-9d0e-1f2a3b4c5d6e")
	}

	// Retrieve conversation
	conversation, err := s.repo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "conversation not found")
	}

	// Verify ownership
	if conversation.UserID != userID {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeNotFound, "conversation not found", nil, "c3d4e5f6-a7b8-4c9d-0e1f-2a3b4c5d6e7f")
	}

	return conversation, nil
}

// UpdateConversation updates a conversation (core function - direct repository call)
func (s *ConversationService) UpdateConversation(ctx context.Context, conv *Conversation) (*Conversation, error) {
	// Validate updated conversation
	if err := s.validator.ValidateConversation(conv); err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "conversation validation failed", err, "d4e5f6a7-b8c9-4d0e-1f2a-3b4c5d6e7f8a")
	}

	// Persist changes
	if err := s.repo.Update(ctx, conv); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to update conversation")
	}

	return conv, nil
}

// DeleteConversation deletes a conversation (core function - marks as deleted)
func (s *ConversationService) DeleteConversation(ctx context.Context, conv *Conversation) error {
	if err := s.repo.Delete(ctx, conv.ID); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to delete conversation")
	}
	return nil
}

// FindConversationsByFilter retrieves conversations using flexible filter criteria with pagination
func (s *ConversationService) FindConversationsByFilter(ctx context.Context, filter ConversationFilter, pagination *query.Pagination) ([]*Conversation, int64, error) {
	// Get conversations
	conversations, err := s.repo.FindByFilter(ctx, filter, pagination)
	if err != nil {
		return nil, 0, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to list conversations")
	}

	// Get total count
	total, err := s.repo.Count(ctx, filter)
	if err != nil {
		return nil, 0, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to count conversations")
	}

	return conversations, total, nil
}

// ===============================================
// Business Logic Operations (High-level)
// ===============================================

// CreateConversationInput represents the input for creating a conversation
type CreateConversationInput struct {
	UserID          string
	Title           *string
	Metadata        map[string]string
	Referrer        *string
	ProjectID       *uint
	ProjectPublicID *string
}

// UpdateConversationInput represents the input for updating a conversation
type UpdateConversationInput struct {
	Title           *string
	Metadata        map[string]string
	Referrer        *string
	ProjectID       *uint
	ProjectPublicID *string
}

// CreateConversationWithInput creates a new conversation with input validation
func (s *ConversationService) CreateConversationWithInput(ctx context.Context, input CreateConversationInput) (*Conversation, error) {
	// Generate public ID
	publicID, err := idgen.GenerateSecureID("conv", 16)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to generate conversation ID")
	}

	// Create conversation entity
	conversation := NewConversationWithProject(publicID, input.UserID, input.Title, input.Metadata, input.ProjectID)
	conversation.Referrer = input.Referrer               // optional metadata
	conversation.ProjectPublicID = input.ProjectPublicID // set project public ID

	// Use core function to create conversation
	return s.CreateConversation(ctx, conversation)
}

// UpdateConversationWithInput updates a conversation's properties with input validation
func (s *ConversationService) UpdateConversationWithInput(ctx context.Context, userID string, publicID string, input UpdateConversationInput) (*Conversation, error) {
	// Retrieve and verify ownership
	conversation, err := s.GetConversationByPublicIDAndUserID(ctx, publicID, userID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if input.Title != nil {
		// Update the title field directly
		conversation.Title = input.Title
	}

	if input.Metadata != nil {
		// Replace metadata entirely (not merged)
		conversation.Metadata = input.Metadata
	}

	if input.Referrer != nil {
		conversation.Referrer = input.Referrer
	}

	if input.ProjectID != nil {
		if *input.ProjectID == 0 {
			conversation.ProjectID = nil
			conversation.ProjectPublicID = nil
		} else {
			conversation.ProjectID = input.ProjectID
			conversation.ProjectPublicID = input.ProjectPublicID
		}
		// Clear cached instruction snapshot so the next request pulls the new project's instruction
		conversation.EffectiveInstructionSnapshot = nil
	}

	// Use core function to update conversation
	return s.UpdateConversation(ctx, conversation)
}

// DeleteConversationByID marks a conversation as deleted (soft delete)
func (s *ConversationService) DeleteConversationByID(ctx context.Context, userID string, publicID string) error {
	// Retrieve and verify ownership
	conversation, err := s.GetConversationByPublicIDAndUserID(ctx, publicID, userID)
	if err != nil {
		return err
	}

	// Use core function to delete conversation
	return s.DeleteConversation(ctx, conversation)
}

// DeleteAllConversationsByUserID deletes all conversations for a specific user.
// This is a destructive operation that removes all conversations owned by the user.
// Returns the count of deleted conversations.
func (s *ConversationService) DeleteAllConversationsByUserID(ctx context.Context, userID string) (int64, error) {
	// Validate userID
	if userID == "" {
		return 0, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "invalid user ID", nil, "delete-all-convs-invalid-user")
	}

	// Delete all conversations for this user
	deletedCount, err := s.repo.DeleteAllByUserID(ctx, userID)
	if err != nil {
		return 0, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to delete all conversations")
	}

	return deletedCount, nil
}

// ===============================================
// Item Management Methods
// ===============================================

// AddItemsToConversation adds multiple items to a conversation branch
func (s *ConversationService) AddItemsToConversation(ctx context.Context, conv *Conversation, branchName string, items []Item) ([]Item, error) {
	if len(items) == 0 {
		return []Item{}, nil
	}

	// Default to MAIN branch if not specified
	if branchName == "" {
		branchName = BranchMain
	}

	// Validate branch exists for non-MAIN branches
	if branchName != BranchMain {
		branch, err := s.repo.GetBranch(ctx, conv.ID, branchName)
		if err != nil || branch == nil {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeNotFound, fmt.Sprintf("branch not found: %s", branchName), nil, "e5f6a7b8-c9d0-4e1f-2a3b-4c5d6e7f8a9b")
		}
	}

	// Get current item count to determine starting sequence number
	currentCount, err := s.repo.CountItems(ctx, conv.ID, branchName)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to get item count")
	}

	// Generate IDs and assign sequence numbers for items
	itemPtrs := make([]*Item, len(items))
	for i := range items {
		if items[i].PublicID == "" {
			publicID, err := idgen.GenerateSecureID("msg", 16)
			if err != nil {
				return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to generate item ID")
			}
			items[i].PublicID = publicID
		}
		items[i].Object = "conversation.item"
		items[i].ConversationID = conv.ID
		items[i].Branch = branchName
		// Assign sequence number: start from current count + 1, increment for each item
		items[i].SequenceNumber = currentCount + i + 1
		itemPtrs[i] = &items[i]
	}

	// Add items to repository - use branch-aware method for all branches
	if err := s.repo.BulkAddItemsToBranch(ctx, conv.ID, branchName, itemPtrs); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to add items to branch")
	}

	// Update conversation's updated_at timestamp
	if len(items) > 0 {
		conv.UpdatedAt = items[len(items)-1].CreatedAt
		if err := s.repo.Update(ctx, conv); err != nil {
			// Log error but don't fail the operation
			_ = err
		}
	}

	return items, nil
}

// GetConversationItems retrieves items from a conversation branch with pagination
func (s *ConversationService) GetConversationItems(ctx context.Context, conv *Conversation, branchName string, pagination *query.Pagination) ([]Item, error) {
	// Get items from the branch with pagination applied at repository level
	items, err := s.repo.GetBranchItems(ctx, conv.ID, branchName, pagination)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to get items")
	}

	return convertItemPtrsToItems(items), nil
}

// CountConversationItems returns the number of items in a conversation branch.
func (s *ConversationService) CountConversationItems(ctx context.Context, conv *Conversation, branchName string) (int, error) {
	if conv == nil {
		return 0, nil
	}
	if branchName == "" {
		branchName = BranchMain
	}
	count, err := s.repo.CountItems(ctx, conv.ID, branchName)
	if err != nil {
		return 0, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to count conversation items")
	}
	return count, nil
}

// GetConversationItem retrieves a single item from a conversation
func (s *ConversationService) GetConversationItem(ctx context.Context, conv *Conversation, itemPublicID string) (*Item, error) {
	// Get the item directly by public ID from repository
	item, err := s.repo.GetItemByPublicID(ctx, conv.ID, itemPublicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "item not found")
	}

	return item, nil
}

// GetConversationItemByCallID retrieves a single item from a conversation by call_id
func (s *ConversationService) GetConversationItemByCallID(ctx context.Context, conv *Conversation, callID string) (*Item, error) {
	// Get the item directly by call ID from repository
	item, err := s.repo.GetItemByCallID(ctx, conv.ID, callID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "item not found by call_id")
	}

	return item, nil
}

// GetConversationItemByCallIDAndType retrieves a single item from a conversation by call_id and item type
func (s *ConversationService) GetConversationItemByCallIDAndType(ctx context.Context, conv *Conversation, callID string, itemType ItemType) (*Item, error) {
	// Get the item by call ID and type from repository
	item, err := s.repo.GetItemByCallIDAndType(ctx, conv.ID, callID, itemType)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "item not found by call_id and type")
	}

	return item, nil
}

// UpdateConversationItem updates an existing item in a conversation
func (s *ConversationService) UpdateConversationItem(ctx context.Context, conv *Conversation, item *Item) error {
	// Update the item in the repository
	if err := s.repo.UpdateItem(ctx, conv.ID, item); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to update item")
	}

	// Update conversation timestamp
	conv.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, conv); err != nil {
		// Log error but don't fail the update
		_ = err
	}

	return nil
}

// DeleteConversationItem deletes an item from a conversation
func (s *ConversationService) DeleteConversationItem(ctx context.Context, conv *Conversation, itemPublicID string) error {
	// Get the item to find its numeric ID
	item, err := s.repo.GetItemByPublicID(ctx, conv.ID, itemPublicID)
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to get item for deletion")
	}

	// Delete the item from the database
	if err := s.repo.DeleteItem(ctx, conv.ID, item.ID); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to delete item")
	}

	// Update conversation timestamp
	conv.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, conv); err != nil {
		// Log error but don't fail the deletion
		_ = err
	}

	return nil
}

// ===============================================
// Helper Functions
// ===============================================

// convertItemPtrsToItems converts []*Item to []Item
func convertItemPtrsToItems(itemPtrs []*Item) []Item {
	items := make([]Item, len(itemPtrs))
	for i, ptr := range itemPtrs {
		if ptr != nil {
			items[i] = *ptr
		}
	}
	return items
}
