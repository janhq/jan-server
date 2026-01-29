package modelprompthandler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"jan-server/mono/apps/backend/internal/application/audit"
	"jan-server/mono/apps/backend/internal/domain/modelprompttemplate"
	"jan-server/mono/apps/backend/internal/domain/prompttemplate"
	middleware "jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// getModelID extracts the model_id from wildcard param (strips leading slash)
func getModelID(c *gin.Context) string {
	modelID := c.Param("model_id")
	// Wildcard params include leading slash, strip it
	return strings.TrimPrefix(modelID, "/")
}

// ModelPromptTemplateHandler handles HTTP requests for model-specific prompt templates
type ModelPromptTemplateHandler struct {
	service  *modelprompttemplate.Service
	validate *validator.Validate
	audit    *audit.AdminAuditLogger
}

// NewModelPromptTemplateHandler creates a new model prompt template handler
func NewModelPromptTemplateHandler(
	service *modelprompttemplate.Service,
	auditLogger *audit.AdminAuditLogger,
) *ModelPromptTemplateHandler {
	return &ModelPromptTemplateHandler{
		service:  service,
		validate: validator.New(validator.WithRequiredStructEnabled()),
		audit:    auditLogger,
	}
}

// ModelPromptTemplateResponse is the API response format for a model prompt template
type ModelPromptTemplateResponse struct {
	ID               string                  `json:"id"`
	ModelCatalogID   string                  `json:"model_catalog_id"`
	TemplateKey      string                  `json:"template_key"`
	PromptTemplateID string                  `json:"prompt_template_id"`
	Priority         int                     `json:"priority"`
	IsActive         bool                    `json:"is_active"`
	CreatedAt        string                  `json:"created_at"`
	UpdatedAt        string                  `json:"updated_at"`
	PromptTemplate   *PromptTemplateResponse `json:"prompt_template,omitempty"`
}

