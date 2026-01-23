package messagesresponses

import (
	"encoding/json"
)

// AnthropicMessagesResponse represents an Anthropic Messages API response
type AnthropicMessagesResponse struct {
	// ID is the unique message identifier
	ID string `json:"id"`

	// Type is always "message"
	Type string `json:"type"`

	// Role is always "assistant"
	Role string `json:"role"`

	// Content is the array of content blocks
	Content []AnthropicResponseContentBlock `json:"content"`

	// Model is the model that generated the response
	Model string `json:"model"`

	// StopReason indicates why generation stopped
	// Values: "end_turn", "max_tokens", "stop_sequence", "tool_use"
	StopReason *string `json:"stop_reason"`

	// StopSequence is the stop sequence that triggered stopping (if any)
	StopSequence *string `json:"stop_sequence,omitempty"`

	// Usage contains token usage information
	Usage AnthropicUsage `json:"usage"`

	// Conversation context (Jan Server extension)
	Conversation *ConversationContext `json:"conversation,omitempty"`
}

// AnthropicResponseContentBlock represents a content block in the response
type AnthropicResponseContentBlock struct {
	// Type is "text", "tool_use", or "thinking"
	Type string `json:"type"`

	// Text content (for type="text")
	Text string `json:"text,omitempty"`

	// Tool use fields (for type="tool_use")
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// Thinking content (for type="thinking")
	Thinking string `json:"thinking,omitempty"`
}

// AnthropicUsage contains token usage information
type AnthropicUsage struct {
	// InputTokens is the number of input tokens
	InputTokens int `json:"input_tokens"`

	// OutputTokens is the number of output tokens
	OutputTokens int `json:"output_tokens"`

	// CacheCreationInputTokens for prompt caching
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`

	// CacheReadInputTokens for prompt caching
	CacheReadInputTokens int `json:"cache_read_input_tokens,omitempty"`
}

// ConversationContext for Jan Server integration
type ConversationContext struct {
	ID    string  `json:"id"`
	Title *string `json:"title,omitempty"`
}

// AnthropicCountTokensResponse represents a token counting response
type AnthropicCountTokensResponse struct {
	// InputTokens is the number of input tokens
	InputTokens int `json:"input_tokens"`
}

// ========== Streaming Event Types ==========

// AnthropicStreamEvent is the base interface for streaming events
type AnthropicStreamEvent struct {
	Type string `json:"type"`
}

// MessageStartEvent is sent at the start of a message
type MessageStartEvent struct {
	Type    string                    `json:"type"` // "message_start"
	Message AnthropicStreamingMessage `json:"message"`
}

// AnthropicStreamingMessage is the partial message in message_start
type AnthropicStreamingMessage struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"` // "message"
	Role         string         `json:"role"` // "assistant"
	Content      []interface{}  `json:"content"`
	Model        string         `json:"model"`
	StopReason   *string        `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence"`
	Usage        AnthropicUsage `json:"usage"`
}

// ContentBlockStartEvent signals the start of a content block
type ContentBlockStartEvent struct {
	Type         string                        `json:"type"` // "content_block_start"
	Index        int                           `json:"index"`
	ContentBlock AnthropicResponseContentBlock `json:"content_block"`
}

// ContentBlockDeltaEvent contains incremental content
type ContentBlockDeltaEvent struct {
	Type  string                `json:"type"` // "content_block_delta"
	Index int                   `json:"index"`
	Delta AnthropicContentDelta `json:"delta"`
}

// AnthropicContentDelta represents incremental content
type AnthropicContentDelta struct {
	Type string `json:"type"` // "text_delta", "input_json_delta", "thinking_delta"

	// For text_delta
	Text string `json:"text,omitempty"`

	// For input_json_delta (tool use arguments)
	PartialJSON string `json:"partial_json,omitempty"`

	// For thinking_delta
	Thinking string `json:"thinking,omitempty"`
}

// ContentBlockStopEvent signals the end of a content block
type ContentBlockStopEvent struct {
	Type  string `json:"type"` // "content_block_stop"
	Index int    `json:"index"`
}

// MessageDeltaEvent contains incremental message-level updates
type MessageDeltaEvent struct {
	Type  string                 `json:"type"` // "message_delta"
	Delta AnthropicMessageDelta  `json:"delta"`
	Usage *AnthropicMessageUsage `json:"usage,omitempty"`
}

// AnthropicMessageDelta represents message-level delta
type AnthropicMessageDelta struct {
	StopReason   *string `json:"stop_reason,omitempty"`
	StopSequence *string `json:"stop_sequence,omitempty"`
}

