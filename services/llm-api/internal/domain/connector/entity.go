// Package connector provides domain models and behaviors for external service connectors.
package connector

import (
	"time"
)

// ConnectorType represents the type of external service connector.
type ConnectorType string

const (
	ConnectorTypeGitHub         ConnectorType = "github"
	ConnectorTypeGmail          ConnectorType = "gmail"
	ConnectorTypeGoogleDrive    ConnectorType = "google_drive"
	ConnectorTypeGoogleCalendar ConnectorType = "google_calendar"
)

// AllConnectorTypes returns all supported connector types.
func AllConnectorTypes() []ConnectorType {
	return []ConnectorType{
		ConnectorTypeGitHub,
		ConnectorTypeGmail,
		ConnectorTypeGoogleDrive,
		ConnectorTypeGoogleCalendar,
	}
}

// IsValid checks if the connector type is valid.
func (ct ConnectorType) IsValid() bool {
	switch ct {
	case ConnectorTypeGitHub, ConnectorTypeGmail, ConnectorTypeGoogleDrive, ConnectorTypeGoogleCalendar:
		return true
	}
	return false
}

// IsGoogle returns true if the connector uses Google OAuth.
func (ct ConnectorType) IsGoogle() bool {
	switch ct {
	case ConnectorTypeGmail, ConnectorTypeGoogleDrive, ConnectorTypeGoogleCalendar:
		return true
	}
	return false
}

// DisplayName returns a human-readable name for the connector.
func (ct ConnectorType) DisplayName() string {
	switch ct {
	case ConnectorTypeGitHub:
		return "GitHub"
	case ConnectorTypeGmail:
		return "Gmail"
	case ConnectorTypeGoogleDrive:
		return "Google Drive"
	case ConnectorTypeGoogleCalendar:
		return "Google Calendar"
	default:
		return string(ct)
	}
}

// Connection represents an OAuth connection to an external service.
type Connection struct {
	ID                    string
	UserID                uint
	ConnectorType         ConnectorType
	AccessTokenEncrypted  string
	RefreshTokenEncrypted string
	EncryptionKeyID       string
	TokenType             string
	ExpiresAt             *time.Time
	ProviderUserID        string
	ProviderUsername      string
	ProviderEmail         string
	ProviderAvatarURL     string
	Scopes                []string
	IsConnected           bool
	LastSyncAt            *time.Time
	LastError             string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// IsExpired checks if the access token has expired.
func (c *Connection) IsExpired() bool {
	if c.ExpiresAt == nil {
		return false // Tokens without expiry don't expire (e.g., GitHub)
	}
	return time.Now().After(*c.ExpiresAt)
}

// NeedsRefresh checks if the token should be refreshed (5 min buffer).
func (c *Connection) NeedsRefresh() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return time.Now().Add(5 * time.Minute).After(*c.ExpiresAt)
}

// OAuthState represents temporary state during OAuth flow.
type OAuthState struct {
	ID            string
	UserID        uint
	ConnectorType ConnectorType
	State         string
	StateHash     string
	CodeVerifier  string
	RedirectURI   string
	ExpiresAt     time.Time
	CreatedAt     time.Time
}

// IsExpired checks if the OAuth state has expired.
func (s *OAuthState) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// AuditAction represents an auditable connector action.
type AuditAction string

const (
	AuditActionConnect        AuditAction = "connect"
	AuditActionDisconnect     AuditAction = "disconnect"
	AuditActionRefresh        AuditAction = "refresh"
	AuditActionAPICall        AuditAction = "api_call"
	AuditActionWriteOperation AuditAction = "write_operation"
)

// AuditLog represents an audit log entry for connector operations.
type AuditLog struct {
	ID            int64
	UserID        uint
	ConnectorType ConnectorType
	Action        AuditAction
	ToolName      string
	Success       bool
	ErrorMessage  string
	IPAddress     string
	UserAgent     string
	Metadata      map[string]interface{}
	CreatedAt     time.Time
}

// ConnectorInfo provides metadata about a connector type.
type ConnectorInfo struct {
	Type        ConnectorType `json:"type"`
	DisplayName string        `json:"display_name"`
	Description string        `json:"description"`
	IconURL     string        `json:"icon_url"`
	IsConnected bool          `json:"is_connected"`
	Username    string        `json:"username,omitempty"`
	Email       string        `json:"email,omitempty"`
	AvatarURL   string        `json:"avatar_url,omitempty"`
	ConnectedAt *time.Time    `json:"connected_at,omitempty"`
	Scopes      []string      `json:"scopes,omitempty"`
	HasWrite    bool          `json:"has_write"` // Whether this connector supports write operations
}

// GetConnectorInfos returns metadata for all connector types with connection status.
func GetConnectorInfos(connections map[ConnectorType]*Connection) []ConnectorInfo {
	infos := []ConnectorInfo{
		{
			Type:        ConnectorTypeGitHub,
			DisplayName: "GitHub",
			Description: "Access repositories, issues, pull requests, and code",
			IconURL:     "/icons/github.svg",
			HasWrite:    true,
		},
		{
			Type:        ConnectorTypeGmail,
			DisplayName: "Gmail",
			Description: "Search and read email messages",
			IconURL:     "/icons/gmail.svg",
			HasWrite:    false,
		},
		{
			Type:        ConnectorTypeGoogleDrive,
			DisplayName: "Google Drive",
			Description: "Search and read files from Google Drive",
			IconURL:     "/icons/google-drive.svg",
			HasWrite:    false,
		},
		{
			Type:        ConnectorTypeGoogleCalendar,
			DisplayName: "Google Calendar",
			Description: "View and manage calendar events",
			IconURL:     "/icons/google-calendar.svg",
			HasWrite:    true,
		},
	}

	for i := range infos {
		if conn, ok := connections[infos[i].Type]; ok && conn != nil && conn.IsConnected {
			infos[i].IsConnected = true
			infos[i].Username = conn.ProviderUsername
			infos[i].Email = conn.ProviderEmail
			infos[i].AvatarURL = conn.ProviderAvatarURL
			infos[i].ConnectedAt = &conn.CreatedAt
			infos[i].Scopes = conn.Scopes
		}
	}

	return infos
}
