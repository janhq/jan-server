package connector

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

// Common errors
var (
	ErrConnectorNotFound    = errors.New("connector connection not found")
	ErrConnectorNotEnabled  = errors.New("connector type is not enabled")
	ErrInvalidConnectorType = errors.New("invalid connector type")
	ErrOAuthStateNotFound   = errors.New("OAuth state not found")
	ErrOAuthStateExpired    = errors.New("OAuth state has expired")
	ErrOAuthStateInvalid    = errors.New("OAuth state is invalid")
	ErrTokenEncryption      = errors.New("token encryption failed")
	ErrTokenDecryption      = errors.New("token decryption failed")
	ErrTokenRefreshFailed   = errors.New("token refresh failed")
	ErrAlreadyConnected     = errors.New("connector is already connected")
)

// TokenEncryptor interface for encrypting/decrypting OAuth tokens.
type TokenEncryptor interface {
	Encrypt(plaintext string) (ciphertext string, keyID string, err error)
	Decrypt(ciphertext string, keyID string) (plaintext string, err error)
}

// OAuthProvider interface for OAuth operations.
type OAuthProvider interface {
	GetAuthURL(connectorType ConnectorType, state, redirectURI, codeChallenge string) (string, error)
	ExchangeCode(ctx context.Context, connectorType ConnectorType, code, codeVerifier, redirectURI string) (*OAuthTokens, error)
	RefreshToken(ctx context.Context, connectorType ConnectorType, refreshToken string) (*OAuthTokens, error)
	GetUserInfo(ctx context.Context, connectorType ConnectorType, accessToken string) (*ProviderUserInfo, error)
	RevokeToken(ctx context.Context, connectorType ConnectorType, token string) error
}

// OAuthTokens represents tokens from OAuth exchange.
type OAuthTokens struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int // seconds until expiry
	Scope        string
}

// ProviderUserInfo represents user info from the OAuth provider.
type ProviderUserInfo struct {
	ID        string
	Username  string
	Email     string
	Name      string
	AvatarURL string
}

// ServiceConfig holds configuration for the connector service.
type ServiceConfig struct {
	GitHubEnabled        bool
	GoogleEnabled        bool
	OAuthStateExpiration time.Duration
	OAuthRedirectBaseURL string
	OAuthFrontendURL     string
}

// Service provides connector domain operations.
type Service struct {
	repo       Repository
	encryptor  TokenEncryptor
	provider   OAuthProvider
	config     ServiceConfig
	stateHMAC  []byte
	logger     zerolog.Logger
}

// NewService creates a new connector service.
func NewService(
	repo Repository,
	encryptor TokenEncryptor,
	provider OAuthProvider,
	config ServiceConfig,
	stateHMAC []byte,
	logger zerolog.Logger,
) *Service {
	return &Service{
		repo:      repo,
		encryptor: encryptor,
		provider:  provider,
		config:    config,
		stateHMAC: stateHMAC,
		logger:    logger.With().Str("component", "connector_service").Logger(),
	}
}

// IsEnabled checks if a connector type is enabled.
func (s *Service) IsEnabled(connectorType ConnectorType) bool {
	switch connectorType {
	case ConnectorTypeGitHub:
		return s.config.GitHubEnabled
	case ConnectorTypeGmail, ConnectorTypeGoogleDrive, ConnectorTypeGoogleCalendar:
		return s.config.GoogleEnabled
	default:
		return false
	}
}

// ListConnectors returns all connector types with their connection status for a user.
func (s *Service) ListConnectors(ctx context.Context, userID uint) ([]ConnectorInfo, error) {
	connections, err := s.repo.FindAllByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}

	connMap := make(map[ConnectorType]*Connection)
	for _, conn := range connections {
		connMap[conn.ConnectorType] = conn
	}

	return GetConnectorInfos(connMap), nil
}

// GetConnection returns a user's connection for a specific connector type.
func (s *Service) GetConnection(ctx context.Context, userID uint, connectorType ConnectorType) (*Connection, error) {
	if !connectorType.IsValid() {
		return nil, ErrInvalidConnectorType
	}

	conn, err := s.repo.FindByUserAndType(ctx, userID, connectorType)
	if err != nil {
		return nil, fmt.Errorf("find connection: %w", err)
	}

	return conn, nil
}

