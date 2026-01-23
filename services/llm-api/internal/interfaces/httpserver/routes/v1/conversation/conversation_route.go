package conversation

import (
	"net/http"
	"strings"

	"github.com/janhq/jan-server/packages/go-common/analytics"
	analyticsMiddleware "github.com/janhq/jan-server/packages/go-common/analytics/middleware"

	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/conversationhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/requests"
	conversationrequests "jan-server/services/llm-api/internal/interfaces/httpserver/requests/conversation"
	"jan-server/services/llm-api/internal/interfaces/httpserver/responses"
	conversationresponses "jan-server/services/llm-api/internal/interfaces/httpserver/responses/conversation"
	"jan-server/services/llm-api/internal/utils/platformerrors"

	"github.com/gin-gonic/gin"
)

type ConversationRoute struct {
	handler     *conversationhandler.ConversationHandler
	authHandler *authhandler.AuthHandler
}

func NewConversationRoute(
	handler *conversationhandler.ConversationHandler,
	authHandler *authhandler.AuthHandler,
) *ConversationRoute {
	return &ConversationRoute{
		handler:     handler,
		authHandler: authHandler,
	}
}

func (route *ConversationRoute) RegisterRouter(router gin.IRouter) {
	conversations := router.Group("/conversations")
	conversations.GET("", route.authHandler.WithAppUserAuthChain(route.listConversations)...)
	conversations.POST("", route.authHandler.WithAppUserAuthChain(route.createConversation)...)
	conversations.DELETE("", route.authHandler.WithAppUserAuthChain(route.deleteAllConversations)...)
	conversations.GET("/:conv_public_id", route.authHandler.WithAppUserAuthChain(route.handler.ConversationMiddleware(), route.getConversation)...)
	conversations.POST("/:conv_public_id", route.authHandler.WithAppUserAuthChain(route.handler.ConversationMiddleware(), route.updateConversation)...)
	conversations.DELETE("/:conv_public_id", route.authHandler.WithAppUserAuthChain(route.handler.ConversationMiddleware(), route.deleteConversation)...)
	conversations.GET("/:conv_public_id/items", route.authHandler.WithAppUserAuthChain(route.handler.ConversationMiddleware(), route.listItems)...)
	conversations.POST("/:conv_public_id/items", route.authHandler.WithAppUserAuthChain(route.handler.ConversationMiddleware(), route.createItems)...)
	conversations.GET("/:conv_public_id/items/:item_id", route.authHandler.WithAppUserAuthChain(route.handler.ConversationMiddleware(), route.getItem)...)
	conversations.DELETE("/:conv_public_id/items/:item_id", route.authHandler.WithAppUserAuthChain(route.handler.ConversationMiddleware(), route.deleteItem)...)
	// MCP tool tracking: update item by call_id
	conversations.PATCH("/:conv_public_id/items/by-call-id/:call_id", route.authHandler.WithAppUserAuthChain(route.handler.ConversationMiddleware(), route.updateItemByCallID)...)
}

// listConversations godoc
// @Summary List conversations
// @Description List conversations for the authenticated user with optional referrer filtering.
// @Tags Conversations API
// @Security BearerAuth
// @Produce json
// @Param referrer query string false "Referrer filter"
// @Param limit query int false "Maximum number of conversations to return"
// @Param after query string false "Return conversations created after the given numeric ID"
// @Param order query string false "Sort order (asc or desc)"
// @Param scope query string false "Set to 'all' to list conversations across the workspace (requires elevated permissions)"
// @Success 200 {object} conversationresponses.ConversationListResponse "Successfully retrieved conversations"
// @Failure 400 {object} responses.ErrorResponse "Invalid request parameters"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations [get]
func (route *ConversationRoute) listConversations(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "3296ce86-783b-4c05-9fdb-930d3713024e")
		return
	}

	var params conversationrequests.ListConversationsQueryParams
	if err := reqCtx.ShouldBindQuery(&params); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "invalid query parameters", "f8a3d4e2-6b9c-4d7e-a1f3-2c5e8d9f0b4a")
		return
	}

	// Use the standard cursor pagination helper that properly resolves public IDs to numeric IDs
	// This follows the same pattern as API keys route (see apikeys_route.go:84)
	pagination, err := requests.GetCursorPaginationFromQuery(reqCtx, func(publicID string) (*uint, error) {
		// Resolve conversation public ID to numeric ID for cursor pagination
		// We need to call the handler's method which internally uses the service
		id, err := route.handler.ResolveConversationPublicIDToNumericID(ctx, user.ID, publicID)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "invalid cursor: conversation not found or not accessible")
		}
		return id, nil
	})
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to process pagination")
		return
	}

	// Override limit if provided in params (GetCursorPaginationFromQuery uses query params)
	if params.Limit != nil {
		if *params.Limit <= 0 {
			responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "limit must be greater than zero", "a7b2c3d4-e5f6-4g7h-8i9j-0k1l2m3n4o5p")
			return
		}
		pagination.Limit = params.Limit
	}

	if params.Order != nil {
		pagination.Order = strings.ToLower(strings.TrimSpace(*params.Order))
	}

	var referrerPtr *string
	if params.Referrer != nil {
		trimmed := strings.TrimSpace(*params.Referrer)
		if trimmed != "" {
			referrerValue := trimmed
			referrerPtr = &referrerValue
		}
	}

	var response *conversationresponses.ConversationListResponse
	response, err = route.handler.ListConversations(ctx, &user.ID, referrerPtr, pagination)

	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to list conversations")
		return
	}

	reqCtx.JSON(http.StatusOK, response)
}

