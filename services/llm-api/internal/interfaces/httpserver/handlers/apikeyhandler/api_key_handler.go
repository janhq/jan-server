package apikeyhandler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/llm-api/internal/domain/apikey"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/responses"
)

// Handler manages API key HTTP endpoints.
type Handler struct {
	service *apikey.Service
	logger  zerolog.Logger
}

// NewHandler constructs a new API key handler.
func NewHandler(service *apikey.Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With().Str("component", "api-key-handler").Logger(),
	}
}

type createRequest struct {
	Name      string         `json:"name" binding:"required"`
	ExpiresIn *time.Duration `json:"expires_in,omitempty"`
	IsSystem  bool           `json:"is_system,omitempty"` // Internal: create as system key
}

type getOrCreateSystemKeyRequest struct {
	ExpiresIn *time.Duration `json:"expires_in,omitempty"`
}

type apiKeyResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Suffix     string     `json:"suffix"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	Status     string     `json:"status"`
	Key        string     `json:"key,omitempty"`
}

// Create issues a new API key for the authenticated user.
// If X-System-Key header is "true", creates a system key (not shown in user's key list).
func (h *Handler) Create(c *gin.Context) {
	user, ok := authhandler.GetUserFromContext(c)
	if !ok {
		responses.HandleErrorWithStatus(c, http.StatusUnauthorized, nil, "user context missing")
		return
	}

	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.HandleErrorWithStatus(c, http.StatusBadRequest, err, "invalid request payload")
		return
	}

	var ttl time.Duration
	if req.ExpiresIn != nil {
		ttl = *req.ExpiresIn
	}

	// Check for X-System-Key header or request body flag
	isSystem := req.IsSystem || c.GetHeader("X-System-Key") == "true"

	var key *apikey.APIKey
	var secret string
	var err error

	if isSystem {
		key, secret, err = h.service.CreateSystemKey(c.Request.Context(), user, req.Name, ttl)
	} else {
		key, secret, err = h.service.CreateKey(c.Request.Context(), user, req.Name, ttl)
	}

	if err != nil {
		h.logger.Error().Err(err).Bool("is_system", isSystem).Msg("failed to create api key")
		if err == apikey.ErrLimitExceeded {
			responses.HandleErrorWithStatus(c, http.StatusBadRequest, err, "api key limit reached")
			return
		}
		responses.HandleError(c, err, "failed to create api key")
		return
	}

	c.JSON(http.StatusCreated, apiKeyResponse{
		ID:        key.ID,
		Name:      key.Name,
		Prefix:    key.Prefix,
		Suffix:    key.Suffix,
		CreatedAt: key.CreatedAt,
		ExpiresAt: key.ExpiresAt,
		Status:    keyStatus(key),
		Key:       secret,
	})
}

// GetOrCreateSystemKey returns an existing active system key or creates a new one.
// This endpoint enables reuse of system keys instead of creating new ones per request.
func (h *Handler) GetOrCreateSystemKey(c *gin.Context) {
	user, ok := authhandler.GetUserFromContext(c)
	if !ok {
		responses.HandleErrorWithStatus(c, http.StatusUnauthorized, nil, "user context missing")
		return
	}

	var req getOrCreateSystemKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
		req = getOrCreateSystemKeyRequest{}
	}

	var ttl time.Duration
	if req.ExpiresIn != nil {
		ttl = *req.ExpiresIn
	} else {
		ttl = time.Hour // Default 1 hour for system keys
	}

	key, secret, err := h.service.GetOrCreateSystemKey(c.Request.Context(), user, ttl)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get or create system key")
		responses.HandleError(c, err, "failed to get or create system key")
		return
	}

	// If secret is empty, we're reusing an existing key
	// In this case, we need to indicate that the key value is not available
	resp := apiKeyResponse{
		ID:        key.ID,
		Name:      key.Name,
		Prefix:    key.Prefix,
		Suffix:    key.Suffix,
		CreatedAt: key.CreatedAt,
		ExpiresAt: key.ExpiresAt,
		Status:    keyStatus(key),
	}

	if secret != "" {
		resp.Key = secret
		c.JSON(http.StatusCreated, resp)
	} else {
		// Reusing existing key - caller needs to use the key they already have
		// or we need a different approach
		c.JSON(http.StatusOK, resp)
	}
}

// List returns API keys for the authenticated user.
func (h *Handler) List(c *gin.Context) {
	user, ok := authhandler.GetUserFromContext(c)
	if !ok {
		responses.HandleErrorWithStatus(c, http.StatusUnauthorized, nil, "user context missing")
		return
	}

	items, err := h.service.ListKeys(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list api keys")
		responses.HandleError(c, err, "failed to list api keys")
		return
	}

	resp := make([]apiKeyResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, apiKeyResponse{
			ID:         item.ID,
			Name:       item.Name,
			Prefix:     item.Prefix,
			Suffix:     item.Suffix,
			CreatedAt:  item.CreatedAt,
			ExpiresAt:  item.ExpiresAt,
			RevokedAt:  item.RevokedAt,
			LastUsedAt: item.LastUsedAt,
			Status:     keyStatus(&item),
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": resp})
}

// Delete revokes the specified API key.
func (h *Handler) Delete(c *gin.Context) {
	user, ok := authhandler.GetUserFromContext(c)
	if !ok {
		responses.HandleErrorWithStatus(c, http.StatusUnauthorized, nil, "user context missing")
		return
	}

	keyID := c.Param("id")
	if keyID == "" || keyID == "null" {
		responses.HandleErrorWithStatus(c, http.StatusBadRequest, nil, "api key id required and must be valid UUID")
		return
	}

	if err := h.service.RevokeKey(c.Request.Context(), user, keyID); err != nil {
		if errors.Is(err, apikey.ErrNotFound) {
			responses.HandleErrorWithStatus(c, http.StatusNotFound, err, "api key not found")
			return
		}
		h.logger.Error().Err(err).Str("key_id", keyID).Msg("failed to revoke api key")
		responses.HandleError(c, err, "failed to revoke api key")
		return
	}

	c.Status(http.StatusNoContent)
}

// Validate validates an API key and returns user information (for Kong plugin)
func (h *Handler) Validate(c *gin.Context) {
	var req struct {
		APIKey string `json:"api_key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		responses.HandleErrorWithStatus(c, http.StatusBadRequest, err, "invalid request payload")
		return
	}

	userInfo, err := h.service.ValidateAPIKey(c.Request.Context(), req.APIKey)
	if err != nil {
		h.logger.Debug().Err(err).Msg("api key validation failed")
		responses.HandleErrorWithStatus(c, http.StatusUnauthorized, err, "invalid api key")
		return
	}

	c.JSON(http.StatusOK, userInfo)
}

func keyStatus(key *apikey.APIKey) string {
	now := time.Now()
	switch {
	case key.RevokedAt != nil:
		return "revoked"
	case now.After(key.ExpiresAt):
		return "expired"
	default:
		return "active"
	}
}
