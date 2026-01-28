package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"jan-server/services/llm-api/internal/domain/connector"
)

const (
	// GitHub OAuth endpoints
	GitHubAuthURL  = "https://github.com/login/oauth/authorize"
	GitHubTokenURL = "https://github.com/login/oauth/access_token"
	GitHubAPIURL   = "https://api.github.com"

	// Google OAuth endpoints
	GoogleAuthURL   = "https://accounts.google.com/o/oauth2/v2/auth"
	GoogleTokenURL  = "https://oauth2.googleapis.com/token"
	GoogleRevokeURL = "https://oauth2.googleapis.com/revoke"
	GoogleUserURL   = "https://www.googleapis.com/oauth2/v2/userinfo"
)

// Scopes for each connector type
var connectorScopes = map[connector.ConnectorType][]string{
	connector.ConnectorTypeGitHub: {
		"repo",
		"read:user",
		"user:email",
		"workflow",
	},
	connector.ConnectorTypeGmail: {
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	},
	connector.ConnectorTypeGoogleDrive: {
		"https://www.googleapis.com/auth/drive.readonly",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	},
	connector.ConnectorTypeGoogleCalendar: {
		"https://www.googleapis.com/auth/calendar",
		"https://www.googleapis.com/auth/calendar.events",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	},
}

// OAuthProviderConfig holds OAuth provider configuration.
type OAuthProviderConfig struct {
	GitHubClientID     string
	GitHubClientSecret string
	GoogleClientID     string
	GoogleClientSecret string
}

// OAuthProvider implements the connector.OAuthProvider interface.
type OAuthProvider struct {
	config     OAuthProviderConfig
	httpClient *http.Client
}

