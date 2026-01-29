package usersettingsrepo

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"jan-server/mono/apps/backend/internal/domain/usersettings"
	"jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// UserSettingsGormRepository implements usersettings.Repository using GORM.
type UserSettingsGormRepository struct {
	db *gorm.DB
}

var _ usersettings.Repository = (*UserSettingsGormRepository)(nil)

// NewUserSettingsGormRepository constructs a new repository.
func NewUserSettingsGormRepository(db *gorm.DB) usersettings.Repository {
	return &UserSettingsGormRepository{db: db}
}

// FindByUserID retrieves user settings by user ID.
func (repo *UserSettingsGormRepository) FindByUserID(ctx context.Context, userID string) (*usersettings.UserSettings, error) {
	var entity dbschema.UserSettings
	err := repo.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&entity).
		Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to find user settings by user ID",
			err,
			"us-01",
		)
	}

	return entity.EtoD(), nil
}

// Upsert inserts or updates user settings.
func (repo *UserSettingsGormRepository) Upsert(ctx context.Context, settings *usersettings.UserSettings) (*usersettings.UserSettings, error) {
	entity := dbschema.NewSchemaUserSettings(settings)

	assignments := map[string]interface{}{
		"enable_trace":      entity.EnableTrace,
		"enable_tools":      entity.EnableTools,
		"memory_config":     entity.MemoryConfig,
		"profile_settings":  entity.ProfileSettings,
		"advanced_settings": entity.AdvancedSettings,
		"preferences":       entity.Preferences,
		"updated_at":        gorm.Expr("NOW()"),
	}

	err := repo.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(assignments),
		}).
		Create(entity).
		Error

	if err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to upsert user settings",
			err,
			"us-02",
		)
	}

	// Reload to get generated ID and timestamps
	var persisted dbschema.UserSettings
	if err := repo.db.WithContext(ctx).
		Where("user_id = ?", settings.UserID).
		First(&persisted).
		Error; err != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to reload upserted user settings",
			err,
			"us-03",
		)
	}

	return persisted.EtoD(), nil
}

// Update updates existing user settings.
func (repo *UserSettingsGormRepository) Update(ctx context.Context, settings *usersettings.UserSettings) error {
	entity := dbschema.NewSchemaUserSettings(settings)

	err := repo.db.WithContext(ctx).
		Model(&dbschema.UserSettings{}).
		Where("user_id = ?", settings.UserID).
		Updates(map[string]interface{}{
			"enable_trace":      entity.EnableTrace,
			"enable_tools":      entity.EnableTools,
			"memory_config":     entity.MemoryConfig,
			"profile_settings":  entity.ProfileSettings,
			"advanced_settings": entity.AdvancedSettings,
			"preferences":       entity.Preferences,
			"updated_at":        gorm.Expr("NOW()"),
		}).
		Error

	if err != nil {
		return platformerrors.NewError(
			ctx,
			platformerrors.LayerRepository,
			platformerrors.ErrorTypeDatabaseError,
			"failed to update user settings",
			err,
			"us-04",
		)
	}

	return nil
}