// InitiateOAuth starts the OAuth flow for a connector.
func (s *Service) InitiateOAuth(ctx context.Context, userID uint, connectorType ConnectorType) (authURL string, state string, err error) {
	if !connectorType.IsValid() {
		return "", "", ErrInvalidConnectorType
	}

	if !s.IsEnabled(connectorType) {
		return "", "", ErrConnectorNotEnabled
	}

	// Check if already connected
	existing, err := s.repo.FindByUserAndType(ctx, userID, connectorType)
	if err != nil {
		return "", "", fmt.Errorf("check existing connection: %w", err)
	}
	if existing != nil && existing.IsConnected {
		return "", "", ErrAlreadyConnected
	}

	// Generate PKCE
	verifier, challenge, err := s.generatePKCE()
	if err != nil {
		return "", "", fmt.Errorf("generate PKCE: %w", err)
	}

	// Generate state with HMAC
	stateValue, stateHash, err := s.generateState()
	if err != nil {
		return "", "", fmt.Errorf("generate state: %w", err)
	}

	// Build redirect URI
	redirectURI := fmt.Sprintf("%s/v1/connectors/%s/callback", s.config.OAuthRedirectBaseURL, connectorType)

	// Store OAuth state
	oauthState := &OAuthState{
		UserID:        userID,
		ConnectorType: connectorType,
		State:         stateValue,
		StateHash:     stateHash,
		CodeVerifier:  verifier,
		RedirectURI:   redirectURI,
		ExpiresAt:     time.Now().Add(s.config.OAuthStateExpiration),
	}

	if _, err := s.repo.CreateOAuthState(ctx, oauthState); err != nil {
		return "", "", fmt.Errorf("store OAuth state: %w", err)
	}

	// Get auth URL from provider
	authURL, err = s.provider.GetAuthURL(connectorType, stateValue, redirectURI, challenge)
	if err != nil {
		return "", "", fmt.Errorf("get auth URL: %w", err)
	}

	s.logger.Info().
		Uint("user_id", userID).
		Str("connector_type", string(connectorType)).
		Msg("OAuth flow initiated")

	return authURL, stateValue, nil
}

// CompleteOAuth completes the OAuth flow with the authorization code.
func (s *Service) CompleteOAuth(ctx context.Context, code, state string) (*Connection, string, error) {
	// Find and validate OAuth state
	oauthState, err := s.repo.FindOAuthStateByState(ctx, state)
	if err != nil {
		return nil, "", fmt.Errorf("find OAuth state: %w", err)
	}
	if oauthState == nil {
		return nil, "", ErrOAuthStateNotFound
	}

	if oauthState.IsExpired() {
		_ = s.repo.DeleteOAuthState(ctx, state)
		return nil, "", ErrOAuthStateExpired
	}

	// Verify state hash
	if !s.verifyState(state, oauthState.StateHash) {
		return nil, "", ErrOAuthStateInvalid
	}

	// Exchange code for tokens
	tokens, err := s.provider.ExchangeCode(ctx, oauthState.ConnectorType, code, oauthState.CodeVerifier, oauthState.RedirectURI)
	if err != nil {
		return nil, "", fmt.Errorf("exchange code: %w", err)
	}

	// Get user info from provider
	userInfo, err := s.provider.GetUserInfo(ctx, oauthState.ConnectorType, tokens.AccessToken)
	if err != nil {
		return nil, "", fmt.Errorf("get user info: %w", err)
	}

	// Encrypt tokens
	accessTokenEnc, keyID, err := s.encryptor.Encrypt(tokens.AccessToken)
	if err != nil {
		return nil, "", ErrTokenEncryption
	}

	var refreshTokenEnc string
	if tokens.RefreshToken != "" {
		refreshTokenEnc, _, err = s.encryptor.Encrypt(tokens.RefreshToken)
		if err != nil {
			return nil, "", ErrTokenEncryption
		}
	}

	// Calculate token expiry
	var expiresAt *time.Time
	if tokens.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
		expiresAt = &exp
	}

	// Parse scopes
	var scopes []string
	if tokens.Scope != "" {
		scopes = parseScopes(tokens.Scope)
	}

	// Create or update connection
	conn := &Connection{
		UserID:                oauthState.UserID,
		ConnectorType:         oauthState.ConnectorType,
		AccessTokenEncrypted:  accessTokenEnc,
		RefreshTokenEncrypted: refreshTokenEnc,
		EncryptionKeyID:       keyID,
		TokenType:             tokens.TokenType,
		ExpiresAt:             expiresAt,
		ProviderUserID:        userInfo.ID,
		ProviderUsername:      userInfo.Username,
		ProviderEmail:         userInfo.Email,
		ProviderAvatarURL:     userInfo.AvatarURL,
		Scopes:                scopes,
		IsConnected:           true,
	}

	// Check if connection exists
	existing, err := s.repo.FindByUserAndType(ctx, oauthState.UserID, oauthState.ConnectorType)
	if err != nil {
		return nil, "", fmt.Errorf("check existing connection: %w", err)
	}

	if existing != nil {
		conn.ID = existing.ID
		conn, err = s.repo.Update(ctx, conn)
	} else {
		conn, err = s.repo.Create(ctx, conn)
	}

	if err != nil {
		return nil, "", fmt.Errorf("save connection: %w", err)
	}

	// Clean up OAuth state
	_ = s.repo.DeleteOAuthState(ctx, state)

	// Create audit log
	_ = s.repo.CreateAuditLog(ctx, &AuditLog{
		UserID:        oauthState.UserID,
		ConnectorType: oauthState.ConnectorType,
		Action:        AuditActionConnect,
		Success:       true,
	})

	s.logger.Info().
		Uint("user_id", oauthState.UserID).
		Str("connector_type", string(oauthState.ConnectorType)).
		Str("provider_username", userInfo.Username).
		Msg("OAuth flow completed")

	// Build frontend redirect URL
	frontendRedirect := fmt.Sprintf(
		"%s/connectors/callback?status=success&connector=%s",
		s.config.OAuthFrontendURL,
		oauthState.ConnectorType,
	)

	return conn, frontendRedirect, nil
}

