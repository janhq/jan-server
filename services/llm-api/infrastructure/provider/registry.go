package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

type fileProvider struct {
	Name    string            `yaml:"name"`
	Kind    string            `yaml:"kind"`
	BaseURL string            `yaml:"base_url"`
	Headers map[string]string `yaml:"headers"`
	Models  []ModelConfig     `yaml:"models"`
}

type fileRouting struct {
	DefaultProvider string `yaml:"default_provider"`
}

type fileConfig struct {
	Providers []fileProvider `yaml:"providers"`
	Routing   fileRouting    `yaml:"routing"`
}

// LoadRegistry creates a registry from the YAML configuration file.
func LoadRegistry(ctx context.Context, configPath string, logger zerolog.Logger, client *http.Client) (*Registry, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read providers config: %w", err)
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal providers config: %w", err)
	}

	if len(cfg.Providers) == 0 {
		return nil, fmt.Errorf("providers config is empty")
	}

	registry := NewRegistry()

	for _, pCfg := range cfg.Providers {
		switch pCfg.Kind {
		case "openai":
			var opt []OpenAIOption
			if client != nil {
				opt = append(opt, WithHTTPClient(client))
			}
			provider, err := NewOpenAIProvider(pCfg.Name, pCfg.BaseURL, pCfg.Headers, logger, opt...)
			if err != nil {
				return nil, err
			}
			if err := provider.HealthCheck(ctx); err != nil {
				logger.Warn().Str("provider", pCfg.Name).Err(err).Msg("provider health check failed during boot")
			}
			registry.RegisterProvider(provider)
			for _, modelCfg := range pCfg.Models {
				if modelCfg.ID == "" || modelCfg.ServedName == "" {
					return nil, fmt.Errorf("provider %s has model with missing id or served_name", pCfg.Name)
				}
				registry.RegisterModel(modelCfg, provider)
			}
		default:
			return nil, fmt.Errorf("unsupported provider kind %s", pCfg.Kind)
		}
	}

	if cfg.Routing.DefaultProvider != "" {
		registry.SetDefaultProvider(cfg.Routing.DefaultProvider)
	}

	return registry, nil
}
