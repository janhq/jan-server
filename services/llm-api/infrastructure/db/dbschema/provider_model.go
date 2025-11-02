package dbschema

import (
	"encoding/json"

	"jan-server/services/llm-api/domain/model"

	"gorm.io/datatypes"
)

type ProviderModel struct {
	BaseModel
	ProviderID              uint           `gorm:"not null;index;index:idx_provider_model_active,priority:1;index:idx_provider_model_catalog_active,priority:1;uniqueIndex:ux_provider_model_public_id,priority:1"`
	PublicID                string         `gorm:"size:64;not null;uniqueIndex"`
	Kind                    string         `gorm:"size:64;not null;index"`
	ModelCatalogID          *uint          `gorm:"index;index:idx_provider_model_catalog_active,priority:2"`
	ModelPublicID           string         `gorm:"size:128;not null;index;uniqueIndex:ux_provider_model_public_id,priority:2"`
	ProviderOriginalModelID string         `gorm:"size:255;not null"`
	DisplayName             string         `gorm:"size:255;not null"`
	Pricing                 datatypes.JSON `gorm:"type:jsonb;not null"`
	TokenLimits             datatypes.JSON `gorm:"type:jsonb"`
	Family                  *string        `gorm:"size:128"`
	SupportsImages          *bool          `gorm:"not null;default:false"`
	SupportsEmbeddings      *bool          `gorm:"not null;default:false"`
	SupportsReasoning       *bool          `gorm:"not null;default:false"`
	SupportsAudio           *bool          `gorm:"not null;default:false"`
	SupportsVideo           *bool          `gorm:"not null;default:false"`
	Active                  *bool          `gorm:"not null;default:true;index;index:idx_provider_model_active,priority:2;index:idx_provider_model_catalog_active,priority:3"`
}

func (ProviderModel) TableName() string {
	return "provider_models"
}

func NewSchemaProviderModel(m *model.ProviderModel) (*ProviderModel, error) {
	pricingJSON, err := json.Marshal(m.Pricing)
	if err != nil {
		return nil, err
	}

	var tokenLimitsJSON datatypes.JSON
	if m.TokenLimits != nil {
		data, err := json.Marshal(m.TokenLimits)
		if err != nil {
			return nil, err
		}
		tokenLimitsJSON = datatypes.JSON(data)
	}

	supportsImages := m.SupportsImages
	supportsEmbeddings := m.SupportsEmbeddings
	supportsReasoning := m.SupportsReasoning
	supportsAudio := m.SupportsAudio
	supportsVideo := m.SupportsVideo
	active := m.Active

	return &ProviderModel{
		BaseModel: BaseModel{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		ProviderID:              m.ProviderID,
		PublicID:                m.PublicID,
		Kind:                    string(m.Kind),
		ModelCatalogID:          m.ModelCatalogID,
		ModelPublicID:           m.ModelPublicID,
		ProviderOriginalModelID: m.ProviderOriginalModelID,
		DisplayName:             m.DisplayName,
		Pricing:                 datatypes.JSON(pricingJSON),
		TokenLimits:             tokenLimitsJSON,
		Family:                  m.Family,
		SupportsImages:          &supportsImages,
		SupportsEmbeddings:      &supportsEmbeddings,
		SupportsReasoning:       &supportsReasoning,
		SupportsAudio:           &supportsAudio,
		SupportsVideo:           &supportsVideo,
		Active:                  &active,
	}, nil
}

// EtoD converts a database provider model into its domain representation
func (m *ProviderModel) EtoD() (*model.ProviderModel, error) {
	var pricing model.Pricing
	if len(m.Pricing) > 0 {
		if err := json.Unmarshal(m.Pricing, &pricing); err != nil {
			return nil, err
		}
	}

	var tokenLimits *model.TokenLimits
	if len(m.TokenLimits) > 0 {
		var limits model.TokenLimits
		if err := json.Unmarshal(m.TokenLimits, &limits); err != nil {
			return nil, err
		}
		tokenLimits = &limits
	}

	supportsImages := false
	if m.SupportsImages != nil {
		supportsImages = *m.SupportsImages
	}
	supportsEmbeddings := false
	if m.SupportsEmbeddings != nil {
		supportsEmbeddings = *m.SupportsEmbeddings
	}
	supportsReasoning := false
	if m.SupportsReasoning != nil {
		supportsReasoning = *m.SupportsReasoning
	}
	supportsAudio := false
	if m.SupportsAudio != nil {
		supportsAudio = *m.SupportsAudio
	}
	supportsVideo := false
	if m.SupportsVideo != nil {
		supportsVideo = *m.SupportsVideo
	}
	active := false
	if m.Active != nil {
		active = *m.Active
	}

	return &model.ProviderModel{
		ID:                      m.ID,
		ProviderID:              m.ProviderID,
		PublicID:                m.PublicID,
		Kind:                    model.ProviderKind(m.Kind),
		ModelCatalogID:          m.ModelCatalogID,
		ModelPublicID:           m.ModelPublicID,
		ProviderOriginalModelID: m.ProviderOriginalModelID,
		DisplayName:             m.DisplayName,
		Pricing:                 pricing,
		TokenLimits:             tokenLimits,
		Family:                  m.Family,
		SupportsImages:          supportsImages,
		SupportsEmbeddings:      supportsEmbeddings,
		SupportsReasoning:       supportsReasoning,
		SupportsAudio:           supportsAudio,
		SupportsVideo:           supportsVideo,
		Active:                  active,
		CreatedAt:               m.CreatedAt,
		UpdatedAt:               m.UpdatedAt,
	}, nil
}
