package messagesrequests

import (
	"encoding/json"
	"fmt"
)

// AnthropicMessagesRequest represents an Anthropic Messages API request
// https://docs.anthropic.com/en/api/messages
type AnthropicMessagesRequest struct {
	// Model is the model ID to use (required)
	Model string `json:"model" binding:"required"`

	// Messages is the list of input messages (required)
	Messages []AnthropicMessage `json:"messages" binding:"required,min=1"`

	// MaxTokens is the maximum number of tokens to generate (required)
	MaxTokens int `json:"max_tokens" binding:"required,min=1"`

	// System is an optional system prompt
	System string `json:"system,omitempty"`

	// Metadata is optional request metadata
	Metadata *AnthropicMetadata `json:"metadata,omitempty"`

	// StopSequences is an optional list of stop sequences
	StopSequences []string `json:"stop_sequences,omitempty"`

	// Stream enables streaming responses
	Stream bool `json:"stream,omitempty"`

	// Temperature controls randomness (0.0 to 1.0)
	Temperature *float32 `json:"temperature,omitempty"`

	// TopK samples from the top K options
	TopK *int `json:"top_k,omitempty"`

	// TopP uses nucleus sampling
	TopP *float32 `json:"top_p,omitempty"`

	// Tools is the list of tools available to the model
	Tools []AnthropicTool `json:"tools,omitempty"`

	// ToolChoice specifies how the model should use tools
	ToolChoice *AnthropicToolChoice `json:"tool_choice,omitempty"`

	// Thinking enables extended thinking (Claude 3.5+)
	Thinking *AnthropicThinking `json:"thinking,omitempty"`

	// Conversation reference for Jan Server integration
	Conversation *ConversationReference `json:"conversation,omitempty"`

	// Store controls whether to persist to conversation
	Store *bool `json:"store,omitempty"`

	// StoreReasoning controls whether to persist reasoning content
	StoreReasoning *bool `json:"store_reasoning,omitempty"`
}

// AnthropicMessage represents a message in the conversation
type AnthropicMessage struct {
	// Role is the message role: "user" or "assistant"
	Role string `json:"role" binding:"required,oneof=user assistant"`

	// Content can be a string or array of content blocks
	Content AnthropicContent `json:"content" binding:"required"`
}

// AnthropicContent can be either a string or an array of content blocks
type AnthropicContent struct {
	Text   string                  `json:"-"` // For simple string content
	Blocks []AnthropicContentBlock `json:"-"` // For multi-part content
}

// UnmarshalJSON handles both string and array content formats
func (c *AnthropicContent) UnmarshalJSON(data []byte) error {
	// Try string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		c.Text = str
		c.Blocks = nil
		return nil
	}

	// Try array of content blocks
	var blocks []AnthropicContentBlock
	if err := json.Unmarshal(data, &blocks); err != nil {
		return fmt.Errorf("content must be a string or array of content blocks: %w", err)
	}
	c.Text = ""
	c.Blocks = blocks
	return nil
}

// MarshalJSON serializes content back to JSON
func (c AnthropicContent) MarshalJSON() ([]byte, error) {
	if c.Text != "" {
		return json.Marshal(c.Text)
	}
	return json.Marshal(c.Blocks)
}

// IsText returns true if the content is a simple text string
func (c *AnthropicContent) IsText() bool {
	return c.Text != "" || len(c.Blocks) == 0
}

// GetText returns the text content (either direct or concatenated from blocks)
func (c *AnthropicContent) GetText() string {
	if c.Text != "" {
		return c.Text
	}
	var result string
	for _, block := range c.Blocks {
		if block.Type == "text" && block.Text != "" {
			result += block.Text
		}
	}
	return result
}

// AnthropicContentBlock represents a content block in a message
type AnthropicContentBlock struct {
	// Type is the block type: "text", "image", "tool_use", "tool_result"
	Type string `json:"type" binding:"required"`

	// Text content (for type="text")
	Text string `json:"text,omitempty"`

	// Source for image content (for type="image")
	Source *AnthropicImageSource `json:"source,omitempty"`

	// Tool use fields (for type="tool_use")
	ID    string          `json:"id,omitempty"`    // Tool use ID
	Name  string          `json:"name,omitempty"`  // Tool name
	Input json.RawMessage `json:"input,omitempty"` // Tool input parameters

	// Tool result fields (for type="tool_result")
	ToolUseID string `json:"tool_use_id,omitempty"` // References the tool_use ID
	IsError   bool   `json:"is_error,omitempty"`    // Whether this is an error result
	// Content can be string or array for tool_result
	Content *AnthropicToolResultContent `json:"content,omitempty"`
}

