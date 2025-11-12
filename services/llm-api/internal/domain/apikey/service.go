package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"jan-server/services/llm-api/internal/domain/user"
	"jan-server/services/llm-api/internal/infrastructure/keycloak"
)

// ErrLimitExceeded indicates the user hit the maximum number of active API keys.
var ErrLimitExceeded = errors.New("api key limit exceeded")

// ErrNotFound indicates the API key does not exist or does not belong to user.
var ErrNotFound = errors.New("api key not found")

// Service orchestrates API key lifecycle operations.
type Service struct {
	repo       Repository
	userRepo   user.Repository
	keycloak   *keycloak.Client
	logger     zerolog.Logger
	defaultTTL time.Duration
	maxTTL     time.Duration
	maxPerUser int
	keyPrefix  string
}

// Config configures the Service.
type Config struct {
	DefaultTTL time.Duration
	MaxTTL     time.Duration
	MaxPerUser int
	KeyPrefix  string
}

// NewService constructs an API key service.
func NewService(repo Repository, userRepo user.Repository, keycloakClient *keycloak.Client, cfg Config, logger zerolog.Logger) *Service {
	return &Service{
		repo:       repo,
		userRepo:   userRepo,
		keycloak:   keycloakClient,
		logger:     logger.With().Str("component", "api-key-service").Logger(),
		defaultTTL: cfg.DefaultTTL,
		maxTTL:     cfg.MaxTTL,
		maxPerUser: cfg.MaxPerUser,
		keyPrefix:  cfg.KeyPrefix,
	}
}

