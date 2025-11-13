package conversationresponses

import (
	"jan-server/services/llm-api/internal/domain/conversation"
)

// ConversationResponse represents the OpenAI-compatible conversation response
type ConversationResponse struct {
	ID        string            `json:"id"`
	Object    string            `json:"object"`
	Title     *string           `json:"title,omitempty"`
	CreatedAt int64             `json:"created_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Referrer  *string           `json:"referrer,omitempty"`
	ProjectID *string           `json:"project_id,omitempty"`
}

// ConversationListResponse represents a paginated list of conversations
type ConversationListResponse struct {
	Object  string                 `json:"object"`
	Data    []ConversationResponse `json:"data"`
	FirstID string                 `json:"first_id"`
	LastID  string                 `json:"last_id"`
	HasMore bool                   `json:"has_more"`
	Total   int64                  `json:"total"`
}

// ConversationDeletedResponse represents the delete confirmation response
type ConversationDeletedResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

// ItemListResponse represents the OpenAI-compatible item list response
type ItemListResponse struct {
	Object  string              `json:"object"`
	Data    []conversation.Item `json:"data"`
	FirstID string              `json:"first_id"`
	LastID  string              `json:"last_id"`
	HasMore bool                `json:"has_more"`
}

// NewConversationResponse creates a response from a domain conversation
func NewConversationResponse(conv *conversation.Conversation) *ConversationResponse {
	response := &ConversationResponse{
		ID:        conv.PublicID,
		Object:    "conversation",
		Title:     conv.Title,
		CreatedAt: conv.CreatedAt.Unix(),
		Metadata:  conv.Metadata,
		Referrer:  conv.Referrer,
		ProjectID: conv.ProjectPublicID,
	}
	return response
}

// NewConversationListResponse creates a conversation list response
func NewConversationListResponse(conversations []*conversation.Conversation, hasMore bool, total int64) *ConversationListResponse {
	data := make([]ConversationResponse, 0, len(conversations))
	for _, conv := range conversations {
		if conv == nil {
			continue
		}
		resp := NewConversationResponse(conv)
		if resp != nil {
			data = append(data, *resp)
		}
	}

	firstID := ""
	lastID := ""
	if len(data) > 0 {
		firstID = data[0].ID
		lastID = data[len(data)-1].ID
	}

	return &ConversationListResponse{
		Object:  "list",
		Data:    data,
		FirstID: firstID,
		LastID:  lastID,
		HasMore: hasMore,
		Total:   total,
	}
}

// NewConversationDeletedResponse creates a delete response
func NewConversationDeletedResponse(publicID string) *ConversationDeletedResponse {
	return &ConversationDeletedResponse{
		ID:      publicID,
		Object:  "conversation.deleted",
		Deleted: true,
	}
}

// NewItemListResponse creates an item list response
func NewItemListResponse(items []conversation.Item, hasMore bool) *ItemListResponse {
	if len(items) == 0 {
		return &ItemListResponse{
			Object:  "list",
			Data:    []conversation.Item{},
			FirstID: "",
			LastID:  "",
			HasMore: false,
		}
	}

	return &ItemListResponse{
		Object:  "list",
		Data:    items,
		FirstID: items[0].PublicID,
		LastID:  items[len(items)-1].PublicID,
		HasMore: hasMore,
	}
}

// ItemResponse is just the item itself (OpenAI compatibility)
type ItemResponse = conversation.Item

// ConversationItemCreatedResponse represents the response after adding items
type ConversationItemCreatedResponse struct {
	Object  string              `json:"object"`
	Data    []conversation.Item `json:"data"`
	FirstID string              `json:"first_id"`
	LastID  string              `json:"last_id"`
	HasMore bool                `json:"has_more"`
}

// NewConversationItemCreatedResponse creates a response for created items
func NewConversationItemCreatedResponse(items []conversation.Item) *ConversationItemCreatedResponse {
	if len(items) == 0 {
		return &ConversationItemCreatedResponse{
			Object:  "list",
			Data:    []conversation.Item{},
			FirstID: "",
			LastID:  "",
			HasMore: false,
		}
	}

	return &ConversationItemCreatedResponse{
		Object:  "list",
		Data:    items,
		FirstID: items[0].PublicID,
		LastID:  items[len(items)-1].PublicID,
		HasMore: false,
	}
}
