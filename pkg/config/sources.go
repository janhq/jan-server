package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// StructDefaultSource provides defaults from Go struct tags (envDefault)
type StructDefaultSource struct{}

func (s *StructDefaultSource) Load(ctx context.Context, cfg *Config) error {
	// Defaults are already set in buildDefaultConfig from codegen/yaml.go
	// We just need to populate the config with those values
	defaults := buildDefaultConfigForLoader()
	*cfg = *defaults
	return nil
}

func (s *StructDefaultSource) Priority() int {
	return 100 // Lowest priority
}

func (s *StructDefaultSource) Name() string {
	return "struct-defaults"
}

// buildDefaultConfigForLoader creates a Config with all default values
func buildDefaultConfigForLoader() *Config {
	return &Config{
		Meta: MetaConfig{
			Version:     "1.0.0",
			Environment: "development",
		},
		Infrastructure: InfrastructureConfig{
			Database: DatabaseConfig{
				Postgres: PostgresConfig{
					Host:            "api-db",
					Port:            5432,
					User:            "jan_user",
					Database:        "jan_llm_api",
					SSLMode:         "disable",
					MaxConnections:  100,
					MaxIdleConns:    5,
					MaxOpenConns:    15,
					ConnMaxLifetime: 30 * time.Minute,
				},
			},
			Auth: AuthConfig{
				Keycloak: KeycloakConfig{
					BaseURL:             "http://keycloak:8085",
					Realm:               "jan",
					HTTPPort:            8085,
					AdminUser:           "admin",
					AdminRealm:          "master",
					AdminClientID:       "admin-cli",
					BackendClientID:     "backend",
					TargetClientID:      "jan-client",
					OAuthRedirectURI:    "http://localhost:8000/auth/callback",
					Issuer:              "http://localhost:8085/realms/jan",
					Audience:            "jan-client",
					RefreshJWKSInterval: 5 * time.Minute,
					AuthClockSkew:       60 * time.Second,
					GuestRole:           "guest",
					Features:            []string{"token-exchange", "preview"},
				},
			},
			Gateway: GatewayConfig{
				Kong: KongConfig{
					HTTPPort:  8000,
					AdminPort: 8001,
					AdminURL:  "http://kong:8001",
					LogLevel:  "info",
				},
			},
		},
		Services: ServicesConfig{
			LLMAPI: LLMAPIConfig{
				HTTPPort:               8080,
				MetricsPort:            9091,
				LogLevel:               "info",
				LogFormat:              "json",
				AutoMigrate:            true,
				ProviderConfigFile:     "config/providers.yml",
				ProviderConfigSet:      "default",
				ProviderConfigsEnabled: true,
				APIKey: APIKeyConfig{
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
			MCPTools: MCPToolsConfig{
				HTTPPort:               8091,
				LogLevel:               "info",
				LogFormat:              "json",
				SearchEngine:           "serper",
				SearxngURL:             "http://searxng:8080",
				VectorStoreURL:         "http://vector-store:3015",
				SandboxFusionURL:       "http://sandboxfusion:8080",
				SandboxRequireApproval: true,
				MCPConfigFile:          "configs/mcp-providers.yml",
			},
			MediaAPI: MediaAPIConfig{
				HTTPPort:           8285,
				LogLevel:           "info",
				MaxUploadBytes:     20971520,
				RetentionDays:      30,
				ProxyDownload:      true,
				RemoteFetchTimeout: 15 * time.Second,
				S3: S3Config{
					Endpoint:     "https://s3.menlo.ai",
					Region:       "us-west-2",
					Bucket:       "platform-dev",
					UsePathStyle: true,
					PresignTTL:   5 * time.Minute,
				},
			},
			ResponseAPI: ResponseAPIConfig{
				HTTPPort:     8082,
				LogLevel:     "info",
				LLMAPIURL:    "http://llm-api:8080",
				MCPToolsURL:  "http://mcp-tools:8091",
				MaxToolDepth: 8,
				ToolTimeout:  45 * time.Second,
			},
		},
		Inference: InferenceConfig{
			VLLM: VLLMConfig{
				Enabled:        true,
				Port:           8001,
				Model:          "Qwen/Qwen2.5-0.5B-Instruct",
				ServedName:     "qwen2.5-0.5b-instruct",
				GPUUtilization: 0.66,
			},
		},
		Monitoring: MonitoringConfig{
			OTEL: OTELConfig{
				Enabled:     false,
				ServiceName: "llm-api",
				Endpoint:    "http://otel-collector:4318",
				HTTPPort:    4318,
				GRPCPort:    4317,
			},
			Prometheus: PrometheusConfig{
				Port: 9090,
			},
			Grafana: GrafanaConfig{
				Port:      3001,
				AdminUser: "admin",
			},
			Jaeger: JaegerConfig{
				UIPort: 16686,
			},
		},
	}
}

// YAMLDefaultSource loads defaults from config/defaults.yaml
type YAMLDefaultSource struct {
	path string
}

func (s *YAMLDefaultSource) Load(ctx context.Context, cfg *Config) error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		// It's OK if defaults.yaml doesn't exist yet, struct defaults will be used
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read yaml file: %w", err)
	}

	var yamlCfg Config
	if err := yaml.Unmarshal(data, &yamlCfg); err != nil {
		return fmt.Errorf("unmarshal yaml: %w", err)
	}

	// Merge YAML config into cfg (non-zero values override)
	mergeConfigs(cfg, &yamlCfg)
	return nil
}

