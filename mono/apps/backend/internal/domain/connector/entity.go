package connector

import (
	"time"
)

// Connector represents an OAuth connector for a user.
type Connector struct {
	ID               string
	UserID           string
	Provider         string // github, google, gmail, google_drive, google_calendar
	ProviderUserID   string
	ProviderUsername string
	ProviderEmail    string
	AccessToken      string // Encrypted
	RefreshToken     string // Encrypted
	TokenType        string
	Scopes           string
	ExpiresAt        *time.Time
	Metadata         map[string]any
	LastSyncAt       *time.Time
	IsActive         bool
	EncryptionKeyID  string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// OAuthState stores temporary OAuth state for CSRF protection.
type OAuthState struct {
	ID          string
	State       string
	UserID      string
	Provider    string
	RedirectURL string
	ExpiresAt   time.Time
	CreatedAt   time.Time
}

// ConnectorConfig holds configuration for a connector provider.
type ConnectorConfig struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	Scopes       []string
	RedirectURL  string
}

// ConnectorResponse is the API response for a connector.
type ConnectorResponse struct {
	ID               string         `json:"id"`
	Provider         string         `json:"provider"`
	ProviderUserID   string         `json:"provider_user_id,omitempty"`
	ProviderUsername string         `json:"provider_username,omitempty"`
	ProviderEmail    string         `json:"provider_email,omitempty"`
	Scopes           string         `json:"scopes,omitempty"`
	IsActive         bool           `json:"is_active"`
	LastSyncAt       *time.Time     `json:"last_sync_at,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
}

// AuthURLResponse is the response containing the OAuth authorization URL.
type AuthURLResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

// ToResponse converts a Connector to ConnectorResponse.
func (c *Connector) ToResponse() ConnectorResponse {
	return ConnectorResponse{
		ID:               c.ID,
		Provider:         c.Provider,
		ProviderUserID:   c.ProviderUserID,
		ProviderUsername: c.ProviderUsername,
		ProviderEmail:    c.ProviderEmail,
		Scopes:           c.Scopes,
		IsActive:         c.IsActive,
		LastSyncAt:       c.LastSyncAt,
		Metadata:         c.Metadata,
		CreatedAt:        c.CreatedAt,
	}
}

// Provider types
const (
	ProviderGitHub         = "github"
	ProviderGoogle         = "google"
	ProviderGmail          = "gmail"
	ProviderGoogleDrive    = "google_drive"
	ProviderGoogleCalendar = "google_calendar"
)

// IsValidProvider checks if a provider is supported.
func IsValidProvider(provider string) bool {
	switch provider {
	case ProviderGitHub, ProviderGoogle, ProviderGmail, ProviderGoogleDrive, ProviderGoogleCalendar:
		return true
	default:
		return false
	}
}
