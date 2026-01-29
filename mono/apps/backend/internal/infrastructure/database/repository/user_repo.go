package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"jan-server/mono/apps/backend/internal/domain/user"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"

	"gorm.io/gorm"
)

// UserRepository implements user.Repository using GORM.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Ensure UserRepository implements user.Repository.
var _ user.Repository = (*UserRepository)(nil)

// ============================================
// User operations
// ============================================

func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	schema := toUserSchema(u)
	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		return err
	}
	u.ID = schema.ID
	u.CreatedAt = schema.CreatedAt
	u.UpdatedAt = schema.UpdatedAt
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*user.User, error) {
	var schema dbschema.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return toUserDomain(&schema), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var schema dbschema.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return toUserDomain(&schema), nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	var schema dbschema.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return toUserDomain(&schema), nil
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	schema := toUserSchema(u)
	return r.db.WithContext(ctx).Model(&dbschema.User{}).Where("id = ?", u.ID).Updates(schema).Error
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&dbschema.User{}, "id = ?", id).Error
}

func (r *UserRepository) List(ctx context.Context, offset, limit int) ([]*user.User, int64, error) {
	var schemas []dbschema.User
	var total int64

	if err := r.db.WithContext(ctx).Model(&dbschema.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Order("created_at DESC").Find(&schemas).Error; err != nil {
		return nil, 0, err
	}

	users := make([]*user.User, len(schemas))
	for i, s := range schemas {
		users[i] = toUserDomain(&s)
	}

	return users, total, nil
}

// ============================================
// Refresh token operations
// ============================================

func (r *UserRepository) CreateRefreshToken(ctx context.Context, token *user.RefreshToken) error {
	schema := toRefreshTokenSchema(token)
	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		return err
	}
	token.ID = schema.ID
	token.CreatedAt = schema.CreatedAt
	return nil
}

func (r *UserRepository) GetRefreshTokenByHash(ctx context.Context, hash string) (*user.RefreshToken, error) {
	var schema dbschema.RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrInvalidToken
		}
		return nil, err
	}
	return toRefreshTokenDomain(&schema), nil
}

func (r *UserRepository) RevokeRefreshToken(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&dbschema.RefreshToken{}).
		Where("id = ?", id).
		Update("revoked_at", &now).Error
}

func (r *UserRepository) RevokeAllRefreshTokens(ctx context.Context, userID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&dbschema.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", &now).Error
}

func (r *UserRepository) DeleteExpiredRefreshTokens(ctx context.Context) error {
	return r.db.WithContext(ctx).Delete(&dbschema.RefreshToken{}, "expires_at < ?", time.Now()).Error
}

// ============================================
// API key operations
// ============================================

func (r *UserRepository) CreateAPIKey(ctx context.Context, key *user.APIKey) error {
	schema := toAPIKeySchema(key)
	if err := r.db.WithContext(ctx).Create(schema).Error; err != nil {
		return err
	}
	key.ID = schema.ID
	key.CreatedAt = schema.CreatedAt
	return nil
}

func (r *UserRepository) GetAPIKeyByHash(ctx context.Context, hash string) (*user.APIKey, error) {
	var schema dbschema.APIKey
	if err := r.db.WithContext(ctx).Where("key_hash = ?", hash).First(&schema).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrAPIKeyNotFound
		}
		return nil, err
	}
	return toAPIKeyDomain(&schema), nil
}

func (r *UserRepository) ListAPIKeysByUser(ctx context.Context, userID string) ([]*user.APIKey, error) {
	var schemas []dbschema.APIKey
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Order("created_at DESC").
		Find(&schemas).Error; err != nil {
		return nil, err
	}

	keys := make([]*user.APIKey, len(schemas))
	for i, s := range schemas {
		keys[i] = toAPIKeyDomain(&s)
	}

	return keys, nil
}

func (r *UserRepository) CountAPIKeysByUser(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&dbschema.APIKey{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Count(&count).Error
	return count, err
}

func (r *UserRepository) RevokeAPIKey(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&dbschema.APIKey{}).
		Where("id = ?", id).
		Update("revoked_at", &now).Error
}

func (r *UserRepository) UpdateAPIKeyLastUsed(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&dbschema.APIKey{}).
		Where("id = ?", id).
		Update("last_used_at", &now).Error
}

// ============================================
// Conversion helpers
// ============================================

func toUserSchema(u *user.User) *dbschema.User {
	isActive := u.IsActive
	isAdmin := u.IsAdmin
	return &dbschema.User{
		ID:           u.ID,
		Email:        u.Email,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Name:         u.Name,
		IsActive:     &isActive,
		IsAdmin:      &isAdmin,
		LastLoginAt:  u.LastLoginAt,
	}
}

func toUserDomain(s *dbschema.User) *user.User {
	isActive := true
	if s.IsActive != nil {
		isActive = *s.IsActive
	}
	isAdmin := false
	if s.IsAdmin != nil {
		isAdmin = *s.IsAdmin
	}
	return &user.User{
		ID:           s.ID,
		Email:        s.Email,
		Username:     s.Username,
		PasswordHash: s.PasswordHash,
		Name:         s.Name,
		IsActive:     isActive,
		IsAdmin:      isAdmin,
		LastLoginAt:  s.LastLoginAt,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

func toRefreshTokenSchema(t *user.RefreshToken) *dbschema.RefreshToken {
	return &dbschema.RefreshToken{
		ID:        t.ID,
		UserID:    t.UserID,
		TokenHash: t.TokenHash,
		ExpiresAt: t.ExpiresAt,
		RevokedAt: t.RevokedAt,
	}
}

func toRefreshTokenDomain(s *dbschema.RefreshToken) *user.RefreshToken {
	return &user.RefreshToken{
		ID:        s.ID,
		UserID:    s.UserID,
		TokenHash: s.TokenHash,
		ExpiresAt: s.ExpiresAt,
		RevokedAt: s.RevokedAt,
		CreatedAt: s.CreatedAt,
	}
}

func toAPIKeySchema(k *user.APIKey) *dbschema.APIKey {
	scopesJSON, _ := json.Marshal(k.Scopes)
	return &dbschema.APIKey{
		ID:         k.ID,
		UserID:     k.UserID,
		Name:       k.Name,
		KeyHash:    k.KeyHash,
		KeyPrefix:  k.KeyPrefix,
		Scopes:     string(scopesJSON),
		LastUsedAt: k.LastUsedAt,
		ExpiresAt:  k.ExpiresAt,
		RevokedAt:  k.RevokedAt,
	}
}

func toAPIKeyDomain(s *dbschema.APIKey) *user.APIKey {
	var scopes []string
	_ = json.Unmarshal([]byte(s.Scopes), &scopes)
	return &user.APIKey{
		ID:         s.ID,
		UserID:     s.UserID,
		Name:       s.Name,
		KeyHash:    s.KeyHash,
		KeyPrefix:  s.KeyPrefix,
		Scopes:     scopes,
		LastUsedAt: s.LastUsedAt,
		ExpiresAt:  s.ExpiresAt,
		RevokedAt:  s.RevokedAt,
		CreatedAt:  s.CreatedAt,
	}
}
