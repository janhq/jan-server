package dbschema

import (
	"time"

	"github.com/lib/pq"

	"jan-server/services/llm-api/internal/domain/connector"
	"jan-server/services/llm-api/internal/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(&ConnectorConnection{})
	database.RegisterSchemaForAutoMigrate(&ConnectorOAuthState{})
	database.RegisterSchemaForAutoMigrate(&ConnectorAuditLog{})
}

// ConnectorConnection represents a user's OAuth connection to an external service.
type ConnectorConnection struct {
	ID                    string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID                uint           `gorm:"not null;index:idx_connector_connections_user_type"`
	ConnectorType         string         `gorm:"type:varchar(32);not null;index:idx_connector_connections_user_type"`
	AccessTokenEncrypted  string         `gorm:"type:text;not null"`
	RefreshTokenEncrypted string         `gorm:"type:text"`
	EncryptionKeyID       string         `gorm:"type:varchar(32);not null;default:'v1'"`
	TokenType             string         `gorm:"type:varchar(32);default:'Bearer'"`
	ExpiresAt             *time.Time     `gorm:"index:idx_connector_connections_expires"`
	ProviderUserID        string         `gorm:"type:varchar(255)"`
	ProviderUsername      string         `gorm:"type:varchar(255)"`
	ProviderEmail         string         `gorm:"type:varchar(255)"`
	ProviderAvatarURL     string         `gorm:"type:text"`
	Scopes                pq.StringArray `gorm:"type:text[]"`
	IsConnected           *bool          `gorm:"not null;default:true;index:idx_connector_connections_connected"`
	LastSyncAt            *time.Time
	LastError             string `gorm:"type:text"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// TableName returns the table name with schema.
func (ConnectorConnection) TableName() string {
	return "llm_api.connector_connections"
}

// EtoD converts schema model to domain representation.
func (c *ConnectorConnection) EtoD() *connector.Connection {
	if c == nil {
		return nil
	}

	isConnected := true
	if c.IsConnected != nil {
		isConnected = *c.IsConnected
	}

	return &connector.Connection{
		ID:                    c.ID,
		UserID:                c.UserID,
		ConnectorType:         connector.ConnectorType(c.ConnectorType),
		AccessTokenEncrypted:  c.AccessTokenEncrypted,
		RefreshTokenEncrypted: c.RefreshTokenEncrypted,
		EncryptionKeyID:       c.EncryptionKeyID,
		TokenType:             c.TokenType,
		ExpiresAt:             c.ExpiresAt,
		ProviderUserID:        c.ProviderUserID,
		ProviderUsername:      c.ProviderUsername,
		ProviderEmail:         c.ProviderEmail,
		ProviderAvatarURL:     c.ProviderAvatarURL,
		Scopes:                []string(c.Scopes),
		IsConnected:           isConnected,
		LastSyncAt:            c.LastSyncAt,
		LastError:             c.LastError,
		CreatedAt:             c.CreatedAt,
		UpdatedAt:             c.UpdatedAt,
	}
}

// FromDomainConnection converts domain model to schema representation.
func FromDomainConnection(conn *connector.Connection) *ConnectorConnection {
	if conn == nil {
		return nil
	}

	isConnected := conn.IsConnected

	return &ConnectorConnection{
		ID:                    conn.ID,
		UserID:                conn.UserID,
		ConnectorType:         string(conn.ConnectorType),
		AccessTokenEncrypted:  conn.AccessTokenEncrypted,
		RefreshTokenEncrypted: conn.RefreshTokenEncrypted,
		EncryptionKeyID:       conn.EncryptionKeyID,
		TokenType:             conn.TokenType,
		ExpiresAt:             conn.ExpiresAt,
		ProviderUserID:        conn.ProviderUserID,
		ProviderUsername:      conn.ProviderUsername,
		ProviderEmail:         conn.ProviderEmail,
		ProviderAvatarURL:     conn.ProviderAvatarURL,
		Scopes:                pq.StringArray(conn.Scopes),
		IsConnected:           &isConnected,
		LastSyncAt:            conn.LastSyncAt,
		LastError:             conn.LastError,
		CreatedAt:             conn.CreatedAt,
		UpdatedAt:             conn.UpdatedAt,
	}
}

// ConnectorOAuthState represents temporary OAuth state during authorization flow.
type ConnectorOAuthState struct {
	ID            string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID        uint   `gorm:"not null"`
	ConnectorType string `gorm:"type:varchar(32);not null"`
	State         string `gorm:"type:varchar(128);not null;uniqueIndex"`
	StateHash     string `gorm:"type:varchar(64);not null"`
	CodeVerifier  string `gorm:"type:varchar(128)"`
	RedirectURI   string `gorm:"type:text"`
	ExpiresAt     time.Time
	CreatedAt     time.Time
}

// TableName returns the table name with schema.
func (ConnectorOAuthState) TableName() string {
	return "llm_api.connector_oauth_states"
}

// EtoD converts schema model to domain representation.
func (s *ConnectorOAuthState) EtoD() *connector.OAuthState {
	if s == nil {
		return nil
	}

	return &connector.OAuthState{
		ID:            s.ID,
		UserID:        s.UserID,
		ConnectorType: connector.ConnectorType(s.ConnectorType),
		State:         s.State,
		StateHash:     s.StateHash,
		CodeVerifier:  s.CodeVerifier,
		RedirectURI:   s.RedirectURI,
		ExpiresAt:     s.ExpiresAt,
		CreatedAt:     s.CreatedAt,
	}
}

// FromDomainOAuthState converts domain model to schema representation.
func FromDomainOAuthState(state *connector.OAuthState) *ConnectorOAuthState {
	if state == nil {
		return nil
	}

	return &ConnectorOAuthState{
		ID:            state.ID,
		UserID:        state.UserID,
		ConnectorType: string(state.ConnectorType),
		State:         state.State,
		StateHash:     state.StateHash,
		CodeVerifier:  state.CodeVerifier,
		RedirectURI:   state.RedirectURI,
		ExpiresAt:     state.ExpiresAt,
		CreatedAt:     state.CreatedAt,
	}
}

// ConnectorAuditLog represents an audit log entry for connector operations.
type ConnectorAuditLog struct {
	ID            int64  `gorm:"primaryKey;autoIncrement"`
	UserID        uint   `gorm:"not null;index:idx_connector_audit_log_user"`
	ConnectorType string `gorm:"type:varchar(32);not null"`
	Action        string `gorm:"type:varchar(64);not null;index:idx_connector_audit_log_action"`
	ToolName      string `gorm:"type:varchar(128)"`
	Success       *bool  `gorm:"not null;default:true"`
	ErrorMessage  string `gorm:"type:text"`
	IPAddress     string `gorm:"type:varchar(45)"`
	UserAgent     string `gorm:"type:text"`
	Metadata      []byte `gorm:"type:jsonb"`
	CreatedAt     time.Time
}

// TableName returns the table name with schema.
func (ConnectorAuditLog) TableName() string {
	return "llm_api.connector_audit_log"
}

// EtoD converts schema model to domain representation.
func (l *ConnectorAuditLog) EtoD() *connector.AuditLog {
	if l == nil {
		return nil
	}

	success := true
	if l.Success != nil {
		success = *l.Success
	}

	return &connector.AuditLog{
		ID:            l.ID,
		UserID:        l.UserID,
		ConnectorType: connector.ConnectorType(l.ConnectorType),
		Action:        connector.AuditAction(l.Action),
		ToolName:      l.ToolName,
		Success:       success,
		ErrorMessage:  l.ErrorMessage,
		IPAddress:     l.IPAddress,
		UserAgent:     l.UserAgent,
		CreatedAt:     l.CreatedAt,
	}
}

// FromDomainAuditLog converts domain model to schema representation.
func FromDomainAuditLog(log *connector.AuditLog) *ConnectorAuditLog {
	if log == nil {
		return nil
	}

	return &ConnectorAuditLog{
		ID:            log.ID,
		UserID:        log.UserID,
		ConnectorType: string(log.ConnectorType),
		Action:        string(log.Action),
		ToolName:      log.ToolName,
		Success:       &log.Success,
		ErrorMessage:  log.ErrorMessage,
		IPAddress:     log.IPAddress,
		UserAgent:     log.UserAgent,
		CreatedAt:     log.CreatedAt,
	}
}