// createConversation godoc
// @Summary Create a conversation
// @Description Create a new conversation to store and retrieve conversation state across Response API calls
// @Description
// @Description **Features:**
// @Description - Create conversation with optional metadata (max 16 key-value pairs)
// @Description - Add up to 20 initial items to the conversation
// @Description - Returns conversation ID with `conv_` prefix
// @Description - Supports OpenAI Conversations API format
// @Description
// @Description **Metadata Constraints:**
// @Description - Maximum 16 key-value pairs
// @Description - Keys: max 64 characters
// @Description - Values: max 512 characters
// @Tags Conversations API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body conversationrequests.CreateConversationRequest true "Create conversation request with optional items and metadata"
// @Success 200 {object} conversationresponses.ConversationResponse "Successfully created conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid request - validation failed or too many items"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 500 {object} responses.ErrorResponse "Internal server error - conversation creation failed"
// @Router /v1/conversations [post]
func (route *ConversationRoute) createConversation(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "3296ce86-783b-4c05-9fdb-930d3713024e")
		return
	}

	var req conversationrequests.CreateConversationRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "invalid request body", "b9c8d7e6-f5a4-4d3e-a1b2-0c9d8e7f6g5h")
		return
	}
	response, err := route.handler.CreateConversation(ctx, user.ID, req)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to create conversation")
		return
	}

	// Track conversation_created event
	values := analytics.ValuesFromContext(ctx)
	if values.DistinctID != "" && response != nil {
		props := map[string]interface{}{
			analytics.PropConversationID: response.ID,
			analytics.PropHasProject:     req.ProjectID != nil && *req.ProjectID != "",
			analytics.PropUserStatus:     values.UserStatus,
			analytics.PropPlatform:       values.Platform,
		}
		_ = analyticsMiddleware.TrackEvent(reqCtx, analytics.EventConversationCreated, props)
	}

	reqCtx.JSON(http.StatusOK, response)
}

// deleteAllConversations godoc
// @Summary Delete all conversations
// @Description Permanently delete all conversations for the authenticated user
// @Description
// @Description **WARNING: This is a destructive operation that cannot be undone.**
// @Description
// @Description **Features:**
// @Description - Deletes ALL conversations owned by the authenticated user
// @Description - Automatically cascades to delete all associated items and shares
// @Description - Returns the count of deleted conversations
// @Description - Requires valid authentication to ensure ownership verification
// @Description
// @Description **Security:**
// @Description - Only deletes conversations owned by the authenticated user
// @Description - Cannot delete other users' conversations
// @Description - Authentication is mandatory
// @Tags Conversations API
// @Security BearerAuth
// @Produce json
// @Success 200 {object} conversationresponses.BulkConversationsDeletedResponse "Successfully deleted all conversations"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 500 {object} responses.ErrorResponse "Internal server error - deletion failed"
// @Router /v1/conversations [delete]
func (route *ConversationRoute) deleteAllConversations(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "delete-all-convs-unauthorized")
		return
	}

	response, err := route.handler.DeleteAllConversations(ctx, user.ID)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to delete all conversations")
		return
	}
	reqCtx.JSON(http.StatusOK, response)
}

