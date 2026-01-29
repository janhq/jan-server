package authhandler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/mono/apps/backend/internal/infrastructure/keycloak"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/responses"
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
	ID         string   `json:"id"`
	Username   string   `json:"username,omitempty"`
	Email      string   `json:"email,omitempty"`
	Subject    string   `json:"subject"`
	AuthMethod string   `json:"auth_method"`
	Name       string   `json:"name,omitempty"`
	Roles      []string `json:"roles,omitempty"`
	IsAdmin    bool     `json:"is_admin"`
}

// Logout removes authentication tokens
// @Summary Logout
// @Description Remove refresh tokens to perform logout and invalidate Keycloak session. Accepts refresh token from cookie, Authorization header, or request body.
// @Tags Authentication API
// @Accept json
// @Produce json
// @Param refresh_token body string false "Refresh token to revoke"
// @Param Authorization header string false "Bearer refresh_token"
// @Success 200 {object} map[string]string "Successfully logged out"
// @Failure 400 {object} responses.ErrorResponse "Bad Request"
// @Router /v1/auth/logout [get]
// @Router /v1/auth/logout [post]
func (h *TokenHandler) Logout(c *gin.Context) {
	h.logger.Info().Str("method", c.Request.Method).Str("path", c.Request.URL.Path).Msg("[LOGOUT] Processing logout request")

	// Log all cookies for debugging
	allCookies := []string{}
	for _, cookie := range c.Request.Cookies() {
		allCookies = append(allCookies, cookie.Name)
	}
	h.logger.Info().
		Strs("available_cookies", allCookies).
		Int("cookie_count", len(allCookies)).
		Msg("[LOGOUT] Received cookies")

	var refreshToken string

	// Try to get refresh token from multiple sources in order of priority:
	// 1. From refresh_token cookie (standard flow)
	refreshTokenCookie, err := c.Cookie(RefreshTokenCookieName)
	if err == nil && refreshTokenCookie != "" {
		refreshToken = refreshTokenCookie
		h.logger.Info().
			Str("source", "cookie").
			Str("token_preview", refreshToken[:30]+"...").
			Msg("[LOGOUT] Found refresh token in cookie")
	} else {
		h.logger.Debug().
			Bool("cookie_exists", err == nil).
			Msg("[LOGOUT] No refresh token in cookie")
	}

	// 2. From request body (JSON) - Higher priority than header because body is explicit
	if refreshToken == "" {
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&body); err == nil && body.RefreshToken != "" {
			refreshToken = body.RefreshToken
			h.logger.Info().
				Str("source", "body").
				Str("token_preview", refreshToken[:30]+"...").
				Int("token_length", len(refreshToken)).
				Msg("[LOGOUT] Found refresh token in request body")
		} else {
			h.logger.Debug().
				Bool("bind_success", err == nil).
				Str("bind_error", fmt.Sprintf("%v", err)).
				Msg("[LOGOUT] No refresh token in request body")
		}
	}

	// 3. From Authorization header (Bearer token) - Lower priority, may contain access token
	// NOTE: Authorization header often contains ACCESS TOKEN, not REFRESH TOKEN
	// Only use this as last resort
	if refreshToken == "" {
		authHeader := c.GetHeader("Authorization")
		h.logger.Debug().
			Str("auth_header_preview", func() string {
				if len(authHeader) > 40 {
					return authHeader[:40] + "..."
				}
				return authHeader
			}()).
			Bool("has_bearer", len(authHeader) > 7 && authHeader[:7] == "Bearer ").
			Msg("[LOGOUT] Checking Authorization header")

		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			refreshToken = authHeader[7:]
			h.logger.Warn().
				Str("source", "header").
				Str("token_preview", refreshToken[:30]+"...").
				Int("token_length", len(refreshToken)).
				Msg("[LOGOUT] Using token from Authorization header (may be access token, not refresh token)")
		} else {
			h.logger.Debug().
				Bool("header_exists", authHeader != "").
				Msg("[LOGOUT] No valid Bearer token in Authorization header")
		}
	}

	// Call Keycloak logout endpoint to invalidate the session
	if refreshToken != "" {
		h.logger.Info().
			Str("token_length", fmt.Sprintf("%d", len(refreshToken))).
			Msg("[LOGOUT] Calling Keycloak logout endpoint")

		ctx := c.Request.Context()
		logoutErr := h.kc.LogoutUser(ctx, refreshToken)
		if logoutErr != nil {
			h.logger.Error().
				Err(logoutErr).
				Str("error_type", fmt.Sprintf("%T", logoutErr)).
				Msg("[LOGOUT] Failed to logout from Keycloak, but continuing with local logout")
		} else {
			h.logger.Info().Msg("[LOGOUT] Successfully logged out from Keycloak")
		}
	} else {
		h.logger.Warn().Msg("[LOGOUT] No refresh token found, skipping Keycloak logout")
	}

	// Clear the refresh token cookie locally
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

	// Use username as name if name is empty
	name := principal.Name
	if name == "" {
		name = principal.Username
	}

	// Determine admin status: realm role, client role, or attribute flag
	isAdmin := false
	for _, role := range principal.Roles {
		if strings.EqualFold(role, "admin") {
			isAdmin = true
			break
		}
	}
	if !isAdmin && principal.Attributes != nil {
		if flag, ok := principal.Attributes["is_admin"].(bool); ok && flag {
			isAdmin = true
		}
	}

	c.JSON(http.StatusOK, GetMeResponse{
		ID:         principal.ID,
		Username:   principal.Username,
		Name:       name,
		Email:      principal.Email,
		Subject:    principal.Subject,
		AuthMethod: string(principal.AuthMethod),
		Roles:      principal.Roles,
		IsAdmin:    isAdmin,
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

	// Return tokens in JSON response (token-based authentication, not cookies)
	c.JSON(http.StatusOK, AccessTokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresIn,
		ExpiresAt:    time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second),
	})
}