// Disconnect removes a connector connection.
func (s *Service) Disconnect(ctx context.Context, userID uint, connectorType ConnectorType) error {
	if !connectorType.IsValid() {
		return ErrInvalidConnectorType
	}

	conn, err := s.repo.FindByUserAndType(ctx, userID, connectorType)
	if err != nil {
		return fmt.Errorf("find connection: %w", err)
	}
	if conn == nil {
		return ErrConnectorNotFound
	}

	// Try to revoke token with provider (best effort)
	if conn.RefreshTokenEncrypted != "" {
		refreshToken, err := s.encryptor.Decrypt(conn.RefreshTokenEncrypted, conn.EncryptionKeyID)
		if err == nil {
			_ = s.provider.RevokeToken(ctx, connectorType, refreshToken)
		}
	}

	// Delete connection
	if err := s.repo.Delete(ctx, userID, connectorType); err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}

	// Create audit log
	_ = s.repo.CreateAuditLog(ctx, &AuditLog{
		UserID:        userID,
		ConnectorType: connectorType,
		Action:        AuditActionDisconnect,
		Success:       true,
	})

	s.logger.Info().
		Uint("user_id", userID).
		Str("connector_type", string(connectorType)).
		Msg("Connector disconnected")

	return nil
}

// GetDecryptedAccessToken returns the decrypted access token, refreshing if necessary.
func (s *Service) GetDecryptedAccessToken(ctx context.Context, userID uint, connectorType ConnectorType) (string, error) {
	conn, err := s.repo.FindByUserAndType(ctx, userID, connectorType)
	if err != nil {
		return "", fmt.Errorf("find connection: %w", err)
	}
	if conn == nil || !conn.IsConnected {
		return "", ErrConnectorNotFound
	}

	// Check if token needs refresh
	if conn.NeedsRefresh() && conn.RefreshTokenEncrypted != "" {
		conn, err = s.refreshConnection(ctx, conn)
		if err != nil {
			// Log error but try to use existing token
			s.logger.Warn().Err(err).
				Uint("user_id", userID).
				Str("connector_type", string(connectorType)).
				Msg("Token refresh failed, using existing token")
		}
	}

	// Decrypt access token
	accessToken, err := s.encryptor.Decrypt(conn.AccessTokenEncrypted, conn.EncryptionKeyID)
	if err != nil {
		return "", ErrTokenDecryption
	}

	return accessToken, nil
}

