package mcptool

import (
	"context"
	"time"

	"jan-server/mono/apps/backend/internal/domain/query"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// Service provides business logic for MCP tool operations
type Service struct {
	repo MCPToolRepository
}

// NewService creates a new MCP tool service
func NewService(repo MCPToolRepository) *Service {
	return &Service{repo: repo}
}

// List retrieves MCP tools based on filter and pagination
func (s *Service) List(ctx context.Context, filter MCPToolFilter, p *query.Pagination) ([]*MCPTool, int64, error) {
	tools, err := s.repo.FindByFilter(ctx, filter, p)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return tools, count, nil
}

// GetByPublicID retrieves an MCP tool by its public ID
func (s *Service) GetByPublicID(ctx context.Context, publicID string) (*MCPTool, error) {
	if publicID == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "public ID is required", nil, "mcp-tool-001")
	}

	tool, err := s.repo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}

	return tool, nil
}

// GetByToolKey retrieves an MCP tool by its tool key
func (s *Service) GetByToolKey(ctx context.Context, toolKey string) (*MCPTool, error) {
	if toolKey == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "tool key is required", nil, "mcp-tool-002")
	}

	tool, err := s.repo.FindByToolKey(ctx, toolKey)
	if err != nil {
		return nil, err
	}

	return tool, nil
}

// GetAllActive retrieves all active MCP tools (for mcp-tools service)
func (s *Service) GetAllActive(ctx context.Context) ([]*MCPTool, error) {
	tools, err := s.repo.FindAllActive(ctx)
	if err != nil {
		return nil, err
	}

	return tools, nil
}

// Update updates an MCP tool configuration
// Note: Name and ToolKey are read-only and cannot be changed
func (s *Service) Update(ctx context.Context, publicID string, req *UpdateMCPToolRequest, updatedBy *string) (*MCPTool, error) {
	if publicID == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "public ID is required", nil, "mcp-tool-003")
	}

	tool, err := s.repo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}

	// Apply updates (Name is read-only, not editable)
	if req.Description != nil {
		tool.Description = *req.Description
	}
	if req.Category != nil {
		tool.Category = *req.Category
	}
	if req.IsActive != nil {
		tool.IsActive = *req.IsActive
	}
	if req.DisallowedKeywords != nil {
		tool.DisallowedKeywords = req.DisallowedKeywords
	}
	if req.Metadata != nil {
		tool.Metadata = req.Metadata
	}

	tool.UpdatedBy = updatedBy
	tool.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, tool); err != nil {
		return nil, err
	}

	return tool, nil
}
