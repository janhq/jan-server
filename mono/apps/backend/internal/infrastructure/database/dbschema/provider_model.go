package dbschema

import (
	"encoding/json"

	"gorm.io/datatypes"

	domainmodel "jan-server/mono/apps/backend/internal/domain/model"
	"jan-server/mono/apps/backend/internal/infrastructure/database/registry"
)

func init() {
	registry.RegisterSchemaForAutoMigrate(ProviderModel{})
}

type ProviderModel struct {
	BaseModel
	ProviderID              uint           `gorm:"not null;index;index:idx_provider_model_active,priority:1;index:idx_provider_model_catalog_active,priority:1;uniqueIndex:ux_provider_model_public_id,priority:1"`
	PublicID                string         `gorm:"size:64;not null;uniqueIndex"`
	Kind                    string         `gorm:"size:64;not null;index"`
	ModelCatalogID          *uint          `gorm:"index;index:idx_provider_model_catalog_active,priority:2"`
	ModelPublicID           string         `gorm:"size:128;not null;index;uniqueIndex:ux_provider_model_public_id,priority:2"`
	ModelDisplayName        string         `gorm:"size:255;not null;default:''"`
	ProviderOriginalModelID string         `gorm:"size:255;not null"`
	Category                string         `gorm:"size:128;not null;default:'';index"`
	CategoryOrderNumber     int            `gorm:"not null;default:0;index"`
	ModelOrderNumber        int            `gorm:"not null;default:0;index"`
	Pricing                 datatypes.JSON `gorm:"type:jsonb;not null"`
	TokenLimits             datatypes.JSON `gorm:"type:jsonb"`
	SupportsAutoMode        *bool          `gorm:"not null;default:false"`
	SupportsThinkingMode    *bool          `gorm:"not null;default:false"`
	DefaultConversationMode string         `gorm:"size:64;not null;default:'standard'"`
	ReasoningConfig         datatypes.JSON `gorm:"type:jsonb"`
	ProviderFlags           datatypes.JSON `gorm:"type:jsonb"`
	Active                  *bool          `gorm:"not null;default:true;index;index:idx_provider_model_active,priority:2;index:idx_provider_model_catalog_active,priority:3"`
	InstructModelID         *uint          `gorm:"index"` // Self-referencing FK to the instruct model variant (used when enable_thinking=false)
}

func NewSchemaProviderModel(m *domainmodel.ProviderModel) (*ProviderModel, error) {

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

	supportsAuto := m.SupportsAutoMode
	supportsThinking := m.SupportsThinkingMode
	defaultMode := m.DefaultConversationMode

	var reasoningConfig datatypes.JSON
	if m.ReasoningConfig != nil {
		data, err := json.Marshal(m.ReasoningConfig)
		if err != nil {
			return nil, err
		}
		reasoningConfig = datatypes.JSON(data)
	}

	var providerFlags datatypes.JSON
	if len(m.ProviderFlags) > 0 {
		data, err := json.Marshal(m.ProviderFlags)
		if err != nil {
			return nil, err
		}
		providerFlags = datatypes.JSON(data)
	}
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
		ModelDisplayName:        m.ModelDisplayName,
		ProviderOriginalModelID: m.ProviderOriginalModelID,
		Category:                m.Category,
		CategoryOrderNumber:     m.CategoryOrderNumber,
		ModelOrderNumber:        m.ModelOrderNumber,
		Pricing:                 datatypes.JSON(pricingJSON),
		TokenLimits:             tokenLimitsJSON,
		SupportsAutoMode:        &supportsAuto,
		SupportsThinkingMode:    &supportsThinking,
		DefaultConversationMode: defaultMode,
		ReasoningConfig:         reasoningConfig,
		ProviderFlags:           providerFlags,
		Active:                  &active,
		InstructModelID:         m.InstructModelID,
	}, nil
}

// EtoD converts a database provider model into its domain representation.
func (m *ProviderModel) EtoD() (*domainmodel.ProviderModel, error) {
	var pricing domainmodel.Pricing
	if len(m.Pricing) > 0 {
		if err := json.Unmarshal(m.Pricing, &pricing); err != nil {
			return nil, err
		}
	}

	var tokenLimits *domainmodel.TokenLimits
	if len(m.TokenLimits) > 0 {
		var limits domainmodel.TokenLimits
		if err := json.Unmarshal(m.TokenLimits, &limits); err != nil {
			return nil, err
		}
		tokenLimits = &limits
	}

	supportsAuto := false
	if m.SupportsAutoMode != nil {
		supportsAuto = *m.SupportsAutoMode
	}
	supportsThinking := false
	if m.SupportsThinkingMode != nil {
		supportsThinking = *m.SupportsThinkingMode
	}
	active := false
	if m.Active != nil {
		active = *m.Active
	}

	var reasoningConfig *domainmodel.ReasoningConfig
	if len(m.ReasoningConfig) > 0 {
		var config domainmodel.ReasoningConfig
		if err := json.Unmarshal(m.ReasoningConfig, &config); err != nil {
			return nil, err
		}
		reasoningConfig = &config
	}

	var providerFlags map[string]any
	if len(m.ProviderFlags) > 0 {
		if err := json.Unmarshal(m.ProviderFlags, &providerFlags); err != nil {
			return nil, err
		}
	}

	return &domainmodel.ProviderModel{
		ID:                      m.ID,
		ProviderID:              m.ProviderID,
		PublicID:                m.PublicID,
		Kind:                    domainmodel.ProviderKind(m.Kind),
		ModelCatalogID:          m.ModelCatalogID,
		ModelPublicID:           m.ModelPublicID,
		ModelDisplayName:        m.ModelDisplayName,
		ProviderOriginalModelID: m.ProviderOriginalModelID,
		Category:                m.Category,
		CategoryOrderNumber:     m.CategoryOrderNumber,
		ModelOrderNumber:        m.ModelOrderNumber,
		Pricing:                 pricing,
		TokenLimits:             tokenLimits,
		SupportsAutoMode:        supportsAuto,
		SupportsThinkingMode:    supportsThinking,
		DefaultConversationMode: m.DefaultConversationMode,
		ReasoningConfig:         reasoningConfig,
		ProviderFlags:           providerFlags,
		Active:                  active,
		InstructModelID:         m.InstructModelID,
		CreatedAt:               m.CreatedAt,
		UpdatedAt:               m.UpdatedAt,
	}, nil
}
