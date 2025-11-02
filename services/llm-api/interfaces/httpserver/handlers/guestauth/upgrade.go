package guestauth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/llm-api/domain"
	"jan-server/services/llm-api/infrastructure/keycloak"
	"jan-server/services/llm-api/interfaces/httpserver/middlewares"
	"jan-server/services/llm-api/interfaces/httpserver/responses"
)

// UpgradeHandler upgrades guest users to named users.
type UpgradeHandler struct {
	kc     *keycloak.Client
	logger zerolog.Logger
}

// NewUpgradeHandler constructs the handler.
func NewUpgradeHandler(kc *keycloak.Client, logger zerolog.Logger) *UpgradeHandler {
	return &UpgradeHandler{kc: kc, logger: logger}
}

// Upgrade processes POST /auth/upgrade.
func (h *UpgradeHandler) Upgrade(c *gin.Context) {
	principal, ok := middlewares.PrincipalFromContext(c)
	if !ok || principal.ID == "" {
		unauthorized(c)
		return
	}

	var payload keycloak.UpgradePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse{
			Type:      responses.ErrorTypeInvalidRequest,
			Code:      "invalid_payload",
			Message:   err.Error(),
			RequestID: middlewares.RequestIDFromContext(c),
		})
		return
	}

	if err := h.kc.UpgradeUser(c.Request.Context(), subjectFromPrincipal(principal), payload); err != nil {
		h.logger.Error().Err(err).Str("subject", principal.Subject).Msg("upgrade user failed")
		c.JSON(http.StatusBadGateway, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "keycloak_error",
			Message:   "failed to upgrade user",
			RequestID: middlewares.RequestIDFromContext(c),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "upgraded"})
}

func unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, responses.ErrorResponse{
		Type:      responses.ErrorTypeAuth,
		Code:      "unauthorized",
		Message:   "principal missing",
		RequestID: middlewares.RequestIDFromContext(c),
	})
}

func subjectFromPrincipal(p domain.Principal) string {
	if p.Subject != "" {
		return p.Subject
	}
	return p.ID
}
