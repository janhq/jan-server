package connector

import (
	"context"
)

// Repository defines storage operations for connector connections.
type Repository interface {
	// Connection operations
	Create(ctx context.Context, conn *Connection) (*Connection, error)
	Update(ctx context.Context, conn *Connection) (*Connection, error)
	FindByUserAndType(ctx context.Context, userID uint, connectorType ConnectorType) (*Connection, error)
	FindAllByUser(ctx context.Context, userID uint) ([]*Connection, error)
	Delete(ctx context.Context, userID uint, connectorType ConnectorType) error

	// OAuth state operations
	CreateOAuthState(ctx context.Context, state *OAuthState) (*OAuthState, error)
	FindOAuthStateByState(ctx context.Context, state string) (*OAuthState, error)
	DeleteOAuthState(ctx context.Context, state string) error
	DeleteExpiredOAuthStates(ctx context.Context) (int64, error)

	// Audit operations
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	FindAuditLogsByUser(ctx context.Context, userID uint, limit int) ([]*AuditLog, error)
}
