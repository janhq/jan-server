package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrAPIKeyNotFound     = errors.New("api key not found")
	ErrAPIKeyExpired      = errors.New("api key expired")
	ErrAPIKeyRevoked      = errors.New("api key revoked")
	ErrMaxAPIKeysReached  = errors.New("maximum number of API keys reached")
)

// Repository defines the interface for user data operations.
type Repository interface {
	// User operations
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, offset, limit int) ([]*User, int64, error)

	// Refresh token operations
	CreateRefreshToken(ctx context.Context, token *RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, hash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id string) error
	RevokeAllRefreshTokens(ctx context.Context, userID string) error
	DeleteExpiredRefreshTokens(ctx context.Context) error

	// API key operations
	CreateAPIKey(ctx context.Context, key *APIKey) error
	GetAPIKeyByHash(ctx context.Context, hash string) (*APIKey, error)
	ListAPIKeysByUser(ctx context.Context, userID string) ([]*APIKey, error)
	CountAPIKeysByUser(ctx context.Context, userID string) (int64, error)
	RevokeAPIKey(ctx context.Context, id string) error
	UpdateAPIKeyLastUsed(ctx context.Context, id string) error
}

// Service handles user-related business logic.
type Service struct {
	repo            Repository
	jwtSecret       []byte
	jwtIssuer       string
	jwtExpiration   time.Duration
	refreshTokenTTL time.Duration
	bcryptCost      int
	apiKeyPrefix    string
	apiKeyMaxPerUser int
	apiKeyDefaultTTL time.Duration
}

// ServiceConfig holds configuration for the user service.
type ServiceConfig struct {
	JWTSecret        string
	JWTIssuer        string
	JWTExpiration    time.Duration
	RefreshTokenTTL  time.Duration
	BcryptCost       int
	APIKeyPrefix     string
	APIKeyMaxPerUser int
	APIKeyDefaultTTL time.Duration
}

// NewService creates a new user service.
func NewService(repo Repository, cfg ServiceConfig) *Service {
	return &Service{
		repo:            repo,
		jwtSecret:       []byte(cfg.JWTSecret),
		jwtIssuer:       cfg.JWTIssuer,
		jwtExpiration:   cfg.JWTExpiration,
		refreshTokenTTL: cfg.RefreshTokenTTL,
		bcryptCost:      cfg.BcryptCost,
		apiKeyPrefix:    cfg.APIKeyPrefix,
		apiKeyMaxPerUser: cfg.APIKeyMaxPerUser,
		apiKeyDefaultTTL: cfg.APIKeyDefaultTTL,
	}
}

// Register creates a new user account.
func (s *Service) Register(ctx context.Context, req CreateUserRequest) (*User, *TokenPair, error) {
	// Check if email already exists
	existing, err := s.repo.GetByEmail(ctx, strings.ToLower(req.Email))
	if err == nil && existing != nil {
		return nil, nil, ErrUserAlreadyExists
	}

	// Check if username already exists
	existing, err = s.repo.GetByUsername(ctx, strings.ToLower(req.Username))
	if err == nil && existing != nil {
		return nil, nil, ErrUserAlreadyExists
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	// Create user
	isActive := true
	user := &User{
		Email:        strings.ToLower(req.Email),
		Username:     strings.ToLower(req.Username),
		PasswordHash: string(passwordHash),
		Name:         req.Name,
		IsActive:     isActive,
		IsAdmin:      false,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	// Generate tokens
	tokens, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	return user, tokens, nil
}

// Login authenticates a user and returns tokens.
func (s *Service) Login(ctx context.Context, email, password string) (*User, *TokenPair, error) {
	user, err := s.repo.GetByEmail(ctx, strings.ToLower(email))
	if err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, nil, ErrUserInactive
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	_ = s.repo.Update(ctx, user)

	// Generate tokens
	tokens, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	return user, tokens, nil
}

// RefreshTokens generates new tokens from a refresh token.
func (s *Service) RefreshTokens(ctx context.Context, refreshToken string) (*User, *TokenPair, error) {
	// Hash the refresh token
	tokenHash := hashToken(refreshToken)

	// Look up refresh token
	rt, err := s.repo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, nil, ErrInvalidToken
	}

	if rt.IsExpired() {
		return nil, nil, ErrTokenExpired
	}

	if rt.IsRevoked() {
		return nil, nil, ErrInvalidToken
	}

	// Get user
	user, err := s.repo.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, nil, ErrUserInactive
	}

	// Revoke old refresh token
	_ = s.repo.RevokeRefreshToken(ctx, rt.ID)

	// Generate new tokens
	tokens, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	return user, tokens, nil
}

// ChangePassword changes a user's password.
func (s *Service) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	// Hash new password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user.PasswordHash = string(passwordHash)
	if err := s.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	// Revoke all refresh tokens for this user
	_ = s.repo.RevokeAllRefreshTokens(ctx, userID)

	return nil
}

// GetUserByID retrieves a user by ID.
func (s *Service) GetUserByID(ctx context.Context, id string) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

