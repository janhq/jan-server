package conversation

import (
	"context"
	"time"

	"jan-server/services/llm-api/internal/utils/idgen"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

// MessageActionService handles message edit, regenerate, and delete operations
type MessageActionService struct {
	convRepo ConversationRepository
}

// NewMessageActionService creates a new message action service
func NewMessageActionService(convRepo ConversationRepository) *MessageActionService {
	return &MessageActionService{
		convRepo: convRepo,
	}
}

// EditResult contains the result of an edit message operation
type EditResult struct {
	NewBranch      string `json:"new_branch"`      // Always "MAIN" after swap
	OldMainBackup  string `json:"old_main_backup"` // Backup name for old MAIN
	UserItem       *Item  `json:"user_item"`
	ConversationID string `json:"conversation_id"`
}

// RegenerateResult contains the result of a regenerate operation
type RegenerateResult struct {
	NewBranch     string `json:"new_branch"`      // Always "MAIN" after swap
	OldMainBackup string `json:"old_main_backup"` // Backup name for old MAIN
	ConvID        string `json:"conversation_id"`
	UserItemID    string `json:"user_item_id"` // The user message to regenerate from
}

// EditMessage creates a new branch from the edited message point
// It creates a fork of the conversation at the specified item with new content
func (s *MessageActionService) EditMessage(ctx context.Context, conv *Conversation, itemPublicID string, newContent string) (*EditResult, error) {
	// Get the original item to verify it exists and is a user message
	originalItem, err := s.convRepo.GetItemByPublicID(ctx, conv.ID, itemPublicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "item not found")
	}

	// Verify it's a user message
	if originalItem.Role == nil || *originalItem.Role != ItemRoleUser {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "can only edit user messages", nil, "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d")
	}

	// Generate new branch name
	newBranchName := GenerateEditBranchName(conv.ID)

	// Fork the branch at the item before this one (parent item)
	// We need to find the previous item in the sequence
	branchItems, err := s.convRepo.GetBranchItems(ctx, conv.ID, conv.ActiveBranch, nil)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to get branch items")
	}

	// Find the item and determine fork point (one item before)
	forkFromItemID := ""
	for i, item := range branchItems {
		if item.PublicID == itemPublicID {
			if i > 0 {
				forkFromItemID = branchItems[i-1].PublicID
			}
			break
		}
	}

	// Create the new branch
	now := time.Now()
	description := "Edited message branch"
	metadata := &BranchMetadata{
		Name:             newBranchName,
		Description:      &description,
		ParentBranch:     &conv.ActiveBranch,
		ForkedAt:         &now,
		ForkedFromItemID: &forkFromItemID,
		ItemCount:        0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Fork branch copies items up to fork point
	if forkFromItemID != "" {
		if err := s.convRepo.ForkBranch(ctx, conv.ID, conv.ActiveBranch, newBranchName, forkFromItemID, &description); err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to fork branch")
		}
	} else {
		// If no fork point (editing first message), just create empty branch
		if err := s.convRepo.CreateBranch(ctx, conv.ID, newBranchName, metadata); err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to create branch")
		}
	}

	// Create new user item with edited content
	newPublicID, err := idgen.GenerateSecureID("msg", 16)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to generate item ID")
	}

	// Get item count in new branch for sequence number
	itemCount, err := s.convRepo.CountItems(ctx, conv.ID, newBranchName)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to count items")
	}

	newItem := &Item{
		PublicID:       newPublicID,
		Object:         "conversation.item",
		Type:           ItemTypeMessage,
		Role:           originalItem.Role,
		Content:        []Content{{Type: "input_text", TextString: &newContent}},
		ConversationID: conv.ID,
		Branch:         newBranchName,
		SequenceNumber: itemCount + 1,
		CreatedAt:      now,
	}

	// Add the new item to the branch
	if err := s.convRepo.AddItemToBranch(ctx, conv.ID, newBranchName, newItem); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to add edited item")
	}

	// Swap the new branch to become MAIN (old MAIN becomes a backup)
	oldMainBackup, err := s.convRepo.SwapBranchToMain(ctx, conv.ID, newBranchName)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to swap branch to MAIN")
	}

	// Update the item's branch to MAIN after swap
	newItem.Branch = "MAIN"

	return &EditResult{
		NewBranch:      "MAIN",
		OldMainBackup:  oldMainBackup,
		UserItem:       newItem,
		ConversationID: conv.PublicID,
	}, nil
}

