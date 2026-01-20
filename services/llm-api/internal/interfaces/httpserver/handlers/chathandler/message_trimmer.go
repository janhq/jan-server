package chathandler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	openai "github.com/sashabaranov/go-openai"
)

const (
	// DefaultContextLength is used when model context length is unknown.
	DefaultContextLength = 220000 // 220k tokens as fallback

	// TokenEstimateRatio estimates ~4 characters per token (conservative estimate).
	TokenEstimateRatio = 4

	// TokenEstimateRatioCJK estimates ~1.5 characters per token for CJK content.
	TokenEstimateRatioCJK = 1.5

	// MinMessagesToKeep ensures we always keep system prompt + at least one user message.
	MinMessagesToKeep = 2

	// MinMessagesTokenFloor is the hard minimum tokens required for messages.
	MinMessagesTokenFloor = 1000

	// SafetyMarginRatio reserves space for response and overhead (15% margin for response).
	SafetyMarginRatio = 0.66

	// FixedOverheadTokens is fixed overhead for API request structure.
	FixedOverheadTokens = 100

	// MaxToolSchemaBytes caps tool parameter schema size to prevent runaway serialization.
	MaxToolSchemaBytes = 16384 // 16KB

	// MaxToolResultTokens is max tokens per tool result before truncation.
	MaxToolResultTokens = 20000

	// MaxToolArgumentTokens is max tokens for tool call arguments.
	MaxToolArgumentTokens = 2000

	// MaxUserContentTokens is max tokens per user message text content before truncation.
	MaxUserContentTokens = 240000

	// MaxMultiContentTextTokens is max tokens per text part in multi-content arrays.
	MaxMultiContentTextTokens = 6000

	// Image limits to prevent context overflow
	MaxToolImages = 6  // Maximum images across all tool messages
	MaxUserImages = 15 // Maximum images across all user messages

	// Image token estimates (conservative for safety)
	ImageTokensLowRes  = 85  // Low resolution image
	ImageTokensHighRes = 850 // High resolution image (average)
)

// ===============================
// TokenBudget - Central budget management
// ===============================

// TokenBudget represents the complete token budget for a request.
// This struct flows through the trimmer so callers don't recompute.
type TokenBudget struct {
	ContextLength       int // Total context window size
	ToolsTokens         int // Tokens consumed by tool definitions
	MaxCompletionTokens int // User-requested max_tokens (0 = use default margin)
	FixedOverhead       int // Fixed overhead (API structure, formatting)

	// Computed fields (set by Validate())
	AvailableForMessages int // Tokens available for message content
	ResponseReserve      int // Tokens reserved for response
}

// Validate checks the budget and computes available space.
// Returns error if budget is invalid (e.g., max_tokens exceeds context).
func (b *TokenBudget) Validate() error {
	if b.ContextLength <= 0 {
		return fmt.Errorf("invalid context length: %d (must be positive)", b.ContextLength)
	}

	// Calculate response reserve
	if b.MaxCompletionTokens > 0 {
		b.ResponseReserve = b.MaxCompletionTokens
	} else {
		b.ResponseReserve = int(float64(b.ContextLength) * (1 - SafetyMarginRatio))
	}

	// Calculate available space for messages
	b.AvailableForMessages = b.ContextLength - b.ToolsTokens - b.ResponseReserve - b.FixedOverhead

	// Hard floor check: if available space is too small, return error
	if b.AvailableForMessages < MinMessagesTokenFloor {
		return fmt.Errorf(
			"token budget exhausted: context=%d, tools=%d, response_reserve=%d, overhead=%d → only %d tokens available (minimum required: %d). Reduce max_tokens, use fewer tools, or choose a model with larger context",
			b.ContextLength, b.ToolsTokens, b.ResponseReserve, b.FixedOverhead,
			b.AvailableForMessages, MinMessagesTokenFloor,
		)
	}

	return nil
}

