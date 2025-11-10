package entities

import (
	"time"

	"gorm.io/datatypes"
)

// ConversationItem stores each message for a conversation.
type ConversationItem struct {
	ID             uint           `gorm:"primaryKey"`
	ConversationID uint           `gorm:"index"`
	Role           string         `gorm:"size:32"`
	Status         string         `gorm:"size:32"`
	Content        datatypes.JSON `gorm:"type:jsonb"`
	Sequence       int            `gorm:"index"`
	CreatedAt      time.Time
}
