package db

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	iofs "github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrations embed.FS

// AutoMigrate applies all pending SQL migrations bundled with the service.
func AutoMigrate(gormDB *gorm.DB) (err error) {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("retrieve sql db: %w", err)
	}

	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		return fmt.Errorf("acquire dedicated connection: %w", err)
	}

	driver, err := postgres.WithConnection(context.Background(), conn, &postgres.Config{})
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("initialize postgres driver: %w", err)
	}
	defer func() {
		if closeErr := driver.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close migration connection: %w", closeErr)
		}
	}()

	source, err := iofs.New(migrations, "migrations")
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

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