func (s *YAMLDefaultSource) Priority() int {
	return 200
}

func (s *YAMLDefaultSource) Name() string {
	return "yaml-defaults"
}

// YAMLEnvSource loads environment-specific overrides from config/environments/{env}.yaml
type YAMLEnvSource struct {
	environment string
}

func (s *YAMLEnvSource) Load(ctx context.Context, cfg *Config) error {
	path := filepath.Join("config", "environments", fmt.Sprintf("%s.yaml", s.environment))

	data, err := os.ReadFile(path)
	if err != nil {
		// It's OK if environment file doesn't exist
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read environment yaml: %w", err)
	}

	var envCfg Config
	if err := yaml.Unmarshal(data, &envCfg); err != nil {
		return fmt.Errorf("unmarshal environment yaml: %w", err)
	}

	mergeConfigs(cfg, &envCfg)
	return nil
}

func (s *YAMLEnvSource) Priority() int {
	return 300
}

func (s *YAMLEnvSource) Name() string {
	return fmt.Sprintf("yaml-env-%s", s.environment)
}

// EnvVarSource loads configuration from environment variables
type EnvVarSource struct{}

func (s *EnvVarSource) Load(ctx context.Context, cfg *Config) error {
	// Use reflection to find all env tags and apply environment variables
	applyEnvVars(reflect.ValueOf(cfg).Elem())
	return nil
}

func (s *EnvVarSource) Priority() int {
	return 500
}

func (s *EnvVarSource) Name() string {
	return "env-vars"
}

// applyEnvVars recursively applies environment variables to config fields
func applyEnvVars(v reflect.Value) {
	if !v.IsValid() {
		return
	}

	t := v.Type()

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fieldType := t.Field(i)

			// Check for env tag
			envTag := fieldType.Tag.Get("env")
			if envTag != "" {
				// Get environment variable value
				if envVal := os.Getenv(envTag); envVal != "" {
					setFieldFromString(field, envVal)
				}
			}

			// Recurse into nested structs
			if field.Kind() == reflect.Struct {
				applyEnvVars(field)
			}
		}

	case reflect.Ptr:
		if !v.IsNil() {
			applyEnvVars(v.Elem())
		}
	}
}

// setFieldFromString sets a field value from a string representation
func setFieldFromString(field reflect.Value, value string) {
	if !field.CanSet() {
		return
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			// Handle time.Duration
			if d, err := time.ParseDuration(value); err == nil {
				field.Set(reflect.ValueOf(d))
			}
		} else {
			if i, err := strconv.ParseInt(value, 10, 64); err == nil {
				field.SetInt(i)
			}
		}

	case reflect.Float32, reflect.Float64:
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			field.SetFloat(f)
		}

	case reflect.Bool:
		if b, err := strconv.ParseBool(value); err == nil {
			field.SetBool(b)
		}

	case reflect.Slice:
		// Handle string slices
		if field.Type().Elem().Kind() == reflect.String {
			// Split by comma for slice values
			parts := strings.Split(value, ",")
			slice := reflect.MakeSlice(field.Type(), len(parts), len(parts))
			for i, part := range parts {
				slice.Index(i).SetString(strings.TrimSpace(part))
			}
			field.Set(slice)
		}
	}
}

// mergeConfigs merges source into target (non-zero source values override target)
func mergeConfigs(target, source *Config) {
	mergeStruct(reflect.ValueOf(target).Elem(), reflect.ValueOf(source).Elem())
}

// mergeStruct recursively merges source struct into target struct
func mergeStruct(target, source reflect.Value) {
	if !target.IsValid() || !source.IsValid() {
		return
	}

	for i := 0; i < source.NumField(); i++ {
		sourceField := source.Field(i)
		targetField := target.Field(i)

		if !targetField.CanSet() {
			continue
		}

		// Skip zero values (don't override with empty/zero values)
		if sourceField.IsZero() {
			continue
		}

		switch sourceField.Kind() {
		case reflect.Struct:
			// Recurse into nested structs
			mergeStruct(targetField, sourceField)

		case reflect.Slice:
			// For slices, replace if source has values
			if sourceField.Len() > 0 {
				targetField.Set(sourceField)
			}

		case reflect.Map:
			// For maps, merge keys
			if sourceField.Len() > 0 {
				if targetField.IsNil() {
					targetField.Set(reflect.MakeMap(targetField.Type()))
				}
				iter := sourceField.MapRange()
				for iter.Next() {
					targetField.SetMapIndex(iter.Key(), iter.Value())
				}
			}

		default:
			// For primitive types, just set the value
			targetField.Set(sourceField)
		}
	}
}
