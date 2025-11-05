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
	targetClientID      string
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
func NewClient(baseURL, realm, backendClientID, backendClientSecret, targetClientID, guestRole string, httpClient *http.Client, logger zerolog.Logger, adminUsername, adminPassword, adminRealm, adminClientID, adminClientSecret string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:             strings.TrimRight(baseURL, "/"),
		realm:               realm,
		backendClientID:     backendClientID,
		backendClientSecret: backendClientSecret,
		targetClientID:      targetClientID,
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
	PrincipalID string   `json:"pid"`
	Tokens      TokenSet `json:"tokens"`
}

// UpgradePayload describes the upgrade request body.
type UpgradePayload struct {
	Username string `json:"username"`
	Email    string `json:"email"`
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
			user, err := c.createGuestUser(ctx, adminToken.AccessToken)
			if err != nil {
				return nil, err
			}

			if err := c.assignGuestRole(ctx, adminToken.AccessToken, user.UserID); err != nil {
				return nil, err
			}

			password := strings.ReplaceAll(uuid.NewString(), "-", "")
			if err := c.setUserPassword(ctx, adminToken.AccessToken, user.UserID, password); err != nil {
				return nil, err
			}

			tokens, err := c.passwordGrantTokens(ctx, user.Username, password)
			if err != nil {
				return nil, err
			}
			user.Tokens = *tokens
			return user, nil
		}
	}

	serviceToken, err := c.serviceAccountToken(ctx)
	if err != nil {
		return nil, err
	}

	user, err := c.createGuestUser(ctx, serviceToken.AccessToken)
	if err != nil {
		return nil, err
	}

	if err := c.assignGuestRole(ctx, serviceToken.AccessToken, user.UserID); err != nil {
		return nil, err
	}

	tokens, err := c.exchangeForUser(ctx, serviceToken.AccessToken, user.UserID)
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

	update := map[string]any{
		"attributes": attributes,
		"email":      payload.Email,
		"username":   payload.Username,
		"firstName":  payload.FullName,
		"enabled":    true,
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
	userPayload := map[string]any{
		"username":   username,
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

func (c *Client) passwordGrantTokens(ctx context.Context, username, password string) (*TokenSet, error) {
	values := url.Values{}
	values.Set("grant_type", "password")
	values.Set("client_id", c.targetClientID)
	values.Set("username", username)
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
	values.Set("audience", c.targetClientID)
	values.Set("scope", "openid profile email")

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

// RefreshToken exchanges a refresh token for new tokens
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenSet, error) {
	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("client_id", c.targetClientID)
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
