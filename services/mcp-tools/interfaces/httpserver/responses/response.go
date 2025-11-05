package responses

import (
	"errors"
	"net/http"
	"time"

	"jan-server/services/mcp-tools/utils/platformerrors"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Code          string `json:"code"` // UUID from PlatformError
	Error         string `json:"error"`
	ErrorInstance error  `json:"-"`
	RequestID     string `json:"request_id,omitempty"`
}

func NewInternalServerError(reqCtx *gin.Context, errResp ErrorResponse) {
	if errResp.ErrorInstance != nil {
		reqCtx.Error(errResp.ErrorInstance)
	}
	reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, errResp)
}

// HandleError handles domain errors and returns appropriate HTTP responses
// The message parameter is used directly as the error message in the response
// Status code is automatically determined from the error type
func HandleError(reqCtx *gin.Context, err error, message string) {
	var domainErr *platformerrors.PlatformError
	if errors.As(err, &domainErr) {
		statusCode := platformerrors.ErrorTypeToHTTPStatus(domainErr.GetErrorType())

		errResp := ErrorResponse{
			Code:          domainErr.GetUUID(),
			Error:         message,
			ErrorInstance: domainErr,
			RequestID:     domainErr.GetRequestID(),
		}

		reqCtx.AbortWithStatusJSON(statusCode, errResp)
		return
	} else {
		// assign generic error response for non-domain errors
		errResp := ErrorResponse{
			Error:         message,
			ErrorInstance: err,
		}
		reqCtx.AbortWithStatusJSON(http.StatusInternalServerError, errResp)
	}
}

// HandleErrorWithStatus handles domain errors with a custom status code
// Use this when you need to override the default status code mapping
func HandleErrorWithStatus(reqCtx *gin.Context, statusCode int, err error, message string) {
	var domainErr *platformerrors.PlatformError
	if errors.As(err, &domainErr) {
		errResp := ErrorResponse{
			Code:          domainErr.GetUUID(),
			Error:         message,
			ErrorInstance: domainErr,
			RequestID:     domainErr.GetRequestID(),
		}

		reqCtx.AbortWithStatusJSON(statusCode, errResp)
		return
	} else {
		// assign generic error response for non-domain errors
		errResp := ErrorResponse{
			Error:         message,
			ErrorInstance: err,
		}
		reqCtx.AbortWithStatusJSON(statusCode, errResp)
	}
}

// HandleNewError creates a new typed error at the route layer and handles it
// This is a convenience function for route-level validations and errors
// The uuid parameter should be provided from the route for error tracking
func HandleNewError(reqCtx *gin.Context, errorType platformerrors.ErrorType, message string, uuid string) {
	ctx := reqCtx.Request.Context()
	// Use the provided UUID
	err := platformerrors.NewError(ctx, platformerrors.LayerRoute, errorType, message, nil, uuid)

	statusCode := platformerrors.ErrorTypeToHTTPStatus(err.GetErrorType())

	errResp := ErrorResponse{
		Code:          err.GetUUID(),
		Error:         message,
		ErrorInstance: err,
		RequestID:     err.GetRequestID(),
	}

	reqCtx.AbortWithStatusJSON(statusCode, errResp)
}

type GeneralResponse[T any] struct {
	Status string `json:"status"`
	Result T      `json:"result"`
}

type ListResponse[T any] struct {
	Total   int64   `json:"total"`
	Results []T     `json:"results"`
	FirstID *string `json:"first_id"`
	LastID  *string `json:"last_id"`
	HasMore bool    `json:"has_more"`
}

type PageCursor struct {
	FirstID *string
	LastID  *string
	HasMore bool
	Total   int64
}

func BuildCursorPage[T any](
	items []*T,
	getID func(*T) *string,
	hasMoreFunc func() ([]*T, error),
	CountFunc func() (int64, error),
) (*PageCursor, error) {
	cursorPage := &PageCursor{}
	if len(items) > 0 {
		cursorPage.FirstID = getID(items[0])
		cursorPage.LastID = getID(items[len(items)-1])
		moreRecords, err := hasMoreFunc()
		if len(moreRecords) > 0 {
			cursorPage.HasMore = true
		}
		if err != nil {
			return nil, err
		}
	}
	count, err := CountFunc()
	if err != nil {
		return cursorPage, err
	}
	cursorPage.Total = count
	return cursorPage, nil
}

func NewCookieWithSecurity(name string, value string, expires time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Expires:  expires,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	}
}
