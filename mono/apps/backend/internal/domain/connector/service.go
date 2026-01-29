package connector

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var (
	ErrConnectorNotFound     = errors.New("connector not found")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrInvalidProvider       = errors.New("invalid provider")
	ErrProviderNotEnabled    = errors.New("provider not enabled")
	ErrInvalidState          = errors.New("invalid or expired state")
	ErrConnectorExists       = errors.New("connector already exists")
)

// Repository defines the interface for connector data operations.
type Repository interface {
	Create(ctx context.Context, connector *Connector) error
	GetByID(ctx context.Context, id string) (*Connector, error)
	GetByUserAndProvider(ctx context.Context, userID, provider string) (*Connector, error)
	Update(ctx context.Context, connector *Connector) error
	Delete(ctx context.Context, id string) error
	ListByUser(ctx context.Context, userID string) ([]*Connector, error)

	// OAuth state operations
	CreateState(ctx context.Context, state *OAuthState) error
	GetState(ctx context.Context, state string) (*OAuthState, error)
	DeleteState(ctx context.Context, state string) error
	DeleteExpiredStates(ctx context.Context) error
}

// TokenEncryptor defines the interface for token encryption.
type TokenEncryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// OAuthClient defines the interface for OAuth operations.
type OAuthClient interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*TokenResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)
	GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)
}

// TokenResponse contains tokens from OAuth exchange.
type TokenResponse struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int64
}

// UserInfo contains user information from the provider.
type UserInfo struct {
	ID       string
	Username string
	Email    string
	Name     string
	Metadata map[string]any
}

// ServiceConfig holds configuration for the connector service.
type ServiceConfig struct {
	GitHubEnabled    bool
	GitHubConfig     ConnectorConfig
	GoogleEnabled    bool
	GoogleConfig     ConnectorConfig
	StateExpiration  time.Duration
	FrontendURL      string
	EncryptionKeyID  string
}

// Service handles connector-related business logic.
type Service struct {
	repo        Repository
	encryptor   TokenEncryptor
	config      ServiceConfig
	oauthClients map[string]OAuthClient
}

// NewService creates a new connector service.
func NewService(repo Repository, encryptor TokenEncryptor, config ServiceConfig) *Service {
	return &Service{
		repo:         repo,
		encryptor:    encryptor,
		config:       config,
		oauthClients: make(map[string]OAuthClient),
	}
}

// RegisterOAuthClient registers an OAuth client for a provider.
func (s *Service) RegisterOAuthClient(provider string, client OAuthClient) {
	s.oauthClients[provider] = client
}

// GetAuthURL generates an OAuth authorization URL for a provider.
func (s *Service) GetAuthURL(ctx context.Context, userID, provider, redirectURL string) (*AuthURLResponse, error) {
	if !IsValidProvider(provider) {
		return nil, ErrInvalidProvider
	}

	if !s.isProviderEnabled(provider) {
		return nil, ErrProviderNotEnabled
	}

	client, ok := s.oauthClients[provider]
	if !ok {
		return nil, ErrProviderNotEnabled
	}

	// Generate state
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, err
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// Save state
	oauthState := &OAuthState{
		State:       state,
		UserID:      userID,
		Provider:    provider,
		RedirectURL: redirectURL,
		ExpiresAt:   time.Now().Add(s.config.StateExpiration),
	}

	if err := s.repo.CreateState(ctx, oauthState); err != nil {
		return nil, err
	}

	authURL := client.GetAuthURL(state)

	return &AuthURLResponse{
		AuthURL: authURL,
		State:   state,
	}, nil
}

// HandleCallback handles the OAuth callback and creates/updates the connector.
func (s *Service) HandleCallback(ctx context.Context, provider, code, state string) (*Connector, string, error) {
	// Validate state
	oauthState, err := s.repo.GetState(ctx, state)
	if err != nil || oauthState == nil {
		return nil, "", ErrInvalidState
	}

	if time.Now().After(oauthState.ExpiresAt) {
		_ = s.repo.DeleteState(ctx, state)
		return nil, "", ErrInvalidState
	}

	if oauthState.Provider != provider {
		return nil, "", ErrInvalidState
	}

	// Delete state (one-time use)
	_ = s.repo.DeleteState(ctx, state)

	client, ok := s.oauthClients[provider]
	if !ok {
		return nil, "", ErrProviderNotEnabled
	}

	// Exchange code for tokens
	tokens, err := client.ExchangeCode(ctx, code)
	if err != nil {
		return nil, "", fmt.Errorf("exchange code: %w", err)
	}

	// Get user info
	userInfo, err := client.GetUserInfo(ctx, tokens.AccessToken)
	if err != nil {
		return nil, "", fmt.Errorf("get user info: %w", err)
	}

	// Encrypt tokens
	encAccessToken, err := s.encryptor.Encrypt(tokens.AccessToken)
	if err != nil {
		return nil, "", fmt.Errorf("encrypt access token: %w", err)
	}

	var encRefreshToken string
	if tokens.RefreshToken != "" {
		encRefreshToken, err = s.encryptor.Encrypt(tokens.RefreshToken)
		if err != nil {
			return nil, "", fmt.Errorf("encrypt refresh token: %w", err)
		}
	}

	// Calculate expiration
	var expiresAt *time.Time
	if tokens.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	// Check if connector already exists
	existing, _ := s.repo.GetByUserAndProvider(ctx, oauthState.UserID, provider)
	if existing != nil {
		// Update existing connector
		existing.ProviderUserID = userInfo.ID
		existing.ProviderUsername = userInfo.Username
		existing.ProviderEmail = userInfo.Email
		existing.AccessToken = encAccessToken
		existing.RefreshToken = encRefreshToken
		existing.TokenType = tokens.TokenType
		existing.ExpiresAt = expiresAt
		existing.Metadata = userInfo.Metadata
		existing.IsActive = true
		existing.EncryptionKeyID = s.config.EncryptionKeyID

		if err := s.repo.Update(ctx, existing); err != nil {
			return nil, "", err
		}

		return existing, oauthState.RedirectURL, nil
	}

	// Create new connector
	connector := &Connector{
		UserID:           oauthState.UserID,
		Provider:         provider,
		ProviderUserID:   userInfo.ID,
		ProviderUsername: userInfo.Username,
		ProviderEmail:    userInfo.Email,
		AccessToken:      encAccessToken,
		RefreshToken:     encRefreshToken,
		TokenType:        tokens.TokenType,
		ExpiresAt:        expiresAt,
		Metadata:         userInfo.Metadata,
		IsActive:         true,
		EncryptionKeyID:  s.config.EncryptionKeyID,
	}

	if err := s.repo.Create(ctx, connector); err != nil {
		return nil, "", err
	}

	return connector, oauthState.RedirectURL, nil
}

