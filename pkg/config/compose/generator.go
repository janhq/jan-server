package compose

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v3"
)

// ConfigData represents the configuration data for templates
type ConfigData map[string]interface{}

// Generator generates docker-compose files from config
type Generator struct {
	config    ConfigData
	templates map[string]*template.Template
}

// NewGenerator creates a new compose generator
func NewGenerator(cfg ConfigData) *Generator {
	return &Generator{
		config:    cfg,
		templates: make(map[string]*template.Template),
	}
}

// GenerateInfrastructure generates docker-compose for infrastructure services
func (g *Generator) GenerateInfrastructure(outputPath string) error {
	tmpl := `# Generated from config/defaults.yaml
# DO NOT EDIT - Changes will be overwritten
# To modify, edit config YAML and run: make compose-generate

services:
  # PostgreSQL Database
  api-db:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: {{ .Database.Postgres.User }}
      POSTGRES_PASSWORD: {{ .Database.Postgres.Password }}
      POSTGRES_DB: {{ .Database.Postgres.Database }}
    ports:
      - "{{ .Database.Postgres.Port }}:5432"
    volumes:
      - api-db-data:/var/lib/postgresql/data
      - ./docker/postgres/init:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U {{ .Database.Postgres.User }} -d {{ .Database.Postgres.Database }}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - jan-network
    profiles:
      - infra
      - full

  # Keycloak Database
  keycloak-db:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: {{ .Auth.Keycloak.DbUser }}
      POSTGRES_PASSWORD: {{ .Auth.Keycloak.DbPassword }}
      POSTGRES_DB: {{ .Auth.Keycloak.DbDatabase }}
    ports:
      - "{{ .Auth.Keycloak.DbPort }}:5432"
    volumes:
      - keycloak-db-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U {{ .Auth.Keycloak.DbUser }}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - jan-network
    profiles:
      - infra
      - full

  # Keycloak
  keycloak:
    image: quay.io/keycloak/keycloak:24.0.5
    command: start-dev --import-realm
    restart: unless-stopped
    depends_on:
      keycloak-db:
        condition: service_healthy
    environment:
      KC_DB: postgres
      KC_DB_URL_HOST: keycloak-db
      KC_DB_URL_PORT: 5432
      KC_DB_URL_DATABASE: {{ .Auth.Keycloak.DbDatabase }}
      KC_DB_USERNAME: {{ .Auth.Keycloak.DbUser }}
      KC_DB_PASSWORD: {{ .Auth.Keycloak.DbPassword }}
      KC_HTTP_PORT: {{ .Auth.Keycloak.HttpPort }}
      KEYCLOAK_ADMIN: {{ .Auth.Keycloak.AdminUser }}
      KEYCLOAK_ADMIN_PASSWORD: {{ .Auth.Keycloak.AdminPassword }}
    ports:
      - "{{ .Auth.Keycloak.HttpPort }}:{{ .Auth.Keycloak.HttpPort }}"
    volumes:
      - ./keycloak/import:/opt/keycloak/data/import:ro
    healthcheck:
      test: ["CMD-SHELL", "exec 3<>/dev/tcp/localhost/{{ .Auth.Keycloak.HttpPort }} || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 30
    networks:
      - jan-network
    profiles:
      - infra
      - full

  # Kong Gateway
  kong:
    image: kong:3.7.1-ubuntu
    restart: unless-stopped
    environment:
      KONG_DATABASE: "off"
      KONG_DECLARATIVE_CONFIG: /kong/kong.yml
      KONG_PROXY_ACCESS_LOG: /dev/stdout
      KONG_ADMIN_ACCESS_LOG: /dev/stdout
      KONG_PROXY_ERROR_LOG: /dev/stderr
      KONG_ADMIN_ERROR_LOG: /dev/stderr
      KONG_ADMIN_LISTEN: "0.0.0.0:{{ .Gateway.Kong.AdminPort }}"
      KONG_LOG_LEVEL: {{ .Gateway.Kong.LogLevel }}
    ports:
      - "{{ .Gateway.Kong.HttpPort }}:8000"
      - "{{ .Gateway.Kong.AdminPort }}:8001"
    volumes:
      - ./kong/kong.yml:/kong/kong.yml:ro
      - ./kong/plugins:/usr/local/share/lua/5.1/kong/plugins:ro
    healthcheck:
      test: ["CMD", "kong", "health"]
      interval: 10s
      timeout: 10s
      retries: 10
    networks:
      - jan-network
    profiles:
      - infra
      - full

volumes:
  api-db-data:
  keycloak-db-data:

networks:
  jan-network:
    driver: bridge
`

	t, err := template.New("infrastructure").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, g.config); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// GenerateServices generates docker-compose for API services
