package guestauth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/llm-api/internal/domain"
	"jan-server/services/llm-api/internal/infrastructure/keycloak"
	"jan-server/services/llm-api/internal/interfaces/httpserver/middlewares"
	"jan-server/services/llm-api/internal/interfaces/httpserver/responses"
)

const (
	// RefreshTokenCookieName is the name of the cookie that stores the refresh token
	RefreshTokenCookieName = "refresh_token"
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

// UpgradeHandler handles user upgrade flows.
type UpgradeHandler struct {
	kc     *keycloak.Client
	logger zerolog.Logger
}

// NewUpgradeHandler constructs an upgrade handler instance.
func NewUpgradeHandler(kc *keycloak.Client, logger zerolog.Logger) *UpgradeHandler {
	return &UpgradeHandler{kc: kc, logger: logger}
}

// CreateGuest handles POST /auth/guest-login requests.
func (h *GuestHandler) CreateGuest(c *gin.Context) {
	creds, err := h.kc.CreateGuest(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("create guest user")
		responses.HandleErrorWithStatus(c, http.StatusBadGateway, err, "failed to provision guest")

		return
	}

	// Set refresh token as an HTTP-only cookie
	if creds.Tokens.RefreshToken != "" {
		// Calculate cookie expiration based on token expires_in (in seconds)
		expiresAt := time.Now().Add(time.Duration(creds.Tokens.ExpiresIn) * time.Second)

		// Set the cookie with security settings
		http.SetCookie(c.Writer, responses.NewCookieWithSecurity(
			RefreshTokenCookieName,
			creds.Tokens.RefreshToken,
			expiresAt,
		))
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

// Upgrade processes POST /auth/upgrade.
func (h *UpgradeHandler) Upgrade(c *gin.Context) {
	principal, ok := middlewares.PrincipalFromContext(c)
	if !ok || principal.ID == "" {
		responses.HandleErrorWithStatus(c, http.StatusUnauthorized, nil, "principal missing")
		return
	}

	var payload keycloak.UpgradePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		responses.HandleErrorWithStatus(c, http.StatusBadRequest, err, "invalid payload")

		return
	}

	if err := h.kc.UpgradeUser(c.Request.Context(), subjectFromPrincipal(principal), payload); err != nil {
		h.logger.Error().Err(err).Str("subject", principal.Subject).Msg("upgrade user failed")
		responses.HandleErrorWithStatus(c, http.StatusBadGateway, err, "failed to upgrade user")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "upgraded"})
}

func subjectFromPrincipal(p domain.Principal) string {
	if p.Subject != "" {
		return p.Subject
	}
	return p.ID
}
