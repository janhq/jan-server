package agent

import (
	"context"
	"fmt"

	"jan-server/services/response-api/internal/domain/tool"

	"github.com/rs/zerolog/log"
)

// SandboxHelper provides reusable sandbox lifecycle operations.
// It wraps MCP tool calls to manage E2B sandbox state.
type SandboxHelper struct {
	mcpClient MCPToolCaller
}

// MCPToolCaller interface for calling MCP tools.
type MCPToolCaller interface {
	CallTool(ctx context.Context, req tool.CallRequest) (*tool.Result, error)
}

// NewSandboxHelper creates a new sandbox helper.
func NewSandboxHelper(mcpClient MCPToolCaller) *SandboxHelper {
	return &SandboxHelper{mcpClient: mcpClient}
}

// StartSandboxOptions configures sandbox startup.
// Note: UserID is extracted from JWT token by mcp-tools, not passed explicitly.
type StartSandboxOptions struct {
	Timeout        int    // Timeout in seconds (default: 1800 = 30 min)
	ConversationID string // Required: creates workspace for this conversation
	RequestID      string // Optional: for request tracking
}

// EnsureSandboxStarted starts the E2B sandbox if not already running.
// The call is idempotent - if sandbox is already running, it returns current status.
// ConversationID is required for workspace creation.
func (h *SandboxHelper) EnsureSandboxStarted(ctx context.Context, opts StartSandboxOptions) error {
	if h.mcpClient == nil {
		return fmt.Errorf("MCP client not configured")
	}

	// Validate required fields
	if opts.ConversationID == "" {
		return fmt.Errorf("conversation_id is required for sandbox startup")
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 1800 // Default 30 minutes
	}

	args := map[string]interface{}{
		"timeout":         timeout,
		"conversation_id": opts.ConversationID,
	}

	callReq := tool.CallRequest{
		Name:           "e2b_sandbox_start",
		Arguments:      args,
		RequestID:      opts.RequestID,
		ConversationID: opts.ConversationID,
	}

	log.Info().
		Int("timeout", timeout).
		Str("conversation_id", opts.ConversationID).
		Msg("[sandbox] ensuring sandbox is started")

	result, err := h.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		return fmt.Errorf("start sandbox: %w", err)
	}

	if result.IsError {
		return fmt.Errorf("start sandbox failed: %s", extractErrorText(result))
	}

	log.Info().Msg("[sandbox] sandbox started/confirmed running")
	return nil
}

// PauseSandbox pauses the E2B sandbox to preserve state.
// Can be resumed within 30 days.
// Note: UserID is extracted from JWT token by mcp-tools.
func (h *SandboxHelper) PauseSandbox(ctx context.Context, requestID, conversationID string) error {
	if h.mcpClient == nil {
		return fmt.Errorf("MCP client not configured")
	}

	callReq := tool.CallRequest{
		Name:           "e2b_sandbox_pause",
		RequestID:      requestID,
		ConversationID: conversationID,
	}

	log.Info().Msg("[sandbox] pausing sandbox")

	result, err := h.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		return fmt.Errorf("pause sandbox: %w", err)
	}

	if result.IsError {
		return fmt.Errorf("pause sandbox failed: %s", extractErrorText(result))
	}

	log.Info().Msg("[sandbox] sandbox paused successfully")
	return nil
}

// KillSandbox kills the E2B sandbox completely.
// Note: UserID is extracted from JWT token by mcp-tools.
func (h *SandboxHelper) KillSandbox(ctx context.Context, requestID, conversationID string) error {
	if h.mcpClient == nil {
		return fmt.Errorf("MCP client not configured")
	}

	callReq := tool.CallRequest{
		Name:           "e2b_sandbox_stop",
		RequestID:      requestID,
		ConversationID: conversationID,
	}

	log.Info().Msg("[sandbox] killing sandbox")

	result, err := h.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		return fmt.Errorf("kill sandbox: %w", err)
	}

	if result.IsError {
		return fmt.Errorf("kill sandbox failed: %s", extractErrorText(result))
	}

	log.Info().Msg("[sandbox] sandbox killed successfully")
	return nil
}

// ExtendSandbox extends the sandbox timeout.
// Note: UserID is extracted from JWT token by mcp-tools.
func (h *SandboxHelper) ExtendSandbox(ctx context.Context, additionalSeconds int, requestID, conversationID string) error {
	if h.mcpClient == nil {
		return fmt.Errorf("MCP client not configured")
	}

	if additionalSeconds <= 0 {
		additionalSeconds = 1800 // Default 30 minutes
	}

	callReq := tool.CallRequest{
		Name: "e2b_sandbox_extend",
		Arguments: map[string]interface{}{
			"additional_seconds": additionalSeconds,
		},
		RequestID:      requestID,
		ConversationID: conversationID,
	}

	log.Info().Int("additional_seconds", additionalSeconds).Msg("[sandbox] extending sandbox timeout")

	result, err := h.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		return fmt.Errorf("extend sandbox: %w", err)
	}

	if result.IsError {
		return fmt.Errorf("extend sandbox failed: %s", extractErrorText(result))
	}

	log.Info().Msg("[sandbox] sandbox timeout extended")
	return nil
}

// extractErrorText extracts error message from tool result.
func extractErrorText(result *tool.Result) string {
	if result == nil || len(result.Content) == 0 {
		return "unknown error"
	}

	var texts []string
	for _, content := range result.Content {
		if content.Type == "text" && content.Text != "" {
			texts = append(texts, content.Text)
		}
	}
	if len(texts) == 0 {
		return "unknown error"
	}
	return texts[0]
}
