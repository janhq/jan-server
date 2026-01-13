package status_test

import (
	"testing"

	"jan-server/services/response-api/internal/domain/status"
)

func TestStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		name     string
		status   status.Status
		expected bool
	}{
		{"pending is not terminal", status.StatusPending, false},
		{"planning is not terminal", status.StatusPlanning, false},
		{"in_progress is not terminal", status.StatusInProgress, false},
		{"wait_for_user is not terminal", status.StatusWaitForUser, false},
		{"completed is terminal", status.StatusCompleted, true},
		{"failed is terminal", status.StatusFailed, true},
		{"cancelled is terminal", status.StatusCancelled, true},
		{"expired is terminal", status.StatusExpired, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.expected {
				t.Errorf("Status.IsTerminal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStatus_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		status   status.Status
		expected bool
	}{
		{"pending is not retryable", status.StatusPending, false},
		{"failed is retryable", status.StatusFailed, true},
		{"completed is not retryable", status.StatusCompleted, false},
		{"cancelled is not retryable", status.StatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsRetryable(); got != tt.expected {
				t.Errorf("Status.IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStatus_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   status.Status
		expected bool
	}{
		{"pending is active", status.StatusPending, true},
		{"planning is active", status.StatusPlanning, true},
		{"in_progress is active", status.StatusInProgress, true},
		{"wait_for_user is active", status.StatusWaitForUser, true},
		{"completed is not active", status.StatusCompleted, false},
		{"failed is not active", status.StatusFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsActive(); got != tt.expected {
				t.Errorf("Status.IsActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name  string
		from  status.Status
		to    status.Status
		canDo bool
	}{
		// Valid transitions from pending
		{"pending to planning", status.StatusPending, status.StatusPlanning, true},
		{"pending to in_progress", status.StatusPending, status.StatusInProgress, true},
		{"pending to failed", status.StatusPending, status.StatusFailed, true},
		{"pending to cancelled", status.StatusPending, status.StatusCancelled, true},
		{"pending to completed - invalid", status.StatusPending, status.StatusCompleted, false},

		// Valid transitions from planning
		{"planning to in_progress", status.StatusPlanning, status.StatusInProgress, true},
		{"planning to failed", status.StatusPlanning, status.StatusFailed, true},
		{"planning to pending - invalid", status.StatusPlanning, status.StatusPending, false},

		// Valid transitions from in_progress
		{"in_progress to wait_for_user", status.StatusInProgress, status.StatusWaitForUser, true},
		{"in_progress to completed", status.StatusInProgress, status.StatusCompleted, true},
		{"in_progress to failed", status.StatusInProgress, status.StatusFailed, true},
		{"in_progress to cancelled", status.StatusInProgress, status.StatusCancelled, true},

		// Valid transitions from wait_for_user
		{"wait_for_user to in_progress", status.StatusWaitForUser, status.StatusInProgress, true},
		{"wait_for_user to expired", status.StatusWaitForUser, status.StatusExpired, true},
		{"wait_for_user to cancelled", status.StatusWaitForUser, status.StatusCancelled, true},

		// Retry from failed
		{"failed to in_progress (retry)", status.StatusFailed, status.StatusInProgress, true},

		// Terminal states have no valid transitions
		{"completed to anything - invalid", status.StatusCompleted, status.StatusInProgress, false},
		{"cancelled to anything - invalid", status.StatusCancelled, status.StatusInProgress, false},
		{"expired to anything - invalid", status.StatusExpired, status.StatusInProgress, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.canDo {
				t.Errorf("Status.CanTransitionTo() = %v, want %v", got, tt.canDo)
			}
		})
	}
}

func TestStatus_TransitionTo(t *testing.T) {
	// Valid transition
	s := status.StatusPending
	newStatus, err := s.TransitionTo(status.StatusPlanning)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if newStatus != status.StatusPlanning {
		t.Errorf("Expected status to be planning, got %v", newStatus)
	}

	// Invalid transition
	s = status.StatusCompleted
	_, err = s.TransitionTo(status.StatusInProgress)
	if err != status.ErrInvalidTransition {
		t.Errorf("Expected ErrInvalidTransition, got %v", err)
	}
}

func TestErrorSeverity_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		severity status.ErrorSeverity
		expected bool
	}{
		{"retryable is retryable", status.ErrorSeverityRetryable, true},
		{"fallback is not retryable", status.ErrorSeverityFallback, false},
		{"skippable is not retryable", status.ErrorSeveritySkippable, false},
		{"fatal is not retryable", status.ErrorSeverityFatal, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.severity.IsRetryable(); got != tt.expected {
				t.Errorf("ErrorSeverity.IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrorSeverity_IsFatal(t *testing.T) {
	tests := []struct {
		name     string
		severity status.ErrorSeverity
		expected bool
	}{
		{"fatal is fatal", status.ErrorSeverityFatal, true},
		{"retryable is not fatal", status.ErrorSeverityRetryable, false},
		{"fallback is not fatal", status.ErrorSeverityFallback, false},
		{"skippable is not fatal", status.ErrorSeveritySkippable, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.severity.IsFatal(); got != tt.expected {
				t.Errorf("ErrorSeverity.IsFatal() = %v, want %v", got, tt.expected)
			}
		})
	}
}
