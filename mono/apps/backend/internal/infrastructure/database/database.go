package database

import (
	"fmt"
	"time"

	"jan-server/mono/apps/backend/internal/infrastructure/database/registry"
	"jan-server/mono/apps/backend/internal/infrastructure/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

// Config holds database configuration
type Config struct {
	DatabaseURL string
	MaxIdle     int
	MaxOpen     int
	MaxLifetime time.Duration
	LogLevel    gormlogger.LogLevel
}

// Connect creates a new database connection with the given configuration
func Connect(cfg Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "llm_api.",
			SingularTable: false,
		},
		Logger: gormlogger.Default.LogMode(cfg.LogLevel),
	})
	if err != nil {
		log := logger.GetLogger()
		log.Error().
			Str("error_code", "5c16fb53-d98c-4fc6-8bb4-9abd3c0b9e88").
			Err(err).
			Msg("unable to connect to database")
		return nil, err
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdle)
	sqlDB.SetMaxOpenConns(cfg.MaxOpen)
	sqlDB.SetConnMaxLifetime(cfg.MaxLifetime)

	log := logger.GetLogger()
	log.Info().Msg("Successfully connected to database")
	DB = db
	return DB, nil
}

// NewDB creates a new database connection using DSN
func NewDB(dsn string) (*gorm.DB, error) {
	return Connect(Config{
		DatabaseURL: dsn,
		MaxIdle:     10,
		MaxOpen:     25,
		MaxLifetime: 1 * time.Hour,
		LogLevel:    gormlogger.Silent,
	})
}

type DatabaseMigration struct {
	gorm.Model
	Version string `gorm:"not null;uniqueIndex"`
}

func Migration(db *gorm.DB, tablePrefix string) error {
	schemaName := "llm_api"
	if tablePrefix != "" {
		// Extract schema from table prefix (e.g., "llm_api." -> "llm_api")
		if len(tablePrefix) > 0 && tablePrefix[len(tablePrefix)-1] == '.' {
			schemaName = tablePrefix[:len(tablePrefix)-1]
		}
	}

	if err := db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", schemaName)).Error; err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	hasTable := db.Migrator().HasTable(&DatabaseMigration{})
	if !hasTable {
		if err := db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE;", schemaName)).Error; err != nil {
			return fmt.Errorf("failed to drop %s schema: %w", schemaName, err)
		}
		if err := db.Exec(fmt.Sprintf("CREATE SCHEMA %s;", schemaName)).Error; err != nil {
			return fmt.Errorf("failed to create %s schema: %w", schemaName, err)
		}
		if err := db.AutoMigrate(&DatabaseMigration{}); err != nil {
			return fmt.Errorf("failed to create 'database_migration' table: %w", err)
		}
		for _, model := range registry.SchemaRegistry {
			err := db.AutoMigrate(model)
			if err != nil {
				log := logger.GetLogger()
				log.Error().
					Str("error_code", "75333e43-8157-4f0a-8e34-aa34e6e7c285").
					Err(err).
					Msgf("failed to auto migrate schema: %T", model)
				return err
			}
		}
	}
	return nil
}
