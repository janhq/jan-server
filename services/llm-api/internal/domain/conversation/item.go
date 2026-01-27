package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"jan-server/services/llm-api/internal/domain/query"
)

// ===============================================
// Item Types and Enums
// ===============================================

// @Enum(message, function_call, function_call_output, reasoning, file_search_call, web_search_call, image_generation_call, image_edit_call, computer_call, computer_call_output, code_interpreter_call, local_shell_call, local_shell_call_output, shell_call, shell_call_output, apply_patch_call, apply_patch_call_output, mcp_list_tools, mcp_approval_request, mcp_approval_response, mcp_call, custom_tool_call, custom_tool_call_output)
type ItemType string

const (
	// Core message types
	ItemTypeMessage         ItemType = "message"
	ItemTypeFunctionCall    ItemType = "function_call"
	ItemTypeFunctionCallOut ItemType = "function_call_output"
	ItemTypeReasoning       ItemType = "reasoning" // For o1/reasoning models

	// OpenAI-compatible specific tool call types
	ItemTypeFileSearchCall       ItemType = "file_search_call"
	ItemTypeWebSearchCall        ItemType = "web_search_call"
	ItemTypeImageGenerationCall  ItemType = "image_generation_call"
	ItemTypeImageEditCall        ItemType = "image_edit_call"
	ItemTypeComputerCall         ItemType = "computer_call"
	ItemTypeComputerCallOutput   ItemType = "computer_call_output"
	ItemTypeCodeInterpreterCall  ItemType = "code_interpreter_call"
	ItemTypeLocalShellCall       ItemType = "local_shell_call"
	ItemTypeLocalShellCallOutput ItemType = "local_shell_call_output"
	ItemTypeShellCall            ItemType = "shell_call"
	ItemTypeShellCallOutput      ItemType = "shell_call_output"
	ItemTypeApplyPatchCall       ItemType = "apply_patch_call"
	ItemTypeApplyPatchCallOutput ItemType = "apply_patch_call_output"

	// MCP-specific types
	ItemTypeMcpListTools        ItemType = "mcp_list_tools"
	ItemTypeMcpApprovalRequest  ItemType = "mcp_approval_request"
	ItemTypeMcpApprovalResponse ItemType = "mcp_approval_response"
	ItemTypeMcpCall             ItemType = "mcp_call"

	// Custom tool types
	ItemTypeCustomToolCall       ItemType = "custom_tool_call"
	ItemTypeCustomToolCallOutput ItemType = "custom_tool_call_output"

	// Legacy types (for backward compatibility)
	ItemTypeFileSearch      ItemType = "file_search"      // Legacy: maps to file_search_call
	ItemTypeWebSearch       ItemType = "web_search"       // Legacy: maps to web_search_call
	ItemTypeCodeInterpreter ItemType = "code_interpreter" // Legacy: maps to code_interpreter_call
	ItemTypeComputerUse     ItemType = "computer_use"     // Legacy: maps to computer_call
	ItemTypeMCPItem         ItemType = "mcp_item"         // Legacy: maps to mcp_call
	ItemTypeImageGeneration ItemType = "image_generation" // Legacy: maps to image_generation_call
)

func ValidateItemType(input string) bool {
	switch ItemType(input) {
	case ItemTypeMessage, ItemTypeFunctionCall, ItemTypeFunctionCallOut, ItemTypeReasoning,
		ItemTypeFileSearchCall, ItemTypeWebSearchCall, ItemTypeImageGenerationCall, ItemTypeImageEditCall,
		ItemTypeComputerCall, ItemTypeComputerCallOutput,
		ItemTypeCodeInterpreterCall,
		ItemTypeLocalShellCall, ItemTypeLocalShellCallOutput,
		ItemTypeShellCall, ItemTypeShellCallOutput,
		ItemTypeApplyPatchCall, ItemTypeApplyPatchCallOutput,
		ItemTypeMcpListTools, ItemTypeMcpApprovalRequest, ItemTypeMcpApprovalResponse, ItemTypeMcpCall,
		ItemTypeCustomToolCall, ItemTypeCustomToolCallOutput,
		// Legacy types
		ItemTypeFileSearch, ItemTypeWebSearch, ItemTypeCodeInterpreter,
		ItemTypeComputerUse, ItemTypeMCPItem, ItemTypeImageGeneration:
		return true
	default:
		return false
	}
}

// @Enum(system, user, assistant, tool, developer, critic, discriminator, unknown)
type ItemRole string

const (
	ItemRoleSystem        ItemRole = "system"
	ItemRoleUser          ItemRole = "user"
	ItemRoleAssistant     ItemRole = "assistant"
	ItemRoleTool          ItemRole = "tool"
	ItemRoleDeveloper     ItemRole = "developer"     // System-level instructions (OpenAI replacement for system)
	ItemRoleCritic        ItemRole = "critic"        // For critique/evaluation workflows
	ItemRoleDiscriminator ItemRole = "discriminator" // For adversarial/validation workflows
	ItemRoleUnknown       ItemRole = "unknown"       // Fallback for unrecognized roles
)