// AnthropicToolResultContent handles tool_result content which can be string or array
type AnthropicToolResultContent struct {
	Text   string                  `json:"-"`
	Blocks []AnthropicContentBlock `json:"-"`
}

// UnmarshalJSON handles both string and array formats for tool result content
func (c *AnthropicToolResultContent) UnmarshalJSON(data []byte) error {
	// Try string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		c.Text = str
		c.Blocks = nil
		return nil
	}

	// Try array of content blocks
	var blocks []AnthropicContentBlock
	if err := json.Unmarshal(data, &blocks); err != nil {
		return fmt.Errorf("tool_result content must be a string or array: %w", err)
	}
	c.Text = ""
	c.Blocks = blocks
	return nil
}

// MarshalJSON serializes tool result content
func (c AnthropicToolResultContent) MarshalJSON() ([]byte, error) {
	if c.Text != "" {
		return json.Marshal(c.Text)
	}
	return json.Marshal(c.Blocks)
}

// GetText returns the text content
func (c *AnthropicToolResultContent) GetText() string {
	if c.Text != "" {
		return c.Text
	}
	var result string
	for _, block := range c.Blocks {
		if block.Type == "text" && block.Text != "" {
			result += block.Text
		}
	}
	return result
}

// AnthropicImageSource represents an image source
type AnthropicImageSource struct {
	// Type is "base64" or "url"
	Type string `json:"type" binding:"required"`

	// MediaType is the MIME type (e.g., "image/jpeg", "image/png")
	MediaType string `json:"media_type,omitempty"`

	// Data is the base64-encoded image data (for type="base64")
	Data string `json:"data,omitempty"`

	// URL is the image URL (for type="url")
	URL string `json:"url,omitempty"`
}

// AnthropicTool represents a tool definition
type AnthropicTool struct {
	// Name is the tool name
	Name string `json:"name" binding:"required"`

	// Description describes what the tool does
	Description string `json:"description,omitempty"`

	// InputSchema is the JSON Schema for the tool's input
	InputSchema json.RawMessage `json:"input_schema" binding:"required"`
}

// AnthropicToolChoice specifies how tools should be used
type AnthropicToolChoice struct {
	// Type is "auto", "any", or "tool"
	Type string `json:"type" binding:"required"`

	// Name is required when type is "tool"
	Name string `json:"name,omitempty"`

	// DisableParallelToolUse prevents parallel tool calls
	DisableParallelToolUse bool `json:"disable_parallel_tool_use,omitempty"`
}

// AnthropicThinking configures extended thinking
type AnthropicThinking struct {
	// Type must be "enabled"
	Type string `json:"type" binding:"required"`

	// BudgetTokens is the maximum tokens for thinking
	BudgetTokens int `json:"budget_tokens,omitempty"`
}

// AnthropicMetadata contains request metadata
type AnthropicMetadata struct {
	// UserID is an external user identifier
	UserID string `json:"user_id,omitempty"`
}

// ConversationReference for Jan Server conversation integration
type ConversationReference struct {
	ID *string `json:"-"` // Conversation ID when provided as string
}

// UnmarshalJSON handles string conversation reference
func (c *ConversationReference) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		c.ID = &str
		return nil
	}

	// Try as object with id field
	var obj struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	c.ID = &obj.ID
	return nil
}

// MarshalJSON serializes conversation reference
func (c *ConversationReference) MarshalJSON() ([]byte, error) {
	if c.ID != nil {
		return json.Marshal(*c.ID)
	}
	return json.Marshal(nil)
}

// GetID returns the conversation ID
func (c *ConversationReference) GetID() string {
	if c == nil || c.ID == nil {
		return ""
	}
	return *c.ID
}

// AnthropicCountTokensRequest represents a token counting request
type AnthropicCountTokensRequest struct {
	// Model is the model ID to use (required)
	Model string `json:"model" binding:"required"`

	// Messages is the list of input messages (required)
	Messages []AnthropicMessage `json:"messages" binding:"required,min=1"`

	// System is an optional system prompt
	System string `json:"system,omitempty"`

	// Tools is the list of tools to count tokens for
	Tools []AnthropicTool `json:"tools,omitempty"`

	// ToolChoice specifies tool usage
	ToolChoice *AnthropicToolChoice `json:"tool_choice,omitempty"`

	// Thinking configuration
	Thinking *AnthropicThinking `json:"thinking,omitempty"`
}
