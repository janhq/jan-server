package prompttemplatehandler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"jan-server/mono/apps/backend/internal/application/audit"
	"jan-server/mono/apps/backend/internal/domain/prompttemplate"
	"jan-server/mono/apps/backend/internal/domain/query"
	middleware "jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// PromptTemplateHandler handles HTTP requests for prompt templates
type PromptTemplateHandler struct {
	service  *prompttemplate.Service
	validate *validator.Validate
	audit    *audit.AdminAuditLogger
}

// NewPromptTemplateHandler creates a new prompt template handler
func NewPromptTemplateHandler(
	service *prompttemplate.Service,
	auditLogger *audit.AdminAuditLogger,
) *PromptTemplateHandler {
	return &PromptTemplateHandler{
		service:  service,
		validate: validator.New(validator.WithRequiredStructEnabled()),
		audit:    auditLogger,
	}
}

// PromptTemplateResponse is the API response format for a prompt template
type PromptTemplateResponse struct {
	ID          string         `json:"id"`
	PublicID    string         `json:"public_id"`
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Category    string         `json:"category"`
	TemplateKey string         `json:"template_key"`
	Content     string         `json:"content"`
	Variables   []string       `json:"variables,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	IsActive    bool           `json:"is_active"`
	IsSystem    bool           `json:"is_system"`
	Version     int            `json:"version"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

// ListResponse is the paginated list response
type ListResponse struct {
	Data  []PromptTemplateResponse `json:"data"`
	Total int64                    `json:"total"`
}

func toResponse(pt *prompttemplate.PromptTemplate) PromptTemplateResponse {
	return PromptTemplateResponse{
		ID:          pt.ID,
		PublicID:    pt.PublicID,
		Name:        pt.Name,
		Description: pt.Description,
		Category:    pt.Category,
		TemplateKey: pt.TemplateKey,
		Content:     pt.Content,
		Variables:   pt.Variables,
		Metadata:    pt.Metadata,
		IsActive:    pt.IsActive,
		IsSystem:    pt.IsSystem,
		Version:     pt.Version,
		CreatedAt:   pt.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   pt.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// List godoc
// @Summary List prompt templates
// @Description Get a paginated list of prompt templates with optional filtering
// @Tags Admin - Prompt Templates
// @Accept json
// @Produce json
// @Param category query string false "Filter by category"
// @Param is_active query boolean false "Filter by active status"
// @Param is_system query boolean false "Filter by system status"
// @Param search query string false "Search in name and description"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} ListResponse
// @Failure 500 {object} map[string]string
// @Router /v1/admin/prompt-templates [get]
func (h *PromptTemplateHandler) List(c *gin.Context) {
	filter := prompttemplate.PromptTemplateFilter{}

	if category := c.Query("category"); category != "" {
		filter.Category = &category
	}
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActive := isActiveStr == "true"
		filter.IsActive = &isActive
	}
	if isSystemStr := c.Query("is_system"); isSystemStr != "" {
		isSystem := isSystemStr == "true"
		filter.IsSystem = &isSystem
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

	templates, total, err := h.service.List(c.Request.Context(), filter, pagination)
	if err != nil {
		h.handleError(c, err)
		return
	}

	responses := make([]PromptTemplateResponse, 0, len(templates))
	for _, pt := range templates {
		responses = append(responses, toResponse(pt))
	}

	c.JSON(http.StatusOK, ListResponse{
		Data:  responses,
		Total: total,
	})
}

// Get godoc
// @Summary Get a prompt template
// @Description Get a prompt template by public ID
// @Tags Admin - Prompt Templates
// @Accept json
// @Produce json
// @Param id path string true "Public ID"
// @Success 200 {object} PromptTemplateResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/admin/prompt-templates/{id} [get]
func (h *PromptTemplateHandler) Get(c *gin.Context) {
	publicID := c.Param("id")

	template, err := h.service.GetByPublicID(c.Request.Context(), publicID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toResponse(template)})
}

// GetByKey godoc
// @Summary Get a prompt template by key
// @Description Get a prompt template by its unique template key (public endpoint)
// @Tags Prompt Templates
// @Accept json
// @Produce json
// @Param key path string true "Template Key"
// @Success 200 {object} PromptTemplateResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/prompt-templates/{key} [get]
func (h *PromptTemplateHandler) GetByKey(c *gin.Context) {
	templateKey := c.Param("key")

	template, err := h.service.GetByKey(c.Request.Context(), templateKey)
	if err != nil {
		h.handleError(c, err)
		return
	}

	if !template.IsActive {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "template not found or inactive"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toResponse(template)})
}

// Create godoc
// @Summary Create a prompt template
// @Description Create a new prompt template
// @Tags Admin - Prompt Templates
// @Accept json
// @Produce json
// @Param body body prompttemplate.CreatePromptTemplateRequest true "Request body"
// @Success 201 {object} PromptTemplateResponse
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/admin/prompt-templates [post]
func (h *PromptTemplateHandler) Create(c *gin.Context) {
	var req prompttemplate.CreatePromptTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}

	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": err.Error()})
		return
	}

	principal, hasPrincipal := middleware.PrincipalFromContext(c)
	var createdBy *string
	if hasPrincipal {
		createdBy = &principal.ID
	}

	template, err := h.service.Create(c.Request.Context(), req, createdBy)
	if err != nil {
		h.handleError(c, err)
		return
	}

	h.logAudit(c, "create_prompt_template", "prompt_template", template.PublicID, req, http.StatusCreated, nil)
	c.JSON(http.StatusCreated, gin.H{"data": toResponse(template)})
}

