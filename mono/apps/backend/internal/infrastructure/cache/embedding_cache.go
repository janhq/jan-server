package cache

import (
	"context"
	"encoding/binary"
	"math"
	"time"
)

// EmbeddingCache wraps RedisCache for embedding-specific operations
type EmbeddingCache struct {
	cache     *RedisCache
	keyPrefix string
	ttl       time.Duration
}

// NewEmbeddingCache creates a new embedding cache
func NewEmbeddingCache(redisURL, keyPrefix string, ttl time.Duration) (*EmbeddingCache, error) {
	cache, err := NewRedisCache(redisURL)
	if err != nil {
		return nil, err
	}

	return &EmbeddingCache{
		cache:     cache,
		keyPrefix: keyPrefix,
		ttl:       ttl,
	}, nil
}

// Get retrieves an embedding from cache
func (c *EmbeddingCache) Get(key string) ([]float32, bool) {
	ctx := context.Background()
	data, err := c.cache.client.Get(ctx, c.keyPrefix+key).Bytes()
	if err != nil {
		return nil, false
	}

	// Deserialize float32 array
	embedding := make([]float32, len(data)/4)
	for i := range embedding {
		bits := binary.LittleEndian.Uint32(data[i*4:])
		embedding[i] = math.Float32frombits(bits)
	}

	return embedding, true
}

// Set stores an embedding in cache
func (c *EmbeddingCache) Set(key string, value []float32, ttl time.Duration) {
	ctx := context.Background()

	// Serialize float32 array
	data := make([]byte, len(value)*4)
	for i, f := range value {
		bits := math.Float32bits(f)
		binary.LittleEndian.PutUint32(data[i*4:], bits)
	}

	if ttl == 0 {
		ttl = c.ttl
	}

	c.cache.client.Set(ctx, c.keyPrefix+key, data, ttl)
}

// Delete removes an embedding from cache
func (c *EmbeddingCache) Delete(key string) error {
	ctx := context.Background()
	return c.cache.Delete(ctx, c.keyPrefix+key)
}

// Clear removes all cached embeddings with the prefix
func (c *EmbeddingCache) Clear() error {
	ctx := context.Background()
	return c.cache.DeletePattern(ctx, c.keyPrefix+"*")
}

// Stats returns cache statistics
func (c *EmbeddingCache) Stats() (map[string]interface{}, error) {
	ctx := context.Background()

	info, err := c.cache.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	// Count keys with our prefix
	var count int64
	iter := c.cache.client.Scan(ctx, 0, c.keyPrefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		count++
	}

	return map[string]interface{}{
		"type":       "redis",
		"key_prefix": c.keyPrefix,
		"key_count":  count,
		"ttl":        c.ttl.String(),
		"info":       info,
	}, nil
}

// Close closes the underlying Redis connection
func (c *EmbeddingCache) Close() error {
	return c.cache.Close()
}

// HealthCheck checks if the cache is healthy
func (c *EmbeddingCache) HealthCheck(ctx context.Context) error {
	return c.cache.HealthCheck(ctx)
}
