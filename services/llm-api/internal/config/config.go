package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
)

// Global singleton for backwards compatibility with envs package
var globalConfig *Config

// Config holds all environment backed configuration for llm-api.
type Config struct {
	// HTTP Server
	HTTPPort    int    `env:"HTTP_PORT" envDefault:"8080"`
	MetricsPort int    `env:"METRICS_PORT" envDefault:"9091"`
	DatabaseURL string `env:"DATABASE_URL,notEmpty"`

	// Keycloak / Auth
	KeycloakBaseURL     string        `env:"KEYCLOAK_BASE_URL,notEmpty"`
	KeycloakRealm       string        `env:"KEYCLOAK_REALM" envDefault:"jan"`
	BackendClientID     string        `env:"BACKEND_CLIENT_ID,notEmpty"`
	BackendClientSecret string        `env:"BACKEND_CLIENT_SECRET,notEmpty"`
	TargetClientID      string        `env:"TARGET_CLIENT_ID,notEmpty"`
	GuestRole           string        `env:"GUEST_ROLE" envDefault:"guest"`
	KeycloakAdminUser   string        `env:"KEYCLOAK_ADMIN"`
	KeycloakAdminPass   string        `env:"KEYCLOAK_ADMIN_PASSWORD"`
	KeycloakAdminRealm  string        `env:"KEYCLOAK_ADMIN_REALM" envDefault:"master"`
	KeycloakAdminClient string        `env:"KEYCLOAK_ADMIN_CLIENT_ID" envDefault:"admin-cli"`
	KeycloakAdminSecret string        `env:"KEYCLOAK_ADMIN_CLIENT_SECRET"`
	JWKSURL             string        `env:"JWKS_URL"`
	OIDCDiscoveryURL    string        `env:"OIDC_DISCOVERY_URL"`
	Issuer              string        `env:"ISSUER,notEmpty"`
	Audience            string        `env:"AUDIENCE,notEmpty"`
	RefreshJWKSInterval time.Duration `env:"JWKS_REFRESH_INTERVAL" envDefault:"5m"`

	// API Keys
	APIKeySecret []byte `env:"APIKEY_SECRET"`

	// PostgreSQL
	DBPostgresqlWriteDSN string `env:"DB_POSTGRESQL_WRITE_DSN"`
	DBPostgresqlRead1DSN string `env:"DB_POSTGRESQL_READ1_DSN"`

	// Model Provider
	ModelProviderSecret       string                   `env:"MODEL_PROVIDER_SECRET" envDefault:"jan-model-provider-secret-2024"`
	JanDefaultNodeSetup       bool                     `env:"JAN_DEFAULT_NODE_SETUP" envDefault:"true"`
	JanDefaultNodeURL         string                   `env:"JAN_DEFAULT_NODE_URL" envDefault:"http://localhost:8001/v1"`
	JanDefaultNodeAPIKey      string                   `env:"JAN_DEFAULT_NODE_API_KEY" envDefault:"changeme"`
	JanProviderConfigsEnabled bool                     `env:"JAN_PROVIDER_CONFIGS" envDefault:"false"`
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
	MediaResolveURL     string        `env:"MEDIA_RESOLVE_URL" envDefault:"http://media-api:8285/v1/media/resolve"`
	MediaServiceKey     string        `env:"MEDIA_SERVICE_KEY"`
	MediaResolveTimeout time.Duration `env:"MEDIA_RESOLVE_TIMEOUT" envDefault:"5s"`

	// Internal
	EnvReloadedAt time.Time
}

// Load parses environment variables into Config and performs minimal validation.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}

	cfg.JanProviderConfigSet = strings.TrimSpace(cfg.JanProviderConfigSet)
	if cfg.JanProviderConfigSet == "" {
		cfg.JanProviderConfigSet = "default"
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

	if _, err := url.ParseRequestURI(cfg.KeycloakBaseURL); err != nil {
		return nil, fmt.Errorf("invalid KEYCLOAK_BASE_URL: %w", err)
	}

	cfg.LogLevel = strings.ToLower(cfg.LogLevel)
	cfg.LogFormat = strings.ToLower(cfg.LogFormat)
	cfg.EnvReloadedAt = time.Now()

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
