package entities

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TableName specifies the table name for Response.
func (Response) TableName() string {
	return "responses"
}

// Response represents the persisted response record.
type Response struct {
	ID                 uint           `gorm:"primaryKey"`
	PublicID           string         `gorm:"uniqueIndex;size:64"`
	UserID             string         `gorm:"size:64"`
	Model              string         `gorm:"size:128"`
	SystemPrompt       *string        `gorm:"type:text"`
	Input              datatypes.JSON `gorm:"type:jsonb"`
	Output             datatypes.JSON `gorm:"type:jsonb"`
	Status             string         `gorm:"size:32;index:idx_status"`
	Stream             bool
	Background         bool           `gorm:"default:false"`
	Store              bool           `gorm:"default:false"`
	APIKey             *string        `gorm:"type:text"` // Store API key (X-API-Key or Bearer token) for background tasks
	Metadata           datatypes.JSON `gorm:"type:jsonb"`
	Usage              datatypes.JSON `gorm:"type:jsonb"`
	Error              datatypes.JSON `gorm:"type:jsonb"`
	Artifacts          datatypes.JSON `gorm:"type:jsonb;default:'[]'"` // Array of MediaArtifact (uploaded to media-api)
	Citations          datatypes.JSON `gorm:"type:jsonb;default:'[]'"` // Array of Citation (from search/research steps)
	ConversationID     *uint
	Conversation       *Conversation
	PreviousResponseID *string `gorm:"size:64"`
	Object             string  `gorm:"size:32"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	QueuedAt           *time.Time
	StartedAt          *time.Time
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
