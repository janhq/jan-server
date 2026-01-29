package dbschema

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Connector represents an OAuth connector for a user (GitHub, Google, etc.).
type Connector struct {
	ID                 string         `gorm:"type:varchar(36);primaryKey"`
	UserID             string         `gorm:"type:varchar(36);index;not null"`
	Provider           string         `gorm:"type:varchar(50);index;not null"` // github, google, gmail, google_drive, google_calendar
	ProviderUserID     string         `gorm:"type:varchar(100)"`
	ProviderUsername   string         `gorm:"type:varchar(100)"`
	ProviderEmail      string         `gorm:"type:varchar(255)"`
	AccessToken        string         `gorm:"type:text"`                       // Encrypted
	RefreshToken       string         `gorm:"type:text"`                       // Encrypted
	TokenType          string         `gorm:"type:varchar(50)"`
	Scopes             string         `gorm:"type:text"`                       // Space-separated scopes
	ExpiresAt          *time.Time
	Metadata           datatypes.JSON `gorm:"type:jsonb"`
	LastSyncAt         *time.Time
	IsActive           *bool          `gorm:"not null;default:true"`
	EncryptionKeyID    string         `gorm:"type:varchar(50)"` // For key rotation
	CreatedAt          time.Time      `gorm:"autoCreateTime"`
	UpdatedAt          time.Time      `gorm:"autoUpdateTime"`
}

func (Connector) TableName() string {
	return "connectors"
}

func (c *Connector) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// ConnectorOAuthState stores temporary OAuth state for CSRF protection.
type ConnectorOAuthState struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	State     string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	UserID    string    `gorm:"type:varchar(36);index;not null"`
	Provider  string    `gorm:"type:varchar(50);not null"`
	RedirectURL string  `gorm:"type:varchar(500)"`
	ExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (ConnectorOAuthState) TableName() string {
	return "connector_oauth_states"
}

func (s *ConnectorOAuthState) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}