// ===============================
// Tool Token Estimation
// ===============================

// EstimateToolsTokens estimates tokens for tool definitions.
// Logs warnings for marshal errors and caps schema size.
func EstimateToolsTokens(tools []openai.Tool) int {
	if len(tools) == 0 {
		return 0
	}

	total := 50 // Base overhead for tools array structure

	for _, tool := range tools {
		total += 20 // Overhead per tool
		if tool.Function != nil {
			total += estimateTokenCount(tool.Function.Name)
			total += estimateTokenCount(tool.Function.Description)

			// Parameters schema can be large - cap and handle errors
			if tool.Function.Parameters != nil {
				paramsJSON, err := json.Marshal(tool.Function.Parameters)
				if err != nil {
					total += 200 // Conservative fallback
					continue
				}

				// Cap schema size to prevent extremely large schemas
				if len(paramsJSON) > MaxToolSchemaBytes {
					paramsJSON = paramsJSON[:MaxToolSchemaBytes]
				}

				total += estimateTokenCount(string(paramsJSON))
			}
		}
	}

	return total
}

// ===============================
// Image Token Estimation
// ===============================

// estimateImageTokens estimates tokens for an image based on detail level.
// Missing or empty detail is normalized to "high" for conservative estimation.
func estimateImageTokens(imageURL *openai.ChatMessageImageURL) int {
	if imageURL == nil {
		return 0
	}

	// Normalize: treat empty/missing detail as "high" for safety
	detail := imageURL.Detail
	if detail == "" {
		detail = openai.ImageURLDetailHigh
	}

	switch detail {
	case openai.ImageURLDetailLow:
		return ImageTokensLowRes
	case openai.ImageURLDetailHigh:
		return ImageTokensHighRes
	case openai.ImageURLDetailAuto:
		return ImageTokensHighRes
	default:
		return ImageTokensHighRes
	}
}

// estimateMultiContentTokens handles different content part types.
func estimateMultiContentTokens(parts []openai.ChatMessagePart) int {
	total := 0
	for _, part := range parts {
		switch part.Type {
		case openai.ChatMessagePartTypeText:
			total += estimateTokenCount(part.Text)
		case openai.ChatMessagePartTypeImageURL:
			total += estimateImageTokens(part.ImageURL)
		}
	}
	return total
}

// countImagesInToolResult detects images embedded in tool result content.
// Uses lightweight JSON sniffing to avoid false positives/negatives.
func countImagesInToolResult(content string) int {
	if len(content) == 0 {
		return 0
	}

	// Quick pre-check: if content doesn't look like JSON array, skip parsing
	trimmed := strings.TrimSpace(content)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return countDataURLImages(content)
	}

	// Try to parse as JSON array
	var items []map[string]any
	if err := json.Unmarshal([]byte(content), &items); err != nil {
		return countDataURLImages(content)
	}

	imageCount := 0
	for _, item := range items {
		if isImageType(item) {
			imageCount++
			continue
		}
		if hasImageDataURL(item) {
			imageCount++
		}
	}

	return imageCount
}

// isImageType checks if a map represents an image content type.
func isImageType(item map[string]any) bool {
	for _, key := range []string{"type", "kind", "contentType"} {
		if val, ok := item[key].(string); ok {
			if val == "image" || strings.HasPrefix(val, "image/") {
				return true
			}
		}
	}
	return false
}

// hasImageDataURL checks if a map contains an image data URL.
func hasImageDataURL(item map[string]any) bool {
	for _, key := range []string{"data", "url", "src", "imageUrl"} {
		if val, ok := item[key].(string); ok {
			if strings.HasPrefix(val, "data:image/") {
				return true
			}
		}
	}
	return false
}

// countDataURLImages counts data URL images in non-JSON content (fallback).
func countDataURLImages(content string) int {
	count := 0
	count += strings.Count(content, "data:image/png;base64")
	count += strings.Count(content, "data:image/jpeg;base64")
	count += strings.Count(content, "data:image/webp;base64")
	count += strings.Count(content, "data:image/gif;base64")
	return count
}

