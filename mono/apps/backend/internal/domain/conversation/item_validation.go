package conversation

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"jan-server/mono/apps/backend/internal/utils/idgen"
)

// ===============================================
// Item Validation
// ===============================================

// ItemValidationConfig holds item-level validation rules
type ItemValidationConfig struct {
	MaxContentBlocks     int
	MaxTextContentLength int
	MaxCodeLength        int
	MaxReasoningLength   int
	MaxThinkingLength    int
	MaxAudioSize         int64
	MaxImageSize         int64
	MaxFileSize          int64
	MaxToolCalls         int
	MaxAnnotations       int
	MaxItemsPerBatch     int
}

// DefaultItemValidationConfig returns OpenAI-aligned item validation rules
func DefaultItemValidationConfig() *ItemValidationConfig {
	return &ItemValidationConfig{
		MaxContentBlocks:     100,               // OpenAI supports multiple content blocks
		MaxTextContentLength: 100000,            // ~100K chars for text content
		MaxCodeLength:        50000,             // Code blocks up to 50K chars
		MaxReasoningLength:   100000,            // Reasoning content up to 100K chars
		MaxThinkingLength:    50000,             // Thinking content up to 50K chars
		MaxAudioSize:         25 * 1024 * 1024,  // 25MB for audio
		MaxImageSize:         20 * 1024 * 1024,  // 20MB for images
		MaxFileSize:          512 * 1024 * 1024, // 512MB for files
		MaxToolCalls:         16,                // Max tool calls per message
		MaxAnnotations:       100,               // Max annotations per content block
		MaxItemsPerBatch:     100,               // Max items per batch operation
	}
}

// ItemValidator handles item-level validation
type ItemValidator struct {
	config     *ItemValidationConfig
	itemIDRx   *regexp.Regexp
	urlPattern *regexp.Regexp
}

// NewItemValidator creates a validator for items
func NewItemValidator(config *ItemValidationConfig) *ItemValidator {
	if config == nil {
		config = DefaultItemValidationConfig()
	}

	return &ItemValidator{
		config:     config,
		itemIDRx:   regexp.MustCompile(`^msg_[a-zA-Z0-9]{16,}$`),
		urlPattern: regexp.MustCompile(`^https?://|^data:|^file://`),
	}
}

// ValidateItem performs full item validation
func (v *ItemValidator) ValidateItem(item Item) error {
	// Validate PublicID
	if item.PublicID != "" {
		if err := v.ValidateItemID(item.PublicID); err != nil {
			return fmt.Errorf("invalid item ID: %w", err)
		}
	}

	// Validate type
	if err := v.ValidateItemType(item.Type); err != nil {
		return fmt.Errorf("invalid item type: %w", err)
	}

	// Validate role if present
	if item.Role != nil {
		if err := v.ValidateItemRole(*item.Role); err != nil {
			return fmt.Errorf("invalid item role: %w", err)
		}
	}

	// Validate status if present
	if item.Status != nil {
		if err := v.ValidateItemStatus(*item.Status); err != nil {
			return fmt.Errorf("invalid item status: %w", err)
		}
	}

	// Validate content array
	if len(item.Content) > 0 {
		if err := v.ValidateContentArray(item.Content); err != nil {
			return fmt.Errorf("invalid content: %w", err)
		}
	}

	return nil
}

// ValidateBaseItem performs validation on BaseItem
func (v *ItemValidator) ValidateBaseItem(item BaseItem) error {
	// Validate PublicID
	if item.PublicID != "" {
		if err := v.ValidateItemID(item.PublicID); err != nil {
			return fmt.Errorf("invalid item ID: %w", err)
		}
	}

	// Validate type
	if err := v.ValidateItemType(item.Type); err != nil {
		return fmt.Errorf("invalid item type: %w", err)
	}

	// Validate status if present
	if item.Status != nil {
		if err := v.ValidateItemStatus(*item.Status); err != nil {
			return fmt.Errorf("invalid item status: %w", err)
		}
	}

	return nil
}

// ValidateMessageItem validates a MessageItem
func (v *ItemValidator) ValidateMessageItem(item *MessageItem) error {
	if item == nil {
		return fmt.Errorf("message item cannot be nil")
	}

	// Validate base item
	if err := v.ValidateBaseItem(item.BaseItem); err != nil {
		return err
	}

	// Validate role
	if err := v.ValidateItemRole(item.Role); err != nil {
		return fmt.Errorf("invalid item role: %w", err)
	}

	// Message must have content
	if len(item.Content) == 0 {
		return fmt.Errorf("message item must have at least one content block")
	}

	// Validate content array
	if err := v.ValidateContentArray(item.Content); err != nil {
		return fmt.Errorf("invalid content: %w", err)
	}

	return nil
}

