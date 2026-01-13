package responses

import (
	"errors"
	"net/http"

	"jan-server/services/response-api/internal/domain/response"
	"jan-server/services/response-api/internal/utils/platformerrors"

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

// ResponsePayload is returned to clients.
type ResponsePayload struct {
	ID                 string                 `json:"id"`
	Object             string                 `json:"object"`
	Created            int64                  `json:"created"`
	CreatedAt          int64                  `json:"created_at"` // Same as Created, for compatibility
	Model              string                 `json:"model"`
	Status             string                 `json:"status"`
	Input              interface{}            `json:"input"`
	Output             interface{}            `json:"output,omitempty"`
	Usage              interface{}            `json:"usage,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	ConversationID     *string                `json:"conversation_id,omitempty"`
	PreviousResponseID *string                `json:"previous_response_id,omitempty"`
	SystemPrompt       *string                `json:"system_prompt,omitempty"`
	Stream             bool                   `json:"stream"`
	Background         bool                   `json:"background"`
	Store              bool                   `json:"store"`
	Error              interface{}            `json:"error,omitempty"`
}

// FromDomain maps the domain response to DTO.
func FromDomain(r *response.Response) ResponsePayload {
	createdUnix := r.CreatedAt.Unix()
	return ResponsePayload{
		ID:                 r.PublicID,
		Object:             r.Object,
		Created:            createdUnix,
		CreatedAt:          createdUnix, // Duplicate for compatibility
		Model:              r.Model,
		Status:             string(r.Status),
		Input:              r.Input,
		Output:             r.Output,
		Usage:              r.Usage,
		Metadata:           r.Metadata,
		ConversationID:     r.ConversationPublicID,
		PreviousResponseID: r.PreviousResponseID,
		SystemPrompt:       r.SystemPrompt,
		Stream:             r.Stream,
		Background:         r.Background,
		Store:              r.Store,
		Error:              r.Error,
	}
}

// ConversationItemsResponse wraps conversation input items for consistent responses.
type ConversationItemsResponse struct {
	Data []response.ConversationItem `json:"data"`
}

// FullResponsePayload combines response with plan details in one payload.
type FullResponsePayload struct {
	ResponsePayload
	Plan *PlanDetailResponse `json:"plan,omitempty"`
}

// FromDomainWithPlan maps the domain response with plan to DTO.
func FromDomainWithPlan(rwp *response.ResponseWithPlan, planMapper func(interface{}) *PlanDetailResponse) FullResponsePayload {
	payload := FullResponsePayload{
		ResponsePayload: FromDomain(rwp.Response),
	}
	if rwp.PlanDetails != nil && planMapper != nil {
		payload.Plan = planMapper(rwp.PlanDetails)
	}
	return payload
}