// ===============================
// CJK Character Detection
// ===============================

// isCJK checks if a rune is a CJK character.
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Unified Ideographs Extension A
		(r >= 0x3040 && r <= 0x309F) || // Hiragana
		(r >= 0x30A0 && r <= 0x30FF) || // Katakana
		(r >= 0xAC00 && r <= 0xD7AF) // Hangul Syllables
}

// estimateTokenCount provides a rough estimate of token count for content.
// Handles CJK characters with adjusted ratio for better accuracy.
func estimateTokenCount(content interface{}) int {
	var text string
	switch v := content.(type) {
	case string:
		text = v
	case nil:
		return 0
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			return 50 // Fallback estimate for marshal errors
		}
		text = string(bytes)
	}

	if len(text) == 0 {
		return 0
	}

	runeCount := utf8.RuneCountInString(text)

	// Count CJK characters for adjusted estimation
	cjkCount := 0
	for _, r := range text {
		if isCJK(r) {
			cjkCount++
		}
	}

	// If more than 30% CJK, use CJK ratio for that portion
	if runeCount > 0 && float64(cjkCount)/float64(runeCount) > 0.3 {
		cjkTokens := float64(cjkCount) / TokenEstimateRatioCJK
		otherTokens := float64(runeCount-cjkCount) / float64(TokenEstimateRatio)
		return int(cjkTokens + otherTokens)
	}

	return runeCount / TokenEstimateRatio
}

// estimateMessagesTokenCount estimates total tokens across all messages.
// Includes proper handling for images in MultiContent and tool results.
func estimateMessagesTokenCount(messages []openai.ChatCompletionMessage) int {
	total := 0
	for _, msg := range messages {
		// Add overhead for role and structure (~10 tokens per message)
		total += 10
		total += estimateTokenCount(msg.Content)

		// Handle multipart content with image support
		if len(msg.MultiContent) > 0 {
			total += estimateMultiContentTokens(msg.MultiContent)
		}

		// Count images in tool results (browser screenshots, etc.)
		if msg.Role == "tool" && msg.Content != "" {
			imageCount := countImagesInToolResult(msg.Content)
			total += imageCount * ImageTokensHighRes
		}

		// Add tokens for tool calls
		if msg.ToolCalls != nil {
			for _, tc := range msg.ToolCalls {
				total += 20 // Overhead for tool call structure
				total += estimateTokenCount(tc.Function.Name)
				total += estimateTokenCount(tc.Function.Arguments)
			}
		}
	}
	return total
}

// ===============================
// Tool Content Truncation
// ===============================

// TruncationEvent represents a truncation for logging/metrics.
type TruncationEvent struct {
	MessageIndex    int
	ToolName        string
	ToolCallID      string
	OriginalTokens  int
	TruncatedTokens int
	TruncationType  string // "tool_result" or "tool_argument"
}

