package connector

import (
	"encoding/hex"
	"time"

	"github.com/google/wire"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/domain/connector"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/connectorrepo"
)

// ProviderSet is the wire provider set for connector infrastructure.
var ProviderSet = wire.NewSet(
	ProvideTokenEncryptor,
	ProvideOAuthProvider,
	ProvideConnectorService,
)

// ProvideTokenEncryptor creates a token encryptor from config.
func ProvideTokenEncryptor(cfg *config.Config) (*TokenEncryptor, error) {
	if cfg.ConnectorTokenEncryptionKey == "" {
		log.Warn().Msg("ConnectorTokenEncryptionKey is not set, using insecure default key. Do NOT use this in production.")
		cfg.ConnectorTokenEncryptionKey = "0000000000000000000000000000000000000000000000000000000000000000"
	}

	return NewTokenEncryptor(
		cfg.ConnectorTokenEncryptionKey,
		cfg.ConnectorTokenEncryptionKeyID,
		cfg.ConnectorTokenEncryptionKeyPrevious,
		cfg.ConnectorTokenEncryptionKeyIDPrevious,
	)
}

// ProvideOAuthProvider creates an OAuth provider from config.
func ProvideOAuthProvider(cfg *config.Config) *OAuthProvider {
	return NewOAuthProvider(OAuthProviderConfig{
		GitHubClientID:     cfg.GitHubClientID,
		GitHubClientSecret: cfg.GitHubClientSecret,
		GoogleClientID:     cfg.GoogleClientID,
		GoogleClientSecret: cfg.GoogleClientSecret,
	})
}

// ProvideConnectorService creates a connector service with all dependencies.
func ProvideConnectorService(
	db *gorm.DB,
	cfg *config.Config,
	encryptor *TokenEncryptor,
	provider *OAuthProvider,
	logger zerolog.Logger,
) *connector.Service {
	repo := connectorrepo.NewConnectorRepository(db)

	// Parse state HMAC secret
	stateHMAC := []byte(cfg.OAuthStateSecret)
	if len(stateHMAC) == 64 {
		// If it looks like hex, decode it
		if decoded, err := hex.DecodeString(cfg.OAuthStateSecret); err == nil {
			stateHMAC = decoded
		}
	}
	if len(stateHMAC) == 0 {
		logger.Warn().Msg("OAuthStateSecret is not set, using insecure default secret. Do NOT use this in production.")
		stateHMAC = []byte("development-oauth-state-secret-key-32b")
	}

	serviceConfig := connector.ServiceConfig{
		GitHubEnabled:        cfg.GitHubConnectorEnabled,
		GoogleEnabled:        cfg.GoogleConnectorEnabled,
		OAuthStateExpiration: cfg.OAuthStateExpiration,
		OAuthRedirectBaseURL: cfg.OAuthRedirectBaseURL,
		OAuthFrontendURL:     cfg.OAuthFrontendURL,
	}

	if serviceConfig.OAuthStateExpiration == 0 {
		serviceConfig.OAuthStateExpiration = 5 * time.Minute
	}

	return connector.NewService(
		repo,
		encryptor,
		provider,
		serviceConfig,
		stateHMAC,
		logger,
	)
}