// getConversation godoc
// @Summary Get a conversation
// @Description Retrieve a conversation by ID with ownership verification
// @Description
// @Description **Features:**
// @Description - Retrieves conversation metadata including creation timestamp
// @Description - Automatic ownership verification (user can only access their own conversations)
// @Description - Returns OpenAI-compatible conversation object
// @Description
// @Description **Response Fields:**
// @Description - `id`: Conversation ID with `conv_` prefix
// @Description - `object`: Always "conversation"
// @Description - `created_at`: Unix timestamp
// @Description - `metadata`: User-defined key-value pairs
// @Tags Conversations API
// @Security BearerAuth
// @Produce json
// @Param conv_public_id path string true "Conversation ID (format: conv_xxxxx)"
// @Success 200 {object} conversationresponses.ConversationResponse "Successfully retrieved conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid conversation ID format"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found or access denied"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conv_public_id} [get]
func (route *ConversationRoute) getConversation(reqCtx *gin.Context) {
	// Get conversation from context (set by middleware)
	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "conversation not found in context", "c1d2e3f4-a5b6-4c7d-8e9f-0a1b2c3d4e5f")
		return
	}

	response := conversationresponses.NewConversationResponse(conv)
	reqCtx.JSON(http.StatusOK, response)
}

// updateConversation godoc
// @Summary Update a conversation
// @Description Update a conversation's metadata while preserving existing items
// @Description
// @Description **Features:**
// @Description - Update metadata key-value pairs
// @Description - Replaces entire metadata object (not merged)
// @Description - Items remain unchanged
// @Description - Automatic ownership verification
// @Description
// @Description **Metadata Constraints:**
// @Description - Maximum 16 key-value pairs
// @Description - Keys: max 64 characters
// @Description - Values: max 512 characters
// @Tags Conversations API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conv_public_id path string true "Conversation ID (format: conv_xxxxx)"
// @Param request body conversationrequests.UpdateConversationRequest true "Update conversation request with new metadata"
// @Success 200 {object} conversationresponses.ConversationResponse "Successfully updated conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid request - validation failed or invalid metadata"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found or access denied"
// @Failure 500 {object} responses.ErrorResponse "Internal server error - update failed"
// @Router /v1/conversations/{conv_public_id} [post]
func (route *ConversationRoute) updateConversation(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	// Get conversation and user from context (set by middlewares)
	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "conversation not found in context", "d2e3f4a5-b6c7-4d8e-9f0a-1b2c3d4e5f6g")
		return
	}

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "e3f4a5b6-c7d8-4e9f-0a1b-2c3d4e5f6g7h")
		return
	}

	var req conversationrequests.UpdateConversationRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "invalid request body", "f4a5b6c7-d8e9-4f0a-1b2c-3d4e5f6g7h8i")
		return
	}

	response, err := route.handler.UpdateConversation(ctx, user.ID, conv.PublicID, req)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to update conversation")
		return
	}
	reqCtx.JSON(http.StatusOK, response)
}

// deleteConversation godoc
// @Summary Delete a conversation
// @Description Delete a conversation (soft delete). Items in the conversation will not be deleted but will be inaccessible.
// @Description
// @Description **Features:**
// @Description - Soft delete (conversation marked as deleted, not physically removed)
// @Description - Items remain in database but become inaccessible
// @Description - Automatic ownership verification
// @Description - Returns deletion confirmation with conversation ID
// @Description
// @Description **Response:**
// @Description - `id`: Deleted conversation ID
// @Description - `object`: Always "conversation.deleted"
// @Description - `deleted`: Always true
// @Tags Conversations API
// @Security BearerAuth
// @Produce json
// @Param conv_public_id path string true "Conversation ID (format: conv_xxxxx)"
// @Success 200 {object} conversationresponses.ConversationDeletedResponse "Successfully deleted conversation"
// @Failure 400 {object} responses.ErrorResponse "Invalid conversation ID format"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found or access denied"
// @Failure 500 {object} responses.ErrorResponse "Internal server error - deletion failed"
// @Router /v1/conversations/{conv_public_id} [delete]
func (route *ConversationRoute) deleteConversation(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	// Get conversation and user from context (set by middlewares)
	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "conversation not found in context", "a5b6c7d8-e9f0-4a1b-2c3d-4e5f6g7h8i9j")
		return
	}

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "b6c7d8e9-f0a1-4b2c-3d4e-5f6g7h8i9j0k")
		return
	}

	response, err := route.handler.DeleteConversation(ctx, user.ID, conv.PublicID)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to delete conversation")
		return
	}
	reqCtx.JSON(http.StatusOK, response)
}