// truncateTextPreservingJSON truncates text content while trying to preserve JSON structure.
// If content is JSON-stringified MultiContent, it parses and truncates the nested text fields.
func truncateTextPreservingJSON(content string, maxTokens int) (string, bool) {
	maxChars := maxTokens * TokenEstimateRatio
	trimmed := strings.TrimSpace(content)

	// Check if it looks like a JSON array (MultiContent format)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var parts []map[string]interface{}
		if err := json.Unmarshal([]byte(content), &parts); err == nil {
			// Successfully parsed as JSON array - truncate nested text fields
			modified := false
			for i := range parts {
				if textVal, ok := parts[i]["text"]; ok {
					if textStr, ok := textVal.(string); ok {
						textTokens := estimateTokenCount(textStr)
						if textTokens > maxTokens {
							textRunes := []rune(textStr)
							if len(textRunes) > maxChars {
								parts[i]["text"] = string(textRunes[:maxChars]) + "\n\n[Content truncated]"
								modified = true
							}
						}
					}
				}
			}
			if modified {
				if newContent, err := json.Marshal(parts); err == nil {
					return string(newContent), true
				}
			}
			return content, false
		}
	}

	// Check if it looks like a JSON object with nested content
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(content), &obj); err == nil {
			modified := false
			// Truncate common large text fields
			for _, key := range []string{"text", "content", "markdown", "raw_text", "body"} {
				if textVal, ok := obj[key]; ok {
					if textStr, ok := textVal.(string); ok {
						textTokens := estimateTokenCount(textStr)
						if textTokens > maxTokens {
							textRunes := []rune(textStr)
							if len(textRunes) > maxChars {
								obj[key] = string(textRunes[:maxChars]) + "\n\n[Content truncated]"
								modified = true
							}
						}
					}
				}
			}
			if modified {
				if newContent, err := json.Marshal(obj); err == nil {
					return string(newContent), true
				}
			}
			return content, false
		}
	}

	// Plain text - simple truncation
	runes := []rune(content)
	if len(runes) > maxChars {
		return string(runes[:maxChars]) + "\n\n[Content truncated - exceeded " + strconv.Itoa(maxTokens) + " token limit]", true
	}
	return content, false
}

// TruncateLargeToolContent reduces oversized tool results AND arguments.
// Now with MultiContent-aware JSON parsing to truncate nested text fields properly.
// Returns the modified messages and a list of truncation events for logging.
func TruncateLargeToolContent(messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, []TruncationEvent) {
	result := make([]openai.ChatCompletionMessage, len(messages))
	copy(result, messages)

	var events []TruncationEvent

	for i := range result {
		// Truncate tool results with MultiContent-aware parsing
		if result[i].Role == "tool" && result[i].Content != "" {
			originalTokens := estimateTokenCount(result[i].Content)
			if originalTokens > MaxToolResultTokens {
				truncatedContent, didTruncate := truncateTextPreservingJSON(result[i].Content, MaxToolResultTokens)
				if didTruncate {
					result[i].Content = truncatedContent

					event := TruncationEvent{
						MessageIndex:    i,
						ToolCallID:      result[i].ToolCallID,
						OriginalTokens:  originalTokens,
						TruncatedTokens: MaxToolResultTokens,
						TruncationType:  "tool_result",
					}
					events = append(events, event)
				}
			}
		}

		// Truncate tool call arguments (in assistant messages)
		if result[i].ToolCalls != nil {
			for j := range result[i].ToolCalls {
				tc := &result[i].ToolCalls[j]
				originalTokens := estimateTokenCount(tc.Function.Arguments)
				if originalTokens > MaxToolArgumentTokens {
					maxChars := MaxToolArgumentTokens * TokenEstimateRatio
					runes := []rune(tc.Function.Arguments)
					if len(runes) > maxChars {
						tc.Function.Arguments = string(runes[:maxChars]) + "...[truncated]"

						event := TruncationEvent{
							MessageIndex:    i,
							ToolName:        tc.Function.Name,
							ToolCallID:      tc.ID,
							OriginalTokens:  originalTokens,
							TruncatedTokens: MaxToolArgumentTokens,
							TruncationType:  "tool_argument",
						}
						events = append(events, event)
					}
				}
			}
		}
	}

	return result, events
}

// UserInputValidationError represents an error when user input exceeds token limits.
type UserInputValidationError struct {
	EstimatedTokens int
	MaxTokens       int
	Message         string
}

func (e *UserInputValidationError) Error() string {
	return e.Message
}

