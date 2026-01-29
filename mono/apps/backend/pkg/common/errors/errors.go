package errors

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// Layer represents the architectural layer where an error originated
type Layer string

const (
	LayerDomain         Layer = "domain"
	LayerRepository     Layer = "repository"
	LayerInfrastructure Layer = "infrastructure"
	LayerHTTP           Layer = "http"
	LayerExternal       Layer = "external"
)

// ErrorType represents the category of error
type ErrorType string

const (
	ErrorTypeNotFound       ErrorType = "not_found"
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeConflict       ErrorType = "conflict"
	ErrorTypeUnauthorized   ErrorType = "unauthorized"
	ErrorTypeForbidden      ErrorType = "forbidden"
	ErrorTypeInternal       ErrorType = "internal"
	ErrorTypeBadRequest     ErrorType = "bad_request"
	ErrorTypeUnavailable    ErrorType = "unavailable"
	ErrorTypeTimeout        ErrorType = "timeout"
	ErrorTypeRateLimit      ErrorType = "rate_limit"
	ErrorTypeNotImplemented ErrorType = "not_implemented"
)

// PlatformError is the standard error type for the platform
type PlatformError struct {
	Layer     Layer     `json:"layer"`
	Type      ErrorType `json:"type"`
	Message   string    `json:"message"`
	Code      string    `json:"code,omitempty"`
	UUID      string    `json:"uuid,omitempty"`
	Wrapped   error     `json:"-"`
	RequestID string    `json:"request_id,omitempty"`
}

func (e *PlatformError) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("[%s/%s] %s: %v", e.Layer, e.Type, e.Message, e.Wrapped)
	}
	return fmt.Sprintf("[%s/%s] %s", e.Layer, e.Type, e.Message)
}

func (e *PlatformError) Unwrap() error {
	return e.Wrapped
}

// NewError creates a new PlatformError
func NewError(ctx context.Context, layer Layer, errType ErrorType, message string, wrapped error, uuid string) *PlatformError {
	requestID := ""
	if ctx != nil {
		if rid, ok := ctx.Value("request_id").(string); ok {
			requestID = rid
		}
	}
	return &PlatformError{
		Layer:     layer,
		Type:      errType,
		Message:   message,
		UUID:      uuid,
		Wrapped:   wrapped,
		RequestID: requestID,
	}
}

// AsError wraps an existing PlatformError with additional context
func AsError(ctx context.Context, layer Layer, message string, err error) *PlatformError {
	var platformErr *PlatformError
	if errors.As(err, &platformErr) {
		// Preserve the original error details but add context
		return &PlatformError{
			Layer:     layer,
			Type:      platformErr.Type,
			Message:   message + ": " + platformErr.Message,
			UUID:      platformErr.UUID,
			Wrapped:   err,
			RequestID: platformErr.RequestID,
		}
	}
	// Wrap unknown errors as internal errors
	return NewError(ctx, layer, ErrorTypeInternal, message, err, "")
}

// HTTPStatusCode returns the appropriate HTTP status code for the error type
func (e *PlatformError) HTTPStatusCode() int {
	switch e.Type {
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeValidation, ErrorTypeBadRequest:
		return http.StatusBadRequest
	case ErrorTypeConflict:
		return http.StatusConflict
	case ErrorTypeUnauthorized:
		return http.StatusUnauthorized
	case ErrorTypeForbidden:
		return http.StatusForbidden
	case ErrorTypeUnavailable:
		return http.StatusServiceUnavailable
	case ErrorTypeTimeout:
		return http.StatusGatewayTimeout
	case ErrorTypeRateLimit:
		return http.StatusTooManyRequests
	case ErrorTypeNotImplemented:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

// Common error constructors

func NotFound(ctx context.Context, layer Layer, resource string, uuid string) *PlatformError {
	return NewError(ctx, layer, ErrorTypeNotFound, resource+" not found", nil, uuid)
}

func Validation(ctx context.Context, layer Layer, message string, uuid string) *PlatformError {
	return NewError(ctx, layer, ErrorTypeValidation, message, nil, uuid)
}

func Conflict(ctx context.Context, layer Layer, message string, uuid string) *PlatformError {
	return NewError(ctx, layer, ErrorTypeConflict, message, nil, uuid)
}

func Unauthorized(ctx context.Context, layer Layer, message string, uuid string) *PlatformError {
	return NewError(ctx, layer, ErrorTypeUnauthorized, message, nil, uuid)
}

func Forbidden(ctx context.Context, layer Layer, message string, uuid string) *PlatformError {
	return NewError(ctx, layer, ErrorTypeForbidden, message, nil, uuid)
}

func Internal(ctx context.Context, layer Layer, message string, err error, uuid string) *PlatformError {
	return NewError(ctx, layer, ErrorTypeInternal, message, err, uuid)
}

// Is checks if an error is of a specific type
func Is(err error, errType ErrorType) bool {
	var platformErr *PlatformError
	if errors.As(err, &platformErr) {
		return platformErr.Type == errType
	}
	return false
}