// listItems godoc
// @Summary List conversation items
// @Description List all items in a conversation with cursor-based pagination support
// @Description
// @Description **Features:**
// @Description - Cursor-based pagination using item IDs
// @Description - Configurable page size (1-100 items, default 20)
// @Description - Sort order control (ascending or descending)
// @Description - Optional include parameter for additional fields
// @Description - Returns paginated list with navigation cursors
// @Description
// @Description **Pagination:**
// @Description - Use `after` cursor from previous response for next page
// @Description - `has_more` indicates if more items are available
// @Description - `first_id` and `last_id` provide cursor references
// @Description
// @Description **Query Parameters:**
// @Description - `limit`: Number of items (1-100, default 20)
// @Description - `order`: Sort order ("asc" or "desc", default "desc")
// @Description - `after`: Item ID cursor for pagination
// @Description - `include`: Additional fields to include (optional)
// @Tags Conversations API
// @Security BearerAuth
// @Produce json
// @Param conv_public_id path string true "Conversation ID (format: conv_xxxxx)"
// @Param after query string false "Item ID cursor to list items after (pagination)"
// @Param limit query integer false "Number of items to return (1-100)" default(20) minimum(1) maximum(100)
// @Param order query string false "Sort order: asc or desc" default(desc) Enums(asc, desc)
// @Param include query []string false "Additional fields to include in response"
// @Success 200 {object} conversationresponses.ItemListResponse "Successfully retrieved items list"
// @Failure 400 {object} responses.ErrorResponse "Invalid request - invalid parameters or conversation ID"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found or access denied"
// @Failure 500 {object} responses.ErrorResponse "Internal server error - listing failed"
// @Router /v1/conversations/{conv_public_id}/items [get]
func (route *ConversationRoute) listItems(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	// Get conversation from context (set by middleware)
	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "conversation not found in context", "c7d8e9f0-a1b2-4c3d-4e5f-6g7h8i9j0k1l")
		return
	}

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "d8e9f0a1-b2c3-4d4e-5f6g-7h8i9j0k1l2m")
		return
	}

	var params conversationrequests.ListItemsQueryParams
	if err := reqCtx.ShouldBindQuery(&params); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "invalid query parameters", "e9f0a1b2-c3d4-4e5f-6g7h-8i9j0k1l2m3n")
		return
	}

	// Build pagination using standard cursor helper for query parameter parsing
	pagination, err := requests.GetCursorPaginationFromQuery(reqCtx, func(itemPublicID string) (*uint, error) {
		id, err := route.handler.ResolveItemPublicIDToNumericID(ctx, user.ID, conv.PublicID, itemPublicID)
		if err != nil {
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "invalid cursor: item not found or not accessible")
		}
		return id, nil
	})
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to process pagination")
		return
	}

	// Apply default limit if not specified (default 20, max 100)
	requestedLimit := 20 // default
	if pagination.Limit == nil {
		pagination.Limit = &requestedLimit
	} else if *pagination.Limit < 1 {
		requestedLimit = 1
		pagination.Limit = &requestedLimit
	} else if *pagination.Limit > 100 {
		requestedLimit = 100
		pagination.Limit = &requestedLimit
	} else {
		requestedLimit = *pagination.Limit
	}

	// Note: We use manual pagination building instead of responses.BuildCursorPage because:
	// 1. OpenAI Conversations API format doesn't include total counts
	// 2. The limit+1 fetch pattern is more efficient than separate hasMore queries
	// 3. ItemListResponse structure differs from the generic PageCursor structure

	// Fetch limit+1 items to determine if there are more pages
	fetchLimit := requestedLimit + 1
	pagination.Limit = &fetchLimit

	// Get items from handler with optional branch filter
	items, err := route.handler.ListItems(ctx, user.ID, conv.PublicID, params.Branch, pagination)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to list items")
		return
	}

	// Calculate hasMore by checking if we got more than requested
	hasMore := len(items) > requestedLimit
	if hasMore {
		// Trim to requested limit
		items = items[:requestedLimit]
	}

	// Calculate cursor IDs
	var firstID, lastID string
	if len(items) > 0 {
		firstID = items[0].PublicID
		lastID = items[len(items)-1].PublicID
	}

	// Build response matching OpenAI format
	response := conversationresponses.ItemListResponse{
		Object:  "list",
		Data:    items,
		FirstID: firstID,
		LastID:  lastID,
		HasMore: hasMore,
	}

	reqCtx.JSON(http.StatusOK, response)
}