// ValidateUserInputSize checks if the last user message (current input) exceeds MaxUserContentTokens.
// Returns an error if the user input is too large, preventing the request from proceeding.
// This only validates the LAST user message (current input), not historical messages.
func ValidateUserInputSize(messages []openai.ChatCompletionMessage) error {
	if len(messages) == 0 {
		return nil
	}

	// Find the last user message (current user input)
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" {
			continue
		}

		// Check plain string content
		if messages[i].Content != "" && len(messages[i].MultiContent) == 0 {
			tokens := estimateTokenCount(messages[i].Content)
			if tokens > MaxUserContentTokens {
				return &UserInputValidationError{
					EstimatedTokens: tokens,
					MaxTokens:       MaxUserContentTokens,
					Message: fmt.Sprintf(
						"User input too large: estimated %d tokens exceeds maximum allowed %d tokens. Please reduce your message size.",
						tokens, MaxUserContentTokens,
					),
				}
			}
		}

		// Check MultiContent array
		if len(messages[i].MultiContent) > 0 {
			totalTextTokens := 0
			for _, part := range messages[i].MultiContent {
				if part.Type == openai.ChatMessagePartTypeText && part.Text != "" {
					totalTextTokens += estimateTokenCount(part.Text)
				}
			}
			if totalTextTokens > MaxUserContentTokens {
				return &UserInputValidationError{
					EstimatedTokens: totalTextTokens,
					MaxTokens:       MaxUserContentTokens,
					Message: fmt.Sprintf(
						"User input too large: estimated %d tokens exceeds maximum allowed %d tokens. Please reduce your message size.",
						totalTextTokens, MaxUserContentTokens,
					),
				}
			}
		}

		// Only check the last user message
		break
	}

	return nil
}

// TruncateLargeUserContent reduces oversized user message content.
// Handles both plain string content and MultiContent arrays.
// Returns the modified messages and a list of truncation events for logging.
func TruncateLargeUserContent(messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, []TruncationEvent) {
	result := make([]openai.ChatCompletionMessage, len(messages))
	copy(result, messages)

	var events []TruncationEvent

	for i := range result {
		if result[i].Role != "user" {
			continue
		}

		// Handle plain string content
		if result[i].Content != "" && len(result[i].MultiContent) == 0 {
			originalTokens := estimateTokenCount(result[i].Content)
			if originalTokens > MaxUserContentTokens {
				truncatedContent, didTruncate := truncateTextPreservingJSON(result[i].Content, MaxUserContentTokens)
				if didTruncate {
					result[i].Content = truncatedContent

					event := TruncationEvent{
						MessageIndex:    i,
						OriginalTokens:  originalTokens,
						TruncatedTokens: MaxUserContentTokens,
						TruncationType:  "user_content",
					}
					events = append(events, event)
				}
			}
		}

		// Handle MultiContent array
		if len(result[i].MultiContent) > 0 {
			for j := range result[i].MultiContent {
				part := &result[i].MultiContent[j]
				if part.Type == openai.ChatMessagePartTypeText && part.Text != "" {
					originalTokens := estimateTokenCount(part.Text)
					if originalTokens > MaxMultiContentTextTokens {
						maxChars := MaxMultiContentTextTokens * TokenEstimateRatio
						runes := []rune(part.Text)
						if len(runes) > maxChars {
							part.Text = string(runes[:maxChars]) + "\n\n[Content truncated - exceeded " + strconv.Itoa(MaxMultiContentTextTokens) + " token limit]"

							event := TruncationEvent{
								MessageIndex:    i,
								OriginalTokens:  originalTokens,
								TruncatedTokens: MaxMultiContentTextTokens,
								TruncationType:  "user_multicontent_text",
							}
							events = append(events, event)
						}
					}
				}
			}
		}
	}

	return result, events
}

// TrimMessagesResult contains the result of trimming messages.
type TrimMessagesResult struct {
	Messages        []openai.ChatCompletionMessage
	TrimmedCount    int
	EstimatedTokens int
}

// imageLocation tracks the position of an image in the messages array.
type imageLocation struct {
	messageIdx int
	partIdx    int
}

