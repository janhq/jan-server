package entities

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Response represents the persisted response record.
type Response struct {
	ID                 uint           `gorm:"primaryKey"`
	PublicID           string         `gorm:"uniqueIndex;size:64"`
	UserID             string         `gorm:"size:64"`
	Model              string         `gorm:"size:128"`
	SystemPrompt       *string        `gorm:"type:text"`
	Input              datatypes.JSON `gorm:"type:jsonb"`
	Output             datatypes.JSON `gorm:"type:jsonb"`
	Status             string         `gorm:"size:32"`
	Stream             bool
	Metadata           datatypes.JSON `gorm:"type:jsonb"`
	Usage              datatypes.JSON `gorm:"type:jsonb"`
	Error              datatypes.JSON `gorm:"type:jsonb"`
	ConversationID     *uint
	Conversation       *Conversation
	PreviousResponseID *string `gorm:"size:64"`
	Object             string  `gorm:"size:32"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	CompletedAt        *time.Time
	CancelledAt        *time.Time
	FailedAt           *time.Time
}

// BeforeCreate ensures defaults.
func (r *Response) BeforeCreate(tx *gorm.DB) error {
	if r.Object == "" {
		r.Object = "response"
	}
	return nil
}
