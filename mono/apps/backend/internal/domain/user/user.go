// Package user provides user domain models and behaviors.
package user

import (
	"context"
	"errors"
	"time"
)

// OIDCUser models an application user resolved from an external identity provider.
type OIDCUser struct {
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

// OIDCRepository defines storage operations for OIDC users.
type OIDCRepository interface {
	FindByIssuerAndSubject(ctx context.Context, issuer, subject string) (*OIDCUser, error)
	FindOIDCByID(ctx context.Context, id uint) (*OIDCUser, error)
	UpsertOIDC(ctx context.Context, user *OIDCUser) (*OIDCUser, error)
}

// ErrInvalidIdentity indicates missing issuer or subject on the identity payload.
var ErrInvalidIdentity = errors.New("invalid identity: issuer and subject are required")

// OIDCService persists and resolves users from external identities.
type OIDCService struct {
	repo OIDCRepository
}

// NewOIDCService constructs an OIDCService with required dependencies.
func NewOIDCService(repo OIDCRepository) *OIDCService {
	return &OIDCService{repo: repo}
}

// EnsureOIDCUser persists the given identity and returns the internal user record.
func (s *OIDCService) EnsureOIDCUser(ctx context.Context, identity Identity) (*OIDCUser, error) {
	if identity.Issuer == "" || identity.Subject == "" {
		return nil, ErrInvalidIdentity
	}

	authProvider := identity.Provider
	if authProvider == "" {
		authProvider = "keycloak"
	}

	user := &OIDCUser{
		AuthProvider: authProvider,
		Issuer:       identity.Issuer,
		Subject:      identity.Subject,
		Username:     identity.Username,
		Email:        identity.Email,
		Name:         identity.Name,
		Picture:      identity.Picture,
	}

	return s.repo.UpsertOIDC(ctx, user)
}
