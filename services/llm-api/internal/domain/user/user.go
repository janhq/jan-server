// Package user provides user domain models and behaviors.
package user

import (
	"context"
	"errors"
	"time"
)

// User models an application user resolved from an external identity provider.
type User struct {
	ID           uint
	AuthProvider string
	Issuer       string
	Subject      string
	Username     *string
	Email        *string
	Name         *string
	Picture      *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Identity encapsulates the externally provided identity attributes.
type Identity struct {
	Provider string
	Issuer   string
	Subject  string
	Username *string
	Email    *string
	Name     *string
	Picture  *string
}

// Repository defines storage operations for users.
type Repository interface {
	FindByIssuerAndSubject(ctx context.Context, issuer, subject string) (*User, error)
	FindByID(ctx context.Context, id uint) (*User, error)
	Upsert(ctx context.Context, user *User) (*User, error)
}

// ErrInvalidIdentity indicates missing issuer or subject on the identity payload.
var ErrInvalidIdentity = errors.New("invalid identity: issuer and subject are required")

// Service persists and resolves users from external identities.
type Service struct {
	repo Repository
}

// NewService constructs a Service with required dependencies.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// EnsureUser persists the given identity and returns the internal user record.
func (s *Service) EnsureUser(ctx context.Context, identity Identity) (*User, error) {
	if identity.Issuer == "" || identity.Subject == "" {
		return nil, ErrInvalidIdentity
	}

	authProvider := identity.Provider
	if authProvider == "" {
		authProvider = "keycloak"
	}

	user := &User{
		AuthProvider: authProvider,
		Issuer:       identity.Issuer,
		Subject:      identity.Subject,
		Username:     identity.Username,
		Email:        identity.Email,
		Name:         identity.Name,
		Picture:      identity.Picture,
	}

	return s.repo.Upsert(ctx, user)
}
