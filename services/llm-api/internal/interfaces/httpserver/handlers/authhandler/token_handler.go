package authhandler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/llm-api/internal/infrastructure/keycloak"
	"jan-server/services/llm-api/internal/interfaces/httpserver/middlewares"
	"jan-server/services/llm-api/internal/interfaces/httpserver/responses"
)

const (
	// RefreshTokenCookieName is the name of the cookie that stores the refresh token
	RefreshTokenCookieName = "refresh_token"
)

// TokenHandler handles token-related operations (logout, refresh, etc.)
type TokenHandler struct {
	kc     *keycloak.Client
	logger zerolog.Logger
}

// NewTokenHandler creates a new token handler
func NewTokenHandler(kc *keycloak.Client, logger zerolog.Logger) *TokenHandler {
	return &TokenHandler{
		kc:     kc,
		logger: logger,
	}
}

type AccessTokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type GetMeResponse struct {
	ID         string `json:"id"`
	Username   string `json:"username,omitempty"`
	Email      string `json:"email,omitempty"`
	Subject    string `json:"subject"`
	AuthMethod string `json:"auth_method"`
}

// Logout removes authentication tokens
// @Summary Logout
// @Description Remove refresh tokens to perform logout
// @Tags Authentication API
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "Successfully logged out"
// @Failure 400 {object} responses.ErrorResponse "Bad Request"
// @Router /v1/auth/logout [get]
func (h *TokenHandler) Logout(c *gin.Context) {
	// Clear the refresh token cookie
	http.SetCookie(c.Writer, responses.NewCookieWithSecurity(
		RefreshTokenCookieName,
		"",
		time.Unix(0, 0), // Set expiration to past time to delete cookie
	))

	c.JSON(http.StatusOK, gin.H{"status": "logged out"})
}

// GetMe returns the current authenticated user's information
// @Summary Get user profile
// @Description Retrieves the profile of the authenticated user
// @Tags Authentication API
// @Security BearerAuth
// @Produce json
// @Success 200 {object} GetMeResponse "Successfully retrieved user profile"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Router /v1/auth/me [get]
func (h *TokenHandler) GetMe(c *gin.Context) {
	principal, ok := middlewares.PrincipalFromContext(c)
	if !ok {
		responses.HandleErrorWithStatus(c, http.StatusUnauthorized, nil, "principal missing")
		return
	}

	c.JSON(http.StatusOK, GetMeResponse{
		ID:         principal.ID,
		Username:   principal.Username,
		Email:      principal.Email,
		Subject:    principal.Subject,
		AuthMethod: string(principal.AuthMethod),
	})
}

// RefreshToken exchanges a refresh token for a new access token
// @Summary Refresh an access token
// @Description Use a valid refresh token to obtain a new access token
// @Tags Authentication API
// @Accept json
// @Produce json
// @Param refresh_token body string false "Refresh token (can also be in Authorization header)"
// @Success 200 {object} AccessTokenResponse "Successfully refreshed the access token"
// @Failure 400 {object} responses.ErrorResponse "Bad Request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Router /v1/auth/refresh-token [post]
func (h *TokenHandler) RefreshToken(c *gin.Context) {
	var payload struct {
		RefreshToken string `json:"refresh_token"`
	}

	// Try to get refresh token from request body
	if err := c.ShouldBindJSON(&payload); err != nil {
		// If not in body, try to get from cookie
		if cookie, err := c.Cookie(RefreshTokenCookieName); err == nil && cookie != "" {
			payload.RefreshToken = cookie
		} else {
			// If not in cookie, try Authorization header
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				responses.HandleErrorWithStatus(c, http.StatusBadRequest, err, "refresh_token required")
				return
			}
			// Remove "Bearer " prefix if present
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				payload.RefreshToken = authHeader[7:]
			} else {
				payload.RefreshToken = authHeader
			}
		}
	}

	if payload.RefreshToken == "" {
		responses.HandleErrorWithStatus(c, http.StatusBadRequest, nil, "refresh_token required")
		return
	}

	// Use Keycloak to refresh the token
	tokens, err := h.kc.RefreshToken(c.Request.Context(), payload.RefreshToken)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to refresh token")
		responses.HandleErrorWithStatus(c, http.StatusUnauthorized, err, "failed to refresh token")
		return
	}

	// Set new refresh token as cookie if we got a new one
	if tokens.RefreshToken != "" {
		expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
		http.SetCookie(c.Writer, responses.NewCookieWithSecurity(
			RefreshTokenCookieName,
			tokens.RefreshToken,
			expiresAt,
		))
	}

	c.JSON(http.StatusOK, AccessTokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresIn,
		ExpiresAt:    time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second),
	})
}
