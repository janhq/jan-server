package conversation

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"jan-server/mono/apps/backend/internal/utils/idgen"
)

// ===============================================
// Conversation Validation
// ===============================================

// ConversationValidationConfig holds conversation-level validation rules
type ConversationValidationConfig struct {
	MaxTitleLength          int
	MaxMetadataKeys         int
	MaxMetadataKeyLength    int
	MaxMetadataValueLength  int
	MaxItemsPerConversation int // TODO: Implement validation for maximum items in a conversation
	MaxReferrerLength       int
}

// DefaultConversationValidationConfig returns OpenAI-aligned conversation validation rules
func DefaultConversationValidationConfig() *ConversationValidationConfig {
	return &ConversationValidationConfig{
		MaxTitleLength:          256,  // OpenAI default
		MaxMetadataKeys:         16,   // OpenAI default
		MaxMetadataKeyLength:    64,   // OpenAI default
		MaxMetadataValueLength:  512,  // OpenAI default
		MaxItemsPerConversation: 1000, // Reasonable conversation size limit
		MaxReferrerLength:       64,
	}
}

// ConversationValidator handles conversation-level validation
type ConversationValidator struct {
	config             *ConversationValidationConfig
	metadataKeyPattern *regexp.Regexp
}

// NewConversationValidator creates a validator for conversations
func NewConversationValidator(config *ConversationValidationConfig) *ConversationValidator {
	if config == nil {
		config = DefaultConversationValidationConfig()
	}

	return &ConversationValidator{
		config:             config,
		metadataKeyPattern: regexp.MustCompile(`^[a-zA-Z0-9_]+$`),
	}
}

// ValidateConversation performs full conversation validation
func (v *ConversationValidator) ValidateConversation(conv *Conversation) error {
	if conv == nil {
		return fmt.Errorf("conversation cannot be nil")
	}

	// Validate PublicID format
	if conv.PublicID != "" {
		if err := v.ValidateConversationID(conv.PublicID); err != nil {
			return fmt.Errorf("invalid conversation ID: %w", err)
		}
	}

	// Validate title
	if conv.Title != nil {
		if err := v.validateTitle(*conv.Title); err != nil {
			return fmt.Errorf("invalid title: %w", err)
		}
	}

	// Validate metadata
	if conv.Metadata != nil {
		if err := v.validateMetadata(conv.Metadata); err != nil {
			return fmt.Errorf("invalid metadata: %w", err)
		}
	}

	if conv.Referrer != nil {
		if err := v.validateReferrer(*conv.Referrer); err != nil {
			return fmt.Errorf("invalid referrer: %w", err)
		}
	}

	// Validate status
	if conv.Status != "" {
		if err := v.validateStatus(conv.Status); err != nil {
			return fmt.Errorf("invalid status: %w", err)
		}
	}

	return nil
}

// validateReferrer validates the referrer value (internal use only)
func (v *ConversationValidator) validateReferrer(referrer string) error {
	referrer = strings.TrimSpace(referrer)
	if referrer == "" {
		return fmt.Errorf("referrer cannot be empty")
	}

	if utf8.RuneCountInString(referrer) > v.config.MaxReferrerLength {
		return fmt.Errorf("referrer cannot exceed %d characters", v.config.MaxReferrerLength)
	}

	if strings.Contains(referrer, "\x00") {
		return fmt.Errorf("referrer cannot contain null bytes")
	}

	return nil
}

// ValidateConversationID validates conversation ID format
func (v *ConversationValidator) ValidateConversationID(id string) error {
	if id == "" {
		return fmt.Errorf("conversation ID cannot be empty")
	}

	// Must start with "conv_" prefix
	if !strings.HasPrefix(id, "conv_") {
		return fmt.Errorf("conversation ID must start with 'conv_' prefix")
	}

	// Use domain-specific ID validation
	if !idgen.ValidateIDFormat(id, "conv") {
		return fmt.Errorf("invalid conversation ID format")
	}

	return nil
}

// validateTitle validates conversation title (internal use only)
func (v *ConversationValidator) validateTitle(title string) error {
	// Title can be empty (optional field)
	if title == "" {
		return nil
	}

	// Check length (character count, not bytes)
	length := utf8.RuneCountInString(title)
	if length > v.config.MaxTitleLength {
		return fmt.Errorf("title cannot exceed %d characters (got %d)", v.config.MaxTitleLength, length)
	}

	// Trim and check for only whitespace
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return fmt.Errorf("title cannot be only whitespace")
	}

	// Check for null bytes (security)
	if strings.Contains(title, "\x00") {
		return fmt.Errorf("title cannot contain null bytes")
	}

	return nil
}

// validateMetadata validates conversation metadata (internal use only)
func (v *ConversationValidator) validateMetadata(metadata map[string]string) error {
	if metadata == nil {
		return nil
	}

	// Check number of keys
	if len(metadata) > v.config.MaxMetadataKeys {
		return fmt.Errorf("metadata cannot have more than %d keys (got %d)", v.config.MaxMetadataKeys, len(metadata))
	}

	// Validate each key-value pair
	for key, value := range metadata {
		if err := v.validateMetadataKey(key); err != nil {
			return fmt.Errorf("invalid metadata key '%s': %w", key, err)
		}

		if err := v.validateMetadataValue(key, value); err != nil {
			return fmt.Errorf("invalid metadata value for key '%s': %w", key, err)
		}
	}

	return nil
}

// validateStatus validates conversation status (internal use only)
func (v *ConversationValidator) validateStatus(status ConversationStatus) error {
	switch status {
	case ConversationStatusActive, ConversationStatusArchived, ConversationStatusDeleted:
		return nil
	default:
		return fmt.Errorf("invalid conversation status: %s (must be active, archived, or deleted)", status)
	}
}

// Private helper methods

func (v *ConversationValidator) validateMetadataKey(key string) error {
	if key == "" {
		return fmt.Errorf("metadata key cannot be empty")
	}

	length := len(key) // OpenAI uses byte length for keys
	if length > v.config.MaxMetadataKeyLength {
		return fmt.Errorf("metadata key cannot exceed %d bytes (got %d)", v.config.MaxMetadataKeyLength, length)
	}

	// OpenAI requires alphanumeric + underscore only
	if !v.metadataKeyPattern.MatchString(key) {
		return fmt.Errorf("metadata key must contain only alphanumeric characters and underscores")
	}

	// Cannot start with underscore (reserved for system metadata)
	if strings.HasPrefix(key, "_") {
		return fmt.Errorf("metadata key cannot start with underscore (reserved for system use)")
	}

	return nil
}

func (v *ConversationValidator) validateMetadataValue(key, value string) error {
	length := utf8.RuneCountInString(value)
	if length > v.config.MaxMetadataValueLength {
		return fmt.Errorf("metadata value cannot exceed %d characters (got %d)", v.config.MaxMetadataValueLength, length)
	}

	// Check for null bytes (security)
	if strings.Contains(value, "\x00") {
		return fmt.Errorf("metadata value cannot contain null bytes")
	}

	return nil
}
