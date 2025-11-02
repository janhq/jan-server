package responses

// ErrorType enumerates standard error responses.
type ErrorType string

const (
	ErrorTypeInvalidRequest ErrorType = "invalid_request_error"
	ErrorTypeAuth           ErrorType = "auth_error"
	ErrorTypeRateLimit      ErrorType = "rate_limit_error"
	ErrorTypeInternal       ErrorType = "internal_error"
)

// ErrorResponse is the canonical error payload returned to clients.
type ErrorResponse struct {
	Type      ErrorType `json:"type"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Param     string    `json:"param,omitempty"`
	RequestID string    `json:"request_id"`
}