func (g *Generator) GenerateServices(outputPath string) error {
	tmpl := `# Generated from config/defaults.yaml
# DO NOT EDIT - Changes will be overwritten
# To modify, edit config YAML and run: make compose-generate

services:
  # LLM API Service
  llm-api:
    build: ../services/llm-api
    restart: unless-stopped
    depends_on:
      api-db:
        condition: service_healthy
    environment:
      # HTTP Server
      HTTP_PORT: {{ .Services.LLMApi.HttpPort }}
      METRICS_PORT: {{ .Services.LLMApi.MetricsPort }}
      
      # Database (constructed DSN)
      DB_POSTGRESQL_WRITE_DSN: "postgres://{{ .Database.Postgres.User }}:{{ .Database.Postgres.Password }}@api-db:5432/{{ .Database.Postgres.Database }}?sslmode=disable"
      DB_POSTGRESQL_READ1_DSN: ""
      
      # Auth
      KEYCLOAK_BASE_URL: {{ .Auth.Keycloak.BaseUrl }}
      KEYCLOAK_REALM: {{ .Auth.Keycloak.Realm }}
      KEYCLOAK_ADMIN: {{ .Auth.Keycloak.AdminUser }}
      KEYCLOAK_ADMIN_PASSWORD: {{ .Auth.Keycloak.AdminPassword }}
      BACKEND_CLIENT_ID: {{ .Auth.Keycloak.BackendClientId }}
      BACKEND_CLIENT_SECRET: {{ .Auth.Keycloak.BackendClientSecret }}
      TARGET_CLIENT_ID: {{ .Auth.Keycloak.TargetClientId }}
      OAUTH_REDIRECT_URI: {{ .Auth.Keycloak.OAuthRedirectUri }}
      JWKS_URL: {{ .Auth.Keycloak.JwksUrl }}
      ISSUER: {{ .Auth.Keycloak.Issuer }}
      AUDIENCE: {{ .Auth.Keycloak.Audience }}
      
      # API Keys
      API_KEY_PREFIX: {{ .Services.LLMApi.ApiKeys.Prefix }}
      API_KEY_DEFAULT_TTL: {{ .Services.LLMApi.ApiKeys.DefaultTtl }}
      API_KEY_MAX_TTL: {{ .Services.LLMApi.ApiKeys.MaxTtl }}
      API_KEY_MAX_PER_USER: {{ .Services.LLMApi.ApiKeys.MaxPerUser }}
      
      # Gateway
      KONG_ADMIN_URL: {{ .Gateway.Kong.AdminUrl }}
      
      # Model Provider
      MODEL_PROVIDER_SECRET: {{ .Services.LLMApi.ModelProvider.Secret }}
      JAN_PROVIDER_CONFIGS: {{ .Services.LLMApi.ModelProvider.Enabled }}
      JAN_PROVIDER_CONFIG_SET: {{ .Services.LLMApi.ModelProvider.ConfigSet }}
      JAN_PROVIDER_CONFIGS_FILE: {{ .Services.LLMApi.ModelProvider.ConfigFile }}
      
      # Model Sync
      MODEL_SYNC_ENABLED: {{ .Services.LLMApi.ModelSync.Enabled }}
      MODEL_SYNC_INTERVAL_MINUTES: {{ .Services.LLMApi.ModelSync.IntervalMinutes }}
      
      # Logging
      LOG_LEVEL: {{ .Services.LLMApi.LogLevel }}
      LOG_FORMAT: {{ .Services.LLMApi.LogFormat }}
      
      # Features
      AUTO_MIGRATE: {{ .Services.LLMApi.AutoMigrate }}
      OTEL_ENABLED: {{ .Monitoring.Otel.Enabled }}
    ports:
      - "{{ .Services.LLMApi.HttpPort }}:{{ .Services.LLMApi.HttpPort }}"
      - "{{ .Services.LLMApi.MetricsPort }}:{{ .Services.LLMApi.MetricsPort }}"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:{{ .Services.LLMApi.HttpPort }}/healthz"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - jan-network
    profiles:
      - api
      - full

  # Media API Service
  media-api:
    build: ../services/media-api
    restart: unless-stopped
    depends_on:
      api-db:
        condition: service_healthy
    environment:
      # HTTP Server
      MEDIA_HTTP_PORT: {{ .Services.MediaApi.HttpPort }}
      
      # Database
      DB_POSTGRESQL_WRITE_DSN: "postgres://{{ .Database.Postgres.User }}:{{ .Database.Postgres.Password }}@api-db:5432/{{ .Database.Postgres.Database }}?sslmode=disable"
      DB_POSTGRESQL_READ1_DSN: ""
      
      # S3 Storage
      MEDIA_S3_BUCKET_NAME: {{ .Services.MediaApi.S3.BucketName }}
      MEDIA_S3_REGION: {{ .Services.MediaApi.S3.Region }}
      MEDIA_S3_ENDPOINT: {{ .Services.MediaApi.S3.Endpoint }}
      MEDIA_S3_ACCESS_KEY_ID: {{ .Services.MediaApi.S3.AccessKeyId }}
      MEDIA_S3_SECRET_ACCESS_KEY: {{ .Services.MediaApi.S3.SecretAccessKey }}
      MEDIA_S3_USE_SSL: {{ .Services.MediaApi.S3.UseSsl }}
      MEDIA_S3_USE_PATH_STYLE: {{ .Services.MediaApi.S3.UsePathStyle }}
      
      # Features
      MEDIA_MAX_UPLOAD_BYTES: {{ .Services.MediaApi.MaxUploadBytes }}
      MEDIA_RETENTION_DAYS: {{ .Services.MediaApi.RetentionDays }}
      
      # Auth
      MEDIA_JWKS_URL: {{ .Auth.Keycloak.JwksUrl }}
      MEDIA_ISSUER: {{ .Auth.Keycloak.Issuer }}
      MEDIA_AUDIENCE: {{ .Auth.Keycloak.Audience }}
      
      # Logging
      MEDIA_LOG_LEVEL: {{ .Services.MediaApi.LogLevel }}
      MEDIA_LOG_FORMAT: {{ .Services.MediaApi.LogFormat }}
      MEDIA_OTEL_ENABLED: {{ .Monitoring.Otel.Enabled }}
    ports:
      - "{{ .Services.MediaApi.HttpPort }}:{{ .Services.MediaApi.HttpPort }}"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:{{ .Services.MediaApi.HttpPort }}/healthz"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - jan-network
    profiles:
      - api
      - full

  # Response API Service
  response-api:
    build: ../services/response-api
    restart: unless-stopped
    depends_on:
      api-db:
        condition: service_healthy
    environment:
      # HTTP Server
      RESPONSE_HTTP_PORT: {{ .Services.ResponseApi.HttpPort }}
      
      # Database
      DB_POSTGRESQL_WRITE_DSN: "postgres://{{ .Database.Postgres.User }}:{{ .Database.Postgres.Password }}@api-db:5432/{{ .Database.Postgres.Database }}?sslmode=disable"
      DB_POSTGRESQL_READ1_DSN: ""
      
      # Service Integration
      RESPONSE_LLM_API_URL: {{ .Services.ResponseApi.LlmApiUrl }}
      RESPONSE_MCP_TOOLS_URL: {{ .Services.ResponseApi.McpToolsUrl }}
      
      # Features
      RESPONSE_MAX_TOOL_DEPTH: {{ .Services.ResponseApi.MaxToolDepth }}
      
      # Auth
      RESPONSE_JWKS_URL: {{ .Auth.Keycloak.JwksUrl }}
      RESPONSE_ISSUER: {{ .Auth.Keycloak.Issuer }}
      RESPONSE_AUDIENCE: {{ .Auth.Keycloak.Audience }}
      
      # Logging
      RESPONSE_LOG_LEVEL: {{ .Services.ResponseApi.LogLevel }}
      RESPONSE_LOG_FORMAT: {{ .Services.ResponseApi.LogFormat }}
      RESPONSE_OTEL_ENABLED: {{ .Monitoring.Otel.Enabled }}
    ports:
      - "{{ .Services.ResponseApi.HttpPort }}:{{ .Services.ResponseApi.HttpPort }}"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:{{ .Services.ResponseApi.HttpPort }}/healthz"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - jan-network
    profiles:
      - api
      - full

networks:
  jan-network:
    external: true
`

	t, err := template.New("services").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, g.config); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// GenerateMCP generates docker-compose for MCP services
