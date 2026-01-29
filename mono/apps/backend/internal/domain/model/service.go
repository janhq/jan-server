package model

import (
	"context"
	"errors"
)

var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrModelNotFound    = errors.New("model not found")
	ErrProviderExists   = errors.New("provider already exists")
	ErrModelExists      = errors.New("model already exists")
)

// Repository defines the interface for model/provider data operations.
type Repository interface {
	// Provider operations
	CreateProvider(ctx context.Context, provider *Provider) error
	GetProviderByID(ctx context.Context, id string) (*Provider, error)
	GetProviderByName(ctx context.Context, name string) (*Provider, error)
	UpdateProvider(ctx context.Context, provider *Provider) error
	DeleteProvider(ctx context.Context, id string) error
	ListProviders(ctx context.Context, enabledOnly bool) ([]*Provider, error)

	// Model operations
	CreateModel(ctx context.Context, model *Model) error
	GetModelByID(ctx context.Context, id string) (*Model, error)
	GetModelByName(ctx context.Context, providerID, name string) (*Model, error)
	UpdateModel(ctx context.Context, model *Model) error
	DeleteModel(ctx context.Context, id string) error
	ListModels(ctx context.Context, filter ListModelsFilter) ([]*Model, int64, error)
}

// Service handles model-related business logic.
type Service struct {
	repo Repository
}

// NewService creates a new model service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ============================================
// Provider operations
// ============================================

// CreateProvider creates a new provider.
func (s *Service) CreateProvider(ctx context.Context, req CreateProviderRequest) (*Provider, error) {
	// Check if provider already exists
	existing, _ := s.repo.GetProviderByName(ctx, req.Name)
	if existing != nil {
		return nil, ErrProviderExists
	}

	provider := &Provider{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		BaseURL:     req.BaseURL,
		APIKey:      req.APIKey, // Should be encrypted before storage
		IsEnabled:   true,
		Config:      req.Config,
	}

	if provider.DisplayName == "" {
		provider.DisplayName = provider.Name
	}

	if err := s.repo.CreateProvider(ctx, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

// GetProviderByID retrieves a provider by ID.
func (s *Service) GetProviderByID(ctx context.Context, id string) (*Provider, error) {
	provider, err := s.repo.GetProviderByID(ctx, id)
	if err != nil {
		return nil, ErrProviderNotFound
	}
	return provider, nil
}

// UpdateProvider updates a provider.
func (s *Service) UpdateProvider(ctx context.Context, id string, updates map[string]any) (*Provider, error) {
	provider, err := s.repo.GetProviderByID(ctx, id)
	if err != nil {
		return nil, ErrProviderNotFound
	}

	if name, ok := updates["name"].(string); ok && name != "" {
		provider.Name = name
	}
	if displayName, ok := updates["display_name"].(string); ok {
		provider.DisplayName = displayName
	}
	if baseURL, ok := updates["base_url"].(string); ok {
		provider.BaseURL = baseURL
	}
	if apiKey, ok := updates["api_key"].(string); ok && apiKey != "" {
		provider.APIKey = apiKey
	}
	if isEnabled, ok := updates["is_enabled"].(bool); ok {
		provider.IsEnabled = isEnabled
	}
	if config, ok := updates["config"].(map[string]any); ok {
		provider.Config = config
	}

	if err := s.repo.UpdateProvider(ctx, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

// DeleteProvider deletes a provider.
func (s *Service) DeleteProvider(ctx context.Context, id string) error {
	_, err := s.repo.GetProviderByID(ctx, id)
	if err != nil {
		return ErrProviderNotFound
	}
	return s.repo.DeleteProvider(ctx, id)
}

// ListProviders lists all providers.
func (s *Service) ListProviders(ctx context.Context, enabledOnly bool) ([]*Provider, error) {
	return s.repo.ListProviders(ctx, enabledOnly)
}

// ============================================
// Model operations
// ============================================

// CreateModel creates a new model.
func (s *Service) CreateModel(ctx context.Context, req CreateModelRequest) (*Model, error) {
	// Verify provider exists
	_, err := s.repo.GetProviderByID(ctx, req.ProviderID)
	if err != nil {
		return nil, ErrProviderNotFound
	}

	// Check if model already exists
	existing, _ := s.repo.GetModelByName(ctx, req.ProviderID, req.Name)
	if existing != nil {
		return nil, ErrModelExists
	}

	model := &Model{
		ProviderID:    req.ProviderID,
		Name:          req.Name,
		DisplayName:   req.DisplayName,
		Description:   req.Description,
		ContextWindow: req.ContextWindow,
		MaxTokens:     req.MaxTokens,
		IsEnabled:     true,
		Capabilities:  req.Capabilities,
		Pricing:       req.Pricing,
	}

	if model.DisplayName == "" {
		model.DisplayName = model.Name
	}
	if model.ContextWindow == 0 {
		model.ContextWindow = 4096
	}
	if model.MaxTokens == 0 {
		model.MaxTokens = 4096
	}

	if err := s.repo.CreateModel(ctx, model); err != nil {
		return nil, err
	}

	return model, nil
}

// GetModelByID retrieves a model by ID.
func (s *Service) GetModelByID(ctx context.Context, id string) (*Model, error) {
	model, err := s.repo.GetModelByID(ctx, id)
	if err != nil {
		return nil, ErrModelNotFound
	}
	return model, nil
}

// UpdateModel updates a model.
func (s *Service) UpdateModel(ctx context.Context, id string, updates map[string]any) (*Model, error) {
	model, err := s.repo.GetModelByID(ctx, id)
	if err != nil {
		return nil, ErrModelNotFound
	}

	if name, ok := updates["name"].(string); ok && name != "" {
		model.Name = name
	}
	if displayName, ok := updates["display_name"].(string); ok {
		model.DisplayName = displayName
	}
	if description, ok := updates["description"].(string); ok {
		model.Description = description
	}
	if contextWindow, ok := updates["context_window"].(int); ok && contextWindow > 0 {
		model.ContextWindow = contextWindow
	}
	if maxTokens, ok := updates["max_tokens"].(int); ok && maxTokens > 0 {
		model.MaxTokens = maxTokens
	}
	if isEnabled, ok := updates["is_enabled"].(bool); ok {
		model.IsEnabled = isEnabled
	}

	if err := s.repo.UpdateModel(ctx, model); err != nil {
		return nil, err
	}

	return model, nil
}

// DeleteModel deletes a model.
func (s *Service) DeleteModel(ctx context.Context, id string) error {
	_, err := s.repo.GetModelByID(ctx, id)
	if err != nil {
		return ErrModelNotFound
	}
	return s.repo.DeleteModel(ctx, id)
}

// ListModels lists models with optional filters.
func (s *Service) ListModels(ctx context.Context, filter ListModelsFilter) ([]*Model, int64, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}

	return s.repo.ListModels(ctx, filter)
}
