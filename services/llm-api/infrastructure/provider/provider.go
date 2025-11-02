package provider

import (
	"context"
	"fmt"
	"sync"
)

// ErrModelNotFound is returned when a model is not registered in the provider registry.
var ErrModelNotFound = fmt.Errorf("model not registered")

// ChatCompletionRequest models the OpenAI chat completion request.
type ChatCompletionRequest struct {
	Model       string                 `json:"model"`
	Messages    []map[string]any       `json:"messages"`
	Stream      bool                   `json:"stream,omitempty"`
	Temperature *float32               `json:"temperature,omitempty"`
	TopP        *float32               `json:"top_p,omitempty"`
	MaxTokens   *int                   `json:"max_tokens,omitempty"`
	Metadata    map[string]any         `json:"metadata,omitempty"`
	Extras      map[string]interface{} `json:"-"`
}

// ChatCompletionResponse is a lightweight passthrough representation.
type ChatCompletionResponse struct {
	Body       []byte
	StatusCode int
	Headers    map[string]string
}

// Route describes the model/provider association.
type Route struct {
	Model    ModelConfig
	Provider Provider
}

// Provider describes behaviour needed by the registry.
type Provider interface {
	Name() string
	Supports(model ModelConfig) bool
	ChatCompletions(ctx context.Context, req ChatCompletionRequest, principalHeaders map[string]string) (*ChatCompletionResponse, error)
	ChatCompletionsStream(ctx context.Context, req ChatCompletionRequest, principalHeaders map[string]string) (StreamResponse, error)
	ListModels(ctx context.Context) ([]RemoteModel, error)
	HealthCheck(ctx context.Context) error
}

// StreamResponse captures streamed responses from a provider.
type StreamResponse interface {
	Stream(ctx context.Context, cb func(data []byte) error) error
	Close() error
	Headers() map[string]string
	StatusCode() int
}

// RemoteModel describes a model advertised by a provider.
type RemoteModel struct {
	ID           string
	DisplayName  string
	Family       string
	Capabilities []string
}

// Registry routes model IDs to providers.
type Registry struct {
	mu              sync.RWMutex
	providers       map[string]Provider
	models          map[string]Route
	defaultProvider string
}

// NewRegistry constructs an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		models:    make(map[string]Route),
	}
}

// RegisterProvider adds a provider to the registry.
func (r *Registry) RegisterProvider(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

// RegisterModel maps a model ID to a provider route.
func (r *Registry) RegisterModel(cfg ModelConfig, provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[cfg.ID] = Route{Model: cfg, Provider: provider}
}

// SetDefaultProvider configures default provider for fallback.
func (r *Registry) SetDefaultProvider(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultProvider = name
}

// Resolve finds the route for a model ID.
func (r *Registry) Resolve(modelID string) (Route, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if rt, ok := r.models[modelID]; ok {
		return rt, nil
	}
	return Route{}, ErrModelNotFound
}

// Providers returns the currently configured providers.
func (r *Registry) Providers() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	return out
}

// Models returns the configured model configurations.
func (r *Registry) Models() []ModelConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ModelConfig, 0, len(r.models))
	for _, rt := range r.models {
		out = append(out, rt.Model)
	}
	return out
}

// ModelConfig defines configuration for routed models.
type ModelConfig struct {
	ID           string   `yaml:"id" json:"id"`
	ServedName   string   `yaml:"served_name" json:"served_name"`
	Capabilities []string `yaml:"capabilities" json:"capabilities"`
}