func ValidateItemRole(input string) bool {
	switch ItemRole(input) {
	case ItemRoleSystem, ItemRoleUser, ItemRoleAssistant, ItemRoleTool,
		ItemRoleDeveloper, ItemRoleCritic, ItemRoleDiscriminator, ItemRoleUnknown:
		return true
	default:
		return false
	}
}

// @Enum(incomplete, in_progress, completed, failed, cancelled, searching, generating, calling, streaming, rate_limited)
type ItemStatus string

const (
	ItemStatusIncomplete  ItemStatus = "incomplete"   // Not started or partially complete (OpenAI uses this instead of "pending")
	ItemStatusInProgress  ItemStatus = "in_progress"  // Currently processing
	ItemStatusCompleted   ItemStatus = "completed"    // Successfully finished
	ItemStatusFailed      ItemStatus = "failed"       // Failed with error
	ItemStatusCancelled   ItemStatus = "cancelled"    // Cancelled by user or system
	ItemStatusSearching   ItemStatus = "searching"    // File/web search in progress
	ItemStatusGenerating  ItemStatus = "generating"   // Image generation in progress
	ItemStatusCalling     ItemStatus = "calling"      // Function/tool call in progress
	ItemStatusStreaming   ItemStatus = "streaming"    // Streaming response in progress
	ItemStatusRateLimited ItemStatus = "rate_limited" // Rate limit hit
)

func ValidateItemStatus(input string) bool {
	switch ItemStatus(input) {
	case ItemStatusIncomplete, ItemStatusInProgress, ItemStatusCompleted,
		ItemStatusFailed, ItemStatusCancelled, ItemStatusSearching,
		ItemStatusGenerating, ItemStatusCalling, ItemStatusStreaming,
		ItemStatusRateLimited:
		return true
	default:
		return false
	}
}

// ToItemStatusPtr returns a pointer to the given ItemStatus
func ToItemStatusPtr(s ItemStatus) *ItemStatus {
	return &s
}

// ItemStatusToStringPtr converts *ItemStatus to *string
func ItemStatusToStringPtr(s *ItemStatus) *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}

// ===============================================
// Item Structures
// ===============================================