// ValidateItemID validates item ID format
func (v *ItemValidator) ValidateItemID(id string) error {
	if id == "" {
		return fmt.Errorf("item ID cannot be empty")
	}

	// Must start with "msg_" prefix
	if !strings.HasPrefix(id, "msg_") {
		return fmt.Errorf("item ID must start with 'msg_' prefix")
	}

	// Use domain-specific ID validation
	if !idgen.ValidateIDFormat(id, "msg") {
		return fmt.Errorf("invalid item ID format")
	}

	return nil
}

// ValidateItemType validates item type
func (v *ItemValidator) ValidateItemType(itemType ItemType) error {
	if !ValidateItemType(string(itemType)) {
		return fmt.Errorf("invalid item type: %s", itemType)
	}
	return nil
}

// ValidateItemRole validates item role
func (v *ItemValidator) ValidateItemRole(role ItemRole) error {
	if !ValidateItemRole(string(role)) {
		return fmt.Errorf("invalid item role: %s", role)
	}
	return nil
}

// ValidateItemStatus validates item status
func (v *ItemValidator) ValidateItemStatus(status ItemStatus) error {
	if !ValidateItemStatus(string(status)) {
		return fmt.Errorf("invalid item status: %s", status)
	}
	return nil
}

// ValidateContentArray validates an array of content blocks
func (v *ItemValidator) ValidateContentArray(content []Content) error {
	if len(content) == 0 {
		return nil // Empty content is allowed for some item types
	}

	if len(content) > v.config.MaxContentBlocks {
		return fmt.Errorf("content array cannot exceed %d blocks (got %d)", v.config.MaxContentBlocks, len(content))
	}

	for i, c := range content {
		if err := v.ValidateContent(c); err != nil {
			return fmt.Errorf("invalid content block at index %d: %w", i, err)
		}
	}

	return nil
}

// ValidateContent validates a single content block
func (v *ItemValidator) ValidateContent(content Content) error {
	if content.Type == "" {
		return fmt.Errorf("content type cannot be empty")
	}

	// Validate based on content type
	switch content.Type {
	case "text":
		if content.TextString != nil {
			return v.validateSimpleText(*content.TextString, "text")
		}
		return fmt.Errorf("text content type requires text field")

	case "input_text":
		if content.TextString != nil {
			return v.validateSimpleText(*content.TextString, "input_text")
		}
		return fmt.Errorf("input_text content type requires text field")

	case "output_text":
		if content.OutputText != nil {
			return v.validateOutputText(content.OutputText)
		}
		return fmt.Errorf("output_text content type requires output_text field")

	case "image":
		if content.Image != nil {
			return v.validateImageContent(content.Image)
		}
		return fmt.Errorf("image content type requires image field")

	case "file":
		if content.File != nil {
			return v.validateFileContent(content.File)
		}
		return fmt.Errorf("file content type requires file field")

	case "audio":
		if content.Audio != nil {
			return v.validateAudioContent(content.Audio)
		}
		return fmt.Errorf("audio content type requires audio field")

	case "input_audio":
		if content.InputAudio != nil {
			return v.validateInputAudioContent(content.InputAudio)
		}
		return fmt.Errorf("input_audio content type requires input_audio field")

	case "refusal":
		if content.Refusal != nil {
			return v.validateSimpleText(*content.Refusal, "refusal")
		}
		return fmt.Errorf("refusal content type requires refusal field")

	case "thinking":
		if content.Thinking != nil {
			return v.validateThinkingContent(*content.Thinking)
		}
		return fmt.Errorf("thinking content type requires thinking field")

	case "code":
		if content.Code != nil {
			return v.validateCodeContent(content.Code)
		}
		return fmt.Errorf("code content type requires code field")

	case "computer_screenshot":
		if content.ComputerScreenshot != nil {
			return v.validateScreenshotContent(content.ComputerScreenshot)
		}
		return fmt.Errorf("computer_screenshot content type requires computer_screenshot field")

	case "computer_action":
		if content.ComputerAction != nil {
			return v.validateComputerAction(content.ComputerAction)
		}
		return fmt.Errorf("computer_action content type requires computer_action field")

	case "reasoning_text":
		if content.TextString != nil {
			return v.validateSimpleText(*content.TextString, "reasoning_text")
		}
		return fmt.Errorf("reasoning_text content type requires text field")

	case "tool_result":
		if content.TextString != nil {
			return v.validateSimpleText(*content.TextString, "tool_result")
		}
		return fmt.Errorf("tool_result content type requires text field")

	case "tool_calls":
		if len(content.ToolCalls) == 0 {
			return fmt.Errorf("tool_calls content type requires tool_calls array")
		}
		// Tool calls validation is done elsewhere
		return nil

	case "function_call":
		if content.FunctionCall != nil {
			// Function call validation is done elsewhere
			return nil
		}
		return fmt.Errorf("function_call content type requires function_call field")

	case "function_call_output":
		if content.FunctionCallOut != nil {
			// Function call output validation is done elsewhere
			return nil
		}
		return fmt.Errorf("function_call_output content type requires function_call_output field")

	default:
		return fmt.Errorf("unsupported content type: %s", content.Type)
	}
}

