package conversationhandler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"jan-server/mono/apps/backend/internal/domain/conversation"
	"jan-server/mono/apps/backend/internal/domain/project"
	"jan-server/mono/apps/backend/internal/domain/query"
	"jan-server/mono/apps/backend/internal/domain/share"
	authhandler "jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/authhandler"
	conversationrequests "jan-server/mono/apps/backend/internal/interfaces/httpserver/requests/conversation"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/responses"
	conversationresponses "jan-server/mono/apps/backend/internal/interfaces/httpserver/responses/conversation"
	"jan-server/mono/apps/backend/internal/utils/idgen"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
	"jan-server/mono/apps/backend/internal/utils/stringutils"
)

// Context keys for conversation data
type ConversationContextKey string

const (
	ConversationContextKeyPublicID ConversationContextKey = "conv_public_id"
	ConversationContextEntity      ConversationContextKey = "ConversationContextEntity"
)

// ConversationHandler handles conversation-related HTTP requests
type ConversationHandler struct {
	conversationService  *conversation.ConversationService
	messageActionService *conversation.MessageActionService
	projectService       *project.ProjectService
	itemValidator        *conversation.ItemValidator
	shareRepo            share.ShareRepository
}

// NewConversationHandler creates a new conversation handler
func NewConversationHandler(
	conversationService *conversation.ConversationService,
	messageActionService *conversation.MessageActionService,
	projectService *project.ProjectService,
	shareRepo share.ShareRepository,
) *ConversationHandler {
	return &ConversationHandler{
		conversationService:  conversationService,
		messageActionService: messageActionService,
		projectService:       projectService,
		itemValidator:        conversation.NewItemValidator(conversation.DefaultItemValidationConfig()),
		shareRepo:            shareRepo,
	}
}

// CreateConversation creates a new conversation
func (h *ConversationHandler) CreateConversation(
	ctx context.Context,
	userID string,
	req conversationrequests.CreateConversationRequest,
) (*conversationresponses.ConversationResponse, error) {
	// Validate item count (max 20 for initial creation per OpenAI spec)
	if len(req.Items) > 20 {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation,
			"cannot add more than 20 items at a time", nil, "items")
	}

	// Validate items before creating conversation
	for i, item := range req.Items {
		if err := h.itemValidator.ValidateItem(item); err != nil {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation,
				fmt.Sprintf("item validation failed at index %d", i), err, fmt.Sprintf("items[%d]", i))
		}
	}

	// Resolve project_id if provided
	var projectID *uint
	var projectPublicID *string
	if req.ProjectID != nil && *req.ProjectID != "" {
		// Verify project exists and user has access
		proj, err := h.projectService.GetProjectByPublicIDAndUserID(ctx, *req.ProjectID, userID)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "invalid or inaccessible project_id")
		}
		projectID = &proj.ID
		projectPublicID = &proj.PublicID
	}

	// Sanitize title if provided
	var sanitizedTitle *string
	if req.Title != nil && *req.Title != "" {
		rawTitle := strings.TrimSpace(*req.Title)
		if strings.EqualFold(rawTitle, "New Conversation") {
			rawTitle = ""
		}
		title := ""
		if rawTitle != "" {
			title = stringutils.GenerateTitle(rawTitle, 256)
		}
		if title != "" {
			sanitizedTitle = &title
		}
	}

	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata["title_locked"] = "false"

	// Create conversation
	input := conversation.CreateConversationInput{
		UserID:          userID,
		Title:           sanitizedTitle,
		Metadata:        metadata,
		Referrer:        req.Referrer,
		ProjectID:       projectID,
		ProjectPublicID: projectPublicID,
	}

	conv, err := h.conversationService.CreateConversationWithInput(ctx, input)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to create conversation")
	}

	// Add items if provided
	if len(req.Items) > 0 {
		if err := h.addItemsToConversation(ctx, conv, req.Items); err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to add items")
		}
	}

	return conversationresponses.NewConversationResponse(conv), nil
}

// GetConversation retrieves a conversation by ID
func (h *ConversationHandler) GetConversation(
	ctx context.Context,
	userID string,
	conversationID string,
) (*conversationresponses.ConversationResponse, error) {
	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get conversation")
	}

	return conversationresponses.NewConversationResponse(conv), nil
}

