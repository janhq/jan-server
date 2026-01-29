package middlewares

import (
	"net"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// simple token bucket per key (principal or IP).
type rateBucket struct {
	tokens     float64
	lastRefill time.Time
}

// RateLimitMiddleware limits requests per key within a fixed window.
func RateLimitMiddleware(limitPerMinute float64) gin.HandlerFunc {
	var (
		mu      sync.Mutex
		buckets = make(map[string]*rateBucket)
		rate    = limitPerMinute / 60.0
	)

	return func(c *gin.Context) {
		key := rateKey(c)

		mu.Lock()
		bucket, ok := buckets[key]
		now := time.Now()
		if !ok {
			bucket = &rateBucket{tokens: limitPerMinute, lastRefill: now}
			buckets[key] = bucket
		}

		// Refill tokens
		elapsed := now.Sub(bucket.lastRefill).Seconds()
		bucket.tokens = min(limitPerMinute, bucket.tokens+elapsed*rate)
		bucket.lastRefill = now

		if bucket.tokens < 1 {
			mu.Unlock()
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(429, gin.H{
				"error":   "rate_limited",
				"message": "Too many requests",
			})
			return
		}
		bucket.tokens -= 1
		mu.Unlock()

		c.Next()
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func rateKey(c *gin.Context) string {
	if principal, ok := PrincipalFromContext(c); ok && principal.ID != "" {
		return "pid:" + principal.ID
	}
	ip := clientIP(c.ClientIP())
	if ip != "" {
		return "ip:" + ip
	}
	return "anonymous"
}

// Normalize IPv6-mapped IPv4 etc.
func clientIP(raw string) string {
	if raw == "" {
		return ""
	}
	if ip := net.ParseIP(raw); ip != nil {
		return ip.String()
	}
	return raw
}
