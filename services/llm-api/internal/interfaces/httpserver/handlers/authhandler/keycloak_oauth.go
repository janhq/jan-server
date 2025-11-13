package authhandler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"jan-server/services/llm-api/internal/config"
)

// AuthRequest stores PKCE parameters for an authorization request
type AuthRequest struct {
	State        string
	CodeVerifier string
	CreatedAt    time.Time
}

// authRequestStore stores pending authorization requests with TTL cleanup
var (
	authRequests  = &sync.Map{}
	authStoreOnce sync.Once
)

// startAuthRequestCleanup starts a goroutine to clean up expired auth requests
func startAuthRequestCleanup() {
	authStoreOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				now := time.Now()
				authRequests.Range(func(key, value interface{}) bool {
					if req, ok := value.(*AuthRequest); ok {
						if now.Sub(req.CreatedAt) > 10*time.Minute {
							authRequests.Delete(key)
						}
					}
					return true
				})
			}
		}()
	})
}

// generateCodeVerifier generates a cryptographically secure code verifier for PKCE
func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateCodeChallenge generates a code challenge from a verifier using SHA256
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// KeycloakOAuthHandler handles Keycloak OAuth2/OIDC flow
type KeycloakOAuthHandler struct {
	keycloakBaseURL   string // Server-to-server URL (e.g., http://keycloak:8085)
	keycloakPublicURL string // Browser-accessible URL (e.g., http://localhost:8085)
	realm             string
	clientID          string
	clientSecret      string
	redirectURI       string
}

// NewKeycloakOAuthHandler creates a new Keycloak OAuth handler
func NewKeycloakOAuthHandler(
	keycloakBaseURL string,
	keycloakPublicURL string,
	realm string,
	clientID string,
	clientSecret string,
	redirectURI string,
) *KeycloakOAuthHandler {
	// Start background cleanup of expired auth requests
	startAuthRequestCleanup()

	// Default publicURL to baseURL if not provided
	if keycloakPublicURL == "" {
		keycloakPublicURL = keycloakBaseURL
	}

	handler := &KeycloakOAuthHandler{
		keycloakBaseURL:   strings.TrimSuffix(keycloakBaseURL, "/"),
		keycloakPublicURL: strings.TrimSuffix(keycloakPublicURL, "/"),
		realm:             realm,
		clientID:          clientID,
		clientSecret:      clientSecret,
		redirectURI:       redirectURI,
	}

	return handler
} // KeycloakLoginRequest represents the login request
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
// @Description Redirects the user to Keycloak's authorization endpoint to authenticate. Returns the authorization URL for frontend redirection with PKCE.
// @Tags Authentication API
// @Accept json
// @Produce json
// @Param redirect_url query string false "URL to redirect after successful login"
// @Success 200 {object} object{authorization_url=string,state=string} "Authorization URL and state parameter"
// @Failure 500 {object} object{error=string} "Failed to generate state or PKCE parameters"
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

	// Generate PKCE parameters
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate code verifier",
		})
		return
	}

	codeChallenge := generateCodeChallenge(codeVerifier)

	// Store state and code_verifier for later validation in callback
	authRequests.Store(state, &AuthRequest{
		State:        state,
		CodeVerifier: codeVerifier,
		CreatedAt:    time.Now(),
	})

	// Build authorization URL with PKCE using public URL (browser-accessible)
	authURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/auth",
		h.keycloakPublicURL, h.realm)
	// Get redirect URL from query parameter
	redirectURL := c.Query("redirect_url")
	if redirectURL == "" {
		// Default redirect URL if not provided
		redirectURL = "http://localhost:3000/auth/callback"
	}

	params := url.Values{}
	params.Add("client_id", h.clientID)
	params.Add("redirect_uri", redirectURL)
	params.Add("response_type", "code")
	params.Add("scope", "openid profile email")
	params.Add("state", state)
	params.Add("code_challenge", codeChallenge)
	params.Add("code_challenge_method", "S256")

	fullAuthURL := fmt.Sprintf("%s?%s", authURL, params.Encode())

	// Return the authorization URL for frontend to redirect
	c.JSON(http.StatusOK, gin.H{
		"authorization_url": fullAuthURL,
		"state":             state,
	})
}

// HandleCallback godoc
// @Summary Handle Keycloak OAuth2 callback
// @Description Handles the OAuth2 callback from Keycloak, exchanges authorization code for tokens using PKCE
// @Tags Authentication API
// @Accept json
// @Produce json
// @Param code query string true "Authorization code from Keycloak"
// @Param state query string true "State parameter for CSRF protection"
// @Param redirect_url query string false "Frontend URL to redirect after successful authentication"
// @Param error query string false "Error from Keycloak (if authentication failed)"
// @Param error_description query string false "Error description from Keycloak"
// @Success 302 "Redirects to frontend URL with tokens in URL fragment"
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

	// Validate state and retrieve code_verifier from storage
	authRequestVal, ok := authRequests.Load(state)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":        "Invalid state parameter",
			"error_detail": "state not found or expired",
		})
		return
	}

	authRequest, ok := authRequestVal.(*AuthRequest)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid auth request data",
		})
		return
	}

	// Remove used state from storage
	authRequests.Delete(state)

	// Exchange code for tokens using PKCE
	tokenResp, err := h.exchangeCodeForTokens(code, authRequest.CodeVerifier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to exchange code for tokens: %v", err),
		})
		return
	}

	// Get redirect URL from query parameter
	redirectURL := c.Query("redirect_url")
	if redirectURL == "" {
		// Default redirect URL if not provided
		redirectURL = "http://localhost:3000/auth/callback"
	}

	// Parse the redirect URL to append tokens in fragment
	parsedURL, err := url.Parse(redirectURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid redirect URL",
		})
		return
	}

	// Build token fragment for frontend extraction
	fragment := fmt.Sprintf("access_token=%s&refresh_token=%s&expires_in=%d&token_type=%s",
		url.QueryEscape(tokenResp.AccessToken),
		url.QueryEscape(tokenResp.RefreshToken),
		tokenResp.ExpiresIn,
		url.QueryEscape(tokenResp.TokenType),
	)

	if tokenResp.IDToken != "" {
		fragment += fmt.Sprintf("&id_token=%s", url.QueryEscape(tokenResp.IDToken))
	}

	parsedURL.Fragment = fragment

	// Redirect to frontend with tokens in URL fragment
	c.Redirect(http.StatusFound, parsedURL.String())
}

// exchangeCodeForTokens exchanges authorization code for access and refresh tokens using PKCE
func (h *KeycloakOAuthHandler) exchangeCodeForTokens(code string, codeVerifier string) (*KeycloakTokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token",
		h.keycloakBaseURL, h.realm)

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", h.redirectURI)
	data.Set("client_id", h.clientID)
	data.Set("code_verifier", codeVerifier) // PKCE parameter
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
		cfg.KeycloakPublicURL,
		cfg.KeycloakRealm,
		cfg.TargetClientID,
		cfg.BackendClientSecret,
		cfg.OAuthRedirectURI,
	)
}
