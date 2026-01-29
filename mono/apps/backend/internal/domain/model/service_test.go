package model

import (
	"context"
	"testing"
)

// MockModelRepository is a mock implementation for testing
type MockModelRepository struct {
	providers map[string]*Provider
	models    map[string]*Model
}

func NewMockModelRepository() *MockModelRepository {
	return &MockModelRepository{
		providers: make(map[string]*Provider),
		models:    make(map[string]*Model),
	}
}

func (m *MockModelRepository) CreateProvider(ctx context.Context, p *Provider) error {
	m.providers[p.ID] = p
	return nil
}

func (m *MockModelRepository) FindProviderByID(ctx context.Context, id string) (*Provider, error) {
	p, ok := m.providers[id]
	if !ok {
		return nil, ErrProviderNotFound
	}
	return p, nil
}

func (m *MockModelRepository) ListProviders(ctx context.Context, filter ProviderFilter) ([]*Provider, int64, error) {
	var result []*Provider
	for _, p := range m.providers {
		if filter.IsEnabled != nil && p.IsEnabled != *filter.IsEnabled {
			continue
		}
		result = append(result, p)
	}
	return result, int64(len(result)), nil
}

func (m *MockModelRepository) UpdateProvider(ctx context.Context, p *Provider) error {
	if _, ok := m.providers[p.ID]; !ok {
		return ErrProviderNotFound
	}
	m.providers[p.ID] = p
	return nil
}

func (m *MockModelRepository) DeleteProvider(ctx context.Context, id string) error {
	delete(m.providers, id)
	return nil
}

func (m *MockModelRepository) CreateModel(ctx context.Context, model *Model) error {
	m.models[model.ID] = model
	return nil
}

func (m *MockModelRepository) FindModelByID(ctx context.Context, id string) (*Model, error) {
	model, ok := m.models[id]
	if !ok {
		return nil, ErrModelNotFound
	}
	return model, nil
}

func (m *MockModelRepository) FindModelByName(ctx context.Context, name string) (*Model, error) {
	for _, model := range m.models {
		if model.Name == name {
			return model, nil
		}
	}
	return nil, ErrModelNotFound
}

func (m *MockModelRepository) ListModels(ctx context.Context, filter ModelFilter) ([]*Model, int64, error) {
	var result []*Model
	for _, model := range m.models {
		if filter.ProviderID != "" && model.ProviderID != filter.ProviderID {
			continue
		}
		if filter.IsEnabled != nil && model.IsEnabled != *filter.IsEnabled {
			continue
		}
		result = append(result, model)
	}
	return result, int64(len(result)), nil
}

func (m *MockModelRepository) UpdateModel(ctx context.Context, model *Model) error {
	if _, ok := m.models[model.ID]; !ok {
		return ErrModelNotFound
	}
	m.models[model.ID] = model
	return nil
}

func (m *MockModelRepository) DeleteModel(ctx context.Context, id string) error {
	delete(m.models, id)
	return nil
}

func TestService_CreateProvider(t *testing.T) {
	repo := NewMockModelRepository()
	svc := NewService(repo)

	tests := []struct {
		name    string
		req     CreateProviderRequest
		wantErr bool
	}{
		{
			name: "create OpenAI provider",
			req: CreateProviderRequest{
				Name:    "OpenAI",
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "sk-test-key",
			},
			wantErr: false,
		},
		{
			name: "create Anthropic provider",
			req: CreateProviderRequest{
				Name:    "Anthropic",
				Type:    "anthropic",
				BaseURL: "https://api.anthropic.com",
				APIKey:  "sk-ant-test",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: CreateProviderRequest{
				Type:    "openai",
				BaseURL: "https://api.openai.com/v1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := svc.CreateProvider(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.CreateProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if provider.Name != tt.req.Name {
					t.Errorf("Expected name %s, got %s", tt.req.Name, provider.Name)
				}
			}
		})
	}
}

func TestService_CreateModel(t *testing.T) {
	repo := NewMockModelRepository()
	svc := NewService(repo)

	// Create a provider first
	provider, err := svc.CreateProvider(context.Background(), CreateProviderRequest{
		Name:    "OpenAI",
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "sk-test",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name    string
		req     CreateModelRequest
		wantErr bool
	}{
		{
			name: "create GPT-4 model",
			req: CreateModelRequest{
				ProviderID:  provider.ID,
				Name:        "gpt-4",
				DisplayName: "GPT-4",
				MaxTokens:   8192,
			},
			wantErr: false,
		},
		{
			name: "create GPT-3.5 model",
			req: CreateModelRequest{
				ProviderID:  provider.ID,
				Name:        "gpt-3.5-turbo",
				DisplayName: "GPT-3.5 Turbo",
				MaxTokens:   4096,
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			req: CreateModelRequest{
				Name:        "test-model",
				DisplayName: "Test Model",
			},
			wantErr: true,
		},
		{
			name: "invalid provider",
			req: CreateModelRequest{
				ProviderID:  "invalid-provider",
				Name:        "test-model",
				DisplayName: "Test Model",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := svc.CreateModel(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.CreateModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if model.Name != tt.req.Name {
					t.Errorf("Expected name %s, got %s", tt.req.Name, model.Name)
				}
			}
		})
	}
}

func TestService_ListModels(t *testing.T) {
	repo := NewMockModelRepository()
	svc := NewService(repo)

	// Create provider and models
	provider, _ := svc.CreateProvider(context.Background(), CreateProviderRequest{
		Name:    "OpenAI",
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "sk-test",
	})

	for _, name := range []string{"gpt-4", "gpt-3.5-turbo", "gpt-4-turbo"} {
		svc.CreateModel(context.Background(), CreateModelRequest{
			ProviderID: provider.ID,
			Name:       name,
		})
	}

	// Test listing
	models, total, err := svc.ListModels(context.Background(), ModelFilter{
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Service.ListModels() error = %v", err)
	}

	if total != 3 {
		t.Errorf("Expected total 3, got %d", total)
	}

	if len(models) != 3 {
		t.Errorf("Expected 3 models, got %d", len(models))
	}
}

func TestService_GetModelByID(t *testing.T) {
	repo := NewMockModelRepository()
	svc := NewService(repo)

	// Create provider and model
	provider, _ := svc.CreateProvider(context.Background(), CreateProviderRequest{
		Name:    "OpenAI",
		Type:    "openai",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "sk-test",
	})

	model, _ := svc.CreateModel(context.Background(), CreateModelRequest{
		ProviderID: provider.ID,
		Name:       "gpt-4",
	})

	tests := []struct {
		name    string
		modelID string
		wantErr bool
	}{
		{
			name:    "get existing model by ID",
			modelID: model.ID,
			wantErr: false,
		},
		{
			name:    "get model by name",
			modelID: "gpt-4",
			wantErr: false,
		},
		{
			name:    "non-existent model",
			modelID: "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.GetModelByID(context.Background(), tt.modelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.GetModelByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result.Name != "gpt-4" {
				t.Errorf("Expected model name gpt-4, got %s", result.Name)
			}
		})
	}
}
