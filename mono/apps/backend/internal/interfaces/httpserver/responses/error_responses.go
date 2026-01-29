package responses

import (
	"errors"
	"net/http"

	"jan-server/mono/apps/backend/internal/utils/platformerrors"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents an error response with platform error details
type ErrorResponse struct {
	Code          string `json:"code"` // UUID from PlatformError
	Error         string `json:"error"`
	Message       string `json:"message,omitempty"`
	ErrorInstance error  `json:"-"`
	RequestID     string `json:"request_id,omitempty"`
}

// HandleError handles domain errors and returns appropriate HTTP responses
func HandleError(reqCtx *gin.Context, err error, message string) {
	var domainErr *platformerrors.PlatformError
	if errors.As(err, &domainErr) {
		statusCode := platformerrors.ErrorTypeToHTTPStatus(domainErr.GetErrorType())

		errorMessage := domainErr.Message
		if errorMessage == "" {
			errorMessage = message
		}

		errResp := ErrorResponse{
			Code:          domainErr.GetUUID(),
			Error:         errorMessage,
			Message:       errorMessage,
			ErrorInstance: domainErr,
			RequestID:     domainErr.GetRequestID(),
		}

		reqCtx.AbortWithStatusJSON(statusCode, errResp)
		return
	}
	// Non-platform errors
	errResp := ErrorResponse{
		Error:         message,
		Message:       message,
		ErrorInstance: err,
	}
	reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, errResp)
}

// HandleNewError creates a new typed error at the route layer and handles it
func HandleNewError(reqCtx *gin.Context, errorType platformerrors.ErrorType, message string, uuid string) {
	ctx := reqCtx.Request.Context()
	err := platformerrors.NewError(ctx, platformerrors.LayerRoute, errorType, message, nil, uuid)

	statusCode := platformerrors.ErrorTypeToHTTPStatus(err.GetErrorType())

	errResp := ErrorResponse{
		Code:          err.GetUUID(),
		Error:         message,
		Message:       message,
		ErrorInstance: err,
		RequestID:     err.GetRequestID(),
	}

	reqCtx.AbortWithStatusJSON(statusCode, errResp)
}

// HandleErrorWithStatus handles errors with a custom status code
func HandleErrorWithStatus(reqCtx *gin.Context, statusCode int, err error, message string) {
	var domainErr *platformerrors.PlatformError
	if errors.As(err, &domainErr) {
		errResp := ErrorResponse{
			Code:          domainErr.GetUUID(),
			Error:         message,
			Message:       message,
			ErrorInstance: domainErr,
			RequestID:     domainErr.GetRequestID(),
		}
		reqCtx.AbortWithStatusJSON(statusCode, errResp)
		return
	}
	errResp := ErrorResponse{
		Error:         message,
		Message:       message,
		ErrorInstance: err,
	}
	reqCtx.AbortWithStatusJSON(statusCode, errResp)
}
