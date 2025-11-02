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

// Config holds all environment backed configuration for llm-api.
type Config struct {
	HTTPPort            int           `env:"HTTP_PORT" envDefault:"8080"`
	MetricsPort         int           `env:"METRICS_PORT" envDefault:"9091"`
	DatabaseURL         string        `env:"DATABASE_URL,notEmpty"`
	ProvidersConfigPath string        `env:"PROVIDERS_CONFIG,notEmpty"`
	VLLMInternalKey     string        `env:"VLLM_INTERNAL_KEY,notEmpty"`
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
	HTTPTimeout         time.Duration `env:"HTTP_TIMEOUT" envDefault:"30s"`
	OTLPEndpoint        string        `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OTLPHeaders         string        `env:"OTEL_EXPORTER_OTLP_HEADERS"`
	ServiceName         string        `env:"SERVICE_NAME" envDefault:"llm-api"`
	ServiceNamespace    string        `env:"SERVICE_NAMESPACE" envDefault:"jan"`
	Environment         string        `env:"ENVIRONMENT" envDefault:"development"`
	LogLevel            string        `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat           string        `env:"LOG_FORMAT" envDefault:"console"`
	AutoMigrate         bool          `env:"AUTO_MIGRATE" envDefault:"true"`
	EnableSwagger       bool          `env:"ENABLE_SWAGGER" envDefault:"true"`
}

// Load parses environment variables into Config and performs minimal validation.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
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

	return cfg, nil
}

// ResolveJWKSURL returns the JWKS endpoint using either the explicit JWKS_URL or the OIDC discovery document.
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
