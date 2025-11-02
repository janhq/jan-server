package domain

import (
	"time"

	"gorm.io/datatypes"
)

// Conversation represents a persisted chat session.
type Conversation struct {
	ID               string         `json:"id" gorm:"column:id;primaryKey"`
	OwnerPrincipalID string         `json:"owner_principal_id" gorm:"column:owner_principal_id"`
	Title            string         `json:"title" gorm:"column:title"`
	Metadata         datatypes.JSON `json:"metadata,omitempty" gorm:"column:metadata"`
	CreatedAt        time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt        time.Time      `json:"updated_at" gorm:"column:updated_at"`
}

// TableName resolves table name.
func (Conversation) TableName() string {
	return "conversations"
}
