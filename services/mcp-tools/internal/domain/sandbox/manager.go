package sandbox

import (
	"context"
	"time"
)

// Manager defines the interface for sandbox lifecycle management.
// This is separate from Provider because not all backends support lifecycle management.
// Currently only E2B supports this interface; AIO does not have lifecycle management.
type Manager interface {
	// Lifecycle operations (E2B only)

	// Start creates or resumes a sandbox for the given user and conversation.
	// Returns sandbox info including view URL for desktop access.
	// If sandbox already exists and is running, returns current state.
	// If sandbox is paused, resumes it.
	// If sandbox is stopped, creates a new one.
	Start(ctx context.Context, userID, conversationID string, timeout int) (*SandboxInfo, error)

	// Stop terminates and deletes a user's sandbox.
	// All data in the sandbox is lost.
	Stop(ctx context.Context, userID string) error

	// Pause pauses a running sandbox.
	// Paused sandboxes can be resumed within 30 days (E2B limit).
	// This saves compute costs while preserving state.
	Pause(ctx context.Context, userID string) error

	// Extend extends the timeout of a running sandbox.
	// additionalSeconds is added to the current timeout.
	// Max runtime is 24 hours from start (E2B limit).
	Extend(ctx context.Context, userID string, additionalSeconds int) (*SandboxInfo, error)

	// State operations

	// GetState returns the current state of a user's sandbox.
	// Returns nil if no sandbox exists for the user.
	GetState(ctx context.Context, userID string) (*SandboxState, error)

	// IsRunning returns true if the user has a running sandbox.
	// This is a convenience method for checking state.
	IsRunning(ctx context.Context, userID string) (bool, error)

	// Dynamic tools

	// GetDynamicTools fetches tools from MCP servers running inside the sandbox.
	// These are tools like browser automation from search-mcp-server.
	// Returns empty slice if sandbox is not running or has no dynamic tools.
	GetDynamicTools(ctx context.Context, userID string) ([]DynamicTool, error)

	// Tool execution

	// CallTool executes a tool via MCP protocol in the sandbox.
	// Used for dynamic tools like browser automation.
	// toolName is the actual tool name in the sandbox (without browser_ prefix).
	CallTool(ctx context.Context, userID, toolName string, args map[string]interface{}) (map[string]interface{}, error)
}

// SandboxInfo contains information about a sandbox after lifecycle operations.
type SandboxInfo struct {
	SandboxID  string    `json:"sandbox_id"`
	Status     string    `json:"status"` // "running", "paused", "stopped"
	ViewURL    string    `json:"view_url,omitempty"`
	ControlURL string    `json:"control_url,omitempty"`
	TimeoutAt  time.Time `json:"timeout_at,omitempty"`
	StartedAt  time.Time `json:"started_at,omitempty"`
}

// SandboxState represents the current state of a sandbox.
type SandboxState struct {
	Status         string    `json:"status"` // "running", "paused", "stopped", ""
	SandboxID      string    `json:"sandbox_id,omitempty"`
	ViewURL        string    `json:"view_url,omitempty"`
	ControlURL     string    `json:"control_url,omitempty"`
	TimeoutAt      time.Time `json:"timeout_at,omitempty"`
	StartedAt      time.Time `json:"started_at,omitempty"`
	PausedAt       time.Time `json:"paused_at,omitempty"`
	PauseExpiresAt time.Time `json:"pause_expires_at,omitempty"`
}

// IsHealthy returns true if the sandbox is in a healthy running state.
func (s *SandboxState) IsHealthy() bool {
	return s != nil && s.Status == "running"
}

// DynamicTool represents a tool discovered from MCP servers inside the sandbox.
type DynamicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ErrSandboxNotFound is returned when a sandbox doesn't exist for a user.
type ErrSandboxNotFound struct {
	UserID string
}

func (e *ErrSandboxNotFound) Error() string {
	return "sandbox not found for user: " + e.UserID
}

// NewSandboxNotFoundError creates a new ErrSandboxNotFound error.
func NewSandboxNotFoundError(userID string) error {
	return &ErrSandboxNotFound{UserID: userID}
}

// ErrSandboxStateError is returned when an operation is invalid for current state.
type ErrSandboxStateError struct {
	Operation     string
	CurrentStatus string
	Message       string
}

func (e *ErrSandboxStateError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "cannot " + e.Operation + " sandbox in state: " + e.CurrentStatus
}

// NewSandboxStateError creates a new ErrSandboxStateError error.
func NewSandboxStateError(operation, currentStatus, message string) error {
	return &ErrSandboxStateError{
		Operation:     operation,
		CurrentStatus: currentStatus,
		Message:       message,
	}
}
