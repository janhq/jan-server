// Package status defines shared status types for plans and responses.
package status

import "errors"

// Status represents the lifecycle status of a plan or response.
type Status string

const (
	// Non-terminal states
	StatusPending     Status = "pending"       // Created, not yet started
	StatusPlanning    Status = "planning"      // Agent analyzing, creating plan
	StatusInProgress  Status = "in_progress"   // Actively executing
	StatusWaitForUser Status = "wait_for_user" // Blocked on user input

	// Terminal states (no further transitions allowed)
	StatusCompleted Status = "completed" // Successfully finished
	StatusFailed    Status = "failed"    // Unrecoverable error
	StatusCancelled Status = "cancelled" // User or system cancelled
	StatusExpired   Status = "expired"   // Timeout waiting for user
	StatusSkipped   Status = "skipped"   // Step was skipped
)

// ErrInvalidTransition is returned when a status transition is not allowed.
var ErrInvalidTransition = errors.New("invalid status transition")

// IsTerminal returns true if the status is a terminal state.
func (s Status) IsTerminal() bool {
	return s == StatusCompleted || s == StatusFailed ||
		s == StatusCancelled || s == StatusExpired
}

// IsRetryable returns true if the status allows retry.
func (s Status) IsRetryable() bool {
	return s == StatusFailed
}

// IsActive returns true if the status indicates active processing.
func (s Status) IsActive() bool {
	return s == StatusPending || s == StatusPlanning ||
		s == StatusInProgress || s == StatusWaitForUser
}

// String returns the string representation of the status.
func (s Status) String() string {
	return string(s)
}

// ValidTransitions defines allowed status transitions.
var ValidTransitions = map[Status][]Status{
	StatusPending:     {StatusPlanning, StatusInProgress, StatusFailed, StatusCancelled},
	StatusPlanning:    {StatusInProgress, StatusFailed, StatusCancelled},
	StatusInProgress:  {StatusWaitForUser, StatusCompleted, StatusFailed, StatusCancelled},
	StatusWaitForUser: {StatusInProgress, StatusExpired, StatusCancelled},
	StatusFailed:      {StatusInProgress}, // Retry allowed
	// Terminal states have no valid transitions
	StatusCompleted: {},
	StatusCancelled: {},
	StatusExpired:   {},
}

// CanTransitionTo checks if a transition from current status to target status is valid.
func (s Status) CanTransitionTo(target Status) bool {
	validTargets, ok := ValidTransitions[s]
	if !ok {
		return false
	}
	for _, t := range validTargets {
		if t == target {
			return true
		}
	}
	return false
}

// TransitionTo attempts to transition to the target status and returns error if invalid.
func (s Status) TransitionTo(target Status) (Status, error) {
	if !s.CanTransitionTo(target) {
		return s, ErrInvalidTransition
	}
	return target, nil
}

// ErrorSeverity indicates how an error should be handled.
type ErrorSeverity string

const (
	ErrorSeverityRetryable ErrorSeverity = "retryable" // Retry with backoff
	ErrorSeverityFallback  ErrorSeverity = "fallback"  // Use fallback provider/method
	ErrorSeveritySkippable ErrorSeverity = "skippable" // Skip step, continue plan
	ErrorSeverityFatal     ErrorSeverity = "fatal"     // Fail entire plan
)

// String returns the string representation of the error severity.
func (e ErrorSeverity) String() string {
	return string(e)
}

// IsRetryable returns true if the error can be retried.
func (e ErrorSeverity) IsRetryable() bool {
	return e == ErrorSeverityRetryable
}

// IsFatal returns true if the error should fail the plan.
func (e ErrorSeverity) IsFatal() bool {
	return e == ErrorSeverityFatal
}
