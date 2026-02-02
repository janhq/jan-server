package config

import (
	"fmt"
	"os"
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
	AuthEnabled bool   `env:"AUTH_ENABLED" envDefault:"false"`
	AuthIssuer  string `env:"AUTH_ISSUER"`
	Account     string `env:"ACCOUNT"`
	AuthJWKSURL string `env:"AUTH_JWKS_URL"`

	// External Services
	LLMAPIURL            string `env:"RESPONSE_LLM_API_URL" envDefault:"http://localhost:8080"`
	MCPToolsURL          string `env:"RESPONSE_MCP_TOOLS_URL" envDefault:"http://localhost:8091"`
	MediaAPIURL          string `env:"RESPONSE_MEDIA_API_URL" envDefault:"http://media-api:8285"`
	AIOURL               string `env:"AIO_URL" envDefault:""`
	SlideRendererScript  string `env:"SLIDE_RENDERER_SCRIPT" envDefault:""`
	SlideRendererEnabled bool   `env:"SLIDE_RENDERER_ENABLED" envDefault:"true"`

	// Tool Execution
	MaxToolDepth  int           `env:"RESPONSE_MAX_TOOL_DEPTH" envDefault:"50"`
	ToolTimeout   time.Duration `env:"TOOL_EXECUTION_TIMEOUT" envDefault:"300s"`
	LLMStreamMode string        `env:"RESPONSE_LLM_STREAM_MODE" envDefault:"auto"`

	// Code Execution Retry
	CodeFixModel                string `env:"CODE_FIX_MODEL" envDefault:"gpt-4o-mini"`
	LLMDisableCustomTemperature bool   `env:"RESPONSE_LLM_DISABLE_CUSTOM_TEMPERATURE" envDefault:"false"`

	// Skill Execution
	SkillExecutionEnabled    bool          `env:"SKILL_EXECUTION_ENABLED" envDefault:"true"`
	SkillExecutionTimeout    time.Duration `env:"SKILL_EXECUTION_TIMEOUT" envDefault:"120s"`
	SkillMaxFileSize         int64         `env:"SKILL_MAX_FILE_SIZE" envDefault:"52428800"` // 50MB
	SkillMaxCodeFixRetries   int           `env:"SKILL_MAX_CODE_FIX_RETRIES" envDefault:"3"`
	SkillMaxInstallRetries   int           `env:"SKILL_MAX_INSTALL_RETRIES" envDefault:"3"`
	SkillSlidesEnabled       bool          `env:"SKILL_SLIDES_ENABLED" envDefault:"true"`
	SkillDocsEnabled         bool          `env:"SKILL_DOCS_ENABLED" envDefault:"true"`
	SkillPDFsEnabled         bool          `env:"SKILL_PDFS_ENABLED" envDefault:"true"`
	SkillSpreadsheetsEnabled bool          `env:"SKILL_SPREADSHEETS_ENABLED" envDefault:"true"`

	// Background Task Processing
	BackgroundWorkerCount  int           `env:"BACKGROUND_WORKER_COUNT" envDefault:"4"`
	BackgroundTaskTimeout  time.Duration `env:"BACKGROUND_TASK_TIMEOUT" envDefault:"600s"`
	BackgroundPollInterval time.Duration `env:"BACKGROUND_POLL_INTERVAL" envDefault:"2s"`
	WebhookTimeout         time.Duration `env:"WEBHOOK_TIMEOUT" envDefault:"10s"`
	WebhookMaxRetries      int           `env:"WEBHOOK_MAX_RETRIES" envDefault:"3"`
	WebhookRetryDelay      time.Duration `env:"WEBHOOK_RETRY_DELAY" envDefault:"2s"`

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

	if strings.TrimSpace(os.Getenv("RESPONSE_LOG_LEVEL")) == "" {
		if global := strings.TrimSpace(os.Getenv("LOG_LEVEL")); global != "" {
			cfg.LogLevel = global
		}
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
		cfg.MaxToolDepth = 50
	}

	if cfg.ToolTimeout <= 0 {
		cfg.ToolTimeout = 300 * time.Second
	}

	cfg.LLMStreamMode = strings.ToLower(strings.TrimSpace(cfg.LLMStreamMode))
	switch cfg.LLMStreamMode {
	case "auto", "rest", "sse":
	default:
		cfg.LLMStreamMode = "auto"
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
