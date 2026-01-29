package userrepo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"jan-server/mono/apps/backend/internal/domain/user"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// LocalUserRepository implements user.Repository for local authentication.
type LocalUserRepository struct {
	db *gorm.DB
}

var _ user.Repository = (*LocalUserRepository)(nil)

// NewLocalUserRepository creates a new local user repository.
func NewLocalUserRepository(db *gorm.DB) user.Repository {
	return &LocalUserRepository{db: db}
}

// Create creates a new user.
func (r *LocalUserRepository) Create(ctx context.Context, u *user.User) error {
	schema := &dbschema.LocalUser{
		ID:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Name:         u.Name,
		IsActive:     boolPtr(u.IsActive),
		IsAdmin:      boolPtr(u.IsAdmin),
		IsGuest:      boolPtr(u.IsGuest),
		LastLoginAt:  u.LastLoginAt,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}

	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return user.ErrUserAlreadyExists
		}
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to create user",
			err,
			"a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		)
	}
	return nil
}

// GetByID retrieves a user by ID.
func (r *LocalUserRepository) GetByID(ctx context.Context, id string) (*user.User, error) {
	var schema dbschema.LocalUser
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to get user by ID",
			err,
			"b2c3d4e5-f6a7-8901-bcde-f12345678901",
		)
	}
	return schema.ToDomain(), nil
}

// GetByEmail retrieves a user by email.
func (r *LocalUserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var schema dbschema.LocalUser
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to get user by email",
			err,
			"c3d4e5f6-a7b8-9012-cdef-123456789012",
		)
	}
	return schema.ToDomain(), nil
}

// GetByUsername retrieves a user by username.
func (r *LocalUserRepository) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	var schema dbschema.LocalUser
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to get user by username",
			err,
			"d4e5f6a7-b8c9-0123-def0-234567890123",
		)
	}
	return schema.ToDomain(), nil
}

// Update updates a user.
func (r *LocalUserRepository) Update(ctx context.Context, u *user.User) error {
	updates := map[string]interface{}{
		"username":      u.Username,
		"email":         u.Email,
		"password_hash": u.PasswordHash,
		"name":          u.Name,
		"is_active":     u.IsActive,
		"is_admin":      u.IsAdmin,
		"is_guest":      u.IsGuest,
		"last_login_at": u.LastLoginAt,
		"updated_at":    time.Now(),
	}

	result := r.db.WithContext(ctx).Model(&dbschema.LocalUser{}).Where("id = ?", u.ID).Updates(updates)
	if result.Error != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to update user",
			result.Error,
			"e5f6a7b8-c9d0-1234-ef01-345678901234",
		)
	}
	if result.RowsAffected == 0 {
		return user.ErrUserNotFound
	}
	return nil
}

// Delete deletes a user.
func (r *LocalUserRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&dbschema.LocalUser{}, "id = ?", id)
	if result.Error != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to delete user",
			result.Error,
			"f6a7b8c9-d0e1-2345-f012-456789012345",
		)
	}
	return nil
}

// List lists users with pagination.
func (r *LocalUserRepository) List(ctx context.Context, offset, limit int) ([]*user.User, int64, error) {
	var schemas []dbschema.LocalUser
	var total int64

	if err := r.db.WithContext(ctx).Model(&dbschema.LocalUser{}).Count(&total).Error; err != nil {
		return nil, 0, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to count users",
			err,
			"a7b8c9d0-e1f2-3456-0123-567890123456",
		)
	}

	if err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&schemas).Error; err != nil {
		return nil, 0, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to list users",
			err,
			"b8c9d0e1-f234-5670-1234-678901234567",
		)
	}

	users := make([]*user.User, len(schemas))
	for i, s := range schemas {
		users[i] = s.ToDomain()
	}
	return users, total, nil
}

// CreateRefreshToken creates a new refresh token.
func (r *LocalUserRepository) CreateRefreshToken(ctx context.Context, token *user.RefreshToken) error {
	schema := &dbschema.RefreshToken{
		ID:        token.ID,
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
		RevokedAt: token.RevokedAt,
		CreatedAt: token.CreatedAt,
	}

	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to create refresh token",
			err,
			"c9d0e1f2-3456-7801-2345-789012345678",
		)
	}
	return nil
}

// GetRefreshTokenByHash retrieves a refresh token by its hash.
func (r *LocalUserRepository) GetRefreshTokenByHash(ctx context.Context, hash string) (*user.RefreshToken, error) {
	var schema dbschema.RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrInvalidToken
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to get refresh token",
			err,
			"d0e1f234-5678-9012-3456-890123456789",
		)
	}
	return schema.ToDomain(), nil
}

