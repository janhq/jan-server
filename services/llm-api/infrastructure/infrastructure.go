package infrastructure

import (
	"jan-server/services/llm-api/infrastructure/cache"
	"jan-server/services/llm-api/infrastructure/provider"

	"gorm.io/gorm"
)

// Infrastructure holds all infrastructure dependencies
type Infrastructure struct {
	DB       *gorm.DB
	Registry *provider.Registry
	Cache    cache.Cache // Interface for future cache implementations
}

// NewInfrastructure creates a new infrastructure instance
func NewInfrastructure(
	db *gorm.DB,
	registry *provider.Registry,
	cache cache.Cache,
) *Infrastructure {
	return &Infrastructure{
		DB:       db,
		Registry: registry,
		Cache:    cache,
	}
}
