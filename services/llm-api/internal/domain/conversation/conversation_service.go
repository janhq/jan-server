package conversation

import (
	"context"
	"fmt"
	"time"

	"jan-server/services/llm-api/internal/domain/query"
	"jan-server/services/llm-api/internal/utils/idgen"
	"jan-server/services/llm-api/internal/utils/platformerrors"
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
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "conversation validation failed", err, "")
	}

	// Persist conversation
	if err := s.repo.Create(ctx, conv); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to create conversation")
	}

	return conv, nil
}

// GetConversationByPublicIDAndUserID retrieves a conversation by public ID and validates ownership (core function)
func (s *ConversationService) GetConversationByPublicIDAndUserID(ctx context.Context, publicID string, userID uint) (*Conversation, error) {
	// Validate conversation ID format
	if err := s.validator.ValidateConversationID(publicID); err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "invalid conversation ID", err, "")
	}

	// Retrieve conversation
	conversation, err := s.repo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "conversation not found")
	}

	// Verify ownership
	if conversation.UserID != userID {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeNotFound, "conversation not found", nil, "")
	}

	return conversation, nil
}

// UpdateConversation updates a conversation (core function - direct repository call)
func (s *ConversationService) UpdateConversation(ctx context.Context, conv *Conversation) (*Conversation, error) {
	// Validate updated conversation
	if err := s.validator.ValidateConversation(conv); err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "conversation validation failed", err, "")
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
	UserID          uint
	Title           *string
	Metadata        map[string]string
	Referrer        *string
	ProjectID       *uint
	ProjectPublicID *string
}

// UpdateConversationInput represents the input for updating a conversation
type UpdateConversationInput struct {
	Title    *string
	Metadata map[string]string
	Referrer *string
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
func (s *ConversationService) UpdateConversationWithInput(ctx context.Context, userID uint, publicID string, input UpdateConversationInput) (*Conversation, error) {
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

	// Use core function to update conversation
	return s.UpdateConversation(ctx, conversation)
}

// DeleteConversationByID marks a conversation as deleted (soft delete)
func (s *ConversationService) DeleteConversationByID(ctx context.Context, userID uint, publicID string) error {
	// Retrieve and verify ownership
	conversation, err := s.GetConversationByPublicIDAndUserID(ctx, publicID, userID)
	if err != nil {
		return err
	}

	// Use core function to delete conversation
	return s.DeleteConversation(ctx, conversation)
}

// ===============================================
// Item Management Methods
// ===============================================

// AddItemsToConversation adds multiple items to a conversation branch
func (s *ConversationService) AddItemsToConversation(ctx context.Context, conv *Conversation, branchName string, items []Item) ([]Item, error) {
	if len(items) == 0 {
		return []Item{}, nil
	}

	// Validate branch exists (for now, only MAIN is supported)
	if branchName != BranchMain {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeNotFound, fmt.Sprintf("branch not found: %s", branchName), nil, "")
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

	// Add items to repository
	if branchName == BranchMain || branchName == "" {
		if err := s.repo.BulkAddItems(ctx, conv.ID, itemPtrs); err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to add items")
		}
	} else {
		if err := s.repo.BulkAddItemsToBranch(ctx, conv.ID, branchName, itemPtrs); err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to add items to branch")
		}
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

// GetConversationItem retrieves a single item from a conversation
func (s *ConversationService) GetConversationItem(ctx context.Context, conv *Conversation, itemPublicID string) (*Item, error) {
	// Get the item directly by public ID from repository
	item, err := s.repo.GetItemByPublicID(ctx, conv.ID, itemPublicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "item not found")
	}

	return item, nil
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
