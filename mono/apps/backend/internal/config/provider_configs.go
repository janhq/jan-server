package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"jan-server/mono/apps/backend/internal/infrastructure/logger"
)

const DefaultProviderConfigFile = "configs/providers.yml"

// ProviderBootstrapEntry describes a provider that should be bootstrapped on startup.
type ProviderBootstrapEntry struct {
	Name                string
	Vendor              string
	Category            string // "llm" or "image"; defaults to "llm"
	BaseURL             string
	Endpoints           []EndpointConfig
	APIKey              string
	Active              bool
	Metadata            map[string]string
	AutoEnableNewModels bool
	SyncModels          bool
}

// ProviderBootstrapConfig maintains all configured provider sets.
type ProviderBootstrapConfig struct {
	sets map[string][]ProviderBootstrapEntry
}

// ProvidersForSet returns a copy of the providers defined for the requested set.
func (c *ProviderBootstrapConfig) ProvidersForSet(name string) []ProviderBootstrapEntry {
	if c == nil {
		return nil
	}
	set := strings.TrimSpace(name)
	if set == "" {
		set = "default"
	}
	list := c.sets[set]
	if len(list) == 0 {
		return nil
	}
	result := make([]ProviderBootstrapEntry, len(list))
	copy(result, list)
	return result
}

// LoadProviderBootstrapConfig parses the yaml file at the provided path.
func LoadProviderBootstrapConfig(path string) (*ProviderBootstrapConfig, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("provider config path is empty!!!")
	}

	log := logger.GetLogger()
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !filepath.IsAbs(cleanPath) {
			altPath := filepath.Clean(filepath.Join("services", "llm-api", cleanPath))
			altData, altErr := os.ReadFile(altPath)
			if altErr != nil {
				return nil, fmt.Errorf("read provider config %q: %w", altPath, altErr)
			}
			data = altData
			cleanPath = altPath
		} else {
			return nil, fmt.Errorf("read provider config %q: %w", cleanPath, err)
		}
	}
	log.Info().Str("path", cleanPath).Msg("loading provider config file")

	var doc providerConfigDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse provider config %q: %w", cleanPath, err)
	}

	if len(doc.Providers) == 0 {
		return nil, fmt.Errorf("provider config %q has no providers defined", cleanPath)
	}

	result := &ProviderBootstrapConfig{
		sets: make(map[string][]ProviderBootstrapEntry),
	}

	for rawSet, entries := range doc.Providers {
		setName := strings.TrimSpace(rawSet)
		if setName == "" || len(entries) == 0 {
			continue
		}
		for idx, entry := range entries {
			entryLogger := log.With().Str("set", setName).Int("index", idx).Str("name", entry.Name).Logger()
			enabled, err := parseEnabled(entry.EnableRaw)
			if err != nil {
				return nil, fmt.Errorf("providers.%s[%d]: %w", setName, idx, err)
			}
			if !enabled {
				entryLogger.Info().Msg("skipping provider (enable=false)")
				continue
			}
			normalized, err := normalizeProviderEntry(entry)
			if err != nil {
				return nil, fmt.Errorf("providers.%s[%d]: %w", setName, idx, err)
			}
			entryLogger.Info().
				Str("vendor", normalized.Vendor).
				Str("base_url", normalized.BaseURL).
				Bool("sync_models", normalized.SyncModels).
				Bool("auto_enable_new_models", normalized.AutoEnableNewModels).
				Msg("including provider for bootstrap")
			result.sets[setName] = append(result.sets[setName], normalized)
		}
	}

	if len(result.sets) == 0 {
		return nil, fmt.Errorf("provider config %q has no valid provider entries", cleanPath)
	}

	return result, nil
}

type providerConfigDocument struct {
	Providers map[string][]providerConfigEntry `yaml:"providers"`
}

type providerConfigEntry struct {
	EnableRaw   string            `yaml:"enable"`
	Name        string            `yaml:"name"`
	Type        string            `yaml:"type"`
	Vendor      string            `yaml:"vendor"`
	Category    string            `yaml:"category"` // "llm" or "image"; defaults to "llm"
	URL         string            `yaml:"url"`
	BaseURL     string            `yaml:"base_url"`
	Endpoints   []EndpointConfig  `yaml:"endpoints"`
	APIKey      string            `yaml:"api_key"`
	Key         string            `yaml:"key"`
	Active      *bool             `yaml:"active"`
	Description string            `yaml:"description"`
	Metadata    map[string]string `yaml:"metadata"`
	AutoEnable  *bool             `yaml:"auto_enable_new_models"`
	SyncModels  *bool             `yaml:"sync_models"`
}

