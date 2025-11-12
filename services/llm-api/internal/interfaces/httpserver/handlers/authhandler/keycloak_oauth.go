package authhandler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"jan-server/services/llm-api/internal/config"
)

// KeycloakOAuthHandler handles Keycloak OAuth2/OIDC flow
type KeycloakOAuthHandler struct {
	keycloakBaseURL string
	realm           string
	clientID        string
	clientSecret    string
	redirectURI     string
	cookieDomain    string // Domain for cookies (extracted from redirectURI)
}

// NewKeycloakOAuthHandler creates a new Keycloak OAuth handler
func NewKeycloakOAuthHandler(
	keycloakBaseURL string,
	realm string,
	clientID string,
	clientSecret string,
	redirectURI string,
) *KeycloakOAuthHandler {
	// Extract cookie domain from redirect URI
	cookieDomain := ""
	if parsedURL, err := url.Parse(redirectURI); err == nil {
		host := parsedURL.Hostname()
		// For production domains like api-gateway-dev.jan.ai, use .jan.ai
		// For localhost, leave empty (defaults to current host)
		if host != "localhost" && host != "127.0.0.1" {
			// Extract root domain (e.g., jan.ai from api-gateway-dev.jan.ai)
			parts := strings.Split(host, ".")
			if len(parts) >= 2 {
				cookieDomain = "." + strings.Join(parts[len(parts)-2:], ".")
			}
		}
	}

	return &KeycloakOAuthHandler{
		keycloakBaseURL: strings.TrimSuffix(keycloakBaseURL, "/"),
		realm:           realm,
		clientID:        clientID,
		clientSecret:    clientSecret,
		redirectURI:     redirectURI,
		cookieDomain:    cookieDomain,
	}
}

// KeycloakLoginRequest represents the login request
type KeycloakLoginRequest struct {
	RedirectURL string `json:"redirect_url" form:"redirect_url"`
}

