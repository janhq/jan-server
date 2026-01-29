package analytics

import "time"

// Config holds the unified analytics configuration
type Config struct {
	// Master switch
	Enabled bool `env:"ANALYTICS_ENABLED" envDefault:"true"`

	// PostHog settings
	PostHog PostHogConfig

	// OTel/Prometheus settings
	OTel OTelConfig

	// Privacy
	PIILevel string `env:"ANALYTICS_PII_LEVEL" envDefault:"hashed"`

	// Environment for segmentation (dev, staging, prod)
	Environment string `env:"ENVIRONMENT" envDefault:"dev"`
}

// PostHogConfig holds PostHog-specific configuration
type PostHogConfig struct {
	Enabled       bool          `env:"POSTHOG_ENABLED" envDefault:"false"`
	APIKey        string        `env:"POSTHOG_API_KEY"`
	Host          string        `env:"POSTHOG_HOST" envDefault:"https://eu.posthog.com"`
	Debug         bool          `env:"POSTHOG_DEBUG" envDefault:"false"`
	BatchSize     int           `env:"POSTHOG_BATCH_SIZE" envDefault:"100"`
	FlushInterval time.Duration `env:"POSTHOG_FLUSH_INTERVAL" envDefault:"10s"`
}

// OTelConfig holds OpenTelemetry-specific configuration
type OTelConfig struct {
	Enabled     bool   `env:"OTEL_ANALYTICS" envDefault:"false"`
	Endpoint    string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	Headers     string `env:"OTEL_EXPORTER_OTLP_HEADERS"`
	MetricsPort int    `env:"METRICS_PORT" envDefault:"8080"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		Enabled:     true,
		Environment: "dev",
		PIILevel:    "hashed",
		PostHog: PostHogConfig{
			Enabled:       false,
			Host:          "https://eu.posthog.com",
			BatchSize:     100,
			FlushInterval: 10 * time.Second,
		},
		OTel: OTelConfig{
			Enabled:     false,
			MetricsPort: 8080,
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.PostHog.Enabled && c.PostHog.APIKey == "" {
		return ErrMissingPostHogAPIKey
	}
	return nil
}