func normalizeProviderEntry(entry providerConfigEntry) (ProviderBootstrapEntry, error) {
	enabled, err := parseEnabled(entry.EnableRaw)
	if err != nil {
		return ProviderBootstrapEntry{}, err
	}

	vendor := firstNonEmpty(entry.Type, entry.Vendor)
	vendor = strings.TrimSpace(vendor)
	if vendor == "" {
		return ProviderBootstrapEntry{}, errors.New("provider type is required")
	}

	endpoints, err := parseEndpointConfigs(entry.Endpoints)
	if err != nil {
		return ProviderBootstrapEntry{}, err
	}
	baseURL := firstNonEmpty(entry.URL, entry.BaseURL)
	baseURL = strings.TrimSpace(os.ExpandEnv(baseURL))
	if len(endpoints) == 0 {
		if baseURL == "" {
			return ProviderBootstrapEntry{}, errors.New("provider url is required")
		}
		parsed, err := parseEndpointString(baseURL)
		if err != nil {
			return ProviderBootstrapEntry{}, fmt.Errorf("provider url: %w", err)
		}
		if len(parsed) == 0 {
			return ProviderBootstrapEntry{}, errors.New("provider url is required")
		}
		endpoints = parsed
	}

	baseURL = endpoints[0].URL

	name := strings.TrimSpace(entry.Name)
	if name == "" {
		name = fmt.Sprintf("%s Provider", strings.ToUpper(vendor))
	}
	name = os.ExpandEnv(name)

	apiKey := strings.TrimSpace(firstNonEmpty(entry.APIKey, entry.Key))
	if apiKey != "" {
		apiKey = os.ExpandEnv(apiKey)
	}

	active := true
	if entry.Active != nil {
		active = *entry.Active
	}

	autoEnable := true
	if entry.AutoEnable != nil {
		autoEnable = *entry.AutoEnable
	}

	syncModels := true
	if entry.SyncModels != nil {
		syncModels = *entry.SyncModels
	}

	metadata := cloneStringMap(entry.Metadata)
	if desc := strings.TrimSpace(os.ExpandEnv(entry.Description)); desc != "" {
		metadata = ensureStringMap(metadata)
		metadata["description"] = desc
	}
	metadata = ensureStringMap(metadata)
	metadata["auto_enable_new_models"] = strconv.FormatBool(autoEnable)
	if len(metadata) == 0 {
		metadata = nil
	}

	// Default category to "llm" if not specified.
	category := strings.TrimSpace(entry.Category)
	if category == "" {
		category = "llm"
	}

	return ProviderBootstrapEntry{
		Name:                name,
		Vendor:              vendor,
		Category:            category,
		BaseURL:             baseURL,
		Endpoints:           endpoints,
		APIKey:              apiKey,
		Active:              active && enabled,
		Metadata:            metadata,
		AutoEnableNewModels: autoEnable,
		SyncModels:          syncModels,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(os.ExpandEnv(v))
		if key == "" || val == "" {
			continue
		}
		out[key] = val
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func ensureStringMap(in map[string]string) map[string]string {
	if in == nil {
		return make(map[string]string)
	}
	return in
}

type EndpointConfig struct {
	URL      string `yaml:"url"`
	Weight   int    `yaml:"weight"`
	Priority int    `yaml:"priority"`
}

func parseEndpointConfigs(raw []EndpointConfig) ([]EndpointConfig, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	endpoints := make([]EndpointConfig, 0, len(raw))
	for idx, ep := range raw {
		normalized, err := normalizeEndpointURL(ep.URL)
		if err != nil {
			return nil, fmt.Errorf("endpoints[%d]: %w", idx, err)
		}
		if normalized == "" {
			continue
		}
		weight := ep.Weight
		if weight <= 0 {
			weight = 1
		}
		endpoints = append(endpoints, EndpointConfig{
			URL:      normalized,
			Weight:   weight,
			Priority: ep.Priority,
		})
	}
	return endpoints, nil
}

func parseEndpointString(input string) ([]EndpointConfig, error) {
	trimmed := strings.TrimSpace(os.ExpandEnv(input))
	if trimmed == "" {
		return nil, nil
	}

	if strings.HasPrefix(trimmed, "[") {
		var parsed []EndpointConfig
		if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
			return nil, fmt.Errorf("invalid endpoint array: %w", err)
		}
		return parseEndpointConfigs(parsed)
	}

	parts := strings.Split(trimmed, ",")
	endpoints := make([]EndpointConfig, 0, len(parts))
	for idx, part := range parts {
		normalized, err := normalizeEndpointURL(part)
		if err != nil {
			return nil, fmt.Errorf("endpoints[%d]: %w", idx, err)
		}
		if normalized == "" {
			continue
		}
		endpoints = append(endpoints, EndpointConfig{
			URL:    normalized,
			Weight: 1,
		})
	}
	return endpoints, nil
}

func normalizeEndpointURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(os.ExpandEnv(raw))
	if trimmed == "" {
		return "", nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint url %q: %w", trimmed, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("endpoint url must have http or https scheme: %s", trimmed)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("endpoint url must include host: %s", trimmed)
	}
	return strings.TrimSuffix(parsed.String(), "/"), nil
}

func parseEnabled(raw string) (bool, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return true, nil
	}

	resolved := expandWithDefault(value)
	resolved = strings.TrimSpace(resolved)
	if resolved == "" {
		return true, nil
	}

	parsed, err := strconv.ParseBool(resolved)
	if err != nil {
		return false, fmt.Errorf("enable: %w", err)
	}
	return parsed, nil
}

// expandWithDefault expands ${VAR} and ${VAR:-default} syntax using os envs.
func expandWithDefault(raw string) string {
	if !strings.Contains(raw, "${") {
		return os.ExpandEnv(raw)
	}
	start := strings.Index(raw, "${")
	end := strings.Index(raw[start:], "}")
	if start == -1 || end == -1 {
		return os.ExpandEnv(raw)
	}
	end = start + end
	expr := raw[start+2 : end]
	defaultVal := ""
	varName := expr
	if strings.Contains(expr, ":-") {
		parts := strings.SplitN(expr, ":-", 2)
		varName = parts[0]
		defaultVal = parts[1]
	}
	val := os.Getenv(varName)
	if val == "" {
		val = defaultVal
	}
	resolved := raw[:start] + val + raw[end+1:]
	return os.ExpandEnv(resolved)
}
