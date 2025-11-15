package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// getTempDir returns a writable temporary directory
func getTempDir() string {
	baseDir := os.Getenv("TEMP")
	if baseDir == "" {
		baseDir = os.Getenv("TMP")
	}
	if baseDir == "" {
		baseDir = "."
	}
	return filepath.Join(baseDir, fmt.Sprintf("config-test-%d", time.Now().UnixNano()))
}

// TestPrecedenceOrder verifies that configuration sources are applied in correct priority order
func TestPrecedenceOrder(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		setupFiles  func(t *testing.T) string
		expected    Config
		checkFields func(t *testing.T, cfg *Config)
	}{
		{
			name:     "struct defaults only",
			setupEnv: func() {},
			setupFiles: func(t *testing.T) string {
				dir := getTempDir()
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			checkFields: func(t *testing.T, cfg *Config) {
				// Should have struct defaults
				if cfg.Meta.Version != "1.0.0" {
					t.Errorf("expected version 1.0.0, got %s", cfg.Meta.Version)
				}
				if cfg.Infrastructure.Database.Postgres.Port != 5432 {
					t.Errorf("expected port 5432, got %d", cfg.Infrastructure.Database.Postgres.Port)
				}
			},
		},
		{
			name:     "yaml defaults override struct",
			setupEnv: func() {},
			setupFiles: func(t *testing.T) string {
				dir := getTempDir()
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				yamlContent := `
meta:
  version: "2.0.0"
infrastructure:
  database:
    postgres:
      port: 5433
`
				os.MkdirAll(filepath.Join(dir, "config"), 0755)
				os.WriteFile(filepath.Join(dir, "config", "defaults.yaml"), []byte(yamlContent), 0644)
				return dir
			},
			checkFields: func(t *testing.T, cfg *Config) {
				if cfg.Meta.Version != "2.0.0" {
					t.Errorf("expected version 2.0.0, got %s", cfg.Meta.Version)
				}
				if cfg.Infrastructure.Database.Postgres.Port != 5433 {
					t.Errorf("expected port 5433, got %d", cfg.Infrastructure.Database.Postgres.Port)
				}
			},
		},
		{
			name:     "environment yaml overrides defaults",
			setupEnv: func() {},
			setupFiles: func(t *testing.T) string {
				dir := getTempDir()
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				// defaults.yaml
				defaultYAML := `
meta:
  version: "2.0.0"
  environment: "development"
infrastructure:
  database:
    postgres:
      port: 5433
`
				os.MkdirAll(filepath.Join(dir, "config"), 0755)
				os.WriteFile(filepath.Join(dir, "config", "defaults.yaml"), []byte(defaultYAML), 0644)

				// environments/staging.yaml
				stagingYAML := `
meta:
  environment: "staging"
infrastructure:
  database:
    postgres:
      port: 5434
      host: "staging-db"
`
				os.MkdirAll(filepath.Join(dir, "config", "environments"), 0755)
				os.WriteFile(filepath.Join(dir, "config", "environments", "staging.yaml"), []byte(stagingYAML), 0644)
				return dir
			},
			checkFields: func(t *testing.T, cfg *Config) {
				if cfg.Meta.Version != "2.0.0" {
					t.Errorf("expected version 2.0.0 from defaults, got %s", cfg.Meta.Version)
				}
				if cfg.Meta.Environment != "staging" {
					t.Errorf("expected environment staging, got %s", cfg.Meta.Environment)
				}
				if cfg.Infrastructure.Database.Postgres.Port != 5434 {
					t.Errorf("expected port 5434 from staging, got %d", cfg.Infrastructure.Database.Postgres.Port)
				}
				if cfg.Infrastructure.Database.Postgres.Host != "staging-db" {
					t.Errorf("expected host staging-db, got %s", cfg.Infrastructure.Database.Postgres.Host)
				}
			},
		},
		{
			name: "environment variables override yaml",
			setupEnv: func() {
				os.Setenv("POSTGRES_HOST", "env-db")
				os.Setenv("POSTGRES_PORT", "5435")
			},
			setupFiles: func(t *testing.T) string {
				dir := getTempDir()
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				defaultYAML := `
infrastructure:
  database:
    postgres:
      port: 5433
      host: "default-db"
`
				os.MkdirAll(filepath.Join(dir, "config"), 0755)
				os.WriteFile(filepath.Join(dir, "config", "defaults.yaml"), []byte(defaultYAML), 0644)
				return dir
			},
			checkFields: func(t *testing.T, cfg *Config) {
				if cfg.Infrastructure.Database.Postgres.Host != "env-db" {
					t.Errorf("expected host env-db from env var, got %s", cfg.Infrastructure.Database.Postgres.Host)
				}
				if cfg.Infrastructure.Database.Postgres.Port != 5435 {
					t.Errorf("expected port 5435 from env var, got %d", cfg.Infrastructure.Database.Postgres.Port)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment
			os.Clearenv()

			// Setup environment variables
			tt.setupEnv()
			defer os.Clearenv()

			// Setup files
			dir := tt.setupFiles(t)
			defer os.RemoveAll(dir)

			originalDir, _ := os.Getwd()
			os.Chdir(dir)
			defer os.Chdir(originalDir)

			// Create loader and load config
			loader := NewConfigLoader("staging", "config/defaults.yaml")
			cfg, err := loader.Load(context.Background())
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			// Check fields
			tt.checkFields(t, cfg)
		})
	}
}

// TestEnvVarParsing tests different data type parsing from environment variables
func TestEnvVarParsing(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name: "string values",
			envVars: map[string]string{
				"POSTGRES_HOST": "testhost",
				"POSTGRES_DB":   "testdb",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Infrastructure.Database.Postgres.Host != "testhost" {
					t.Errorf("expected host testhost, got %s", cfg.Infrastructure.Database.Postgres.Host)
				}
				if cfg.Infrastructure.Database.Postgres.Database != "testdb" {
					t.Errorf("expected database testdb, got %s", cfg.Infrastructure.Database.Postgres.Database)
				}
			},
		},
		{
			name: "integer values",
			envVars: map[string]string{
				"POSTGRES_PORT":            "9999",
				"POSTGRES_MAX_CONNECTIONS": "200",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Infrastructure.Database.Postgres.Port != 9999 {
					t.Errorf("expected port 9999, got %d", cfg.Infrastructure.Database.Postgres.Port)
				}
				if cfg.Infrastructure.Database.Postgres.MaxConnections != 200 {
					t.Errorf("expected max connections 200, got %d", cfg.Infrastructure.Database.Postgres.MaxConnections)
				}
			},
		},
		{
			name: "boolean values",
			envVars: map[string]string{
				"AUTO_MIGRATE": "false",
				"OTEL_ENABLED": "true",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Services.LLMAPI.AutoMigrate {
					t.Errorf("expected auto_migrate false")
				}
				if !cfg.Monitoring.OTEL.Enabled {
					t.Errorf("expected otel enabled true")
				}
			},
		},
		{
			name: "duration values",
			envVars: map[string]string{
				"DB_CONN_MAX_LIFETIME": "45m",
				"MEDIA_S3_PRESIGN_TTL": "10m",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Infrastructure.Database.Postgres.ConnMaxLifetime != 45*time.Minute {
					t.Errorf("expected 45m, got %v", cfg.Infrastructure.Database.Postgres.ConnMaxLifetime)
				}
				if cfg.Services.MediaAPI.S3.PresignTTL != 10*time.Minute {
					t.Errorf("expected 10m, got %v", cfg.Services.MediaAPI.S3.PresignTTL)
				}
			},
		},
		{
			name: "slice values",
			envVars: map[string]string{
				"KEYCLOAK_FEATURES": "token-exchange,preview,admin-api",
			},
			validate: func(t *testing.T, cfg *Config) {
				expected := []string{"token-exchange", "preview", "admin-api"}
				if len(cfg.Infrastructure.Auth.Keycloak.Features) != len(expected) {
					t.Errorf("expected %d features, got %d", len(expected), len(cfg.Infrastructure.Auth.Keycloak.Features))
				}
				for i, v := range expected {
					if cfg.Infrastructure.Auth.Keycloak.Features[i] != v {
						t.Errorf("expected feature[%d]=%s, got %s", i, v, cfg.Infrastructure.Auth.Keycloak.Features[i])
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer os.Clearenv()

			loader := NewConfigLoader("development", "")
			cfg, err := loader.Load(context.Background())
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			tt.validate(t, cfg)
		})
	}
}

// TestProvenance verifies that we can track which source provided each value
func TestProvenance(t *testing.T) {
	os.Clearenv()
	os.Setenv("POSTGRES_HOST", "envvar-host")
	defer os.Clearenv()

	dir := getTempDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	yamlContent := `
infrastructure:
  database:
    postgres:
      port: 5433
`
	os.MkdirAll(filepath.Join(dir, "config"), 0755)
	os.WriteFile(filepath.Join(dir, "config", "defaults.yaml"), []byte(yamlContent), 0644)

	originalDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(originalDir)

	loader := NewConfigLoader("development", "config/defaults.yaml")
	cfg, err := loader.Load(context.Background())
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	provenance := loader.Provenance()
	t.Logf("Provenance:\n%s", provenance)

	// Verify the provenance tracking works
	if cfg.Infrastructure.Database.Postgres.Host != "envvar-host" {
		t.Errorf("expected host from env var")
	}
	if cfg.Infrastructure.Database.Postgres.Port != 5433 {
		t.Errorf("expected port from yaml defaults")
	}

	// Provenance string should mention both sources
	if !contains(provenance, "env-vars") {
		t.Errorf("provenance should mention env-vars source")
	}
	if !contains(provenance, "yaml-defaults") {
		t.Errorf("provenance should mention yaml-defaults source")
	}
}

// TestValidation tests the config validation
func TestValidation(t *testing.T) {
	tests := []struct {
		name         string
		modifyConfig func(*Config)
		expectError  bool
		errorMsg     string
	}{
		{
			name: "valid config",
			modifyConfig: func(cfg *Config) {
				// Default config should be valid
			},
			expectError: false,
		},
		{
			name: "invalid database port",
			modifyConfig: func(cfg *Config) {
				cfg.Infrastructure.Database.Postgres.Port = 0
			},
			expectError: true,
			errorMsg:    "port",
		},
		{
			name: "missing database host",
			modifyConfig: func(cfg *Config) {
				cfg.Infrastructure.Database.Postgres.Host = ""
			},
			expectError: true,
			errorMsg:    "host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewConfigLoader("development", "")
			cfg, err := loader.Load(context.Background())
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			tt.modifyConfig(cfg)

			err = loader.Validate(cfg)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
				} else if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

// TestMergeConfigs tests the config merging logic
func TestMergeConfigs(t *testing.T) {
	target := &Config{
		Meta: MetaConfig{
			Version:     "1.0.0",
			Environment: "dev",
		},
		Infrastructure: InfrastructureConfig{
			Database: DatabaseConfig{
				Postgres: PostgresConfig{
					Host: "localhost",
					Port: 5432,
				},
			},
		},
	}

	source := &Config{
		Meta: MetaConfig{
			Environment: "prod", // Should override
		},
		Infrastructure: InfrastructureConfig{
			Database: DatabaseConfig{
				Postgres: PostgresConfig{
					Port: 5433,      // Should override
					User: "newuser", // Should add
				},
			},
		},
	}

	mergeConfigs(target, source)

	if target.Meta.Version != "1.0.0" {
		t.Errorf("version should remain 1.0.0, got %s", target.Meta.Version)
	}
	if target.Meta.Environment != "prod" {
		t.Errorf("environment should be prod, got %s", target.Meta.Environment)
	}
	if target.Infrastructure.Database.Postgres.Host != "localhost" {
		t.Errorf("host should remain localhost, got %s", target.Infrastructure.Database.Postgres.Host)
	}
	if target.Infrastructure.Database.Postgres.Port != 5433 {
		t.Errorf("port should be 5433, got %d", target.Infrastructure.Database.Postgres.Port)
	}
	if target.Infrastructure.Database.Postgres.User != "newuser" {
		t.Errorf("user should be newuser, got %s", target.Infrastructure.Database.Postgres.User)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
