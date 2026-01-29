package mcpprovider

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ProviderType defines the type of MCP provider
type ProviderType string

const (
	ProviderTypeHTTP    ProviderType = "http"     // Regular HTTP API
	ProviderTypeMCPHTTP ProviderType = "mcp-http" // MCP protocol over HTTP
)

// ProviderTool represents a tool exposed by a provider
type ProviderTool struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Enabled     bool   `yaml:"enabled"`
}

// Provider represents an external MCP service provider
type Provider struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Enabled     bool           `yaml:"enabled"`
	Endpoint    string         `yaml:"endpoint"`
	Type        ProviderType   `yaml:"type"`
	ProxyMode   bool           `yaml:"proxy_mode"`
	Timeout     string         `yaml:"timeout"`
	Tools       []ProviderTool `yaml:"tools,omitempty"`
}

// TimeoutDuration returns the timeout as a time.Duration
func (p *Provider) TimeoutDuration() time.Duration {
	if p.Timeout == "" {
		return 30 * time.Second
	}
	d, err := time.ParseDuration(p.Timeout)
	if err != nil {
		return 30 * time.Second
	}
	return d
}

// Settings represents global provider settings
type Settings struct {
	MaxTimeout    string `yaml:"max_timeout"`
	DebugLogging  bool   `yaml:"debug_logging"`
	RetryAttempts int    `yaml:"retry_attempts"`
	RetryDelay    string `yaml:"retry_delay"`
}

// Config represents the MCP provider configuration
type Config struct {
	Providers []Provider `yaml:"providers"`
	Settings  Settings   `yaml:"settings"`
}

// LoadConfig loads the MCP provider configuration from a YAML file
func LoadConfig(configPath string) (*Config, error) {
	// Expand environment variables in config path
	configPath = os.ExpandEnv(configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Expand environment variables in the YAML content
	expanded := os.ExpandEnv(string(data))

	var config Config
	if err := yaml.Unmarshal([]byte(expanded), &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetEnabledProviders returns only enabled providers
func (c *Config) GetEnabledProviders() []Provider {
	var enabled []Provider
	for _, p := range c.Providers {
		if p.Enabled {
			enabled = append(enabled, p)
		}
	}
	return enabled
}

// GetProvider returns a provider by name
func (c *Config) GetProvider(name string) *Provider {
	for _, p := range c.Providers {
		if p.Name == name {
			return &p
		}
	}
	return nil
}
