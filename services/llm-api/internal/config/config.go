package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
)

// Global singleton for backwards compatibility with envs package
var globalConfig *Config

// Config holds all environment backed configuration for llm-api.
type Config struct {
	// HTTP Server
	HTTPPort    int `env:"HTTP_PORT" envDefault:"8080"`
	MetricsPort int `env:"METRICS_PORT" envDefault:"9091"`

	// Database - Read/Write Split (required, no defaults)
	DBPostgresqlWriteDSN string `env:"DB_POSTGRESQL_WRITE_DSN,notEmpty"`
	DBPostgresqlRead1DSN string `env:"DB_POSTGRESQL_READ1_DSN"` // Optional read replica

	// Keycloak / Auth
	KeycloakBaseURL     string        `env:"KEYCLOAK_BASE_URL,notEmpty"`
	KeycloakPublicURL   string        `env:"KEYCLOAK_PUBLIC_URL"` // Browser-accessible URL (defaults to KeycloakBaseURL)
	KeycloakRealm       string        `env:"KEYCLOAK_REALM" envDefault:"jan"`
	BackendClientID     string        `env:"BACKEND_CLIENT_ID,notEmpty"`
	BackendClientSecret string        `env:"BACKEND_CLIENT_SECRET,notEmpty"`
	Client              string        `env:"CLIENT,notEmpty"`
	OAuthRedirectURI    string        `env:"OAUTH_REDIRECT_URI,notEmpty"`
	GuestRole           string        `env:"GUEST_ROLE" envDefault:"guest"`
	KeycloakAdminUser   string        `env:"KEYCLOAK_ADMIN"`
	KeycloakAdminPass   string        `env:"KEYCLOAK_ADMIN_PASSWORD"`
	KeycloakAdminRealm  string        `env:"KEYCLOAK_ADMIN_REALM" envDefault:"master"`
	KeycloakAdminClient string        `env:"KEYCLOAK_ADMIN_CLIENT_ID" envDefault:"admin-cli"`
	KeycloakAdminSecret string        `env:"KEYCLOAK_ADMIN_CLIENT_SECRET"`
	JWKSURL             string        `env:"JWKS_URL"`
	OIDCDiscoveryURL    string        `env:"OIDC_DISCOVERY_URL"`
	Issuer              string        `env:"ISSUER,notEmpty"`
	Account             string        `env:"ACCOUNT,notEmpty"`
	RefreshJWKSInterval time.Duration `env:"JWKS_REFRESH_INTERVAL" envDefault:"5m"`
	AuthClockSkew       time.Duration `env:"AUTH_CLOCK_SKEW" envDefault:"60s"`

	// API Keys
	APIKeySecret     []byte        `env:"APIKEY_SECRET"`
	APIKeyDefaultTTL time.Duration `env:"API_KEY_DEFAULT_TTL" envDefault:"2160h"` // 90 days
	APIKeyMaxTTL     time.Duration `env:"API_KEY_MAX_TTL" envDefault:"2160h"`
	APIKeyMaxPerUser int           `env:"API_KEY_MAX_PER_USER" envDefault:"5"`
	APIKeyPrefix     string        `env:"API_KEY_PREFIX" envDefault:"sk_live"`
	KongAdminURL     string        `env:"KONG_ADMIN_URL" envDefault:"http://kong:8001"`

	// Model Provider
	ModelProviderSecret       string                   `env:"MODEL_PROVIDER_SECRET" envDefault:"jan-model-provider-secret-2024"`
	JanProviderConfigsEnabled bool                     `env:"JAN_PROVIDER_CONFIGS" envDefault:"true"`
	JanProviderConfigSet      string                   `env:"JAN_PROVIDER_CONFIG_SET" envDefault:"default"`
	JanProviderConfigFile     string                   `env:"JAN_PROVIDER_CONFIGS_FILE"`
	ProviderBootstrap         *ProviderBootstrapConfig `env:"-"`

	// Model Sync
	ModelSyncIntervalMinutes int  `env:"MODEL_SYNC_INTERVAL_MINUTES" envDefault:"60"`
	ModelSyncEnabled         bool `env:"MODEL_SYNC_ENABLED" envDefault:"true"`

	// Observability / Logging
	HTTPTimeout      time.Duration `env:"HTTP_TIMEOUT" envDefault:"30s"`
	OTLPEndpoint     string        `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OTLPHeaders      string        `env:"OTEL_EXPORTER_OTLP_HEADERS"`
	ServiceName      string        `env:"SERVICE_NAME" envDefault:"llm-api"`
	ServiceNamespace string        `env:"SERVICE_NAMESPACE" envDefault:"jan"`
	Environment      string        `env:"ENVIRONMENT" envDefault:"development"`
	LogLevel         string        `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat        string        `env:"LOG_FORMAT" envDefault:"console"`

	// Features
	AutoMigrate   bool `env:"AUTO_MIGRATE" envDefault:"true"`
	EnableSwagger bool `env:"ENABLE_SWAGGER" envDefault:"true"`

	// Media integration
	MediaResolveURL     string        `env:"MEDIA_RESOLVE_URL" envDefault:"http://media-api:8285/v1/media"`
	MediaIngestURL      string        `env:"MEDIA_INGEST_URL" envDefault:"http://kong:8000/media/v1/media"`
	MediaResolveTimeout time.Duration `env:"MEDIA_RESOLVE_TIMEOUT" envDefault:"5s"`

	// Streaming timeout for LLM responses (increase for large/complex requests)
	StreamTimeout time.Duration `env:"STREAM_TIMEOUT" envDefault:"600s"`

	// Prompt Orchestration
	PromptOrchestrationEnabled         bool `env:"PROMPT_ORCHESTRATION_ENABLED" envDefault:"false"`
	PromptOrchestrationEnableMemory    bool `env:"PROMPT_ORCHESTRATION_MEMORY" envDefault:"false"`
	PromptOrchestrationEnableTemplates bool `env:"PROMPT_ORCHESTRATION_TEMPLATES" envDefault:"false"`
	PromptOrchestrationEnableTools     bool `env:"PROMPT_ORCHESTRATION_TOOLS" envDefault:"false"`

	// Memory integration
	MemoryEnabled bool          `env:"MEMORY_ENABLED" envDefault:"false"`
	MemoryBaseURL string        `env:"MEMORY_BASE_URL" envDefault:"http://memory-tools:8090"`
	MemoryTimeout time.Duration `env:"MEMORY_TIMEOUT" envDefault:"5s"`

	// Conversation Sharing
	ConversationSharingEnabled bool `env:"CONVERSATION_SHARING_ENABLED" envDefault:"false"`

	// Conversation Title Generation
	ConversationTitleGenerationEnabled bool   `env:"CONVERSATION_TITLE_GENERATION_ENABLED" envDefault:"false"`
	ConversationTitleGenerationModelID string `env:"CONVERSATION_TITLE_GENERATION_MODEL_ID" envDefault:"LFM2-8B-A1B"`

	// Image Generation
	ImageGenerationEnabled     bool          `env:"IMAGE_GENERATION_ENABLED" envDefault:"false"`
	ImageGenerationTimeout     time.Duration `env:"IMAGE_GENERATION_TIMEOUT" envDefault:"120s"`
	ImageDefaultModel          string        `env:"IMAGE_DEFAULT_MODEL" envDefault:"z-image"`
	ImageDefaultSize           string        `env:"IMAGE_DEFAULT_SIZE" envDefault:"1024x1024"`
	ImageMaxN                  int           `env:"IMAGE_MAX_N" envDefault:"4"`
	ImageDefaultQuality        string        `env:"IMAGE_DEFAULT_QUALITY" envDefault:"standard"`
	ImageDefaultStyle          string        `env:"IMAGE_DEFAULT_STYLE" envDefault:"natural"`
	ImageDefaultResponseFormat string        `env:"IMAGE_DEFAULT_RESPONSE_FORMAT" envDefault:"url"`
	ImageMediaPresignTTL       time.Duration `env:"IMAGE_MEDIA_PRESIGN_TTL" envDefault:"1h"`

	// Document OCR
	DocumentOCREnabled     bool          `env:"DOCUMENT_OCR_ENABLED" envDefault:"false"`
	DocumentOCRTimeout     time.Duration `env:"DOCUMENT_OCR_TIMEOUT" envDefault:"300s"`
	DocumentOCRModel       string        `env:"DOCUMENT_OCR_MODEL" envDefault:"docling-v1"`
	DocumentMaxBytes       int64         `env:"DOCUMENT_MAX_BYTES" envDefault:"52428800"` // 50MB
	DocumentSupportedTypes string        `env:"DOCUMENT_SUPPORTED_TYPES" envDefault:"application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/vnd.openxmlformats-officedocument.presentationml.presentation,text/plain,text/markdown,text/html,application/rtf"`

	// Connectors (OAuth integrations)
	GitHubClientID               string        `env:"GITHUB_CLIENT_ID"`
	GitHubClientSecret           string        `env:"GITHUB_CLIENT_SECRET"`
	GitHubConnectorEnabled       bool          `env:"GITHUB_CONNECTOR_ENABLED" envDefault:"false"`
	GoogleClientID               string        `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret           string        `env:"GOOGLE_CLIENT_SECRET"`
	GoogleConnectorEnabled       bool          `env:"GOOGLE_CONNECTOR_ENABLED" envDefault:"false"`
	ConnectorTokenEncryptionKey  string        `env:"CONNECTOR_TOKEN_ENCRYPTION_KEY"`
	ConnectorTokenEncryptionKeyID string       `env:"CONNECTOR_TOKEN_ENCRYPTION_KEY_ID" envDefault:"v1"`
	ConnectorTokenEncryptionKeyPrevious string `env:"CONNECTOR_TOKEN_ENCRYPTION_KEY_PREVIOUS"`
	ConnectorTokenEncryptionKeyIDPrevious string `env:"CONNECTOR_TOKEN_ENCRYPTION_KEY_ID_PREVIOUS"`
	OAuthStateSecret             string        `env:"OAUTH_STATE_SECRET"`
	OAuthStateExpiration         time.Duration `env:"OAUTH_STATE_EXPIRATION" envDefault:"5m"`
	OAuthRedirectBaseURL         string        `env:"OAUTH_REDIRECT_BASE_URL" envDefault:"http://localhost:8000"`
	OAuthFrontendURL             string        `env:"OAUTH_FRONTEND_URL" envDefault:"http://localhost:3001"`

	// Analytics (PostHog + OTel)
	AnalyticsEnabled     bool          `env:"ANALYTICS_ENABLED" envDefault:"true"`
	PostHogEnabled       bool          `env:"POSTHOG_ENABLED" envDefault:"false"`
	PostHogAPIKey        string        `env:"POSTHOG_API_KEY"`
	PostHogHost          string        `env:"POSTHOG_HOST" envDefault:"https://eu.posthog.com"`
	PostHogDebug         bool          `env:"POSTHOG_DEBUG" envDefault:"false"`
	PostHogBatchSize     int           `env:"POSTHOG_BATCH_SIZE" envDefault:"100"`
	PostHogFlushInterval time.Duration `env:"POSTHOG_FLUSH_INTERVAL" envDefault:"10s"`
	OTelAnalyticsEnabled bool          `env:"OTEL_ANALYTICS" envDefault:"false"`
	AnalyticsPIILevel    string        `env:"ANALYTICS_PII_LEVEL" envDefault:"hashed"`
	AnalyticsEnvironment string        `env:"ANALYTICS_ENVIRONMENT" envDefault:"dev"`

	// Internal
	EnvReloadedAt time.Time
}

// Load parses environment variables into Config and performs minimal validation.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}

	if llmLogLevel := strings.TrimSpace(os.Getenv("LLM_API_LOG_LEVEL")); llmLogLevel != "" {
		cfg.LogLevel = llmLogLevel
	}
	if llmLogFormat := strings.TrimSpace(os.Getenv("LLM_API_LOG_FORMAT")); llmLogFormat != "" {
		cfg.LogFormat = llmLogFormat
	}

	cfg.JanProviderConfigSet = strings.TrimSpace(cfg.JanProviderConfigSet)
	if cfg.JanProviderConfigSet == "" {
		cfg.JanProviderConfigSet = "default"
	}

	// Default KeycloakPublicURL to KeycloakBaseURL if not set
	if cfg.KeycloakPublicURL == "" {
		cfg.KeycloakPublicURL = cfg.KeycloakBaseURL
	}

	if cfg.JanProviderConfigsEnabled {
		configFile := strings.TrimSpace(cfg.JanProviderConfigFile)
		if configFile == "" {
			configFile = DefaultProviderConfigFile
		}
		bootstrap, err := LoadProviderBootstrapConfig(configFile)
		if err != nil {
			return nil, fmt.Errorf("load provider configs: %w", err)
		}
		cfg.ProviderBootstrap = bootstrap
		if len(bootstrap.ProvidersForSet(cfg.JanProviderConfigSet)) == 0 {
			return nil, fmt.Errorf("provider config set %q is missing or empty in %s", cfg.JanProviderConfigSet, configFile)
		}
	}

	if cfg.JWKSURL == "" && cfg.OIDCDiscoveryURL == "" {
		return nil, errors.New("either JWKS_URL or OIDC_DISCOVERY_URL must be provided")
	}

	if cfg.JWKSURL != "" {
		if _, err := url.ParseRequestURI(cfg.JWKSURL); err != nil {
			return nil, fmt.Errorf("invalid JWKS_URL: %w", err)
		}
	}

	if cfg.OIDCDiscoveryURL != "" {
		if _, err := url.ParseRequestURI(cfg.OIDCDiscoveryURL); err != nil {
			return nil, fmt.Errorf("invalid OIDC_DISCOVERY_URL: %w", err)
		}
	}

	if strings.TrimSpace(cfg.KongAdminURL) == "" {
		return nil, errors.New("KONG_ADMIN_URL is required")
	}
	if _, err := url.ParseRequestURI(cfg.KongAdminURL); err != nil {
		return nil, fmt.Errorf("invalid KONG_ADMIN_URL: %w", err)
	}

	if cfg.APIKeyDefaultTTL <= 0 {
		return nil, errors.New("API_KEY_DEFAULT_TTL must be > 0")
	}
	if cfg.APIKeyMaxTTL < cfg.APIKeyDefaultTTL {
		return nil, errors.New("API_KEY_MAX_TTL must be >= API_KEY_DEFAULT_TTL")
	}
	if cfg.APIKeyMaxPerUser <= 0 {
		cfg.APIKeyMaxPerUser = 5
	}
	cfg.APIKeyPrefix = strings.TrimSpace(cfg.APIKeyPrefix)
	if cfg.APIKeyPrefix == "" {
		cfg.APIKeyPrefix = "sk_live"
	}

	if cfg.AuthClockSkew < 0 {
		cfg.AuthClockSkew = cfg.AuthClockSkew * -1
	}

	if _, err := url.ParseRequestURI(cfg.KeycloakBaseURL); err != nil {
		return nil, fmt.Errorf("invalid KEYCLOAK_BASE_URL: %w", err)
	}

	cfg.LogLevel = strings.ToLower(cfg.LogLevel)
	cfg.LogFormat = strings.ToLower(cfg.LogFormat)
	cfg.EnvReloadedAt = time.Now()

	cfg.ConversationTitleGenerationModelID = strings.TrimSpace(cfg.ConversationTitleGenerationModelID)
	if cfg.ConversationTitleGenerationModelID == "" {
		cfg.ConversationTitleGenerationModelID = "LFM2-8B-A1B"
	}

	// Update global singletons for backwards compatibility
	globalConfig = cfg

	return cfg, nil
} // ResolveJWKSURL returns the JWKS endpoint using either the explicit JWKS_URL or the OIDC discovery document.
func (c *Config) ResolveJWKSURL(ctx context.Context) (string, error) {
	if c.JWKSURL != "" {
		return c.JWKSURL, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.OIDCDiscoveryURL, nil)
	if err != nil {
		return "", fmt.Errorf("oidc discovery request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch oidc discovery: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oidc discovery unexpected status: %s", resp.Status)
	}

	var doc struct {
		JWKSURL string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", fmt.Errorf("decode oidc discovery: %w", err)
	}

	if doc.JWKSURL == "" {
		return "", errors.New("jwks_uri not found in discovery document")
	}

	return doc.JWKSURL, nil
}

// GetDatabaseWriteDSN returns the write database connection string.
func (c *Config) GetDatabaseWriteDSN() string {
	return c.DBPostgresqlWriteDSN
}

// GetDatabaseReadDSN returns the read database connection string.
// If DB_POSTGRESQL_READ1_DSN is set, it returns that.
// Otherwise, falls back to write DSN (no replica configured).
func (c *Config) GetDatabaseReadDSN() string {
	if c.DBPostgresqlRead1DSN != "" {
		return c.DBPostgresqlRead1DSN
	}
	return c.GetDatabaseWriteDSN()
}

// GetGlobal returns the global config instance for backwards compatibility.
// Deprecated: Use dependency injection with Load() instead.
func GetGlobal() *Config {
	return globalConfig
}

// GetEnvReloadedAt returns when the environment was last reloaded
// Deprecated: Use GetGlobal().EnvReloadedAt instead
func GetEnvReloadedAt() time.Time {
	if globalConfig != nil {
		return globalConfig.EnvReloadedAt
	}
	return time.Time{}
}

// ProviderBootstrapEntries returns the configured provider definitions for the active set.
func (c *Config) ProviderBootstrapEntries() []ProviderBootstrapEntry {
	if c == nil || c.ProviderBootstrap == nil {
		return nil
	}
	return c.ProviderBootstrap.ProvidersForSet(c.JanProviderConfigSet)
}

var Version = "dev"

func IsDev() bool {
	return strings.HasPrefix(Version, "dev")
}