// BaseItem contains common fields for all item types
type BaseItem struct {
	ID                uint               `json:"-"`
	ConversationID    uint               `json:"-"`
	PublicID          string             `json:"id"`
	Object            string             `json:"object"` // Always "conversation.item"
	Type              ItemType           `json:"type"`
	Status            *ItemStatus        `json:"status,omitempty"`
	IncompleteAt      *time.Time         `json:"incomplete_at,omitempty"`
	IncompleteDetails *IncompleteDetails `json:"incomplete_details,omitempty"`
	CompletedAt       *time.Time         `json:"completed_at,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
}

// MessageItem represents a message in the conversation
type MessageItem struct {
	BaseItem
	Role    ItemRole  `json:"role"`
	Content []Content `json:"content"`
}

// FunctionCallItem represents a function/tool call
type FunctionCallItem struct {
	BaseItem
	CallID    string  `json:"call_id"`
	Name      string  `json:"name"`
	Arguments string  `json:"arguments"`
	ToolType  *string `json:"tool_type,omitempty"` // "function", "file_search", "code_interpreter", etc.
}

// FunctionCallOutputItem represents the output of a function call
type FunctionCallOutputItem struct {
	BaseItem
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

// ReasoningItem represents internal reasoning from models like o1
type ReasoningItem struct {
	BaseItem
	Summary  string    `json:"summary"`
	Thinking []Content `json:"thinking,omitempty"` // Internal reasoning steps
}

// FileSearchItem represents a file search operation
type FileSearchItem struct {
	BaseItem
	Query    string             `json:"query"`
	FileIDs  []string           `json:"file_ids,omitempty"`
	Results  []FileSearchResult `json:"results,omitempty"`
	Metadata map[string]string  `json:"metadata,omitempty"`
}

// FileSearchResult represents a single file search result
type FileSearchResult struct {
	FileID      string       `json:"file_id"`
	Filename    string       `json:"filename"`
	Score       float64      `json:"score"`
	Content     string       `json:"content"`
	PageNumber  *int         `json:"page_number,omitempty"`
	Annotations []Annotation `json:"annotations,omitempty"`
}

// WebSearchItem represents a web search operation
type WebSearchItem struct {
	BaseItem
	Query   string            `json:"query"`
	Results []WebSearchResult `json:"results,omitempty"`
}

// WebSearchResult represents a single web search result
type WebSearchResult struct {
	Title   string   `json:"title"`
	URL     string   `json:"url"`
	Snippet string   `json:"snippet"`
	Score   *float64 `json:"score,omitempty"`
}

// CodeInterpreterItem represents code execution
type CodeInterpreterItem struct {
	BaseItem
	Language string         `json:"language"`
	Code     string         `json:"code"`
	Output   *string        `json:"output,omitempty"`
	Error    *string        `json:"error,omitempty"`
	ExitCode *int           `json:"exit_code,omitempty"`
	Files    []string       `json:"files,omitempty"` // Generated file IDs
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ComputerUseItem represents computer interaction
type ComputerUseItem struct {
	BaseItem
	Action     ComputerAction     `json:"action"`
	Screenshot *ScreenshotContent `json:"screenshot,omitempty"`
	Result     *string            `json:"result,omitempty"`
	Error      *string            `json:"error,omitempty"`
}

// CustomToolCallItem represents a custom tool invocation
type CustomToolCallItem struct {
	BaseItem
	ToolID   string         `json:"tool_id"`
	ToolName string         `json:"tool_name"`
	Input    map[string]any `json:"input"`
	Output   map[string]any `json:"output,omitempty"`
}

// MCPItem represents a Model Context Protocol item
type MCPItem struct {
	BaseItem
	Protocol string         `json:"protocol"`
	Action   string         `json:"action"`
	Data     map[string]any `json:"data"`
}

// ImageGenerationItem represents image generation (DALL-E)
type ImageGenerationItem struct {
	BaseItem
	Prompt        string   `json:"prompt"`
	Model         *string  `json:"model,omitempty"`
	Size          *string  `json:"size,omitempty"`    // "256x256", "512x512", "1024x1024", etc.
	Quality       *string  `json:"quality,omitempty"` // "standard", "hd"
	Style         *string  `json:"style,omitempty"`   // "vivid", "natural"
	ImageURLs     []string `json:"image_urls,omitempty"`
	RevisedPrompt *string  `json:"revised_prompt,omitempty"`
}

// ===============================================
// OpenAI-Compatible Item Structures
// ===============================================

// LocalShellCallAction represents the action for local shell execution
type LocalShellCallAction struct {
	Type             string            `json:"type"` // Always "exec"
	Command          []string          `json:"command"`
	Env              map[string]string `json:"env"`
	TimeoutMs        *int64            `json:"timeout_ms,omitempty"`
	User             *string           `json:"user,omitempty"`
	WorkingDirectory *string           `json:"working_directory,omitempty"`
}

// LocalShellCallItem represents a local shell command execution call
type LocalShellCallItem struct {
	BaseItem
	CallID string               `json:"call_id"`
	Action LocalShellCallAction `json:"action"`
	Status string               `json:"status"` // in_progress, completed, incomplete
}

// LocalShellCallOutputItem represents the output of a local shell call
type LocalShellCallOutputItem struct {
	BaseItem
	CallID string  `json:"call_id"`
	Output string  `json:"output"`
	Status *string `json:"status,omitempty"` // in_progress, completed, incomplete
}

// McpTool represents a tool available on an MCP server
type McpTool struct {
	Name        string  `json:"name"`
	InputSchema any     `json:"input_schema"`
	Description *string `json:"description,omitempty"`
	Annotations any     `json:"annotations,omitempty"`
}

// McpListToolsItem represents a list of tools available on an MCP server
type McpListToolsItem struct {
	BaseItem
	ServerLabel string    `json:"server_label"`
	Tools       []McpTool `json:"tools"`
	Error       *string   `json:"error,omitempty"`
}

// McpApprovalRequestItem represents a request for human approval of a tool invocation
type McpApprovalRequestItem struct {
	BaseItem
	Name        string `json:"name"`
	Arguments   string `json:"arguments"`
	ServerLabel string `json:"server_label"`
}

// McpApprovalResponseItem represents a response to an MCP approval request
type McpApprovalResponseItem struct {
	BaseItem
	ApprovalRequestID string  `json:"approval_request_id"`
	Approve           bool    `json:"approve"`
	Reason            *string `json:"reason,omitempty"`
}

// McpCallItem represents an invocation of a tool on an MCP server
type McpCallItem struct {
	BaseItem
	Name              string  `json:"name"`
	Arguments         string  `json:"arguments"`
	ServerLabel       string  `json:"server_label"`
	ApprovalRequestID *string `json:"approval_request_id,omitempty"`
	Output            *string `json:"output,omitempty"`
	Error             *string `json:"error,omitempty"`
	Status            string  `json:"status"` // in_progress, completed, incomplete, calling, failed
}

// SafetyCheck represents a safety check for computer use
type SafetyCheck struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

// ComputerCallItem represents a computer use tool call
type ComputerCallItem struct {
	BaseItem
	CallID              string         `json:"call_id"`
	Action              ComputerAction `json:"action"`
	Status              string         `json:"status"`
	PendingSafetyChecks []SafetyCheck  `json:"pending_safety_checks,omitempty"`
}

// ComputerCallOutputItem represents the output of a computer use call
type ComputerCallOutputItem struct {
	BaseItem
	CallID                   string        `json:"call_id"`
	Output                   any           `json:"output"` // Can be screenshot or other output
	Status                   string        `json:"status"`
	AcknowledgedSafetyChecks []SafetyCheck `json:"acknowledged_safety_checks,omitempty"`
}

// PatchOperation represents a file patching operation
type PatchOperation struct {
	Type      string `json:"type"` // Type of patch operation
	FilePath  string `json:"file_path"`
	PatchData string `json:"patch_data"`
}

// ApplyPatchCallItem represents a patch application call
type ApplyPatchCallItem struct {
	BaseItem
	CallID    string         `json:"call_id"`
	Operation PatchOperation `json:"operation"`
	Status    string         `json:"status"`
}

// ApplyPatchCallOutputItem represents the output of a patch call
type ApplyPatchCallOutputItem struct {
	BaseItem
	CallID string `json:"call_id"`
	Output string `json:"output"`
	Status string `json:"status"`
}

// ShellOutput represents output from a shell command
type ShellOutput struct {
	Type  string `json:"type"` // stdout, stderr, exit_code
	Value string `json:"value"`
}

// ShellCallItem represents a server-side shell execution call
type ShellCallItem struct {
	BaseItem
	CallID          string   `json:"call_id"`
	Commands        []string `json:"commands"`
	MaxOutputLength *int64   `json:"max_output_length,omitempty"`
	Status          string   `json:"status"`
}

// ShellCallOutputItem represents the output of a shell call
type ShellCallOutputItem struct {
	BaseItem
	CallID string        `json:"call_id"`
	Output []ShellOutput `json:"output"`
	Status string        `json:"status"`
}

// Item is the legacy/generic item structure for backward compatibility
// New code should use the type-specific item structs above
type Item struct {
	ID                uint               `json:"-"`
	ConversationID    uint               `json:"-"`
	PublicID          string             `json:"id"`
	Object            string             `json:"object"`                    // Always "conversation.item" for OpenAI compatibility
	Branch            string             `json:"branch,omitempty"`          // Branch identifier (MAIN, EDIT_1, etc.)
	SequenceNumber    int                `json:"sequence_number,omitempty"` // Order within branch
	Type              ItemType           `json:"type"`
	Role              *ItemRole          `json:"role,omitempty"`
	Content           []Content          `json:"content,omitempty"`
	Status            *ItemStatus        `json:"status,omitempty"`
	IncompleteAt      *time.Time         `json:"incomplete_at,omitempty"`
	IncompleteDetails *IncompleteDetails `json:"incomplete_details,omitempty"`
	CompletedAt       *time.Time         `json:"completed_at,omitempty"`
	ResponseID        *uint              `json:"-"`

	// User feedback/rating
	Rating        *ItemRating `json:"rating,omitempty"`         // Like/unlike rating
	RatedAt       *time.Time  `json:"rated_at,omitempty"`       // When rating was given
	RatingComment *string     `json:"rating_comment,omitempty"` // Optional comment with rating

	// OpenAI-compatible fields for specific item types
	CallID                   *string                `json:"call_id,omitempty"`                    // For function/tool calls
	Name                     *string                `json:"name,omitempty"`                       // For MCP tool calls - tool name
	ServerLabel              *string                `json:"server_label,omitempty"`               // For MCP calls
	ApprovalRequestID        *string                `json:"approval_request_id,omitempty"`        // For MCP approval
	Arguments                *string                `json:"arguments,omitempty"`                  // For tool calls (JSON string)
	Output                   *string                `json:"output,omitempty"`                     // For tool call outputs
	Error                    *string                `json:"error,omitempty"`                      // For failed calls
	Action                   map[string]interface{} `json:"action,omitempty"`                     // For computer/shell actions
	Tools                    []McpTool              `json:"tools,omitempty"`                      // For mcp_list_tools
	PendingSafetyChecks      []SafetyCheck          `json:"pending_safety_checks,omitempty"`      // For computer calls
	AcknowledgedSafetyChecks []SafetyCheck          `json:"acknowledged_safety_checks,omitempty"` // For computer call outputs
	Approve                  *bool                  `json:"approve,omitempty"`                    // For MCP approval response
	Reason                   *string                `json:"reason,omitempty"`                     // For MCP approval reason
	Commands                 []string               `json:"commands,omitempty"`                   // For shell calls
	MaxOutputLength          *int64                 `json:"max_output_length,omitempty"`          // For shell calls
	ShellOutputs             []ShellOutput          `json:"shell_outputs,omitempty"`              // For shell call outputs
	Operation                map[string]interface{} `json:"operation,omitempty"`                  // For patch operations

	CreatedAt time.Time `json:"created_at"`
}

// ===============================================
// Rating Support
// ===============================================

// ItemRating represents like/unlike feedback on an item
type ItemRating string

const (
	ItemRatingLike   ItemRating = "like"   // Positive feedback (like)
	ItemRatingUnlike ItemRating = "unlike" // Negative feedback (unlike)
)

// Validate checks if the rating is valid
func (r ItemRating) Validate() bool {
	return r == ItemRatingLike || r == ItemRatingUnlike
}

// String returns the string representation
func (r ItemRating) String() string {
	return string(r)
}

// ToItemRatingPtr returns a pointer to the given ItemRating
func ToItemRatingPtr(r ItemRating) *ItemRating {
	return &r
}

// ParseItemRating converts a string to ItemRating
func ParseItemRating(s string) (*ItemRating, error) {
	rating := ItemRating(s)
	if !rating.Validate() {
		return nil, fmt.Errorf("invalid rating: must be 'like' or 'unlike'")
	}
	return &rating, nil
}

// ===============================================
// Content Structures
// ===============================================

type Content struct {
	Type               string             `json:"type"`
	FinishReason       *string            `json:"finish_reason,omitempty"`        // Finish reason
	Text               *Text              `json:"-"`                              // Complex text content with annotations (for "text" type)
	TextString         *string            `json:"-"`                              // Simple text string (for input_text, reasoning_text, tool_result)
	OutputText         *OutputText        `json:"output_text,omitempty"`          // AI output text (with annotations)
	Refusal            *string            `json:"refusal,omitempty"`              // Model refusal message
	SummaryText        *string            `json:"summary_text,omitempty"`         // Summary content
	Thinking           *string            `json:"thinking,omitempty"`             // Internal reasoning (o1 models)
	Image              *ImageContent      `json:"image,omitempty"`                // Image content
	File               *FileContent       `json:"file,omitempty"`                 // File content
	Audio              *AudioContent      `json:"audio,omitempty"`                // Audio content for speech
	InputAudio         *InputAudio        `json:"input_audio,omitempty"`          // User audio input
	Code               *CodeContent       `json:"code,omitempty"`                 // Code block with execution metadata
	ComputerScreenshot *ScreenshotContent `json:"computer_screenshot,omitempty"`  // Screenshot from computer use
	ComputerAction     *ComputerAction    `json:"computer_action,omitempty"`      // Computer interaction details
	FunctionCall       *FunctionCall      `json:"function_call,omitempty"`        // Function call content (deprecated, use tool_calls)
	FunctionCallOut    *FunctionCallOut   `json:"function_call_output,omitempty"` // Function call output
	ToolCalls          []ToolCall         `json:"tool_calls,omitempty"`           // Tool calls (for assistant messages)
	ToolCallID         *string            `json:"tool_call_id,omitempty"`         // Tool call ID (for tool responses)
	Reasoning          *string            `json:"reasoning_text,omitempty"`       // Internal reasoning (o1 models)
}

// Text content - matches OpenAI's text content format
type Text struct {
	Text        string       `json:"text"` // Changed from "value" to match OpenAI spec
	Annotations []Annotation `json:"annotations,omitempty"`
}

type OutputText struct {
	Text        string       `json:"text"`
	Annotations []Annotation `json:"annotations"`        // Required for OpenAI compatibility
	LogProbs    []LogProb    `json:"logprobs,omitempty"` // Token probabilities
}

// Image content for multimodal support
type ImageContent struct {
	URL    string `json:"url,omitempty"`
	FileID string `json:"file_id,omitempty"`
	Detail string `json:"detail,omitempty"` // "low", "high", "auto"
}

// File content for attachments
type FileContent struct {
	FileID   string `json:"file_id,omitempty"`
	URL      string `json:"url,omitempty"`      // Direct URL to the file
	Name     string `json:"name,omitempty"`     // Display name / filename
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Detail   string `json:"detail,omitempty"` // "auto", "low", "high" for processing detail level
}

// Audio content for speech output
type AudioContent struct {
	ID         string  `json:"id,omitempty"`
	Transcript *string `json:"transcript,omitempty"` // Text transcription of audio
	Data       *string `json:"data,omitempty"`       // Base64 encoded audio data
	Format     *string `json:"format,omitempty"`     // Audio format: mp3, wav, pcm16, etc.
}

// InputAudio for user audio input
type InputAudio struct {
	Data       string  `json:"data"`                 // Base64 encoded audio data
	Format     string  `json:"format"`               // Audio format: mp3, wav, pcm16, etc.
	Transcript *string `json:"transcript,omitempty"` // Optional text transcription
}

// CodeContent represents code with execution metadata
type CodeContent struct {
	Language    string         `json:"language"`               // Programming language
	Code        string         `json:"code"`                   // Code content
	ExecutionID *string        `json:"execution_id,omitempty"` // Execution session ID
	Output      *string        `json:"output,omitempty"`       // Execution output
	Error       *string        `json:"error,omitempty"`        // Execution error
	ExitCode    *int           `json:"exit_code,omitempty"`    // Process exit code
	Metadata    map[string]any `json:"metadata,omitempty"`     // Additional metadata
}

// ScreenshotContent represents a screenshot from computer use
type ScreenshotContent struct {
	ImageURL    string  `json:"image_url"`             // URL to screenshot image
	ImageData   *string `json:"image_data,omitempty"`  // Base64 encoded image data
	Width       int     `json:"width"`                 // Image width in pixels
	Height      int     `json:"height"`                // Image height in pixels
	Timestamp   int64   `json:"timestamp"`             // Unix timestamp when screenshot was taken
	Description *string `json:"description,omitempty"` // Optional description
}

// ComputerAction represents a computer interaction action
type ComputerAction struct {
	Action      string         `json:"action"`                 // Action type: "click", "type", "key", "scroll", "move", etc.
	Coordinates *Coordinates   `json:"coordinates,omitempty"`  // Screen coordinates for mouse actions
	Text        *string        `json:"text,omitempty"`         // Text for typing actions
	Key         *string        `json:"key,omitempty"`          // Key for keyboard actions
	ScrollDelta *ScrollDelta   `json:"scroll_delta,omitempty"` // Scroll amount
	Metadata    map[string]any `json:"metadata,omitempty"`     // Additional action metadata
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
	ID        string `json:"id,omitempty"`        // Call ID
	Name      string `json:"name"`                // Function name
	Arguments string `json:"arguments,omitempty"` // JSON-encoded arguments
}

// FunctionCallOut represents the output of a function call
type FunctionCallOut struct {
	CallID string `json:"call_id"` // ID of the function call this responds to
	Output string `json:"output"`  // String output from the function
}

// ToolCall represents a tool invocation (superset of function calls)
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function", "file_search", "code_interpreter"
	Function FunctionCall `json:"function,omitempty"`
}

type Annotation struct {
	Type        string   `json:"type"`                   // "file_citation", "url_citation", "file_path", etc.
	Text        string   `json:"text,omitempty"`         // Display text
	FileID      string   `json:"file_id,omitempty"`      // For file citations
	Filename    *string  `json:"filename,omitempty"`     // File name for citations
	ContainerID *string  `json:"container_id,omitempty"` // Document container reference
	URL         string   `json:"url,omitempty"`          // For URL citations
	Quote       *string  `json:"quote,omitempty"`        // Actual quoted text from source
	PageNumber  *int     `json:"page_number,omitempty"`  // Page reference for documents
	BoundingBox *BBox    `json:"bounding_box,omitempty"` // Bounding box for image/PDF annotations
	Confidence  *float64 `json:"confidence,omitempty"`   // Citation confidence score (0.0-1.0)
	StartIndex  int      `json:"start_index"`            // Start position in text
	EndIndex    int      `json:"end_index"`              // End position in text
	Index       int      `json:"index,omitempty"`        // Citation index
}

// BBox represents a bounding box for spatial annotations
type BBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Log probability for AI responses
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

type IncompleteDetails struct {
	Reason string  `json:"reason"`          // "max_tokens", "content_filter", "tool_calls", etc.
	Error  *string `json:"error,omitempty"` // Error message if applicable
}

// ===============================================
// Item Repository
// ===============================================

type ItemFilter struct {
	ID             *uint
	PublicID       *string
	CallID         *string // For finding items by tool call ID (e.g., call_xxx)
	ConversationID *uint
	Role           *ItemRole
	ResponseID     *uint
	Branch         *string // Filter by branch name
}

type ItemRepository interface {
	Create(ctx context.Context, item *Item) error
	FindByID(ctx context.Context, id uint) (*Item, error)
	FindByPublicID(ctx context.Context, publicID string) (*Item, error) // Find by OpenAI-compatible string ID
	FindByConversationID(ctx context.Context, conversationID uint) ([]*Item, error)
	Search(ctx context.Context, conversationID uint, query string) ([]*Item, error)
	Delete(ctx context.Context, id uint) error
	BulkCreate(ctx context.Context, items []*Item) error
	CountByConversation(ctx context.Context, conversationID uint) (int64, error)
	ExistsByIDAndConversation(ctx context.Context, itemID uint, conversationID uint) (bool, error)
	FindByFilter(ctx context.Context, filter ItemFilter, pagination *query.Pagination) ([]*Item, error)
	Count(ctx context.Context, filter ItemFilter) (int64, error)
}

// ===============================================
// Item Factory Functions
// ===============================================

// NewItem creates a new conversation item with the given parameters (legacy)
func NewItem(publicID string, itemType ItemType, role ItemRole, content []Content, conversationID uint, responseID *uint) *Item {
	return &Item{
		PublicID:       publicID,
		Object:         "conversation.item",
		Type:           itemType,
		Role:           &role,
		Content:        content,
		ConversationID: conversationID,
		ResponseID:     responseID,
		CreatedAt:      time.Now(),
	}
}

// NewMessageItem creates a new message item
func NewMessageItem(publicID string, role ItemRole, content []Content, conversationID uint) *MessageItem {
	return &MessageItem{
		BaseItem: BaseItem{
			ConversationID: conversationID,
			PublicID:       publicID,
			Object:         "conversation.item",
			Type:           ItemTypeMessage,
			CreatedAt:      time.Now(),
		},
		Role:    role,
		Content: content,
	}
}

// NewFunctionCallItem creates a new function call item
func NewFunctionCallItem(publicID string, callID string, name string, arguments string, conversationID uint) *FunctionCallItem {
	return &FunctionCallItem{
		BaseItem: BaseItem{
			ConversationID: conversationID,
			PublicID:       publicID,
			Object:         "conversation.item",
			Type:           ItemTypeFunctionCall,
			Status:         ToItemStatusPtr(ItemStatusCalling),
			CreatedAt:      time.Now(),
		},
		CallID:    callID,
		Name:      name,
		Arguments: arguments,
	}
}

// NewFunctionCallOutputItem creates a new function call output item
func NewFunctionCallOutputItem(publicID string, callID string, output string, conversationID uint) *FunctionCallOutputItem {
	return &FunctionCallOutputItem{
		BaseItem: BaseItem{
			ConversationID: conversationID,
			PublicID:       publicID,
			Object:         "conversation.item",
			Type:           ItemTypeFunctionCallOut,
			Status:         ToItemStatusPtr(ItemStatusCompleted),
			CreatedAt:      time.Now(),
		},
		CallID: callID,
		Output: output,
	}
}

// NewReasoningItem creates a new reasoning item
func NewReasoningItem(publicID string, summary string, thinking []Content, conversationID uint) *ReasoningItem {
	return &ReasoningItem{
		BaseItem: BaseItem{
			ConversationID: conversationID,
			PublicID:       publicID,
			Object:         "conversation.item",
			Type:           ItemTypeReasoning,
			CreatedAt:      time.Now(),
		},
		Summary:  summary,
		Thinking: thinking,
	}
}

// NewFileSearchItem creates a new file search item
func NewFileSearchItem(publicID string, query string, fileIDs []string, conversationID uint) *FileSearchItem {
	return &FileSearchItem{
		BaseItem: BaseItem{
			ConversationID: conversationID,
			PublicID:       publicID,
			Object:         "conversation.item",
			Type:           ItemTypeFileSearch,
			Status:         ToItemStatusPtr(ItemStatusSearching),
			CreatedAt:      time.Now(),
		},
		Query:   query,
		FileIDs: fileIDs,
	}
}

// NewWebSearchItem creates a new web search item
func NewWebSearchItem(publicID string, query string, conversationID uint) *WebSearchItem {
	return &WebSearchItem{
		BaseItem: BaseItem{
			ConversationID: conversationID,
			PublicID:       publicID,
			Object:         "conversation.item",
			Type:           ItemTypeWebSearch,
			Status:         ToItemStatusPtr(ItemStatusSearching),
			CreatedAt:      time.Now(),
		},
		Query: query,
	}
}

// NewCodeInterpreterItem creates a new code interpreter item
func NewCodeInterpreterItem(publicID string, language string, code string, conversationID uint) *CodeInterpreterItem {
	return &CodeInterpreterItem{
		BaseItem: BaseItem{
			ConversationID: conversationID,
			PublicID:       publicID,
			Object:         "conversation.item",
			Type:           ItemTypeCodeInterpreter,
			Status:         ToItemStatusPtr(ItemStatusInProgress),
			CreatedAt:      time.Now(),
		},
		Language: language,
		Code:     code,
	}
}

// NewComputerUseItem creates a new computer use item
func NewComputerUseItem(publicID string, action ComputerAction, conversationID uint) *ComputerUseItem {
	return &ComputerUseItem{
		BaseItem: BaseItem{
			ConversationID: conversationID,
			PublicID:       publicID,
			Object:         "conversation.item",
			Type:           ItemTypeComputerUse,
			Status:         ToItemStatusPtr(ItemStatusInProgress),
			CreatedAt:      time.Now(),
		},
		Action: action,
	}
}

// NewCustomToolCallItem creates a new custom tool call item
func NewCustomToolCallItem(publicID string, toolID string, toolName string, input map[string]any, conversationID uint) *CustomToolCallItem {
	return &CustomToolCallItem{
		BaseItem: BaseItem{
			ConversationID: conversationID,
			PublicID:       publicID,
			Object:         "conversation.item",
			Type:           ItemTypeCustomToolCall,
			Status:         ToItemStatusPtr(ItemStatusCalling),
			CreatedAt:      time.Now(),
		},
		ToolID:   toolID,
		ToolName: toolName,
		Input:    input,
	}
}

// NewMCPItem creates a new Model Context Protocol item
func NewMCPItem(publicID string, protocol string, action string, data map[string]any, conversationID uint) *MCPItem {
	return &MCPItem{
		BaseItem: BaseItem{
			ConversationID: conversationID,
			PublicID:       publicID,
			Object:         "conversation.item",
			Type:           ItemTypeMCPItem,
			CreatedAt:      time.Now(),
		},
		Protocol: protocol,
		Action:   action,
		Data:     data,
	}
}

// NewImageGenerationItem creates a new image generation item
func NewImageGenerationItem(publicID string, prompt string, conversationID uint) *ImageGenerationItem {
	return &ImageGenerationItem{
		BaseItem: BaseItem{
			ConversationID: conversationID,
			PublicID:       publicID,
			Object:         "conversation.item",
			Type:           ItemTypeImageGeneration,
			Status:         ToItemStatusPtr(ItemStatusGenerating),
			CreatedAt:      time.Now(),
		},
		Prompt: prompt,
	}
}

// ===============================================
// Content Factory Functions
// ===============================================

// NewTextContent creates a new text content item
func NewTextContent(text string) Content {
	return Content{
		Type:       "text",
		TextString: &text,
	}
}

// NewInputTextContent creates a new input text content (for user messages)
func NewInputTextContent(text string) Content {
	return Content{
		Type:       "input_text",
		TextString: &text,
	}
}

// NewOutputTextContent creates a new output text content with annotations
func NewOutputTextContent(text string, annotations []Annotation) Content {
	return Content{
		Type: "output_text",
		OutputText: &OutputText{
			Text:        text,
			Annotations: annotations,
		},
	}
}

// NewImageContent creates a new image content
func NewImageContent(url, fileID, detail string) Content {
	return Content{
		Type: "image",
		Image: &ImageContent{
			URL:    url,
			FileID: fileID,
			Detail: detail,
		},
	}
}

// NewAudioContent creates a new audio content
func NewAudioContent(id string, transcript *string, format *string) Content {
	return Content{
		Type: "audio",
		Audio: &AudioContent{
			ID:         id,
			Transcript: transcript,
			Format:     format,
		},
	}
}

// NewFileContent creates a new file content for document attachments
func NewFileContent(url, filename, mimeType, detail string) Content {
	return Content{
		Type: "file",
		File: &FileContent{
			URL:      url,
			Name:     filename,
			MimeType: mimeType,
			Detail:   detail,
		},
	}
}

// NewRefusalContent creates a refusal content (when model refuses to answer)
func NewRefusalContent(refusalMessage string) Content {
	return Content{
		Type:    "refusal",
		Refusal: &refusalMessage,
	}
}

// NewThinkingContent creates thinking content (for o1 reasoning models)
func NewThinkingContent(thinking string) Content {
	return Content{
		Type:     "thinking",
		Thinking: &thinking,
	}
}

// NewCodeContent creates code content with execution metadata
func NewCodeContent(language string, code string, output *string, exitCode *int) Content {
	return Content{
		Type: "code",
		Code: &CodeContent{
			Language: language,
			Code:     code,
			Output:   output,
			ExitCode: exitCode,
		},
	}
}

// NewComputerScreenshotContent creates a computer screenshot content
func NewComputerScreenshotContent(imageURL string, width int, height int) Content {
	return Content{
		Type: "computer_screenshot",
		ComputerScreenshot: &ScreenshotContent{
			ImageURL:  imageURL,
			Width:     width,
			Height:    height,
			Timestamp: time.Now().Unix(),
		},
	}
}

// NewComputerActionContent creates a computer action content
func NewComputerActionContent(action string, coords *Coordinates, text *string) Content {
	compAction := ComputerAction{
		Action:      action,
		Coordinates: coords,
		Text:        text,
	}
	return Content{
		Type:           "computer_action",
		ComputerAction: &compAction,
	}
}

// MarshalJSON implements custom JSON marshaling for Content
// This handles the text field which can be either a string or an object with annotations
func (c Content) MarshalJSON() ([]byte, error) {
	type Alias Content

	// Marshal the base struct without text field
	baseJSON, err := json.Marshal((*Alias)(&c))
	if err != nil {
		return nil, err
	}

	// Unmarshal to a map so we can add dynamic fields
	var result map[string]interface{}
	if err := json.Unmarshal(baseJSON, &result); err != nil {
		return nil, err
	}

	// Determine what to use for the text field based on content type
	switch c.Type {
	case "input_text", "reasoning_text", "tool_result", "mcp_call":
		// Use type-specific field name (e.g., "input_text": "...")
		if c.TextString != nil {
			result[c.Type] = *c.TextString
		}
	case "text":
		// Use "text" field for generic text type
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
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Parse text field based on content type
	// Unmarshal into a map to check for dynamic field names
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	switch c.Type {
	case "input_text", "reasoning_text", "tool_result", "mcp_call":
		// Try type-specific field first (e.g., "input_text")
		if textRaw, ok := rawMap[c.Type]; ok {
			var textStr string
			if err := json.Unmarshal(textRaw, &textStr); err == nil {
				c.TextString = &textStr
				return nil
			}
		}
		// Fall back to "text" field for backward compatibility
		if textRaw, ok := rawMap["text"]; ok {
			var textStr string
			if err := json.Unmarshal(textRaw, &textStr); err == nil {
				c.TextString = &textStr
			} else {
				// Fallback: try as object with text field
				var textObj struct {
					Text string `json:"text"`
				}
				if err := json.Unmarshal(textRaw, &textObj); err == nil {
					c.TextString = &textObj.Text
				}
			}
		}
	case "text":
		// For "text" type, use "text" field
		if textRaw, ok := rawMap["text"]; ok {
			var textStr string
			if err := json.Unmarshal(textRaw, &textStr); err == nil {
				c.TextString = &textStr
			} else {
				// Fallback: try as object with text field
				var textObj struct {
					Text string `json:"text"`
				}
				if err := json.Unmarshal(textRaw, &textObj); err == nil {
					c.TextString = &textObj.Text
				}
			}
		}
	}

	return nil
}
