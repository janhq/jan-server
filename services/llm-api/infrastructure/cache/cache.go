package cache

import (
	"context"
	"time"
)

// Cache defines the interface for cache operations
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// NoOpCache is a cache implementation that does nothing (for when cache is disabled)
type NoOpCache struct{}

func NewNoOpCache() Cache {
	return &NoOpCache{}
}

func (c *NoOpCache) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (c *NoOpCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return nil
}

func (c *NoOpCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (c *NoOpCache) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}
