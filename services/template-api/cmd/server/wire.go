//go:build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"jan-server/services/template-api/internal/config"
	domain "jan-server/services/template-api/internal/domain/sample"
	"jan-server/services/template-api/internal/infrastructure/auth"
	"jan-server/services/template-api/internal/infrastructure/database"
	"jan-server/services/template-api/internal/infrastructure/logger"
	repo "jan-server/services/template-api/internal/infrastructure/repository/sample"
	"jan-server/services/template-api/internal/interfaces/httpserver"
)

var sampleSet = wire.NewSet(
	repo.NewPostgresRepository,
	wire.Bind(new(domain.Repository), new(*repo.PostgresRepository)),
	domain.NewService,
)

// BuildApplication demonstrates how to assemble the template service with Wire.
func BuildApplication(ctx context.Context) (*Application, error) {
	wire.Build(
		config.Load,
		logger.New,
		newDatabaseConfig,
		newGormDB,
		newAuthValidator,
		sampleSet,
		httpserver.New,
		NewApplication,
	)
	return nil, nil
}

func newDatabaseConfig(cfg *config.Config) database.Config {
	return database.Config{
		DSN:             cfg.DatabaseURL,
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

func newAuthValidator(ctx context.Context, cfg *config.Config, log zerolog.Logger) (*auth.Validator, error) {
	return auth.NewValidator(ctx, cfg, log)
}
