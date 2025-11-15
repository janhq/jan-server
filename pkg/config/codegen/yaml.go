package codegen

import (
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/janhq/jan-server/pkg/config"
	"gopkg.in/yaml.v3"
)

// GenerateDefaultsYAML generates config/defaults.yaml from Go struct default tags
func GenerateDefaultsYAML(outputPath string) error {
	cfg := buildDefaultConfig()

	// Create YAML encoder
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

	// Write header comments
	header := `# Jan Server Default Configuration
# Generated from pkg/config/types.go
# DO NOT EDIT MANUALLY - this file is auto-generated
#
# To customize, create environment-specific overrides in:
#   - config/environments/development.yaml
#   - config/environments/staging.yaml
#   - config/environments/production.yaml

`
	if _, err := f.WriteString(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Encode YAML
	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)

	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("encode yaml: %w", err)
	}

	fmt.Printf("✓ Generated %s\n", outputPath)
	return nil
}

// buildDefaultConfig creates a Config with all default values from struct tags
func buildDefaultConfig() *config.Config {
	return &config.Config{
		Meta: config.MetaConfig{
			Version:     "1.0.0",
			Environment: "development",
		},
		Infrastructure: config.InfrastructureConfig{
			Database: config.DatabaseConfig{
				Postgres: config.PostgresConfig{
					Host:            "api-db",
					Port:            5432,
					User:            "jan_user",
					Database:        "jan_llm_api",
					Password:        "", // From secrets
					SSLMode:         "disable",
					MaxConnections:  100,
					MaxIdleConns:    5,
					MaxOpenConns:    15,
					ConnMaxLifetime: 30 * time.Minute,
				},
			},
			Auth: config.AuthConfig{
				Keycloak: config.KeycloakConfig{
					BaseURL:             "http://keycloak:8085",
					PublicURL:           "",
					Realm:               "jan",
					HTTPPort:            8085,
					AdminUser:           "admin",
					AdminPassword:       "", // From secrets
					AdminRealm:          "master",
					AdminClientID:       "admin-cli",
					BackendClientID:     "backend",
					BackendClientSecret: "", // From secrets
					TargetClientID:      "jan-client",
					OAuthRedirectURI:    "http://localhost:8000/auth/callback",
					JWKSURL:             "",
					OIDCDiscoveryURL:    "",
					Issuer:              "http://localhost:8085/realms/jan",
					Audience:            "jan-client",
					RefreshJWKSInterval: 5 * time.Minute,
					AuthClockSkew:       60 * time.Second,
					GuestRole:           "guest",
					Features:            []string{"token-exchange", "preview"},
				},
			},
			Gateway: config.GatewayConfig{
				Kong: config.KongConfig{
					HTTPPort:  8000,
					AdminPort: 8001,
					AdminURL:  "http://kong:8001",
					LogLevel:  "info",
				},
			},
		},
		Services: config.ServicesConfig{
			LLMAPI: config.LLMAPIConfig{
				HTTPPort:               8080,
				MetricsPort:            9091,
				LogLevel:               "info",
				LogFormat:              "json",
				AutoMigrate:            true,
				ProviderConfigFile:     "config/providers.yml",
				ProviderConfigSet:      "default",
				ProviderConfigsEnabled: true,
				APIKey: config.APIKeyConfig{
					Prefix:     "sk_live",
					DefaultTTL: 2160 * time.Hour,
					MaxTTL:     2160 * time.Hour,
					MaxPerUser: 5,
				},
				ModelProviderSecret:      "jan-model-provider-secret-2024",
				ModelSyncEnabled:         true,
				ModelSyncIntervalMinutes: 60,
				MediaResolveURL:          "http://kong:8000/media/v1/media/resolve",
				MediaResolveTimeout:      5 * time.Second,
			},
			MCPTools: config.MCPToolsConfig{
				HTTPPort:               8091,
				LogLevel:               "info",
				LogFormat:              "json",
				SearchEngine:           "serper",
				SerperAPIKey:           "", // From secrets
				SearxngURL:             "http://searxng:8080",
				VectorStoreURL:         "http://vector-store:3015",
				SandboxFusionURL:       "http://sandboxfusion:8080",
				SandboxRequireApproval: true,
				MCPConfigFile:          "configs/mcp-providers.yml",
			},
			MediaAPI: config.MediaAPIConfig{
				HTTPPort:           8285,
				LogLevel:           "info",
				MaxUploadBytes:     20971520, // 20MB
				RetentionDays:      30,
				ProxyDownload:      true,
				RemoteFetchTimeout: 15 * time.Second,
				S3: config.S3Config{
					Endpoint:       "https://s3.menlo.ai",
					PublicEndpoint: "",
					Region:         "us-west-2",
					Bucket:         "platform-dev",
					AccessKey:      "", // From secrets
					SecretKey:      "", // From secrets
					UsePathStyle:   true,
					PresignTTL:     5 * time.Minute,
				},
			},
			ResponseAPI: config.ResponseAPIConfig{
				HTTPPort:     8082,
				LogLevel:     "info",
				LLMAPIURL:    "http://llm-api:8080",
				MCPToolsURL:  "http://mcp-tools:8091",
				MaxToolDepth: 8,
				ToolTimeout:  45 * time.Second,
			},
		},
		Inference: config.InferenceConfig{
			VLLM: config.VLLMConfig{
				Enabled:        true,
				Port:           8001,
				Model:          "Qwen/Qwen2.5-0.5B-Instruct",
				ServedName:     "qwen2.5-0.5b-instruct",
				GPUUtilization: 0.66,
				InternalKey:    "", // From secrets
				HFToken:        "", // From secrets
			},
		},
		Monitoring: config.MonitoringConfig{
			OTEL: config.OTELConfig{
				Enabled:     false,
				ServiceName: "llm-api",
				Endpoint:    "http://otel-collector:4318",
				HTTPPort:    4318,
				GRPCPort:    4317,
			},
			Prometheus: config.PrometheusConfig{
				Port: 9090,
			},
			Grafana: config.GrafanaConfig{
				Port:          3001,
				AdminUser:     "admin",
				AdminPassword: "", // From secrets
			},
			Jaeger: config.JaegerConfig{
				UIPort: 16686,
			},
		},
	}
}

// getStructTag extracts a specific tag value from a struct field
func getStructTag(field reflect.StructField, tagName string) string {
	return field.Tag.Get(tagName)
}
