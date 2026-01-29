package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"jan-server/mono/apps/backend/internal/domain/connector"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"

	"gorm.io/gorm"
)

// ConnectorRepository implements connector.Repository using GORM.
type ConnectorRepository struct {
	db *gorm.DB
}

// NewConnectorRepository creates a new connector repository.
func NewConnectorRepository(db *gorm.DB) *ConnectorRepository {
	return &ConnectorRepository{db: db}
}

var _ connector.Repository = (*ConnectorRepository)(nil)

func (r *ConnectorRepository) Create(ctx context.Context, c *connector.Connector) error {
	schema := toConnectorSchema(c)
	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		return err
	}
	c.ID = schema.ID
	c.CreatedAt = schema.CreatedAt
	c.UpdatedAt = schema.UpdatedAt
	return nil
}

func (r *ConnectorRepository) GetByID(ctx context.Context, id string) (*connector.Connector, error) {
	var schema dbschema.Connector
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connector.ErrConnectorNotFound
		}
		return nil, err
	}
	return toConnectorDomain(&schema), nil
}

func (r *ConnectorRepository) GetByUserAndProvider(ctx context.Context, userID, provider string) (*connector.Connector, error) {
	var schema dbschema.Connector
	if err := r.db.WithContext(ctx).Where("user_id = ? AND provider = ?", userID, provider).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connector.ErrConnectorNotFound
		}
		return nil, err
	}
	return toConnectorDomain(&schema), nil
}

func (r *ConnectorRepository) Update(ctx context.Context, c *connector.Connector) error {
	schema := toConnectorSchema(c)
	return r.db.WithContext(ctx).Model(&dbschema.Connector{}).Where("id = ?", c.ID).Updates(schema).Error
}

func (r *ConnectorRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&dbschema.Connector{}, "id = ?", id).Error
}

func (r *ConnectorRepository) ListByUser(ctx context.Context, userID string) ([]*connector.Connector, error) {
	var schemas []dbschema.Connector
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&schemas).Error; err != nil {
		return nil, err
	}

	connectors := make([]*connector.Connector, len(schemas))
	for i, s := range schemas {
		connectors[i] = toConnectorDomain(&s)
	}

	return connectors, nil
}

// ============================================
// OAuth state operations
// ============================================

func (r *ConnectorRepository) CreateState(ctx context.Context, state *connector.OAuthState) error {
	schema := &dbschema.ConnectorOAuthState{
		State:       state.State,
		UserID:      state.UserID,
		Provider:    state.Provider,
		RedirectURL: state.RedirectURL,
		ExpiresAt:   state.ExpiresAt,
	}
	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		return err
	}
	state.ID = schema.ID
	state.CreatedAt = schema.CreatedAt
	return nil
}

func (r *ConnectorRepository) GetState(ctx context.Context, state string) (*connector.OAuthState, error) {
	var schema dbschema.ConnectorOAuthState
	if err := r.db.WithContext(ctx).Where("state = ?", state).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connector.ErrInvalidState
		}
		return nil, err
	}
	return &connector.OAuthState{
		ID:          schema.ID,
		State:       schema.State,
		UserID:      schema.UserID,
		Provider:    schema.Provider,
		RedirectURL: schema.RedirectURL,
		ExpiresAt:   schema.ExpiresAt,
		CreatedAt:   schema.CreatedAt,
	}, nil
}

func (r *ConnectorRepository) DeleteState(ctx context.Context, state string) error {
	return r.db.WithContext(ctx).Delete(&dbschema.ConnectorOAuthState{}, "state = ?", state).Error
}

func (r *ConnectorRepository) DeleteExpiredStates(ctx context.Context) error {
	return r.db.WithContext(ctx).Delete(&dbschema.ConnectorOAuthState{}, "expires_at < ?", time.Now()).Error
}

// ============================================
// Conversion helpers
// ============================================

func toConnectorSchema(c *connector.Connector) *dbschema.Connector {
	metadataJSON, _ := json.Marshal(c.Metadata)
	isActive := c.IsActive
	return &dbschema.Connector{
		ID:               c.ID,
		UserID:           c.UserID,
		Provider:         c.Provider,
		ProviderUserID:   c.ProviderUserID,
		ProviderUsername: c.ProviderUsername,
		ProviderEmail:    c.ProviderEmail,
		AccessToken:      c.AccessToken,
		RefreshToken:     c.RefreshToken,
		TokenType:        c.TokenType,
		Scopes:           c.Scopes,
		ExpiresAt:        c.ExpiresAt,
		Metadata:         metadataJSON,
		LastSyncAt:       c.LastSyncAt,
		IsActive:         &isActive,
		EncryptionKeyID:  c.EncryptionKeyID,
	}
}

func toConnectorDomain(s *dbschema.Connector) *connector.Connector {
	var metadata map[string]any
	_ = json.Unmarshal(s.Metadata, &metadata)

	isActive := true
	if s.IsActive != nil {
		isActive = *s.IsActive
	}

	return &connector.Connector{
		ID:               s.ID,
		UserID:           s.UserID,
		Provider:         s.Provider,
		ProviderUserID:   s.ProviderUserID,
		ProviderUsername: s.ProviderUsername,
		ProviderEmail:    s.ProviderEmail,
		AccessToken:      s.AccessToken,
		RefreshToken:     s.RefreshToken,
		TokenType:        s.TokenType,
		Scopes:           s.Scopes,
		ExpiresAt:        s.ExpiresAt,
		Metadata:         metadata,
		LastSyncAt:       s.LastSyncAt,
		IsActive:         isActive,
		EncryptionKeyID:  s.EncryptionKeyID,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}
