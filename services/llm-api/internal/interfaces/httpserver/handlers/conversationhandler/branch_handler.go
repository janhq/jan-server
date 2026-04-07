package conversationhandler

import (
	"context"

	"jan-server/services/llm-api/internal/domain/conversation"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

// BranchHandler handles branch-related HTTP requests
type BranchHandler struct {
	conversationService  *conversation.ConversationService
	messageActionService *conversation.MessageActionService
	repo                 conversation.ConversationRepository
}

// NewBranchHandler creates a new branch handler
func NewBranchHandler(
	conversationService *conversation.ConversationService,
	messageActionService *conversation.MessageActionService,
	repo conversation.ConversationRepository,
) *BranchHandler {
	return &BranchHandler{
		conversationService:  conversationService,
		messageActionService: messageActionService,
		repo:                 repo,
	}
}

// ===============================================
// Request/Response Types
// ===============================================

// CreateBranchRequest represents the request to create a branch
type CreateBranchRequest struct {
	Name           string  `json:"name" binding:"required"`
	ParentBranch   *string `json:"parent_branch,omitempty"`
	ForkFromItemID *string `json:"fork_from_item_id,omitempty"`
	Description    *string `json:"description,omitempty"`
}

// EditMessageRequest represents the request to edit a message
type EditMessageRequest struct {
	Content    string `json:"content" binding:"required"`
	Regenerate *bool  `json:"regenerate,omitempty"` // Auto-trigger new response (default: true)
}

// RegenerateMessageRequest represents the request to regenerate a message
type RegenerateMessageRequest struct {
	Model       *string  `json:"model,omitempty"`       // Override model
	Temperature *float32 `json:"temperature,omitempty"` // Override temperature
	MaxTokens   *int     `json:"max_tokens,omitempty"`  // Override max tokens
}

// BranchResponse represents a branch in API responses
type BranchResponse struct {
	Name             string  `json:"name"`
	Description      *string `json:"description,omitempty"`
	ParentBranch     *string `json:"parent_branch,omitempty"`
	ForkedAt         *int64  `json:"forked_at,omitempty"`
	ForkedFromItemID *string `json:"forked_from_item_id,omitempty"`
	ItemCount        int     `json:"item_count"`
	CreatedAt        int64   `json:"created_at"`
	UpdatedAt        int64   `json:"updated_at"`
	IsActive         bool    `json:"is_active"`
}

// ListBranchesResponse represents the response for listing branches
type ListBranchesResponse struct {
	Object       string           `json:"object"` // "list"
	Data         []BranchResponse `json:"data"`
	ActiveBranch string           `json:"active_branch"`
}

// EditMessageResponse represents the response for editing a message
type EditMessageResponse struct {
	Branch        string             `json:"branch"`          // Always "MAIN" after swap
	OldMainBackup string             `json:"old_main_backup"` // Backup name for old MAIN
	BranchCreated bool               `json:"branch_created"`
	NewBranch     *BranchResponse    `json:"new_branch,omitempty"`
	UserItem      *conversation.Item `json:"user_item"`
}

// RegenerateMessageResponse represents the response for regenerating a message
type RegenerateMessageResponse struct {
	Branch        string          `json:"branch"`          // Always "MAIN" after swap
	OldMainBackup string          `json:"old_main_backup"` // Backup name for old MAIN
	BranchCreated bool            `json:"branch_created"`
	NewBranch     *BranchResponse `json:"new_branch,omitempty"`
	UserItemID    string          `json:"user_item_id"`
}

// DeleteMessageResponse represents the response for deleting a message
type DeleteMessageResponse struct {
	Branch        string `json:"branch"`          // Always "MAIN" after swap
	OldMainBackup string `json:"old_main_backup"` // Backup name for old MAIN
	BranchCreated bool   `json:"branch_created"`
	Deleted       bool   `json:"deleted"`
}

// ActivateBranchResponse represents the response for activating a branch
type ActivateBranchResponse struct {
	ActiveBranch string `json:"active_branch"`
	Message      string `json:"message"`
}

// ===============================================
// Handler Methods
// ===============================================

// ListBranches lists all branches for a conversation
func (h *BranchHandler) ListBranches(ctx context.Context, conv *conversation.Conversation) (*ListBranchesResponse, error) {
	branches, err := h.repo.ListBranches(ctx, conv.ID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to list branches")
	}

	data := make([]BranchResponse, len(branches))
	for i, branch := range branches {
		data[i] = toBranchResponse(branch, conv.ActiveBranch)
	}

	// If no branches exist, return MAIN as default
	if len(data) == 0 {
		data = []BranchResponse{{
			Name:      "MAIN",
			ItemCount: 0,
			IsActive:  true,
		}}
	}

	return &ListBranchesResponse{
		Object:       "list",
		Data:         data,
		ActiveBranch: conv.ActiveBranch,
	}, nil
}

// CreateBranch creates a new branch
func (h *BranchHandler) CreateBranch(ctx context.Context, conv *conversation.Conversation, req CreateBranchRequest) (*BranchResponse, error) {
	// Validate branch name
	if req.Name == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "branch name is required", nil, "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d")
	}

	if req.Name == "MAIN" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "cannot create branch named MAIN", nil, "b2c3d4e5-f6a7-4b8c-9d0e-1f2a3b4c5d6e")
	}

	// Set default parent branch
	parentBranch := conv.ActiveBranch
	if req.ParentBranch != nil && *req.ParentBranch != "" {
		parentBranch = *req.ParentBranch
	}

	// Fork the branch if fork point is specified
	if req.ForkFromItemID != nil && *req.ForkFromItemID != "" {
		if err := h.repo.ForkBranch(ctx, conv.ID, parentBranch, req.Name, *req.ForkFromItemID, req.Description); err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to fork branch")
		}
	} else {
		// Create empty branch
		metadata := conv.CreateBranchMetadata(req.Name, &parentBranch, nil, req.Description)
		if err := h.repo.CreateBranch(ctx, conv.ID, req.Name, &metadata); err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to create branch")
		}
	}

	// Get the created branch
	branch, err := h.repo.GetBranch(ctx, conv.ID, req.Name)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get created branch")
	}

	response := toBranchResponse(branch, conv.ActiveBranch)
	return &response, nil
}

