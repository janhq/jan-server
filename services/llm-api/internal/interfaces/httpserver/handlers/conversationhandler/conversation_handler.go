package conversationhandler

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"jan-server/services/llm-api/internal/domain/conversation"
	"jan-server/services/llm-api/internal/domain/project"
	"jan-server/services/llm-api/internal/domain/query"
	authhandler "jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	conversationrequests "jan-server/services/llm-api/internal/interfaces/httpserver/requests/conversation"
	"jan-server/services/llm-api/internal/interfaces/httpserver/responses"
	conversationresponses "jan-server/services/llm-api/internal/interfaces/httpserver/responses/conversation"
	"jan-server/services/llm-api/internal/utils/idgen"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

// Context keys for conversation data
type ConversationContextKey string

const (
	ConversationContextKeyPublicID ConversationContextKey = "conv_public_id"
	ConversationContextEntity      ConversationContextKey = "ConversationContextEntity"
)

// ConversationHandler handles conversation-related HTTP requests
type ConversationHandler struct {
	conversationService *conversation.ConversationService
	projectService      *project.ProjectService
	itemValidator       *conversation.ItemValidator
}

// NewConversationHandler creates a new conversation handler
func NewConversationHandler(
	conversationService *conversation.ConversationService,
	projectService *project.ProjectService,
) *ConversationHandler {
	return &ConversationHandler{
		conversationService: conversationService,
		projectService:      projectService,
		itemValidator:       conversation.NewItemValidator(conversation.DefaultItemValidationConfig()),
	}
}

// CreateConversation creates a new conversation
func (h *ConversationHandler) CreateConversation(
	ctx context.Context,
	userID uint,
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
	if req.ProjectID != nil && *req.ProjectID != "" {
		// Verify project exists and user has access
		proj, err := h.projectService.GetProjectByPublicIDAndUserID(ctx, *req.ProjectID, userID)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "invalid or inaccessible project_id")
		}
		projectID = &proj.ID
	}

	// Create conversation
	input := conversation.CreateConversationInput{
		UserID:    userID,
		Title:     req.Title, // Use title from request
		Metadata:  req.Metadata,
		Referrer:  req.Referrer,
		ProjectID: projectID,
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
	userID uint,
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
	userID uint,
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
	userID uint,
	conversationID string,
	req conversationrequests.UpdateConversationRequest,
) (*conversationresponses.ConversationResponse, error) {
	input := conversation.UpdateConversationInput{
		Title:    req.Title,
		Metadata: req.Metadata,
		Referrer: req.Referrer,
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
	userID *uint,
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
	userID uint,
	conversationID string,
) (*conversationresponses.ConversationDeletedResponse, error) {
	if err := h.conversationService.DeleteConversationByID(ctx, userID, conversationID); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to delete conversation")
	}

	return conversationresponses.NewConversationDeletedResponse(conversationID), nil
}

// ListItems lists items in a conversation
func (h *ConversationHandler) ListItems(
	ctx context.Context,
	userID uint,
	conversationID string,
	pagination *query.Pagination,
) ([]conversation.Item, error) {
	// Verify conversation ownership
	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get conversation")
	}

	// Get items from repository for the active branch
	items, err := h.conversationService.GetConversationItems(ctx, conv, conv.ActiveBranch, pagination)
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
	userID uint,
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
	userID uint,
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
	userID uint,
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

// DeleteItem deletes an item from a conversation
func (h *ConversationHandler) DeleteItem(
	ctx context.Context,
	userID uint,
	conversationID string,
	itemID string,
) (*conversationresponses.ConversationResponse, error) {
	// Verify conversation ownership
	conv, err := h.conversationService.GetConversationByPublicIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get conversation")
	}

	// Delete item
	if err := h.conversationService.DeleteConversationItem(ctx, conv, itemID); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to delete item")
	}

	// Return the conversation (per OpenAI spec)
	return conversationresponses.NewConversationResponse(conv), nil
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
