package admin

import (
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

type AdminGroupHandler struct {
	kc       *keycloak.Client
	validate *validator.Validate
	audit    *audit.AdminAuditLogger
}

func NewAdminGroupHandler(kc *keycloak.Client, auditLogger *audit.AdminAuditLogger) *AdminGroupHandler {
	return &AdminGroupHandler{
		kc:       kc,
		validate: validator.New(validator.WithRequiredStructEnabled()),
		audit:    auditLogger,
	}
}

type createGroupRequest struct {
	Name string `json:"name" validate:"required"`
}

type updateGroupRequest struct {
	Name       string              `json:"name,omitempty"`
	Attributes map[string][]string `json:"attributes,omitempty"`
}

type setFlagsRequest struct {
	Flags []string `json:"flags" validate:"required,dive,required"`
}

func (h *AdminGroupHandler) ListGroups(c *gin.Context) {
	first, _ := strconv.Atoi(c.DefaultQuery("first", "0"))
	max, _ := strconv.Atoi(c.DefaultQuery("max", "50"))
	groups, err := h.kc.ListGroups(c.Request.Context(), first, max)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": groups})
}

func (h *AdminGroupHandler) GetGroup(c *gin.Context) {
	id := c.Param("id")
	g, err := h.kc.GetGroup(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, g)
}

func (h *AdminGroupHandler) CreateGroup(c *gin.Context) {
	var req createGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.Contains(req.Name, "{{") {
		req.Name = "pilotusers-" + strconv.FormatInt(time.Now().UnixMilli(), 10)
	}
	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": err.Error()})
		return
	}

	id, err := h.kc.CreateGroup(c.Request.Context(), req.Name)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	h.logAudit(c, "create_group", "group", id, req, http.StatusCreated, nil)
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *AdminGroupHandler) UpdateGroup(c *gin.Context) {
	id := c.Param("id")
	var req updateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": err.Error()})
		return
	}

	group, err := h.kc.GetGroup(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	name := group.Name
	if strings.TrimSpace(req.Name) != "" {
		name = req.Name
	}

	attrs := group.Attributes
	if req.Attributes != nil {
		attrs = make(map[string][]any, len(req.Attributes))
		for k, v := range req.Attributes {
			arr := make([]any, 0, len(v))
			for _, s := range v {
				arr = append(arr, s)
			}
			attrs[k] = arr
		}
	}

	if err := h.kc.UpdateGroup(c.Request.Context(), id, name, attrs); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	h.logAudit(c, "update_group", "group", id, req, http.StatusOK, nil)
	c.Status(http.StatusOK)
}

func (h *AdminGroupHandler) DeleteGroup(c *gin.Context) {
	id := c.Param("id")
	if err := h.kc.DeleteGroup(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	h.logAudit(c, "delete_group", "group", id, nil, http.StatusNoContent, nil)
	c.Status(http.StatusNoContent)
}

func (h *AdminGroupHandler) GetGroupMembers(c *gin.Context) {
	id := c.Param("id")
	first, _ := strconv.Atoi(c.DefaultQuery("first", "0"))
	max, _ := strconv.Atoi(c.DefaultQuery("max", "50"))
	members, err := h.kc.ListGroupMembers(c.Request.Context(), id, first, max)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": members})
}

func (h *AdminGroupHandler) AddUserToGroup(c *gin.Context) {
	userID := c.Param("id")
	groupID := c.Param("groupId")
	if err := h.kc.AddUserToGroup(c.Request.Context(), userID, groupID); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	h.logAudit(c, "add_user_to_group", "group", groupID, gin.H{"user_id": userID}, http.StatusOK, nil)
	c.Status(http.StatusOK)
}

func (h *AdminGroupHandler) RemoveUserFromGroup(c *gin.Context) {
	userID := c.Param("id")
	groupID := c.Param("groupId")
	if err := h.kc.RemoveUserFromGroup(c.Request.Context(), userID, groupID); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	h.logAudit(c, "remove_user_from_group", "group", groupID, gin.H{"user_id": userID}, http.StatusOK, nil)
	c.Status(http.StatusOK)
}

func (h *AdminGroupHandler) GetGroupFeatureFlags(c *gin.Context) {
	groupID := c.Param("id")
	flags, err := h.kc.GetGroupFeatureFlags(c.Request.Context(), groupID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"flags": flags})
}

func (h *AdminGroupHandler) SetGroupFeatureFlags(c *gin.Context) {
	groupID := c.Param("id")
	var req setFlagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": err.Error()})
		return
	}
	if err := h.kc.SetGroupFeatureFlags(c.Request.Context(), groupID, req.Flags); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	h.logAudit(c, "set_group_feature_flags", "group", groupID, req, http.StatusOK, nil)
	c.Status(http.StatusOK)
}

func (h *AdminGroupHandler) EnableGroupFeatureFlag(c *gin.Context) {
	groupID := c.Param("id")
	flag := strings.TrimSpace(c.Param("flagKey"))
	if flag == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": "flag required"})
		return
	}
	flags, err := h.kc.GetGroupFeatureFlags(c.Request.Context(), groupID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	if !contains(flags, flag) {
		flags = append(flags, flag)
	}
	if err := h.kc.SetGroupFeatureFlags(c.Request.Context(), groupID, flags); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	h.logAudit(c, "enable_group_feature_flag", "group", groupID, gin.H{"flag": flag}, http.StatusOK, nil)
	c.Status(http.StatusOK)
}

func (h *AdminGroupHandler) DisableGroupFeatureFlag(c *gin.Context) {
	groupID := c.Param("id")
	flag := strings.TrimSpace(c.Param("flagKey"))
	if flag == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": "flag required"})
		return
	}
	flags, err := h.kc.GetGroupFeatureFlags(c.Request.Context(), groupID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	filtered := make([]string, 0, len(flags))
	for _, f := range flags {
		if f != flag {
			filtered = append(filtered, f)
		}
	}
	if err := h.kc.SetGroupFeatureFlags(c.Request.Context(), groupID, filtered); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "keycloak_error", "message": err.Error()})
		return
	}
	h.logAudit(c, "disable_group_feature_flag", "group", groupID, gin.H{"flag": flag}, http.StatusOK, nil)
	c.Status(http.StatusOK)
}

func contains(list []string, needle string) bool {
	for _, item := range list {
		if item == needle {
			return true
		}
	}
	return false
}

func (h *AdminGroupHandler) logAudit(c *gin.Context, action, resourceType, resourceID string, payload any, status int, err error) {
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
