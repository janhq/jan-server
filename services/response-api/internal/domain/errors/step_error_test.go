package errors_test

import (
	"errors"
	"testing"

	stepErrors "jan-server/services/response-api/internal/domain/errors"
	"jan-server/services/response-api/internal/domain/status"
)

func TestStepError_Error(t *testing.T) {
	stepErr := stepErrors.NewStepError("TOOL_TIMEOUT", "Tool execution timed out", status.ErrorSeverityRetryable)

	expected := "TOOL_TIMEOUT: Tool execution timed out"
	if got := stepErr.Error(); got != expected {
		t.Errorf("StepError.Error() = %v, want %v", got, expected)
	}
}

func TestStepError_ErrorWithCause(t *testing.T) {
	cause := errors.New("underlying error")
	stepErr := stepErrors.NewStepError("WRAPPED", "Wrapped error", status.ErrorSeverityFatal).WithCause(cause)

	expected := "WRAPPED: Wrapped error (caused by: underlying error)"
	if got := stepErr.Error(); got != expected {
		t.Errorf("StepError.Error() = %v, want %v", got, expected)
	}
}

func TestStepError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	stepErr := stepErrors.NewStepError("WRAPPED", "Wrapped error", status.ErrorSeverityFatal).WithCause(originalErr)

	if got := stepErr.Unwrap(); got != originalErr {
		t.Errorf("StepError.Unwrap() = %v, want %v", got, originalErr)
	}
}

