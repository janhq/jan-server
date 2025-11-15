package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v10"
)

// Config holds the environment driven configuration for the response service.
type Config struct {
	// Service Configuration
	ServiceName     string        `env:"SERVICE_NAME" envDefault:"response-api"`
	Environment     string        `env:"ENVIRONMENT" envDefault:"development"`
	HTTPPort        int           `env:"RESPONSE_API_PORT" envDefault:"8082"`
	LogLevel        string        `env:"RESPONSE_LOG_LEVEL" envDefault:"info"`
	EnableTracing   bool          `env:"ENABLE_TRACING" envDefault:"false"`
	OTLPEndpoint    string        `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:""`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`

	// Database - Read/Write Split (required, no defaults)
	DBPostgresqlWriteDSN string `env:"DB_POSTGRESQL_WRITE_DSN,notEmpty"`
	DBPostgresqlRead1DSN string `env:"DB_POSTGRESQL_READ1_DSN"` // Optional read replica

	// Database Connection Pool
	DBMaxIdleConns int           `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
	DBMaxOpenConns int           `env:"DB_MAX_OPEN_CONNS" envDefault:"15"`
	DBConnLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"30m"`

	// Authentication
	AuthEnabled  bool   `env:"AUTH_ENABLED" envDefault:"false"`
	AuthIssuer   string `env:"AUTH_ISSUER"`
	AuthAudience string `env:"AUTH_AUDIENCE"`
	AuthJWKSURL  string `env:"AUTH_JWKS_URL"`

	// External Services
	LLMAPIURL   string `env:"RESPONSE_LLM_API_URL" envDefault:"http://localhost:8080"`
	MCPToolsURL string `env:"RESPONSE_MCP_TOOLS_URL" envDefault:"http://localhost:8091"`

	// Tool Execution
	MaxToolDepth int           `env:"RESPONSE_MAX_TOOL_DEPTH" envDefault:"8"`
	ToolTimeout  time.Duration `env:"TOOL_EXECUTION_TIMEOUT" envDefault:"45s"`
}

// Load parses environment variables into Config.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env config: %w", err)
	}

	if cfg.AuthEnabled {
		if strings.TrimSpace(cfg.AuthIssuer) == "" {
			return nil, fmt.Errorf("AUTH_ISSUER is required when AUTH_ENABLED is true")
		}
		if strings.TrimSpace(cfg.AuthJWKSURL) == "" {
			return nil, fmt.Errorf("AUTH_JWKS_URL is required when AUTH_ENABLED is true")
		}
	}

	if cfg.MaxToolDepth <= 0 {
		cfg.MaxToolDepth = 8
	}

	if cfg.ToolTimeout <= 0 {
		cfg.ToolTimeout = 45 * time.Second
	}

	return cfg, nil
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

// Addr returns the HTTP listen address.
func (c *Config) Addr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}
