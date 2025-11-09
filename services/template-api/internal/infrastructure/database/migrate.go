package database

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"jan-server/services/template-api/internal/infrastructure/database/entities"
)

// AutoMigrate applies database schema changes and seeds baseline rows.
func AutoMigrate(ctx context.Context, db *gorm.DB, log zerolog.Logger) error {
	if err := db.WithContext(ctx).AutoMigrate(&entities.Sample{}); err != nil {
		return err
	}

	var count int64
	if err := db.Model(&entities.Sample{}).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		defaultSample := entities.Sample{
			ID:      uuid.NewString(),
			Message: "Hello from PostgreSQL! Replace this with your own repository logic.",
		}
		if err := db.Create(&defaultSample).Error; err != nil {
			return err
		}
		log.Info().Str("sample_id", defaultSample.ID).Msg("seeded default sample row")
	} else {
		log.Debug().Int64("rows", count).Msg("sample table already seeded")
	}

	return nil
}