// LimitImagesInMessages limits the number of images in messages to prevent context overflow.
// It keeps the most recent images and removes older ones.
// - maxToolImages: maximum images across all tool messages (default: MaxToolImages = 10)
// - maxUserImages: maximum images across all user messages (default: MaxUserImages = 15)
// Images are removed from oldest messages first.
func LimitImagesInMessages(messages []openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
	// Create a deep copy to avoid modifying the original
	result := make([]openai.ChatCompletionMessage, len(messages))
	for i := range messages {
		result[i] = messages[i]
		// Deep copy MultiContent if present
		if len(messages[i].MultiContent) > 0 {
			result[i].MultiContent = make([]openai.ChatMessagePart, len(messages[i].MultiContent))
			copy(result[i].MultiContent, messages[i].MultiContent)
		}
	}

	// Count images by role, tracking message index and part index for each image
	var toolImages []imageLocation
	var userImages []imageLocation

	for i := len(result) - 1; i >= 0; i-- {
		msg := &result[i]
		if len(msg.MultiContent) == 0 {
			continue
		}

		for j := len(msg.MultiContent) - 1; j >= 0; j-- {
			part := msg.MultiContent[j]
			if part.Type == openai.ChatMessagePartTypeImageURL && part.ImageURL != nil && part.ImageURL.URL != "" {
				loc := imageLocation{messageIdx: i, partIdx: j}
				if msg.Role == "tool" {
					toolImages = append(toolImages, loc)
				} else if msg.Role == "user" {
					userImages = append(userImages, loc)
				}
			}
		}
	}

	// Reverse to get oldest first (we collected newest first by iterating backwards)
	reverseImageLocations(toolImages)
	reverseImageLocations(userImages)

	toolImagesRemoved := 0
	userImagesRemoved := 0

	// Remove excess tool images (oldest first)
	if len(toolImages) > MaxToolImages {
		excessCount := len(toolImages) - MaxToolImages
		for i := 0; i < excessCount; i++ {
			loc := toolImages[i]
			result[loc.messageIdx].MultiContent = removePartAtIndex(result[loc.messageIdx].MultiContent, loc.partIdx)
			// Adjust indices for subsequent removals in the same message
			for j := i + 1; j < len(toolImages); j++ {
				if toolImages[j].messageIdx == loc.messageIdx && toolImages[j].partIdx > loc.partIdx {
					toolImages[j].partIdx--
				}
			}
			toolImagesRemoved++
		}
	}

	// Remove excess user images (oldest first)
	if len(userImages) > MaxUserImages {
		excessCount := len(userImages) - MaxUserImages
		for i := 0; i < excessCount; i++ {
			loc := userImages[i]
			result[loc.messageIdx].MultiContent = removePartAtIndex(result[loc.messageIdx].MultiContent, loc.partIdx)
			// Adjust indices for subsequent removals in the same message
			for j := i + 1; j < len(userImages); j++ {
				if userImages[j].messageIdx == loc.messageIdx && userImages[j].partIdx > loc.partIdx {
					userImages[j].partIdx--
				}
			}
			userImagesRemoved++
		}
	}

	return result
}

// reverseImageLocations reverses a slice of imageLocation in place.
func reverseImageLocations(locs []imageLocation) {
	for i, j := 0, len(locs)-1; i < j; i, j = i+1, j-1 {
		locs[i], locs[j] = locs[j], locs[i]
	}
}

// removePartAtIndex removes the element at index i from the slice.
func removePartAtIndex(parts []openai.ChatMessagePart, i int) []openai.ChatMessagePart {
	if i < 0 || i >= len(parts) {
		return parts
	}
	return append(parts[:i], parts[i+1:]...)
}

// TrimMessagesToFitBudget trims messages using the provided TokenBudget.
// The budget must be validated before calling this function.
func TrimMessagesToFitBudget(messages []openai.ChatCompletionMessage, budget *TokenBudget) TrimMessagesResult {
	return trimMessagesInternal(messages, budget.AvailableForMessages)
}

