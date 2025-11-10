package database

import (
	"context"

	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"jan-server/services/response-api/internal/infrastructure/database/entities"
)

// AutoMigrate applies database schema changes for the response domain.
func AutoMigrate(ctx context.Context, db *gorm.DB, log zerolog.Logger) error {
	if err := db.WithContext(ctx).AutoMigrate(
		&entities.Conversation{},
		&entities.ConversationItem{},
		&entities.Response{},
		&entities.ToolExecution{},
	); err != nil {
		return err
	}

	log.Info().Msg("database schema up to date")
	return nil
}
