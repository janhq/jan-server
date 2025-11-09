package database

import (
	"context"

	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"jan-server/services/media-api/internal/infrastructure/database/entities"
)

// AutoMigrate applies database schema changes.
func AutoMigrate(ctx context.Context, db *gorm.DB, log zerolog.Logger) error {
	if err := db.WithContext(ctx).AutoMigrate(&entities.MediaObject{}); err != nil {
		return err
	}
	log.Info().Msg("applied media object migrations")
	return nil
}
