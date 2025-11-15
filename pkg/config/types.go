package config

import "time"

// Config is the root configuration structure for Jan Server.
// This is the canonical source of truth - all schemas, defaults, and documentation
// are generated from these struct definitions.
type Config struct {
	Meta           MetaConfig           `yaml:"meta" json:"meta" jsonschema:"required"`
	Infrastructure InfrastructureConfig `yaml:"infrastructure" json:"infrastructure" jsonschema:"required"`
	Services       ServicesConfig       `yaml:"services" json:"services" jsonschema:"required"`
	Inference      InferenceConfig      `yaml:"inference" json:"inference"`
	Monitoring     MonitoringConfig     `yaml:"monitoring" json:"monitoring"`
}

// MetaConfig contains metadata about the configuration itself
type MetaConfig struct {
	// Version of the configuration schema
	Version string `yaml:"version" json:"version" env:"CONFIG_VERSION" envDefault:"1.0.0" jsonschema:"required" description:"Configuration schema version"`

	// Environment name (development, staging, production, etc.)
	Environment string `yaml:"environment" json:"environment" env:"ENVIRONMENT" envDefault:"development" jsonschema:"required" description:"Deployment environment name"`
}

// InfrastructureConfig contains settings for core infrastructure services
type InfrastructureConfig struct {
	Database DatabaseConfig `yaml:"database" json:"database" jsonschema:"required"`
	Auth     AuthConfig     `yaml:"auth" json:"auth" jsonschema:"required"`
	Gateway  GatewayConfig  `yaml:"gateway" json:"gateway" jsonschema:"required"`
}

// DatabaseConfig contains PostgreSQL database settings
type DatabaseConfig struct {
	Postgres PostgresConfig `yaml:"postgres" json:"postgres" jsonschema:"required"`
}

// PostgresConfig contains PostgreSQL-specific settings
type PostgresConfig struct {
	// Database host (Docker internal DNS or FQDN)
	Host string `yaml:"host" json:"host" env:"POSTGRES_HOST" envDefault:"api-db" jsonschema:"required" description:"PostgreSQL host (Docker service name or FQDN)"`

	// Database port
	Port int `yaml:"port" json:"port" env:"POSTGRES_PORT" envDefault:"5432" jsonschema:"required,minimum=1,maximum=65535" description:"PostgreSQL port"`

	// Database user
	User string `yaml:"user" json:"user" env:"POSTGRES_USER" envDefault:"jan_user" jsonschema:"required" description:"PostgreSQL username"`

	// Database name
	Database string `yaml:"database" json:"database" env:"POSTGRES_DB" envDefault:"jan_llm_api" jsonschema:"required" description:"PostgreSQL database name"`

	// Database password (loaded from secrets)
	Password string `yaml:"password,omitempty" json:"password,omitempty" env:"POSTGRES_PASSWORD" jsonschema:"required" description:"PostgreSQL password (from secret provider)"`

	// SSL mode (disable, require, verify-ca, verify-full)
	SSLMode string `yaml:"ssl_mode" json:"ssl_mode" env:"POSTGRES_SSL_MODE" envDefault:"disable" jsonschema:"enum=disable,enum=require,enum=verify-ca,enum=verify-full" description:"PostgreSQL SSL mode"`

	// Maximum number of open connections
	MaxConnections int `yaml:"max_connections" json:"max_connections" env:"POSTGRES_MAX_CONNECTIONS" envDefault:"100" jsonschema:"minimum=1,maximum=1000" description:"Maximum number of database connections"`

	// Maximum idle connections
	MaxIdleConns int `yaml:"max_idle_conns" json:"max_idle_conns" env:"DB_MAX_IDLE_CONNS" envDefault:"5" jsonschema:"minimum=1" description:"Maximum idle connections in pool"`

	// Maximum open connections
	MaxOpenConns int `yaml:"max_open_conns" json:"max_open_conns" env:"DB_MAX_OPEN_CONNS" envDefault:"15" jsonschema:"minimum=1" description:"Maximum open connections in pool"`

	// Connection max lifetime
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime" env:"DB_CONN_MAX_LIFETIME" envDefault:"30m" description:"Maximum connection lifetime"`
}