// TrimMessagesToFitContext removes oldest tool results and assistant messages
// to fit within the context length limit.
// Priority order for removal (oldest first):
// 1. Tool result messages (role="tool")
// 2. Assistant messages with tool calls
// 3. Regular assistant messages
// Never removes: system prompts, user messages
//
// Deprecated: Use TrimMessagesToFitBudget with a validated TokenBudget instead.
func TrimMessagesToFitContext(messages []openai.ChatCompletionMessage, contextLength int) TrimMessagesResult {
	if contextLength <= 0 {
		contextLength = DefaultContextLength
	}

	// Apply safety margin
	maxTokens := int(float64(contextLength) * SafetyMarginRatio)
	return trimMessagesInternal(messages, maxTokens)
}

// trimMessagesInternal is the core trimming logic used by both public functions.
// Removes oldest conversation items first (any role except system) to fit within token budget.
func trimMessagesInternal(messages []openai.ChatCompletionMessage, maxTokens int) TrimMessagesResult {
	currentTokens := estimateMessagesTokenCount(messages)
	if currentTokens <= maxTokens {
		return TrimMessagesResult{
			Messages:        messages,
			TrimmedCount:    0,
			EstimatedTokens: currentTokens,
		}
	}

	// Create a working copy
	result := make([]openai.ChatCompletionMessage, len(messages))
	copy(result, messages)
	trimmedCount := 0

	// Build a token count cache for efficient removal
	messageTokens := make([]int, len(result))
	for i := range result {
		messageTokens[i] = estimateSingleMessageTokens(&result[i])
	}

	// Remove oldest items first (any role except system at index 0)
	// This approach removes conversation items chronologically from oldest to newest
	for currentTokens > maxTokens && len(result) > MinMessagesToKeep {
		// Find the oldest removable message (skip index 0 which is system prompt)
		removedIdx := -1
		for i := 1; i < len(result); i++ {
			// Skip system messages anywhere in the conversation
			if result[i].Role == "system" {
				continue
			}
			// Remove the oldest non-system message
			removedIdx = i
			break
		}

		if removedIdx == -1 {
			// No removable messages found
			break
		}

		removedTokens := messageTokens[removedIdx]
		currentTokens -= removedTokens

		result = append(result[:removedIdx], result[removedIdx+1:]...)
		messageTokens = append(messageTokens[:removedIdx], messageTokens[removedIdx+1:]...)
		trimmedCount++
	}

	return TrimMessagesResult{
		Messages:        result,
		TrimmedCount:    trimmedCount,
		EstimatedTokens: currentTokens,
	}
}

// estimateSingleMessageTokens calculates tokens for a single message.
func estimateSingleMessageTokens(msg *openai.ChatCompletionMessage) int {
	tokens := 10 // Overhead for role and structure
	tokens += estimateTokenCount(msg.Content)

	if len(msg.MultiContent) > 0 {
		tokens += estimateMultiContentTokens(msg.MultiContent)
	}

	// Count images in tool results
	if msg.Role == "tool" && msg.Content != "" {
		imageCount := countImagesInToolResult(msg.Content)
		tokens += imageCount * ImageTokensHighRes
	}

	if msg.ToolCalls != nil {
		for _, tc := range msg.ToolCalls {
			tokens += 20
			tokens += estimateTokenCount(tc.Function.Name)
			tokens += estimateTokenCount(tc.Function.Arguments)
		}
	}

	return tokens
}

// BuildTokenBudget creates a TokenBudget from request parameters.
func BuildTokenBudget(contextLength int, tools []openai.Tool, maxCompletionTokens int) *TokenBudget {
	return &TokenBudget{
		ContextLength:       contextLength,
		ToolsTokens:         EstimateToolsTokens(tools),
		MaxCompletionTokens: maxCompletionTokens,
		FixedOverhead:       FixedOverheadTokens,
	}
}
