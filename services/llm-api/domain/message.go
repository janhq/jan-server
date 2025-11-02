package domain

import (
	"time"

	"gorm.io/datatypes"
)

// Message represents a chat message within a conversation.
type Message struct {
	ID             string         `json:"id" gorm:"column:id;primaryKey"`
	ConversationID string         `json:"conversation_id" gorm:"column:conversation_id"`
	Role           string         `json:"role" gorm:"column:role"`
	Content        datatypes.JSON `json:"content" gorm:"column:content"`
	ToolCalls      datatypes.JSON `json:"tool_calls,omitempty" gorm:"column:tool_calls"`
	CreatedAt      time.Time      `json:"created_at" gorm:"column:created_at"`
}

// TableName resolves table name.
func (Message) TableName() string {
	return "messages"
}