func (g *Generator) GenerateMCP(outputPath string) error {
	tmpl := `# Generated from config/defaults.yaml
# DO NOT EDIT - Changes will be overwritten
# To modify, edit config YAML and run: make compose-generate

services:
  # MCP Tools Service
  mcp-tools:
    build: ../services/mcp-tools
    restart: unless-stopped
    environment:
      MCP_TOOLS_HTTP_PORT: {{ .Services.McpTools.HttpPort }}
      MCP_TOOLS_LOG_LEVEL: {{ .Services.McpTools.LogLevel }}
      MCP_TOOLS_LOG_FORMAT: {{ .Services.McpTools.LogFormat }}
      MCP_SEARCH_ENGINE: {{ .Services.McpTools.SearchEngine }}
      MCP_TOOLS_OTEL_ENABLED: {{ .Monitoring.Otel.Enabled }}
      
      # External Services
      VECTOR_STORE_URL: {{ .Services.McpTools.VectorStoreUrl }}
      SEARXNG_URL: {{ .Services.McpTools.SearxngUrl }}
      SANDBOXFUSION_URL: {{ .Services.McpTools.SandboxFusionUrl }}
    ports:
      - "{{ .Services.McpTools.HttpPort }}:{{ .Services.McpTools.HttpPort }}"
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:{{ .Services.McpTools.HttpPort }}/healthz"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - jan-network
    profiles:
      - mcp
      - full

  # Vector Store
  vector-store:
    image: qdrant/qdrant:v1.7.4
    restart: unless-stopped
    ports:
      - "{{ .Services.McpTools.VectorStore.Port }}:6333"
    volumes:
      - vector-store-data:/qdrant/storage
    networks:
      - jan-network
    profiles:
      - mcp
      - full

volumes:
  vector-store-data:

networks:
  jan-network:
    external: true
`

	t, err := template.New("mcp").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, g.config); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// GenerateAll generates all compose files
func (g *Generator) GenerateAll(outputDir string) error {
	if err := g.GenerateInfrastructure(filepath.Join(outputDir, "docker-compose.infrastructure.generated.yml")); err != nil {
		return fmt.Errorf("generate infrastructure: %w", err)
	}

	if err := g.GenerateServices(filepath.Join(outputDir, "docker-compose.services.generated.yml")); err != nil {
		return fmt.Errorf("generate services: %w", err)
	}

	if err := g.GenerateMCP(filepath.Join(outputDir, "docker-compose.mcp.generated.yml")); err != nil {
		return fmt.Errorf("generate mcp: %w", err)
	}

	return nil
}

// ValidateGenerated validates that generated files are valid YAML
func (g *Generator) ValidateGenerated(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	return nil
}