// PromptTemplateResponse is a minimal response for the joined prompt template
type PromptTemplateResponse struct {
	PublicID    string  `json:"public_id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Category    string  `json:"category"`
	TemplateKey string  `json:"template_key"`
	IsActive    bool    `json:"is_active"`
}

// ListResponse is the list response for model prompt templates
type ListResponse struct {
	Data  []ModelPromptTemplateResponse `json:"data"`
	Total int                           `json:"total"`
}

// EffectiveTemplateResponse represents a resolved template with source info
type EffectiveTemplateResponse struct {
	Template *FullPromptTemplateResponse `json:"template"`
	Source   string                      `json:"source"` // "model_specific", "global_default", "hardcoded"
}

// FullPromptTemplateResponse is the full prompt template for effective templates
type FullPromptTemplateResponse struct {
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
}

// EffectiveTemplatesResponse is the response for effective templates endpoint
type EffectiveTemplatesResponse struct {
	Templates map[string]EffectiveTemplateResponse `json:"templates"`
}

// AssignRequest is the request body for assigning a template
type AssignRequest struct {
	TemplateKey      string `json:"template_key" binding:"required"`
	PromptTemplateID string `json:"prompt_template_id" binding:"required"`
	Priority         *int   `json:"priority,omitempty"`
	IsActive         *bool  `json:"is_active,omitempty"`
}

// UpdateRequest is the request body for updating an assignment
type UpdateRequest struct {
	PromptTemplateID *string `json:"prompt_template_id,omitempty"`
	Priority         *int    `json:"priority,omitempty"`
	IsActive         *bool   `json:"is_active,omitempty"`
}

func toResponse(mpt *modelprompttemplate.ModelPromptTemplate) ModelPromptTemplateResponse {
	// When PromptTemplate is preloaded, use its PublicID for the response
	promptTemplateID := mpt.PromptTemplateID
	if mpt.PromptTemplate != nil {
		promptTemplateID = mpt.PromptTemplate.PublicID
	}

	resp := ModelPromptTemplateResponse{
		ID:               mpt.ID,
		ModelCatalogID:   mpt.ModelCatalogID,
		TemplateKey:      mpt.TemplateKey,
		PromptTemplateID: promptTemplateID,
		Priority:         mpt.Priority,
		IsActive:         mpt.IsActive,
		CreatedAt:        mpt.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:        mpt.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if mpt.PromptTemplate != nil {
		resp.PromptTemplate = &PromptTemplateResponse{
			PublicID:    mpt.PromptTemplate.PublicID,
			Name:        mpt.PromptTemplate.Name,
			Description: mpt.PromptTemplate.Description,
			Category:    mpt.PromptTemplate.Category,
			TemplateKey: mpt.PromptTemplate.TemplateKey,
			IsActive:    mpt.PromptTemplate.IsActive,
		}
	}

	return resp
}

func toFullTemplateResponse(pt *prompttemplate.PromptTemplate) *FullPromptTemplateResponse {
	if pt == nil {
		return nil
	}
	return &FullPromptTemplateResponse{
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
	}
}

// List godoc
// @Summary List model prompt template assignments
// @Description Get all prompt template assignments for a model catalog
// @Tags Admin - Model Prompt Templates
// @Accept json
// @Produce json
// @Param model_id path string true "Model Catalog Public ID"
// @Success 200 {object} ListResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/admin/models/prompt-templates/list/{model_id} [get]
func (h *ModelPromptTemplateHandler) List(c *gin.Context) {
	modelID := getModelID(c)
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model_id is required"})
		return
	}

	assignments, err := h.service.ListTemplatesForModel(c.Request.Context(), modelID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	responses := make([]ModelPromptTemplateResponse, 0, len(assignments))
	for _, mpt := range assignments {
		responses = append(responses, toResponse(mpt))
	}

	c.JSON(http.StatusOK, ListResponse{
		Data:  responses,
		Total: len(responses),
	})
}

// Assign godoc
// @Summary Assign a prompt template to a model
// @Description Assign or update a prompt template assignment for a model catalog
// @Tags Admin - Model Prompt Templates
// @Accept json
// @Produce json
// @Param model_id path string true "Model Catalog Public ID"
// @Param body body AssignRequest true "Assignment request"
// @Success 200 {object} ModelPromptTemplateResponse
// @Success 201 {object} ModelPromptTemplateResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/admin/models/prompt-templates/assign/{model_id} [post]
func (h *ModelPromptTemplateHandler) Assign(c *gin.Context) {
	modelID := getModelID(c)
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model_id is required"})
		return
	}

	var req AssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserIDFromContext(c)
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	assignReq := modelprompttemplate.AssignTemplateRequest{
		TemplateKey:      req.TemplateKey,
		PromptTemplateID: req.PromptTemplateID,
		Priority:         req.Priority,
		IsActive:         req.IsActive,
	}

	mpt, err := h.service.AssignTemplate(c.Request.Context(), modelID, assignReq, userIDPtr)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Log audit event
	h.logAudit(c, "model_prompt_template.assign", "model_prompt_template", modelID, map[string]interface{}{
		"model_catalog_id":   modelID,
		"template_key":       req.TemplateKey,
		"prompt_template_id": req.PromptTemplateID,
	}, http.StatusOK, nil)

	c.JSON(http.StatusOK, toResponse(mpt))
}

// Unassign godoc
// @Summary Remove a prompt template assignment from a model
// @Description Remove a prompt template assignment, reverting to global default
// @Tags Admin - Model Prompt Templates
// @Accept json
// @Produce json
// @Param model_id path string true "Model Catalog Public ID"
// @Param template_key path string true "Template Key (e.g., deep_research, timing)"
// @Success 204 "No Content"
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/admin/models/prompt-templates/unassign/{template_key}/{model_id} [delete]
func (h *ModelPromptTemplateHandler) Unassign(c *gin.Context) {
	modelID := getModelID(c)
	templateKey := c.Param("template_key")

	if modelID == "" || templateKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model_id and template_key are required"})
		return
	}

	err := h.service.UnassignTemplate(c.Request.Context(), modelID, templateKey)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Log audit event
	h.logAudit(c, "model_prompt_template.unassign", "model_prompt_template", modelID, map[string]interface{}{
		"model_catalog_id": modelID,
		"template_key":     templateKey,
	}, http.StatusNoContent, nil)

	c.Status(http.StatusNoContent)
}

// GetEffective godoc
// @Summary Get effective templates for a model
// @Description Get all resolved templates for a model, including global defaults
// @Tags Admin - Model Prompt Templates
// @Accept json
// @Produce json
// @Param model_id path string true "Model Catalog Public ID"
// @Success 200 {object} EffectiveTemplatesResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/admin/models/prompt-templates/effective/{model_id} [get]
func (h *ModelPromptTemplateHandler) GetEffective(c *gin.Context) {
	modelID := getModelID(c)
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model_id is required"})
		return
	}

	effective, err := h.service.GetEffectiveTemplates(c.Request.Context(), modelID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response := EffectiveTemplatesResponse{
		Templates: make(map[string]EffectiveTemplateResponse),
	}

	for key, eff := range effective.Templates {
		response.Templates[key] = EffectiveTemplateResponse{
			Template: toFullTemplateResponse(eff.Template),
			Source:   eff.Source,
		}
	}

	c.JSON(http.StatusOK, response)
}

// Update godoc
// @Summary Update a prompt template assignment
// @Description Update an existing prompt template assignment
// @Tags Admin - Model Prompt Templates
// @Accept json
// @Produce json
// @Param model_id path string true "Model Catalog Public ID"
// @Param template_key path string true "Template Key"
// @Param body body UpdateRequest true "Update request"
// @Success 200 {object} ModelPromptTemplateResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/admin/models/prompt-templates/update/{template_key}/{model_id} [patch]
func (h *ModelPromptTemplateHandler) Update(c *gin.Context) {
	modelID := getModelID(c)
	templateKey := c.Param("template_key")

	if modelID == "" || templateKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model_id and template_key are required"})
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserIDFromContext(c)
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	updateReq := modelprompttemplate.UpdateAssignmentRequest{
		PromptTemplateID: req.PromptTemplateID,
		Priority:         req.Priority,
		IsActive:         req.IsActive,
	}

	mpt, err := h.service.UpdateAssignment(c.Request.Context(), modelID, templateKey, updateReq, userIDPtr)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Log audit event
	h.logAudit(c, "model_prompt_template.update", "model_prompt_template", modelID, map[string]interface{}{
		"model_catalog_id": modelID,
		"template_key":     templateKey,
	}, http.StatusOK, nil)

	c.JSON(http.StatusOK, toResponse(mpt))
}

// logAudit logs admin audit events
func (h *ModelPromptTemplateHandler) logAudit(c *gin.Context, action, resourceType, resourceID string, payload any, status int, err error) {
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

// handleError handles errors and returns appropriate HTTP responses
func (h *ModelPromptTemplateHandler) handleError(c *gin.Context, err error) {
	if platformerrors.IsErrorType(err, platformerrors.ErrorTypeNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if platformerrors.IsErrorType(err, platformerrors.ErrorTypeValidation) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if platformerrors.IsErrorType(err, platformerrors.ErrorTypeConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
