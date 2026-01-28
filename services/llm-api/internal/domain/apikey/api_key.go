package apikey

import (
	"context"
	"time"
)

// APIKey represents persistent metadata for an API key.
type APIKey struct {
	ID         string
	UserID     uint
	Name       string
	Prefix     string
	Suffix     string
	Hash       string
	IsSystem   bool // True for system-generated keys (e.g., Response API temp keys)
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Repository defines storage operations for API keys.
type Repository interface {
	Create(ctx context.Context, key *APIKey) (*APIKey, error)
	ListByUser(ctx context.Context, userID uint) ([]APIKey, error)
	ListUserKeys(ctx context.Context, userID uint) ([]APIKey, error) // Excludes system keys
	FindByID(ctx context.Context, id string) (*APIKey, error)
	FindByHash(ctx context.Context, hash string) (*APIKey, error)
	FindActiveSystemKey(ctx context.Context, userID uint) (*APIKey, error) // Find reusable system key
	CountActiveByUser(ctx context.Context, userID uint) (int64, error)
	MarkRevoked(ctx context.Context, id string, revokedAt time.Time) error
}
