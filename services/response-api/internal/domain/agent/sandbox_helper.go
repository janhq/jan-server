package agent

import (
	"context"
	"fmt"
	"strings"

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
// Returns nil if mcpClient is nil.
func NewSandboxHelper(mcpClient MCPToolCaller) *SandboxHelper {
	if mcpClient == nil {
		return nil
	}
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
// ConversationID is required for workspace creation and is passed via X-Conversation-ID header.
func (h *SandboxHelper) EnsureSandboxStarted(ctx context.Context, opts StartSandboxOptions) error {
	if h == nil || h.mcpClient == nil {
		return fmt.Errorf("sandbox helper not configured")
	}

	// Validate required fields
	if opts.ConversationID == "" {
		return fmt.Errorf("conversation_id is required for sandbox startup")
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 1800 // Default 30 minutes
	}

	// Build arguments - conversation_id is passed via header, not in args
	args := map[string]interface{}{
		"timeout": timeout,
	}

	callReq := tool.CallRequest{
		Name:           "sandbox_start",
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
		if isSandboxToolUnavailable(err) {
			log.Debug().Msg("[sandbox] sandbox_start not available, skipping")
			return nil
		}
		return fmt.Errorf("start sandbox: %w", err)
	}

	if result.IsError {
		return fmt.Errorf("start sandbox failed: %s", extractSandboxErrorText(result))
	}

	log.Info().Msg("[sandbox] sandbox started/confirmed running")
	return nil
}

// PauseSandbox pauses the E2B sandbox to preserve state.
// Can be resumed within 30 days using sandbox_start.
// Note: UserID is extracted from JWT token by mcp-tools.
func (h *SandboxHelper) PauseSandbox(ctx context.Context, requestID, conversationID string) error {
	if h == nil || h.mcpClient == nil {
		return fmt.Errorf("sandbox helper not configured")
	}

	callReq := tool.CallRequest{
		Name:           "sandbox_pause",
		RequestID:      requestID,
		ConversationID: conversationID,
	}

	log.Info().
		Str("conversation_id", conversationID).
		Msg("[sandbox] pausing sandbox")

	result, err := h.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		if isSandboxToolUnavailable(err) {
			log.Debug().Msg("[sandbox] sandbox_pause not available, skipping")
			return nil
		}
		return fmt.Errorf("pause sandbox: %w", err)
	}

	if result.IsError {
		return fmt.Errorf("pause sandbox failed: %s", extractSandboxErrorText(result))
	}

	log.Info().Msg("[sandbox] sandbox paused successfully")
	return nil
}

// StopSandbox stops and deletes the E2B sandbox completely.
// All data in the sandbox will be lost.
// Note: UserID is extracted from JWT token by mcp-tools.
func (h *SandboxHelper) StopSandbox(ctx context.Context, requestID, conversationID string) error {
	if h == nil || h.mcpClient == nil {
		return fmt.Errorf("sandbox helper not configured")
	}

	callReq := tool.CallRequest{
		Name:           "sandbox_stop",
		RequestID:      requestID,
		ConversationID: conversationID,
	}

	log.Info().
		Str("conversation_id", conversationID).
		Msg("[sandbox] stopping sandbox")

	result, err := h.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		if isSandboxToolUnavailable(err) {
			log.Debug().Msg("[sandbox] sandbox_stop not available, skipping")
			return nil
		}
		return fmt.Errorf("stop sandbox: %w", err)
	}

	if result.IsError {
		return fmt.Errorf("stop sandbox failed: %s", extractSandboxErrorText(result))
	}

	log.Info().Msg("[sandbox] sandbox stopped successfully")
	return nil
}

// ExtendSandbox extends the sandbox timeout.
// Maximum total runtime is 24 hours from start.
// Note: UserID is extracted from JWT token by mcp-tools.
func (h *SandboxHelper) ExtendSandbox(ctx context.Context, additionalSeconds int, requestID, conversationID string) error {
	if h == nil || h.mcpClient == nil {
		return fmt.Errorf("sandbox helper not configured")
	}

	if additionalSeconds <= 0 {
		additionalSeconds = 1800 // Default 30 minutes
	}

	callReq := tool.CallRequest{
		Name: "sandbox_extend",
		Arguments: map[string]interface{}{
			"additional_seconds": additionalSeconds,
		},
		RequestID:      requestID,
		ConversationID: conversationID,
	}

	log.Info().
		Int("additional_seconds", additionalSeconds).
		Str("conversation_id", conversationID).
		Msg("[sandbox] extending sandbox timeout")

	result, err := h.mcpClient.CallTool(ctx, callReq)
	if err != nil {
		if isSandboxToolUnavailable(err) {
			log.Debug().Msg("[sandbox] sandbox_extend not available, skipping")
			return nil
		}
		return fmt.Errorf("extend sandbox: %w", err)
	}

	if result.IsError {
		return fmt.Errorf("extend sandbox failed: %s", extractSandboxErrorText(result))
	}

	log.Info().Msg("[sandbox] sandbox timeout extended")
	return nil
}

// extractSandboxErrorText extracts error message from tool result.
func extractSandboxErrorText(result *tool.Result) string {
	if result == nil || len(result.Content) == 0 {
		return "unknown error"
	}

	for _, content := range result.Content {
		if content.Type == "text" && content.Text != "" {
			return content.Text
		}
	}
	return "unknown error"
}

func isSandboxToolUnavailable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "mcp error (-32601)") ||
		strings.Contains(msg, "method not found") ||
		strings.Contains(msg, "tool not found")
}