// ResolveConversationPublicIDToNumericID resolves a conversation public ID to its numeric ID
// This is used for cursor-based pagination where the API exposes public IDs but the
// underlying pagination system uses numeric IDs
func (h *ConversationHandler) ResolveConversationPublicIDToNumericID(
	ctx context.Context,
	userID string,
	publicID string,
) (*uint, error) {
	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, publicID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to resolve conversation ID")
	}
	return &conv.ID, nil
}

// UpdateConversation updates a conversation
func (h *ConversationHandler) UpdateConversation(
	ctx context.Context,
	userID string,
	conversationID string,
	req conversationrequests.UpdateConversationRequest,
) (*conversationresponses.ConversationResponse, error) {
	// Sanitize title if provided
	var sanitizedTitle *string
	if req.Title != nil && *req.Title != "" {
		rawTitle := strings.TrimSpace(*req.Title)
		if strings.EqualFold(rawTitle, "New Conversation") {
			rawTitle = ""
		}
		title := ""
		if rawTitle != "" {
			title = stringutils.GenerateTitle(rawTitle, 256)
		}
		if title != "" {
			sanitizedTitle = &title
		}
	}

	metadata := req.Metadata
	if sanitizedTitle != nil {
		conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, userID)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get conversation")
		}
		branchName := conv.ActiveBranch
		if branchName == "" {
			branchName = conversation.BranchMain
		}
		itemCount, err := h.conversationService.CountConversationItems(ctx, conv, branchName)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to count conversation items")
		}
		if metadata == nil {
			metadata = map[string]string{}
		}
		if itemCount > 1 {
			metadata["title_locked"] = "true"
		}
	}

	input := conversation.UpdateConversationInput{
		Title:    sanitizedTitle,
		Metadata: metadata,
		Referrer: req.Referrer,
	}

	// Resolve and update project when provided
	if req.ProjectID != nil {
		projectID := strings.TrimSpace(*req.ProjectID)
		if projectID == "" {
			zeroID := uint(0)
			emptyStr := ""
			input.ProjectID = &zeroID
			input.ProjectPublicID = &emptyStr
		} else {
			// Verify project exists and user has access
			if h.projectService == nil {
				return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeInternal, "project service unavailable", nil, "a3b4c5d6-e7f8-4a9b-0c1d-2e3f4a5b6c7d")
			}
			proj, err := h.projectService.GetProjectByPublicIDAndUserID(ctx, projectID, userID)
			if err != nil {
				return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "invalid or inaccessible project_id")
			}
			input.ProjectID = &proj.ID
			input.ProjectPublicID = &proj.PublicID
		}
	}

	conv, err := h.conversationService.UpdateConversationWithInput(ctx, userID, conversationID, input)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to update conversation")
	}

	return conversationresponses.NewConversationResponse(conv), nil
}

// ListConversations lists conversations with flexible filtering
func (h *ConversationHandler) ListConversations(
	ctx context.Context,
	userID *string,
	referrer *string,
	pagination *query.Pagination,
) (*conversationresponses.ConversationListResponse, error) {
	// Build filter
	filter := conversation.ConversationFilter{}

	if userID != nil {
		filter.UserID = userID
	}

	if referrer != nil && *referrer != "" {
		filter.Referrer = referrer
	}

	// To properly calculate hasMore, we fetch limit+1 items and trim if needed
	// This is the standard pagination pattern that works correctly
	var requestedLimit *int
	if pagination != nil && pagination.Limit != nil {
		requestedLimit = pagination.Limit
		extraLimit := *pagination.Limit + 1
		pagination.Limit = &extraLimit
	}

	// Use unified service method (fetching limit+1)
	conversations, total, err := h.conversationService.FindConversationsByFilter(ctx, filter, pagination)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to list conversations")
	}

	// Calculate hasMore by checking if we got more than requested
	hasMore := false
	if requestedLimit != nil && len(conversations) > *requestedLimit {
		// We got limit+1 items, so there are more pages
		hasMore = true
		// Trim to the requested limit
		conversations = conversations[:*requestedLimit]
	}

	return conversationresponses.NewConversationListResponse(conversations, hasMore, total), nil
}

// DeleteConversation deletes a conversation
func (h *ConversationHandler) DeleteConversation(
	ctx context.Context,
	userID string,
	conversationID string,
) (*conversationresponses.ConversationDeletedResponse, error) {
	// Get the conversation first to get its numeric ID for share revocation
	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get conversation")
	}

	// Revoke all shares for this conversation before deleting
	if h.shareRepo != nil {
		if err := h.shareRepo.RevokeAllByConversationID(ctx, conv.ID); err != nil {
			// Log but don't fail the delete - shares should still be revoked
			// The share lookup will fail anyway since the conversation is deleted
			_ = err // Ignore error, conversation delete takes priority
		}
	}

	if err := h.conversationService.DeleteConversationByID(ctx, userID, conversationID); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to delete conversation")
	}

	return conversationresponses.NewConversationDeletedResponse(conversationID), nil
}

