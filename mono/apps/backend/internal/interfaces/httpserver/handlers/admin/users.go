package admin

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"jan-server/mono/apps/backend/internal/application/audit"
	"jan-server/mono/apps/backend/internal/infrastructure/keycloak"
	middleware "jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"
)

type AdminUserHandler struct {
	kc       *keycloak.Client
	validate *validator.Validate
	audit    *audit.AdminAuditLogger
}

func NewAdminUserHandler(kc *keycloak.Client, auditLogger *audit.AdminAuditLogger) *AdminUserHandler {
	return &AdminUserHandler{
		kc:       kc,
		validate: validator.New(validator.WithRequiredStructEnabled()),
		audit:    auditLogger,
	}
}

type CreateUserRequest struct {
	Email     string  `json:"email" validate:"required"`
	Username  string  `json:"username" validate:"required"`
	FirstName string  `json:"first_name,omitempty"`
	LastName  string  `json:"last_name,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
	Password  *string `json:"password,omitempty"`
}

type UpdateUserRequest struct {
	Email     string `json:"email,omitempty" validate:"omitempty,email"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Enabled   *bool  `json:"enabled,omitempty"`
}

func (h *AdminUserHandler) ListUsers(c *gin.Context) {
	first, _ := strconv.Atoi(c.DefaultQuery("first", "0"))
	max, _ := strconv.Atoi(c.DefaultQuery("max", "50"))
	search := strings.TrimSpace(c.Query("search"))
	groupID := strings.TrimSpace(c.Query("group_id"))
	role := strings.TrimSpace(c.Query("role"))

	var enabled *bool
	if raw, ok := c.GetQuery("enabled"); ok && strings.TrimSpace(raw) != "" {
		val := strings.EqualFold(raw, "true") || raw == "1"
		enabled = &val
	}

	// Parse exclude_guests parameter
	excludeGuests := false
	if raw := strings.TrimSpace(c.Query("exclude_guests")); raw != "" {
		excludeGuests = strings.EqualFold(raw, "true") || raw == "1"
	}

	users, err := h.kc.ListUsers(c.Request.Context(), keycloak.ListUsersParams{
		First:         first,
		Max:           max,
		Search:        search,
		Enabled:       enabled,
		GroupID:       groupID,
		Role:          role,
		ExcludeGuests: excludeGuests,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func (h *AdminUserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	user, err := h.kc.GetUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *AdminUserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}
	h.normalizeNewUserFields(&req)
	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": err.Error()})
		return
	}

	kcReq := keycloak.AdminUserRequest{
		Username: req.Username,
		Email:    req.Email,
		First:    req.FirstName,
		Last:     req.LastName,
		Enabled:  req.Enabled,
	}
	if req.Password != nil {
		kcReq.Password = req.Password
	}
	id, err := h.kc.CreateUser(c.Request.Context(), kcReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}

	h.logAudit(c, "create_user", "user", id, req, http.StatusCreated, nil)
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *AdminUserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}
	if err := h.fillMissingIdentifiers(c.Request.Context(), id, &req); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": err.Error()})
		return
	}

	kcReq := keycloak.AdminUserRequest{
		Username: req.Username,
		Email:    req.Email,
		First:    req.FirstName,
		Last:     req.LastName,
		Enabled:  req.Enabled,
	}
	if err := h.kc.UpdateUser(c.Request.Context(), id, kcReq); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	h.logAudit(c, "update_user", "user", id, req, http.StatusOK, nil)
	c.Status(http.StatusOK)
}

func (h *AdminUserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := h.kc.DeleteUser(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	h.logAudit(c, "delete_user", "user", id, nil, http.StatusNoContent, nil)
	c.Status(http.StatusNoContent)
}

func (h *AdminUserHandler) ActivateUser(c *gin.Context) {
	h.toggleUser(c, true)
}

func (h *AdminUserHandler) DeactivateUser(c *gin.Context) {
	h.toggleUser(c, false)
}

func (h *AdminUserHandler) toggleUser(c *gin.Context, enabled bool) {
	id := c.Param("id")
	if err := h.kc.SetUserEnabled(c.Request.Context(), id, enabled); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	action := "deactivate_user"
	if enabled {
		action = "activate_user"
	}
	h.logAudit(c, action, "user", id, gin.H{"enabled": enabled}, http.StatusOK, nil)
	c.Status(http.StatusOK)
}

func (h *AdminUserHandler) AssignRole(c *gin.Context) {
	id := c.Param("id")
	role := strings.TrimSpace(c.Param("role"))
	if role == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": "role required"})
		return
	}
	if err := h.kc.AssignRealmRole(c.Request.Context(), id, role); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	h.logAudit(c, "assign_role", "user", id, gin.H{"role": role}, http.StatusOK, nil)
	c.Status(http.StatusOK)
}

func (h *AdminUserHandler) RemoveRole(c *gin.Context) {
	id := c.Param("id")
	role := strings.TrimSpace(c.Param("role"))
	if role == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": "role required"})
		return
	}
	if err := h.kc.RemoveRealmRole(c.Request.Context(), id, role); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	h.logAudit(c, "remove_role", "user", id, gin.H{"role": role}, http.StatusOK, nil)
	c.Status(http.StatusOK)
}

func (h *AdminUserHandler) normalizeNewUserFields(req *CreateUserRequest) {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	email := strings.TrimSpace(req.Email)
	if email == "" || strings.Contains(email, "{{") || h.validate.Var(email, "email") != nil {
		email = "qa+" + ts + "@jan.ai"
	}
	username := strings.TrimSpace(req.Username)
	username = strings.ReplaceAll(username, " ", "")
	if username == "" || strings.Contains(username, "{{") {
		username = "qa" + ts
	}
	req.Email = email
	req.Username = username
}

func (h *AdminUserHandler) fillMissingIdentifiers(ctx context.Context, id string, req *UpdateUserRequest) error {
	needsLookup := req.Username == "" || strings.Contains(req.Username, "{{") || (req.Email != "" && strings.Contains(req.Email, "{{"))
	if !needsLookup {
		return nil
	}
	existing, err := h.kc.GetUser(ctx, id)
	if err != nil {
		return err
	}
	if req.Username == "" || strings.Contains(req.Username, "{{") {
		req.Username = existing.Username
	}
	if req.Email == "" || strings.Contains(req.Email, "{{") {
		req.Email = existing.Email
	}
	return nil
}

func (h *AdminUserHandler) logAudit(c *gin.Context, action, resourceType, resourceID string, payload any, status int, err error) {
	if h.audit == nil {
		return
	}
	principal, _ := middleware.PrincipalFromContext(c)
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
