package conversation

import (
	"encoding/json"
	"time"
)

// ===============================================
// Conversation Types
// ===============================================

type ConversationStatus string

const (
	ConversationStatusActive   ConversationStatus = "active"
	ConversationStatusArchived ConversationStatus = "archived"
	ConversationStatusDeleted  ConversationStatus = "deleted"
)

// ConversationBranch represents a specific flow/path in a conversation
// Used to support editing items while maintaining conversation history
const (
	BranchMain = "MAIN" // Default main conversation flow
)

// ===============================================
// Conversation Structure
// ===============================================

// Conversation represents a logical chat thread for the Responses API.
type Conversation struct {
	ID              uint                      `json:"-"`
	PublicID        string                    `json:"id"`     // OpenAI-compatible string ID like "conv_abc123"
	Object          string                    `json:"object"` // Always "conversation" for OpenAI compatibility
	Title           *string                   `json:"title,omitempty"`
	UserID          string                    `json:"-"` // String for compatibility with response-api service
	UserIDInt       uint                      `json:"-"` // Internal uint ID when needed
	ProjectID       *uint                     `json:"-"` // Optional project grouping
	ProjectPublicID *string                   `json:"-"` // Public ID of the project
	Status          ConversationStatus        `json:"status"`
	Items           []Item                    `json:"items,omitempty"`           // Legacy: items without branch (defaults to MAIN)
	Branches        map[string][]Item         `json:"branches,omitempty"`        // Branched items organized by branch name
	ActiveBranch    string                    `json:"active_branch,omitempty"`   // Currently active branch (default: "MAIN")
	BranchMetadata  map[string]BranchMetadata `json:"branch_metadata,omitempty"` // Metadata about each branch
	Metadata        map[string]string         `json:"metadata,omitempty"`
	Referrer        *string                   `json:"referrer,omitempty"`
	IsPrivate       bool                      `json:"is_private"`

	// Project instruction inheritance
	InstructionVersion           int     `json:"instruction_version"`                      // Version of project instruction when conversation was created
	EffectiveInstructionSnapshot *string `json:"effective_instruction_snapshot,omitempty"` // Snapshot of merged instruction for reproducibility

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BranchMetadata contains information about a conversation branch
type BranchMetadata struct {
	Name             string     `json:"name"`                          // Branch identifier (MAIN, EDIT_1, etc.)
	Description      *string    `json:"description,omitempty"`         // Optional description of this branch
	ParentBranch     *string    `json:"parent_branch,omitempty"`       // Branch this was forked from
	ForkedAt         *time.Time `json:"forked_at,omitempty"`           // When this branch was created
	ForkedFromItemID *string    `json:"forked_from_item_id,omitempty"` // Item ID where fork occurred
	ItemCount        int        `json:"item_count"`                    // Number of items in this branch
	CreatedAt        time.Time  `json:"created_at"`                    // Branch creation time
	UpdatedAt        time.Time  `json:"updated_at"`                    // Last update time
}

// ===============================================
// Item Types
// ===============================================

// ItemType defines the type of conversation item
type ItemType string

const (
	ItemTypeMessage          ItemType = "message"
	ItemTypeFunctionCall     ItemType = "function_call"
	ItemTypeFunctionCallOut  ItemType = "function_call_output"
	ItemTypeReasoning        ItemType = "reasoning"
	ItemTypeFileSearch       ItemType = "file_search_call"
	ItemTypeWebSearch        ItemType = "web_search_call"
	ItemTypeCodeInterpreter  ItemType = "code_interpreter_call"
	ItemTypeComputerUse      ItemType = "computer_use_call"
	ItemTypeCustomToolCall   ItemType = "custom_tool_call"
	ItemTypeMCPItem          ItemType = "mcp_item"
	ItemTypeImageGeneration  ItemType = "image_generation_call"
	ItemTypeLocalShellCall   ItemType = "local_shell_call"
	ItemTypeLocalShellOutput ItemType = "local_shell_call_output"
	ItemTypeMCPListTools     ItemType = "mcp_list_tools"
	ItemTypeMCPApprovalReq   ItemType = "mcp_approval_request"
	ItemTypeMCPApprovalResp  ItemType = "mcp_approval_response"
	ItemTypeMCPCall          ItemType = "mcp_call"
	ItemTypeComputerCall     ItemType = "computer_call"
	ItemTypeComputerCallOut  ItemType = "computer_call_output"
	ItemTypeApplyPatchCall   ItemType = "apply_patch_call"
	ItemTypeApplyPatchOutput ItemType = "apply_patch_call_output"
	ItemTypeShellCall        ItemType = "shell_call"
	ItemTypeShellOutput      ItemType = "shell_call_output"
)

// ItemRole indicates who authored the conversation item.
type ItemRole string

const (
	RoleSystem        ItemRole = "system"
	RoleUser          ItemRole = "user"
	RoleAssistant     ItemRole = "assistant"
	RoleTool          ItemRole = "tool"
	RoleDeveloper     ItemRole = "developer"
	RoleCritic        ItemRole = "critic"
	RoleDiscriminator ItemRole = "discriminator"
	RoleUnknown       ItemRole = "unknown"
)

// ItemStatus tracks whether the item is finalised.
type ItemStatus string

const (
	ItemStatusIncomplete ItemStatus = "incomplete"
	ItemStatusInProgress ItemStatus = "in_progress"
	ItemStatusCompleted  ItemStatus = "completed"
	ItemStatusFailed     ItemStatus = "failed"
	ItemStatusCancelled  ItemStatus = "cancelled"
	ItemStatusCalling    ItemStatus = "calling"
	ItemStatusSearching  ItemStatus = "searching"
	ItemStatusExecuting  ItemStatus = "executing"
	ItemStatusGenerating ItemStatus = "generating"
	ItemStatusPending    ItemStatus = "pending"
)

// ItemRating represents like/unlike feedback on an item
type ItemRating string

const (
	ItemRatingLike   ItemRating = "like"
	ItemRatingUnlike ItemRating = "unlike"
)

// Validate checks if the rating is valid
func (r ItemRating) Validate() bool {
	return r == ItemRatingLike || r == ItemRatingUnlike
}

// ===============================================
// Item Structure
// ===============================================

// Item contains individual conversation message state.
// This struct supports both the new OpenAI-compatible format (Content []Content)
// and legacy format (LegacyContent, Sequence) for backward compatibility.
type Item struct {
	ID                uint                   `json:"-"`
	ConversationID    uint                   `json:"-"`
	PublicID          string                 `json:"id"`
	Object            string                 `json:"object"`                    // Always "conversation.item" for OpenAI compatibility
	Branch            string                 `json:"branch,omitempty"`          // Branch identifier (MAIN, EDIT_1, etc.)
	SequenceNumber    int                    `json:"sequence_number,omitempty"` // Order within branch (new field)
	Sequence          int                    `json:"-"`                         // Legacy: Order within conversation (deprecated, use SequenceNumber)
	Type              ItemType               `json:"type"`
	Role              ItemRole               `json:"role,omitempty"`    // Direct value for legacy compatibility
	RolePtr           *ItemRole              `json:"-"`                 // Pointer for new code
	Content           []Content              `json:"content,omitempty"` // New: typed content array
	LegacyContent     map[string]interface{} `json:"-"`                 // Legacy: untyped content map
	Status            ItemStatus             `json:"status,omitempty"`  // Direct value for legacy compatibility
	StatusPtr         *ItemStatus            `json:"-"`                 // Pointer for new code
	IncompleteAt      *time.Time             `json:"incomplete_at,omitempty"`
	IncompleteDetails *IncompleteDetails     `json:"incomplete_details,omitempty"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
	ResponseID        *uint                  `json:"-"`

	// User feedback/rating
	Rating        *ItemRating `json:"rating,omitempty"`
	RatedAt       *time.Time  `json:"rated_at,omitempty"`
	RatingComment *string     `json:"rating_comment,omitempty"`

	// OpenAI-compatible fields for specific item types
	CallID                   *string                `json:"call_id,omitempty"`
	Name                     *string                `json:"name,omitempty"`
	ServerLabel              *string                `json:"server_label,omitempty"`
	ApprovalRequestID        *string                `json:"approval_request_id,omitempty"`
	Arguments                *string                `json:"arguments,omitempty"`
	Output                   *string                `json:"output,omitempty"`
	Error                    *string                `json:"error,omitempty"`
	Action                   map[string]interface{} `json:"action,omitempty"`
	Tools                    []McpTool              `json:"tools,omitempty"`
	PendingSafetyChecks      []SafetyCheck          `json:"pending_safety_checks,omitempty"`
	AcknowledgedSafetyChecks []SafetyCheck          `json:"acknowledged_safety_checks,omitempty"`
	Approve                  *bool                  `json:"approve,omitempty"`
	Reason                   *string                `json:"reason,omitempty"`
	Commands                 []string               `json:"commands,omitempty"`
	MaxOutputLength          *int64                 `json:"max_output_length,omitempty"`
	ShellOutputs             []ShellOutput          `json:"shell_outputs,omitempty"`
	Operation                map[string]interface{} `json:"operation,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// GetRole returns the item role, checking both legacy and new fields
func (i *Item) GetRole() ItemRole {
	if i.RolePtr != nil {
		return *i.RolePtr
	}
	return i.Role
}

// GetStatus returns the item status, checking both legacy and new fields
func (i *Item) GetStatus() ItemStatus {
	if i.StatusPtr != nil {
		return *i.StatusPtr
	}
	return i.Status
}

// GetSequence returns the sequence number, checking both fields
func (i *Item) GetSequence() int {
	if i.SequenceNumber > 0 {
		return i.SequenceNumber
	}
	return i.Sequence
}

// GetContent returns content as a map for legacy compatibility
func (i *Item) GetContent() map[string]interface{} {
	if i.LegacyContent != nil {
		return i.LegacyContent
	}
	// Convert Content array to map
	if len(i.Content) > 0 {
		result := make(map[string]interface{})
		if len(i.Content) == 1 {
			c := i.Content[0]
			result["type"] = c.Type
			if c.TextString != nil {
				result["text"] = *c.TextString
			}
			if c.OutputText != nil {
				result["text"] = c.OutputText.Text
			}
		} else {
			items := make([]map[string]interface{}, len(i.Content))
			for j, c := range i.Content {
				items[j] = map[string]interface{}{"type": c.Type}
				if c.TextString != nil {
					items[j]["text"] = *c.TextString
				}
			}
			result["type"] = "list"
			result["items"] = items
		}
		return result
	}
	return nil
}

// IncompleteDetails provides details about why an item is incomplete
type IncompleteDetails struct {
	Reason string  `json:"reason"`
	Error  *string `json:"error,omitempty"`
}

// McpTool represents a tool available on an MCP server
type McpTool struct {
	Name        string  `json:"name"`
	InputSchema any     `json:"input_schema"`
	Description *string `json:"description,omitempty"`
	Annotations any     `json:"annotations,omitempty"`
}

// SafetyCheck represents a safety check for computer use
type SafetyCheck struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

// ShellOutput represents output from a shell command
type ShellOutput struct {
	Type  string `json:"type"` // stdout, stderr, exit_code
	Value string `json:"value"`
}

// ===============================================
// Content Structures
// ===============================================

type Content struct {
	Type               string             `json:"type"`
	FinishReason       *string            `json:"finish_reason,omitempty"`
	Text               *Text              `json:"-"`
	TextString         *string            `json:"-"`
	OutputText         *OutputText        `json:"output_text,omitempty"`
	Refusal            *string            `json:"refusal,omitempty"`
	SummaryText        *string            `json:"summary_text,omitempty"`
	Thinking           *string            `json:"thinking,omitempty"`
	Image              *ImageContent      `json:"image,omitempty"`
	File               *FileContent       `json:"file,omitempty"`
	Audio              *AudioContent      `json:"audio,omitempty"`
	InputAudio         *InputAudio        `json:"input_audio,omitempty"`
	Code               *CodeContent       `json:"code,omitempty"`
	ComputerScreenshot *ScreenshotContent `json:"computer_screenshot,omitempty"`
	ComputerAction     *ComputerAction    `json:"computer_action,omitempty"`
	FunctionCall       *FunctionCall      `json:"function_call,omitempty"`
	FunctionCallOut    *FunctionCallOut   `json:"function_call_output,omitempty"`
	ToolCalls          []ToolCall         `json:"tool_calls,omitempty"`
	ToolCallID         *string            `json:"tool_call_id,omitempty"`
	Reasoning          *string            `json:"reasoning_text,omitempty"`
}

// Text content - matches OpenAI's text content format
type Text struct {
	Text        string       `json:"text"`
	Annotations []Annotation `json:"annotations,omitempty"`
}

type OutputText struct {
	Text        string       `json:"text"`
	Annotations []Annotation `json:"annotations"`
	LogProbs    []LogProb    `json:"logprobs,omitempty"`
}

// ImageContent for multimodal support
type ImageContent struct {
	URL    string `json:"url,omitempty"`
	FileID string `json:"file_id,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// FileContent for attachments
type FileContent struct {
	FileID   string `json:"file_id"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

// AudioContent for speech output
type AudioContent struct {
	ID         string  `json:"id,omitempty"`
	Transcript *string `json:"transcript,omitempty"`
	Data       *string `json:"data,omitempty"`
	Format     *string `json:"format,omitempty"`
}

// InputAudio for user audio input
type InputAudio struct {
	Data       string  `json:"data"`
	Format     string  `json:"format"`
	Transcript *string `json:"transcript,omitempty"`
}

// CodeContent represents code with execution metadata
type CodeContent struct {
	Language    string         `json:"language"`
	Code        string         `json:"code"`
	ExecutionID *string        `json:"execution_id,omitempty"`
	Output      *string        `json:"output,omitempty"`
	Error       *string        `json:"error,omitempty"`
	ExitCode    *int           `json:"exit_code,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ScreenshotContent represents a screenshot from computer use
type ScreenshotContent struct {
	ImageURL    string  `json:"image_url"`
	ImageData   *string `json:"image_data,omitempty"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Timestamp   int64   `json:"timestamp"`
	Description *string `json:"description,omitempty"`
}

// ComputerAction represents a computer interaction action
type ComputerAction struct {
	Action      string         `json:"action"`
	Coordinates *Coordinates   `json:"coordinates,omitempty"`
	Text        *string        `json:"text,omitempty"`
	Key         *string        `json:"key,omitempty"`
	ScrollDelta *ScrollDelta   `json:"scroll_delta,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Coordinates represents screen coordinates
type Coordinates struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// ScrollDelta represents scroll amount
type ScrollDelta struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// FunctionCall represents a function/tool call
type FunctionCall struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
}

// FunctionCallOut represents the output of a function call
type FunctionCallOut struct {
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function,omitempty"`
}

// Annotation for text content
type Annotation struct {
	Type        string   `json:"type"`
	Text        string   `json:"text,omitempty"`
	FileID      string   `json:"file_id,omitempty"`
	Filename    *string  `json:"filename,omitempty"`
	ContainerID *string  `json:"container_id,omitempty"`
	URL         string   `json:"url,omitempty"`
	Quote       *string  `json:"quote,omitempty"`
	PageNumber  *int     `json:"page_number,omitempty"`
	BoundingBox *BBox    `json:"bounding_box,omitempty"`
	Confidence  *float64 `json:"confidence,omitempty"`
	StartIndex  int      `json:"start_index"`
	EndIndex    int      `json:"end_index"`
	Index       int      `json:"index,omitempty"`
}

// BBox represents a bounding box for spatial annotations
type BBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// LogProb for AI responses
type LogProb struct {
	Token       string       `json:"token"`
	LogProb     float64      `json:"logprob"`
	Bytes       []int        `json:"bytes,omitempty"`
	TopLogProbs []TopLogProb `json:"top_logprobs,omitempty"`
}

type TopLogProb struct {
	Token   string  `json:"token"`
	LogProb float64 `json:"logprob"`
	Bytes   []int   `json:"bytes,omitempty"`
}

// ===============================================
// Factory Functions
// ===============================================

// NewConversation creates a new conversation with the given parameters (string UserID for backward compat)
func NewConversation(publicID string, userID string, title *string, metadata map[string]string) *Conversation {
	now := time.Now()
	if metadata == nil {
		metadata = make(map[string]string)
	}
	return &Conversation{
		PublicID:                     publicID,
		Object:                       "conversation",
		Title:                        title,
		UserID:                       userID,
		Status:                       ConversationStatusActive,
		ActiveBranch:                 BranchMain,
		Branches:                     make(map[string][]Item),
		BranchMetadata:               make(map[string]BranchMetadata),
		Metadata:                     metadata,
		IsPrivate:                    false,
		InstructionVersion:           1,
		EffectiveInstructionSnapshot: nil,
		CreatedAt:                    now,
		UpdatedAt:                    now,
	}
}

// NewItem creates a new conversation item
func NewItem(publicID string, itemType ItemType, role ItemRole, content []Content, conversationID uint, responseID *uint) *Item {
	return &Item{
		PublicID:       publicID,
		Object:         "conversation.item",
		Type:           itemType,
		Role:           role,
		Content:        content,
		ConversationID: conversationID,
		ResponseID:     responseID,
		CreatedAt:      time.Now(),
	}
}

// NewLegacyItem creates an item with legacy fields for backward compatibility
func NewLegacyItem(conversationID uint, sequence int, role ItemRole, status ItemStatus, content map[string]interface{}) Item {
	return Item{
		ConversationID: conversationID,
		Sequence:       sequence,
		SequenceNumber: sequence,
		Role:           role,
		Status:         status,
		LegacyContent:  content,
		CreatedAt:      time.Now(),
	}
}

// ===============================================
// Content Custom JSON Marshaling
// ===============================================

// MarshalJSON implements custom JSON marshaling for Content
func (c Content) MarshalJSON() ([]byte, error) {
	type Alias Content
	baseJSON, err := json.Marshal((*Alias)(&c))
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(baseJSON, &result); err != nil {
		return nil, err
	}
	switch c.Type {
	case "input_text", "reasoning_text", "tool_result", "mcp_call":
		if c.TextString != nil {
			result[c.Type] = *c.TextString
		}
	case "text":
		if c.TextString != nil {
			result["text"] = *c.TextString
		}
	}
	return json.Marshal(result)
}

// UnmarshalJSON implements custom JSON unmarshaling for Content
func (c *Content) UnmarshalJSON(data []byte) error {
	type Alias Content
	aux := &struct {
		*Alias
	}{Alias: (*Alias)(c)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}
	switch c.Type {
	case "input_text", "reasoning_text", "tool_result", "mcp_call":
		if textRaw, ok := rawMap[c.Type]; ok {
			var textStr string
			if err := json.Unmarshal(textRaw, &textStr); err == nil {
				c.TextString = &textStr
				return nil
			}
		}
		if textRaw, ok := rawMap["text"]; ok {
			var textStr string
			if err := json.Unmarshal(textRaw, &textStr); err == nil {
				c.TextString = &textStr
			}
		}
	case "text":
		if textRaw, ok := rawMap["text"]; ok {
			var textStr string
			if err := json.Unmarshal(textRaw, &textStr); err == nil {
				c.TextString = &textStr
			}
		}
	}
	return nil
}

// Helper function to create a pointer to ItemStatus
func ToItemStatusPtr(s ItemStatus) *ItemStatus {
	return &s
}

// Helper function to create a pointer to ItemRole
func ToItemRolePtr(r ItemRole) *ItemRole {
	return &r
}