// ValidateBatchSize ensures batch operations are within limits
func (v *ItemValidator) ValidateBatchSize(itemCount int) error {
	if itemCount == 0 {
		return fmt.Errorf("batch cannot be empty")
	}

	if itemCount > v.config.MaxItemsPerBatch {
		return fmt.Errorf("cannot process more than %d items in a single batch (got %d)", v.config.MaxItemsPerBatch, itemCount)
	}

	return nil
}

// ===============================================
// Private Validation Methods
// ===============================================

func (v *ItemValidator) validateTextContent(text *Text) error {
	if text == nil {
		return fmt.Errorf("text content cannot be nil")
	}

	if err := v.validateSimpleText(text.Text, "text"); err != nil {
		return err
	}

	// Validate annotations
	if len(text.Annotations) > v.config.MaxAnnotations {
		return fmt.Errorf("text cannot have more than %d annotations (got %d)", v.config.MaxAnnotations, len(text.Annotations))
	}

	for i, ann := range text.Annotations {
		if err := v.validateAnnotation(ann); err != nil {
			return fmt.Errorf("invalid annotation at index %d: %w", i, err)
		}
	}

	return nil
}

func (v *ItemValidator) validateOutputText(output *OutputText) error {
	if output == nil {
		return fmt.Errorf("output text content cannot be nil")
	}

	if err := v.validateSimpleText(output.Text, "output_text"); err != nil {
		return err
	}

	// Validate annotations (required field for OpenAI)
	if len(output.Annotations) > v.config.MaxAnnotations {
		return fmt.Errorf("output text cannot have more than %d annotations (got %d)", v.config.MaxAnnotations, len(output.Annotations))
	}

	for i, ann := range output.Annotations {
		if err := v.validateAnnotation(ann); err != nil {
			return fmt.Errorf("invalid annotation at index %d: %w", i, err)
		}
	}

	return nil
}

func (v *ItemValidator) validateSimpleText(text, fieldName string) error {
	if text == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}

	length := utf8.RuneCountInString(text)
	if length > v.config.MaxTextContentLength {
		return fmt.Errorf("%s cannot exceed %d characters (got %d)", fieldName, v.config.MaxTextContentLength, length)
	}

	// Check for null bytes (security)
	if strings.Contains(text, "\x00") {
		return fmt.Errorf("%s cannot contain null bytes", fieldName)
	}

	return nil
}

func (v *ItemValidator) validateThinkingContent(thinking string) error {
	if thinking == "" {
		return fmt.Errorf("thinking content cannot be empty")
	}

	length := utf8.RuneCountInString(thinking)
	if length > v.config.MaxThinkingLength {
		return fmt.Errorf("thinking content cannot exceed %d characters (got %d)", v.config.MaxThinkingLength, length)
	}

	return nil
}

func (v *ItemValidator) validateImageContent(image *ImageContent) error {
	if image == nil {
		return fmt.Errorf("image content cannot be nil")
	}

	// Must have either URL or FileID
	if image.URL == "" && image.FileID == "" {
		return fmt.Errorf("image content must have either url or file_id")
	}

	// Cannot have both
	if image.URL != "" && image.FileID != "" {
		return fmt.Errorf("image content cannot have both url and file_id")
	}

	// Validate URL format if present
	if image.URL != "" {
		if !v.urlPattern.MatchString(image.URL) {
			return fmt.Errorf("invalid image URL format (must start with http://, https://, data:, or file://)")
		}
	}

	// Validate detail level if present
	if image.Detail != "" {
		switch image.Detail {
		case "low", "high", "auto":
			// Valid
		default:
			return fmt.Errorf("invalid image detail level: %s (must be low, high, or auto)", image.Detail)
		}
	}

	return nil
}