// RegenerateResponse creates a new branch and prepares for regenerating the assistant response
// Accepts either a user message ID (uses it directly as fork point) or an assistant message ID
// (finds the preceding user message). Returns the user message that should be used to regenerate a response.
func (s *MessageActionService) RegenerateResponse(ctx context.Context, conv *Conversation, itemPublicID string) (*RegenerateResult, error) {
	// Get the item
	item, err := s.convRepo.GetItemByPublicID(ctx, conv.ID, itemPublicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "item not found")
	}

	// Use the item's branch to find sibling items (not active branch, as item may be in a different branch)
	itemBranch := item.Branch
	if itemBranch == "" {
		itemBranch = "MAIN"
	}

	var userItem *Item

	// Handle based on role
	if item.Role != nil && *item.Role == ItemRoleUser {
		// User message passed directly - use it as fork point
		userItem = item
	} else if item.Role != nil && *item.Role == ItemRoleAssistant {
		// Assistant message - find preceding user message
		branchItems, err := s.convRepo.GetBranchItems(ctx, conv.ID, itemBranch, nil)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to get branch items")
		}

		// Find the corresponding user message (the item right before the assistant message)
		for i, branchItem := range branchItems {
			if branchItem.PublicID == itemPublicID {
				// Look for the user message before this assistant message
				for j := i - 1; j >= 0; j-- {
					if branchItems[j].Role != nil && *branchItems[j].Role == ItemRoleUser {
						userItem = branchItems[j]
						break
					}
				}
				break
			}
		}
	} else {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "can only regenerate from user or assistant messages", nil, "b2c3d4e5-f6a7-4b8c-9d0e-1f2a3b4c5d6e")
	}

	if userItem == nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeNotFound, "user message not found for regeneration", nil, "c3d4e5f6-a7b8-4c9d-0e1f-2a3b4c5d6e7f")
	}

	// Generate new branch name
	newBranchName := GenerateRegenBranchName(conv.ID)

	// Fork at the user message (so we keep history up to and including user message)
	// Use the item's branch as source, not active branch
	description := "Regenerated response branch"
	if err := s.convRepo.ForkBranch(ctx, conv.ID, itemBranch, newBranchName, userItem.PublicID, &description); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to fork branch for regeneration")
	}

	// Swap the new branch to become MAIN (old MAIN becomes a backup)
	oldMainBackup, err := s.convRepo.SwapBranchToMain(ctx, conv.ID, newBranchName)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to swap branch to MAIN")
	}

	// Get the new user item ID from MAIN (it was copied during fork)
	mainItems, err := s.convRepo.GetBranchItems(ctx, conv.ID, "MAIN", nil)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to get MAIN items")
	}

	// Find the last user message in MAIN (should be the one we forked to)
	var newUserItemID string
	for i := len(mainItems) - 1; i >= 0; i-- {
		if mainItems[i].Role != nil && *mainItems[i].Role == ItemRoleUser {
			newUserItemID = mainItems[i].PublicID
			break
		}
	}

	return &RegenerateResult{
		NewBranch:     "MAIN",
		OldMainBackup: oldMainBackup,
		ConvID:        conv.PublicID,
		UserItemID:    newUserItemID,
	}, nil
}

// DeleteResult contains the result of a delete message operation
type DeleteResult struct {
	NewBranch     string `json:"new_branch"`      // Always "MAIN" after swap
	OldMainBackup string `json:"old_main_backup"` // Backup name for old MAIN
}

// DeleteMessage deletes a message by creating a new branch without it
// The new branch becomes MAIN and the old MAIN is preserved as a backup
func (s *MessageActionService) DeleteMessage(ctx context.Context, conv *Conversation, itemPublicID string) (*DeleteResult, error) {
	// Get the item to verify it exists
	item, err := s.convRepo.GetItemByPublicID(ctx, conv.ID, itemPublicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "item not found")
	}

	// Get the item's branch
	itemBranch := item.Branch
	if itemBranch == "" {
		itemBranch = "MAIN"
	}

	// Get all branch items to find the item before the one to delete
	branchItems, err := s.convRepo.GetBranchItems(ctx, conv.ID, itemBranch, nil)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to get branch items")
	}

	// Find the item to delete and determine fork point (one item before)
	forkFromItemID := ""
	for i, branchItem := range branchItems {
		if branchItem.PublicID == itemPublicID {
			if i > 0 {
				forkFromItemID = branchItems[i-1].PublicID
			}
			break
		}
	}

	// Generate new branch name
	newBranchName := generateBranchNameWithPrefix(conv.ID, "DELETE")

	// Fork the branch at the item before the deleted one
	description := "Deleted message branch"
	if forkFromItemID != "" {
		if err := s.convRepo.ForkBranch(ctx, conv.ID, itemBranch, newBranchName, forkFromItemID, &description); err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to fork branch for delete")
		}
	} else {
		// If deleting the first message, create an empty branch
		now := time.Now()
		metadata := &BranchMetadata{
			Name:        newBranchName,
			Description: &description,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := s.convRepo.CreateBranch(ctx, conv.ID, newBranchName, metadata); err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to create branch for delete")
		}
	}

	// Swap the new branch to become MAIN (old MAIN becomes a backup)
	oldMainBackup, err := s.convRepo.SwapBranchToMain(ctx, conv.ID, newBranchName)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to swap branch to MAIN")
	}

	return &DeleteResult{
		NewBranch:     "MAIN",
		OldMainBackup: oldMainBackup,
	}, nil
}

// GenerateRegenBranchName generates a unique branch name for regenerated responses
func GenerateRegenBranchName(conversationID uint) string {
	return generateBranchNameWithPrefix(conversationID, "REGEN")
}

// generateBranchNameWithPrefix generates a unique branch name with a prefix
func generateBranchNameWithPrefix(conversationID uint, prefix string) string {
	return prefix + "_" + time.Now().Format("20060102150405")
}
