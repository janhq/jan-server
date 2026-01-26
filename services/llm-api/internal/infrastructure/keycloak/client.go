package keycloak

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Client wraps interactions with the Keycloak Admin and Token APIs.
type Client struct {
	baseURL             string
	realm               string
	backendClientID     string
	backendClientSecret string
	clientID            string
	guestRole           string
	httpClient          *http.Client
	logger              zerolog.Logger
	adminUsername       string
	adminPassword       string
	adminRealm          string
	adminClientID       string
	adminClientSecret   string
}

// NewClient constructs a Keycloak client.
func NewClient(baseURL, realm, backendClientID, backendClientSecret, clientID, guestRole string, httpClient *http.Client, logger zerolog.Logger, adminUsername, adminPassword, adminRealm, adminClientID, adminClientSecret string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:             strings.TrimRight(baseURL, "/"),
		realm:               realm,
		backendClientID:     backendClientID,
		backendClientSecret: backendClientSecret,
		clientID:            clientID,
		guestRole:           guestRole,
		httpClient:          httpClient,
		logger:              logger,
		adminUsername:       adminUsername,
		adminPassword:       adminPassword,
		adminRealm:          adminRealm,
		adminClientID:       adminClientID,
		adminClientSecret:   adminClientSecret,
	}
}

// TokenSet bundles token information returned by Keycloak.
type TokenSet struct {
	AccessToken     string `json:"access_token"`
	RefreshToken    string `json:"refresh_token,omitempty"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in"`
	IssuedTokenType string `json:"issued_token_type,omitempty"`
	Scope           string `json:"scope,omitempty"`
}

// GuestCredentials represents the result of creating a guest user.
type GuestCredentials struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Email       string   `json:"email,omitempty"`
	PrincipalID string   `json:"pid"`
	Tokens      TokenSet `json:"tokens"`
}

// UpgradePayload describes the upgrade request body.
type UpgradePayload struct {
	Username string `json:"username"`
	Email    string `json:"email" binding:"required,email"` // Required to overwrite temporary email
	FullName string `json:"full_name"`
}

// CreateGuest provisions a new guest user and returns impersonated tokens.
func (c *Client) CreateGuest(ctx context.Context) (*GuestCredentials, error) {
	if c.adminUsername != "" && c.adminPassword != "" {
		adminToken, err := c.adminUserToken(ctx)
		if err != nil {
			c.logger.Warn().
				Err(err).
				Msg("admin credentials present but password grant failed, falling back to service account")
		} else {
			return c.createGuestWithPasswordGrant(ctx, adminToken.AccessToken)
		}
	}

	serviceToken, err := c.serviceAccountToken(ctx)
	if err != nil {
		return nil, err
	}

	adminToken := c.adminAccessToken(ctx, serviceToken.AccessToken)
	return c.createGuestWithPasswordGrant(ctx, adminToken)
}

func (c *Client) createGuestWithPasswordGrant(ctx context.Context, adminToken string) (*GuestCredentials, error) {
	user, err := c.createGuestUser(ctx, adminToken)
	if err != nil {
		return nil, err
	}

	if err := c.assignGuestRole(ctx, adminToken, user.UserID); err != nil {
		return nil, err
	}

	password := strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := c.setUserPassword(ctx, adminToken, user.UserID, password); err != nil {
		return nil, err
	}

	tokens, err := c.passwordGrantTokens(ctx, user.Email, password)
	if err != nil {
		return nil, err
	}

	user.Tokens = *tokens
	return user, nil
}

// UpgradeUser toggles the guest attribute off and updates profile fields.
func (c *Client) UpgradeUser(ctx context.Context, userID string, payload UpgradePayload) error {
	serviceToken, err := c.serviceAccountToken(ctx)
	if err != nil {
		return err
	}

	adminToken := c.adminAccessToken(ctx, serviceToken.AccessToken)

	existing, err := c.getUser(ctx, adminToken, userID)
	if err != nil {
		return err
	}

	attributes := map[string][]string{}
	if raw, ok := existing["attributes"].(map[string]any); ok {
		for key, value := range raw {
			switch v := value.(type) {
			case []any:
				var out []string
				for _, item := range v {
					if s, ok := item.(string); ok {
						out = append(out, s)
					}
				}
				if len(out) > 0 {
					attributes[key] = out
				}
			}
		}
	}
	attributes["guest"] = []string{"false"}

	// Note: username is read-only by default in Keycloak after user creation
	// Only update email, firstName, and attributes to avoid "error-user-attribute-read-only"
	// When upgrading, we overwrite the temporary email (e.g., guest-xxx@temp.jan.ai) with the real email
	update := map[string]any{
		"attributes":    attributes,
		"email":         payload.Email,
		"emailVerified": true, // Mark email as verified when upgrading from guest
		"firstName":     payload.FullName,
		"enabled":       true,
	}

	body, err := json.Marshal(update)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.adminEndpoint("/users/"+url.PathEscape(userID)), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("update user failed: %s", strings.TrimSpace(string(payload)))
	}

	return nil
}