// AuthConfig contains authentication and authorization settings
type AuthConfig struct {
	Keycloak KeycloakConfig `yaml:"keycloak" json:"keycloak" jsonschema:"required"`
}

// KeycloakConfig contains Keycloak authentication server settings
type KeycloakConfig struct {
	// Keycloak base URL (internal service URL)
	BaseURL string `yaml:"base_url" json:"base_url" env:"KEYCLOAK_BASE_URL" envDefault:"http://keycloak:8085" jsonschema:"required,format=uri" description:"Keycloak base URL (internal)"`

	// Keycloak public URL (browser-accessible, defaults to BaseURL)
	PublicURL string `yaml:"public_url" json:"public_url" env:"KEYCLOAK_PUBLIC_URL" jsonschema:"format=uri" description:"Keycloak public URL (browser-accessible)"`

	// Keycloak realm name
	Realm string `yaml:"realm" json:"realm" env:"KEYCLOAK_REALM" envDefault:"jan" jsonschema:"required" description:"Keycloak realm name"`

	// Keycloak HTTP port
	HTTPPort int `yaml:"http_port" json:"http_port" env:"KEYCLOAK_HTTP_PORT" envDefault:"8085" jsonschema:"minimum=1,maximum=65535" description:"Keycloak HTTP port"`

	// Keycloak admin username
	AdminUser string `yaml:"admin_user" json:"admin_user" env:"KEYCLOAK_ADMIN" envDefault:"admin" jsonschema:"required" description:"Keycloak admin username"`

	// Keycloak admin password (from secrets)
	AdminPassword string `yaml:"admin_password,omitempty" json:"admin_password,omitempty" env:"KEYCLOAK_ADMIN_PASSWORD" jsonschema:"required" description:"Keycloak admin password (from secret provider)"`

	// Keycloak admin realm
	AdminRealm string `yaml:"admin_realm" json:"admin_realm" env:"KEYCLOAK_ADMIN_REALM" envDefault:"master" description:"Keycloak admin realm"`

	// Keycloak admin client ID
	AdminClientID string `yaml:"admin_client_id" json:"admin_client_id" env:"KEYCLOAK_ADMIN_CLIENT_ID" envDefault:"admin-cli" description:"Keycloak admin client ID"`

	// Backend client ID for service-to-service auth
	BackendClientID string `yaml:"backend_client_id" json:"backend_client_id" env:"BACKEND_CLIENT_ID" envDefault:"backend" jsonschema:"required" description:"Backend service client ID"`

	// Backend client secret (from secrets)
	BackendClientSecret string `yaml:"backend_client_secret,omitempty" json:"backend_client_secret,omitempty" env:"BACKEND_CLIENT_SECRET" jsonschema:"required" description:"Backend client secret (from secret provider)"`

	// Target client ID
	TargetClientID string `yaml:"target_client_id" json:"target_client_id" env:"TARGET_CLIENT_ID" envDefault:"jan-client" jsonschema:"required" description:"Target client ID for token exchange"`

	// OAuth redirect URI
	OAuthRedirectURI string `yaml:"oauth_redirect_uri" json:"oauth_redirect_uri" env:"OAUTH_REDIRECT_URI" envDefault:"http://localhost:8000/auth/callback" jsonschema:"required,format=uri" description:"OAuth redirect URI"`

	// JWKS URL for JWT verification
	JWKSURL string `yaml:"jwks_url" json:"jwks_url" env:"JWKS_URL" jsonschema:"format=uri" description:"JWKS URL for JWT verification"`

	// OIDC discovery URL (alternative to JWKS URL)
	OIDCDiscoveryURL string `yaml:"oidc_discovery_url" json:"oidc_discovery_url" env:"OIDC_DISCOVERY_URL" jsonschema:"format=uri" description:"OIDC discovery URL"`

	// JWT issuer
	Issuer string `yaml:"issuer" json:"issuer" env:"ISSUER" envDefault:"http://localhost:8085/realms/jan" jsonschema:"required,format=uri" description:"JWT issuer URL"`

	// JWT audience
	Audience string `yaml:"audience" json:"audience" env:"AUDIENCE" envDefault:"jan-client" jsonschema:"required" description:"JWT audience claim"`

	// JWKS refresh interval
	RefreshJWKSInterval time.Duration `yaml:"refresh_jwks_interval" json:"refresh_jwks_interval" env:"JWKS_REFRESH_INTERVAL" envDefault:"5m" description:"JWKS refresh interval"`

	// Auth clock skew tolerance
	AuthClockSkew time.Duration `yaml:"auth_clock_skew" json:"auth_clock_skew" env:"AUTH_CLOCK_SKEW" envDefault:"60s" description:"Clock skew tolerance for auth"`

	// Guest role name
	GuestRole string `yaml:"guest_role" json:"guest_role" env:"GUEST_ROLE" envDefault:"guest" description:"Guest role name"`

	// Keycloak features to enable
	Features []string `yaml:"features" json:"features" env:"KEYCLOAK_FEATURES" envSeparator:"," envDefault:"token-exchange,preview" description:"Keycloak features to enable"`
}

