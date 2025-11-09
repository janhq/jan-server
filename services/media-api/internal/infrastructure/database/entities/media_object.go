package entities

import "time"

// MediaObject represents the persisted media metadata.
type MediaObject struct {
	ID              string `gorm:"type:varchar(40);primaryKey"`
	StorageProvider string `gorm:"type:varchar(32);not null"`
	StorageKey      string `gorm:"type:varchar(255);not null"`
	MimeType        string `gorm:"type:varchar(64);not null"`
	Bytes           int64  `gorm:"not null"`
	Sha256          string `gorm:"type:char(64);uniqueIndex;not null"`
	CreatedBy       string `gorm:"type:varchar(64)"`
	RetentionUntil  time.Time
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (MediaObject) TableName() string {
	return "media_objects"
}
