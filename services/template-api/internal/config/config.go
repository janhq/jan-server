package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v10"
)

// Config holds the environment driven configuration for the template service.
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
}

// Load parses environment variables into Config.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env config: %w", err)
	}
	return cfg, nil
}

// Addr returns the HTTP listen address.
func (c *Config) Addr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}
