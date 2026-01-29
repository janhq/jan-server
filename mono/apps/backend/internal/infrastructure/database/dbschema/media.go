package dbschema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Media represents an uploaded file.
type Media struct {
	ID           string         `gorm:"type:varchar(36);primaryKey"`
	UserID       string         `gorm:"type:varchar(36);index;not null"`
	Filename     string         `gorm:"type:varchar(255);not null"`
	OriginalName string         `gorm:"type:varchar(255)"`
	MimeType     string         `gorm:"type:varchar(100)"`
	Size         int64          `gorm:"not null"`
	StorageKey   string         `gorm:"type:varchar(500);uniqueIndex;not null"` // S3 key
	Bucket       string         `gorm:"type:varchar(100)"`
	ContentHash  string         `gorm:"type:varchar(64);index"` // SHA256
	Metadata     datatypes.JSON `gorm:"type:jsonb"`
	Purpose      string         `gorm:"type:varchar(50);index"` // attachment, avatar, artifact
	ExpiresAt    *time.Time     `gorm:"index"`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
}

func (Media) TableName() string {
	return "media_files"
}

func (m *Media) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}
