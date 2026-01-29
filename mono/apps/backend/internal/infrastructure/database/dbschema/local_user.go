package dbschema

import (
	"time"

	"jan-server/mono/apps/backend/internal/domain/user"
	"jan-server/mono/apps/backend/internal/infrastructure/database/registry"
)

func init() {
	registry.RegisterSchemaForAutoMigrate(LocalUser{})
	registry.RegisterSchemaForAutoMigrate(RefreshToken{})
	registry.RegisterSchemaForAutoMigrate(LocalAPIKey{})
}

// LocalUser represents a local user account with password authentication.
type LocalUser struct {
	ID           string     `gorm:"type:varchar(36);primaryKey"`
	Email        string     `gorm:"type:varchar(320);uniqueIndex;not null"`
	Username     string     `gorm:"type:varchar(150);uniqueIndex;not null"`
	PasswordHash string     `gorm:"type:varchar(255);not null"`
	Name         string     `gorm:"type:varchar(255)"`
	IsActive     *bool      `gorm:"not null;default:true"`
	IsAdmin      *bool      `gorm:"not null;default:false"`
	IsGuest      *bool      `gorm:"not null;default:false"`
	LastLoginAt  *time.Time `gorm:"type:timestamptz"`
	CreatedAt    time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP"`
}

// TableName specifies the table name for LocalUser.
func (LocalUser) TableName() string {
	return "local_users"
}

// ToDomain converts a LocalUser schema to domain User.
func (u *LocalUser) ToDomain() *user.User {
	isActive := true
	if u.IsActive != nil {
		isActive = *u.IsActive
	}
	isAdmin := false
	if u.IsAdmin != nil {
		isAdmin = *u.IsAdmin
	}
	isGuest := false
	if u.IsGuest != nil {
		isGuest = *u.IsGuest
	}

	return &user.User{
		ID:           u.ID,
		Email:        u.Email,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Name:         u.Name,
		IsActive:     isActive,
		IsAdmin:      isAdmin,
		IsGuest:      isGuest,
		LastLoginAt:  u.LastLoginAt,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

// RefreshToken represents a refresh token for session management.
type RefreshToken struct {
	ID        string     `gorm:"type:varchar(36);primaryKey"`
	UserID    string     `gorm:"type:varchar(36);index;not null"`
	TokenHash string     `gorm:"type:varchar(64);uniqueIndex;not null"`
	ExpiresAt time.Time  `gorm:"type:timestamptz;not null"`
	RevokedAt *time.Time `gorm:"type:timestamptz"`
	CreatedAt time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP"`
}

// TableName specifies the table name for RefreshToken.
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// ToDomain converts a RefreshToken schema to domain RefreshToken.
func (t *RefreshToken) ToDomain() *user.RefreshToken {
	return &user.RefreshToken{
		ID:        t.ID,
		UserID:    t.UserID,
		TokenHash: t.TokenHash,
		ExpiresAt: t.ExpiresAt,
		RevokedAt: t.RevokedAt,
		CreatedAt: t.CreatedAt,
	}
}

// LocalAPIKey represents an API key for local authentication.
type LocalAPIKey struct {
	ID         string     `gorm:"type:varchar(36);primaryKey"`
	UserID     string     `gorm:"type:varchar(36);index;not null"`
	Name       string     `gorm:"type:varchar(255);not null"`
	KeyHash    string     `gorm:"type:varchar(64);uniqueIndex;not null"`
	KeyPrefix  string     `gorm:"type:varchar(20);not null"`
	Scopes     string     `gorm:"type:text"` // JSON array of scopes
	ExpiresAt  *time.Time `gorm:"type:timestamptz"`
	RevokedAt  *time.Time `gorm:"type:timestamptz"`
	LastUsedAt *time.Time `gorm:"type:timestamptz"`
	CreatedAt  time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP"`
}

// TableName specifies the table name for LocalAPIKey.
func (LocalAPIKey) TableName() string {
	return "local_api_keys"
}

// ToDomain converts a LocalAPIKey schema to domain APIKey.
func (k *LocalAPIKey) ToDomain() *user.APIKey {
	// Parse scopes from JSON string if present
	var scopes []string
	// For now, we'll leave scopes as nil; proper JSON parsing can be added if needed

	return &user.APIKey{
		ID:         k.ID,
		UserID:     k.UserID,
		Name:       k.Name,
		KeyHash:    k.KeyHash,
		KeyPrefix:  k.KeyPrefix,
		Scopes:     scopes,
		ExpiresAt:  k.ExpiresAt,
		RevokedAt:  k.RevokedAt,
		LastUsedAt: k.LastUsedAt,
		CreatedAt:  k.CreatedAt,
	}
}
