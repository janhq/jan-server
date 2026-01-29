package toolconfig

import (
	"context"
	"regexp"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"jan-server/mono/apps/backend/internal/infrastructure/llmapi"
)

// CachedTool holds a tool config with compiled regex patterns
type CachedTool struct {
	Config          llmapi.MCPToolConfig
	CompiledFilters []*regexp.Regexp
}

// Cache provides in-memory caching for MCP tool configurations
// with a 2-minute TTL as per requirements
type Cache struct {
	client      *llmapi.Client
	cacheTTL    time.Duration
	mu          sync.RWMutex
	tools       map[string]*CachedTool // keyed by tool_key
	allTools    []*CachedTool
	lastFetched time.Time
}

// NewCache creates a new tool config cache
func NewCache(client *llmapi.Client) *Cache {
	return &Cache{
		client:   client,
		cacheTTL: 2 * time.Minute,
		tools:    make(map[string]*CachedTool),
	}
}

// NewCacheWithTTL creates a cache with custom TTL (for testing)
func NewCacheWithTTL(client *llmapi.Client, ttl time.Duration) *Cache {
	return &Cache{
		client:   client,
		cacheTTL: ttl,
		tools:    make(map[string]*CachedTool),
	}
}

// GetAllTools returns all active tools from cache, refreshing if expired
func (c *Cache) GetAllTools(ctx context.Context) ([]*CachedTool, error) {
	c.mu.RLock()
	if time.Since(c.lastFetched) < c.cacheTTL && len(c.allTools) > 0 {
		log.Debug().
			Int("cached_count", len(c.allTools)).
			Dur("age", time.Since(c.lastFetched)).
			Msg("Returning cached tools (not expired)")
		tools := c.allTools
		c.mu.RUnlock()
		return tools, nil
	}
	c.mu.RUnlock()

	log.Debug().
		Dur("age", time.Since(c.lastFetched)).
		Bool("has_cached", len(c.allTools) > 0).
		Msg("Cache expired or empty, refreshing")
	return c.refresh(ctx)
}

// GetToolByKey returns a tool by its key from cache
func (c *Cache) GetToolByKey(ctx context.Context, toolKey string) (*CachedTool, error) {
	c.mu.RLock()
	if time.Since(c.lastFetched) < c.cacheTTL {
		tool := c.tools[toolKey]
		c.mu.RUnlock()
		return tool, nil
	}
	c.mu.RUnlock()

	// Refresh cache and try again
	if _, err := c.refresh(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	tool := c.tools[toolKey]
	c.mu.RUnlock()
	return tool, nil
}

// refresh fetches fresh data from LLM-API and updates the cache
func (c *Cache) refresh(ctx context.Context) ([]*CachedTool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(c.lastFetched) < c.cacheTTL && len(c.allTools) > 0 {
		return c.allTools, nil
	}

	log.Debug().Msg("Refreshing MCP tool config cache from LLM-API")

	configs, err := c.client.GetActiveMCPTools(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch MCP tools from LLM-API")
		// If we have stale data, return it rather than failing
		if len(c.allTools) > 0 {
			log.Warn().Msg("Using stale cache data")
			return c.allTools, nil
		}
		return nil, err
	}

	// Build new cache
	newTools := make(map[string]*CachedTool, len(configs))
	newAllTools := make([]*CachedTool, 0, len(configs))

	for _, cfg := range configs {
		cached := &CachedTool{
			Config:          cfg,
			CompiledFilters: compilePatterns(cfg.DisallowedKeywords),
		}
		newTools[cfg.ToolKey] = cached
		newAllTools = append(newAllTools, cached)
	}

	c.tools = newTools
	c.allTools = newAllTools
	c.lastFetched = time.Now()

	log.Info().
		Int("count", len(configs)).
		Msg("MCP tool config cache refreshed")

	return c.allTools, nil
}

// Invalidate forces a cache refresh on next access
func (c *Cache) Invalidate() {
	c.mu.Lock()
	c.lastFetched = time.Time{}
	c.mu.Unlock()
}

// compilePatterns compiles regex patterns, logging any invalid ones
func compilePatterns(patterns []string) []*regexp.Regexp {
	if len(patterns) == 0 {
		return nil
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Warn().
				Str("pattern", pattern).
				Err(err).
				Msg("Invalid regex pattern in disallowed_keywords, skipping")
			continue
		}
		compiled = append(compiled, re)
	}
	return compiled
}

// MatchesDisallowedKeyword checks if content matches any disallowed keyword pattern
func (ct *CachedTool) MatchesDisallowedKeyword(content string) bool {
	for _, re := range ct.CompiledFilters {
		if re.MatchString(content) {
			return true
		}
	}
	return false
}

// FilterSearchResults filters out results containing disallowed keywords
// Returns the filtered results and the count of removed items
func (ct *CachedTool) FilterSearchResults(results []string) ([]string, int) {
	if len(ct.CompiledFilters) == 0 {
		return results, 0
	}

	filtered := make([]string, 0, len(results))
	removed := 0

	for _, result := range results {
		if ct.MatchesDisallowedKeyword(result) {
			removed++
			log.Debug().
				Str("tool_key", ct.Config.ToolKey).
				Msg("Filtered result due to disallowed keyword match")
		} else {
			filtered = append(filtered, result)
		}
	}

	return filtered, removed
}
