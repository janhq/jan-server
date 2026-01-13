package search

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig defines circuit breaker behavior
type CircuitBreakerConfig struct {
	Enabled          bool
	FailureThreshold int           // Number of failures before opening
	SuccessThreshold int           // Number of successes to close from half-open
	Timeout          time.Duration // How long to stay open before trying half-open
	MaxHalfOpenCalls int           // Max concurrent calls in half-open state
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 15,
		SuccessThreshold: 5,
		Timeout:          45 * time.Second,
		MaxHalfOpenCalls: 10,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	cfg CircuitBreakerConfig
	mu  sync.RWMutex

	state           CircuitState
	failures        int
	successes       int
	lastFailureTime time.Time
	halfOpenCalls   int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		cfg:   cfg,
		state: StateClosed,
	}
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(operation string, fn func() error) error {
	if !cb.allowRequest() {
		return fmt.Errorf("circuit breaker is open for %s", operation)
	}

	err := fn()
	cb.recordResult(operation, err)
	return err
}

// allowRequest determines if a request should be allowed
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.cfg.Enabled {
		return true
	}

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(cb.lastFailureTime) > cb.cfg.Timeout {
			log.Info().Msg("circuit breaker transitioning to half-open")
			cb.state = StateHalfOpen
			cb.halfOpenCalls = 0
			return true
		}
		return false
	case StateHalfOpen:
		// Limit concurrent calls in half-open state
		if cb.halfOpenCalls < cb.cfg.MaxHalfOpenCalls {
			cb.halfOpenCalls++
			return true
		}
		return false
	default:
		return false
	}
}

// recordResult updates circuit breaker state based on result
func (cb *CircuitBreaker) recordResult(operation string, err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.cfg.Enabled {
		return
	}

	if err != nil {
		cb.failures++
		cb.successes = 0
		cb.lastFailureTime = time.Now()

		if cb.state == StateHalfOpen {
			log.Warn().
				Str("operation", operation).
				Msg("circuit breaker opening from half-open due to failure")
			cb.state = StateOpen
			cb.halfOpenCalls = 0
		} else if cb.state == StateClosed && cb.failures >= cb.cfg.FailureThreshold {
			log.Warn().
				Str("operation", operation).
				Int("failures", cb.failures).
				Msg("circuit breaker opening due to failure threshold")
			cb.state = StateOpen
		}
	} else {
		cb.successes++

		if cb.state == StateHalfOpen {
			if cb.successes >= cb.cfg.SuccessThreshold {
				log.Info().
					Str("operation", operation).
					Int("successes", cb.successes).
					Msg("circuit breaker closing from half-open")
				cb.state = StateClosed
				cb.failures = 0
				cb.successes = 0
				cb.halfOpenCalls = 0
			}
		} else if cb.state == StateClosed {
			// Reset failure count on success
			cb.failures = 0
		}
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	if !cb.cfg.Enabled {
		return StateClosed
	}
	return cb.state
}

// GetMetrics returns current circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() map[string]any {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]any{
		"state":             cb.state.String(),
		"failures":          cb.failures,
		"successes":         cb.successes,
		"last_failure_time": cb.lastFailureTime,
		"half_open_calls":   cb.halfOpenCalls,
	}
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.cfg.Enabled {
		return
	}

	log.Info().Msg("manually resetting circuit breaker")
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenCalls = 0
}