// GatewayConfig contains API gateway settings
type GatewayConfig struct {
	Kong KongConfig `yaml:"kong" json:"kong" jsonschema:"required"`
}

// KongConfig contains Kong API Gateway settings
type KongConfig struct {
	// Kong HTTP port
	HTTPPort int `yaml:"http_port" json:"http_port" env:"KONG_HTTP_PORT" envDefault:"8000" jsonschema:"minimum=1,maximum=65535" description:"Kong HTTP port"`

	// Kong admin port
	AdminPort int `yaml:"admin_port" json:"admin_port" env:"KONG_ADMIN_PORT" envDefault:"8001" jsonschema:"minimum=1,maximum=65535" description:"Kong admin API port"`

	// Kong admin URL (internal)
	AdminURL string `yaml:"admin_url" json:"admin_url" env:"KONG_ADMIN_URL" envDefault:"http://kong:8001" jsonschema:"format=uri" description:"Kong admin API URL"`

	// Kong log level
	LogLevel string `yaml:"log_level" json:"log_level" env:"KONG_LOG_LEVEL" envDefault:"info" jsonschema:"enum=debug,enum=info,enum=warn,enum=error" description:"Kong log level"`
}

// ServicesConfig contains settings for all Jan Server services
type ServicesConfig struct {
	LLMAPI      LLMAPIConfig      `yaml:"llm_api" json:"llm_api" jsonschema:"required"`
	MCPTools    MCPToolsConfig    `yaml:"mcp_tools" json:"mcp_tools" jsonschema:"required"`
	MediaAPI    MediaAPIConfig    `yaml:"media_api" json:"media_api" jsonschema:"required"`
	ResponseAPI ResponseAPIConfig `yaml:"response_api" json:"response_api" jsonschema:"required"`
}

