package chatrequests

import (
	"encoding/json"
	"strings"

	"jan-server/services/llm-api/internal/domain/conversation"

	"github.com/rs/zerolog/log"
	openai "github.com/sashabaranov/go-openai"
)

// FileURLContent represents a file URL reference with metadata
type FileURLContent struct {
	URL      string `json:"url"`
	Detail   string `json:"detail,omitempty"`   // "auto", "low", "high"
	Filename string `json:"filename,omitempty"` // Original filename
	Name     string `json:"name,omitempty"`     // Display name / filename
	MimeType string `json:"mime_type,omitempty"`
}

// FlexibleContentPart represents a content part that can handle multiple formats:
// - OpenAI format: {"type": "image_url", "image_url": {"url": "..."}}
// - File URL format: {"type": "file_url", "file_url": {"url": "...", "filename": "..."}}
// - File format: {"type": "file", "file": {"url": "...", "name": "...", "mime_type": "..."}}
// - Client format (browser-mcp): {"type": "image", "data": "<image url>", "mimeType": "image/png"}
// - Text format: {"type": "text", "text": "..."} or {"type": "input_text", "input_text": "..."}
// - Tool result format: {"type": "tool_result", "tool_result": "..."}
type FlexibleContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// Responses-style text format
	InputText string `json:"input_text,omitempty"`
	// OpenAI format for images
	ImageURL *openai.ChatMessageImageURL `json:"image_url,omitempty"`
	// File URL format for document attachments
	FileURL *FileURLContent `json:"file_url,omitempty"`
	// File format for document attachments
	File *FileURLContent `json:"file,omitempty"`
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
	case "input_text":
		text := p.InputText
		if text == "" {
			text = p.Text
		}
		return openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeText,
			Text: text,
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
	case "file_url":
		// File URL format - mark with special prefix for later text injection
		// The actual file content will be injected in chat handler
		if p.FileURL != nil && p.FileURL.URL != "" {
			filename := p.FileURL.Filename
			if filename == "" {
				filename = p.FileURL.Name
			}
			if filename == "" {
				filename = "document"
			}
			// Return a text placeholder that will be replaced with actual content
			// Format: [FILE_URL:url:filename:mime_type]
			mimeType := p.FileURL.MimeType
			if mimeType == "" {
				mimeType = "application/octet-stream"
			}
			return openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeText,
				Text: "[FILE_URL:" + p.FileURL.URL + ":" + filename + ":" + mimeType + "]",
			}
		}
		// Fallback: return empty part
		return openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeImageURL,
		}
	case "file":
		// File format - mark with special prefix for later text injection
		if p.File != nil && p.File.URL != "" {
			filename := p.File.Filename
			if filename == "" {
				filename = p.File.Name
			}
			if filename == "" {
				filename = "document"
			}
			mimeType := p.File.MimeType
			if mimeType == "" {
				mimeType = "application/octet-stream"
			}
			return openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeText,
				Text: "[FILE_URL:" + p.File.URL + ":" + filename + ":" + mimeType + "]",
			}
		}
		// Fallback: return empty part
		return openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeImageURL,
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

// IsFileURLPlaceholder checks if a text contains a file URL placeholder
func IsFileURLPlaceholder(text string) bool {
	if len(text) < 12 {
		return false
	}
	if !strings.HasPrefix(text, "[FILE_URL:") {
		return false
	}
	if !strings.HasSuffix(text, "]") {
		return false
	}
	return true
}

// ParseFileURLPlaceholder extracts URL, filename, and mime type from a file URL placeholder
// Returns url, filename, mimeType, ok
func ParseFileURLPlaceholder(text string) (string, string, string, bool) {
	if !IsFileURLPlaceholder(text) {
		return "", "", "", false
	}
	// Remove [FILE_URL: prefix and ] suffix
	inner := text[10 : len(text)-1]
	// Split by : - but URL may contain :, so we need to be careful
	// Format: url:filename:mime_type
	// Find last two colons
	lastColon := -1
	secondLastColon := -1
	for i := len(inner) - 1; i >= 0; i-- {
		if inner[i] == ':' {
			if lastColon == -1 {
				lastColon = i
			} else {
				secondLastColon = i
				break
			}
		}
	}
	if secondLastColon == -1 || lastColon == -1 {
		return "", "", "", false
	}
	url := inner[:secondLastColon]
	filename := inner[secondLastColon+1 : lastColon]
	mimeType := inner[lastColon+1:]
	return url, filename, mimeType, true
}

