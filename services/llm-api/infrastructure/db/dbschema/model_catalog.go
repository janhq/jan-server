package dbschema

import (
	"encoding/json"

	"jan-server/services/llm-api/domain/model"

	"gorm.io/datatypes"
)

type ModelCatalog struct {
	BaseModel
	PublicID            string         `gorm:"size:64;not null;uniqueIndex"`
	SupportedParameters datatypes.JSON `gorm:"type:jsonb;not null"`
	Architecture        datatypes.JSON `gorm:"type:jsonb;not null"`
	Tags                datatypes.JSON `gorm:"type:jsonb"`
	Notes               *string        `gorm:"type:text"`
	IsModerated         *bool          `gorm:"index"`
	Active              *bool          `gorm:"default:true;index;index:idx_model_catalog_status_active,priority:2"`
	Status              string         `gorm:"size:32;not null;default:'init';index;index:idx_model_catalog_status_active,priority:1"`
	Extras              datatypes.JSON `gorm:"type:jsonb"`
}

func (ModelCatalog) TableName() string {
	return "model_catalogs"
}

func NewSchemaModelCatalog(m *model.ModelCatalog) (*ModelCatalog, error) {
	status := string(m.Status)
	if status == "" {
		status = string(model.ModelCatalogStatusInit)
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

	return &ModelCatalog{
		BaseModel: BaseModel{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		},
		PublicID:            m.PublicID,
		SupportedParameters: datatypes.JSON(supportedParametersJSON),
		Architecture:        datatypes.JSON(architectureJSON),
		Tags:                tagsJSON,
		Notes:               m.Notes,
		IsModerated:         m.IsModerated,
		Active:              m.Active,
		Status:              status,
		Extras:              extrasJSON,
	}, nil
}

func (m *ModelCatalog) EtoD() (*model.ModelCatalog, error) {
	var supportedParameters model.SupportedParameters
	if len(m.SupportedParameters) > 0 {
		if err := json.Unmarshal(m.SupportedParameters, &supportedParameters); err != nil {
			return nil, err
		}
	}

	var architecture model.Architecture
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

	status := model.ModelCatalogStatus(m.Status)
	if status == "" {
		status = model.ModelCatalogStatusInit
	}

	return &model.ModelCatalog{
		ID:                  m.ID,
		PublicID:            m.PublicID,
		SupportedParameters: supportedParameters,
		Architecture:        architecture,
		Tags:                tags,
		Notes:               m.Notes,
		IsModerated:         m.IsModerated,
		Active:              m.Active,
		Extras:              extras,
		Status:              status,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}, nil
}