// LLMAPIConfig contains settings for the LLM API service
type LLMAPIConfig struct {
	// HTTP port
	HTTPPort int `yaml:"http_port" json:"http_port" env:"HTTP_PORT" envDefault:"8080" jsonschema:"minimum=1,maximum=65535" description:"LLM API HTTP port"`

	// Metrics port
	MetricsPort int `yaml:"metrics_port" json:"metrics_port" env:"METRICS_PORT" envDefault:"9091" jsonschema:"minimum=1,maximum=65535" description:"Metrics port"`

	// Log level
	LogLevel string `yaml:"log_level" json:"log_level" env:"LOG_LEVEL" envDefault:"info" jsonschema:"enum=debug,enum=info,enum=warn,enum=error" description:"Log level"`

	// Log format
	LogFormat string `yaml:"log_format" json:"log_format" env:"LOG_FORMAT" envDefault:"json" jsonschema:"enum=json,enum=console" description:"Log format"`

	// Auto-migrate database on startup
	AutoMigrate bool `yaml:"auto_migrate" json:"auto_migrate" env:"AUTO_MIGRATE" envDefault:"true" description:"Auto-migrate database on startup"`

	// Provider config file path (relative to service root)
	ProviderConfigFile string `yaml:"provider_config_file" json:"provider_config_file" env:"JAN_PROVIDER_CONFIGS_FILE" envDefault:"config/providers.yml" description:"Provider config file path (CI/CD managed)"`

	// Provider config set to use
	ProviderConfigSet string `yaml:"provider_config_set" json:"provider_config_set" env:"JAN_PROVIDER_CONFIG_SET" envDefault:"default" description:"Provider config set name"`

	// Enable provider configs
	ProviderConfigsEnabled bool `yaml:"provider_configs_enabled" json:"provider_configs_enabled" env:"JAN_PROVIDER_CONFIGS" envDefault:"true" description:"Enable provider config file"`

	// API key settings
	APIKey APIKeyConfig `yaml:"api_key" json:"api_key"`

	// Model provider secret
	ModelProviderSecret string `yaml:"model_provider_secret,omitempty" json:"model_provider_secret,omitempty" env:"MODEL_PROVIDER_SECRET" envDefault:"jan-model-provider-secret-2024" description:"Model provider secret"`

	// Model sync settings
	ModelSyncEnabled         bool `yaml:"model_sync_enabled" json:"model_sync_enabled" env:"MODEL_SYNC_ENABLED" envDefault:"true" description:"Enable model synchronization"`
	ModelSyncIntervalMinutes int  `yaml:"model_sync_interval_minutes" json:"model_sync_interval_minutes" env:"MODEL_SYNC_INTERVAL_MINUTES" envDefault:"60" jsonschema:"minimum=1" description:"Model sync interval in minutes"`

	// Media integration
	MediaResolveURL     string        `yaml:"media_resolve_url" json:"media_resolve_url" env:"MEDIA_RESOLVE_URL" envDefault:"http://kong:8000/media/v1/media/resolve" jsonschema:"format=uri" description:"Media resolve URL"`
	MediaResolveTimeout time.Duration `yaml:"media_resolve_timeout" json:"media_resolve_timeout" env:"MEDIA_RESOLVE_TIMEOUT" envDefault:"5s" description:"Media resolve timeout"`
}

// APIKeyConfig contains API key management settings
type APIKeyConfig struct {
	// API key prefix
	Prefix string `yaml:"prefix" json:"prefix" env:"API_KEY_PREFIX" envDefault:"sk_live" description:"API key prefix"`

	// Default TTL for new API keys
	DefaultTTL time.Duration `yaml:"default_ttl" json:"default_ttl" env:"API_KEY_DEFAULT_TTL" envDefault:"2160h" description:"Default API key TTL (90 days)"`

	// Maximum TTL for API keys
	MaxTTL time.Duration `yaml:"max_ttl" json:"max_ttl" env:"API_KEY_MAX_TTL" envDefault:"2160h" description:"Maximum API key TTL"`

	// Maximum API keys per user
	MaxPerUser int `yaml:"max_per_user" json:"max_per_user" env:"API_KEY_MAX_PER_USER" envDefault:"5" jsonschema:"minimum=1" description:"Maximum API keys per user"`
}