// NewOAuthProvider creates a new OAuth provider.
func NewOAuthProvider(config OAuthProviderConfig) *OAuthProvider {
	return &OAuthProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetAuthURL generates the OAuth authorization URL for a connector type.
func (p *OAuthProvider) GetAuthURL(connectorType connector.ConnectorType, state, redirectURI, codeChallenge string) (string, error) {
	switch connectorType {
	case connector.ConnectorTypeGitHub:
		return p.getGitHubAuthURL(state, redirectURI, codeChallenge), nil
	case connector.ConnectorTypeGmail, connector.ConnectorTypeGoogleDrive, connector.ConnectorTypeGoogleCalendar:
		return p.getGoogleAuthURL(connectorType, state, redirectURI, codeChallenge), nil
	default:
		return "", fmt.Errorf("unsupported connector type: %s", connectorType)
	}
}

// ExchangeCode exchanges an authorization code for tokens.
func (p *OAuthProvider) ExchangeCode(ctx context.Context, connectorType connector.ConnectorType, code, codeVerifier, redirectURI string) (*connector.OAuthTokens, error) {
	switch connectorType {
	case connector.ConnectorTypeGitHub:
		return p.exchangeGitHubCode(ctx, code, codeVerifier, redirectURI)
	case connector.ConnectorTypeGmail, connector.ConnectorTypeGoogleDrive, connector.ConnectorTypeGoogleCalendar:
		return p.exchangeGoogleCode(ctx, code, codeVerifier, redirectURI)
	default:
		return nil, fmt.Errorf("unsupported connector type: %s", connectorType)
	}
}

// RefreshToken refreshes OAuth tokens.
func (p *OAuthProvider) RefreshToken(ctx context.Context, connectorType connector.ConnectorType, refreshToken string) (*connector.OAuthTokens, error) {
	switch connectorType {
	case connector.ConnectorTypeGitHub:
		// GitHub tokens don't expire by default, so refresh is not needed
		return nil, fmt.Errorf("GitHub tokens do not need refresh")
	case connector.ConnectorTypeGmail, connector.ConnectorTypeGoogleDrive, connector.ConnectorTypeGoogleCalendar:
		return p.refreshGoogleToken(ctx, refreshToken)
	default:
		return nil, fmt.Errorf("unsupported connector type: %s", connectorType)
	}
}

// GetUserInfo fetches user info from the OAuth provider.
func (p *OAuthProvider) GetUserInfo(ctx context.Context, connectorType connector.ConnectorType, accessToken string) (*connector.ProviderUserInfo, error) {
	switch connectorType {
	case connector.ConnectorTypeGitHub:
		return p.getGitHubUser(ctx, accessToken)
	case connector.ConnectorTypeGmail, connector.ConnectorTypeGoogleDrive, connector.ConnectorTypeGoogleCalendar:
		return p.getGoogleUser(ctx, accessToken)
	default:
		return nil, fmt.Errorf("unsupported connector type: %s", connectorType)
	}
}

// RevokeToken revokes an OAuth token.
func (p *OAuthProvider) RevokeToken(ctx context.Context, connectorType connector.ConnectorType, token string) error {
	switch connectorType {
	case connector.ConnectorTypeGitHub:
		// GitHub doesn't have a standard revoke endpoint for OAuth apps
		return nil
	case connector.ConnectorTypeGmail, connector.ConnectorTypeGoogleDrive, connector.ConnectorTypeGoogleCalendar:
		return p.revokeGoogleToken(ctx, token)
	default:
		return fmt.Errorf("unsupported connector type: %s", connectorType)
	}
}

// GitHub methods

func (p *OAuthProvider) getGitHubAuthURL(state, redirectURI, codeChallenge string) string {
	scopes := connectorScopes[connector.ConnectorTypeGitHub]
	params := url.Values{
		"client_id":             {p.config.GitHubClientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {strings.Join(scopes, " ")},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}
	return GitHubAuthURL + "?" + params.Encode()
}

func (p *OAuthProvider) exchangeGitHubCode(ctx context.Context, code, codeVerifier, redirectURI string) (*connector.OAuthTokens, error) {
	data := url.Values{
		"client_id":     {p.config.GitHubClientID},
		"client_secret": {p.config.GitHubClientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", GitHubTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("GitHub OAuth error: %s - %s", result.Error, result.ErrorDesc)
	}

	return &connector.OAuthTokens{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		Scope:       result.Scope,
	}, nil
}

func (p *OAuthProvider) getGitHubUser(ctx context.Context, accessToken string) (*connector.ProviderUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", GitHubAPIURL+"/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var user struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	// If email is not public, fetch from emails endpoint
	email := user.Email
	if email == "" {
		email, _ = p.getGitHubPrimaryEmail(ctx, accessToken)
	}

	return &connector.ProviderUserInfo{
		ID:        fmt.Sprintf("%d", user.ID),
		Username:  user.Login,
		Email:     email,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
	}, nil
}

func (p *OAuthProvider) getGitHubPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", GitHubAPIURL+"/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	return "", nil
}

// Google methods

func (p *OAuthProvider) getGoogleAuthURL(connectorType connector.ConnectorType, state, redirectURI, codeChallenge string) string {
	scopes := connectorScopes[connectorType]
	params := url.Values{
		"client_id":              {p.config.GoogleClientID},
		"redirect_uri":           {redirectURI},
		"response_type":          {"code"},
		"scope":                  {strings.Join(scopes, " ")},
		"state":                  {state},
		"access_type":            {"offline"},
		"prompt":                 {"consent"},
		"code_challenge":         {codeChallenge},
		"code_challenge_method":  {"S256"},
		"include_granted_scopes": {"true"},
	}
	return GoogleAuthURL + "?" + params.Encode()
}

func (p *OAuthProvider) exchangeGoogleCode(ctx context.Context, code, codeVerifier, redirectURI string) (*connector.OAuthTokens, error) {
	data := url.Values{
		"client_id":     {p.config.GoogleClientID},
		"client_secret": {p.config.GoogleClientSecret},
		"code":          {code},
		"code_verifier": {codeVerifier},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", GoogleTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google token exchange failed: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &connector.OAuthTokens{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    result.TokenType,
		ExpiresIn:    result.ExpiresIn,
		Scope:        result.Scope,
	}, nil
}

func (p *OAuthProvider) refreshGoogleToken(ctx context.Context, refreshToken string) (*connector.OAuthTokens, error) {
	data := url.Values{
		"client_id":     {p.config.GoogleClientID},
		"client_secret": {p.config.GoogleClientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", GoogleTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google token refresh failed: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		Scope       string `json:"scope"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &connector.OAuthTokens{
		AccessToken:  result.AccessToken,
		RefreshToken: refreshToken, // Keep the original refresh token
		TokenType:    result.TokenType,
		ExpiresIn:    result.ExpiresIn,
		Scope:        result.Scope,
	}, nil
}

func (p *OAuthProvider) getGoogleUser(ctx context.Context, accessToken string) (*connector.ProviderUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", GoogleUserURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google API error: %d - %s", resp.StatusCode, string(body))
	}

	var user struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &connector.ProviderUserInfo{
		ID:        user.ID,
		Username:  user.Email, // Use email as username for Google
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.Picture,
	}, nil
}

func (p *OAuthProvider) revokeGoogleToken(ctx context.Context, token string) error {
	data := url.Values{
		"token": {token},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", GoogleRevokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Google returns 200 for success, but we don't fail if revoke fails
	return nil
}
