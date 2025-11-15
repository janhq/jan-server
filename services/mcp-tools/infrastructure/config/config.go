package config

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Config holds all configuration for the MCP Tools service
type Config struct {
	// HTTP Server - using MCP_TOOLS_ prefix to avoid collisions
	HTTPPort  string `env:"MCP_TOOLS_HTTP_PORT" envDefault:"8091"`
	LogLevel  string `env:"MCP_TOOLS_LOG_LEVEL" envDefault:"info"`
	LogFormat string `env:"MCP_TOOLS_LOG_FORMAT" envDefault:"json"` // json or console

	// Search Configuration
	SerperAPIKey       string   `env:"SERPER_API_KEY"`
	SearchEngine       string   `env:"MCP_SEARCH_ENGINE" envDefault:"serper"`
	SearxngURL         string   `env:"SEARXNG_URL" envDefault:"http://searxng:8080"`
	SerperDomainFilter []string `env:"SERPER_DOMAIN_FILTER" envSeparator:","`
	SerperLocationHint string   `env:"SERPER_LOCATION_HINT"`
	SerperOfflineMode  bool     `env:"SERPER_OFFLINE_MODE" envDefault:"false"`

	// External Services
	VectorStoreURL   string `env:"VECTOR_STORE_URL" envDefault:"http://vector-store-mcp:3015"`
	SandboxFusionURL string `env:"SANDBOXFUSION_URL" envDefault:"http://sandbox-fusion:8080"`

	// Sandbox Configuration
	SandboxFusionRequireApproval bool `env:"MCP_SANDBOX_REQUIRE_APPROVAL" envDefault:"false"`

	// Authentication
	AuthEnabled  bool   `env:"AUTH_ENABLED" envDefault:"false"`
	AuthIssuer   string `env:"AUTH_ISSUER"`
	AuthAudience string `env:"AUTH_AUDIENCE"`
	AuthJWKSURL  string `env:"AUTH_JWKS_URL"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
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
