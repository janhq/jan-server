package entities

import (
	"time"

	"gorm.io/datatypes"
)

// TableName specifies the table name for Artifact.
func (Artifact) TableName() string {
	return "artifacts"
}

// Artifact represents the persisted artifact record.
type Artifact struct {
	ID              uint           `gorm:"primaryKey"`
	PublicID        string         `gorm:"uniqueIndex;size:64"`
	ResponseID      uint           `gorm:"index"`
	Response        *Response      `gorm:"foreignKey:ResponseID"`
	PlanID          *uint          `gorm:"index"`
	Plan            *Plan          `gorm:"foreignKey:PlanID"`
	ContentType     string         `gorm:"size:32"`
	MimeType        string         `gorm:"size:128"`
	Title           string         `gorm:"size:512"`
	Content         *string        `gorm:"type:text"`
	StoragePath     *string        `gorm:"size:1024"`
	SizeBytes       int64          `gorm:"default:0"`
	Version         int            `gorm:"default:1"`
	ParentID        *uint          `gorm:"index"`
	Parent          *Artifact      `gorm:"foreignKey:ParentID"`
	IsLatest        *bool          `gorm:"default:true;index:idx_is_latest"`
	RetentionPolicy string         `gorm:"size:32"`
	Metadata        datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ExpiresAt       *time.Time `gorm:"index:idx_expires_at"`
}

// TableName specifies the table name for IdempotencyKey.
func (IdempotencyKey) TableName() string {
	return "idempotency_keys"
}

// IdempotencyKey represents the persisted idempotency key record.
type IdempotencyKey struct {
	ID          uint      `gorm:"primaryKey"`
	Key         string    `gorm:"uniqueIndex;size:128"`
	UserID      string    `gorm:"size:64;index:idx_user_id"`
	RequestHash string    `gorm:"size:64"`
	ResponseID  *uint     `gorm:"index"`
	Response    *Response `gorm:"foreignKey:ResponseID"`
	ExpiresAt   time.Time `gorm:"index:idx_idempotency_expires"`
	CreatedAt   time.Time
}