// Update godoc
// @Summary Update a prompt template
// @Description Update an existing prompt template
// @Tags Admin - Prompt Templates
// @Accept json
// @Produce json
// @Param id path string true "Public ID"
// @Param body body prompttemplate.UpdatePromptTemplateRequest true "Request body"
// @Success 200 {object} PromptTemplateResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/admin/prompt-templates/{id} [patch]
func (h *PromptTemplateHandler) Update(c *gin.Context) {
	publicID := c.Param("id")

	var req prompttemplate.UpdatePromptTemplateRequest
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

	template, err := h.service.Update(c.Request.Context(), publicID, req, updatedBy)
	if err != nil {
		h.handleError(c, err)
		return
	}

	h.logAudit(c, "update_prompt_template", "prompt_template", publicID, req, http.StatusOK, nil)
	c.JSON(http.StatusOK, gin.H{"data": toResponse(template)})
}

// Delete godoc
// @Summary Delete a prompt template
// @Description Delete a prompt template (system templates cannot be deleted)
// @Tags Admin - Prompt Templates
// @Accept json
// @Produce json
// @Param id path string true "Public ID"
// @Success 204 "No Content"
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/admin/prompt-templates/{id} [delete]
func (h *PromptTemplateHandler) Delete(c *gin.Context) {
	publicID := c.Param("id")

	err := h.service.Delete(c.Request.Context(), publicID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	h.logAudit(c, "delete_prompt_template", "prompt_template", publicID, nil, http.StatusNoContent, nil)
	c.Status(http.StatusNoContent)
}

// Duplicate godoc
// @Summary Duplicate a prompt template
// @Description Create a copy of an existing prompt template with a new name (key is auto-generated)
// @Tags Admin - Prompt Templates
// @Accept json
// @Produce json
// @Param id path string true "Public ID"
// @Param body body DuplicateRequest true "Request body"
// @Success 201 {object} PromptTemplateResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/admin/prompt-templates/{id}/duplicate [post]
func (h *PromptTemplateHandler) Duplicate(c *gin.Context) {
	publicID := c.Param("id")

	var req DuplicateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body - new_name is optional
		req = DuplicateRequest{}
	}

	principal, hasPrincipal := middleware.PrincipalFromContext(c)
	var createdBy *string
	if hasPrincipal {
		createdBy = &principal.ID
	}

	template, err := h.service.Duplicate(c.Request.Context(), publicID, req.NewName, createdBy)
	if err != nil {
		h.handleError(c, err)
		return
	}

	h.logAudit(c, "duplicate_prompt_template", "prompt_template", template.PublicID, req, http.StatusCreated, nil)
	c.JSON(http.StatusCreated, gin.H{"data": toResponse(template)})
}

// DuplicateRequest is the request body for duplicating a template
type DuplicateRequest struct {
	NewName string `json:"new_name" validate:"omitempty,max=200"`
}

func (h *PromptTemplateHandler) handleError(c *gin.Context, err error) {
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

func (h *PromptTemplateHandler) logAudit(c *gin.Context, action, resourceType, resourceID string, payload any, status int, err error) {
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
