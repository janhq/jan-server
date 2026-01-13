package chatrequests

import (
	"encoding/json"

	"jan-server/services/llm-api/internal/domain/conversation"

	"github.com/rs/zerolog/log"
	openai "github.com/sashabaranov/go-openai"
)

// FlexibleContentPart represents a content part that can handle multiple formats:
// - OpenAI format: {"type": "image_url", "image_url": {"url": "..."}}
// - Client format (browser-mcp): {"type": "image", "data": "<image url>", "mimeType": "image/png"}
// - Text format: {"type": "text", "text": "..."}
// - Tool result format: {"type": "tool_result", "tool_result": "..."}
type FlexibleContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// OpenAI format for images
	ImageURL *openai.ChatMessageImageURL `json:"image_url,omitempty"`
	// Client format for images (browser-mcp, etc.)
	Data        string `json:"data,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	Description string `json:"description,omitempty"`
	// Tool result content (browser-mcp, etc.)
	ToolResult string `json:"tool_result,omitempty"`
}

// ToOpenAIChatMessagePart converts FlexibleContentPart to openai.ChatMessagePart
func (p *FlexibleContentPart) ToOpenAIChatMessagePart() openai.ChatMessagePart {
	switch p.Type {
	case "text":
		return openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeText,
			Text: p.Text,
		}
	case "tool_result":
		// Tool result format (browser-mcp, etc.) - convert to text part
		// The tool_result field contains the actual content
		return openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeText,
			Text: p.ToolResult,
		}
	case "image_url":
		// Already in OpenAI format
		return openai.ChatMessagePart{
			Type:     openai.ChatMessagePartTypeImageURL,
			ImageURL: p.ImageURL,
		}
	case "image":
		// Client format - convert to OpenAI format
		// The data field contains the image URL
		if p.Data != "" {
			return openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeImageURL,
				ImageURL: &openai.ChatMessageImageURL{
					URL: p.Data,
				},
			}
		}
		// Fallback: return empty image_url part (will be filtered out later if needed)
		return openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeImageURL,
		}
	default:
		// Unknown type - try to preserve as text if possible
		if p.Text != "" {
			return openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeText,
				Text: p.Text,
			}
		}
		// Return empty part - will be filtered out by caller
		// Note: We can't return nil, so we return an empty image part which will be filtered
		// because empty text parts with omitempty cause {"type": "text"} without text field
		// which fails validation on some LLM providers
		return openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeImageURL, // Will be filtered out by caller
		}
	}
}

// parseFlexibleContentParts parses JSON-stringified content into flexible content parts
// and converts them to OpenAI format
func parseFlexibleContentParts(jsonContent string) ([]openai.ChatMessagePart, error) {
	var flexibleParts []FlexibleContentPart
	if err := json.Unmarshal([]byte(jsonContent), &flexibleParts); err != nil {
		return nil, err
	}

	result := make([]openai.ChatMessagePart, 0, len(flexibleParts))
	for _, fp := range flexibleParts {
		part := fp.ToOpenAIChatMessagePart()
		// Filter out empty image parts (no URL)
		if part.Type == openai.ChatMessagePartTypeImageURL && (part.ImageURL == nil || part.ImageURL.URL == "") {
			log.Warn().Str("original_type", fp.Type).Msg("Skipping empty image part with no URL/data")
			continue
		}
		// Filter out empty text parts (empty Text field would cause validation errors
		// because go-openai uses omitempty, resulting in {"type": "text"} without text field)
		if part.Type == openai.ChatMessagePartTypeText && part.Text == "" {
			log.Warn().Str("original_type", fp.Type).Msg("Skipping empty text part with no content")
			continue
		}
		result = append(result, part)
	}

	return result, nil
}

// ChatCompletionRequest extends OpenAI's ChatCompletionRequest with conversation support
type ChatCompletionRequest struct {
	openai.ChatCompletionRequest

	TopK              *int     `json:"top_k,omitempty"`
	RepetitionPenalty *float32 `json:"repetition_penalty,omitempty"`

	// Conversation can be either a string (conversation ID) or a conversation object
	// Items from this conversation are prepended to Messages for this response request.
	// Input items and output items from this response are automatically added to this conversation after completion.
	Conversation *ConversationReference `json:"conversation,omitempty"`
	// Store controls whether the latest input and generated response should be persisted
	Store *bool `json:"store,omitempty"`
	// StoreReasoning controls whether reasoning content (if present) should also be persisted
	StoreReasoning *bool `json:"store_reasoning,omitempty"`
	// DeepResearch enables the Deep Research mode which uses a specialized prompt
	// for conducting in-depth investigations with tool usage.
	// Requires a model with supports_reasoning: true capability.
	DeepResearch *bool `json:"deep_research,omitempty"`
	// EnableThinking controls whether reasoning/thinking capabilities should be used.
	// Defaults to true. When set to false for a model with supports_reasoning: true
	// and an instruct model configured, the instruct model will be used instead.
	EnableThinking *bool `json:"enable_thinking,omitempty"`
	// Image indicates the user wants to generate images.
	// When true, image generation tools will be made available.
	Image *bool `json:"image,omitempty"`
}

// ConversationReference can unmarshal from either a string (ID) or an object
type ConversationReference struct {
	ID     *string                    `json:"-"` // Conversation ID when provided as string
	Object *conversation.Conversation `json:"-"` // Conversation object when provided as object
}

// UnmarshalJSON implements custom unmarshaling to support both string and object types
// This is required because OpenAI's API allows conversation to be either:
//   - A string: "conversation": "conv_abc123"
//   - An object: "conversation": {"id": "conv_abc123", ...}
func (c *ConversationReference) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		c.ID = &str
		return nil
	}

	// If not a string, try to unmarshal as conversation object
	var obj conversation.Conversation
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	c.Object = &obj
	return nil
}

// MarshalJSON implements custom marshaling
func (c *ConversationReference) MarshalJSON() ([]byte, error) {
	if c.ID != nil {
		return json.Marshal(*c.ID)
	}
	if c.Object != nil {
		return json.Marshal(*c.Object)
	}
	return json.Marshal(nil)
}

// IsEmpty returns true if the conversation reference is empty
// Note: Includes nil check for defensive programming. Callers should still check for nil
// before calling this method to avoid potential panics.
func (c *ConversationReference) IsEmpty() bool {
	return c == nil || (c.ID == nil && c.Object == nil)
}

// UnmarshalJSON implements custom unmarshaling for ChatCompletionRequest
// to handle JSON-stringified content in messages (e.g., tool messages with images)
func (r *ChatCompletionRequest) UnmarshalJSON(data []byte) error {
	// Create an alias to avoid infinite recursion
	type Alias ChatCompletionRequest
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	// Unmarshal into the alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Post-process messages to handle JSON-stringified content
	for i := range r.Messages {
		msg := &r.Messages[i]

		// Check if content is a JSON-stringified array (starts with '[{')
		if msg.Content != "" && len(msg.Content) > 2 && msg.Content[0] == '[' && msg.Content[1] == '{' {
			log.Info().Int("message_index", i).Str("role", msg.Role).Str("content_prefix", msg.Content[:min(50, len(msg.Content))]).Msg("Detected JSON-stringified content")

			// Use flexible parser that handles both OpenAI and client formats
			parts, err := parseFlexibleContentParts(msg.Content)
			if err == nil {
				// Successfully parsed - log details for debugging
				for j, part := range parts {
					if part.Type == openai.ChatMessagePartTypeImageURL && part.ImageURL != nil {
						urlPreview := part.ImageURL.URL
						if len(urlPreview) > 80 {
							urlPreview = urlPreview[:80] + "..."
						}
						log.Info().Int("message_index", i).Int("part_index", j).Str("type", string(part.Type)).Str("image_url", urlPreview).Msg("Parsed image part")
					} else if part.Type == openai.ChatMessagePartTypeText {
						textPreview := part.Text
						if len(textPreview) > 50 {
							textPreview = textPreview[:50] + "..."
						}
						log.Debug().Int("message_index", i).Int("part_index", j).Str("type", string(part.Type)).Str("text_preview", textPreview).Msg("Parsed text part")
					}
				}
				log.Info().Int("message_index", i).Int("parts_count", len(parts)).Msg("Successfully parsed stringified JSON to MultiContent")
				msg.MultiContent = parts
				msg.Content = "" // Clear the string content
			} else {
				log.Warn().Err(err).Int("message_index", i).Msg("Failed to parse stringified JSON, leaving as-is for backward compatibility")
			}
			// If parsing fails, leave content as-is (backward compatibility)
		}
	}

	return nil
}

// GetID returns the conversation ID, whether it was provided directly or from an object
// Returns empty string if the reference is nil or has no ID.
func (c *ConversationReference) GetID() string {
	if c == nil {
		return ""
	}
	if c.ID != nil {
		return *c.ID
	}
	if c.Object != nil {
		return c.Object.PublicID
	}
	return ""
}

// GetConversation returns the conversation object if provided
// Returns nil if the reference is nil or contains only an ID string.
func (c *ConversationReference) GetConversation() *conversation.Conversation {
	if c == nil {
		return nil
	}
	return c.Object
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
