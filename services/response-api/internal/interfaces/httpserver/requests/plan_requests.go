package requests

// CancelPlanRequest represents a request to cancel a plan.
type CancelPlanRequest struct {
	Reason string `json:"reason,omitempty"`
}

// UserInputRequest represents user input to resume a waiting plan.
type UserInputRequest struct {
	Selection string  `json:"selection,omitempty"`
	Approval  *bool   `json:"approval,omitempty"`
	Message   *string `json:"message,omitempty"`
}