// DeleteAllConversations deletes all conversations for a user
func (h *ConversationHandler) DeleteAllConversations(
	ctx context.Context,
	userID string,
) (*conversationresponses.BulkConversationsDeletedResponse, error) {
	// Validate user ID
	if userID == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation,
			"invalid user ID", nil, "delete-all-convs-invalid-user")
	}

	// Revoke all shares for all conversations owned by this user
	// Note: This is handled by CASCADE DELETE in the database schema,
	// but we explicitly revoke to ensure clean-up of any external references
	if h.shareRepo != nil {
		// The shares will be deleted automatically via CASCADE when conversations are deleted
		// No explicit revocation needed here
	}

	// Delete all conversations for this user
	deletedCount, err := h.conversationService.DeleteAllConversationsByUserID(ctx, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to delete all conversations")
	}

	return conversationresponses.NewBulkConversationsDeletedResponse(deletedCount), nil
}

// ListItems lists items in a conversation
func (h *ConversationHandler) ListItems(
	ctx context.Context,
	userID string,
	conversationID string,
	branchName *string,
	pagination *query.Pagination,
) ([]conversation.Item, error) {
	// Verify conversation ownership
	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get conversation")
	}

	// Use specified branch or fall back to active branch
	branch := conv.ActiveBranch
	if branchName != nil && *branchName != "" {
		branch = *branchName
	}

	// Get items from repository for the specified branch
	items, err := h.conversationService.GetConversationItems(ctx, conv, branch, pagination)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to list items")
	}

	return items, nil
}

// ResolveItemPublicIDToNumericID resolves an item public ID to its numeric ID
// This is used for cursor-based pagination where the API exposes public IDs but the
// underlying pagination system uses numeric IDs
func (h *ConversationHandler) ResolveItemPublicIDToNumericID(
	ctx context.Context,
	userID string,
	conversationID string,
	itemPublicID string,
) (*uint, error) {
	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get conversation")
	}

	item, err := h.conversationService.GetConversationItem(ctx, conv, itemPublicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to resolve item ID")
	}

	return &item.ID, nil
} // CreateItems creates items in a conversation
func (h *ConversationHandler) CreateItems(
	ctx context.Context,
	userID string,
	conversationID string,
	req conversationrequests.CreateItemsRequest,
) (*conversationresponses.ConversationItemCreatedResponse, error) {
	// Verify conversation ownership
	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get conversation")
	}

	// Validate item count (max 20)
	if len(req.Items) > 20 {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation,
			"cannot add more than 20 items at a time", nil, "items")
	}

	// Validate each item
	for i, item := range req.Items {
		if err := h.itemValidator.ValidateItem(item); err != nil {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation,
				fmt.Sprintf("item validation failed at index %d", i), err, fmt.Sprintf("items[%d]", i))
		}
	}

	// Add items to conversation
	addedItems, err := h.conversationService.AddItemsToConversation(ctx, conv, conv.ActiveBranch, req.Items)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to add items")
	}

	return conversationresponses.NewConversationItemCreatedResponse(addedItems), nil
}

// GetItem retrieves a single item from a conversation
func (h *ConversationHandler) GetItem(
	ctx context.Context,
	userID string,
	conversationID string,
	itemID string,
) (*conversationresponses.ItemResponse, error) {
	// Verify conversation ownership
	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get conversation")
	}

	// Get item
	item, err := h.conversationService.GetConversationItem(ctx, conv, itemID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get item")
	}

	return item, nil
}

// DeleteItemResponse represents the response for deleting a message
type DeleteItemResponse struct {
	Branch        string `json:"branch"`          // Always "MAIN" after swap
	OldMainBackup string `json:"old_main_backup"` // Backup name for old MAIN
	BranchCreated bool   `json:"branch_created"`
	Deleted       bool   `json:"deleted"`
}

// DeleteItem deletes an item from a conversation by creating a new MAIN branch without it
func (h *ConversationHandler) DeleteItem(
	ctx context.Context,
	userID string,
	conversationID string,
	itemID string,
) (*DeleteItemResponse, error) {
	// Verify conversation ownership
	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get conversation")
	}

	// Delete item using branch swap approach
	result, err := h.messageActionService.DeleteMessage(ctx, conv, itemID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to delete item")
	}

	return &DeleteItemResponse{
		Branch:        result.NewBranch,
		OldMainBackup: result.OldMainBackup,
		BranchCreated: true,
		Deleted:       true,
	}, nil
}