// refreshConnection refreshes the OAuth tokens for a connection.
func (s *Service) refreshConnection(ctx context.Context, conn *Connection) (*Connection, error) {
	if conn.RefreshTokenEncrypted == "" {
		return nil, errors.New("no refresh token available")
	}

	// Decrypt refresh token
	refreshToken, err := s.encryptor.Decrypt(conn.RefreshTokenEncrypted, conn.EncryptionKeyID)
	if err != nil {
		return nil, ErrTokenDecryption
	}

	// Refresh with provider
	tokens, err := s.provider.RefreshToken(ctx, conn.ConnectorType, refreshToken)
	if err != nil {
		// Mark connection as needing re-auth
		conn.LastError = "Token refresh failed: " + err.Error()
		_, _ = s.repo.Update(ctx, conn)
		return nil, ErrTokenRefreshFailed
	}

	// Encrypt new tokens
	accessTokenEnc, keyID, err := s.encryptor.Encrypt(tokens.AccessToken)
	if err != nil {
		return nil, ErrTokenEncryption
	}

	conn.AccessTokenEncrypted = accessTokenEnc
	conn.EncryptionKeyID = keyID

	if tokens.RefreshToken != "" {
		refreshTokenEnc, _, err := s.encryptor.Encrypt(tokens.RefreshToken)
		if err != nil {
			return nil, ErrTokenEncryption
		}
		conn.RefreshTokenEncrypted = refreshTokenEnc
	}

	// Update expiry
	if tokens.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
		conn.ExpiresAt = &exp
	}

	conn.LastError = ""
	now := time.Now()
	conn.LastSyncAt = &now

	// Save updated connection
	conn, err = s.repo.Update(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("update connection: %w", err)
	}

	// Create audit log
	_ = s.repo.CreateAuditLog(ctx, &AuditLog{
		UserID:        conn.UserID,
		ConnectorType: conn.ConnectorType,
		Action:        AuditActionRefresh,
		Success:       true,
	})

	s.logger.Debug().
		Uint("user_id", conn.UserID).
		Str("connector_type", string(conn.ConnectorType)).
		Msg("Token refreshed")

	return conn, nil
}

// LogAPICall logs an API call for auditing.
func (s *Service) LogAPICall(ctx context.Context, userID uint, connectorType ConnectorType, toolName string, success bool, errMsg string) {
	_ = s.repo.CreateAuditLog(ctx, &AuditLog{
		UserID:        userID,
		ConnectorType: connectorType,
		Action:        AuditActionAPICall,
		ToolName:      toolName,
		Success:       success,
		ErrorMessage:  errMsg,
	})
}

// LogWriteOperation logs a write operation for auditing (higher visibility).
func (s *Service) LogWriteOperation(ctx context.Context, userID uint, connectorType ConnectorType, toolName string, success bool, errMsg string, metadata map[string]interface{}) {
	_ = s.repo.CreateAuditLog(ctx, &AuditLog{
		UserID:        userID,
		ConnectorType: connectorType,
		Action:        AuditActionWriteOperation,
		ToolName:      toolName,
		Success:       success,
		ErrorMessage:  errMsg,
		Metadata:      metadata,
	})
}

// CleanupExpiredStates removes expired OAuth states.
func (s *Service) CleanupExpiredStates(ctx context.Context) (int64, error) {
	return s.repo.DeleteExpiredOAuthStates(ctx)
}

// generatePKCE generates a PKCE code verifier and challenge.
func (s *Service) generatePKCE() (verifier, challenge string, err error) {
	// Generate 32 bytes of random data for verifier
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}

	verifier = base64.RawURLEncoding.EncodeToString(b)

	// Generate challenge as SHA256 of verifier
	h := sha256.New()
	h.Write([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return verifier, challenge, nil
}

// generateState generates a random state with HMAC signature.
func (s *Service) generateState() (state, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}

	state = base64.RawURLEncoding.EncodeToString(b)
	hash = s.computeStateHash(state)

	return state, hash, nil
}

// computeStateHash computes HMAC-SHA256 of the state.
func (s *Service) computeStateHash(state string) string {
	h := sha256.New()
	h.Write(s.stateHMAC)
	h.Write([]byte(state))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// verifyState verifies the state against its HMAC hash.
func (s *Service) verifyState(state, expectedHash string) bool {
	return s.computeStateHash(state) == expectedHash
}

// parseScopes parses a space-separated scope string into a slice.
func parseScopes(scopeStr string) []string {
	if scopeStr == "" {
		return nil
	}
	var scopes []string
	for _, s := range splitScopes(scopeStr) {
		if s != "" {
			scopes = append(scopes, s)
		}
	}
	return scopes
}

// splitScopes splits scopes by space or comma.
func splitScopes(s string) []string {
	var result []string
	var current string
	for _, c := range s {
		if c == ' ' || c == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
