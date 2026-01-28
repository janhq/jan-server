package connectorrepo

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"jan-server/services/llm-api/internal/domain/connector"
	"jan-server/services/llm-api/internal/infrastructure/database/dbschema"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

// Repository implements the connector.Repository interface.
type Repository struct {
	db *gorm.DB
}

// NewConnectorRepository creates a new connector repository.
func NewConnectorRepository(db *gorm.DB) connector.Repository {
	return &Repository{db: db}
}

// Create creates a new connector connection.
func (r *Repository) Create(ctx context.Context, conn *connector.Connection) (*connector.Connection, error) {
	model := dbschema.FromDomainConnection(conn)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to create connector connection")
	}
	return model.EtoD(), nil
}

// Update updates an existing connector connection.
func (r *Repository) Update(ctx context.Context, conn *connector.Connection) (*connector.Connection, error) {
	model := dbschema.FromDomainConnection(conn)
	model.UpdatedAt = time.Now()

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to update connector connection")
	}
	return model.EtoD(), nil
}

// FindByUserAndType finds a connection by user ID and connector type.
func (r *Repository) FindByUserAndType(ctx context.Context, userID uint, connectorType connector.ConnectorType) (*connector.Connection, error) {
	var model dbschema.ConnectorConnection
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND connector_type = ?", userID, string(connectorType)).
		First(&model).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to find connector connection")
	}

	return model.EtoD(), nil
}

// FindAllByUser finds all connections for a user.
func (r *Repository) FindAllByUser(ctx context.Context, userID uint) ([]*connector.Connection, error) {
	var models []dbschema.ConnectorConnection
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&models).Error

	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to list connector connections")
	}

	result := make([]*connector.Connection, 0, len(models))
	for _, m := range models {
		if domain := m.EtoD(); domain != nil {
			result = append(result, domain)
		}
	}

	return result, nil
}

// Delete deletes a connector connection.
func (r *Repository) Delete(ctx context.Context, userID uint, connectorType connector.ConnectorType) error {
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND connector_type = ?", userID, string(connectorType)).
		Delete(&dbschema.ConnectorConnection{}).Error

	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to delete connector connection")
	}

	return nil
}

// CreateOAuthState creates a new OAuth state.
func (r *Repository) CreateOAuthState(ctx context.Context, state *connector.OAuthState) (*connector.OAuthState, error) {
	model := dbschema.FromDomainOAuthState(state)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to create OAuth state")
	}
	return model.EtoD(), nil
}

// FindOAuthStateByState finds an OAuth state by state value.
func (r *Repository) FindOAuthStateByState(ctx context.Context, state string) (*connector.OAuthState, error) {
	var model dbschema.ConnectorOAuthState
	err := r.db.WithContext(ctx).
		Where("state = ?", state).
		First(&model).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to find OAuth state")
	}

	return model.EtoD(), nil
}

// DeleteOAuthState deletes an OAuth state by state value.
func (r *Repository) DeleteOAuthState(ctx context.Context, state string) error {
	err := r.db.WithContext(ctx).
		Where("state = ?", state).
		Delete(&dbschema.ConnectorOAuthState{}).Error

	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to delete OAuth state")
	}

	return nil
}

// DeleteExpiredOAuthStates deletes all expired OAuth states.
func (r *Repository) DeleteExpiredOAuthStates(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&dbschema.ConnectorOAuthState{})

	if result.Error != nil {
		return 0, platformerrors.AsError(ctx, platformerrors.LayerRepository, result.Error, "failed to delete expired OAuth states")
	}

	return result.RowsAffected, nil
}

// CreateAuditLog creates an audit log entry.
func (r *Repository) CreateAuditLog(ctx context.Context, log *connector.AuditLog) error {
	model := dbschema.FromDomainAuditLog(log)
	model.CreatedAt = time.Now()

	// Marshal metadata if present
	if log.Metadata != nil {
		data, err := json.Marshal(log.Metadata)
		if err == nil {
			model.Metadata = data
		}
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to create audit log")
	}

	return nil
}

// FindAuditLogsByUser finds audit logs for a user.
func (r *Repository) FindAuditLogsByUser(ctx context.Context, userID uint, limit int) ([]*connector.AuditLog, error) {
	if limit <= 0 {
		limit = 100
	}

	var models []dbschema.ConnectorAuditLog
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&models).Error

	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerRepository, err, "failed to find audit logs")
	}

	result := make([]*connector.AuditLog, 0, len(models))
	for _, m := range models {
		if domain := m.EtoD(); domain != nil {
			result = append(result, domain)
		}
	}

	return result, nil
}
