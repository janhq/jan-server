package platformerrors

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// getRequestIDFromContext extracts request ID from context
func getRequestIDFromContext(ctx context.Context) string {
	val := ctx.Value("requestID")
	if requestID, ok := val.(string); ok {
		return requestID
	}
	return ""
}

// ErrorType represents the category of error
type ErrorType string

const (
	ErrorTypeNotFound       ErrorType = "NOT_FOUND"
	ErrorTypeTooManyRecords ErrorType = "TOO_MANY_RECORDS"
	ErrorTypeValidation     ErrorType = "VALIDATION"
	ErrorTypeConflict       ErrorType = "CONFLICT"
	ErrorTypeUnauthorized   ErrorType = "UNAUTHORIZED"
	ErrorTypeForbidden      ErrorType = "FORBIDDEN"
	ErrorTypeInternal       ErrorType = "INTERNAL"
	ErrorTypeExternal       ErrorType = "EXTERNAL"
	ErrorTypeDatabaseError  ErrorType = "DATABASE_ERROR"
	ErrorTypeNotImplemented ErrorType = "NOT_IMPLEMENTED"
)

// Layer represents the application layer where the error occurred
type Layer string

const (
	LayerRepository     Layer = "repository"
	LayerDomain         Layer = "domain"
	LayerHandler        Layer = "handler"
	LayerRoute          Layer = "route"
	LayerInfrastructure Layer = "infrastructure"
	LayerCommon         Layer = "common"
)

// PlatformError represents an error with context and metadata
type PlatformError struct {
	UUID      string
	Type      ErrorType
	Message   string
	Err       error
	Context   map[string]any
	RequestID string
	Layer     Layer
	Timestamp time.Time
}

// Error implements the error interface
func (e *PlatformError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s][%s][%s] %s: %v", e.Layer, e.Type, e.UUID, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s][%s][%s] %s", e.Layer, e.Type, e.UUID, e.Message)
}

// Unwrap returns the underlying error
func (e *PlatformError) Unwrap() error {
	return e.Err
}

// GetErrorType returns the error type
func (e *PlatformError) GetErrorType() ErrorType {
	return e.Type
}

// GetRequestID returns the request ID
func (e *PlatformError) GetRequestID() string {
	return e.RequestID
}

// GetUUID returns the error UUID
func (e *PlatformError) GetUUID() string {
	return e.UUID
}

// NewError creates a new PlatformError with the specified parameters
func NewError(ctx context.Context, layer Layer, errorType ErrorType, message string, err error, customUUID string) *PlatformError {
	return NewErrorWithContext(ctx, layer, errorType, message, err, customUUID, nil)
}

// NewErrorWithContext creates a new PlatformError with additional context fields
func NewErrorWithContext(ctx context.Context, layer Layer, errorType ErrorType, message string, err error, customUUID string, contextFields map[string]any) *PlatformError {
	requestID := getRequestIDFromContext(ctx)

	errorUUID := customUUID
	if errorUUID == "" {
		errorUUID = "auto-generated-uuid"
	}

	errorContext := make(map[string]any)
	for k, v := range contextFields {
		errorContext[k] = v
	}

	platformError := &PlatformError{
		UUID:      errorUUID,
		Type:      errorType,
		Message:   message,
		Err:       err,
		RequestID: requestID,
		Layer:     layer,
		Timestamp: time.Now().UTC(),
		Context:   errorContext,
	}

	return platformError
}

// AsError wraps an error with layer context
func AsError(ctx context.Context, layer Layer, err error, message string) *PlatformError {
	if err == nil {
		return nil
	}

	var platformErr *PlatformError
	if errors.As(err, &platformErr) {
		return NewError(ctx, layer, platformErr.Type, fmt.Sprintf("%s: %s", message, platformErr.Message), platformErr, platformErr.UUID)
	}

	errorType := ErrorTypeInternal

	return NewError(ctx, layer, errorType, message, err, "")
}

// ErrorTypeToHTTPStatus maps error types to HTTP status codes
func ErrorTypeToHTTPStatus(errorType ErrorType) int {
	switch errorType {
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeValidation:
		return http.StatusBadRequest
	case ErrorTypeConflict:
		return http.StatusConflict
	case ErrorTypeUnauthorized:
		return http.StatusUnauthorized
	case ErrorTypeForbidden:
		return http.StatusForbidden
	case ErrorTypeNotImplemented:
		return http.StatusNotImplemented
	case ErrorTypeTooManyRecords:
		return http.StatusInternalServerError
	case ErrorTypeDatabaseError:
		return http.StatusInternalServerError
	case ErrorTypeExternal:
		return http.StatusBadGateway
	case ErrorTypeInternal:
		fallthrough
	default:
		return http.StatusInternalServerError
	}
}

// IsErrorType checks if an error is a PlatformError with the specified type
func IsErrorType(err error, errorType ErrorType) bool {
	if err == nil {
		return false
	}

	var platformErr *PlatformError
	if errors.As(err, &platformErr) {
		return platformErr.Type == errorType
	}

	return false
}

// LogError logs a platform error with proper structure
func LogError(logger zerolog.Logger, err *PlatformError) {
	if err == nil {
		return
	}

	event := logger.Error().
		Str("error_uuid", err.UUID).
		Str("error_type", string(err.Type)).
		Str("layer", string(err.Layer)).
		Time("timestamp_utc", err.Timestamp)

	if err.RequestID != "" {
		event = event.Str("request_id", err.RequestID)
	}

	for k, v := range err.Context {
		event = event.Interface(k, v)
	}

	if err.Err != nil {
		event = event.Err(err.Err)
	}

	event.Msg(err.Message)
}