// GetBranch gets a branch by name
func (h *BranchHandler) GetBranch(ctx context.Context, conv *conversation.Conversation, branchName string) (*BranchResponse, error) {
	branch, err := h.repo.GetBranch(ctx, conv.ID, branchName)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "branch not found")
	}

	response := toBranchResponse(branch, conv.ActiveBranch)
	return &response, nil
}

// DeleteBranch deletes a branch
func (h *BranchHandler) DeleteBranch(ctx context.Context, conv *conversation.Conversation, branchName string) error {
	// Normalize "main" to "MAIN" for case-insensitive matching
	if branchName == "main" {
		branchName = "MAIN"
	}

	if branchName == "MAIN" {
		return platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "cannot delete MAIN branch", nil, "c3d4e5f6-a7b8-4c9d-0e1f-2a3b4c5d6e7f")
	}

	if branchName == conv.ActiveBranch {
		return platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "cannot delete active branch", nil, "d4e5f6a7-b8c9-4d0e-1f2a-3b4c5d6e7f8a")
	}

	if err := h.repo.DeleteBranch(ctx, conv.ID, branchName); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to delete branch")
	}

	return nil
}

// ActivateBranch sets a branch as active
func (h *BranchHandler) ActivateBranch(ctx context.Context, conv *conversation.Conversation, branchName string) (*ActivateBranchResponse, error) {
	// Normalize "main" to "MAIN" for case-insensitive matching
	if branchName == "main" {
		branchName = "MAIN"
	}

	// Verify branch exists (for non-MAIN branches)
	if branchName != "MAIN" {
		_, err := h.repo.GetBranch(ctx, conv.ID, branchName)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "branch not found")
		}
	}

	if err := h.repo.SetActiveBranch(ctx, conv.ID, branchName); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to activate branch")
	}

	return &ActivateBranchResponse{
		ActiveBranch: branchName,
		Message:      "Branch activated successfully",
	}, nil
}

// EditMessage edits a message and creates a new branch that becomes MAIN
func (h *BranchHandler) EditMessage(ctx context.Context, conv *conversation.Conversation, itemID string, req EditMessageRequest) (*EditMessageResponse, error) {
	result, err := h.messageActionService.EditMessage(ctx, conv, itemID, req.Content)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to edit message")
	}

	response := &EditMessageResponse{
		Branch:        result.NewBranch, // Always "MAIN"
		OldMainBackup: result.OldMainBackup,
		BranchCreated: true, // Edit always creates a new branch (which becomes MAIN)
		UserItem:      result.UserItem,
	}

	// Fetch the MAIN branch details (the new branch was swapped to MAIN)
	if branch, err := h.repo.GetBranch(ctx, conv.ID, "MAIN"); err == nil {
		branchResp := toBranchResponse(branch, "MAIN")
		response.NewBranch = &branchResp
	}

	return response, nil
}

// RegenerateMessage regenerates an assistant response, creating a new MAIN branch
func (h *BranchHandler) RegenerateMessage(ctx context.Context, conv *conversation.Conversation, itemID string, req RegenerateMessageRequest) (*RegenerateMessageResponse, error) {
	result, err := h.messageActionService.RegenerateResponse(ctx, conv, itemID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to regenerate message")
	}

	response := &RegenerateMessageResponse{
		Branch:        result.NewBranch, // Always "MAIN"
		OldMainBackup: result.OldMainBackup,
		BranchCreated: true, // Regenerate always creates a new branch (which becomes MAIN)
		UserItemID:    result.UserItemID,
	}

	// Fetch the MAIN branch details (the new branch was swapped to MAIN)
	if branch, err := h.repo.GetBranch(ctx, conv.ID, "MAIN"); err == nil {
		branchResp := toBranchResponse(branch, "MAIN")
		response.NewBranch = &branchResp
	}

	return response, nil
}

// DeleteMessage deletes a message by creating a new MAIN branch without it
func (h *BranchHandler) DeleteMessage(ctx context.Context, conv *conversation.Conversation, itemID string) (*DeleteMessageResponse, error) {
	result, err := h.messageActionService.DeleteMessage(ctx, conv, itemID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to delete message")
	}

	return &DeleteMessageResponse{
		Branch:        result.NewBranch, // Always "MAIN"
		OldMainBackup: result.OldMainBackup,
		BranchCreated: true,
		Deleted:       true,
	}, nil
}

// ===============================================
// Helper Functions
// ===============================================

func toBranchResponse(branch *conversation.BranchMetadata, activeBranch string) BranchResponse {
	response := BranchResponse{
		Name:        branch.Name,
		Description: branch.Description,
		ItemCount:   branch.ItemCount,
		CreatedAt:   branch.CreatedAt.Unix(),
		UpdatedAt:   branch.UpdatedAt.Unix(),
		IsActive:    branch.Name == activeBranch,
	}

	if branch.ParentBranch != nil {
		response.ParentBranch = branch.ParentBranch
	}
	if branch.ForkedAt != nil {
		ts := branch.ForkedAt.Unix()
		response.ForkedAt = &ts
	}
	if branch.ForkedFromItemID != nil {
		response.ForkedFromItemID = branch.ForkedFromItemID
	}

	return response
}