// CreateAPIKey creates a new API key for a user.
func (s *Service) CreateAPIKey(ctx context.Context, userID string, req CreateAPIKeyRequest) (*APIKeyResponse, error) {
	// Check max API keys
	count, err := s.repo.CountAPIKeysByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count api keys: %w", err)
	}
	if count >= int64(s.apiKeyMaxPerUser) {
		return nil, ErrMaxAPIKeysReached
	}

	// Generate random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	// Format: sk_live_<base64>
	plainKey := fmt.Sprintf("%s_%s", s.apiKeyPrefix, base64.RawURLEncoding.EncodeToString(keyBytes))
	keyHash := hashToken(plainKey)
	keyPrefix := plainKey[:min(20, len(plainKey))]

	expiresAt := req.ExpiresAt
	if expiresAt == nil && s.apiKeyDefaultTTL > 0 {
		t := time.Now().Add(s.apiKeyDefaultTTL)
		expiresAt = &t
	}

	apiKey := &APIKey{
		UserID:    userID,
		Name:      req.Name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Scopes:    req.Scopes,
		ExpiresAt: expiresAt,
	}

	if err := s.repo.CreateAPIKey(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}

	return &APIKeyResponse{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		KeyPrefix: apiKey.KeyPrefix,
		PlainKey:  plainKey, // Only returned once
		Scopes:    apiKey.Scopes,
		ExpiresAt: apiKey.ExpiresAt,
		CreatedAt: apiKey.CreatedAt,
	}, nil
}

// ListAPIKeys lists API keys for a user.
func (s *Service) ListAPIKeys(ctx context.Context, userID string) ([]*APIKeyResponse, error) {
	keys, err := s.repo.ListAPIKeysByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}

	responses := make([]*APIKeyResponse, len(keys))
	for i, key := range keys {
		responses[i] = &APIKeyResponse{
			ID:        key.ID,
			Name:      key.Name,
			KeyPrefix: key.KeyPrefix,
			Scopes:    key.Scopes,
			ExpiresAt: key.ExpiresAt,
			CreatedAt: key.CreatedAt,
		}
	}

	return responses, nil
}

// RevokeAPIKey revokes an API key.
func (s *Service) RevokeAPIKey(ctx context.Context, userID, keyID string) error {
	return s.repo.RevokeAPIKey(ctx, keyID)
}

// ValidateAPIKey validates an API key and returns the associated user.
func (s *Service) ValidateAPIKey(ctx context.Context, plainKey string) (*User, *APIKey, error) {
	keyHash := hashToken(plainKey)

	key, err := s.repo.GetAPIKeyByHash(ctx, keyHash)
	if err != nil {
		return nil, nil, ErrAPIKeyNotFound
	}

	if key.IsExpired() {
		return nil, nil, ErrAPIKeyExpired
	}

	if key.IsRevoked() {
		return nil, nil, ErrAPIKeyRevoked
	}

	// Get user
	user, err := s.repo.GetByID(ctx, key.UserID)
	if err != nil {
		return nil, nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, nil, ErrUserInactive
	}

	// Update last used
	_ = s.repo.UpdateAPIKeyLastUsed(ctx, key.ID)

	return user, key, nil
}

// generateTokenPair creates access and refresh tokens for a user.
func (s *Service) generateTokenPair(ctx context.Context, user *User) (*TokenPair, error) {
	now := time.Now()
	expiresAt := now.Add(s.jwtExpiration)

	// Build roles
	roles := []string{"user"}
	if user.IsAdmin {
		roles = append(roles, "admin")
	}

	// Create access token
	claims := jwt.MapClaims{
		"sub":                user.ID,
		"email":              user.Email,
		"preferred_username": user.Username,
		"name":               user.Name,
		"roles":              roles,
		"iss":                s.jwtIssuer,
		"iat":                now.Unix(),
		"exp":                expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// Create refresh token
	refreshBytes := make([]byte, 32)
	if _, err := rand.Read(refreshBytes); err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshToken := base64.RawURLEncoding.EncodeToString(refreshBytes)
	refreshHash := hashToken(refreshToken)

	rt := &RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: now.Add(s.refreshTokenTTL),
	}

	if err := s.repo.CreateRefreshToken(ctx, rt); err != nil {
		return nil, fmt.Errorf("create refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwtExpiration.Seconds()),
		ExpiresAt:    expiresAt,
	}, nil
}

// hashToken creates a SHA256 hash of a token.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// EnsureUser creates or updates a user based on the given identity.
// This is used to ensure a user record exists for OIDC-authenticated principals.
func (s *Service) EnsureUser(ctx context.Context, identity Identity) (*User, error) {
	if identity.Issuer == "" || identity.Subject == "" {
		return nil, ErrInvalidIdentity
	}

	// Use issuer + subject as a stable identifier
	// This creates a user ID based on the external identity
	userID := identity.Subject

	// Try to find existing user by ID (subject)
	existing, err := s.repo.GetByID(ctx, userID)
	if err == nil && existing != nil {
		// Update fields if they changed
		needUpdate := false
		if identity.Email != nil && *identity.Email != existing.Email {
			existing.Email = *identity.Email
			needUpdate = true
		}
		if identity.Username != nil && *identity.Username != existing.Username {
			existing.Username = *identity.Username
			needUpdate = true
		}
		if identity.Name != nil && *identity.Name != existing.Name {
			existing.Name = *identity.Name
			needUpdate = true
		}
		if needUpdate {
			if err := s.repo.Update(ctx, existing); err != nil {
				return nil, fmt.Errorf("update user: %w", err)
			}
		}
		return existing, nil
	}

	// Create new user
	email := ""
	if identity.Email != nil {
		email = *identity.Email
	}
	username := userID
	if identity.Username != nil {
		username = *identity.Username
	}
	name := ""
	if identity.Name != nil {
		name = *identity.Name
	}

	user := &User{
		ID:       userID,
		Email:    email,
		Username: username,
		Name:     name,
		IsActive: true,
		IsAdmin:  false,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}