func TestStepError_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		severity status.ErrorSeverity
		expected bool
	}{
		{"retryable error", status.ErrorSeverityRetryable, true},
		{"fallback error", status.ErrorSeverityFallback, false},
		{"skippable error", status.ErrorSeveritySkippable, false},
		{"fatal error", status.ErrorSeverityFatal, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepErr := stepErrors.NewStepError("TEST", "test", tt.severity)
			if got := stepErr.IsRetryable(); got != tt.expected {
				t.Errorf("StepError.IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStepError_IsFatal(t *testing.T) {
	tests := []struct {
		name     string
		severity status.ErrorSeverity
		expected bool
	}{
		{"fatal error", status.ErrorSeverityFatal, true},
		{"retryable error", status.ErrorSeverityRetryable, false},
		{"fallback error", status.ErrorSeverityFallback, false},
		{"skippable error", status.ErrorSeveritySkippable, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepErr := stepErrors.NewStepError("TEST", "test", tt.severity)
			if got := stepErr.IsFatal(); got != tt.expected {
				t.Errorf("StepError.IsFatal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewStepError(t *testing.T) {
	stepErr := stepErrors.NewStepError("API_ERROR", "API call failed", status.ErrorSeverityRetryable)

	if stepErr.Code != "API_ERROR" {
		t.Errorf("NewStepError().Code = %v, want API_ERROR", stepErr.Code)
	}
	if stepErr.Message != "API call failed" {
		t.Errorf("NewStepError().Message = %v, want 'API call failed'", stepErr.Message)
	}
	if stepErr.Severity != status.ErrorSeverityRetryable {
		t.Errorf("NewStepError().Severity = %v, want retryable", stepErr.Severity)
	}
	if !stepErr.Retryable {
		t.Error("NewStepError().Retryable should be true for retryable severity")
	}
}

func TestStepError_WithStepContext(t *testing.T) {
	stepErr := stepErrors.NewStepError("TEST", "test", status.ErrorSeverityRetryable).
		WithStepContext("step-1", "task-1", "plan-1")

	if stepErr.StepID != "step-1" {
		t.Errorf("StepError.StepID = %v, want 'step-1'", stepErr.StepID)
	}
	if stepErr.TaskID != "task-1" {
		t.Errorf("StepError.TaskID = %v, want 'task-1'", stepErr.TaskID)
	}
	if stepErr.PlanID != "plan-1" {
		t.Errorf("StepError.PlanID = %v, want 'plan-1'", stepErr.PlanID)
	}
}

func TestStepError_WithDetails(t *testing.T) {
	details := map[string]any{
		"provider": "openai",
		"status":   500,
	}
	stepErr := stepErrors.NewStepError("TEST", "test", status.ErrorSeverityRetryable).WithDetails(details)

	if stepErr.Details["provider"] != "openai" {
		t.Errorf("StepError.Details[provider] = %v, want 'openai'", stepErr.Details["provider"])
	}
	if stepErr.Details["status"] != 500 {
		t.Errorf("StepError.Details[status] = %v, want 500", stepErr.Details["status"])
	}
}

func TestClassifier_Classify(t *testing.T) {
	classifier := stepErrors.NewClassifier()

	t.Run("classifies StepError with existing severity", func(t *testing.T) {
		err := stepErrors.NewStepError("TEST", "test", status.ErrorSeverityFatal)
		severity := classifier.Classify(err)
		if severity != status.ErrorSeverityFatal {
			t.Errorf("Classifier.Classify() = %v, want fatal", severity)
		}
	})

	t.Run("returns empty for nil error", func(t *testing.T) {
		severity := classifier.Classify(nil)
		if severity != "" {
			t.Errorf("Classifier.Classify(nil) = %v, want empty", severity)
		}
	})

	t.Run("defaults to retryable for unknown errors", func(t *testing.T) {
		err := errors.New("some unknown error")
		severity := classifier.Classify(err)
		if severity != status.ErrorSeverityRetryable {
			t.Errorf("Classifier.Classify() = %v, want retryable", severity)
		}
	})
}

func TestWrapFunctions(t *testing.T) {
	cause := errors.New("original error")

	t.Run("Wrap", func(t *testing.T) {
		err := stepErrors.Wrap(cause, "CODE", "message", status.ErrorSeverityFallback)
		if err.Code != "CODE" {
			t.Errorf("Wrap().Code = %v, want CODE", err.Code)
		}
		if err.Cause != cause {
			t.Errorf("Wrap().Cause = %v, want %v", err.Cause, cause)
		}
	})

	t.Run("WrapRetryable", func(t *testing.T) {
		err := stepErrors.WrapRetryable(cause, "retryable message")
		if err.Severity != status.ErrorSeverityRetryable {
			t.Errorf("WrapRetryable().Severity = %v, want retryable", err.Severity)
		}
	})

	t.Run("WrapFatal", func(t *testing.T) {
		err := stepErrors.WrapFatal(cause, "fatal message")
		if err.Severity != status.ErrorSeverityFatal {
			t.Errorf("WrapFatal().Severity = %v, want fatal", err.Severity)
		}
	})

	t.Run("WrapFallback", func(t *testing.T) {
		err := stepErrors.WrapFallback(cause, "fallback message")
		if err.Severity != status.ErrorSeverityFallback {
			t.Errorf("WrapFallback().Severity = %v, want fallback", err.Severity)
		}
	})
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *stepErrors.StepError
		wantCode string
		wantSev  status.ErrorSeverity
	}{
		{"ErrTimeout", stepErrors.ErrTimeout, stepErrors.ErrCodeTimeout, status.ErrorSeverityRetryable},
		{"ErrRateLimit", stepErrors.ErrRateLimit, stepErrors.ErrCodeRateLimit, status.ErrorSeverityRetryable},
		{"ErrServiceUnavailable", stepErrors.ErrServiceUnavailable, stepErrors.ErrCodeServiceUnavail, status.ErrorSeverityRetryable},
		{"ErrProviderError", stepErrors.ErrProviderError, stepErrors.ErrCodeProviderError, status.ErrorSeverityFallback},
		{"ErrInvalidInput", stepErrors.ErrInvalidInput, stepErrors.ErrCodeInvalidInput, status.ErrorSeverityFatal},
		{"ErrSystemError", stepErrors.ErrSystemError, stepErrors.ErrCodeSystemError, status.ErrorSeverityFatal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("%s.Code = %v, want %v", tt.name, tt.err.Code, tt.wantCode)
			}
			if tt.err.Severity != tt.wantSev {
				t.Errorf("%s.Severity = %v, want %v", tt.name, tt.err.Severity, tt.wantSev)
			}
		})
	}
}
