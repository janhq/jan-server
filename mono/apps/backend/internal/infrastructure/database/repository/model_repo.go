package repository

import (
	"context"
	"encoding/json"
	"errors"

	"jan-server/mono/apps/backend/internal/domain/model"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"

	"gorm.io/gorm"
)

// ModelRepository implements model.Repository using GORM.
type ModelRepository struct {
	db *gorm.DB
}

// NewModelRepository creates a new model repository.
func NewModelRepository(db *gorm.DB) *ModelRepository {
	return &ModelRepository{db: db}
}

var _ model.Repository = (*ModelRepository)(nil)

// ============================================
// Provider operations
// ============================================

func (r *ModelRepository) CreateProvider(ctx context.Context, provider *model.Provider) error {
	schema := toProviderSchema(provider)
	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		return err
	}
	provider.ID = schema.ID
	provider.CreatedAt = schema.CreatedAt
	provider.UpdatedAt = schema.UpdatedAt
	return nil
}

func (r *ModelRepository) GetProviderByID(ctx context.Context, id string) (*model.Provider, error) {
	var schema dbschema.Provider
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrProviderNotFound
		}
		return nil, err
	}
	return toProviderDomain(&schema), nil
}

func (r *ModelRepository) GetProviderByName(ctx context.Context, name string) (*model.Provider, error) {
	var schema dbschema.Provider
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrProviderNotFound
		}
		return nil, err
	}
	return toProviderDomain(&schema), nil
}

func (r *ModelRepository) UpdateProvider(ctx context.Context, provider *model.Provider) error {
	schema := toProviderSchema(provider)
	return r.db.WithContext(ctx).Model(&dbschema.Provider{}).Where("id = ?", provider.ID).Updates(schema).Error
}

func (r *ModelRepository) DeleteProvider(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&dbschema.Provider{}, "id = ?", id).Error
}

func (r *ModelRepository) ListProviders(ctx context.Context, enabledOnly bool) ([]*model.Provider, error) {
	var schemas []dbschema.Provider
	query := r.db.WithContext(ctx)

	if enabledOnly {
		query = query.Where("is_enabled = ?", true)
	}

	if err := query.Order("name ASC").Find(&schemas).Error; err != nil {
		return nil, err
	}

	providers := make([]*model.Provider, len(schemas))
	for i, s := range schemas {
		providers[i] = toProviderDomain(&s)
	}

	return providers, nil
}

// ============================================
// Model operations
// ============================================

func (r *ModelRepository) CreateModel(ctx context.Context, m *model.Model) error {
	schema := toModelSchema(m)
	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		return err
	}
	m.ID = schema.ID
	m.CreatedAt = schema.CreatedAt
	m.UpdatedAt = schema.UpdatedAt
	return nil
}

func (r *ModelRepository) GetModelByID(ctx context.Context, id string) (*model.Model, error) {
	var schema dbschema.Model
	if err := r.db.WithContext(ctx).Preload("Provider").Where("id = ?", id).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrModelNotFound
		}
		return nil, err
	}
	return toModelDomain(&schema), nil
}

func (r *ModelRepository) GetModelByName(ctx context.Context, providerID, name string) (*model.Model, error) {
	var schema dbschema.Model
	if err := r.db.WithContext(ctx).Where("provider_id = ? AND name = ?", providerID, name).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrModelNotFound
		}
		return nil, err
	}
	return toModelDomain(&schema), nil
}

func (r *ModelRepository) UpdateModel(ctx context.Context, m *model.Model) error {
	schema := toModelSchema(m)
	return r.db.WithContext(ctx).Model(&dbschema.Model{}).Where("id = ?", m.ID).Updates(schema).Error
}

func (r *ModelRepository) DeleteModel(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&dbschema.Model{}, "id = ?", id).Error
}

func (r *ModelRepository) ListModels(ctx context.Context, filter model.ListModelsFilter) ([]*model.Model, int64, error) {
	var schemas []dbschema.Model
	var total int64

	query := r.db.WithContext(ctx).Model(&dbschema.Model{}).Preload("Provider")

	if filter.ProviderID != "" {
		query = query.Where("provider_id = ?", filter.ProviderID)
	}
	if filter.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *filter.IsEnabled)
	}
	if filter.Search != "" {
		query = query.Where("name ILIKE ? OR display_name ILIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("name ASC").Offset(filter.Offset).Limit(filter.Limit).Find(&schemas).Error; err != nil {
		return nil, 0, err
	}

	models := make([]*model.Model, len(schemas))
	for i, s := range schemas {
		models[i] = toModelDomain(&s)
	}

	return models, total, nil
}

// ============================================
// Conversion helpers
// ============================================

func toProviderSchema(p *model.Provider) *dbschema.Provider {
	configJSON, _ := json.Marshal(p.Config)
	isEnabled := p.IsEnabled
	return &dbschema.Provider{
		ID:          p.ID,
		Name:        p.Name,
		DisplayName: p.DisplayName,
		BaseURL:     p.BaseURL,
		APIKey:      p.APIKey,
		IsEnabled:   &isEnabled,
		Config:      configJSON,
	}
}

func toProviderDomain(s *dbschema.Provider) *model.Provider {
	var config map[string]any
	_ = json.Unmarshal(s.Config, &config)

	isEnabled := true
	if s.IsEnabled != nil {
		isEnabled = *s.IsEnabled
	}

	return &model.Provider{
		ID:          s.ID,
		Name:        s.Name,
		DisplayName: s.DisplayName,
		BaseURL:     s.BaseURL,
		APIKey:      s.APIKey,
		IsEnabled:   isEnabled,
		Config:      config,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func toModelSchema(m *model.Model) *dbschema.Model {
	capabilitiesJSON, _ := json.Marshal(m.Capabilities)
	pricingJSON, _ := json.Marshal(m.Pricing)
	isEnabled := m.IsEnabled

	return &dbschema.Model{
		ID:            m.ID,
		ProviderID:    m.ProviderID,
		Name:          m.Name,
		DisplayName:   m.DisplayName,
		Description:   m.Description,
		ContextWindow: m.ContextWindow,
		MaxTokens:     m.MaxTokens,
		IsEnabled:     &isEnabled,
		Capabilities:  capabilitiesJSON,
		Pricing:       pricingJSON,
	}
}

func toModelDomain(s *dbschema.Model) *model.Model {
	var capabilities model.ModelCapabilities
	var pricing model.ModelPricing
	_ = json.Unmarshal(s.Capabilities, &capabilities)
	_ = json.Unmarshal(s.Pricing, &pricing)

	isEnabled := true
	if s.IsEnabled != nil {
		isEnabled = *s.IsEnabled
	}

	m := &model.Model{
		ID:            s.ID,
		ProviderID:    s.ProviderID,
		Name:          s.Name,
		DisplayName:   s.DisplayName,
		Description:   s.Description,
		ContextWindow: s.ContextWindow,
		MaxTokens:     s.MaxTokens,
		IsEnabled:     isEnabled,
		Capabilities:  capabilities,
		Pricing:       pricing,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}

	if s.Provider.ID != "" {
		provider := toProviderDomain(&s.Provider)
		m.Provider = provider
	}

	return m
}
