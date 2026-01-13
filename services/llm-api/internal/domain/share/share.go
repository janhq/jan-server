package share

import (
	"context"
	"time"

	"jan-server/services/llm-api/internal/domain/query"
)

// ===============================================
// Share Types and Enums
// ===============================================

// Visibility represents the visibility level of a share
type Visibility string

const (
	VisibilityUnlisted Visibility = "unlisted" // Only accessible via direct link
	VisibilityPrivate  Visibility = "private"  // Not accessible (future use)
	VisibilityPublic   Visibility = "public"   // Searchable/discoverable (future use)
)

// ShareScope represents what is being shared
type ShareScope string

const (
	ShareScopeConversation ShareScope = "conversation" // Full conversation share
	ShareScopeItem         ShareScope = "item"         // Single message share
)

// ===============================================
// Share Structure
// ===============================================

// Share represents a public share of a conversation or message
type Share struct {
	ID              uint       `json:"-"`
	PublicID        string     `json:"id"`
	Slug            string     `json:"slug"`
	ConversationID  uint       `json:"-"`
	ItemPublicID    *string    `json:"item_id,omitempty"` // For single-message share
	OwnerUserID     uint       `json:"-"`
	Title           *string    `json:"title,omitempty"`
	Visibility      Visibility `json:"visibility"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
	ViewCount       int        `json:"view_count"`
	LastViewedAt    *time.Time `json:"last_viewed_at,omitempty"`
	SnapshotVersion int        `json:"snapshot_version"`
	Snapshot        *Snapshot  `json:"snapshot,omitempty"`
	ShareOptions    *Options   `json:"share_options,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Options contains configurable options for share creation
type Options struct {
	IncludeImages          bool `json:"include_images"`
	IncludeContextMessages bool `json:"include_context_messages"` // For single-message shares
}

// ===============================================
// Snapshot Structure (Public-safe payload)
// ===============================================

// Snapshot contains the sanitized, public-safe data for a share
type Snapshot struct {
	Title         string         `json:"title"`
	ModelName     *string        `json:"model_name,omitempty"`
	AssistantName *string        `json:"assistant_name,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	Items         []SnapshotItem `json:"items"`
}

// SnapshotItem represents a sanitized conversation item for public display
type SnapshotItem struct {
	ID        string            `json:"id"` // Public ID
	Type      string            `json:"type"`
	Role      string            `json:"role"`
	Content   []SnapshotContent `json:"content"`
	CallID    *string           `json:"call_id,omitempty"` // For tool role messages to match with tool_calls
	CreatedAt time.Time         `json:"created_at"`
}

// SnapshotContent represents sanitized content for public display
// Includes safe content types: text, output_text, file references, reasoning, and tool call metadata
type SnapshotContent struct {
	Type        string       `json:"type"`
	Text        string       `json:"text,omitempty"`
	InputText   string       `json:"input_text,omitempty"`
	OutputText  string       `json:"output_text,omitempty"`
	Thinking    string       `json:"thinking,omitempty"`     // For reasoning/thinking content
	ToolCallID  *string      `json:"tool_call_id,omitempty"` // For tool role messages and mcp_call
	ToolCalls   []ToolCall   `json:"tool_calls,omitempty"`   // For assistant tool calls
	MCPCallData string       `json:"mcp_call,omitempty"`     // For MCP tool calls - JSON string directly
	ToolResult  string       `json:"tool_result,omitempty"`  // For tool result content
	Image       *ImageRef    `json:"image,omitempty"`        // For image content
	FileRef     *FileRef     `json:"file_ref,omitempty"`     // For file attachments
	Annotations []Annotation `json:"annotations,omitempty"`
	Reasoning   string       `json:"reasoning_text,omitempty"` // For reasoning/thinking content
}

// ToolCall represents a tool call in the snapshot
type ToolCall struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Function *ToolCallFunction `json:"function,omitempty"`
}

// ToolCallFunction represents the function details in a tool call
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ImageRef represents an image reference in the snapshot
type ImageRef struct {
	URL    string `json:"url,omitempty"`
	FileID string `json:"file_id,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// FileRef represents a reference to a file (image, document) in the snapshot
type FileRef struct {
	FileID   string  `json:"file_id,omitempty"`
	URL      *string `json:"url,omitempty"` // For data URLs or external image URLs
	MimeType *string `json:"mime_type,omitempty"`
	Name     *string `json:"name,omitempty"`
}

// Annotation represents text annotations (citations, links)
type Annotation struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	StartIdx *int   `json:"start_index,omitempty"`
	EndIdx   *int   `json:"end_index,omitempty"`
	URL      string `json:"url,omitempty"`
	FileID   string `json:"file_id,omitempty"`
}

// ===============================================
// Share Filter and Repository
// ===============================================

// ShareFilter defines criteria for querying shares
type ShareFilter struct {
	ID             *uint
	PublicID       *string
	Slug           *string
	ConversationID *uint
	OwnerUserID    *uint
	IncludeRevoked bool
}

// ShareRepository defines the data access interface for shares
type ShareRepository interface {
	// Core CRUD operations
	Create(ctx context.Context, share *Share) error
	FindByFilter(ctx context.Context, filter ShareFilter, pagination *query.Pagination) ([]*Share, error)
	Count(ctx context.Context, filter ShareFilter) (int64, error)
	FindByID(ctx context.Context, id uint) (*Share, error)
	FindByPublicID(ctx context.Context, publicID string) (*Share, error)
	FindBySlug(ctx context.Context, slug string) (*Share, error)
	Update(ctx context.Context, share *Share) error
	Delete(ctx context.Context, id uint) error

	// Specialized operations
	FindActiveByConversationID(ctx context.Context, conversationID uint) ([]*Share, error)
	IncrementViewCount(ctx context.Context, id uint) error
	Revoke(ctx context.Context, id uint) error
	RevokeAllByConversationID(ctx context.Context, conversationID uint) error
	SlugExists(ctx context.Context, slug string) (bool, error)
}

// ===============================================
// Helper Functions
// ===============================================

// IsRevoked returns true if the share has been revoked
func (s *Share) IsRevoked() bool {
	return s.RevokedAt != nil
}

// IsActive returns true if the share is accessible
func (s *Share) IsActive() bool {
	return s.RevokedAt == nil
}

// GetShareURL returns the full URL for the share
func (s *Share) GetShareURL(baseURL string) string {
	return baseURL + "/v1/public/shares/" + s.Slug
}
