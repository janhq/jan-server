package domain

import "time"

// Model describes a provider-backed model registration.
type Model struct {
	ID           string    `json:"id" gorm:"column:id;primaryKey"`
	Provider     string    `json:"provider" gorm:"column:provider"`
	DisplayName  string    `json:"display_name" gorm:"column:display_name"`
	Family       string    `json:"family" gorm:"column:family"`
	Capabilities []string  `json:"capabilities" gorm:"column:capabilities;type:jsonb;serializer:json"`
	Active       bool      `json:"active" gorm:"column:active"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at"`
}

// TableName satisfies gorm's table naming.
func (Model) TableName() string {
	return "models"
}
