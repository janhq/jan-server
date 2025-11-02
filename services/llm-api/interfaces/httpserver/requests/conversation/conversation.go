package conversationrequests

import "menlo.ai/menlo-platform/internal/domain/conversation"

// CreateConversationRequest represents the request to create a conversation
type CreateConversationRequest struct {
	Title    *string             `json:"title,omitempty"`
	Items    []conversation.Item `json:"items,omitempty"`
	Metadata map[string]string   `json:"metadata,omitempty"`
	Referrer *string             `json:"referrer,omitempty"`
}

// UpdateConversationRequest represents the request to update a conversation
type UpdateConversationRequest struct {
	Title    *string           `json:"title,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Referrer *string           `json:"referrer,omitempty"`
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
}

// GetItemQueryParams represents query parameters for getting a single item
type GetItemQueryParams struct {
	Include []string `form:"include"`
}
