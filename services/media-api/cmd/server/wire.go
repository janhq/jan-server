//go:build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"jan-server/services/media-api/internal/config"
	domain "jan-server/services/media-api/internal/domain/media"
	"jan-server/services/media-api/internal/infrastructure/auth"
	"jan-server/services/media-api/internal/infrastructure/database"
	"jan-server/services/media-api/internal/infrastructure/logger"
	repo "jan-server/services/media-api/internal/infrastructure/repository/media"
	"jan-server/services/media-api/internal/infrastructure/storage"
	"jan-server/services/media-api/internal/interfaces/httpserver"
)

var mediaSet = wire.NewSet(
	repo.NewRepository,
	wire.Bind(new(domain.Repository), new(*repo.Repository)),
	storage.NewS3Storage,
	wire.Bind(new(domain.Storage), new(*storage.S3Storage)),
	domain.NewService,
)

// BuildApplication assembles the media API with Wire.
func BuildApplication(ctx context.Context) (*Application, error) {
	wire.Build(
		config.Load,
		logger.New,
		auth.NewValidator,
		newDatabaseConfig,
		newGormDB,
		mediaSet,
		httpserver.New,
		NewApplication,
	)
	return nil, nil
}

func newDatabaseConfig(cfg *config.Config) database.Config {
	return database.Config{
		DSN:             cfg.GetDatabaseWriteDSN(),
		MaxIdleConns:    cfg.DBMaxIdleConns,
		MaxOpenConns:    cfg.DBMaxOpenConns,
		ConnMaxLifetime: cfg.DBConnLifetime,
		LogLevel:        gormlogger.Warn,
	}
}

func newGormDB(ctx context.Context, cfg database.Config, log zerolog.Logger) (*gorm.DB, error) {
	db, err := database.Connect(cfg)
	if err != nil {
		return nil, err
	}
	if err := database.AutoMigrate(ctx, db, log); err != nil {
		return nil, err
	}
	return db, nil
}