// RevokeRefreshToken revokes a refresh token.
func (r *LocalUserRepository) RevokeRefreshToken(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&dbschema.RefreshToken{}).Where("id = ?", id).Update("revoked_at", &now)
	if result.Error != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to revoke refresh token",
			result.Error,
			"e1f23456-7890-1234-5678-901234567890",
		)
	}
	return nil
}

// RevokeAllRefreshTokens revokes all refresh tokens for a user.
func (r *LocalUserRepository) RevokeAllRefreshTokens(ctx context.Context, userID string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&dbschema.RefreshToken{}).Where("user_id = ? AND revoked_at IS NULL", userID).Update("revoked_at", &now)
	if result.Error != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to revoke all refresh tokens",
			result.Error,
			"f2345678-9012-3456-7890-123456789012",
		)
	}
	return nil
}

// DeleteExpiredRefreshTokens deletes all expired refresh tokens.
func (r *LocalUserRepository) DeleteExpiredRefreshTokens(ctx context.Context) error {
	result := r.db.WithContext(ctx).Delete(&dbschema.RefreshToken{}, "expires_at < ?", time.Now())
	if result.Error != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to delete expired refresh tokens",
			result.Error,
			"01234567-8901-2345-6789-012345678901",
		)
	}
	return nil
}

// CreateAPIKey creates a new API key.
func (r *LocalUserRepository) CreateAPIKey(ctx context.Context, key *user.APIKey) error {
	schema := &dbschema.LocalAPIKey{
		ID:         key.ID,
		UserID:     key.UserID,
		Name:       key.Name,
		KeyHash:    key.KeyHash,
		KeyPrefix:  key.KeyPrefix,
		ExpiresAt:  key.ExpiresAt,
		RevokedAt:  key.RevokedAt,
		LastUsedAt: key.LastUsedAt,
		CreatedAt:  key.CreatedAt,
	}

	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to create API key",
			err,
			"12345678-9012-3456-7890-123456789012",
		)
	}
	return nil
}

// GetAPIKeyByHash retrieves an API key by its hash.
func (r *LocalUserRepository) GetAPIKeyByHash(ctx context.Context, hash string) (*user.APIKey, error) {
	var schema dbschema.LocalAPIKey
	if err := r.db.WithContext(ctx).Where("key_hash = ?", hash).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrAPIKeyNotFound
		}
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to get API key",
			err,
			"23456789-0123-4567-8901-234567890123",
		)
	}
	return schema.ToDomain(), nil
}

// ListAPIKeysByUser lists all API keys for a user.
func (r *LocalUserRepository) ListAPIKeysByUser(ctx context.Context, userID string) ([]*user.APIKey, error) {
	var schemas []dbschema.LocalAPIKey
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&schemas).Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to list API keys",
			err,
			"34567890-1234-5678-9012-345678901234",
		)
	}

	keys := make([]*user.APIKey, len(schemas))
	for i, s := range schemas {
		keys[i] = s.ToDomain()
	}
	return keys, nil
}

// CountAPIKeysByUser counts the number of active (non-revoked) API keys for a user.
func (r *LocalUserRepository) CountAPIKeysByUser(ctx context.Context, userID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&dbschema.LocalAPIKey{}).Where("user_id = ? AND revoked_at IS NULL", userID).Count(&count).Error; err != nil {
		return 0, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to count API keys",
			err,
			"45678901-2345-6789-0123-456789012345",
		)
	}
	return count, nil
}

// RevokeAPIKey revokes an API key.
func (r *LocalUserRepository) RevokeAPIKey(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&dbschema.LocalAPIKey{}).Where("id = ?", id).Update("revoked_at", &now)
	if result.Error != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to revoke API key",
			result.Error,
			"56789012-3456-7890-1234-567890123456",
		)
	}
	return nil
}

// UpdateAPIKeyLastUsed updates the last used timestamp of an API key.
func (r *LocalUserRepository) UpdateAPIKeyLastUsed(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&dbschema.LocalAPIKey{}).Where("id = ?", id).Update("last_used_at", &now)
	if result.Error != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to update API key last used",
			result.Error,
			"67890123-4567-8901-2345-678901234567",
		)
	}
	return nil
}

func boolPtr(b bool) *bool {
	return &b
}
