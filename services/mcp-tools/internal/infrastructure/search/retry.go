package search

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// RetryConfig defines retry behavior for search operations
type RetryConfig struct {
	MaxAttempts        int
	InitialDelay       time.Duration
	MaxDelay           time.Duration
	BackoffFactor      float64
	RetryableErrors    []string
	NonRetryableErrors []string // Errors that should never be retried
}

// DefaultRetryConfig returns sensible defaults for retry behavior
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   5,
		InitialDelay:  250 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 1.5,
		RetryableErrors: []string{
			"timeout",
			"connection refused",
			"temporary failure",
			"429", // Rate limit
			"500", // Internal server error
			"502", // Bad gateway
			"503", // Service unavailable
			"504", // Gateway timeout
		},
		NonRetryableErrors: []string{
			"403", // Forbidden - anti-bot protection, won't succeed on retry
			"401", // Unauthorized - auth issue, won't succeed on retry
			"404", // Not found - page doesn't exist
			"410", // Gone - resource permanently removed
		},
	}
}

// RetryableFunc is a function that can be retried
type RetryableFunc[T any] func() (*T, error)

// WithRetry executes a function with exponential backoff retry logic
func WithRetry[T any](ctx context.Context, cfg RetryConfig, operation string, fn RetryableFunc[T]) (*T, error) {
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		result, err := fn()
		if err == nil {
			if attempt > 1 {
				log.Info().
					Str("operation", operation).
					Int("attempt", attempt).
					Msg("operation succeeded after retry")
			}
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err, cfg) {
			log.Debug().
				Err(err).
				Str("operation", operation).
				Int("attempt", attempt).
				Msg("non-retryable error, aborting")
			return nil, err
		}

		// Don't sleep after last attempt
		if attempt == cfg.MaxAttempts {
			break
		}

		// Calculate backoff delay
		delay := calculateBackoff(attempt, cfg.InitialDelay, cfg.MaxDelay, cfg.BackoffFactor)

		log.Warn().
			Err(err).
			Str("operation", operation).
			Int("attempt", attempt).
			Int("max_attempts", cfg.MaxAttempts).
			Dur("retry_delay", delay).
			Msg("retrying operation after error")

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return nil, fmt.Errorf("operation failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// calculateBackoff computes exponential backoff delay with jitter
func calculateBackoff(attempt int, initial, max time.Duration, factor float64) time.Duration {
	backoff := float64(initial) * math.Pow(factor, float64(attempt-1))

	if backoff > float64(max) {
		backoff = float64(max)
	}

	// Add 10% jitter to prevent thundering herd
	jitter := backoff * 0.1 * (2.0*float64(time.Now().UnixNano()%100)/100.0 - 1.0)

	return time.Duration(backoff + jitter)
}

// isRetryable checks if an error should trigger a retry
func isRetryable(err error, cfg RetryConfig) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Check non-retryable errors first - these should never be retried
	for _, pattern := range cfg.NonRetryableErrors {
		if strings.Contains(errStr, strings.ToLower(pattern)) {
			return false
		}
	}

	// Check if error matches known retryable patterns
	for _, pattern := range cfg.RetryableErrors {
		if strings.Contains(errStr, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}