// MCPToolsConfig contains settings for the MCP Tools service
type MCPToolsConfig struct {
	// HTTP port
	HTTPPort int `yaml:"http_port" json:"http_port" env:"MCP_TOOLS_HTTP_PORT" envDefault:"8091" jsonschema:"minimum=1,maximum=65535" description:"MCP Tools HTTP port"`

	// Log level
	LogLevel string `yaml:"log_level" json:"log_level" env:"LOG_LEVEL" envDefault:"info" jsonschema:"enum=debug,enum=info,enum=warn,enum=error" description:"Log level"`

	// Log format
	LogFormat string `yaml:"log_format" json:"log_format" env:"LOG_FORMAT" envDefault:"json" jsonschema:"enum=json,enum=console" description:"Log format"`

	// Search engine to use
	SearchEngine string `yaml:"search_engine" json:"search_engine" env:"SEARCH_ENGINE" envDefault:"serper" jsonschema:"enum=serper,enum=searxng" description:"Search engine to use"`

	// Serper API key (from secrets)
	SerperAPIKey string `yaml:"serper_api_key,omitempty" json:"serper_api_key,omitempty" env:"SERPER_API_KEY" description:"Serper API key (from secret provider)"`

	// SearXNG URL
	SearxngURL string `yaml:"searxng_url" json:"searxng_url" env:"SEARXNG_URL" envDefault:"http://searxng:8080" jsonschema:"format=uri" description:"SearXNG service URL"`

	// Vector store URL
	VectorStoreURL string `yaml:"vector_store_url" json:"vector_store_url" env:"VECTOR_STORE_URL" envDefault:"http://vector-store:3015" jsonschema:"format=uri" description:"Vector store service URL"`

	// Sandbox Fusion URL
	SandboxFusionURL string `yaml:"sandbox_fusion_url" json:"sandbox_fusion_url" env:"SANDBOX_FUSION_URL" envDefault:"http://sandboxfusion:8080" jsonschema:"format=uri" description:"SandboxFusion service URL"`

	// Sandbox require approval
	SandboxRequireApproval bool `yaml:"sandbox_require_approval" json:"sandbox_require_approval" env:"SANDBOX_FUSION_REQUIRE_APPROVAL" envDefault:"true" description:"Require approval for sandbox execution"`

	// MCP config file path (relative to service root)
	MCPConfigFile string `yaml:"mcp_config_file" json:"mcp_config_file" env:"MCP_CONFIG_FILE" envDefault:"configs/mcp-providers.yml" description:"MCP provider config file path (CI/CD managed)"`
}

// MediaAPIConfig contains settings for the Media API service
type MediaAPIConfig struct {
	// HTTP port
	HTTPPort int `yaml:"http_port" json:"http_port" env:"MEDIA_API_PORT" envDefault:"8285" jsonschema:"minimum=1,maximum=65535" description:"Media API HTTP port"`

	// Log level
	LogLevel string `yaml:"log_level" json:"log_level" env:"LOG_LEVEL" envDefault:"info" jsonschema:"enum=debug,enum=info,enum=warn,enum=error" description:"Log level"`

	// Maximum upload size in bytes
	MaxUploadBytes int64 `yaml:"max_upload_bytes" json:"max_upload_bytes" env:"MEDIA_MAX_BYTES" envDefault:"20971520" jsonschema:"minimum=1" description:"Maximum upload size in bytes (20MB)"`

	// Retention days for media files
	RetentionDays int `yaml:"retention_days" json:"retention_days" env:"MEDIA_RETENTION_DAYS" envDefault:"30" jsonschema:"minimum=1" description:"Media retention in days"`

	// Proxy download through API
	ProxyDownload bool `yaml:"proxy_download" json:"proxy_download" env:"MEDIA_PROXY_DOWNLOAD" envDefault:"true" description:"Proxy downloads through API"`

	// Remote fetch timeout
	RemoteFetchTimeout time.Duration `yaml:"remote_fetch_timeout" json:"remote_fetch_timeout" env:"MEDIA_REMOTE_FETCH_TIMEOUT" envDefault:"15s" description:"Remote fetch timeout"`

	// S3 settings
	S3 S3Config `yaml:"s3" json:"s3"`
}