// convertFlexibleContentParts converts flexible content parts into OpenAI format
func convertFlexibleContentParts(flexibleParts []FlexibleContentPart) []openai.ChatMessagePart {
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
	return result
}

// parseFlexibleContentParts parses JSON-stringified content into flexible content parts
// and converts them to OpenAI format
func parseFlexibleContentParts(jsonContent string) ([]openai.ChatMessagePart, error) {
	var flexibleParts []FlexibleContentPart
	if err := json.Unmarshal([]byte(jsonContent), &flexibleParts); err != nil {
		return nil, err
	}
	return convertFlexibleContentParts(flexibleParts), nil
}

// parseFlexibleContentPartsFromRaw parses JSON array content into flexible content parts
// and converts them to OpenAI format
func parseFlexibleContentPartsFromRaw(rawContent json.RawMessage) ([]openai.ChatMessagePart, error) {
	var flexibleParts []FlexibleContentPart
	if err := json.Unmarshal(rawContent, &flexibleParts); err != nil {
		return nil, err
	}
	return convertFlexibleContentParts(flexibleParts), nil
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
	// Agent indicates the client prefers agent tool usage when available.
	Agent *bool `json:"agent,omitempty"`
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
// to handle JSON content arrays (including input_text/file parts) and JSON-stringified content
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

	// Parse messages with raw content to support content arrays (input_text/file/etc.)
	var raw struct {
		Messages []json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if len(raw.Messages) > 0 {
		parsedMessages := make([]openai.ChatCompletionMessage, 0, len(raw.Messages))
		for i, rawMsg := range raw.Messages {
			var msg struct {
				Role             string               `json:"role"`
				Content          json.RawMessage      `json:"content,omitempty"`
				Refusal          string               `json:"refusal,omitempty"`
				Name             string               `json:"name,omitempty"`
				ReasoningContent string               `json:"reasoning_content,omitempty"`
				FunctionCall     *openai.FunctionCall `json:"function_call,omitempty"`
				ToolCalls        []openai.ToolCall    `json:"tool_calls,omitempty"`
				ToolCallID       string               `json:"tool_call_id,omitempty"`
			}
			if err := json.Unmarshal(rawMsg, &msg); err != nil {
				return err
			}

			parsed := openai.ChatCompletionMessage{
				Role:             msg.Role,
				Refusal:          msg.Refusal,
				Name:             msg.Name,
				ReasoningContent: msg.ReasoningContent,
				FunctionCall:     msg.FunctionCall,
				ToolCalls:        msg.ToolCalls,
				ToolCallID:       msg.ToolCallID,
			}

			contentRaw := strings.TrimSpace(string(msg.Content))
			if contentRaw != "" && contentRaw != "null" {
				switch contentRaw[0] {
				case '"':
					var contentStr string
					if err := json.Unmarshal(msg.Content, &contentStr); err != nil {
						return err
					}
					// Check if content is a JSON-stringified array (starts with '[{')
					if len(contentStr) > 2 && contentStr[0] == '[' && contentStr[1] == '{' {
						log.Info().Int("message_index", i).Str("role", msg.Role).Str("content_prefix", contentStr[:min(50, len(contentStr))]).Msg("Detected JSON-stringified content")
						parts, err := parseFlexibleContentParts(contentStr)
						if err == nil {
							parsed.MultiContent = parts
							continue
						}
						log.Warn().Err(err).Int("message_index", i).Msg("Failed to parse stringified JSON, leaving as-is for backward compatibility")
					}
					parsed.Content = contentStr
				case '[':
					parts, err := parseFlexibleContentPartsFromRaw(msg.Content)
					if err == nil {
						parsed.MultiContent = parts
					} else {
						log.Warn().Err(err).Int("message_index", i).Msg("Failed to parse content array as flexible parts")
						// Fallback to OpenAI parts if possible
						var openAIParts []openai.ChatMessagePart
						if errFallback := json.Unmarshal(msg.Content, &openAIParts); errFallback == nil {
							parsed.MultiContent = openAIParts
						} else {
							parsed.Content = contentRaw
						}
					}
				default:
					parsed.Content = contentRaw
				}
			}

			parsedMessages = append(parsedMessages, parsed)
		}

		r.Messages = parsedMessages
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