// KeycloakTokenResponse represents Keycloak token response
type KeycloakTokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	IDToken          string `json:"id_token,omitempty"`
	NotBeforePolicy  int    `json:"not-before-policy,omitempty"`
	SessionState     string `json:"session_state,omitempty"`
	Scope            string `json:"scope,omitempty"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	IDToken      string `json:"id_token,omitempty"`
}

// generateState generates a random state parameter for CSRF protection
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// InitiateLogin godoc
// @Summary Initiate Keycloak OAuth2 login
// @Description Redirects the user to Keycloak's authorization endpoint to authenticate. Returns the authorization URL for frontend redirection.
// @Tags Authentication API
// @Accept json
// @Produce json
// @Param redirect_url query string false "URL to redirect after successful login"
// @Success 200 {object} object{authorization_url=string,state=string} "Authorization URL and state parameter"
// @Failure 500 {object} object{error=string} "Failed to generate state"
// @Router /auth/keycloak/login [get]
func (h *KeycloakOAuthHandler) InitiateLogin(c *gin.Context) {
	// Generate state for CSRF protection
	state, err := generateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate state",
		})
		return
	}

	// Store state in session/cookie for validation later
	// Use SameSite=None with Secure for cross-origin requests
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		MaxAge:   600,
		Path:     "/",
		Domain:   h.cookieDomain,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	})

	// Store redirect URL if provided
	redirectURL := c.Query("redirect_url")
	if redirectURL != "" {
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "post_login_redirect",
			Value:    redirectURL,
			MaxAge:   600,
			Path:     "/",
			Domain:   h.cookieDomain,
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
		})
	}

	// Build authorization URL
	authURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/auth",
		h.keycloakBaseURL, h.realm)

	params := url.Values{}
	params.Add("client_id", h.clientID)
	params.Add("redirect_uri", h.redirectURI)
	params.Add("response_type", "code")
	params.Add("scope", "openid profile email")
	params.Add("state", state)

	fullAuthURL := fmt.Sprintf("%s?%s", authURL, params.Encode())

	// Return the authorization URL for frontend to redirect
	c.JSON(http.StatusOK, gin.H{
		"authorization_url": fullAuthURL,
		"state":             state,
	})
}

// HandleCallback godoc
// @Summary Handle Keycloak OAuth2 callback
// @Description Handles the OAuth2 callback from Keycloak, exchanges authorization code for tokens
// @Tags Authentication API
// @Accept json
// @Produce json
// @Param code query string true "Authorization code from Keycloak"
// @Param state query string true "State parameter for CSRF protection"
// @Param error query string false "Error from Keycloak (if authentication failed)"
// @Param error_description query string false "Error description from Keycloak"
// @Success 200 {object} LoginResponse "JWT tokens"
// @Failure 400 {object} object{error=string} "Missing code or state, or Keycloak error"
// @Failure 401 {object} object{error=string} "Invalid state parameter"
// @Failure 500 {object} object{error=string} "Failed to exchange code for tokens"
// @Router /auth/keycloak/callback [get]
func (h *KeycloakOAuthHandler) HandleCallback(c *gin.Context) {
	// Check for errors from Keycloak first
	keycloakError := c.Query("error")
	if keycloakError != "" {
		errorDescription := c.Query("error_description")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             keycloakError,
			"error_description": errorDescription,
		})
		return
	}

	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing code or state parameter",
		})
		return
	}

	// Validate state
	storedState, err := c.Cookie("oauth_state")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":          "Invalid state parameter",
			"error_detail":   "oauth_state cookie not found",
			"cookie_error":   err.Error(),
			"received_state": state,
		})
		return
	}
	if storedState != state {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":          "Invalid state parameter",
			"error_detail":   "state mismatch",
			"stored_state":   storedState,
			"received_state": state,
		})
		return
	}

	// Clear state cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Domain:   h.cookieDomain,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	})

	// Exchange code for tokens
	tokenResp, err := h.exchangeCodeForTokens(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to exchange code for tokens: %v", err),
		})
		return
	}

	// Get post-login redirect URL
	postLoginRedirect, _ := c.Cookie("post_login_redirect")
	if postLoginRedirect != "" {
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "post_login_redirect",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			Domain:   h.cookieDomain,
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
		})
	}

	// Set tokens as HTTP-only cookies for security
	// Access token cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    tokenResp.AccessToken,
		MaxAge:   tokenResp.ExpiresIn,
		Path:     "/",
		Domain:   h.cookieDomain,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	})

	// Refresh token cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokenResp.RefreshToken,
		MaxAge:   tokenResp.RefreshExpiresIn,
		Path:     "/",
		Domain:   h.cookieDomain,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	})

	// ID token cookie (if present)
	if tokenResp.IDToken != "" {
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "id_token",
			Value:    tokenResp.IDToken,
			MaxAge:   tokenResp.ExpiresIn,
			Path:     "/",
			Domain:   h.cookieDomain,
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
		})
	}

	// If there's a redirect URL, redirect to client's page with tokens
	if postLoginRedirect != "" {
		// Parse the redirect URL to append tokens
		redirectURL, err := url.Parse(postLoginRedirect)
		if err == nil {
			// Use hash fragment for security (tokens won't be sent to server in subsequent requests)
			// Frontend can extract tokens from window.location.hash
			fragment := fmt.Sprintf("access_token=%s&refresh_token=%s&expires_in=%d&token_type=%s",
				url.QueryEscape(tokenResp.AccessToken),
				url.QueryEscape(tokenResp.RefreshToken),
				tokenResp.ExpiresIn,
				url.QueryEscape(tokenResp.TokenType),
			)
			if tokenResp.IDToken != "" {
				fragment += fmt.Sprintf("&id_token=%s", url.QueryEscape(tokenResp.IDToken))
			}
			redirectURL.Fragment = fragment
			c.Redirect(http.StatusFound, redirectURL.String())
			return
		}
		// Fallback if URL parsing fails
		c.Redirect(http.StatusFound, postLoginRedirect)
		return
	}

	// Default: return JSON response for backward compatibility
	response := LoginResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		TokenType:    tokenResp.TokenType,
		IDToken:      tokenResp.IDToken,
	}
	c.JSON(http.StatusOK, response)
}

// exchangeCodeForTokens exchanges authorization code for access and refresh tokens
func (h *KeycloakOAuthHandler) exchangeCodeForTokens(code string) (*KeycloakTokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token",
		h.keycloakBaseURL, h.realm)

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", h.redirectURI)
	data.Set("client_id", h.clientID)
	if h.clientSecret != "" {
		data.Set("client_secret", h.clientSecret)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp KeycloakTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// ProvideKeycloakOAuthHandler provides a KeycloakOAuthHandler for dependency injection
func ProvideKeycloakOAuthHandler(cfg *config.Config) *KeycloakOAuthHandler {
	return NewKeycloakOAuthHandler(
		cfg.KeycloakBaseURL,
		cfg.KeycloakRealm,
		cfg.TargetClientID,
		cfg.BackendClientSecret,
		cfg.OAuthRedirectURI,
	)
}
