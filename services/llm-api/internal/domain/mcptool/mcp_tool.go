package mcptool

import (
	"context"
	"time"

	"jan-server/services/llm-api/internal/domain/query"
)

// MCPTool represents an admin-configurable MCP tool definition
type MCPTool struct {
	ID                 string         `json:"id"`
	PublicID           string         `json:"public_id"`
	ToolKey            string         `json:"tool_key"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	Category           string         `json:"category"`
	IsActive           bool           `json:"is_active"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	DisallowedKeywords []string       `json:"disallowed_keywords,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	CreatedBy          *string        `json:"created_by,omitempty"`
	UpdatedBy          *string        `json:"updated_by,omitempty"`
}

// MCPToolFilter for querying tools
type MCPToolFilter struct {
	ToolKey  *string
	Category *string
	IsActive *bool
	Search   *string
}

// UpdateMCPToolRequest for updating a tool
// Note: Name is read-only (not editable by admin), tool_key is the identifier
type UpdateMCPToolRequest struct {
	Description        *string        `json:"description,omitempty"`
	Category           *string        `json:"category,omitempty"`
	IsActive           *bool          `json:"is_active,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	DisallowedKeywords []string       `json:"disallowed_keywords,omitempty"` // Regex patterns
}

// MCPToolRepository defines the data access interface
type MCPToolRepository interface {
	FindByID(ctx context.Context, id string) (*MCPTool, error)
	FindByPublicID(ctx context.Context, publicID string) (*MCPTool, error)
	FindByToolKey(ctx context.Context, toolKey string) (*MCPTool, error)
	FindByFilter(ctx context.Context, filter MCPToolFilter, p *query.Pagination) ([]*MCPTool, error)
	FindAllActive(ctx context.Context) ([]*MCPTool, error)
	Update(ctx context.Context, tool *MCPTool) error
	Count(ctx context.Context, filter MCPToolFilter) (int64, error)
}

// Tool keys (must match serper_mcp.go registrations)
const (
	ToolKeyGoogleSearch    = "google_search"
	ToolKeyScrape          = "scrape"
	ToolKeyFileSearchIndex = "file_search_index"
	ToolKeyFileSearchQuery = "file_search_query"
	ToolKeyPythonExec      = "python_exec"
	ToolKeyMemoryRetrieve  = "memory_retrieve"

	// Agent tools
	ToolKeySlideGenerate  = "slide_generate"
	ToolKeyDeepResearch   = "deep_research"
	ToolKeyArtifactCreate = "artifact_create"
	ToolKeyArtifactUpdate = "artifact_update"
)

// Categories
const (
	CategorySearch        = "search"
	CategoryScrape        = "scrape"
	CategoryFileSearch    = "file_search"
	CategoryCodeExecution = "code_execution"
	CategoryMemory        = "memory"
	CategoryGeneration    = "generation"
	CategoryArtifact      = "artifact"
)
