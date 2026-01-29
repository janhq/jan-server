package dbschema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a local user in the database.
type User struct {
	ID           string     `gorm:"type:varchar(36);primaryKey"`
	Email        string     `gorm:"type:varchar(255);uniqueIndex;not null"`
	Username     string     `gorm:"type:varchar(100);uniqueIndex;not null"`
	PasswordHash string     `gorm:"type:varchar(255);not null"`
	Name         string     `gorm:"type:varchar(255)"`
	IsActive     *bool      `gorm:"not null;default:true"`
	IsAdmin      *bool      `gorm:"not null;default:false"`
	LastLoginAt  *time.Time `gorm:"index"`
	CreatedAt    time.Time  `gorm:"autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime"`
}

// TableName returns the table name with prefix support.
func (User) TableName() string {
	return "users"
}

// BeforeCreate generates a UUID if not set.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

// APIKey represents an API key for programmatic access.
type APIKey struct {
	ID          string     `gorm:"type:varchar(36);primaryKey"`
	UserID      string     `gorm:"type:varchar(36);index;not null"`
	Name        string     `gorm:"type:varchar(100);not null"`
	KeyHash     string     `gorm:"type:varchar(255);uniqueIndex;not null"` // SHA256 hash of the key
	KeyPrefix   string     `gorm:"type:varchar(20);not null"`              // First 8 chars for display (e.g., "sk_live_abc")
	Scopes      string     `gorm:"type:text"`                              // JSON array of allowed scopes
	LastUsedAt  *time.Time `gorm:"index"`
	ExpiresAt   *time.Time `gorm:"index"`
	RevokedAt   *time.Time `gorm:"index"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`

	// Relations
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name.
func (APIKey) TableName() string {
	return "api_keys"
}

// BeforeCreate generates a UUID if not set.
func (a *APIKey) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// IsExpired checks if the API key has expired.
func (a *APIKey) IsExpired() bool {
	if a.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*a.ExpiresAt)
}

// IsRevoked checks if the API key has been revoked.
func (a *APIKey) IsRevoked() bool {
	return a.RevokedAt != nil
}

// RefreshToken stores refresh tokens for local auth.
type RefreshToken struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	UserID    string    `gorm:"type:varchar(36);index;not null"`
	TokenHash string    `gorm:"type:varchar(255);uniqueIndex;not null"` // SHA256 hash
	ExpiresAt time.Time `gorm:"index;not null"`
	RevokedAt *time.Time
	CreatedAt time.Time `gorm:"autoCreateTime"`

	// Relations
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name.
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// BeforeCreate generates a UUID if not set.
func (r *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

// IsExpired checks if the refresh token has expired.
func (r *RefreshToken) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

// IsRevoked checks if the refresh token has been revoked.
func (r *RefreshToken) IsRevoked() bool {
	return r.RevokedAt != nil
}
