package config

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// ConfigLoader loads configuration from multiple sources with explicit precedence
type ConfigLoader struct {
	config     *Config
	sources    []ConfigSource
	provenance map[string]ProvenanceInfo
}

// ConfigSource represents a source of configuration values
type ConfigSource interface {
	// Load applies configuration from this source to the config
	Load(ctx context.Context, cfg *Config) error

	// Priority returns the precedence priority (higher = takes precedence)
	// 100 = Struct defaults (lowest)
	// 200 = YAML defaults
	// 300 = Environment YAML
	// 400 = Secrets
	// 500 = Environment variables
	// 600 = CLI flags (highest)
	Priority() int

	// Name returns the human-readable name of this source
	Name() string
}

// ProvenanceInfo tracks where a configuration value came from
type ProvenanceInfo struct {
	Source   string      // Name of the ConfigSource
	Priority int         // Priority level
	Value    interface{} // The actual value
	Path     string      // Config path (e.g., "infrastructure.database.postgres.port")
}

// LoaderOption configures the ConfigLoader
type LoaderOption func(*ConfigLoader) error

// New creates a new ConfigLoader with the specified options
func New(ctx context.Context, environment string, opts ...LoaderOption) (*ConfigLoader, error) {
	loader := &ConfigLoader{
		config:     &Config{},
		provenance: make(map[string]ProvenanceInfo),
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(loader); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	// Build default source stack if not provided
	if len(loader.sources) == 0 {
		loader.sources = []ConfigSource{
			&StructDefaultSource{},
			&YAMLDefaultSource{path: "config/defaults.yaml"},
			&YAMLEnvSource{environment: environment},
			&EnvVarSource{},
		}
	}

	// Load configuration in precedence order (low to high priority)
	for _, source := range loader.sources {
		if err := source.Load(ctx, loader.config); err != nil {
			return nil, fmt.Errorf("load from %s: %w", source.Name(), err)
		}

		// Track provenance (will be implemented properly in next iteration)
		loader.trackProvenance(source)
	}

	// Validate final configuration
	if err := loader.Validate(loader.config); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return loader, nil
}

// NewConfigLoader is a convenience function that creates a loader with default configuration
func NewConfigLoader(environment, defaultsPath string) *ConfigLoader {
	if defaultsPath == "" {
		defaultsPath = "config/defaults.yaml"
	}

	return &ConfigLoader{
		config:     &Config{},
		provenance: make(map[string]ProvenanceInfo),
		sources: []ConfigSource{
			&StructDefaultSource{},                   // Priority 100
			&YAMLDefaultSource{path: defaultsPath},   // Priority 200
			&YAMLEnvSource{environment: environment}, // Priority 300
			&EnvVarSource{},                          // Priority 500
		},
	}
}

// Load executes the configuration loading process
func (l *ConfigLoader) Load(ctx context.Context) (*Config, error) {
	// Load configuration in precedence order (low to high priority)
	for _, source := range l.sources {
		if err := source.Load(ctx, l.config); err != nil {
			return nil, fmt.Errorf("load from %s: %w", source.Name(), err)
		}

		// Track provenance
		l.trackProvenance(source)
	}

	// Validate final configuration
	if err := l.Validate(l.config); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return l.config, nil
}

// WithSources sets custom configuration sources
func WithSources(sources ...ConfigSource) LoaderOption {
	return func(l *ConfigLoader) error {
		l.sources = sources
		return nil
	}
}

// Get returns the loaded configuration
func (l *ConfigLoader) Get() *Config {
	return l.config
}

// AllProvenance returns all provenance information
func (l *ConfigLoader) AllProvenance() map[string]ProvenanceInfo {
	return l.provenance
}

// Provenance returns a human-readable string of all configuration sources
func (l *ConfigLoader) Provenance() string {
	var result strings.Builder
	result.WriteString("Configuration Sources (priority order):\n")

	// Sort sources by priority
	sortedSources := make([]ConfigSource, len(l.sources))
	copy(sortedSources, l.sources)

	for i := 0; i < len(sortedSources)-1; i++ {
		for j := i + 1; j < len(sortedSources); j++ {
			if sortedSources[i].Priority() > sortedSources[j].Priority() {
				sortedSources[i], sortedSources[j] = sortedSources[j], sortedSources[i]
			}
		}
	}

	for _, source := range sortedSources {
		result.WriteString(fmt.Sprintf("  [%d] %s\n", source.Priority(), source.Name()))
	}

	result.WriteString("\nConfiguration Values by Source:\n")
	for path, info := range l.provenance {
		result.WriteString(fmt.Sprintf("  %s: %s (priority %d)\n", path, info.Source, info.Priority))
	}

	return result.String()
}

// Validate performs validation on the loaded configuration
func (l *ConfigLoader) Validate(cfg *Config) error {
	// Basic validation - check required fields
	if cfg.Meta.Version == "" {
		return fmt.Errorf("meta.version is required")
	}
	if cfg.Meta.Environment == "" {
		return fmt.Errorf("meta.environment is required")
	}

	// Infrastructure validation
	if cfg.Infrastructure.Database.Postgres.Host == "" {
		return fmt.Errorf("infrastructure.database.postgres.host is required")
	}
	if cfg.Infrastructure.Database.Postgres.Port < 1 || cfg.Infrastructure.Database.Postgres.Port > 65535 {
		return fmt.Errorf("infrastructure.database.postgres.port must be between 1 and 65535")
	}

	// Auth validation
	if cfg.Infrastructure.Auth.Keycloak.BaseURL == "" {
		return fmt.Errorf("infrastructure.auth.keycloak.base_url is required")
	}

	// Services validation
	if cfg.Services.LLMAPI.HTTPPort < 1 || cfg.Services.LLMAPI.HTTPPort > 65535 {
		return fmt.Errorf("services.llm_api.http_port must be between 1 and 65535")
	}

	return nil
}

// trackProvenance records where configuration values came from
// This is a simplified implementation - full implementation will track all fields
func (l *ConfigLoader) trackProvenance(source ConfigSource) {
	// This will be expanded to track individual fields in the next iteration
	// For now, we just record that this source was applied
	l.provenance[source.Name()] = ProvenanceInfo{
		Source:   source.Name(),
		Priority: source.Priority(),
		Path:     source.Name(),
	}
}

// MergeStrategy defines how to merge configuration values
type MergeStrategy int

const (
	// Replace strategy: higher priority completely replaces lower priority
	Replace MergeStrategy = iota
	// Merge strategy: merge maps and slices (for complex objects)
	Merge
)

// merge applies a value from a source to the target based on strategy
func merge(target, source reflect.Value, strategy MergeStrategy) {
	if !source.IsValid() || source.IsZero() {
		return // Don't override with zero values
	}

	switch strategy {
	case Replace:
		if target.CanSet() {
			target.Set(source)
		}
	case Merge:
		// For maps and slices, merge instead of replace
		if target.Kind() == reflect.Map && source.Kind() == reflect.Map {
			if target.IsNil() {
				target.Set(reflect.MakeMap(target.Type()))
			}
			iter := source.MapRange()
			for iter.Next() {
				target.SetMapIndex(iter.Key(), iter.Value())
			}
		} else {
			// Fall back to replace for other types
			if target.CanSet() {
				target.Set(source)
			}
		}
	}
}
