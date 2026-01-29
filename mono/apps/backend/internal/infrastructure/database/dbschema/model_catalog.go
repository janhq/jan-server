package dbschema

import (
	"encoding/json"

	"gorm.io/datatypes"

	domainmodel "jan-server/mono/apps/backend/internal/domain/model"
	"jan-server/mono/apps/backend/internal/infrastructure/database/registry"
)

func init() {
	registry.RegisterSchemaForAutoMigrate(ModelCatalog{})
}

type ModelCatalog struct {
	BaseModel
	PublicID            string         `gorm:"size:64;not null;uniqueIndex"`
	ModelDisplayName    string         `gorm:"size:255;not null;default:''"`
	Description         *string        `gorm:"type:text"`
	SupportedParameters datatypes.JSON `gorm:"type:jsonb;not null"`
	Architecture        datatypes.JSON `gorm:"type:jsonb;not null"`
	Tags                datatypes.JSON `gorm:"type:jsonb"`
	Notes               *string        `gorm:"type:text"`
	ContextLength       *int           `gorm:"column:context_length"`
	IsModerated         *bool          `gorm:"index"`
	Active              *bool          `gorm:"default:true;index;index:idx_model_catalog_status_active,priority:2"`
	Status              string         `gorm:"size:32;not null;default:'init';index;index:idx_model_catalog_status_active,priority:1"`
	Extras              datatypes.JSON `gorm:"type:jsonb"`
	Experimental        *bool          `gorm:"not null;default:false;index"`
	RequiresFeatureFlag *string        `gorm:"size:50;index"`
	// Capabilities (moved from provider_model)
	SupportsImages     *bool  `gorm:"not null;default:false;index"`
	SupportsEmbeddings *bool  `gorm:"not null;default:false;index"`
	SupportsReasoning  *bool  `gorm:"not null;default:false;index"`
	SupportsInstruct   *bool  `gorm:"not null;default:false;index"` // Model can use an instruct backup
	SupportsAudio      *bool  `gorm:"not null;default:false;index"`
	SupportsVideo      *bool  `gorm:"not null;default:false;index"`
	SupportsTools      *bool  `gorm:"not null;default:true;index"`
	SupportsBrowser    *bool  `gorm:"not null;default:false;index"` // Model supports browser/web browsing
	Family             string `gorm:"size:128;index"`
}

func NewSchemaModelCatalog(m *domainmodel.ModelCatalog) (*ModelCatalog, error) {
	status := string(m.Status)
	if status == "" {
		status = string(domainmodel.ModelCatalogStatusInit)
	}

	supportedParametersJSON, err := json.Marshal(m.SupportedParameters)
	if err != nil {
		return nil, err
	}

	architectureJSON, err := json.Marshal(m.Architecture)
	if err != nil {
		return nil, err
	}

	var tagsJSON datatypes.JSON
	if len(m.Tags) > 0 {
		data, err := json.Marshal(m.Tags)
		if err != nil {
			return nil, err
		}
		tagsJSON = datatypes.JSON(data)
	}

	var extrasJSON datatypes.JSON
	if len(m.Extras) > 0 {
		data, err := json.Marshal(m.Extras)
		if err != nil {
			return nil, err
		}
		extrasJSON = datatypes.JSON(data)
	}

	supportsImages := m.SupportsImages
	supportsEmbeddings := m.SupportsEmbeddings
	supportsReasoning := m.SupportsReasoning
	supportsInstruct := m.SupportsInstruct
	supportsAudio := m.SupportsAudio
	supportsVideo := m.SupportsVideo
	supportsTools := m.SupportsTools
	supportsBrowser := m.SupportsBrowser
	experimental := m.Experimental

	return &ModelCatalog{
		BaseModel: BaseModel{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		PublicID:            m.PublicID,
		ModelDisplayName:    m.ModelDisplayName,
		Description:         m.Description,
		SupportedParameters: datatypes.JSON(supportedParametersJSON),
		Architecture:        datatypes.JSON(architectureJSON),
		Tags:                tagsJSON,
		Notes:               m.Notes,
		ContextLength:       m.ContextLength,
		IsModerated:         m.IsModerated,
		Active:              m.Active,
		Status:              status,
		Extras:              extrasJSON,
		Experimental:        &experimental,
		RequiresFeatureFlag: m.RequiresFeatureFlag,
		SupportsImages:      &supportsImages,
		SupportsEmbeddings:  &supportsEmbeddings,
		SupportsReasoning:   &supportsReasoning,
		SupportsInstruct:    &supportsInstruct,
		SupportsAudio:       &supportsAudio,
		SupportsVideo:       &supportsVideo,
		SupportsTools:       &supportsTools,
		SupportsBrowser:     &supportsBrowser,
		Family:              m.Family,
	}, nil
}

func (m *ModelCatalog) EtoD() (*domainmodel.ModelCatalog, error) {

	var supportedParameters domainmodel.SupportedParameters
	if len(m.SupportedParameters) > 0 {
		if err := json.Unmarshal(m.SupportedParameters, &supportedParameters); err != nil {
			return nil, err
		}
	}

	var architecture domainmodel.Architecture
	if len(m.Architecture) > 0 {
		if err := json.Unmarshal(m.Architecture, &architecture); err != nil {
			return nil, err
		}
	}

	var tags []string
	if len(m.Tags) > 0 {
		if err := json.Unmarshal(m.Tags, &tags); err != nil {
			return nil, err
		}
	}

	var extras map[string]any
	if len(m.Extras) > 0 {
		if err := json.Unmarshal(m.Extras, &extras); err != nil {
			return nil, err
		}
	}

	experimental := false
	if m.Experimental != nil {
		experimental = *m.Experimental
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
	supportsInstruct := false
	if m.SupportsInstruct != nil {
		supportsInstruct = *m.SupportsInstruct
	}
	supportsAudio := false
	if m.SupportsAudio != nil {
		supportsAudio = *m.SupportsAudio
	}
	supportsVideo := false
	if m.SupportsVideo != nil {
		supportsVideo = *m.SupportsVideo
	}
	supportsTools := true
	if m.SupportsTools != nil {
		supportsTools = *m.SupportsTools
	}
	supportsBrowser := false
	if m.SupportsBrowser != nil {
		supportsBrowser = *m.SupportsBrowser
	}

	return &domainmodel.ModelCatalog{
		ID:                  m.ID,
		PublicID:            m.PublicID,
		ModelDisplayName:    m.ModelDisplayName,
		Description:         m.Description,
		SupportedParameters: supportedParameters,
		Architecture:        architecture,
		Tags:                tags,
		Notes:               m.Notes,
		ContextLength:       m.ContextLength,
		IsModerated:         m.IsModerated,
		Active:              m.Active,
		Extras:              extras,
		Status: func() domainmodel.ModelCatalogStatus {
			status := domainmodel.ModelCatalogStatus(m.Status)
			if status == "" {
				return domainmodel.ModelCatalogStatusInit
			}
			return status
		}(),
		Experimental:        experimental,
		RequiresFeatureFlag: m.RequiresFeatureFlag,
		SupportsImages:      supportsImages,
		SupportsEmbeddings:  supportsEmbeddings,
		SupportsReasoning:   supportsReasoning,
		SupportsInstruct:    supportsInstruct,
		SupportsAudio:       supportsAudio,
		SupportsVideo:       supportsVideo,
		SupportsTools:       supportsTools,
		SupportsBrowser:     supportsBrowser,
		Family:              m.Family,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}, nil
}
