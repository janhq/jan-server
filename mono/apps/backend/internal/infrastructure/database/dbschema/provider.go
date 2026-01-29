package dbschema

import (
	"encoding/json"
	"time"

	domainmodel "jan-server/mono/apps/backend/internal/domain/model"
	"jan-server/mono/apps/backend/internal/infrastructure/database/registry"
	"jan-server/mono/apps/backend/internal/infrastructure/logger"

	"gorm.io/datatypes"
)

func init() {
	registry.RegisterSchemaForAutoMigrate(Provider{})
}

type Provider struct {
	BaseModel
	PublicID        string         `gorm:"size:64;not null;uniqueIndex"`
	DisplayName     string         `gorm:"size:255;not null"`
	Kind            string         `gorm:"size:64;not null;index;index:idx_provider_active_kind,priority:2"`
	Category        string         `gorm:"size:20;not null;default:'llm';index"` // "llm" or "image"
	BaseURL         string         `gorm:"size:512"`
	Endpoints       datatypes.JSON `gorm:"type:jsonb"`
	EncryptedAPIKey string         `gorm:"type:text"`
	APIKeyHint      *string        `gorm:"size:128"`
	IsModerated     *bool          `gorm:"not null;default:false;index"`
	Active          *bool          `gorm:"not null;default:true;index;index:idx_provider_active_kind,priority:1"`
	DefaultImageGenerate *bool     `gorm:"column:default_provider_image_generate;not null;default:false"`
	DefaultImageEdit     *bool     `gorm:"column:default_provider_image_edit;not null;default:false"`
	Metadata        datatypes.JSON `gorm:"type:jsonb"`
	LastSyncedAt    *time.Time     `gorm:"index"`
}

func NewSchemaProvider(p *domainmodel.Provider) *Provider {
	var metadataJSON datatypes.JSON
	if len(p.Metadata) > 0 {
		if data, err := json.Marshal(p.Metadata); err == nil {
			metadataJSON = datatypes.JSON(data)
		}
	}

	var endpointsJSON datatypes.JSON
	if len(p.Endpoints) > 0 {
		data, err := json.Marshal(p.Endpoints)
		if err != nil {
			log := logger.GetLogger()
			log.Error().Err(err).Str("provider_id", p.PublicID).Msg("failed to marshal endpoints to JSON")
		} else {
			endpointsJSON = datatypes.JSON(data)
		}
	}

	// Default category to LLM for backward compatibility
	category := string(p.Category)
	if category == "" {
		category = string(domainmodel.ProviderCategoryLLM)
	}

	isModerated := p.IsModerated
	active := p.Active
	defaultImageGenerate := p.DefaultImageGenerate
	defaultImageEdit := p.DefaultImageEdit
	return &Provider{
		BaseModel: BaseModel{
			ID:        p.ID,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
		PublicID:        p.PublicID,
		DisplayName:     p.DisplayName,
		Kind:            string(p.Kind),
		Category:        category,
		BaseURL:         p.BaseURL,
		Endpoints:       endpointsJSON,
		EncryptedAPIKey: p.EncryptedAPIKey,
		APIKeyHint:      p.APIKeyHint,
		IsModerated:     &isModerated,
		Active:          &active,
		DefaultImageGenerate: &defaultImageGenerate,
		DefaultImageEdit:     &defaultImageEdit,
		Metadata:        metadataJSON,
		LastSyncedAt:    p.LastSyncedAt,
	}
}

// EtoD converts a database provider into its domain representation.
func (p *Provider) EtoD() *domainmodel.Provider {
	var metadata map[string]string
	if len(p.Metadata) > 0 {
		err := json.Unmarshal(p.Metadata, &metadata)
		if err != nil {
			log := logger.GetLogger()
			log.Error().Msgf("failed to unmarshal provider metadata for provider ID %d: %v", p.ID, err)
		}
	}

	var endpoints domainmodel.EndpointList
	if len(p.Endpoints) > 0 {
		if err := json.Unmarshal(p.Endpoints, &endpoints); err != nil {
			log := logger.GetLogger()
			log.Error().Msgf("failed to unmarshal provider endpoints for provider ID %d: %v", p.ID, err)
		}
	}
	if len(endpoints) == 0 && p.BaseURL != "" {
		endpoints = domainmodel.EndpointList{{URL: p.BaseURL, Weight: 1, Healthy: true}}
	}

	isModerated := false
	if p.IsModerated != nil {
		isModerated = *p.IsModerated
	}
	active := false
	if p.Active != nil {
		active = *p.Active
	}
	defaultImageGenerate := false
	if p.DefaultImageGenerate != nil {
		defaultImageGenerate = *p.DefaultImageGenerate
	}
	defaultImageEdit := false
	if p.DefaultImageEdit != nil {
		defaultImageEdit = *p.DefaultImageEdit
	}

	// Default category to LLM for backward compatibility
	category := domainmodel.ProviderCategoryLLM
	if p.Category != "" {
		category = domainmodel.ProviderCategory(p.Category)
	}

	return &domainmodel.Provider{
		ID:              p.ID,
		PublicID:        p.PublicID,
		DisplayName:     p.DisplayName,
		Kind:            domainmodel.ProviderKind(p.Kind),
		Category:        category,
		BaseURL:         p.BaseURL,
		Endpoints:       endpoints,
		EncryptedAPIKey: p.EncryptedAPIKey,
		APIKeyHint:      p.APIKeyHint,
		IsModerated:     isModerated,
		Active:          active,
		DefaultImageGenerate: defaultImageGenerate,
		DefaultImageEdit:     defaultImageEdit,
		Metadata:        metadata,
		LastSyncedAt:    p.LastSyncedAt,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}