// createItems godoc
// @Summary Create conversation items
// @Description Add items to a conversation. You may add up to 20 items at a time.
// @Description
// @Description **Features:**
// @Description - Bulk item creation (max 20 items per request)
// @Description - Automatic item ID generation with `msg_` prefix
// @Description - Items added to conversation's active branch (default: MAIN)
// @Description - Returns list of created items with generated IDs
// @Description
// @Description **Item Types:**
// @Description - `message`: User or assistant messages
// @Description - `tool_call`: Tool/function call items
// @Description - `tool_response`: Tool/function response items
// @Description - Other OpenAI-compatible item types
// @Description
// @Description **Constraints:**
// @Description - Maximum 20 items per request
// @Description - Each item must have valid type and content
// @Description - Items are immutable after creation
// @Tags Conversations API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conv_public_id path string true "Conversation ID (format: conv_xxxxx)"
// @Param include query []string false "Additional fields to include in response"
// @Param request body conversationrequests.CreateItemsRequest true "Create items request with array of items"
// @Success 200 {object} conversationresponses.ConversationItemCreatedResponse "Successfully created items"
// @Failure 400 {object} responses.ErrorResponse "Invalid request - too many items, invalid format, or validation failed"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found or access denied"
// @Failure 500 {object} responses.ErrorResponse "Internal server error - item creation failed"
// @Router /v1/conversations/{conv_public_id}/items [post]
func (route *ConversationRoute) createItems(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	// Get conversation from context (set by middleware)
	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "conversation not found in context", "f0a1b2c3-d4e5-4f6g-7h8i-9j0k1l2m3n4o")
		return
	}

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "a1b2c3d4-e5f6-4g7h-8i9j-0k1l2m3n4o5p")
		return
	}

	var req conversationrequests.CreateItemsRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "invalid request body", "b2c3d4e5-f6g7-4h8i-9j0k-1l2m3n4o5p6q")
		return
	}

	response, err := route.handler.CreateItems(ctx, user.ID, conv.PublicID, req)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to create items")
		return
	}
	reqCtx.JSON(http.StatusOK, response)
}

// getItem godoc
// @Summary Get a conversation item
// @Description Retrieve a single item from a conversation by item ID
// @Description
// @Description **Features:**
// @Description - Retrieve specific item by ID
// @Description - Returns complete item with all content
// @Description - Automatic ownership verification via conversation
// @Description - Optional include parameter for additional fields
// @Description
// @Description **Response Fields:**
// @Description - `id`: Item ID with `msg_` prefix
// @Description - `type`: Item type (message, tool_call, etc.)
// @Description - `role`: Role for message items (user, assistant)
// @Description - `content`: Item content array
// @Description - `status`: Item status (completed, incomplete, etc.)
// @Description - `created_at`: Unix timestamp
// @Tags Conversations API
// @Security BearerAuth
// @Produce json
// @Param conv_public_id path string true "Conversation ID (format: conv_xxxxx)"
// @Param item_id path string true "Item ID (format: msg_xxxxx)"
// @Param include query []string false "Additional fields to include in response"
// @Success 200 {object} conversationresponses.ItemResponse "Successfully retrieved item"
// @Failure 400 {object} responses.ErrorResponse "Invalid conversation ID or item ID format"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 404 {object} responses.ErrorResponse "Conversation or item not found, or access denied"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conv_public_id}/items/{item_id} [get]
func (route *ConversationRoute) getItem(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	// Get conversation from context (set by middleware)
	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "conversation not found in context", "c3d4e5f6-g7h8-4i9j-0k1l-2m3n4o5p6q7r")
		return
	}

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "d4e5f6g7-h8i9-4j0k-1l2m-3n4o5p6q7r8s")
		return
	}

	itemID := reqCtx.Param("item_id")
	response, err := route.handler.GetItem(ctx, user.ID, conv.PublicID, itemID)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to get item")
		return
	}
	reqCtx.JSON(http.StatusOK, response)
}

