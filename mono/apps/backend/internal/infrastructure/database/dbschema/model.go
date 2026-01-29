package dbschema

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"

	"jan-server/mono/apps/backend/internal/domain"
	"jan-server/mono/apps/backend/internal/infrastructure/database/registry"
)

func init() {
	registry.RegisterSchemaForAutoMigrate(Model{})
}

type Model struct {
	ID           string         `gorm:"column:id;size:255;primaryKey"`
	Provider     string         `gorm:"column:provider;size:255;not null"`
	DisplayName  string         `gorm:"column:display_name;size:255;not null"`
	Family       string         `gorm:"column:family;size:255"`
	Capabilities datatypes.JSON `gorm:"column:capabilities;type:jsonb"`
	Active       *bool          `gorm:"column:active;not null;default:true"`
	CreatedAt    time.Time      `gorm:"column:created_at;not null"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;not null"`
}

func (Model) TableName() string {
	return "models"
}

func NewSchemaModel(model *domain.Model) (*Model, error) {
	if model == nil {
		return nil, nil
	}

	var capabilities datatypes.JSON
	if len(model.Capabilities) > 0 {
		data, err := json.Marshal(model.Capabilities)
		if err != nil {
			return nil, err
		}
		capabilities = datatypes.JSON(data)
	}

	active := model.Active

	return &Model{
		ID:           model.ID,
		Provider:     model.Provider,
		DisplayName:  model.DisplayName,
		Family:       model.Family,
		Capabilities: capabilities,
		Active:       &active,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}, nil
}

func (m *Model) EtoD() (*domain.Model, error) {
	if m == nil {
		return nil, nil
	}

	var capabilities []string
	if len(m.Capabilities) > 0 {
		if err := json.Unmarshal(m.Capabilities, &capabilities); err != nil {
			return nil, err
		}
	}

	active := false
	if m.Active != nil {
		active = *m.Active
	}

	return &domain.Model{
		ID:           m.ID,
		Provider:     m.Provider,
		DisplayName:  m.DisplayName,
		Family:       m.Family,
		Capabilities: capabilities,
		Active:       active,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}, nil
}
