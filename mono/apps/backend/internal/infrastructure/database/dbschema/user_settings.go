package dbschema

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"jan-server/mono/apps/backend/internal/domain/usersettings"
	"jan-server/mono/apps/backend/internal/infrastructure/database/registry"
)

func init() {
	registry.RegisterSchemaForAutoMigrate(UserSettings{})
}

// UserSettings is the database schema for user_settings table.
type UserSettings struct {
	ID     uint `gorm:"primaryKey"`
	UserID string `gorm:"not null;uniqueIndex:ux_user_settings_user_id"`

	// Other Feature Toggles
	EnableTrace bool `gorm:"not null;default:false"`
	EnableTools bool `gorm:"not null;default:true"`

	// JSONB Settings Groups
	MemoryConfig     MemoryConfigJSON     `gorm:"type:jsonb;serializer:json;not null"`
	ProfileSettings  ProfileSettingsJSON  `gorm:"type:jsonb;serializer:json;not null"`
	AdvancedSettings AdvancedSettingsJSON `gorm:"type:jsonb;serializer:json;not null"`

	// Legacy Preferences - flexible JSON (deprecated)
	Preferences JSONB `gorm:"type:jsonb;not null;default:'{}'"`

	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

// TableName specifies the table name for UserSettings.
func (UserSettings) TableName() string {
	return "llm_api.user_settings"
}

// JSONB is a custom type for JSONB columns.
type JSONB map[string]interface{}

// Value implements driver.Valuer interface for JSONB.
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface for JSONB.
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	result := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}

// Type aliases for JSONB settings groups
type MemoryConfigJSON usersettings.MemoryConfig
type ProfileSettingsJSON usersettings.ProfileSettings
type AdvancedSettingsJSON usersettings.AdvancedSettings

// EtoD converts entity (database schema) to domain model.
func (e *UserSettings) EtoD() *usersettings.UserSettings {
	return &usersettings.UserSettings{
		ID:               e.ID,
		UserID:           e.UserID,
		EnableTrace:      e.EnableTrace,
		EnableTools:      e.EnableTools,
		MemoryConfig:     usersettings.MemoryConfig(e.MemoryConfig),
		ProfileSettings:  usersettings.ProfileSettings(e.ProfileSettings),
		AdvancedSettings: usersettings.AdvancedSettings(e.AdvancedSettings),
		Preferences:      map[string]interface{}(e.Preferences),
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}
}

// NewSchemaUserSettings converts domain model to entity (database schema).
func NewSchemaUserSettings(d *usersettings.UserSettings) *UserSettings {
	prefs := JSONB(d.Preferences)
	if prefs == nil {
		prefs = make(JSONB)
	}

	return &UserSettings{
		ID:               d.ID,
		UserID:           d.UserID,
		EnableTrace:      d.EnableTrace,
		EnableTools:      d.EnableTools,
		MemoryConfig:     MemoryConfigJSON(d.MemoryConfig),
		ProfileSettings:  ProfileSettingsJSON(d.ProfileSettings),
		AdvancedSettings: AdvancedSettingsJSON(d.AdvancedSettings),
		Preferences:      prefs,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
}
