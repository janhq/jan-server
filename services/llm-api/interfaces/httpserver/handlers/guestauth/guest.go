package guestauth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/llm-api/infrastructure/keycloak"
	"jan-server/services/llm-api/interfaces/httpserver/middlewares"
	"jan-server/services/llm-api/interfaces/httpserver/responses"
)

// GuestHandler handles guest authentication flows.
type GuestHandler struct {
	kc     *keycloak.Client
	logger zerolog.Logger
}

// NewGuestHandler constructs a handler instance.
func NewGuestHandler(kc *keycloak.Client, logger zerolog.Logger) *GuestHandler {
	return &GuestHandler{kc: kc, logger: logger}
}

// CreateGuest handles POST /auth/guest requests.
func (h *GuestHandler) CreateGuest(c *gin.Context) {
	creds, err := h.kc.CreateGuest(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("create guest user")
		c.JSON(http.StatusBadGateway, responses.ErrorResponse{
			Type:      responses.ErrorTypeInternal,
			Code:      "keycloak_error",
			Message:   "failed to provision guest",
			RequestID: middlewares.RequestIDFromContext(c),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user_id":       creds.UserID,
		"username":      creds.Username,
		"principal_id":  creds.PrincipalID,
		"access_token":  creds.Tokens.AccessToken,
		"refresh_token": creds.Tokens.RefreshToken,
		"token_type":    creds.Tokens.TokenType,
		"expires_in":    creds.Tokens.ExpiresIn,
	})
}
