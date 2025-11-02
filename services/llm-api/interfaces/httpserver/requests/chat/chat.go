package chatrequests

import (
	"encoding/json"
)

// ChatCompletionRequest represents a chat completion request with conversation support
type ChatCompletionRequest struct {
	// Model is the model ID to use for completion
	Model string `json:"model" binding:"required"`

	// Messages is the array of chat messages
	Messages []ChatMessage `json:"messages" binding:"required,min=1"`

	// Conversation can be either a string (conversation ID) or an object with id
	// Items from this conversation are prepended to Messages for context
	Conversation *ConversationReference `json:"conversation,omitempty"`

	// Store controls whether to persist input and response to conversation
	Store *bool `json:"store,omitempty"`

	// StoreReasoning controls whether to persist reasoning content
	StoreReasoning *bool `json:"store_reasoning,omitempty"`

	// Stream indicates if response should be streamed via SSE
	Stream bool `json:"stream,omitempty"`

	// Temperature controls randomness (0.0 to 2.0)
	Temperature *float32 `json:"temperature,omitempty"`

	// TopP controls nucleus sampling
	TopP *float32 `json:"top_p,omitempty"`

	// MaxTokens limits the response length
	MaxTokens *int `json:"max_tokens,omitempty"`

	// Other OpenAI-compatible parameters
	FrequencyPenalty *float32               `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float32               `json:"presence_penalty,omitempty"`
	Stop             interface{}            `json:"stop,omitempty"` // string or []string
	N                *int                   `json:"n,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ChatMessage represents a single chat message
type ChatMessage struct {
	Role             string      `json:"role" binding:"required"`
	Content          interface{} `json:"content"` // string or array of content parts
	Name             string      `json:"name,omitempty"`
	FunctionCall     interface{} `json:"function_call,omitempty"`
	ToolCalls        interface{} `json:"tool_calls,omitempty"`
	ToolCallID       string      `json:"tool_call_id,omitempty"`
	ReasoningContent string      `json:"reasoning_content,omitempty"`
}

// ConversationReference can be a string ID or an object
type ConversationReference struct {
	ID     *string                `json:"-"` // Conversation ID when provided as string
	Object map[string]interface{} `json:"-"` // Conversation object when provided as object
}

// UnmarshalJSON implements custom unmarshaling for ConversationReference
func (c *ConversationReference) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		c.ID = &str
		return nil
	}

	// Try to unmarshal as object
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	c.Object = obj
	return nil
}

// MarshalJSON implements custom marshaling
func (c *ConversationReference) MarshalJSON() ([]byte, error) {
	if c.ID != nil {
		return json.Marshal(*c.ID)
	}
	if c.Object != nil {
		return json.Marshal(c.Object)
	}
	return json.Marshal(nil)
}

// IsEmpty returns true if the conversation reference is empty
func (c *ConversationReference) IsEmpty() bool {
	return c == nil || (c.ID == nil && c.Object == nil)
}

// GetID returns the conversation ID
func (c *ConversationReference) GetID() string {
	if c == nil {
		return ""
	}
	if c.ID != nil {
		return *c.ID
	}
	if c.Object != nil {
		if id, ok := c.Object["id"].(string); ok {
			return id
		}
	}
	return ""
}
