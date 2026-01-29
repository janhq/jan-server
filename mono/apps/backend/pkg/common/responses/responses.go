package responses

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	platformerrors "jan-server/mono/apps/backend/pkg/common/errors"
)

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Error struct {
		Message   string `json:"message"`
		Type      string `json:"type"`
		Code      string `json:"code,omitempty"`
		RequestID string `json:"request_id,omitempty"`
	} `json:"error"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// PaginatedResponse represents a paginated list response
type PaginatedResponse struct {
	Data   interface{} `json:"data"`
	Total  int64       `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

// HandleError handles errors and sends appropriate HTTP responses
func HandleError(c *gin.Context, err error) {
	var platformErr *platformerrors.PlatformError
	if errors.As(err, &platformErr) {
		resp := ErrorResponse{}
		resp.Error.Message = platformErr.Message
		resp.Error.Type = string(platformErr.Type)
		resp.Error.Code = platformErr.UUID
		resp.Error.RequestID = platformErr.RequestID

		c.JSON(platformErr.HTTPStatusCode(), resp)
		return
	}

	// Unknown error type - return as internal error
	resp := ErrorResponse{}
	resp.Error.Message = "An internal error occurred"
	resp.Error.Type = "internal_error"
	if requestID, exists := c.Get("request_id"); exists {
		resp.Error.RequestID = requestID.(string)
	}

	c.JSON(http.StatusInternalServerError, resp)
}

// OK sends a 200 OK response with data
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// Created sends a 201 Created response with data
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// NoContent sends a 204 No Content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// BadRequest sends a 400 Bad Request response
func BadRequest(c *gin.Context, message string) {
	resp := ErrorResponse{}
	resp.Error.Message = message
	resp.Error.Type = "bad_request"
	if requestID, exists := c.Get("request_id"); exists {
		resp.Error.RequestID = requestID.(string)
	}
	c.JSON(http.StatusBadRequest, resp)
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "Unauthorized"
	}
	resp := ErrorResponse{}
	resp.Error.Message = message
	resp.Error.Type = "unauthorized"
	if requestID, exists := c.Get("request_id"); exists {
		resp.Error.RequestID = requestID.(string)
	}
	c.JSON(http.StatusUnauthorized, resp)
}

// Forbidden sends a 403 Forbidden response
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "Forbidden"
	}
	resp := ErrorResponse{}
	resp.Error.Message = message
	resp.Error.Type = "forbidden"
	if requestID, exists := c.Get("request_id"); exists {
		resp.Error.RequestID = requestID.(string)
	}
	c.JSON(http.StatusForbidden, resp)
}

// NotFound sends a 404 Not Found response
func NotFound(c *gin.Context, resource string) {
	message := "Resource not found"
	if resource != "" {
		message = resource + " not found"
	}
	resp := ErrorResponse{}
	resp.Error.Message = message
	resp.Error.Type = "not_found"
	if requestID, exists := c.Get("request_id"); exists {
		resp.Error.RequestID = requestID.(string)
	}
	c.JSON(http.StatusNotFound, resp)
}

// InternalError sends a 500 Internal Server Error response
func InternalError(c *gin.Context, message string) {
	if message == "" {
		message = "An internal error occurred"
	}
	resp := ErrorResponse{}
	resp.Error.Message = message
	resp.Error.Type = "internal_error"
	if requestID, exists := c.Get("request_id"); exists {
		resp.Error.RequestID = requestID.(string)
	}
	c.JSON(http.StatusInternalServerError, resp)
}

// Paginated sends a paginated response
func Paginated(c *gin.Context, data interface{}, total int64, limit, offset int) {
	c.JSON(http.StatusOK, PaginatedResponse{
		Data:   data,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}
