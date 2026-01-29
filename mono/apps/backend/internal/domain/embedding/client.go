package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/rs/zerolog/log"

	"jan-server/mono/apps/backend/internal/infrastructure/cache"
)

// Cache interface for embedding storage
type Cache interface {
	Get(key string) ([]float32, bool)
	Set(key string, value []float32, ttl time.Duration)
}

type CacheConfig struct {
	Type      string // "redis", "memory", "noop"
	RedisURL  string
	KeyPrefix string
	MaxSize   int
	TTL       time.Duration
}

// Cache implementations

// 1. Redis Cache (recommended for production) - using infrastructure cache
type RedisCache struct {
	cache *cache.EmbeddingCache
}

func NewRedisCache(redisURL, prefix string, ttl time.Duration) (*RedisCache, error) {
	embCache, err := cache.NewEmbeddingCache(redisURL, prefix, ttl)
	if err != nil {
		return nil, fmt.Errorf("create embedding cache: %w", err)
	}

	return &RedisCache{
		cache: embCache,
	}, nil
}

func (c *RedisCache) Get(key string) ([]float32, bool) {
	return c.cache.Get(key)
}

func (c *RedisCache) Set(key string, value []float32, ttl time.Duration) {
	c.cache.Set(key, value, ttl)
}

// 2. In-Memory LRU Cache (alternative, no Redis required)
type MemoryCache struct {
	cache *lru.Cache
	ttl   time.Duration
	mu    sync.RWMutex
}

type cacheEntry struct {
	value     []float32
	expiresAt time.Time
}

func NewMemoryCache(maxSize int, ttl time.Duration) (*MemoryCache, error) {
	cache, err := lru.New(maxSize)
	if err != nil {
		return nil, err
	}

	return &MemoryCache{
		cache: cache,
		ttl:   ttl,
	}, nil
}

func (c *MemoryCache) Get(key string) ([]float32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, found := c.cache.Get(key)
	if !found {
		return nil, false
	}

	entry := val.(cacheEntry)
	if time.Now().After(entry.expiresAt) {
		// Expired
		c.cache.Remove(key)
		return nil, false
	}

	return entry.value, true
}

func (c *MemoryCache) Set(key string, value []float32, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	c.cache.Add(key, entry)
}

// 3. NoOps Cache (disable caching)
type NoOpsCache struct{}

func NewNoOpsCache() *NoOpsCache {
	return &NoOpsCache{}
}

func (c *NoOpsCache) Get(key string) ([]float32, bool) {
	return nil, false // Always cache miss
}

func (c *NoOpsCache) Set(key string, value []float32, ttl time.Duration) {
	// Do nothing
}

// Cache factory
func NewCache(config CacheConfig) (Cache, error) {
	switch config.Type {
	case "redis":
		return NewRedisCache(config.RedisURL, config.KeyPrefix, config.TTL)
	case "memory":
		return NewMemoryCache(config.MaxSize, config.TTL)
	case "noop":
		return NewNoOpsCache(), nil
	default:
		return nil, fmt.Errorf("unknown cache type: %s", config.Type)
	}
}

// Client interface and implementation

type Client interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	EmbedSingle(ctx context.Context, text string) ([]float32, error)
	EmbedSparse(ctx context.Context, texts []string) ([]SparseEmbedding, error)
	ValidateServer(ctx context.Context) error
}

type BGE_M3_Client struct {
	baseURL    string
	httpClient *http.Client
	cache      Cache
}

type EmbedRequest struct {
	Inputs    interface{} `json:"inputs"` // string or []string
	Normalize bool        `json:"normalize"`
	Truncate  bool        `json:"truncate"`
}

type EmbedResponse [][]float32

type SparseEmbedding struct {
	Indices []int     `json:"indices"`
	Values  []float32 `json:"values"`
}

type ModelInfo struct {
	ModelID string `json:"model_id"`
}

func NewBGE_M3_Client(baseURL string, cacheConfig CacheConfig) (*BGE_M3_Client, error) {
	cache, err := NewCache(cacheConfig)
	if err != nil {
		return nil, fmt.Errorf("initialize cache: %w", err)
	}

	return &BGE_M3_Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: cache,
	}, nil
}

func (c *BGE_M3_Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// Check cache first
	cachedResults := make([][]float32, len(texts))
	uncachedIndices := []int{}
	uncachedTexts := []string{}

	for i, text := range texts {
		if cached, found := c.cache.Get(text); found {
			cachedResults[i] = cached
		} else {
			uncachedIndices = append(uncachedIndices, i)
			uncachedTexts = append(uncachedTexts, text)
		}
	}

	if len(uncachedTexts) == 0 {
		return cachedResults, nil
	}

	// Call BGE-M3 API for uncached items
	reqBody := EmbedRequest{
		Inputs:    uncachedTexts,
		Normalize: true,
		Truncate:  true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embed", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding service returned status %d", resp.StatusCode)
	}

	var embeddings EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddings); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Merge results and cache
	for i, idx := range uncachedIndices {
		cachedResults[idx] = embeddings[i]
		c.cache.Set(uncachedTexts[i], embeddings[i], 1*time.Hour)
	}

	return cachedResults, nil
}

func (c *BGE_M3_Client) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := c.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil
}

func (c *BGE_M3_Client) EmbedSparse(ctx context.Context, texts []string) ([]SparseEmbedding, error) {
	reqBody := EmbedRequest{
		Inputs:   texts,
		Truncate: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embed_sparse", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding service returned status %d", resp.StatusCode)
	}

	var sparseEmbeddings [][]struct {
		Index int     `json:"index"`
		Value float32 `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sparseEmbeddings); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert to SparseEmbedding format
	result := make([]SparseEmbedding, len(sparseEmbeddings))
	for i, sparse := range sparseEmbeddings {
		indices := make([]int, len(sparse))
		values := make([]float32, len(sparse))
		for j, sv := range sparse {
			indices[j] = sv.Index
			values[j] = sv.Value
		}
		result[i] = SparseEmbedding{
			Indices: indices,
			Values:  values,
		}
	}

	return result, nil
}

func (c *BGE_M3_Client) ValidateServer(ctx context.Context) error {
	// 1. Check health endpoint
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil || resp.StatusCode != 200 {
		return fmt.Errorf("embedding server not healthy")
	}
	resp.Body.Close()

	// 2. Check model info
	resp, err = c.httpClient.Get(c.baseURL + "/info")
	if err != nil {
		return fmt.Errorf("failed to get model info: %w", err)
	}
	defer resp.Body.Close()

	var info ModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return fmt.Errorf("failed to decode model info: %w", err)
	}

	// 3. Verify it's BGE-M3
	if info.ModelID != "BAAI/bge-m3" {
		log.Warn().Str("model", info.ModelID).Msg("Expected BGE-M3, got different model")
	}

	// 4. Test embedding
	embeddings, err := c.Embed(ctx, []string{"test"})
	if err != nil {
		return fmt.Errorf("test embedding failed: %w", err)
	}
	if len(embeddings) == 0 || len(embeddings[0]) != 1024 {
		return fmt.Errorf("expected 1024 dimensions, got %d", len(embeddings[0]))
	}

	return nil
}
