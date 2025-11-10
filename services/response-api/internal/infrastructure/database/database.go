package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Config controls GORM/PostgreSQL connectivity.
type Config struct {
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	LogLevel        gormlogger.LogLevel
}

// Connect initializes a GORM connection using the provided config.
func Connect(cfg Config) (*gorm.DB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("database DSN is empty")
	}

	if err := ensureDatabaseExists(cfg.DSN); err != nil {
		return nil, fmt.Errorf("ensure database: %w", err)
	}

	if cfg.LogLevel == 0 {
		cfg.LogLevel = gormlogger.Warn
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		PrepareStmt: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		Logger: gormlogger.Default.LogMode(cfg.LogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("retrieve sql db: %w", err)
	}

	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	return db, nil
}

func ensureDatabaseExists(dsn string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil // non-URL formats are ignored
	}

	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" || dbName == "postgres" {
		return nil
	}

	adminURL := *u
	adminURL.Path = "/postgres"

	sqlDB, err := sql.Open("postgres", adminURL.String())
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	var exists bool
	err = sqlDB.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if exists {
		return nil
	}

	_, err = sqlDB.Exec("CREATE DATABASE " + pqQuoteIdentifier(dbName))
	return err
}

func pqQuoteIdentifier(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}
