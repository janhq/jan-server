package mcptoolhandler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"jan-server/mono/apps/backend/internal/application/audit"
	"jan-server/mono/apps/backend/internal/domain/mcptool"
	"jan-server/mono/apps/backend/internal/domain/query"
	middleware "jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// MCPToolHandler handles HTTP requests for MCP tool administration
type MCPToolHandler struct {
	service  *mcptool.Service
	validate *validator.Validate
	audit    *audit.AdminAuditLogger
}

// NewMCPToolHandler creates a new MCP tool handler
func NewMCPToolHandler(
	service *mcptool.Service,
	auditLogger *audit.AdminAuditLogger,
) *MCPToolHandler {
	return &MCPToolHandler{
		service:  service,
		validate: validator.New(validator.WithRequiredStructEnabled()),
		audit:    auditLogger,
	}
}

// MCPToolResponse is the API response format for an MCP tool
type MCPToolResponse struct {
	ID                 string         `json:"id"`
	PublicID           string         `json:"public_id"`
	ToolKey            string         `json:"tool_key"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	Category           string         `json:"category"`
	IsActive           bool           `json:"is_active"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	DisallowedKeywords []string       `json:"disallowed_keywords,omitempty"`
	CreatedAt          string         `json:"created_at"`
	UpdatedAt          string         `json:"updated_at"`
}

// ListResponse is the paginated list response
type ListResponse struct {
	Data  []MCPToolResponse `json:"data"`
	Total int64             `json:"total"`
}

// ActiveToolsResponse is the response for the public active tools endpoint
type ActiveToolsResponse struct {
	Data []MCPToolResponse `json:"data"`
}

func toResponse(tool *mcptool.MCPTool) MCPToolResponse {
	return MCPToolResponse{
		ID:                 tool.ID,
		PublicID:           tool.PublicID,
		ToolKey:            tool.ToolKey,
		Name:               tool.Name,
		Description:        tool.Description,
		Category:           tool.Category,
		IsActive:           tool.IsActive,
		Metadata:           tool.Metadata,
		DisallowedKeywords: tool.DisallowedKeywords,
		CreatedAt:          tool.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          tool.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// List godoc
// @Summary List MCP tools
// @Description Get a paginated list of MCP tools with optional filtering
// @Tags Admin - MCP Tools
// @Accept json
// @Produce json
// @Param category query string false "Filter by category"
// @Param is_active query boolean false "Filter by active status"
// @Param search query string false "Search in name, description, and tool_key"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} ListResponse
// @Failure 500 {object} map[string]string
// @Router /v1/admin/mcp-tools [get]
func (h *MCPToolHandler) List(c *gin.Context) {
	filter := mcptool.MCPToolFilter{}

	if category := c.Query("category"); category != "" {
		filter.Category = &category
	}
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActive := isActiveStr == "true"
		filter.IsActive = &isActive
	}
	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}

	limit := 20
	offset := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	pagination := &query.Pagination{
		Limit:  &limit,
		Offset: &offset,
	}

	tools, total, err := h.service.List(c.Request.Context(), filter, pagination)
	if err != nil {
		h.handleError(c, err)
		return
	}

	responses := make([]MCPToolResponse, 0, len(tools))
	for _, tool := range tools {
		responses = append(responses, toResponse(tool))
	}

	c.JSON(http.StatusOK, ListResponse{
		Data:  responses,
		Total: total,
	})
}

// Get godoc
// @Summary Get an MCP tool
// @Description Get an MCP tool by public ID
// @Tags Admin - MCP Tools
// @Accept json
// @Produce json
// @Param id path string true "Public ID"
// @Success 200 {object} MCPToolResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/admin/mcp-tools/{id} [get]
func (h *MCPToolHandler) Get(c *gin.Context) {
	publicID := c.Param("id")

	tool, err := h.service.GetByPublicID(c.Request.Context(), publicID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toResponse(tool)})
}

// GetByKey godoc
// @Summary Get an MCP tool by key
// @Description Get an MCP tool by its unique tool key (public endpoint for mcp-tools service)
// @Tags MCP Tools
// @Accept json
// @Produce json
// @Param key path string true "Tool Key"
// @Success 200 {object} MCPToolResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/mcp-tools/{key} [get]
func (h *MCPToolHandler) GetByKey(c *gin.Context) {
	toolKey := c.Param("key")

	tool, err := h.service.GetByToolKey(c.Request.Context(), toolKey)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Return 404 for inactive tools
	if !tool.IsActive {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "tool not found or inactive"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toResponse(tool)})
}

// ListActive godoc
// @Summary List all active MCP tools
// @Description Get all active MCP tools (public endpoint for mcp-tools service)
// @Tags MCP Tools
// @Accept json
// @Produce json
// @Success 200 {object} ActiveToolsResponse
// @Failure 500 {object} map[string]string
// @Router /v1/mcp-tools [get]
func (h *MCPToolHandler) ListActive(c *gin.Context) {
	tools, err := h.service.GetAllActive(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}

	responses := make([]MCPToolResponse, 0, len(tools))
	for _, tool := range tools {
		responses = append(responses, toResponse(tool))
	}

	c.JSON(http.StatusOK, ActiveToolsResponse{
		Data: responses,
	})
}

// Update godoc
// @Summary Update an MCP tool
// @Description Update an existing MCP tool configuration (Name is read-only)
// @Tags Admin - MCP Tools
// @Accept json
// @Produce json
// @Param id path string true "Public ID"
// @Param body body mcptool.UpdateMCPToolRequest true "Request body"
// @Success 200 {object} MCPToolResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/admin/mcp-tools/{id} [patch]
func (h *MCPToolHandler) Update(c *gin.Context) {
	publicID := c.Param("id")

	var req mcptool.UpdateMCPToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": err.Error()})
		return
	}

	principal, hasPrincipal := middleware.PrincipalFromContext(c)
	var updatedBy *string
	if hasPrincipal {
		updatedBy = &principal.ID
	}

	tool, err := h.service.Update(c.Request.Context(), publicID, &req, updatedBy)
	if err != nil {
		h.handleError(c, err)
		return
	}

	h.logAudit(c, "update_mcp_tool", "mcp_tool", publicID, req, http.StatusOK, nil)
	c.JSON(http.StatusOK, gin.H{"data": toResponse(tool)})
}

func (h *MCPToolHandler) handleError(c *gin.Context, err error) {
	if platformerrors.IsErrorType(err, platformerrors.ErrorTypeNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": err.Error()})
		return
	}
	if platformerrors.IsErrorType(err, platformerrors.ErrorTypeConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": "conflict", "message": err.Error()})
		return
	}
	if platformerrors.IsErrorType(err, platformerrors.ErrorTypeForbidden) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": err.Error()})
		return
	}
	if platformerrors.IsErrorType(err, platformerrors.ErrorTypeValidation) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": err.Error()})
}

func (h *MCPToolHandler) logAudit(c *gin.Context, action, resourceType, resourceID string, payload any, status int, err error) {
	if h.audit == nil {
		return
	}
	principal, hasPrincipal := middleware.PrincipalFromContext(c)
	if !hasPrincipal {
		return
	}
	h.audit.Log(c.Request.Context(), audit.AdminAuditEntry{
		AdminUserID: principal.ID,
		AdminEmail:  principal.Email,
		Action:      action,
		Resource:    resourceType,
		ResourceID:  resourceID,
		Payload:     payload,
		StatusCode:  status,
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
		Error:       err,
	})
}