// UpdateItemByCallID updates an existing mcp_call item with tool execution results
// The mcp_call item was already created (with in_progress status) when the LLM returned tool_calls
// This is used by MCP tools to report tool execution results
func (h *ConversationHandler) UpdateItemByCallID(
	ctx context.Context,
	userID string,
	conversationID string,
	callID string,
	req conversationrequests.UpdateItemByCallIDRequest,
) (*conversationresponses.ItemResponse, error) {
	// Verify conversation ownership
	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get conversation")
	}

	// Get the mcp_call item by call_id (it was created when LLM returned tool_calls)
	mcpItem, err := h.conversationService.GetConversationItemByCallIDAndType(ctx, conv, callID, conversation.ItemTypeMcpCall)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "mcp_call item not found by call_id")
	}

	// Determine status
	status := conversation.ItemStatusCompleted
	if req.Status != nil {
		status = conversation.ItemStatus(*req.Status)
	}

	// Update the mcp_call item with the execution result
	mcpItem.Status = &status
	mcpItem.Output = req.Output
	mcpItem.Error = req.Error
	now := time.Now()
	mcpItem.CompletedAt = &now

	// Update additional fields if provided
	if req.Name != nil {
		mcpItem.Name = req.Name
	}
	if req.Arguments != nil {
		mcpItem.Arguments = req.Arguments
	}
	if req.ServerLabel != nil {
		mcpItem.ServerLabel = req.ServerLabel
	}

	// Update Content field with the output text so it's returned in the API response
	if req.Output != nil {
		mcpItem.Content = []conversation.Content{
			{
				Type:       "mcp_call",
				ToolCallID: &callID,
				TextString: req.Output,
			},
		}
	} else if req.Error != nil {
		// If there's an error, include it in the content
		mcpItem.Content = []conversation.Content{
			{
				Type:       "mcp_call",
				ToolCallID: &callID,
				TextString: req.Error,
			},
		}
	}

	if err := h.conversationService.UpdateConversationItem(ctx, conv, mcpItem); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to update mcp_call item")
	}

	return mcpItem, nil
}

// Helper functions

// addItemsToConversation adds items to a conversation
func (h *ConversationHandler) addItemsToConversation(ctx context.Context, conv *conversation.Conversation, items []conversation.Item) error {
	if len(items) == 0 {
		return nil
	}

	// Generate public IDs for items if not present
	for i := range items {
		if items[i].PublicID == "" {
			publicID, err := idgen.GenerateSecureID("msg", 16)
			if err != nil {
				return platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to generate item ID")
			}
			items[i].PublicID = publicID
		}
		items[i].Object = "conversation.item"
	}

	// Use service to add items to the default branch (MAIN)
	_, err := h.conversationService.AddItemsToConversation(ctx, conv, conversation.BranchMain, items)
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to add items to conversation")
	} else {
		return nil
	}
}

// ===============================================
// Middleware Functions
// ===============================================
func (h *ConversationHandler) ConversationMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()

		// Get conversation public ID from path parameter
		publicID := reqCtx.Param(string(ConversationContextKeyPublicID))
		if publicID == "" {
			responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "missing conversation public ID", "")
			return
		}

		// Get authenticated user from context
		user, ok := authhandler.GetUserFromContext(reqCtx)
		if !ok {
			responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "")
			return
		}

		// Retrieve conversation with ownership check
		conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, publicID, user.ID)
		if err != nil {
			responses.HandleError(reqCtx, err, "Failed to retrieve conversation")
			return
		} // Store conversation in context
		SetConversationToContext(reqCtx, conv)
		reqCtx.Next()
	}
}

// SetConversationToContext stores a conversation in the request context
func SetConversationToContext(reqCtx *gin.Context, conv *conversation.Conversation) {
	reqCtx.Set(string(ConversationContextEntity), conv)
}

// GetConversationFromContext retrieves a conversation from the request context
func GetConversationFromContext(reqCtx *gin.Context) (*conversation.Conversation, bool) {
	conv, ok := reqCtx.Get(string(ConversationContextEntity))
	if !ok {
		return nil, false
	}
	v, ok := conv.(*conversation.Conversation)
	if !ok {
		return nil, false
	}
	return v, true
}
