package conversationrequests

import "jan-server/mono/apps/backend/internal/domain/conversation"

// CreateConversationRequest represents the request to create a conversation
type CreateConversationRequest struct {
	Title     *string             `json:"title,omitempty"`
	Items     []conversation.Item `json:"items,omitempty"`
	Metadata  map[string]string   `json:"metadata,omitempty"`
	Referrer  *string             `json:"referrer,omitempty"`
	ProjectID *string             `json:"project_id,omitempty"`
}

// UpdateConversationRequest represents the request to update a conversation
type UpdateConversationRequest struct {
	Title     *string           `json:"title,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Referrer  *string           `json:"referrer,omitempty"`
	ProjectID *string           `json:"project_id,omitempty"`
}

// CreateItemsRequest represents the request to create items in a conversation
type CreateItemsRequest struct {
	Items []conversation.Item `json:"items" binding:"required"`
}

// ListConversationsQueryParams represents query parameters for listing conversations
type ListConversationsQueryParams struct {
	Referrer *string `form:"referrer"`
	Limit    *int    `form:"limit"`
	Order    *string `form:"order"`
	After    *string `form:"after"`
	Scope    *string `form:"scope"`
}

// ListItemsQueryParams represents query parameters for listing items
type ListItemsQueryParams struct {
	After   *string  `form:"after"`
	Include []string `form:"include"`
	Limit   *int     `form:"limit"`
	Order   *string  `form:"order"`
	Branch  *string  `form:"branch"` // Filter by branch name (defaults to active branch)
}

// GetItemQueryParams represents query parameters for getting a single item
type GetItemQueryParams struct {
	Include []string `form:"include"`
}

// UpdateItemByCallIDRequest represents the request to update an mcp_call item by call_id
// Used by MCP tools to report tool execution results
type UpdateItemByCallIDRequest struct {
	// Required fields
	Status *string `json:"status" binding:"required"` // "completed", "failed", "cancelled"

	// Result fields
	Output *string `json:"output,omitempty"` // Result of the tool execution (JSON string)
	Error  *string `json:"error,omitempty"`  // Error message if status is "failed"

	// Tool info fields (optional - already set on creation, but can be updated)
	Name        *string `json:"name,omitempty"`         // Tool name
	Arguments   *string `json:"arguments,omitempty"`    // JSON string of arguments
	ServerLabel *string `json:"server_label,omitempty"` // MCP server label
}
