package conversation

import (
	"time"
)

// Conversation represents a chat conversation.
type Conversation struct {
	ID           string
	UserID       string
	Title        string
	ModelID      string
	SystemPrompt string
	Metadata     map[string]any
	IsArchived   bool
	IsPinned     bool
	SharedToken  *string
	SharedAt     *time.Time
	ParentID     *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Messages     []Message
}

// Message represents a message in a conversation.
type Message struct {
	ID             string
	ConversationID string
	Role           string // user, assistant, system, tool
	Content        string
	Name           *string
	ToolCallID     *string
	ToolCalls      []ToolCall
	Attachments    []Attachment
	Metadata       map[string]any
	TokensInput    int
	TokensOutput   int
	ModelID        *string
	ParentID       *string
	CreatedAt      time.Time
}

// ToolCall represents a tool call made by the assistant.
type ToolCall struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Function  FunctionCall `json:"function"`
}

// FunctionCall represents the function being called.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Attachment represents a file attachment.
type Attachment struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // file, image
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	URL      string `json:"url,omitempty"`
}

// CreateConversationRequest contains data for creating a conversation.
type CreateConversationRequest struct {
	Title        string
	ModelID      string
	SystemPrompt string
	Metadata     map[string]any
}

// UpdateConversationRequest contains data for updating a conversation.
type UpdateConversationRequest struct {
	Title        *string
	ModelID      *string
	SystemPrompt *string
	IsArchived   *bool
	IsPinned     *bool
	Metadata     map[string]any
}

// ListConversationsFilter contains filters for listing conversations.
type ListConversationsFilter struct {
	UserID     string
	IsArchived *bool
	IsPinned   *bool
	Search     string
	Limit      int
	Offset     int
}

// ConversationResponse is the API response for a conversation.
type ConversationResponse struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	ModelID      string         `json:"model_id,omitempty"`
	SystemPrompt string         `json:"system_prompt,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	IsArchived   bool           `json:"is_archived"`
	IsPinned     bool           `json:"is_pinned"`
	SharedToken  *string        `json:"shared_token,omitempty"`
	SharedAt     *time.Time     `json:"shared_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	MessageCount int            `json:"message_count,omitempty"`
}

// MessageResponse is the API response for a message.
type MessageResponse struct {
	ID           string         `json:"id"`
	Role         string         `json:"role"`
	Content      string         `json:"content"`
	Name         *string        `json:"name,omitempty"`
	ToolCallID   *string        `json:"tool_call_id,omitempty"`
	ToolCalls    []ToolCall     `json:"tool_calls,omitempty"`
	Attachments  []Attachment   `json:"attachments,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	TokensInput  int            `json:"tokens_input,omitempty"`
	TokensOutput int            `json:"tokens_output,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// ToResponse converts a Conversation to ConversationResponse.
func (c *Conversation) ToResponse() ConversationResponse {
	return ConversationResponse{
		ID:           c.ID,
		Title:        c.Title,
		ModelID:      c.ModelID,
		SystemPrompt: c.SystemPrompt,
		Metadata:     c.Metadata,
		IsArchived:   c.IsArchived,
		IsPinned:     c.IsPinned,
		SharedToken:  c.SharedToken,
		SharedAt:     c.SharedAt,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
		MessageCount: len(c.Messages),
	}
}

// ToResponse converts a Message to MessageResponse.
func (m *Message) ToResponse() MessageResponse {
	return MessageResponse{
		ID:           m.ID,
		Role:         m.Role,
		Content:      m.Content,
		Name:         m.Name,
		ToolCallID:   m.ToolCallID,
		ToolCalls:    m.ToolCalls,
		Attachments:  m.Attachments,
		Metadata:     m.Metadata,
		TokensInput:  m.TokensInput,
		TokensOutput: m.TokensOutput,
		CreatedAt:    m.CreatedAt,
	}
}
