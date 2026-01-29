package chatresponses

import (
	openai "github.com/sashabaranov/go-openai"
)

// ChatCompletionResponse extends OpenAI's ChatCompletionResponse with conversation context
type ChatCompletionResponse struct {
	openai.ChatCompletionResponse
	Conversation *ConversationContext `json:"conversation,omitempty"`
	Trimmed      bool                 `json:"trimmed,omitempty"` // True if messages were trimmed to fit context
}

// ConversationContext represents the conversation associated with this response
type ConversationContext struct {
	ID    string  `json:"id"`              // The unique ID of the conversation
	Title *string `json:"title,omitempty"` // The title of the conversation (optional)
}

// NewChatCompletionResponse creates a response with optional conversation context
func NewChatCompletionResponse(openaiResp *openai.ChatCompletionResponse, conversationID string, conversationTitle *string, trimmed bool) *ChatCompletionResponse {
	resp := &ChatCompletionResponse{
		ChatCompletionResponse: *openaiResp,
		Trimmed:                trimmed,
	}

	if conversationID != "" {
		resp.Conversation = &ConversationContext{
			ID:    conversationID,
			Title: conversationTitle,
		}
	}

	return resp
}