// S3Config contains S3/object storage settings
type S3Config struct {
	// S3 endpoint URL
	Endpoint string `yaml:"endpoint" json:"endpoint" env:"MEDIA_S3_ENDPOINT" envDefault:"https://s3.menlo.ai" jsonschema:"format=uri" description:"S3 endpoint URL"`

	// S3 public endpoint (for presigned URLs)
	PublicEndpoint string `yaml:"public_endpoint" json:"public_endpoint" env:"MEDIA_S3_PUBLIC_ENDPOINT" jsonschema:"format=uri" description:"S3 public endpoint URL"`

	// S3 region
	Region string `yaml:"region" json:"region" env:"MEDIA_S3_REGION" envDefault:"us-west-2" description:"S3 region"`

	// S3 bucket name
	Bucket string `yaml:"bucket" json:"bucket" env:"MEDIA_S3_BUCKET" envDefault:"platform-dev" description:"S3 bucket name"`

	// S3 access key (from secrets)
	AccessKey string `yaml:"access_key,omitempty" json:"access_key,omitempty" env:"MEDIA_S3_ACCESS_KEY" description:"S3 access key (from secret provider)"`

	// S3 secret key (from secrets)
	SecretKey string `yaml:"secret_key,omitempty" json:"secret_key,omitempty" env:"MEDIA_S3_SECRET_KEY" description:"S3 secret key (from secret provider)"`

	// Use path-style addressing
	UsePathStyle bool `yaml:"use_path_style" json:"use_path_style" env:"MEDIA_S3_USE_PATH_STYLE" envDefault:"true" description:"Use S3 path-style addressing"`

	// Presigned URL TTL
	PresignTTL time.Duration `yaml:"presign_ttl" json:"presign_ttl" env:"MEDIA_S3_PRESIGN_TTL" envDefault:"5m" description:"Presigned URL TTL"`
}

// ResponseAPIConfig contains settings for the Response API service
type ResponseAPIConfig struct {
	// HTTP port
	HTTPPort int `yaml:"http_port" json:"http_port" env:"RESPONSE_API_PORT" envDefault:"8082" jsonschema:"minimum=1,maximum=65535" description:"Response API HTTP port"`

	// Log level
	LogLevel string `yaml:"log_level" json:"log_level" env:"LOG_LEVEL" envDefault:"info" jsonschema:"enum=debug,enum=info,enum=warn,enum=error" description:"Log level"`

	// LLM API URL
	LLMAPIURL string `yaml:"llm_api_url" json:"llm_api_url" env:"LLM_API_URL" envDefault:"http://llm-api:8080" jsonschema:"format=uri" description:"LLM API service URL"`

	// MCP Tools URL
	MCPToolsURL string `yaml:"mcp_tools_url" json:"mcp_tools_url" env:"MCP_TOOLS_URL" envDefault:"http://mcp-tools:8091" jsonschema:"format=uri" description:"MCP Tools service URL"`

	// Maximum tool execution depth
	MaxToolDepth int `yaml:"max_tool_depth" json:"max_tool_depth" env:"MAX_TOOL_EXECUTION_DEPTH" envDefault:"8" jsonschema:"minimum=1,maximum=20" description:"Maximum tool execution depth"`

	// Tool execution timeout
	ToolTimeout time.Duration `yaml:"tool_timeout" json:"tool_timeout" env:"TOOL_EXECUTION_TIMEOUT" envDefault:"45s" description:"Tool execution timeout"`
}

// InferenceConfig contains settings for inference services
type InferenceConfig struct {
	VLLM VLLMConfig `yaml:"vllm" json:"vllm"`
}

