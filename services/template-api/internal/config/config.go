package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
)

// Config holds the environment driven configuration for the template service.
//
// NOTE: This service uses the traditional env-based approach for demonstration.
// For new services, consider using the central configuration system at pkg/config
// which provides:
//   - Type-safe configuration with validation
//   - YAML defaults with env var overrides
//   - Automatic documentation generation
//   - Kubernetes values generation
//   - Configuration provenance tracking
//
// See docs/configuration/ for migration guide and examples.
type Config struct {
	ServiceName     string        `env:"SERVICE_NAME" envDefault:"template-api"`
	Environment     string        `env:"ENVIRONMENT" envDefault:"development"`
	HTTPPort        int           `env:"HTTP_PORT" envDefault:"8185"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info"`
	EnableTracing   bool          `env:"ENABLE_TRACING" envDefault:"false"`
	OTLPEndpoint    string        `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:""`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	DatabaseURL     string        `env:"TEMPLATE_DATABASE_URL" envDefault:"postgres://postgres:postgres@localhost:5432/template_api?sslmode=disable"`
	DBMaxIdleConns  int           `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
	DBMaxOpenConns  int           `env:"DB_MAX_OPEN_CONNS" envDefault:"15"`
	DBConnLifetime  time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"30m"`
	AuthEnabled     bool          `env:"AUTH_ENABLED" envDefault:"false"`
	AuthIssuer      string        `env:"AUTH_ISSUER"`
	AuthAudience    string        `env:"AUTH_AUDIENCE"`
	AuthJWKSURL     string        `env:"AUTH_JWKS_URL"`
}

// Load parses environment variables into Config.
//
// Configuration Loading Order (highest to lowest priority):
// 1. Environment variables
// 2. .env file (if present)
// 3. Default values from struct tags
//
// For production deployments, environment variables should be set via:
//   - Docker Compose (docker-compose.yml env_file or environment)
//   - Kubernetes ConfigMaps/Secrets
//   - System environment variables
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env config: %w", err)
	}

	// Validate auth configuration
	if cfg.AuthEnabled {
		if strings.TrimSpace(cfg.AuthIssuer) == "" {
			return nil, fmt.Errorf("AUTH_ISSUER is required when AUTH_ENABLED is true")
		}
		if strings.TrimSpace(cfg.AuthAudience) == "" {
			return nil, fmt.Errorf("AUTH_AUDIENCE is required when AUTH_ENABLED is true")
		}
		if strings.TrimSpace(cfg.AuthJWKSURL) == "" {
			return nil, fmt.Errorf("AUTH_JWKS_URL is required when AUTH_ENABLED is true")
		}
	}

	return cfg, nil
}

// Addr returns the HTTP listen address.
func (c *Config) Addr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}
