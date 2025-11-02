package config

import (
	"fmt"
	"os"

	"jan-server/services/llm-api/domain/model"

	"gopkg.in/yaml.v3"
)

type ProviderConfig struct {
	PublicID    string            `yaml:"public_id"`
	DisplayName string            `yaml:"display_name"`
	Kind        string            `yaml:"kind"`
	BaseURL     string            `yaml:"base_url"`
	Active      bool              `yaml:"active"`
	IsModerated bool              `yaml:"is_moderated"`
	Metadata    map[string]string `yaml:"metadata"`
}

type ProvidersYAML struct {
	Providers []ProviderConfig `yaml:"providers"`
}

// LoadProvidersFromYAML loads provider configurations from YAML file
func LoadProvidersFromYAML(filePath string) ([]*model.Provider, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read providers file: %w", err)
	}

	var cfg ProvidersYAML
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal providers yaml: %w", err)
	}

	providers := make([]*model.Provider, 0, len(cfg.Providers))
	for _, pc := range cfg.Providers {
		provider := &model.Provider{
			PublicID:    pc.PublicID,
			DisplayName: pc.DisplayName,
			Kind:        model.ProviderKind(pc.Kind),
			BaseURL:     pc.BaseURL,
			Active:      pc.Active,
			IsModerated: pc.IsModerated,
			Metadata:    pc.Metadata,
		}
		providers = append(providers, provider)
	}

	return providers, nil
}
