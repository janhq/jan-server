package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
)

// Config holds all configuration for the realtime-api service.
type Config struct {
	// Service settings
	ServiceName     string        `env:"SERVICE_NAME" envDefault:"realtime-api"`
	Environment     string        `env:"ENVIRONMENT" envDefault:"development"`
	HTTPPort        int           `env:"REALTIME_API_PORT" envDefault:"8186"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`

	// OpenTelemetry
	EnableTracing bool   `env:"OTEL_ENABLED" envDefault:"false"`
	OTLPEndpoint  string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:""`

	// Auth (Keycloak) - uses global auth vars
	AuthEnabled  bool   `env:"AUTH_ENABLED" envDefault:"false"`
	AuthIssuer   string `env:"ISSUER"`
	AuthAudience string `env:"AUDIENCE"`
	AuthJWKSURL  string `env:"JWKS_URL"`

	// LiveKit
	LiveKitWsURL     string        `env:"LIVEKIT_WS_URL" envDefault:"ws://localhost:7880"`
	LiveKitAPIKey    string        `env:"LIVEKIT_API_KEY"`
	LiveKitAPISecret string        `env:"LIVEKIT_API_SECRET"`
	LiveKitTokenTTL  time.Duration `env:"LIVEKIT_TOKEN_TTL" envDefault:"24h"`

	// Session Management
	SessionCleanupInterval time.Duration `env:"SESSION_CLEANUP_INTERVAL" envDefault:"15s"`
	SessionStaleTTL        time.Duration `env:"SESSION_STALE_TTL" envDefault:"10m"` // How long before a "created" session is considered stale

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
}

// Load parses environment variables into Config.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env config: %w", err)
	}

	// Validate auth configuration
	if cfg.AuthEnabled {
		if strings.TrimSpace(cfg.AuthIssuer) == "" {
			return nil, fmt.Errorf("ISSUER is required when AUTH_ENABLED is true")
		}
		if strings.TrimSpace(cfg.AuthAudience) == "" {
			return nil, fmt.Errorf("AUDIENCE is required when AUTH_ENABLED is true")
		}
		if strings.TrimSpace(cfg.AuthJWKSURL) == "" {
			return nil, fmt.Errorf("JWKS_URL is required when AUTH_ENABLED is true")
		}
	}

	// Validate LiveKit configuration
	if strings.TrimSpace(cfg.LiveKitAPIKey) == "" {
		return nil, fmt.Errorf("LIVEKIT_API_KEY is required")
	}
	if strings.TrimSpace(cfg.LiveKitAPISecret) == "" {
		return nil, fmt.Errorf("LIVEKIT_API_SECRET is required")
	}

	return cfg, nil
}

// Addr returns the HTTP server address.
func (c *Config) Addr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}
