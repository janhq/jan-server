package main

import (
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"jan-server/services/llm-api/internal/infrastructure/database"
	_ "jan-server/services/llm-api/internal/infrastructure/database/dbschema"
)

var GormGenerator *gen.Generator

func init() {
	// Get database DSN from environment - fail fast if not set
	databaseDSN := os.Getenv("DB_POSTGRESQL_WRITE_DSN")
	if databaseDSN == "" {
		panic("DB_POSTGRESQL_WRITE_DSN environment variable is required")
	}

	// Connect directly without table prefix for schema inspection
	db, err := gorm.Open(postgres.Open(databaseDSN), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false,
		},
	})
	if err != nil {
		panic(err)
	}

	GormGenerator = gen.NewGenerator(gen.Config{
		OutPath:       "./internal/infrastructure/database/gormgen",
		Mode:          gen.WithDefaultQuery | gen.WithQueryInterface | gen.WithoutContext,
		FieldNullable: true,
	})
	GormGenerator.UseDB(db)
}

func main() {
	for _, model := range database.SchemaRegistry {
		GormGenerator.ApplyBasic(model)
		type Querier interface {
		}
		GormGenerator.ApplyInterface(func(Querier) {}, model)
	}
	GormGenerator.Execute()
}