func (c *Client) serviceAccountToken(ctx context.Context) (*TokenSet, error) {
	values := url.Values{}
	values.Set("grant_type", "client_credentials")
	values.Set("client_id", c.backendClientID)
	if c.backendClientSecret != "" {
		values.Set("client_secret", c.backendClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenEndpoint(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("service account token request failed: %s", strings.TrimSpace(string(payload)))
	}

	var token TokenSet
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

func (c *Client) adminUserToken(ctx context.Context) (*TokenSet, error) {
	if c.adminUsername == "" || c.adminPassword == "" {
		return nil, errors.New("admin credentials required")
	}

	realm := c.adminRealm
	if realm == "" {
		realm = "master"
	}

	clientID := c.adminClientID
	if clientID == "" {
		clientID = "admin-cli"
	}

	values := url.Values{}
	values.Set("grant_type", "password")
	values.Set("client_id", clientID)
	if c.adminClientSecret != "" {
		values.Set("client_secret", c.adminClientSecret)
	}
	values.Set("username", c.adminUsername)
	values.Set("password", c.adminPassword)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.realmTokenEndpoint(realm), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("admin token request failed: %s", strings.TrimSpace(string(payload)))
	}

	var token TokenSet
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

// TokenForUser exchanges admin privileges for a user-scoped token.
func (c *Client) TokenForUser(ctx context.Context, userID string) (*TokenSet, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id required")
	}
	adminToken, err := c.adminUserToken(ctx)
	if err != nil {
		return nil, err
	}
	return c.exchangeForUser(ctx, adminToken.AccessToken, userID)
}

func (c *Client) adminAccessToken(ctx context.Context, serviceToken string) string {
	if c.adminUsername == "" || c.adminPassword == "" {
		return serviceToken
	}

	adminToken, err := c.adminUserToken(ctx)
	if err != nil {
		c.logger.Warn().Err(err).Msg("fallback to service account token for admin operations")
		return serviceToken
	}
	return adminToken.AccessToken
}

func (c *Client) createGuestUser(ctx context.Context, adminToken string) (*GuestCredentials, error) {
	username := "guest-" + uuid.NewString()
	// Generate temporary email for guest user (required by Keycloak when duplicateEmailsAllowed is false)
	// Format: guest-{uuid}@temp.jan.ai to clearly identify as temporary
	tempEmail := username + "@temp.jan.ai"

	userPayload := map[string]any{
		"username":   username,
		"email":      tempEmail,
		"enabled":    true,
		"attributes": map[string][]string{"guest": {"true"}},
	}

	body, err := json.Marshal(userPayload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.adminEndpoint("/users"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("create user failed: %s", strings.TrimSpace(string(payload)))
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return nil, errors.New("create user succeeded but location header missing")
	}
	userID := extractIDFromLocation(location)
	if userID == "" {
		return nil, errors.New("create user succeeded but failed to parse user id")
	}

	return &GuestCredentials{
		UserID:      userID,
		Username:    username,
		Email:       tempEmail,
		PrincipalID: userID,
	}, nil
}

func (c *Client) assignGuestRole(ctx context.Context, adminToken, userID string) error {
	role, err := c.getRealmRole(ctx, adminToken, c.guestRole)
	if err != nil {
		return err
	}

	body, err := json.Marshal([]map[string]any{role})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.adminEndpoint(fmt.Sprintf("/users/%s/role-mappings/realm", url.PathEscape(userID))), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("assign role failed: %s", strings.TrimSpace(string(payload)))
	}

	return nil
}

func (c *Client) setUserPassword(ctx context.Context, adminToken, userID, password string) error {
	payload := map[string]any{
		"type":      "password",
		"value":     password,
		"temporary": false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.adminEndpoint(fmt.Sprintf("/users/%s/reset-password", url.PathEscape(userID))), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("set password failed: %s", strings.TrimSpace(string(payload)))
	}

	return nil
}

func (c *Client) passwordGrantTokens(ctx context.Context, email, password string) (*TokenSet, error) {
	values := url.Values{}
	values.Set("grant_type", "password")
	values.Set("client_id", c.clientID)
	if c.clientID == c.backendClientID && c.backendClientSecret != "" {
		values.Set("client_secret", c.backendClientSecret)
	}
	values.Set("username", email)
	values.Set("password", password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenEndpoint(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("password grant failed: %s", strings.TrimSpace(string(payload)))
	}

	var token TokenSet
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

// GetUserTokens obtains tokens for a user using their email and password via password grant.
// This is used after registration to enable auto-login.
func (c *Client) GetUserTokens(ctx context.Context, email, password string) (*TokenSet, error) {
	return c.passwordGrantTokens(ctx, email, password)
}

func (c *Client) getRealmRole(ctx context.Context, adminToken, roleName string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.adminEndpoint(fmt.Sprintf("/roles/%s", url.PathEscape(roleName))), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("fetch role failed: %s", strings.TrimSpace(string(payload)))
	}

	var role map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&role); err != nil {
		return nil, err
	}
	return role, nil
}

func (c *Client) exchangeForUser(ctx context.Context, adminToken, userID string) (*TokenSet, error) {
	values := url.Values{}
	values.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
	values.Set("client_id", c.backendClientID)
	if c.backendClientSecret != "" {
		values.Set("client_secret", c.backendClientSecret)
	}
	values.Set("subject_token", adminToken)
	values.Set("requested_subject", userID)
	values.Set("requested_token_type", "urn:ietf:params:oauth:token-type:access_token")
	if c.clientID != "" {
		values.Set("audience", c.clientID)
	}
	// Request minimal scope so realms without profile/email still work
	values.Set("scope", "openid")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenEndpoint(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("token exchange failed: %s", strings.TrimSpace(string(payload)))
	}

	var tokens TokenSet
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, err
	}
	return &tokens, nil
}

func (c *Client) getUser(ctx context.Context, adminToken, userID string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.adminEndpoint(fmt.Sprintf("/users/%s", userID)), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("get user failed: %s", strings.TrimSpace(string(payload)))
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body, nil
}

func (c *Client) adminEndpoint(p string) string {
	return c.baseURL + "/admin/realms/" + url.PathEscape(c.realm) + p
}

func (c *Client) tokenEndpoint() string {
	return c.baseURL + "/realms/" + url.PathEscape(c.realm) + "/protocol/openid-connect/token"
}

func (c *Client) realmTokenEndpoint(realm string) string {
	return c.baseURL + "/realms/" + url.PathEscape(realm) + "/protocol/openid-connect/token"
}

func (c *Client) adminTokenEndpoint() string {
	return c.baseURL + "/realms/master/protocol/openid-connect/token"
}

func (c *Client) logoutEndpoint() string {
	return c.baseURL + "/realms/" + url.PathEscape(c.realm) + "/protocol/openid-connect/logout"
}

// RefreshToken exchanges a refresh token for new tokens
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenSet, error) {
	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("client_id", c.clientID)
	values.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenEndpoint(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("refresh token request failed: %s", strings.TrimSpace(string(payload)))
	}

	var token TokenSet
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

// LogoutUser logs out a user from Keycloak by calling the logout endpoint
// This will invalidate the user's session on the Keycloak server
func (c *Client) LogoutUser(ctx context.Context, refreshToken string) error {
	c.logger.Info().Msg("[KC-LOGOUT] Starting logout process")

	if refreshToken == "" {
		c.logger.Error().Msg("[KC-LOGOUT] Refresh token is empty")
		return errors.New("refresh token is required for logout")
	}

	c.logger.Info().
		Str("token_preview", refreshToken[:30]+"...").
		Int("token_length", len(refreshToken)).
		Msg("[KC-LOGOUT] Refresh token received")

	values := url.Values{}
	values.Set("client_id", c.clientID)

	// Note: For public clients (like jan-client), we should NOT send client_secret
	// Only send client_secret if this is a confidential client (backend)
	isConfidentialClient := c.clientID == c.backendClientID && c.backendClientSecret != ""
	if isConfidentialClient {
		values.Set("client_secret", c.backendClientSecret)
		c.logger.Info().
			Str("client_id", c.clientID).
			Bool("with_secret", true).
			Msg("[KC-LOGOUT] Using confidential client (with secret)")
	} else {
		c.logger.Info().
			Str("client_id", c.clientID).
			Bool("with_secret", false).
			Msg("[KC-LOGOUT] Using public client (no secret)")
	}
	values.Set("refresh_token", refreshToken)

	logoutURL := c.logoutEndpoint()
	c.logger.Info().
		Str("url", logoutURL).
		Str("method", http.MethodPost).
		Msg("[KC-LOGOUT] Preparing logout request")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, logoutURL, strings.NewReader(values.Encode()))
	if err != nil {
		c.logger.Error().Err(err).Msg("[KC-LOGOUT] Failed to create request")
		return fmt.Errorf("create logout request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	c.logger.Info().
		Str("content_type", "application/x-www-form-urlencoded").
		Int("body_length", len(values.Encode())).
		Msg("[KC-LOGOUT] Sending logout request to Keycloak")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error().
			Err(err).
			Str("url", logoutURL).
			Msg("[KC-LOGOUT] HTTP request failed")
		return fmt.Errorf("execute logout request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Info().
		Int("status_code", resp.StatusCode).
		Str("status", resp.Status).
		Msg("[KC-LOGOUT] Received response from Keycloak")

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		c.logger.Error().
			Int("status_code", resp.StatusCode).
			Str("response_body", strings.TrimSpace(string(payload))).
			Str("client_id", c.clientID).
			Bool("is_confidential", isConfidentialClient).
			Msg("[KC-LOGOUT] Logout request failed")
		return fmt.Errorf("logout request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	c.logger.Info().
		Int("status_code", resp.StatusCode).
		Msg("[KC-LOGOUT] Logout successful")
	return nil
}

func extractIDFromLocation(location string) string {
	if location == "" {
		return ""
	}
	idx := strings.LastIndex(location, "/")
	if idx == -1 || idx+1 >= len(location) {
		return ""
	}
	return location[idx+1:]
}

// buildUserAttributeUpdate ensures required profile fields are retained when updating attributes.
func buildUserAttributeUpdate(existing map[string]any, attributes map[string][]string) map[string]any {
	update := map[string]any{
		"attributes": attributes,
		"email":      getString(existing, "email"),
	}

	if first := getString(existing, "firstName"); first != "" {
		update["firstName"] = first
	}
	if last := getString(existing, "lastName"); last != "" {
		update["lastName"] = last
	}
	if enabled, ok := existing["enabled"].(bool); ok {
		update["enabled"] = enabled
	}
	if verified, ok := existing["emailVerified"].(bool); ok {
		update["emailVerified"] = verified
	}

	return update
}

// StoreAPIKeyHash stores an API key hash in Keycloak user attributes
func (c *Client) StoreAPIKeyHash(ctx context.Context, userID, keyID, keyHash string) error {
	serviceToken, err := c.serviceAccountToken(ctx)
	if err != nil {
		return fmt.Errorf("get service token: %w", err)
	}

	adminToken := c.adminAccessToken(ctx, serviceToken.AccessToken)

	// Get existing user
	existing, err := c.getUser(ctx, adminToken, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	// Parse existing attributes
	attributes := map[string][]string{}
	if raw, ok := existing["attributes"].(map[string]any); ok {
		for key, value := range raw {
			switch v := value.(type) {
			case []any:
				var out []string
				for _, item := range v {
					if s, ok := item.(string); ok {
						out = append(out, s)
					}
				}
				if len(out) > 0 {
					attributes[key] = out
				}
			}
		}
	}

	// Add API key entry in format: keyID:hash
	keyEntry := fmt.Sprintf("%s:%s", keyID, keyHash)
	apiKeys := attributes["api_keys"]
	apiKeys = append(apiKeys, keyEntry)
	attributes["api_keys"] = apiKeys

	// Update user attributes
	update := buildUserAttributeUpdate(existing, attributes)

	body, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.adminEndpoint("/users/"+url.PathEscape(userID)), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("update user failed: %s", strings.TrimSpace(string(payload)))
	}

	return nil
}

// RemoveAPIKeyHash removes an API key hash from Keycloak user attributes
func (c *Client) RemoveAPIKeyHash(ctx context.Context, userID, keyID string) error {
	serviceToken, err := c.serviceAccountToken(ctx)
	if err != nil {
		return fmt.Errorf("get service token: %w", err)
	}

	adminToken := c.adminAccessToken(ctx, serviceToken.AccessToken)

	// Get existing user
	existing, err := c.getUser(ctx, adminToken, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	// Parse existing attributes
	attributes := map[string][]string{}
	if raw, ok := existing["attributes"].(map[string]any); ok {
		for key, value := range raw {
			switch v := value.(type) {
			case []any:
				var out []string
				for _, item := range v {
					if s, ok := item.(string); ok {
						out = append(out, s)
					}
				}
				if len(out) > 0 {
					attributes[key] = out
				}
			}
		}
	}

	// Remove API key entry by keyID
	apiKeys := attributes["api_keys"]
	filtered := []string{}
	for _, entry := range apiKeys {
		if !strings.HasPrefix(entry, keyID+":") {
			filtered = append(filtered, entry)
		}
	}
	attributes["api_keys"] = filtered

	// Update user attributes
	update := buildUserAttributeUpdate(existing, attributes)

	body, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.adminEndpoint("/users/"+url.PathEscape(userID)), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("update user failed: %s", strings.TrimSpace(string(payload)))
	}

	return nil
}

// APIKeyUserInfo represents validated user information from API key
type APIKeyUserInfo struct {
	UserID    string   `json:"user_id"`
	Subject   string   `json:"subject"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Roles     []string `json:"roles"`
}

// KeycloakUser represents a user in Keycloak
type KeycloakUser struct {
	ID           string   `json:"id"`
	Username     string   `json:"username"`
	Email        string   `json:"email"`
	FirstName    string   `json:"firstName"`
	LastName     string   `json:"lastName"`
	Enabled      bool     `json:"enabled"`
	Roles        []string `json:"roles,omitempty"`
	Groups       []string `json:"groups,omitempty"`
	FeatureFlags []string `json:"featureFlags,omitempty"`
}

// ListUsersParams defines filters for listing users.
type ListUsersParams struct {
	First         int
	Max           int
	Search        string
	Enabled       *bool
	GroupID       string
	Role          string
	ExcludeGuests bool // If true, exclude users with 'guest' role
}

// AdminUserRequest represents admin user create/update payload.
type AdminUserRequest struct {
	Username string           `json:"username,omitempty"`
	Email    string           `json:"email,omitempty"`
	First    string           `json:"firstName,omitempty"`
	Last     string           `json:"lastName,omitempty"`
	Enabled  *bool            `json:"enabled,omitempty"`
	Attrs    map[string][]any `json:"attributes,omitempty"`
	Roles    []string         `json:"roles,omitempty"`
	Groups   []string         `json:"groups,omitempty"`
	Password *string          `json:"password,omitempty"`
}

type Group struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Path       string           `json:"path"`
	Attributes map[string][]any `json:"attributes,omitempty"`
}

// GetUserBySubject retrieves a user from Keycloak by their subject (user ID)
func (c *Client) GetUserBySubject(ctx context.Context, subject string) (*KeycloakUser, error) {
	if strings.TrimSpace(subject) == "" {
		return nil, errors.New("subject required")
	}

	serviceToken, err := c.serviceAccountToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get service token: %w", err)
	}

	adminToken := c.adminAccessToken(ctx, serviceToken.AccessToken)

	// Get user by ID (subject is the Keycloak user ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.adminEndpoint(fmt.Sprintf("/users/%s", url.PathEscape(subject))), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, errors.New("user not found in keycloak")
	}

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("get user failed: %s", strings.TrimSpace(string(payload)))
	}

	var rawUser map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rawUser); err != nil {
		return nil, fmt.Errorf("decode user: %w", err)
	}

	user := &KeycloakUser{
		ID:        getString(rawUser, "id"),
		Username:  getString(rawUser, "username"),
		Email:     getString(rawUser, "email"),
		FirstName: getString(rawUser, "firstName"),
		LastName:  getString(rawUser, "lastName"),
		Enabled:   getBool(rawUser, "enabled"),
	}

	return user, nil
}

// ValidateAPIKeyHash validates an API key hash and returns user information
func (c *Client) ValidateAPIKeyHash(ctx context.Context, keyHash string) (*APIKeyUserInfo, error) {
	serviceToken, err := c.serviceAccountToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get service token: %w", err)
	}

	adminToken := c.adminAccessToken(ctx, serviceToken.AccessToken)

	// Search for users with this API key hash in attributes
	// Note: Keycloak doesn't support searching in custom attributes directly,
	// so we need to get all users and filter (or use a more efficient approach with a separate index)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.adminEndpoint("/users?max=10000"), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get users: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get users failed: status %d", resp.StatusCode)
	}

	var users []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("decode users: %w", err)
	}

	// Find user with matching API key hash
	for _, user := range users {
		if attrs, ok := user["attributes"].(map[string]any); ok {
			if apiKeysRaw, ok := attrs["api_keys"]; ok {
				var apiKeys []string
				switch v := apiKeysRaw.(type) {
				case []any:
					for _, item := range v {
						if s, ok := item.(string); ok {
							apiKeys = append(apiKeys, s)
						}
					}
				}

				// Check if any API key entry matches the hash
				for _, entry := range apiKeys {
					parts := strings.SplitN(entry, ":", 2)
					if len(parts) == 2 && parts[1] == keyHash {
						// Found matching user
						userInfo := &APIKeyUserInfo{
							UserID:    getString(user, "id"),
							Subject:   getString(user, "id"),
							Username:  getString(user, "username"),
							Email:     getString(user, "email"),
							FirstName: getString(user, "firstName"),
							LastName:  getString(user, "lastName"),
						}
						return userInfo, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("invalid api key")
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func mapUsers(raw []map[string]any) []KeycloakUser {
	users := make([]KeycloakUser, 0, len(raw))
	for _, u := range raw {
		users = append(users, KeycloakUser{
			ID:        getString(u, "id"),
			Username:  getString(u, "username"),
			Email:     getString(u, "email"),
			FirstName: getString(u, "firstName"),
			LastName:  getString(u, "lastName"),
			Enabled:   getBool(u, "enabled"),
		})
	}
	return users
}

func filterAndLimitUsers(users []KeycloakUser, params ListUsersParams, extraFilter func(KeycloakUser) bool) []KeycloakUser {
	var filtered []KeycloakUser
	search := strings.ToLower(strings.TrimSpace(params.Search))

	matches := func(u KeycloakUser) bool {
		if extraFilter != nil && !extraFilter(u) {
			return false
		}
		if params.Enabled != nil && u.Enabled != *params.Enabled {
			return false
		}
		// Exclude guests if requested
		if params.ExcludeGuests {
			for _, role := range u.Roles {
				if role == "guest" {
					return false
				}
			}
		}
		if search != "" {
			if !strings.Contains(strings.ToLower(u.Username), search) &&
				!strings.Contains(strings.ToLower(u.Email), search) &&
				!strings.Contains(strings.ToLower(u.FirstName), search) &&
				!strings.Contains(strings.ToLower(u.LastName), search) {
				return false
			}
		}
		return true
	}

	for _, u := range users {
		if matches(u) {
			filtered = append(filtered, u)
			if params.Max > 0 && len(filtered) >= params.Max {
				break
			}
		}
	}
	return filtered
}

// getUserRoles fetches realm roles for a specific user
func (c *Client) getUserRoles(ctx context.Context, token, userID string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.adminEndpoint(fmt.Sprintf("/users/%s/role-mappings/realm", url.PathEscape(userID))), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get user roles failed: status %d", resp.StatusCode)
	}

	var roles []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, err
	}

	result := make([]string, 0, len(roles))
	for _, role := range roles {
		if name := getString(role, "name"); name != "" {
			result = append(result, name)
		}
	}
	return result, nil
}

// enrichUsersWithRoles fetches roles for each user and populates the Roles field
func (c *Client) enrichUsersWithRoles(ctx context.Context, token string, users []KeycloakUser) []KeycloakUser {
	enriched := make([]KeycloakUser, len(users))
	for i, u := range users {
		enriched[i] = u
		roles, err := c.getUserRoles(ctx, token, u.ID)
		if err == nil {
			enriched[i].Roles = roles
		}
	}
	return enriched
}

// getUserGroups fetches groups for a specific user
func (c *Client) getUserGroups(ctx context.Context, token, userID string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.adminEndpoint(fmt.Sprintf("/users/%s/groups", url.PathEscape(userID))), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get user groups failed: status %d", resp.StatusCode)
	}

	var groups []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, err
	}

	result := make([]string, 0, len(groups))
	for _, group := range groups {
		if path := getString(group, "path"); path != "" {
			result = append(result, path)
		}
	}
	return result, nil
}

// getUserFeatureFlags fetches effective feature flags for a user by aggregating from all their groups
func (c *Client) getUserFeatureFlags(ctx context.Context, token, userID string) ([]string, error) {
	// Get user's groups
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.adminEndpoint(fmt.Sprintf("/users/%s/groups", url.PathEscape(userID))), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get user groups failed: status %d", resp.StatusCode)
	}

	var groups []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, err
	}

	// Collect unique feature flags from all groups
	flagSet := make(map[string]struct{})
	for _, group := range groups {
		if attrs, ok := group["attributes"].(map[string]any); ok {
			if flags, ok := attrs["feature_flags"].([]any); ok {
				for _, flag := range flags {
					if flagStr, ok := flag.(string); ok {
						flagSet[flagStr] = struct{}{}
					}
				}
			}
		}
	}

	// Convert set to slice
	result := make([]string, 0, len(flagSet))
	for flag := range flagSet {
		result = append(result, flag)
	}
	return result, nil
}

// enrichUsersWithGroupsAndFlags fetches groups and feature flags for each user
func (c *Client) enrichUsersWithGroupsAndFlags(ctx context.Context, token string, users []KeycloakUser) []KeycloakUser {
	enriched := make([]KeycloakUser, len(users))
	for i, u := range users {
		enriched[i] = u
		groups, err := c.getUserGroups(ctx, token, u.ID)
		if err == nil {
			enriched[i].Groups = groups
		}
		flags, err := c.getUserFeatureFlags(ctx, token, u.ID)
		if err == nil {
			enriched[i].FeatureFlags = flags
		}
	}
	return enriched
}

// enrichUsersComplete fetches roles, groups, and feature flags for each user
func (c *Client) enrichUsersComplete(ctx context.Context, token string, users []KeycloakUser) []KeycloakUser {
	enriched := make([]KeycloakUser, len(users))
	for i, u := range users {
		enriched[i] = u
		// Fetch roles
		roles, err := c.getUserRoles(ctx, token, u.ID)
		if err == nil {
			enriched[i].Roles = roles
		}
		// Fetch groups
		groups, err := c.getUserGroups(ctx, token, u.ID)
		if err == nil {
			enriched[i].Groups = groups
		}
		// Fetch feature flags
		flags, err := c.getUserFeatureFlags(ctx, token, u.ID)
		if err == nil {
			enriched[i].FeatureFlags = flags
		}
	}
	return enriched
}

// adminAuthToken returns a bearer token for admin operations.
func (c *Client) adminAuthToken(ctx context.Context) (string, error) {
	serviceToken, err := c.serviceAccountToken(ctx)
	if err != nil {
		return "", err
	}
	return c.adminAccessToken(ctx, serviceToken.AccessToken), nil
}

// ---- Admin User Operations ----

// ListUsers returns users with pagination and optional filters (search/enabled/group/role).
func (c *Client) ListUsers(ctx context.Context, params ListUsersParams) ([]KeycloakUser, error) {
	// defaults
	if params.Max == 0 {
		params.Max = 50
	}
	if params.First < 0 {
		params.First = 0
	}

	trimmedGroup := strings.TrimSpace(params.GroupID)
	trimmedRole := strings.TrimSpace(params.Role)

	switch {
	case trimmedGroup != "" && trimmedRole != "":
		groupUsers, err := c.ListGroupMembers(ctx, trimmedGroup, params.First, params.Max)
		if err != nil {
			return nil, err
		}
		roleUsers, err := c.ListUsersByRole(ctx, trimmedRole, params.First, params.Max)
		if err != nil {
			return nil, err
		}
		roleSet := make(map[string]struct{}, len(roleUsers))
		for _, u := range roleUsers {
			roleSet[u.ID] = struct{}{}
		}
		// Always enrich with roles, groups, and feature flags
		token, err := c.adminAuthToken(ctx)
		if err != nil {
			return nil, err
		}
		groupUsers = c.enrichUsersComplete(ctx, token, groupUsers)
		return filterAndLimitUsers(groupUsers, params, func(u KeycloakUser) bool {
			_, ok := roleSet[u.ID]
			return ok
		}), nil
	case trimmedGroup != "":
		groupUsers, err := c.ListGroupMembers(ctx, trimmedGroup, params.First, params.Max)
		if err != nil {
			return nil, err
		}
		// Always enrich with roles, groups, and feature flags
		token, err := c.adminAuthToken(ctx)
		if err != nil {
			return nil, err
		}
		groupUsers = c.enrichUsersComplete(ctx, token, groupUsers)
		return filterAndLimitUsers(groupUsers, params, nil), nil
	case trimmedRole != "":
		roleUsers, err := c.ListUsersByRole(ctx, trimmedRole, params.First, params.Max)
		if err != nil {
			return nil, err
		}
		// Always enrich with roles, groups, and feature flags
		token, err := c.adminAuthToken(ctx)
		if err != nil {
			return nil, err
		}
		roleUsers = c.enrichUsersComplete(ctx, token, roleUsers)
		return filterAndLimitUsers(roleUsers, params, nil), nil
	default:
		token, err := c.adminAuthToken(ctx)
		if err != nil {
			return nil, err
		}

		query := fmt.Sprintf("first=%d&max=%d", params.First, params.Max)
		if strings.TrimSpace(params.Search) != "" {
			query += "&search=" + url.QueryEscape(strings.TrimSpace(params.Search))
		}
		if params.Enabled != nil {
			if *params.Enabled {
				query += "&enabled=true"
			} else {
				query += "&enabled=false"
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.adminEndpoint("/users?"+query), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			return nil, fmt.Errorf("list users failed: %s", strings.TrimSpace(string(payload)))
		}

		var raw []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			return nil, err
		}

		users := mapUsers(raw)
		// Always enrich users with roles, groups, and feature flags
		users = c.enrichUsersComplete(ctx, token, users)
		return filterAndLimitUsers(users, params, nil), nil
	}
}

// ListUsersByRole returns users with the given realm role.
func (c *Client) ListUsersByRole(ctx context.Context, role string, first, max int) ([]KeycloakUser, error) {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.adminEndpoint(fmt.Sprintf("/roles/%s/users?first=%d&max=%d", url.PathEscape(role), first, max)), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("list users by role failed: %s", strings.TrimSpace(string(payload)))
	}

	var raw []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	return mapUsers(raw), nil
}

// GetUser returns a single user.
func (c *Client) GetUser(ctx context.Context, id string) (*KeycloakUser, error) {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.adminEndpoint("/users/"+url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("get user failed: %s", strings.TrimSpace(string(payload)))
	}
	var u KeycloakUser
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUser creates a user and returns the ID.
func (c *Client) CreateUser(ctx context.Context, req AdminUserRequest) (string, error) {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.adminEndpoint("/users"), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("create user failed: %s", strings.TrimSpace(string(payload)))
	}
	id := extractIDFromLocation(resp.Header.Get("Location"))
	return id, nil
}

// UpdateUser updates user properties.
func (c *Client) UpdateUser(ctx context.Context, id string, req AdminUserRequest) error {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return err
	}
	if req.Username == "" {
		existing, err := c.GetUser(ctx, id)
		if err != nil {
			return err
		}
		req.Username = existing.Username
		if req.Email == "" {
			req.Email = existing.Email
		}
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, c.adminEndpoint("/users/"+url.PathEscape(id)), bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("update user failed: %s", strings.TrimSpace(string(payload)))
	}
	return nil
}

// DeleteUser deletes the user.
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.adminEndpoint("/users/"+url.PathEscape(id)), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("delete user failed: %s", strings.TrimSpace(string(payload)))
	}
	return nil
}

// SetUserPassword sets the password for a user.
func (c *Client) SetUserPassword(ctx context.Context, userID, password string, temporary bool) error {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return err
	}

	credential := map[string]interface{}{
		"type":      "password",
		"value":     password,
		"temporary": temporary,
	}
	body, err := json.Marshal(credential)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		c.adminEndpoint("/users/"+url.PathEscape(userID)+"/reset-password"),
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("set user password failed: %s", strings.TrimSpace(string(payload)))
	}
	return nil
}

// SetUserEnabled toggles enabled flag.
func (c *Client) SetUserEnabled(ctx context.Context, id string, enabled bool) error {
	current, err := c.GetUser(ctx, id)
	if err != nil {
		return err
	}
	req := AdminUserRequest{
		Username: current.Username,
		Email:    current.Email,
		Enabled:  &enabled,
	}
	return c.UpdateUser(ctx, id, req)
}

// AssignRealmRole adds a realm role to a user.
func (c *Client) AssignRealmRole(ctx context.Context, userID, roleName string) error {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return err
	}
	role, err := c.getRealmRole(ctx, token, roleName)
	if err != nil {
		return err
	}
	body, err := json.Marshal([]map[string]any{role})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.adminEndpoint(fmt.Sprintf("/users/%s/role-mappings/realm", url.PathEscape(userID))), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("assign role failed: %s", strings.TrimSpace(string(payload)))
	}
	return nil
}

// RemoveRealmRole removes a realm role from a user.
func (c *Client) RemoveRealmRole(ctx context.Context, userID, roleName string) error {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return err
	}
	role, err := c.getRealmRole(ctx, token, roleName)
	if err != nil {
		return err
	}
	body, err := json.Marshal([]map[string]any{role})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.adminEndpoint(fmt.Sprintf("/users/%s/role-mappings/realm", url.PathEscape(userID))), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("remove role failed: %s", strings.TrimSpace(string(payload)))
	}
	return nil
}

// ---- Group operations ----

func (c *Client) ListGroups(ctx context.Context, first, max int) ([]Group, error) {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.adminEndpoint(fmt.Sprintf("/groups?first=%d&max=%d", first, max)), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("list groups failed: %s", strings.TrimSpace(string(payload)))
	}
	var raw []Group
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *Client) GetGroup(ctx context.Context, id string) (*Group, error) {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.adminEndpoint("/groups/"+url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("get group failed: %s", strings.TrimSpace(string(payload)))
	}
	var g Group
	if err := json.NewDecoder(resp.Body).Decode(&g); err != nil {
		return nil, err
	}
	return &g, nil
}

func (c *Client) CreateGroup(ctx context.Context, name string) (string, error) {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return "", err
	}
	body, _ := json.Marshal(map[string]string{"name": name})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.adminEndpoint("/groups"), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("create group failed: %s", strings.TrimSpace(string(payload)))
	}
	return extractIDFromLocation(resp.Header.Get("Location")), nil
}

func (c *Client) UpdateGroup(ctx context.Context, id, name string, attributes map[string][]any) error {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]any{
		"name":       name,
		"attributes": attributes,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.adminEndpoint("/groups/"+url.PathEscape(id)), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("update group failed: %s", strings.TrimSpace(string(payload)))
	}
	return nil
}

func (c *Client) DeleteGroup(ctx context.Context, id string) error {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.adminEndpoint("/groups/"+url.PathEscape(id)), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("delete group failed: %s", strings.TrimSpace(string(payload)))
	}
	return nil
}

func (c *Client) ListGroupMembers(ctx context.Context, groupID string, first, max int) ([]KeycloakUser, error) {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.adminEndpoint(fmt.Sprintf("/groups/%s/members?first=%d&max=%d", url.PathEscape(groupID), first, max)), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("list group members failed: %s", strings.TrimSpace(string(payload)))
	}
	var raw []KeycloakUser
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *Client) AddUserToGroup(ctx context.Context, userID, groupID string) error {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.adminEndpoint(fmt.Sprintf("/users/%s/groups/%s", url.PathEscape(userID), url.PathEscape(groupID))), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("add user to group failed: %s", strings.TrimSpace(string(payload)))
	}
	return nil
}

func (c *Client) RemoveUserFromGroup(ctx context.Context, userID, groupID string) error {
	token, err := c.adminAuthToken(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.adminEndpoint(fmt.Sprintf("/users/%s/groups/%s", url.PathEscape(userID), url.PathEscape(groupID))), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("remove user from group failed: %s", strings.TrimSpace(string(payload)))
	}
	return nil
}

func (c *Client) GetGroupAttributes(ctx context.Context, groupID string) (map[string][]any, error) {
	group, err := c.GetGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return group.Attributes, nil
}

func (c *Client) SetGroupAttributes(ctx context.Context, groupID string, attrs map[string][]any) error {
	group, err := c.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	return c.UpdateGroup(ctx, groupID, group.Name, attrs)
}

// Feature flag helpers (group attributes)
func (c *Client) GetGroupFeatureFlags(ctx context.Context, groupID string) ([]string, error) {
	attrs, err := c.GetGroupAttributes(ctx, groupID)
	if err != nil {
		return nil, err
	}
	raw := attrs["feature_flags"]
	flags := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			flags = append(flags, s)
		}
	}
	return flags, nil
}

func (c *Client) SetGroupFeatureFlags(ctx context.Context, groupID string, flags []string) error {
	attrVals := make([]any, 0, len(flags))
	for _, f := range flags {
		attrVals = append(attrVals, f)
	}
	return c.SetGroupAttributes(ctx, groupID, map[string][]any{
		"feature_flags": attrVals,
	})
}
