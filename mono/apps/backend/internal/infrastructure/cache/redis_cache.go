package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const CacheVersion = "v1"

type RedisCache struct {
	client redis.UniversalClient
	rs     *redsync.Redsync
}

func NewRedisCache(redisURL string) (*RedisCache, error) {
	if redisURL == "" {
		return nil, fmt.Errorf("Redis URL must be provided")
	}

	opts, err := buildUniversalOptions(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	if len(opts.Addrs) > 1 && opts.DB != 0 {
		log.Warn().Msg("Ignoring non-zero DB when using Redis Cluster configuration")
		opts.DB = 0
	}

	client := redis.NewUniversalClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Info().Msg("Successfully connected to Redis cache")
	rs := redsync.New(goredis.NewPool(client))
	return &RedisCache{
		client: client,
		rs:     rs,
	}, nil
}

func buildUniversalOptions(raw string) (*redis.UniversalOptions, error) {
	parts := strings.Split(raw, ",")
	opts := &redis.UniversalOptions{}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "://") {
			parsed, err := redis.ParseURL(part)
			if err != nil {
				return nil, err
			}

			opts.Addrs = append(opts.Addrs, parsed.Addr)

			if opts.Username == "" {
				opts.Username = parsed.Username
			}

			if opts.Password == "" {
				opts.Password = parsed.Password
			}

			if opts.DB == 0 {
				opts.DB = parsed.DB
			}

			if opts.TLSConfig == nil {
				opts.TLSConfig = parsed.TLSConfig
			}

			if opts.ReadTimeout == 0 {
				opts.ReadTimeout = parsed.ReadTimeout
			}

			if opts.WriteTimeout == 0 {
				opts.WriteTimeout = parsed.WriteTimeout
			}

			if opts.DialTimeout == 0 {
				opts.DialTimeout = parsed.DialTimeout
			}

			if opts.PoolSize == 0 {
				opts.PoolSize = parsed.PoolSize
			}

			if opts.MinIdleConns == 0 {
				opts.MinIdleConns = parsed.MinIdleConns
			}
		} else {
			opts.Addrs = append(opts.Addrs, part)
		}
	}

	if len(opts.Addrs) == 0 {
		return nil, fmt.Errorf("no Redis addresses provided")
	}

	return opts, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisCache) SetWithTimeout(ctx context.Context, key string, value string, expiration time.Duration, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return r.client.Set(timeoutCtx, key, value, expiration).Err()
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		// Cache miss is a normal condition in cache-aside pattern - return redis.Nil as-is
		// Callers should check with errors.Is(err, redis.Nil)
		if err == redis.Nil {
			return "", redis.Nil
		}
		return "", fmt.Errorf("failed to get value from cache: %w", err)
	}

	return val, nil
}

func GetJSON[T any](ctx context.Context, rdb *RedisCache, key string) (*T, error) {
	val, err := rdb.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var obj T
	if unmarshalErr := json.Unmarshal([]byte(val), &obj); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON from cache: %w", unmarshalErr)
	}
	return &obj, nil
}

func (r *RedisCache) GetWithFallback(ctx context.Context, key string, fallback func() (string, error), expiration time.Duration) (string, error) {
	result, err := r.Get(ctx, key)
	if err == nil {
		return result, nil
	}

	result, err = fallback()
	if err != nil {
		return "", fmt.Errorf("fallback function failed: %w", err)
	}

	if err := r.Set(ctx, key, result, expiration); err != nil {
		log.Error().Err(err).Msg("Failed to cache value")
	}

	return result, nil
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisCache) Unlink(ctx context.Context, key string) error {
	return r.client.Unlink(ctx, key).Err()
}

func (r *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	var cursor uint64
	for {
		keys, next, err := r.client.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}
		if len(keys) > 0 {
			pipe := r.client.Pipeline()
			for _, k := range keys {
				pipe.Unlink(ctx, k)
			}
			if _, err := pipe.Exec(ctx); err != nil {
				return fmt.Errorf("failed to unlink keys: %w", err)
			}
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	return nil
}

func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}
	return result > 0, nil
}

func (r *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

func (r *RedisCache) IncrWithTimeout(ctx context.Context, key string, timeout time.Duration) (int64, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return r.Incr(timeoutCtx, key)
}

func (r *RedisCache) Expires(ctx context.Context, key string, duration time.Duration) error {
	return r.client.Expire(ctx, key, duration).Err()
}

func (r *RedisCache) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (any, error) {
	return r.client.EvalSha(ctx, sha1, keys, args...).Result()
}

func (r *RedisCache) ScriptLoad(ctx context.Context, script string) (string, error) {
	return r.client.ScriptLoad(ctx, script).Result()
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}

func (r *RedisCache) HealthCheck(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func WithLock(cache *RedisCache, lockName string, ttl time.Duration, fn func() error) error {
	mutex := cache.rs.NewMutex(lockName, redsync.WithExpiry(ttl))

	if err := mutex.Lock(); err != nil {
		return err
	}

	defer func() {
		if _, err := mutex.Unlock(); err != nil {
			log.Error().Err(err).Msg("Failed to unlock mutex")
		}
	}()

	return fn()
}
