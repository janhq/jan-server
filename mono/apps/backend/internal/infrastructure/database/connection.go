package database

import (
	"fmt"
	"time"

	"jan-server/mono/apps/backend/internal/infrastructure/config"
	"jan-server/mono/apps/backend/internal/infrastructure/database/registry"

	// Import dbschema to register schemas via init()
	_ "jan-server/mono/apps/backend/internal/infrastructure/database/dbschema"

	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewConnection creates a new database connection with the provided configuration.
func NewConnection(cfg *config.Config) (*gorm.DB, error) {
	gormConfig := &gorm.Config{
		Logger:                 newGormLogger(cfg),
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	}

	db, err := gorm.Open(postgres.Open(cfg.GetDatabaseWriteDSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().Msg("database connection established")

	return db, nil
}

// RunMigrations applies database migrations.
func RunMigrations(db *gorm.DB, cfg *config.Config) error {
	log.Info().Msg("running database migrations")

	// Auto-migrate all registered schemas
	for _, model := range registry.SchemaRegistry {
		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("auto-migrate failed for %T: %w", model, err)
		}
	}

	log.Info().Msg("database migrations completed")
	return nil
}

func newGormLogger(cfg *config.Config) logger.Interface {
	logLevel := logger.Silent
	switch cfg.LogLevel {
	case "debug":
		logLevel = logger.Info
	case "info":
		logLevel = logger.Warn
	case "warn", "error":
		logLevel = logger.Error
	}

	return logger.New(
		&zerologWriter{},
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)
}

type zerologWriter struct{}

func (w *zerologWriter) Printf(format string, args ...interface{}) {
	log.Debug().Msgf(format, args...)
}