// Disconnect disconnects a connector.
func (s *Service) Disconnect(ctx context.Context, userID, provider string) error {
	connector, err := s.repo.GetByUserAndProvider(ctx, userID, provider)
	if err != nil {
		return ErrConnectorNotFound
	}

	if connector.UserID != userID {
		return ErrUnauthorized
	}

	return s.repo.Delete(ctx, connector.ID)
}

// List lists all connectors for a user.
func (s *Service) List(ctx context.Context, userID string) ([]*Connector, error) {
	return s.repo.ListByUser(ctx, userID)
}

// GetAccessToken retrieves and decrypts the access token for a connector.
// It will refresh the token if expired.
func (s *Service) GetAccessToken(ctx context.Context, userID, provider string) (string, error) {
	connector, err := s.repo.GetByUserAndProvider(ctx, userID, provider)
	if err != nil {
		return "", ErrConnectorNotFound
	}

	// Check if token is expired
	if connector.ExpiresAt != nil && time.Now().After(*connector.ExpiresAt) {
		// Try to refresh
		if connector.RefreshToken != "" {
			refreshed, err := s.refreshToken(ctx, connector)
			if err == nil {
				connector = refreshed
			}
		}
	}

	// Decrypt access token
	accessToken, err := s.encryptor.Decrypt(connector.AccessToken)
	if err != nil {
		return "", fmt.Errorf("decrypt access token: %w", err)
	}

	return accessToken, nil
}

func (s *Service) refreshToken(ctx context.Context, connector *Connector) (*Connector, error) {
	client, ok := s.oauthClients[connector.Provider]
	if !ok {
		return nil, ErrProviderNotEnabled
	}

	// Decrypt refresh token
	refreshToken, err := s.encryptor.Decrypt(connector.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("decrypt refresh token: %w", err)
	}

	// Refresh tokens
	tokens, err := client.RefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}

	// Encrypt new tokens
	encAccessToken, err := s.encryptor.Encrypt(tokens.AccessToken)
	if err != nil {
		return nil, err
	}

	var encRefreshToken string
	if tokens.RefreshToken != "" {
		encRefreshToken, err = s.encryptor.Encrypt(tokens.RefreshToken)
		if err != nil {
			return nil, err
		}
	} else {
		encRefreshToken = connector.RefreshToken // Keep old refresh token
	}

	// Update connector
	connector.AccessToken = encAccessToken
	connector.RefreshToken = encRefreshToken
	if tokens.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
		connector.ExpiresAt = &t
	}

	if err := s.repo.Update(ctx, connector); err != nil {
		return nil, err
	}

	return connector, nil
}

func (s *Service) isProviderEnabled(provider string) bool {
	switch provider {
	case ProviderGitHub:
		return s.config.GitHubEnabled
	case ProviderGoogle, ProviderGmail, ProviderGoogleDrive, ProviderGoogleCalendar:
		return s.config.GoogleEnabled
	default:
		return false
	}
}

// BuildFrontendRedirectURL builds the redirect URL back to the frontend.
func (s *Service) BuildFrontendRedirectURL(redirectURL, provider string, success bool, errMsg string) string {
	baseURL := redirectURL
	if baseURL == "" {
		baseURL = s.config.FrontendURL + "/connectors"
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	q := u.Query()
	q.Set("provider", provider)
	if success {
		q.Set("success", "true")
	} else {
		q.Set("error", errMsg)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

// GetProviderScopes returns the scopes for a provider.
func GetProviderScopes(provider string) []string {
	switch provider {
	case ProviderGitHub:
		return []string{"read:user", "user:email", "repo"}
	case ProviderGmail:
		return []string{
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.send",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		}
	case ProviderGoogleDrive:
		return []string{
			"https://www.googleapis.com/auth/drive.readonly",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		}
	case ProviderGoogleCalendar:
		return []string{
			"https://www.googleapis.com/auth/calendar.readonly",
			"https://www.googleapis.com/auth/calendar.events",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		}
	case ProviderGoogle:
		return []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		}
	default:
		return nil
	}
}

// JoinScopes joins scopes with a space.
func JoinScopes(scopes []string) string {
	return strings.Join(scopes, " ")
}