// AnthropicMessageUsage represents usage in message_delta
type AnthropicMessageUsage struct {
	OutputTokens int `json:"output_tokens"`
}

// MessageStopEvent signals the end of the message
type MessageStopEvent struct {
	Type string `json:"type"` // "message_stop"
}

// PingEvent is a keep-alive event
type PingEvent struct {
	Type string `json:"type"` // "ping"
}

// ErrorEvent represents a streaming error
type ErrorEvent struct {
	Type  string               `json:"type"` // "error"
	Error AnthropicErrorDetail `json:"error"`
}

// AnthropicErrorDetail contains error information
type AnthropicErrorDetail struct {
	Type    string `json:"type"`    // e.g., "invalid_request_error"
	Message string `json:"message"` // Error message
}

// ========== Helper Functions ==========

// NewMessageStartEvent creates a message_start event
func NewMessageStartEvent(id, model string, inputTokens int) *MessageStartEvent {
	return &MessageStartEvent{
		Type: "message_start",
		Message: AnthropicStreamingMessage{
			ID:         id,
			Type:       "message",
			Role:       "assistant",
			Content:    []interface{}{},
			Model:      model,
			StopReason: nil,
			Usage: AnthropicUsage{
				InputTokens:  inputTokens,
				OutputTokens: 0,
			},
		},
	}
}

// NewContentBlockStartEvent creates a content_block_start event
func NewContentBlockStartEvent(index int, blockType string) *ContentBlockStartEvent {
	block := AnthropicResponseContentBlock{
		Type: blockType,
	}
	if blockType == "text" {
		block.Text = ""
	}
	return &ContentBlockStartEvent{
		Type:         "content_block_start",
		Index:        index,
		ContentBlock: block,
	}
}

// NewContentBlockStartToolUseEvent creates a content_block_start event for tool_use
func NewContentBlockStartToolUseEvent(index int, id, name string) *ContentBlockStartEvent {
	return &ContentBlockStartEvent{
		Type:  "content_block_start",
		Index: index,
		ContentBlock: AnthropicResponseContentBlock{
			Type:  "tool_use",
			ID:    id,
			Name:  name,
			Input: json.RawMessage("{}"),
		},
	}
}

// NewTextDeltaEvent creates a text_delta event
func NewTextDeltaEvent(index int, text string) *ContentBlockDeltaEvent {
	return &ContentBlockDeltaEvent{
		Type:  "content_block_delta",
		Index: index,
		Delta: AnthropicContentDelta{
			Type: "text_delta",
			Text: text,
		},
	}
}

// NewInputJSONDeltaEvent creates an input_json_delta event for tool arguments
func NewInputJSONDeltaEvent(index int, partialJSON string) *ContentBlockDeltaEvent {
	return &ContentBlockDeltaEvent{
		Type:  "content_block_delta",
		Index: index,
		Delta: AnthropicContentDelta{
			Type:        "input_json_delta",
			PartialJSON: partialJSON,
		},
	}
}

// NewThinkingDeltaEvent creates a thinking_delta event
func NewThinkingDeltaEvent(index int, thinking string) *ContentBlockDeltaEvent {
	return &ContentBlockDeltaEvent{
		Type:  "content_block_delta",
		Index: index,
		Delta: AnthropicContentDelta{
			Type:     "thinking_delta",
			Thinking: thinking,
		},
	}
}

// NewContentBlockStopEvent creates a content_block_stop event
func NewContentBlockStopEvent(index int) *ContentBlockStopEvent {
	return &ContentBlockStopEvent{
		Type:  "content_block_stop",
		Index: index,
	}
}

// NewMessageDeltaEvent creates a message_delta event
func NewMessageDeltaEvent(stopReason string, outputTokens int) *MessageDeltaEvent {
	return &MessageDeltaEvent{
		Type: "message_delta",
		Delta: AnthropicMessageDelta{
			StopReason: &stopReason,
		},
		Usage: &AnthropicMessageUsage{
			OutputTokens: outputTokens,
		},
	}
}

// NewMessageStopEvent creates a message_stop event
func NewMessageStopEvent() *MessageStopEvent {
	return &MessageStopEvent{
		Type: "message_stop",
	}
}

// NewPingEvent creates a ping event
func NewPingEvent() *PingEvent {
	return &PingEvent{
		Type: "ping",
	}
}

// NewErrorEvent creates an error event
func NewErrorEvent(errorType, message string) *ErrorEvent {
	return &ErrorEvent{
		Type: "error",
		Error: AnthropicErrorDetail{
			Type:    errorType,
			Message: message,
		},
	}
}
