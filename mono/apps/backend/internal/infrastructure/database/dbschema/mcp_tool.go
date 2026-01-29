package dbschema

import (
	"time"

	"github.com/lib/pq"

	"jan-server/mono/apps/backend/internal/domain/mcptool"
	"jan-server/mono/apps/backend/internal/infrastructure/database/registry"
)

func init() {
	registry.RegisterSchemaForAutoMigrate(AdminMCPTool{})
}

// AdminMCPTool represents the database schema for admin MCP tool configurations
type AdminMCPTool struct {
	ID                 string         `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	PublicID           string         `gorm:"column:public_id;size:50;not null;uniqueIndex"`
	ToolKey            string         `gorm:"column:tool_key;size:100;not null;uniqueIndex"`
	Name               string         `gorm:"column:name;size:255;not null"`
	Description        string         `gorm:"column:description;type:text;not null"`
	Category           string         `gorm:"column:category;size:100;not null;index"`
	IsActive           bool           `gorm:"column:is_active;default:true;index"`
	Metadata           *string        `gorm:"column:metadata;type:jsonb"`
	DisallowedKeywords pq.StringArray `gorm:"column:disallowed_keywords;type:text[]"`
	CreatedAt          time.Time      `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;not null;default:now()"`
	CreatedBy          *string        `gorm:"column:created_by;type:uuid"`
	UpdatedBy          *string        `gorm:"column:updated_by;type:uuid"`
}

// TableName returns the table name for GORM
func (AdminMCPTool) TableName() string {
	return "llm_api.admin_mcp_tools"
}

// ToDomain converts a database schema AdminMCPTool to a domain model
func (t *AdminMCPTool) ToDomain() *mcptool.MCPTool {
	var metadata map[string]any
	// Note: Metadata JSON parsing can be added if needed in future

	return &mcptool.MCPTool{
		ID:                 t.ID,
		PublicID:           t.PublicID,
		ToolKey:            t.ToolKey,
		Name:               t.Name,
		Description:        t.Description,
		Category:           t.Category,
		IsActive:           t.IsActive,
		Metadata:           metadata,
		DisallowedKeywords: []string(t.DisallowedKeywords),
		CreatedAt:          t.CreatedAt,
		UpdatedAt:          t.UpdatedAt,
		CreatedBy:          t.CreatedBy,
		UpdatedBy:          t.UpdatedBy,
	}
}

// NewSchemaAdminMCPTool converts a domain MCPTool to a database schema
func NewSchemaAdminMCPTool(tool *mcptool.MCPTool) *AdminMCPTool {
	return &AdminMCPTool{
		ID:                 tool.ID,
		PublicID:           tool.PublicID,
		ToolKey:            tool.ToolKey,
		Name:               tool.Name,
		Description:        tool.Description,
		Category:           tool.Category,
		IsActive:           tool.IsActive,
		Metadata:           nil, // JSON serialization can be added if needed
		DisallowedKeywords: pq.StringArray(tool.DisallowedKeywords),
		CreatedAt:          tool.CreatedAt,
		UpdatedAt:          tool.UpdatedAt,
		CreatedBy:          tool.CreatedBy,
		UpdatedBy:          tool.UpdatedBy,
	}
}