// CreateKey generates a new API key for the given user and persists metadata.
func (s *Service) CreateKey(ctx context.Context, usr *user.User, name string, requestedTTL time.Duration) (*APIKey, string, error) {
	if usr == nil || usr.ID == 0 {
		return nil, "", fmt.Errorf("user is required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", fmt.Errorf("name is required")
	}

	count, err := s.repo.CountActiveByUser(ctx, usr.ID)
	if err != nil {
		return nil, "", err
	}
	if s.maxPerUser > 0 && count >= int64(s.maxPerUser) {
		return nil, "", ErrLimitExceeded
	}

	ttl := s.defaultTTL
	if requestedTTL > 0 && requestedTTL < s.maxTTL {
		ttl = requestedTTL
	} else if requestedTTL > s.maxTTL {
		ttl = s.maxTTL
	}
	expiresAt := time.Now().Add(ttl)

	rawKey, err := s.generateKeySecret()
	if err != nil {
		return nil, "", err
	}
	displaySuffix := ""
	if len(rawKey) >= 4 {
		displaySuffix = rawKey[len(rawKey)-4:]
	}

	// Hash the API key for storage
	keyHash := hashKey(rawKey)

	record := &APIKey{
		ID:        uuid.NewString(),
		UserID:    usr.ID,
		Name:      name,
		Prefix:    s.keyPrefix,
		Suffix:    displaySuffix,
		Hash:      keyHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	persisted, err := s.repo.Create(ctx, record)
	if err != nil {
		return nil, "", err
	}

	// Store API key hash in Keycloak user attributes
	if s.keycloak != nil {
		if err := s.keycloak.StoreAPIKeyHash(ctx, usr.Subject, record.ID, keyHash); err != nil {
			s.logger.Warn().Err(err).Str("user_id", usr.Subject).Msg("failed to store api key in keycloak")
			// Continue - we have it in database
		}
	}

	return persisted, rawKey, nil
}

// ListKeys returns API keys for the provided user.
func (s *Service) ListKeys(ctx context.Context, userID uint) ([]APIKey, error) {
	items, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// RevokeKey marks the API key as revoked and removes it from Keycloak.
func (s *Service) RevokeKey(ctx context.Context, usr *user.User, keyID string) error {
	if usr == nil {
		return fmt.Errorf("user is required")
	}

	key, err := s.repo.FindByID(ctx, keyID)
	if err != nil {
		return err
	}
	if key == nil || key.UserID != usr.ID {
		return ErrNotFound
	}

	// Mark as revoked in database
	if err := s.repo.MarkRevoked(ctx, key.ID, time.Now()); err != nil {
		return fmt.Errorf("mark revoked: %w", err)
	}

	if s.keycloak != nil {
		if usr.Subject != "" {
			if err := s.keycloak.RemoveAPIKeyHash(ctx, usr.Subject, key.ID); err != nil {
				s.logger.Warn().
					Err(err).
					Str("key_id", key.ID).
					Str("user_subject", usr.Subject).
					Msg("failed to remove api key hash from keycloak")
			}
		} else {
			s.logger.Warn().
				Str("key_id", key.ID).
				Uint("user_id", usr.ID).
				Msg("api key revoked but user subject missing; unable to remove from keycloak")
		}
	}

	return nil
}

// ValidateAPIKey validates an API key using a hybrid approach:
// 1. Fast database lookup to find the API key and user
// 2. Verify key hasn't expired or been revoked
// 3. Double-check user status in Keycloak (enabled, not deleted)
// 4. Return user info if all checks pass
func (s *Service) ValidateAPIKey(ctx context.Context, apiKey string) (*keycloak.APIKeyUserInfo, error) {
	// Step 1: Fast database lookup
	keyHash := hashKey(apiKey)

	key, err := s.repo.FindByHash(ctx, keyHash)
	if err != nil {
		return nil, fmt.Errorf("find api key: %w", err)
	}
	if key == nil {
		s.logger.Debug().Str("key_hash_prefix", keyHash[:8]+"...").Msg("api key not found in database")
		return nil, errors.New("invalid api key")
	}

	// Step 2: Check if revoked or expired (fast database checks)
	if key.RevokedAt != nil {
		s.logger.Debug().
			Str("key_id", key.ID).
			Time("revoked_at", *key.RevokedAt).
			Msg("api key has been revoked")
		return nil, errors.New("api key revoked")
	}

	if time.Now().After(key.ExpiresAt) {
		s.logger.Debug().
			Str("key_id", key.ID).
			Time("expired_at", key.ExpiresAt).
			Msg("api key has expired")
		return nil, errors.New("api key expired")
	}

	// Step 3: Load user from database
	usr, err := s.userRepo.FindByID(ctx, key.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if usr == nil {
		s.logger.Warn().
			Uint("user_id", key.UserID).
			Str("key_id", key.ID).
			Msg("api key references non-existent user")
		return nil, errors.New("user not found")
	}

	// Step 4: Double-check user status in Keycloak
	// This ensures the user is still enabled and exists in Keycloak
	if s.keycloak != nil && usr.Subject != "" {
		keycloakUser, err := s.keycloak.GetUserBySubject(ctx, usr.Subject)
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("subject", usr.Subject).
				Str("key_id", key.ID).
				Msg("failed to verify user in keycloak")
			return nil, fmt.Errorf("verify user status: %w", err)
		}

		// Verify user is enabled in Keycloak
		if !keycloakUser.Enabled {
			s.logger.Warn().
				Str("subject", usr.Subject).
				Str("username", *usr.Username).
				Str("key_id", key.ID).
				Msg("user is disabled in keycloak")
			return nil, errors.New("user account is disabled")
		}

		// Step 5: All checks passed - return user info with Keycloak roles
		s.logger.Debug().
			Str("key_id", key.ID).
			Str("user_id", fmt.Sprintf("%d", usr.ID)).
			Str("username", *usr.Username).
			Msg("api key validated successfully")

		return &keycloak.APIKeyUserInfo{
			UserID:    fmt.Sprintf("%d", usr.ID),
			Subject:   usr.Subject,
			Username:  ptrToString(usr.Username),
			Email:     ptrToString(usr.Email),
			FirstName: keycloakUser.FirstName,
			LastName:  keycloakUser.LastName,
			Roles:     keycloakUser.Roles,
		}, nil
	}

	// If Keycloak is not configured or user has no subject, return basic user info
	s.logger.Debug().
		Str("key_id", key.ID).
		Str("user_id", fmt.Sprintf("%d", usr.ID)).
		Msg("api key validated successfully (no keycloak verification)")

	return &keycloak.APIKeyUserInfo{
		UserID:   fmt.Sprintf("%d", usr.ID),
		Subject:  usr.Subject,
		Username: ptrToString(usr.Username),
		Email:    ptrToString(usr.Email),
		Roles:    []string{},
	}, nil
}

func (s *Service) generateKeySecret() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	randomPart := hex.EncodeToString(buf)
	prefix := strings.TrimSpace(s.keyPrefix)
	if prefix == "" {
		prefix = "sk"
	}
	return fmt.Sprintf("%s_%s", prefix, randomPart), nil
}

func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