func (v *ItemValidator) validateFileContent(file *FileContent) error {
	if file == nil {
		return fmt.Errorf("file content cannot be nil")
	}

	if file.FileID == "" {
		return fmt.Errorf("file content must have a file_id")
	}

	if file.Size < 0 {
		return fmt.Errorf("file size cannot be negative")
	}

	if file.Size > v.config.MaxFileSize {
		return fmt.Errorf("file size cannot exceed %d bytes (got %d)", v.config.MaxFileSize, file.Size)
	}

	return nil
}

func (v *ItemValidator) validateAudioContent(audio *AudioContent) error {
	if audio == nil {
		return fmt.Errorf("audio content cannot be nil")
	}

	// Must have ID
	if audio.ID == "" {
		return fmt.Errorf("audio content must have an id")
	}

	// Validate format if present
	if audio.Format != nil {
		validFormats := []string{"mp3", "wav", "opus", "flac", "pcm16"}
		isValid := false
		for _, fmt := range validFormats {
			if *audio.Format == fmt {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid audio format: %s", *audio.Format)
		}
	}

	return nil
}

func (v *ItemValidator) validateInputAudioContent(audio *InputAudio) error {
	if audio == nil {
		return fmt.Errorf("input audio content cannot be nil")
	}

	if audio.Data == "" {
		return fmt.Errorf("input audio must have data")
	}

	if audio.Format == "" {
		return fmt.Errorf("input audio must have format")
	}

	// Validate format
	validFormats := []string{"mp3", "wav", "opus", "flac", "pcm16"}
	isValid := false
	for _, fmt := range validFormats {
		if audio.Format == fmt {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid audio format: %s", audio.Format)
	}

	return nil
}

func (v *ItemValidator) validateCodeContent(code *CodeContent) error {
	if code == nil {
		return fmt.Errorf("code content cannot be nil")
	}

	if code.Language == "" {
		return fmt.Errorf("code content must have a language")
	}

	if code.Code == "" {
		return fmt.Errorf("code content must have code")
	}

	length := utf8.RuneCountInString(code.Code)
	if length > v.config.MaxCodeLength {
		return fmt.Errorf("code content cannot exceed %d characters (got %d)", v.config.MaxCodeLength, length)
	}

	return nil
}

func (v *ItemValidator) validateScreenshotContent(screenshot *ScreenshotContent) error {
	if screenshot == nil {
		return fmt.Errorf("screenshot content cannot be nil")
	}

	// Must have either ImageURL or ImageData
	if screenshot.ImageURL == "" && screenshot.ImageData == nil {
		return fmt.Errorf("screenshot must have either image_url or image_data")
	}

	if screenshot.Width <= 0 {
		return fmt.Errorf("screenshot width must be positive")
	}

	if screenshot.Height <= 0 {
		return fmt.Errorf("screenshot height must be positive")
	}

	return nil
}

func (v *ItemValidator) validateComputerAction(action *ComputerAction) error {
	if action == nil {
		return fmt.Errorf("computer action cannot be nil")
	}

	if action.Action == "" {
		return fmt.Errorf("computer action must have an action type")
	}

	// Validate action type
	validActions := []string{"click", "type", "key", "scroll", "move", "drag", "screenshot"}
	isValid := false
	for _, a := range validActions {
		if action.Action == a {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid action type: %s", action.Action)
	}

	// Validate required fields based on action type
	switch action.Action {
	case "click", "move", "drag":
		if action.Coordinates == nil {
			return fmt.Errorf("action '%s' requires coordinates", action.Action)
		}
	case "type":
		if action.Text == nil || *action.Text == "" {
			return fmt.Errorf("action 'type' requires text")
		}
	case "key":
		if action.Key == nil || *action.Key == "" {
			return fmt.Errorf("action 'key' requires key")
		}
	case "scroll":
		if action.ScrollDelta == nil {
			return fmt.Errorf("action 'scroll' requires scroll_delta")
		}
	}

	return nil
}

func (v *ItemValidator) validateAnnotation(annotation Annotation) error {
	if annotation.Type == "" {
		return fmt.Errorf("annotation must have a type")
	}

	// Validate annotation type
	validTypes := []string{"file_citation", "url_citation", "file_path", "quote", "highlight"}
	isValid := false
	for _, t := range validTypes {
		if annotation.Type == t {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid annotation type: %s", annotation.Type)
	}

	// Validate required fields based on type
	switch annotation.Type {
	case "file_citation":
		if annotation.FileID == "" {
			return fmt.Errorf("file_citation annotation requires file_id")
		}
	case "url_citation":
		if annotation.URL == "" {
			return fmt.Errorf("url_citation annotation requires url")
		}
		if !v.urlPattern.MatchString(annotation.URL) {
			return fmt.Errorf("invalid url format in annotation")
		}
	}

	return nil
}
