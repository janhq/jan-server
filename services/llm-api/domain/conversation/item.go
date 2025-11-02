package conversation

import (
	"time"
)

// ===============================================
// Item Types and Enums
// ===============================================

type ItemType string

const (
	ItemTypeMessage         ItemType = "message"
	ItemTypeFunctionCall    ItemType = "function_call"
	ItemTypeFunctionCallOut ItemType = "function_call_output"
	ItemTypeReasoning       ItemType = "reasoning"
	ItemTypeFileSearch      ItemType = "file_search"
	ItemTypeWebSearch       ItemType = "web_search"
	ItemTypeCodeInterpreter ItemType = "code_interpreter"
	ItemTypeComputerUse     ItemType = "computer_use"
	ItemTypeCustomToolCall  ItemType = "custom_tool_call"
	ItemTypeMCPItem         ItemType = "mcp_item"
	ItemTypeImageGeneration ItemType = "image_generation"
)

type ItemRole string

const (
	ItemRoleSystem        ItemRole = "system"
	ItemRoleUser          ItemRole = "user"
	ItemRoleAssistant     ItemRole = "assistant"
	ItemRoleTool          ItemRole = "tool"
	ItemRoleDeveloper     ItemRole = "developer"
	ItemRoleCritic        ItemRole = "critic"
	ItemRoleDiscriminator ItemRole = "discriminator"
	ItemRoleUnknown       ItemRole = "unknown"
)

type ItemStatus string

const (
	ItemStatusIncomplete  ItemStatus = "incomplete"
	ItemStatusInProgress  ItemStatus = "in_progress"
	ItemStatusCompleted   ItemStatus = "completed"
	ItemStatusFailed      ItemStatus = "failed"
	ItemStatusCancelled   ItemStatus = "cancelled"
	ItemStatusSearching   ItemStatus = "searching"
	ItemStatusGenerating  ItemStatus = "generating"
	ItemStatusCalling     ItemStatus = "calling"
	ItemStatusStreaming   ItemStatus = "streaming"
	ItemStatusRateLimited ItemStatus = "rate_limited"
)

type ItemRating string

const (
	ItemRatingLike   ItemRating = "like"
	ItemRatingUnlike ItemRating = "unlike"
)

// ===============================================
// Item Structure
// ===============================================

type Item struct {
	ID                uint               `json:"-"`
	ConversationID    uint               `json:"-"`
	PublicID          string             `json:"id"`
	Object            string             `json:"object"`
	Branch            string             `json:"branch,omitempty"`
	SequenceNumber    int                `json:"sequence_number,omitempty"`
	Type              ItemType           `json:"type"`
	Role              *ItemRole          `json:"role,omitempty"`
	Content           []Content          `json:"content"`
	Status            *ItemStatus        `json:"status,omitempty"`
	IncompleteAt      *time.Time         `json:"incomplete_at,omitempty"`
	IncompleteDetails *IncompleteDetails `json:"incomplete_details,omitempty"`
	CompletedAt       *time.Time         `json:"completed_at,omitempty"`
	ResponseID        *uint              `json:"response_id,omitempty"`

	// User feedback/rating
	Rating        *ItemRating `json:"rating,omitempty"`
	RatedAt       *time.Time  `json:"rated_at,omitempty"`
	RatingComment *string     `json:"rating_comment,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// ===============================================
// Content Types
// ===============================================

type ContentType string

const (
	ContentTypeText       ContentType = "text"
	ContentTypeInputText  ContentType = "input_text"
	ContentTypeInputAudio ContentType = "input_audio"
	ContentTypeImage      ContentType = "image"
	ContentTypeAudio      ContentType = "audio"
	ContentTypeVideo      ContentType = "video"
)

type Content struct {
	Type ContentType `json:"type"`
	Text *string     `json:"text,omitempty"`

	// Image content
	ImageURL  *string `json:"image_url,omitempty"`
	ImageFile *string `json:"image_file,omitempty"`
	Detail    *string `json:"detail,omitempty"`

	// Audio content
	AudioURL  *string `json:"audio_url,omitempty"`
	AudioFile *string `json:"audio_file,omitempty"`
	Format    *string `json:"format,omitempty"`

	// Video content
	VideoURL  *string `json:"video_url,omitempty"`
	VideoFile *string `json:"video_file,omitempty"`

	// Transcript for audio/video
	Transcript *string `json:"transcript,omitempty"`
}

// ===============================================
// Incomplete Details
// ===============================================

type IncompleteReason string

const (
	IncompleteReasonInterrupted     IncompleteReason = "interrupted"
	IncompleteReasonMaxOutputTokens IncompleteReason = "max_output_tokens"
	IncompleteReasonContentFilter   IncompleteReason = "content_filter"
)

type IncompleteDetails struct {
	Reason *IncompleteReason `json:"reason,omitempty"`
}

// ===============================================
// Helper Functions
// ===============================================

// NewMessageItem creates a new message item
func NewMessageItem(publicID string, role ItemRole, content []Content) *Item {
	return &Item{
		PublicID:  publicID,
		Object:    "conversation.item",
		Type:      ItemTypeMessage,
		Role:      &role,
		Content:   content,
		CreatedAt: time.Now(),
	}
}

// NewTextContent creates a simple text content
func NewTextContent(text string) Content {
	return Content{
		Type: ContentTypeText,
		Text: &text,
	}
}

// ToItemStatusPtr returns a pointer to the given ItemStatus
func ToItemStatusPtr(s ItemStatus) *ItemStatus {
	return &s
}

// ToItemRolePtr returns a pointer to the given ItemRole
func ToItemRolePtr(r ItemRole) *ItemRole {
	return &r
}

// ToItemRatingPtr returns a pointer to the given ItemRating
func ToItemRatingPtr(r ItemRating) *ItemRating {
	return &r
}
