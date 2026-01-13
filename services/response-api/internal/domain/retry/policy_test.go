package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"jan-server/services/response-api/internal/domain/retry"
	"jan-server/services/response-api/internal/domain/status"
)

func TestPolicy_CalculateDelay(t *testing.T) {
	tests := []struct {
		name        string
		policy      retry.Policy
		attempt     int
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{
			name: "fixed backoff - attempt 1",
			policy: retry.Policy{
				BackoffStrategy: retry.BackoffFixed,
				InitialDelay:    100 * time.Millisecond,
				MaxDelay:        1 * time.Second,
				JitterFactor:    0,
			},
			attempt:     1,
			expectedMin: 100 * time.Millisecond,
			expectedMax: 100 * time.Millisecond,
		},
		{
			name: "fixed backoff - attempt 5",
			policy: retry.Policy{
				BackoffStrategy: retry.BackoffFixed,
				InitialDelay:    100 * time.Millisecond,
				MaxDelay:        1 * time.Second,
				JitterFactor:    0,
			},
			attempt:     5,
			expectedMin: 100 * time.Millisecond,
			expectedMax: 100 * time.Millisecond,
		},
		{
			name: "linear backoff - attempt 1",
			policy: retry.Policy{
				BackoffStrategy: retry.BackoffLinear,
				InitialDelay:    100 * time.Millisecond,
				MaxDelay:        1 * time.Second,
				JitterFactor:    0,
			},
			attempt:     1,
			expectedMin: 100 * time.Millisecond,
			expectedMax: 100 * time.Millisecond,
		},
		{
			name: "linear backoff - attempt 3",
			policy: retry.Policy{
				BackoffStrategy: retry.BackoffLinear,
				InitialDelay:    100 * time.Millisecond,
				MaxDelay:        1 * time.Second,
				JitterFactor:    0,
			},
			attempt:     3,
			expectedMin: 300 * time.Millisecond,
			expectedMax: 300 * time.Millisecond,
		},
		{
			name: "exponential backoff - attempt 1",
			policy: retry.Policy{
				BackoffStrategy: retry.BackoffExponential,
				InitialDelay:    100 * time.Millisecond,
				MaxDelay:        10 * time.Second,
				JitterFactor:    0,
			},
			attempt:     1,
			expectedMin: 100 * time.Millisecond,
			expectedMax: 100 * time.Millisecond,
		},
		{
			name: "exponential backoff - attempt 3",
			policy: retry.Policy{
				BackoffStrategy: retry.BackoffExponential,
				InitialDelay:    100 * time.Millisecond,
				MaxDelay:        10 * time.Second,
				JitterFactor:    0,
			},
			attempt:     3,
			expectedMin: 400 * time.Millisecond,
			expectedMax: 400 * time.Millisecond,
		},
		{
			name: "respects max delay",
			policy: retry.Policy{
				BackoffStrategy: retry.BackoffExponential,
				InitialDelay:    100 * time.Millisecond,
				MaxDelay:        200 * time.Millisecond,
				JitterFactor:    0,
			},
			attempt:     10,
			expectedMin: 200 * time.Millisecond,
			expectedMax: 200 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.policy.CalculateDelay(tt.attempt)
			if got < tt.expectedMin || got > tt.expectedMax {
				t.Errorf("Policy.CalculateDelay() = %v, want between %v and %v", got, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestPolicy_ShouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		policy   retry.Policy
		attempt  int
		severity status.ErrorSeverity
		expected bool
	}{
		{
			name:     "should retry on retryable error within max attempts",
			policy:   retry.Policy{MaxRetries: 3},
			attempt:  1,
			severity: status.ErrorSeverityRetryable,
			expected: true,
		},
		{
			name:     "should not retry when max attempts exceeded",
			policy:   retry.Policy{MaxRetries: 3},
			attempt:  3,
			severity: status.ErrorSeverityRetryable,
			expected: false,
		},
		{
			name:     "should not retry on fatal error",
			policy:   retry.Policy{MaxRetries: 3},
			attempt:  1,
			severity: status.ErrorSeverityFatal,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.policy.ShouldRetry(tt.attempt, tt.severity); got != tt.expected {
				t.Errorf("Policy.ShouldRetry() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDefaultPolicy(t *testing.T) {
	policy := retry.DefaultPolicy()

	if policy.MaxRetries != 3 {
		t.Errorf("DefaultPolicy().MaxRetries = %v, want 3", policy.MaxRetries)
	}
	if policy.BackoffStrategy != retry.BackoffExponential {
		t.Errorf("DefaultPolicy().BackoffStrategy = %v, want BackoffExponential", policy.BackoffStrategy)
	}
	if policy.InitialDelay != 1*time.Second {
		t.Errorf("DefaultPolicy().InitialDelay = %v, want 1s", policy.InitialDelay)
	}
}

func TestExecutor_Execute(t *testing.T) {
	t.Run("succeeds on first attempt", func(t *testing.T) {
		executor := retry.NewExecutor(retry.Policy{
			MaxRetries:      3,
			BackoffStrategy: retry.BackoffFixed,
			InitialDelay:    1 * time.Millisecond,
		})

		callCount := 0
		err := executor.Execute(context.Background(), func(ctx context.Context, attempt int) error {
			callCount++
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if callCount != 1 {
			t.Errorf("Expected 1 call, got %d", callCount)
		}
	})

	t.Run("retries on error", func(t *testing.T) {
		retryableErr := errors.New("retryable")
		executor := retry.NewExecutor(retry.Policy{
			MaxRetries:      3,
			BackoffStrategy: retry.BackoffFixed,
			InitialDelay:    1 * time.Millisecond,
		})

		callCount := 0
		err := executor.Execute(context.Background(), func(ctx context.Context, attempt int) error {
			callCount++
			if callCount < 3 {
				return retryableErr
			}
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if callCount != 3 {
			t.Errorf("Expected 3 calls, got %d", callCount)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		executor := retry.NewExecutor(retry.Policy{
			MaxRetries:      3,
			BackoffStrategy: retry.BackoffFixed,
			InitialDelay:    100 * time.Millisecond,
		})

		err := executor.Execute(ctx, func(ctx context.Context, attempt int) error {
			return errors.New("should not reach here")
		})

		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	})
}

func TestExecuteWithResult(t *testing.T) {
	t.Run("returns result on success", func(t *testing.T) {
		policy := retry.Policy{
			MaxRetries:      3,
			BackoffStrategy: retry.BackoffFixed,
			InitialDelay:    1 * time.Millisecond,
		}

		result, err := retry.ExecuteWithResult(context.Background(), policy, func(ctx context.Context, attempt int) (string, error) {
			return "success", nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != "success" {
			t.Errorf("Expected 'success', got %v", result)
		}
	})

	t.Run("retries and returns result", func(t *testing.T) {
		policy := retry.Policy{
			MaxRetries:      3,
			BackoffStrategy: retry.BackoffFixed,
			InitialDelay:    1 * time.Millisecond,
		}

		callCount := 0
		result, err := retry.ExecuteWithResult(context.Background(), policy, func(ctx context.Context, attempt int) (int, error) {
			callCount++
			if callCount < 2 {
				return 0, errors.New("retryable")
			}
			return 42, nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != 42 {
			t.Errorf("Expected 42, got %v", result)
		}
	})
}
