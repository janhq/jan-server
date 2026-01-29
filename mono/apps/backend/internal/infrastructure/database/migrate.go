package database

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	iofs "github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"

	"jan-server/mono/apps/backend/internal/infrastructure/logger"
	"jan-server/mono/apps/backend/migrations"
)

// AutoMigrate applies all pending SQL migrations bundled with the service.
func AutoMigrate(gormDB *gorm.DB) (err error) {
	log := logger.GetLogger()

	// List migration files
	log.Info().Msg("Scanning migration files...")
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("read migration directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			log.Info().Str("file", entry.Name()).Msg("Found migration file")
		}
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("retrieve sql db: %w", err)
	}

	// Ensure llm_api schema exists before running migrations
	if err := gormDB.Exec("CREATE SCHEMA IF NOT EXISTS llm_api").Error; err != nil {
		log.Warn().Err(err).Msg("Failed to create llm_api schema, may already exist")
	} else {
		log.Info().Msg("Created llm_api schema")
	}

	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		return fmt.Errorf("acquire dedicated connection: %w", err)
	}

	driver, err := postgres.WithConnection(context.Background(), conn, &postgres.Config{
		MigrationsTable: "schema_migrations",
		SchemaName:      "llm_api",
	})
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("initialize postgres driver: %w", err)
	}
	defer func() {
		if closeErr := driver.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close migration connection: %w", closeErr)
		}
	}()

	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	defer func() {
		if closeErr := source.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close migration source: %w", closeErr)
		}
	}()

	migrator, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	// Check current version and dirty state
	version, dirty, err := migrator.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		log.Warn().Err(err).Msg("Error getting migration version")
	} else if errors.Is(err, migrate.ErrNilVersion) {
		log.Info().Msg("No migrations have been applied yet")
	} else {
		log.Info().Uint("version", version).Bool("dirty", dirty).Msg("Current migration state")
	}

	// If database is dirty, force the version to allow re-running
	if dirty {
		log.Warn().Uint("version", version).Msg("Database is in dirty state, forcing version...")
		// Force to the current version to clear dirty state
		if forceErr := migrator.Force(int(version)); forceErr != nil {
			return fmt.Errorf("force version %d to clear dirty state: %w", version, forceErr)
		}
		log.Info().Msg("Dirty state cleared")
	}

	log.Info().Msg("Applying migrations...")
	err = migrator.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info().Msg("No new migrations to apply")
		} else {
			log.Error().Err(err).Msg("Failed to apply migrations")
			return fmt.Errorf("apply migrations: %w", err)
		}
	} else {
		log.Info().Msg("Migrations applied successfully")
	}

	// Get final version
	finalVersion, _, versionErr := migrator.Version()
	if versionErr == nil {
		log.Info().Uint("version", finalVersion).Msg("Current migration version")
	}

	return nil
}
