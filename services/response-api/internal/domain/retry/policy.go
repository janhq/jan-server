// Package retry defines retry policies and backoff strategies.
package retry

import (
	"context"
	"math"
	"math/rand"
	"time"

	"jan-server/services/response-api/internal/domain/status"
)

// Policy defines a retry strategy.
type Policy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffStrategy BackoffType   `json:"backoff_strategy"`
	JitterFactor    float64       `json:"jitter_factor"` // 0.0-1.0
	RetryableErrors []string      `json:"retryable_errors,omitempty"`
}

// BackoffType identifies the backoff strategy.
type BackoffType string

const (
	BackoffFixed       BackoffType = "fixed"       // Same delay each time
	BackoffLinear      BackoffType = "linear"      // Delay increases linearly
	BackoffExponential BackoffType = "exponential" // Delay doubles each time
)

// DefaultPolicy returns a sensible default retry policy.
func DefaultPolicy() Policy {
	return Policy{
		MaxRetries:      3,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffStrategy: BackoffExponential,
		JitterFactor:    0.25,
	}
}

// AggressivePolicy returns a more aggressive retry policy.
func AggressivePolicy() Policy {
	return Policy{
		MaxRetries:      5,
		InitialDelay:    500 * time.Millisecond,
		MaxDelay:        60 * time.Second,
		BackoffStrategy: BackoffExponential,
		JitterFactor:    0.3,
	}
}

// ConservativePolicy returns a conservative retry policy.
func ConservativePolicy() Policy {
	return Policy{
		MaxRetries:      2,
		InitialDelay:    2 * time.Second,
		MaxDelay:        10 * time.Second,
		BackoffStrategy: BackoffLinear,
		JitterFactor:    0.1,
	}
}

// NoRetryPolicy returns a policy that never retries.
func NoRetryPolicy() Policy {
	return Policy{
		MaxRetries:   0,
		InitialDelay: 0,
		MaxDelay:     0,
	}
}

// CalculateDelay calculates the delay for a given attempt.
func (p *Policy) CalculateDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	var delay time.Duration

	switch p.BackoffStrategy {
	case BackoffFixed:
		delay = p.InitialDelay
	case BackoffLinear:
		delay = p.InitialDelay * time.Duration(attempt)
	case BackoffExponential:
		delay = p.InitialDelay * time.Duration(math.Pow(2, float64(attempt-1)))
	default:
		delay = p.InitialDelay
	}

	// Apply max delay cap
	if delay > p.MaxDelay {
		delay = p.MaxDelay
	}

	// Apply jitter
	if p.JitterFactor > 0 {
		jitter := float64(delay) * p.JitterFactor * (rand.Float64()*2 - 1) // -jitter to +jitter
		delay = time.Duration(float64(delay) + jitter)
		if delay < 0 {
			delay = 0
		}
	}

	return delay
}

// ShouldRetry determines if a retry should be attempted.
func (p *Policy) ShouldRetry(attempt int, severity status.ErrorSeverity) bool {
	if attempt >= p.MaxRetries {
		return false
	}
	return severity.IsRetryable()
}

// Executor provides retry execution functionality.
type Executor struct {
	policy Policy
}

// NewExecutor creates a new retry executor with the given policy.
func NewExecutor(policy Policy) *Executor {
	return &Executor{policy: policy}
}

// RetryableFunc is a function that can be retried.
type RetryableFunc func(ctx context.Context, attempt int) error

// Execute runs the function with retries according to the policy.
func (e *Executor) Execute(ctx context.Context, fn RetryableFunc) error {
	var lastErr error

	for attempt := 0; attempt <= e.policy.MaxRetries; attempt++ {
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the function
		err := fn(ctx, attempt)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if attempt >= e.policy.MaxRetries {
			break
		}

		// Wait before retrying
		delay := e.policy.CalculateDelay(attempt + 1)
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}

	return lastErr
}

// ExecuteWithResult runs the function with retries and returns a result.
func ExecuteWithResult[T any](ctx context.Context, policy Policy, fn func(ctx context.Context, attempt int) (T, error)) (T, error) {
	var zero T
	var lastErr error
	var result T

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}

		r, err := fn(ctx, attempt)
		if err == nil {
			return r, nil
		}

		result = r
		lastErr = err

		if attempt >= policy.MaxRetries {
			break
		}

		delay := policy.CalculateDelay(attempt + 1)
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return zero, ctx.Err()
			case <-timer.C:
			}
		}
	}

	return result, lastErr
}

// IsRetryableError checks if an error code is in the retryable list.
func (p *Policy) IsRetryableError(errorCode string) bool {
	if len(p.RetryableErrors) == 0 {
		return true // No specific list means all errors are retryable
	}
	for _, code := range p.RetryableErrors {
		if code == errorCode {
			return true
		}
	}
	return false
}
