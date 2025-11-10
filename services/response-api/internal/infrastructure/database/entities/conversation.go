package entities

import (
	"time"

	"gorm.io/datatypes"
)

// Conversation stores metadata for threaded chats.
type Conversation struct {
	ID        uint           `gorm:"primaryKey"`
	PublicID  string         `gorm:"uniqueIndex;size:64"`
	UserID    string         `gorm:"size:64"`
	Metadata  datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