// deleteItem godoc
// @Summary Delete a conversation item
// @Description Delete an item from a conversation by creating a new MAIN branch without it.
// @Description The old MAIN branch is preserved as a backup.
// @Description
// @Description **Features:**
// @Description - Creates a new branch without the deleted item
// @Description - New branch becomes MAIN, old MAIN becomes backup
// @Description - Automatic ownership verification
// @Description - Preserves conversation history in backup branch
// @Description
// @Description **Important:**
// @Description - The old MAIN branch is renamed to MAIN_YYYYMMDDHHMMSS
// @Description - You can switch back to the backup branch if needed
// @Description - This is a non-destructive delete operation
// @Description
// @Description **Response:**
// @Description Returns branch information including the backup branch name
// @Tags Conversations API
// @Security BearerAuth
// @Produce json
// @Param conv_public_id path string true "Conversation ID (format: conv_xxxxx)"
// @Param item_id path string true "Item ID to delete (format: msg_xxxxx)"
// @Success 200 {object} conversationhandler.DeleteItemResponse "Successfully deleted item, returns branch info"
// @Failure 400 {object} responses.ErrorResponse "Invalid conversation ID or item ID format"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 404 {object} responses.ErrorResponse "Conversation or item not found, or access denied"
// @Failure 500 {object} responses.ErrorResponse "Internal server error - deletion failed"
// @Router /v1/conversations/{conv_public_id}/items/{item_id} [delete]
func (route *ConversationRoute) deleteItem(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	// Get conversation from context (set by middleware)
	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "conversation not found in context", "e5f6g7h8-i9j0-4k1l-2m3n-4o5p6q7r8s9t")
		return
	}

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "f6g7h8i9-j0k1-4l2m-3n4o-5p6q7r8s9t0u")
		return
	}

	itemID := reqCtx.Param("item_id")
	response, err := route.handler.DeleteItem(ctx, user.ID, conv.PublicID, itemID)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to delete item")
		return
	}
	reqCtx.JSON(http.StatusOK, response)
}

// updateItemByCallID godoc
// @Summary Update item by call ID
// @Description Update a conversation item's status and output using its call_id.
// @Description This endpoint is primarily used by MCP tools to report tool execution results.
// @Description
// @Description **Features:**
// @Description - Find item by call_id (e.g., call_xxx) instead of item_id
// @Description - Update status to completed, failed, or cancelled
// @Description - Store tool output or error message
// @Description - Automatic timestamp for completion
// @Description
// @Description **Use Cases:**
// @Description - MCP tool reports successful execution with output
// @Description - MCP tool reports failure with error message
// @Description - Tool call status tracking and observability
// @Tags Conversations API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conv_public_id path string true "Conversation ID (format: conv_xxxxx)"
// @Param call_id path string true "Call ID of the tool call item (format: call_xxxxx)"
// @Param request body conversationrequests.UpdateItemByCallIDRequest true "Update request with status and optional output/error"
// @Success 200 {object} conversationresponses.ItemResponse "Successfully updated item"
// @Failure 400 {object} responses.ErrorResponse "Invalid request - validation failed"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 404 {object} responses.ErrorResponse "Conversation or item not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/conversations/{conv_public_id}/items/by-call-id/{call_id} [patch]
func (route *ConversationRoute) updateItemByCallID(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	// Get conversation from context (set by middleware)
	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "conversation not found in context", "a1b2c3d4-e5f6-4789-abcd-ef0123456789")
		return
	}

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "b2c3d4e5-f6a7-4890-bcde-f01234567890")
		return
	}

	callID := reqCtx.Param("call_id")
	if callID == "" {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "call_id is required", "c3d4e5f6-a7b8-4901-cdef-012345678901")
		return
	}

	var req conversationrequests.UpdateItemByCallIDRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "invalid request body", "d4e5f6a7-b8c9-4012-def0-123456789012")
		return
	}

	response, err := route.handler.UpdateItemByCallID(ctx, user.ID, conv.PublicID, callID, req)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to update item by call_id")
		return
	}
	reqCtx.JSON(http.StatusOK, response)
}
