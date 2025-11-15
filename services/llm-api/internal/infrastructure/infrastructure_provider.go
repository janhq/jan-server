package infrastructure

import (
	"context"
	"net/http"
	"time"

	"github.com/google/wire"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/infrastructure/auth"
	"jan-server/services/llm-api/internal/infrastructure/crontab"
	"jan-server/services/llm-api/internal/infrastructure/database"
	"jan-server/services/llm-api/internal/infrastructure/database/repository"
	"jan-server/services/llm-api/internal/infrastructure/database/transaction"
	"jan-server/services/llm-api/internal/infrastructure/inference"
	"jan-server/services/llm-api/internal/infrastructure/keycloak"
	"jan-server/services/llm-api/internal/infrastructure/kong"
	"jan-server/services/llm-api/internal/infrastructure/logger"
	"jan-server/services/llm-api/internal/infrastructure/mediaresolver"
)

// ProvideConfig loads and provides the application configuration
func ProvideConfig() (*config.Config, error) {
	return config.Load()
}

// ProvideKeycloakClient provides a keycloak client
func ProvideKeycloakClient(cfg *config.Config, log zerolog.Logger) *keycloak.Client {
	return keycloak.NewClient(
		cfg.KeycloakBaseURL,
		cfg.KeycloakRealm,
		cfg.BackendClientID,
		cfg.BackendClientSecret,
		cfg.TargetClientID,
		cfg.GuestRole,
		&http.Client{},
		log,
		cfg.KeycloakAdminUser,
		cfg.KeycloakAdminPass,
		cfg.KeycloakAdminRealm,
		cfg.KeycloakAdminClient,
		cfg.KeycloakAdminSecret,
	)
}

// ProvideKongClient returns a Kong Admin API client.
func ProvideKongClient(cfg *config.Config, log zerolog.Logger) *kong.Client {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	return kong.NewClient(cfg.KongAdminURL, httpClient, log)
}

// ProvideKeycloakValidator provides a JWT validator
func ProvideKeycloakValidator(cfg *config.Config, log zerolog.Logger) (*auth.KeycloakValidator, error) {
	jwksURL := cfg.JWKSURL
	return auth.NewKeycloakValidator(
		context.Background(),
		jwksURL,
		cfg.Issuer,
		cfg.Audience,
		cfg.TargetClientID,
		cfg.RefreshJWKSInterval,
		cfg.AuthClockSkew,
		log,
	)
}

// ProvideDatabase provides a database connection
func ProvideDatabase(cfg *config.Config, log zerolog.Logger) (*gorm.DB, error) {
	db, err := database.NewDB(cfg.GetDatabaseWriteDSN())
	if err != nil {
		return nil, err
	}

	// Run migrations if AUTO_MIGRATE is enabled
	if cfg.AutoMigrate {
		log.Info().Msg("Running database migrations...")
		if err := database.AutoMigrate(db); err != nil {
			log.Error().Err(err).Msg("Failed to run database migrations")
			return nil, err
		}
		log.Info().Msg("Database migrations completed successfully")
	}

	return db, nil
}

// ProvideTransactionDatabase provides a transaction database wrapper
func ProvideTransactionDatabase(db *gorm.DB) *transaction.Database {
	return transaction.NewDatabase(db)
}

// ProvideMediaResolver wires the HTTP-based media placeholder resolver.
func ProvideMediaResolver(cfg *config.Config, log zerolog.Logger, kc *keycloak.Client) mediaresolver.Resolver {
	return mediaresolver.NewResolver(cfg, log, kc)
}

// Infrastructure holds all infrastructure dependencies
type Infrastructure struct {
	DB                *gorm.DB
	KeycloakValidator *auth.KeycloakValidator
	Logger            zerolog.Logger
}

// NewInfrastructure creates a new infrastructure instance
func NewInfrastructure(
	db *gorm.DB,
	keycloakValidator *auth.KeycloakValidator,
	logger zerolog.Logger,
) *Infrastructure {
	return &Infrastructure{
		DB:                db,
		KeycloakValidator: keycloakValidator,
		Logger:            logger,
	}
}

// InfrastructureProvider provides all infrastructure dependencies
var InfrastructureProvider = wire.NewSet(
	// Config
	ProvideConfig,

	// Database
	ProvideDatabase,
	ProvideTransactionDatabase,

	// Repositories
	repository.RepositoryProvider,

	// Provider registry
	inference.NewInferenceProvider,

	// Media resolver
	ProvideMediaResolver,

	// Logger
	logger.GetLogger,

	// Kong client removed - API keys now managed via Keycloak
	// ProvideKongClient,

	// Keycloak
	ProvideKeycloakClient,
	ProvideKeycloakValidator,

	// Crontab for model sync
	crontab.NewCrontab,

	// Infrastructure struct
	NewInfrastructure,
)
