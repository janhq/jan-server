package dbschema

import (
	"encoding/json"
	"time"

	"jan-server/services/llm-api/domain/model"

	"gorm.io/datatypes"
)

type Provider struct {
	BaseModel
	PublicID        string         `gorm:"size:64;not null;uniqueIndex"`
	DisplayName     string         `gorm:"size:255;not null"`
	Kind            string         `gorm:"size:64;not null;index;index:idx_provider_active_kind,priority:2"`
	BaseURL         string         `gorm:"size:512"`
	EncryptedAPIKey string         `gorm:"type:text"`
	APIKeyHint      *string        `gorm:"size:128"`
	IsModerated     *bool          `gorm:"not null;default:false;index"`
	Active          *bool          `gorm:"not null;default:true;index;index:idx_provider_active_kind,priority:1"`
	Metadata        datatypes.JSON `gorm:"type:jsonb"`
	LastSyncedAt    *time.Time     `gorm:"index"`
}

func (Provider) TableName() string {
	return "providers"
}

func NewSchemaProvider(p *model.Provider) *Provider {
	var metadataJSON datatypes.JSON
	if len(p.Metadata) > 0 {
		if data, err := json.Marshal(p.Metadata); err == nil {
			metadataJSON = datatypes.JSON(data)
		}
	}

	isModerated := p.IsModerated
	active := p.Active
	return &Provider{
		BaseModel: BaseModel{
			ID:        p.ID,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
		PublicID:        p.PublicID,
		DisplayName:     p.DisplayName,
		Kind:            string(p.Kind),
		BaseURL:         p.BaseURL,
		EncryptedAPIKey: p.EncryptedAPIKey,
		APIKeyHint:      p.APIKeyHint,
		IsModerated:     &isModerated,
		Active:          &active,
		Metadata:        metadataJSON,
		LastSyncedAt:    p.LastSyncedAt,
	}
}

// EtoD converts a database provider into its domain representation
func (p *Provider) EtoD() *model.Provider {
	var metadata map[string]string
	if len(p.Metadata) > 0 {
		_ = json.Unmarshal(p.Metadata, &metadata)
	}

	isModerated := false
	if p.IsModerated != nil {
		isModerated = *p.IsModerated
	}
	active := false
	if p.Active != nil {
		active = *p.Active
	}

	return &model.Provider{
		ID:              p.ID,
		PublicID:        p.PublicID,
		DisplayName:     p.DisplayName,
		Kind:            model.ProviderKind(p.Kind),
		BaseURL:         p.BaseURL,
		EncryptedAPIKey: p.EncryptedAPIKey,
		APIKeyHint:      p.APIKeyHint,
		IsModerated:     isModerated,
		Active:          active,
		Metadata:        metadata,
		LastSyncedAt:    p.LastSyncedAt,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}