// VLLMConfig contains vLLM inference server settings
type VLLMConfig struct {
	// Enable vLLM
	Enabled bool `yaml:"enabled" json:"enabled" env:"VLLM_ENABLED" envDefault:"true" description:"Enable vLLM inference"`

	// vLLM port
	Port int `yaml:"port" json:"port" env:"VLLM_PORT" envDefault:"8001" jsonschema:"minimum=1,maximum=65535" description:"vLLM HTTP port"`

	// Model to load
	Model string `yaml:"model" json:"model" env:"VLLM_MODEL" envDefault:"Qwen/Qwen2.5-0.5B-Instruct" description:"vLLM model name"`

	// Served model name
	ServedName string `yaml:"served_name" json:"served_name" env:"VLLM_SERVED_NAME" envDefault:"qwen2.5-0.5b-instruct" description:"vLLM served model name"`

	// GPU utilization (0.0-1.0)
	GPUUtilization float64 `yaml:"gpu_utilization" json:"gpu_utilization" env:"VLLM_GPU_UTIL" envDefault:"0.66" jsonschema:"minimum=0,maximum=1" description:"GPU utilization ratio"`

	// vLLM internal API key (from secrets)
	InternalKey string `yaml:"internal_key,omitempty" json:"internal_key,omitempty" env:"VLLM_INTERNAL_KEY" description:"vLLM internal API key (from secret provider)"`

	// HuggingFace token (from secrets)
	HFToken string `yaml:"hf_token,omitempty" json:"hf_token,omitempty" env:"HF_TOKEN" description:"HuggingFace token (from secret provider)"`
}

// MonitoringConfig contains observability and monitoring settings
type MonitoringConfig struct {
	OTEL       OTELConfig       `yaml:"otel" json:"otel"`
	Prometheus PrometheusConfig `yaml:"prometheus" json:"prometheus"`
	Grafana    GrafanaConfig    `yaml:"grafana" json:"grafana"`
	Jaeger     JaegerConfig     `yaml:"jaeger" json:"jaeger"`
}

// OTELConfig contains OpenTelemetry settings
type OTELConfig struct {
	// Enable OpenTelemetry
	Enabled bool `yaml:"enabled" json:"enabled" env:"OTEL_ENABLED" envDefault:"false" description:"Enable OpenTelemetry tracing"`

	// Service name
	ServiceName string `yaml:"service_name" json:"service_name" env:"OTEL_SERVICE_NAME" envDefault:"llm-api" description:"OpenTelemetry service name"`

	// OTLP exporter endpoint
	Endpoint string `yaml:"endpoint" json:"endpoint" env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"http://otel-collector:4318" jsonschema:"format=uri" description:"OTLP exporter endpoint"`

	// HTTP port
	HTTPPort int `yaml:"http_port" json:"http_port" env:"OTEL_HTTP_PORT" envDefault:"4318" jsonschema:"minimum=1,maximum=65535" description:"OTLP HTTP port"`

	// gRPC port
	GRPCPort int `yaml:"grpc_port" json:"grpc_port" env:"OTEL_GRPC_PORT" envDefault:"4317" jsonschema:"minimum=1,maximum=65535" description:"OTLP gRPC port"`
}

// PrometheusConfig contains Prometheus settings
type PrometheusConfig struct {
	// Prometheus port
	Port int `yaml:"port" json:"port" env:"PROMETHEUS_PORT" envDefault:"9090" jsonschema:"minimum=1,maximum=65535" description:"Prometheus HTTP port"`
}

// GrafanaConfig contains Grafana settings
type GrafanaConfig struct {
	// Grafana port
	Port int `yaml:"port" json:"port" env:"GRAFANA_PORT" envDefault:"3001" jsonschema:"minimum=1,maximum=65535" description:"Grafana HTTP port"`

	// Grafana admin user
	AdminUser string `yaml:"admin_user" json:"admin_user" env:"GRAFANA_ADMIN_USER" envDefault:"admin" description:"Grafana admin username"`

	// Grafana admin password (from secrets)
	AdminPassword string `yaml:"admin_password,omitempty" json:"admin_password,omitempty" env:"GRAFANA_ADMIN_PASSWORD" description:"Grafana admin password (from secret provider)"`
}

// JaegerConfig contains Jaeger settings
type JaegerConfig struct {
	// Jaeger UI port
	UIPort int `yaml:"ui_port" json:"ui_port" env:"JAEGER_UI_PORT" envDefault:"16686" jsonschema:"minimum=1,maximum=65535" description:"Jaeger UI port"`
}
