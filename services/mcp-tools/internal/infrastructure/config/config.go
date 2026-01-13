package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds all configuration for the MCP Tools service
type Config struct {
	// HTTP Server - using MCP_TOOLS_ prefix to avoid collisions
	HTTPPort  string `env:"MCP_TOOLS_HTTP_PORT" envDefault:"8091"`
	LogLevel  string `env:"MCP_TOOLS_LOG_LEVEL" envDefault:"debug"`
	LogFormat string `env:"MCP_TOOLS_LOG_FORMAT" envDefault:"json"` // json or console

	// Search Configuration
	SerperAPIKey       string   `env:"SERPER_API_KEY"`
	SerperEnabled      bool     `env:"SERPER_ENABLED" envDefault:"true"`
	SearchEngine       string   `env:"MCP_SEARCH_ENGINE" envDefault:"serper"`
	SearxngURL         string   `env:"SEARXNG_URL" envDefault:"http://searxng:8080"`
	SearxngEnabled     bool     `env:"SEARXNG_ENABLED" envDefault:"false"`
	SerperDomainFilter []string `env:"SERPER_DOMAIN_FILTER" envSeparator:","`
	SerperLocationHint string   `env:"SERPER_LOCATION_HINT"`
	SerperOfflineMode  bool     `env:"SERPER_OFFLINE_MODE" envDefault:"false"`

	ExaAPIKey         string        `env:"EXA_API_KEY"`
	ExaEnabled        bool          `env:"EXA_ENABLED" envDefault:"false"`
	ExaSearchEndpoint string        `env:"EXA_SEARCH_ENDPOINT" envDefault:"https://api.exa.ai/search"`
	ExaTimeout        time.Duration `env:"EXA_TIMEOUT" envDefault:"15s"`

	TavilyAPIKey         string        `env:"TAVILY_API_KEY"`
	TavilyEnabled        bool          `env:"TAVILY_ENABLED" envDefault:"false"`
	TavilySearchEndpoint string        `env:"TAVILY_SEARCH_ENDPOINT" envDefault:"https://api.tavily.com/search"`
	TavilyTimeout        time.Duration `env:"TAVILY_TIMEOUT" envDefault:"15s"`

	// Circuit Breaker Configuration
	SearchCBEnabled          bool `env:"MCP_SEARCH_CB_ENABLED" envDefault:"false"`
	SerperCBFailureThreshold int  `env:"SERPER_CB_FAILURE_THRESHOLD" envDefault:"15"`
	SerperCBSuccessThreshold int  `env:"SERPER_CB_SUCCESS_THRESHOLD" envDefault:"5"`
	SerperCBTimeout          int  `env:"SERPER_CB_TIMEOUT" envDefault:"45"`
	SerperCBMaxHalfOpen      int  `env:"SERPER_CB_MAX_HALF_OPEN" envDefault:"10"`

	// HTTP Client Performance
	SerperHTTPTimeout     int `env:"SERPER_HTTP_TIMEOUT" envDefault:"15"`
	SerperScrapeTimeout   int `env:"SERPER_SCRAPE_TIMEOUT" envDefault:"30"` // Separate longer timeout for scrape operations
	SerperMaxConnsPerHost int `env:"SERPER_MAX_CONNS_PER_HOST" envDefault:"50"`
	SerperMaxIdleConns    int `env:"SERPER_MAX_IDLE_CONNS" envDefault:"100"`
	SerperIdleConnTimeout int `env:"SERPER_IDLE_CONN_TIMEOUT" envDefault:"90"`

	// Retry Configuration
	SerperRetryMaxAttempts   int     `env:"SERPER_RETRY_MAX_ATTEMPTS" envDefault:"5"`
	SerperRetryInitialDelay  int     `env:"SERPER_RETRY_INITIAL_DELAY" envDefault:"250"`
	SerperRetryMaxDelay      int     `env:"SERPER_RETRY_MAX_DELAY" envDefault:"5000"`
	SerperRetryBackoffFactor float64 `env:"SERPER_RETRY_BACKOFF_FACTOR" envDefault:"1.5"`

	// Tool Result Token Limits - Controls maximum output size for MCP tool results
	MaxSnippetChars       int `env:"MCP_MAX_SNIPPET_CHARS" envDefault:"5000"`        // Max chars for search result snippets
	MaxScrapePreviewChars int `env:"MCP_MAX_SCRAPE_PREVIEW_CHARS" envDefault:"5000"` // Max chars for scrape text preview
	MaxScrapeTextChars    int `env:"MCP_MAX_SCRAPE_TEXT_CHARS" envDefault:"50000"`   // Max chars for full scrape text (approx 12.5k tokens)

	// External Services
	VectorStoreURL   string `env:"VECTOR_STORE_URL" envDefault:"http://vector-store-mcp:3015"`
	SandboxFusionURL string `env:"SANDBOXFUSION_URL" envDefault:"http://sandbox-fusion:8080"`
	MemoryToolsURL   string `env:"MEMORY_TOOLS_URL" envDefault:"http://memory-tools:8090"`

	// LLM-API configuration for tool call tracking
	LLMAPIBaseURL      string `env:"LLM_API_BASE_URL" envDefault:"http://llm-api:8080"`
	MCPTrackingEnabled bool   `env:"MCP_TRACKING_ENABLED" envDefault:"true"`

	// Sandbox Configuration
	SandboxFusionRequireApproval bool `env:"MCP_SANDBOX_REQUIRE_APPROVAL" envDefault:"false"`
	EnablePythonExec             bool `env:"MCP_ENABLE_PYTHON_EXEC" envDefault:"true"`
	EnableMemoryRetrieve         bool `env:"MCP_ENABLE_MEMORY_RETRIEVE" envDefault:"true"`
	EnableFileSearch             bool `env:"MCP_ENABLE_FILE_SEARCH" envDefault:"false"`
	EnableImageGenerate          bool `env:"MCP_ENABLE_IMAGE_GENERATE" envDefault:"true"`
	EnableImageEdit              bool `env:"MCP_ENABLE_IMAGE_EDIT" envDefault:"true"`

	// Authentication
	AuthEnabled bool   `env:"AUTH_ENABLED" envDefault:"false"`
	AuthIssuer  string `env:"AUTH_ISSUER"`
	Account     string `env:"ACCOUNT"`
	AuthJWKSURL string `env:"AUTH_JWKS_URL"`

	// AIO Sandbox Configuration
	AIOEnabled bool          `env:"AIO_ENABLED" envDefault:"false"`
	AIOURL     string        `env:"AIO_URL" envDefault:"http://aio-sandbox:8080"`
	AIOTimeout time.Duration `env:"AIO_TIMEOUT" envDefault:"120s"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	if strings.TrimSpace(os.Getenv("MCP_TOOLS_LOG_LEVEL")) == "" {
		if global := strings.TrimSpace(os.Getenv("LOG_LEVEL")); global != "" {
			cfg.LogLevel = global
		}
	}
	if strings.TrimSpace(os.Getenv("MCP_TOOLS_LOG_FORMAT")) == "" {
		if global := strings.TrimSpace(os.Getenv("LOG_FORMAT")); global != "" {
			cfg.LogFormat = global
		}
	}

	serperEnabledSet := envVarSet("SERPER_ENABLED")
	searxngEnabledSet := envVarSet("SEARXNG_ENABLED")
	exaEnabledSet := envVarSet("EXA_ENABLED")
	tavilyEnabledSet := envVarSet("TAVILY_ENABLED")
	if !serperEnabledSet && !searxngEnabledSet && !exaEnabledSet && !tavilyEnabledSet {
		switch strings.ToLower(strings.TrimSpace(cfg.SearchEngine)) {
		case "searxng":
			cfg.SearxngEnabled = true
			cfg.SerperEnabled = false
		case "exa":
			cfg.ExaEnabled = true
			cfg.SerperEnabled = false
		case "tavily":
			cfg.TavilyEnabled = true
			cfg.SerperEnabled = false
		default:
			cfg.SerperEnabled = true
		}
	}
	if cfg.AuthEnabled {
		if strings.TrimSpace(cfg.AuthIssuer) == "" {
			return nil, fmt.Errorf("AUTH_ISSUER is required when AUTH_ENABLED is true")
		}
		if strings.TrimSpace(cfg.Account) == "" {
			return nil, fmt.Errorf("ACCOUNT is required when AUTH_ENABLED is true")
		}
		if strings.TrimSpace(cfg.AuthJWKSURL) == "" {
			return nil, fmt.Errorf("AUTH_JWKS_URL is required when AUTH_ENABLED is true")
		}
	}
	if cfg.SerperEnabled && strings.TrimSpace(cfg.SerperAPIKey) == "" {
		return nil, fmt.Errorf("SERPER_API_KEY is required when SERPER_ENABLED is true")
	}
	if cfg.ExaEnabled && strings.TrimSpace(cfg.ExaAPIKey) == "" {
		return nil, fmt.Errorf("EXA_API_KEY is required when EXA_ENABLED is true")
	}
	if cfg.TavilyEnabled && strings.TrimSpace(cfg.TavilyAPIKey) == "" {
		return nil, fmt.Errorf("TAVILY_API_KEY is required when TAVILY_ENABLED is true")
	}
	if cfg.SearxngEnabled && strings.TrimSpace(cfg.SearxngURL) == "" {
		return nil, fmt.Errorf("SEARXNG_URL is required when SEARXNG_ENABLED is true")
	}
	return cfg, nil
}

func envVarSet(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}
