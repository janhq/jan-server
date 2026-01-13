// Package errors defines error types and classification for agent execution.
package errors

import (
	"errors"
	"fmt"

	"jan-server/services/response-api/internal/domain/status"
)

// StepError represents an error that occurred during step execution.
type StepError struct {
	Code      string               `json:"code"`
	Message   string               `json:"message"`
	Severity  status.ErrorSeverity `json:"severity"`
	StepID    string               `json:"step_id,omitempty"`
	TaskID    string               `json:"task_id,omitempty"`
	PlanID    string               `json:"plan_id,omitempty"`
	Cause     error                `json:"-"`
	Retryable bool                 `json:"retryable"`
	Details   map[string]any       `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *StepError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause.
func (e *StepError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns true if the error can be retried.
func (e *StepError) IsRetryable() bool {
	return e.Retryable && e.Severity.IsRetryable()
}

// IsFatal returns true if the error should fail the entire plan.
func (e *StepError) IsFatal() bool {
	return e.Severity.IsFatal()
}

// NewStepError creates a new step error.
func NewStepError(code, message string, severity status.ErrorSeverity) *StepError {
	return &StepError{
		Code:      code,
		Message:   message,
		Severity:  severity,
		Retryable: severity.IsRetryable(),
	}
}

// WithCause adds an underlying cause to the error.
func (e *StepError) WithCause(cause error) *StepError {
	e.Cause = cause
	return e
}

// WithStepContext adds step context to the error.
func (e *StepError) WithStepContext(stepID, taskID, planID string) *StepError {
	e.StepID = stepID
	e.TaskID = taskID
	e.PlanID = planID
	return e
}

// WithDetails adds additional details to the error.
func (e *StepError) WithDetails(details map[string]any) *StepError {
	e.Details = details
	return e
}

// Common error codes.
const (
	// Retryable errors
	ErrCodeTimeout        = "TIMEOUT"
	ErrCodeRateLimit      = "RATE_LIMIT"
	ErrCodeServiceUnavail = "SERVICE_UNAVAILABLE"
	ErrCodeTemporary      = "TEMPORARY_FAILURE"

	// Fallback-capable errors
	ErrCodeProviderError = "PROVIDER_ERROR"
	ErrCodeToolNotFound  = "TOOL_NOT_FOUND"
	ErrCodeModelUnavail  = "MODEL_UNAVAILABLE"

	// Skippable errors
	ErrCodeOptionalFailed = "OPTIONAL_FAILED"
	ErrCodeNonCritical    = "NON_CRITICAL"

	// Fatal errors
	ErrCodeInvalidInput  = "INVALID_INPUT"
	ErrCodeAuthFailed    = "AUTH_FAILED"
	ErrCodeQuotaExceeded = "QUOTA_EXCEEDED"
	ErrCodePlanInvalid   = "PLAN_INVALID"
	ErrCodeSystemError   = "SYSTEM_ERROR"
)

// Predefined errors for common scenarios.
var (
	ErrTimeout = &StepError{
		Code:      ErrCodeTimeout,
		Message:   "operation timed out",
		Severity:  status.ErrorSeverityRetryable,
		Retryable: true,
	}

	ErrRateLimit = &StepError{
		Code:      ErrCodeRateLimit,
		Message:   "rate limit exceeded",
		Severity:  status.ErrorSeverityRetryable,
		Retryable: true,
	}

	ErrServiceUnavailable = &StepError{
		Code:      ErrCodeServiceUnavail,
		Message:   "service temporarily unavailable",
		Severity:  status.ErrorSeverityRetryable,
		Retryable: true,
	}

	ErrProviderError = &StepError{
		Code:      ErrCodeProviderError,
		Message:   "provider returned an error",
		Severity:  status.ErrorSeverityFallback,
		Retryable: false,
	}

	ErrInvalidInput = &StepError{
		Code:      ErrCodeInvalidInput,
		Message:   "invalid input provided",
		Severity:  status.ErrorSeverityFatal,
		Retryable: false,
	}

	ErrSystemError = &StepError{
		Code:      ErrCodeSystemError,
		Message:   "internal system error",
		Severity:  status.ErrorSeverityFatal,
		Retryable: false,
	}
)

// Classifier classifies errors into severity levels.
type Classifier struct {
	rules []ClassificationRule
}

// ClassificationRule defines a rule for classifying errors.
type ClassificationRule struct {
	Match    func(error) bool
	Severity status.ErrorSeverity
}

// NewClassifier creates a new error classifier with default rules.
func NewClassifier() *Classifier {
	c := &Classifier{}
	c.addDefaultRules()
	return c
}

// addDefaultRules adds the default classification rules.
func (c *Classifier) addDefaultRules() {
	// Context cancellation is fatal
	c.rules = append(c.rules, ClassificationRule{
		Match:    func(err error) bool { return errors.Is(err, errors.New("context canceled")) },
		Severity: status.ErrorSeverityFatal,
	})

	// Timeout errors are retryable
	c.rules = append(c.rules, ClassificationRule{
		Match: func(err error) bool {
			var se *StepError
			if errors.As(err, &se) {
				return se.Code == ErrCodeTimeout
			}
			return false
		},
		Severity: status.ErrorSeverityRetryable,
	})

	// Rate limits are retryable
	c.rules = append(c.rules, ClassificationRule{
		Match: func(err error) bool {
			var se *StepError
			if errors.As(err, &se) {
				return se.Code == ErrCodeRateLimit
			}
			return false
		},
		Severity: status.ErrorSeverityRetryable,
	})
}

// AddRule adds a classification rule.
func (c *Classifier) AddRule(rule ClassificationRule) {
	c.rules = append(c.rules, rule)
}

// Classify determines the severity of an error.
func (c *Classifier) Classify(err error) status.ErrorSeverity {
	if err == nil {
		return ""
	}

	// Check if already a StepError
	var se *StepError
	if errors.As(err, &se) {
		return se.Severity
	}

	// Apply rules
	for _, rule := range c.rules {
		if rule.Match(err) {
			return rule.Severity
		}
	}

	// Default to retryable
	return status.ErrorSeverityRetryable
}

// Wrap wraps an error with step context.
func Wrap(err error, code, message string, severity status.ErrorSeverity) *StepError {
	return &StepError{
		Code:      code,
		Message:   message,
		Severity:  severity,
		Cause:     err,
		Retryable: severity.IsRetryable(),
	}
}

// WrapRetryable wraps an error as retryable.
func WrapRetryable(err error, message string) *StepError {
	return Wrap(err, ErrCodeTemporary, message, status.ErrorSeverityRetryable)
}

// WrapFatal wraps an error as fatal.
func WrapFatal(err error, message string) *StepError {
	return Wrap(err, ErrCodeSystemError, message, status.ErrorSeverityFatal)
}

// WrapFallback wraps an error for fallback handling.
func WrapFallback(err error, message string) *StepError {
	return Wrap(err, ErrCodeProviderError, message, status.ErrorSeverityFallback)
}
