package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"jan-server/mono/apps/backend/internal/application/audit"
	"jan-server/mono/apps/backend/internal/infrastructure/database/transaction"
	middleware "jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"
)

type FeatureFlagHandler struct {
	db       *transaction.Database
	validate *validator.Validate
	audit    *audit.AdminAuditLogger
}

func NewFeatureFlagHandler(db *transaction.Database, auditLogger *audit.AdminAuditLogger) *FeatureFlagHandler {
	return &FeatureFlagHandler{
		db:       db,
		validate: validator.New(validator.WithRequiredStructEnabled()),
		audit:    auditLogger,
	}
}

type featureFlag struct {
	ID          string     `json:"id"`
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Category    string     `json:"category,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type createFeatureFlagRequest struct {
	Key         string `json:"key" validate:"required,alphanumunicode"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
}

type updateFeatureFlagRequest struct {
	Key         string `json:"key" validate:"omitempty,alphanumunicode"`
	Name        string `json:"name" validate:"omitempty"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
}

func (h *FeatureFlagHandler) ListFeatureFlags(c *gin.Context) {
	var flags []featureFlag
	if err := h.db.GetTx(c.Request.Context()).
		Raw(`SELECT id::text, key, name, description, category, created_at, updated_at FROM llm_api.feature_flags ORDER BY key`).
		Scan(&flags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": flags})
}

func (h *FeatureFlagHandler) CreateFeatureFlag(c *gin.Context) {
	var req createFeatureFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}
	normalizeFeatureFlag(&req)
	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": err.Error()})
		return
	}
	if err := h.db.GetTx(c.Request.Context()).Exec(
		`INSERT INTO llm_api.feature_flags (key, name, description, category) VALUES (?, ?, ?, ?)`,
		req.Key, req.Name, req.Description, req.Category,
	).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}
	h.logAudit(c, "create_feature_flag", "feature_flag", req.Key, req, http.StatusCreated, nil)
	c.Status(http.StatusCreated)
}

func (h *FeatureFlagHandler) UpdateFeatureFlag(c *gin.Context) {
	id := c.Param("id")
	var req updateFeatureFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation_failed", "message": err.Error()})
		return
	}
	if err := h.db.GetTx(c.Request.Context()).Exec(
		`UPDATE llm_api.feature_flags SET key = COALESCE(NULLIF(?, ''), key), name = COALESCE(NULLIF(?, ''), name), description = ?, category = COALESCE(NULLIF(?, ''), category), updated_at = NOW() WHERE id::text = ?`,
		req.Key, req.Name, req.Description, req.Category, id,
	).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}
	h.logAudit(c, "update_feature_flag", "feature_flag", id, req, http.StatusOK, nil)
	c.Status(http.StatusOK)
}

func normalizeFeatureFlag(req *createFeatureFlagRequest) {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	key := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, req.Key)
	if key == "" || strings.Contains(key, "{{") {
		key = "flag" + ts
	}
	req.Key = key
	if strings.TrimSpace(req.Name) == "" || strings.Contains(req.Name, "{{") {
		req.Name = "Flag " + ts
	}
}

func (h *FeatureFlagHandler) DeleteFeatureFlag(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.GetTx(c.Request.Context()).Exec(
		`DELETE FROM llm_api.feature_flags WHERE id::text = ?`, id,
	).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error", "message": err.Error()})
		return
	}
	h.logAudit(c, "delete_feature_flag", "feature_flag", id, nil, http.StatusNoContent, nil)
	c.Status(http.StatusNoContent)
}

func (h *FeatureFlagHandler) logAudit(c *gin.Context, action, resourceType, resourceID string, payload any, status int, err error) {
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
