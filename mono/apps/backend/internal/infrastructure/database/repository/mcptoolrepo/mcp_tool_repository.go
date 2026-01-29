package mcptoolrepo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"jan-server/mono/apps/backend/internal/domain/mcptool"
	"jan-server/mono/apps/backend/internal/domain/query"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"
	"jan-server/mono/apps/backend/internal/infrastructure/database/transaction"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// MCPToolGormRepository implements MCPToolRepository using GORM
type MCPToolGormRepository struct {
	db *transaction.Database
}

var _ mcptool.MCPToolRepository = (*MCPToolGormRepository)(nil)

// NewMCPToolGormRepository creates a new GORM-based MCP tool repository
func NewMCPToolGormRepository(db *transaction.Database) mcptool.MCPToolRepository {
	return &MCPToolGormRepository{db: db}
}

// FindByID finds an MCP tool by its internal ID
func (r *MCPToolGormRepository) FindByID(ctx context.Context, id string) (*mcptool.MCPTool, error) {
	var schema dbschema.AdminMCPTool
	tx := r.db.GetTx(ctx)
	if err := tx.Where("id = ?", id).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound, "mcp tool not found", err, "f1a2b3c4-5678-9abc-def0-111111111111")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to find mcp tool", err, "f1a2b3c4-5678-9abc-def0-111111111112")
	}
	return schema.ToDomain(), nil
}

// FindByPublicID finds an MCP tool by its public ID
func (r *MCPToolGormRepository) FindByPublicID(ctx context.Context, publicID string) (*mcptool.MCPTool, error) {
	var schema dbschema.AdminMCPTool
	tx := r.db.GetTx(ctx)
	if err := tx.Where("public_id = ?", publicID).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound, "mcp tool not found", err, "f1a2b3c4-5678-9abc-def0-222222222221")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to find mcp tool", err, "f1a2b3c4-5678-9abc-def0-222222222222")
	}
	return schema.ToDomain(), nil
}

// FindByToolKey finds an MCP tool by its unique tool key
func (r *MCPToolGormRepository) FindByToolKey(ctx context.Context, toolKey string) (*mcptool.MCPTool, error) {
	var schema dbschema.AdminMCPTool
	tx := r.db.GetTx(ctx)
	if err := tx.Where("tool_key = ?", toolKey).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound, "mcp tool not found", err, "f1a2b3c4-5678-9abc-def0-333333333331")
		}
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to find mcp tool", err, "f1a2b3c4-5678-9abc-def0-333333333332")
	}
	return schema.ToDomain(), nil
}

// FindByFilter finds MCP tools matching the given filter
func (r *MCPToolGormRepository) FindByFilter(ctx context.Context, filter mcptool.MCPToolFilter, p *query.Pagination) ([]*mcptool.MCPTool, error) {
	tx := r.db.GetTx(ctx)
	q := tx.Model(&dbschema.AdminMCPTool{})

	// Apply filters
	if filter.ToolKey != nil {
		q = q.Where("tool_key = ?", *filter.ToolKey)
	}
	if filter.Category != nil {
		q = q.Where("category = ?", *filter.Category)
	}
	if filter.IsActive != nil {
		q = q.Where("is_active = ?", *filter.IsActive)
	}
	if filter.Search != nil && *filter.Search != "" {
		searchPattern := "%" + *filter.Search + "%"
		q = q.Where("name ILIKE ? OR description ILIKE ? OR tool_key ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	// Apply pagination
	if p != nil {
		if p.Limit != nil && *p.Limit > 0 {
			q = q.Limit(*p.Limit)
		}
		if p.Offset != nil && *p.Offset > 0 {
			q = q.Offset(*p.Offset)
		}
	}

	// Order by name
	q = q.Order("name ASC")

	var schemas []dbschema.AdminMCPTool
	if err := q.Find(&schemas).Error; err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to find mcp tools", err, "f1a2b3c4-5678-9abc-def0-444444444441")
	}

	tools := make([]*mcptool.MCPTool, 0, len(schemas))
	for _, schema := range schemas {
		tools = append(tools, schema.ToDomain())
	}

	return tools, nil
}

// FindAllActive finds all active MCP tools
func (r *MCPToolGormRepository) FindAllActive(ctx context.Context) ([]*mcptool.MCPTool, error) {
	isActive := true
	return r.FindByFilter(ctx, mcptool.MCPToolFilter{IsActive: &isActive}, nil)
}

// Update updates an existing MCP tool
func (r *MCPToolGormRepository) Update(ctx context.Context, tool *mcptool.MCPTool) error {
	schema := dbschema.NewSchemaAdminMCPTool(tool)
	schema.UpdatedAt = time.Now()

	tx := r.db.GetTx(ctx)
	result := tx.Model(&dbschema.AdminMCPTool{}).
		Where("id = ?", schema.ID).
		Updates(map[string]interface{}{
			"description":         schema.Description,
			"category":            schema.Category,
			"is_active":           schema.IsActive,
			"metadata":            schema.Metadata,
			"disallowed_keywords": schema.DisallowedKeywords,
			"updated_at":          schema.UpdatedAt,
			"updated_by":          schema.UpdatedBy,
		})

	if result.Error != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to update mcp tool", result.Error, "f1a2b3c4-5678-9abc-def0-555555555551")
	}

	if result.RowsAffected == 0 {
		return platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeNotFound, "mcp tool not found", nil, "f1a2b3c4-5678-9abc-def0-555555555552")
	}

	tool.UpdatedAt = schema.UpdatedAt

	return nil
}

// Count returns the count of MCP tools matching the given filter
func (r *MCPToolGormRepository) Count(ctx context.Context, filter mcptool.MCPToolFilter) (int64, error) {
	tx := r.db.GetTx(ctx)
	q := tx.Model(&dbschema.AdminMCPTool{})

	// Apply filters
	if filter.ToolKey != nil {
		q = q.Where("tool_key = ?", *filter.ToolKey)
	}
	if filter.Category != nil {
		q = q.Where("category = ?", *filter.Category)
	}
	if filter.IsActive != nil {
		q = q.Where("is_active = ?", *filter.IsActive)
	}
	if filter.Search != nil && *filter.Search != "" {
		searchPattern := "%" + *filter.Search + "%"
		q = q.Where("name ILIKE ? OR description ILIKE ? OR tool_key ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, platformerrors.NewError(ctx, platformerrors.LayerRepository, platformerrors.ErrorTypeDatabaseError, "failed to count mcp tools", err, "f1a2b3c4-5678-9abc-def0-666666666661")
	}

	return count, nil
}
