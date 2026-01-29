package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
)

var Version = "dev"

// Config holds all configuration for the unified backend.
// Combines settings from llm-api, response-api, media-api, mcp-tools, memory-tools, and realtime-api.
type Config struct {
	// ============================================
	// HTTP Server
	// ============================================
	HTTPPort    int `env:"HTTP_PORT" envDefault:"8080"`
	MetricsPort int `env:"METRICS_PORT" envDefault:"9091"`

	// ============================================
	// Database (Single shared DB with table prefixes)
	// ============================================
	DBPostgresqlWriteDSN string `env:"DB_POSTGRESQL_WRITE_DSN,notEmpty"`
	DBPostgresqlRead1DSN string `env:"DB_POSTGRESQL_READ1_DSN"` // Optional read replica

	// Table prefixes for each domain
	DBPrefixLLM      string `env:"DB_PREFIX_LLM" envDefault:"llm_"`
	DBPrefixResponse string `env:"DB_PREFIX_RESPONSE" envDefault:"resp_"`
	DBPrefixMedia    string `env:"DB_PREFIX_MEDIA" envDefault:"media_"`
	DBPrefixMemory   string `env:"DB_PREFIX_MEMORY" envDefault:"mem_"`

	// ============================================
	// Authentication - Dual Auth Support
	// ============================================

	// Keycloak Auth (optional - can be disabled for local-only deployments)
	KeycloakEnabled     bool   `env:"KEYCLOAK_ENABLED" envDefault:"true"`
	KeycloakBaseURL     string `env:"KEYCLOAK_BASE_URL"`
	KeycloakPublicURL   string `env:"KEYCLOAK_PUBLIC_URL"` // Browser-accessible URL
	KeycloakRealm       string `env:"KEYCLOAK_REALM" envDefault:"jan"`
	BackendClientID     string `env:"BACKEND_CLIENT_ID"`
	BackendClientSecret string `env:"BACKEND_CLIENT_SECRET"`
	Client              string `env:"CLIENT"`
	OAuthRedirectURI    string `env:"OAUTH_REDIRECT_URI"`
	GuestRole           string `env:"GUEST_ROLE" envDefault:"guest"`
	KeycloakAdminUser   string `env:"KEYCLOAK_ADMIN"`
	KeycloakAdminPass   string `env:"KEYCLOAK_ADMIN_PASSWORD"`
	KeycloakAdminRealm  string `env:"KEYCLOAK_ADMIN_REALM" envDefault:"master"`
	KeycloakAdminClient string `env:"KEYCLOAK_ADMIN_CLIENT_ID" envDefault:"admin-cli"`
	KeycloakAdminSecret string `env:"KEYCLOAK_ADMIN_CLIENT_SECRET"`
	JWKSURL             string `env:"JWKS_URL"`
	OIDCDiscoveryURL    string `env:"OIDC_DISCOVERY_URL"`
	Issuer              string `env:"ISSUER"`
	Account             string `env:"ACCOUNT"`
	RefreshJWKSInterval time.Duration `env:"JWKS_REFRESH_INTERVAL" envDefault:"5m"`
	AuthClockSkew       time.Duration `env:"AUTH_CLOCK_SKEW" envDefault:"60s"`

	// Local Auth (PostgreSQL + bcrypt)
	LocalAuthEnabled   bool          `env:"LOCAL_AUTH_ENABLED" envDefault:"true"`
	LocalJWTSecret     string        `env:"LOCAL_JWT_SECRET"` // HS256 secret for local JWTs
	LocalJWTIssuer     string        `env:"LOCAL_JWT_ISSUER" envDefault:"jan-server-local"`
	LocalJWTExpiration time.Duration `env:"LOCAL_JWT_EXPIRATION" envDefault:"24h"`
	LocalJWTRefreshTTL time.Duration `env:"LOCAL_JWT_REFRESH_TTL" envDefault:"168h"` // 7 days
	BcryptCost         int           `env:"BCRYPT_COST" envDefault:"12"`

	// API Keys
	APIKeySecret     []byte        `env:"APIKEY_SECRET"`
	APIKeyDefaultTTL time.Duration `env:"API_KEY_DEFAULT_TTL" envDefault:"2160h"` // 90 days
	APIKeyMaxTTL     time.Duration `env:"API_KEY_MAX_TTL" envDefault:"2160h"`
	APIKeyMaxPerUser int           `env:"API_KEY_MAX_PER_USER" envDefault:"5"`
	APIKeyPrefix     string        `env:"API_KEY_PREFIX" envDefault:"sk_live"`

	// ============================================
	// Kong Gateway (Optional)
	// ============================================
	KongEnabled  bool   `env:"KONG_ENABLED" envDefault:"false"`
	KongAdminURL string `env:"KONG_ADMIN_URL" envDefault:"http://kong:8001"`

	// ============================================
	// Observability / Logging
	// ============================================
	HTTPTimeout      time.Duration `env:"HTTP_TIMEOUT" envDefault:"30s"`
	OTLPEndpoint     string        `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OTLPHeaders      string        `env:"OTEL_EXPORTER_OTLP_HEADERS"`
	ServiceName      string        `env:"SERVICE_NAME" envDefault:"jan-server"`
	ServiceNamespace string        `env:"SERVICE_NAMESPACE" envDefault:"jan"`
	Environment      string        `env:"ENVIRONMENT" envDefault:"development"`
	LogLevel         string        `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat        string        `env:"LOG_FORMAT" envDefault:"console"`
	EnablePprof      bool          `env:"ENABLE_PPROF" envDefault:"false"`

	// ============================================
	// Features
	// ============================================
	AutoMigrate   bool `env:"AUTO_MIGRATE" envDefault:"true"`
	EnableSwagger bool `env:"ENABLE_SWAGGER" envDefault:"true"`

	// ============================================
	// LLM API Settings
	// ============================================
	StreamTimeout                      time.Duration `env:"STREAM_TIMEOUT" envDefault:"600s"`
	ModelProviderSecret                string        `env:"MODEL_PROVIDER_SECRET" envDefault:"jan-model-provider-secret-2024"`
	ModelSyncIntervalMinutes           int           `env:"MODEL_SYNC_INTERVAL_MINUTES" envDefault:"60"`
	ModelSyncEnabled                   bool          `env:"MODEL_SYNC_ENABLED" envDefault:"true"`
	ConversationSharingEnabled         bool          `env:"CONVERSATION_SHARING_ENABLED" envDefault:"false"`
	ConversationTitleGenerationEnabled bool          `env:"CONVERSATION_TITLE_GENERATION_ENABLED" envDefault:"false"`
	ConversationTitleGenerationModelID string        `env:"CONVERSATION_TITLE_GENERATION_MODEL_ID" envDefault:"LFM2-8B-A1B"`

	// Prompt Orchestration
	PromptOrchestrationEnabled         bool `env:"PROMPT_ORCHESTRATION_ENABLED" envDefault:"false"`
	PromptOrchestrationEnableMemory    bool `env:"PROMPT_ORCHESTRATION_MEMORY" envDefault:"false"`
	PromptOrchestrationEnableTemplates bool `env:"PROMPT_ORCHESTRATION_TEMPLATES" envDefault:"false"`
	PromptOrchestrationEnableTools     bool `env:"PROMPT_ORCHESTRATION_TOOLS" envDefault:"false"`

	// ============================================
	// Media API Settings (S3)
	// ============================================
	S3Endpoint        string `env:"S3_ENDPOINT"`
	S3Region          string `env:"S3_REGION" envDefault:"us-east-1"`
	S3Bucket          string `env:"S3_BUCKET" envDefault:"jan-media"`
	S3AccessKeyID     string `env:"S3_ACCESS_KEY_ID"`
	S3SecretAccessKey string `env:"S3_SECRET_ACCESS_KEY"`
	S3UsePathStyle    bool   `env:"S3_USE_PATH_STYLE" envDefault:"false"`
	MediaPresignTTL   time.Duration `env:"MEDIA_PRESIGN_TTL" envDefault:"1h"`
	MediaMaxUploadSize int64 `env:"MEDIA_MAX_UPLOAD_SIZE" envDefault:"104857600"` // 100MB

	// ============================================
	// MCP Tools Settings
	// ============================================

	// Search providers (cascading fallback: Serper → Exa → Tavily → SearXNG)
	SerperEnabled bool   `env:"SERPER_ENABLED" envDefault:"false"`
	SerperAPIKey  string `env:"SERPER_API_KEY"`

	ExaEnabled bool   `env:"EXA_ENABLED" envDefault:"false"`
	ExaAPIKey  string `env:"EXA_API_KEY"`

	TavilyEnabled bool   `env:"TAVILY_ENABLED" envDefault:"false"`
	TavilyAPIKey  string `env:"TAVILY_API_KEY"`

	SearXNGEnabled bool   `env:"SEARXNG_ENABLED" envDefault:"false"`
	SearXNGURL     string `env:"SEARXNG_URL"`

	// Sandbox (code execution)
	SandboxProvider string `env:"SANDBOX_PROVIDER" envDefault:"e2b"` // "e2b" or "aio"
	E2BAPIKey       string `env:"E2B_API_KEY"`
	AIOSandboxURL   string `env:"AIO_SANDBOX_URL"`

	// Image generation
	ImageGenerationEnabled bool          `env:"IMAGE_GENERATION_ENABLED" envDefault:"false"`
	ImageGenerationTimeout time.Duration `env:"IMAGE_GENERATION_TIMEOUT" envDefault:"120s"`
	ImageDefaultModel      string        `env:"IMAGE_DEFAULT_MODEL" envDefault:"z-image"`

	// ============================================
	// Memory Tools Settings
	// ============================================
	MemoryEnabled        bool          `env:"MEMORY_ENABLED" envDefault:"false"`
	MemoryEmbeddingModel string        `env:"MEMORY_EMBEDDING_MODEL" envDefault:"bge-m3"`
	MemoryEmbeddingDim   int           `env:"MEMORY_EMBEDDING_DIM" envDefault:"1024"`
	MemoryCacheEnabled   bool          `env:"MEMORY_CACHE_ENABLED" envDefault:"true"`
	MemoryCacheTTL       time.Duration `env:"MEMORY_CACHE_TTL" envDefault:"5m"`

	// ============================================
	// Realtime API Settings (LiveKit)
	// ============================================
	RealtimeEnabled   bool   `env:"REALTIME_ENABLED" envDefault:"false"`
	LiveKitURL        string `env:"LIVEKIT_URL"`
	LiveKitAPIKey     string `env:"LIVEKIT_API_KEY"`
	LiveKitAPISecret  string `env:"LIVEKIT_API_SECRET"`

	// ============================================
	// OAuth Connectors (GitHub, Google)
	// ============================================
	GitHubClientID         string `env:"GITHUB_CLIENT_ID"`
	GitHubClientSecret     string `env:"GITHUB_CLIENT_SECRET"`
	GitHubConnectorEnabled bool   `env:"GITHUB_CONNECTOR_ENABLED" envDefault:"false"`

	GoogleClientID         string `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret     string `env:"GOOGLE_CLIENT_SECRET"`
	GoogleConnectorEnabled bool   `env:"GOOGLE_CONNECTOR_ENABLED" envDefault:"false"`

	ConnectorTokenEncryptionKey   string        `env:"CONNECTOR_TOKEN_ENCRYPTION_KEY"`
	ConnectorTokenEncryptionKeyID string        `env:"CONNECTOR_TOKEN_ENCRYPTION_KEY_ID" envDefault:"v1"`
	OAuthStateSecret              string        `env:"OAUTH_STATE_SECRET"`
	OAuthStateExpiration          time.Duration `env:"OAUTH_STATE_EXPIRATION" envDefault:"5m"`
	OAuthRedirectBaseURL          string        `env:"OAUTH_REDIRECT_BASE_URL" envDefault:"http://localhost:8080"`
	OAuthFrontendURL              string        `env:"OAUTH_FRONTEND_URL" envDefault:"http://localhost:3001"`

	// ============================================
	// Analytics
	// ============================================
	AnalyticsEnabled     bool          `env:"ANALYTICS_ENABLED" envDefault:"false"`
	PostHogEnabled       bool          `env:"POSTHOG_ENABLED" envDefault:"false"`
	PostHogAPIKey        string        `env:"POSTHOG_API_KEY"`
	PostHogHost          string        `env:"POSTHOG_HOST" envDefault:"https://eu.posthog.com"`
	PostHogBatchSize     int           `env:"POSTHOG_BATCH_SIZE" envDefault:"100"`
	PostHogFlushInterval time.Duration `env:"POSTHOG_FLUSH_INTERVAL" envDefault:"10s"`

	// ============================================
	// Redis (optional caching)
	// ============================================
	RedisEnabled  bool   `env:"REDIS_ENABLED" envDefault:"false"`
	RedisURL      string `env:"REDIS_URL" envDefault:"redis://localhost:6379"`
	RedisPassword string `env:"REDIS_PASSWORD"`
	RedisDB       int    `env:"REDIS_DB" envDefault:"0"`

	// Internal tracking
	EnvReloadedAt time.Time
}

// Load parses environment variables into Config and performs validation.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}

	// Normalize log settings
	cfg.LogLevel = strings.ToLower(cfg.LogLevel)
	cfg.LogFormat = strings.ToLower(cfg.LogFormat)

	// Validate authentication config
	if err := cfg.validateAuth(); err != nil {
		return nil, err
	}

	// Validate database config
	if _, err := url.Parse(cfg.DBPostgresqlWriteDSN); err != nil {
		return nil, fmt.Errorf("invalid DB_POSTGRESQL_WRITE_DSN: %w", err)
	}

	// Set defaults
	if cfg.KeycloakPublicURL == "" {
		cfg.KeycloakPublicURL = cfg.KeycloakBaseURL
	}

	if cfg.APIKeyPrefix == "" {
		cfg.APIKeyPrefix = "sk_live"
	}

	if cfg.AuthClockSkew < 0 {
		cfg.AuthClockSkew = -cfg.AuthClockSkew
	}

	cfg.EnvReloadedAt = time.Now()

	return cfg, nil
}

func (c *Config) validateAuth() error {
	// At least one auth method must be enabled
	if !c.KeycloakEnabled && !c.LocalAuthEnabled {
		return errors.New("at least one auth method must be enabled (KEYCLOAK_ENABLED or LOCAL_AUTH_ENABLED)")
	}

	// Validate Keycloak config if enabled
	if c.KeycloakEnabled {
		if c.KeycloakBaseURL == "" {
			return errors.New("KEYCLOAK_BASE_URL is required when KEYCLOAK_ENABLED=true")
		}
		if _, err := url.ParseRequestURI(c.KeycloakBaseURL); err != nil {
			return fmt.Errorf("invalid KEYCLOAK_BASE_URL: %w", err)
		}
		if c.JWKSURL == "" && c.OIDCDiscoveryURL == "" {
			return errors.New("either JWKS_URL or OIDC_DISCOVERY_URL is required when KEYCLOAK_ENABLED=true")
		}
		if c.Issuer == "" {
			return errors.New("ISSUER is required when KEYCLOAK_ENABLED=true")
		}
	}

	// Validate local auth config if enabled
	if c.LocalAuthEnabled {
		if c.LocalJWTSecret == "" {
			// Generate a warning but allow startup with a random secret (not recommended for production)
			c.LocalJWTSecret = generateRandomSecret()
			fmt.Fprintf(os.Stderr, "WARNING: LOCAL_JWT_SECRET not set, using random secret (tokens will be invalid after restart)\n")
		}
		if len(c.LocalJWTSecret) < 32 {
			return errors.New("LOCAL_JWT_SECRET must be at least 32 characters")
		}
		if c.BcryptCost < 10 || c.BcryptCost > 14 {
			return errors.New("BCRYPT_COST must be between 10 and 14")
		}
	}

	// Validate Kong config if enabled
	if c.KongEnabled {
		if c.KongAdminURL == "" {
			return errors.New("KONG_ADMIN_URL is required when KONG_ENABLED=true")
		}
		if _, err := url.ParseRequestURI(c.KongAdminURL); err != nil {
			return fmt.Errorf("invalid KONG_ADMIN_URL: %w", err)
		}
	}

	return nil
}

// GetDatabaseWriteDSN returns the write database connection string.
func (c *Config) GetDatabaseWriteDSN() string {
	return c.DBPostgresqlWriteDSN
}

// GetDatabaseReadDSN returns the read database connection string.
func (c *Config) GetDatabaseReadDSN() string {
	if c.DBPostgresqlRead1DSN != "" {
		return c.DBPostgresqlRead1DSN
	}
	return c.GetDatabaseWriteDSN()
}

// IsDev returns true if running in development mode.
func IsDev() bool {
	return strings.HasPrefix(Version, "dev")
}

func generateRandomSecret() string {
	// Simple random string generation for development
	// In production, LOCAL_JWT_SECRET should always be set explicitly
	return fmt.Sprintf("dev-secret-%d", time.Now().UnixNano())
}
