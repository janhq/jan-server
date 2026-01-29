package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sandboxdomain "jan-server/services/mcp-tools/internal/domain/sandbox"
	"jan-server/services/mcp-tools/internal/infrastructure/metrics"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

// SandboxManagement handles sandbox lifecycle management tools.
// These tools are only available when SANDBOX_PROVIDER=e2b.
type SandboxManagement struct {
	manager sandboxdomain.Manager
}

// Tool descriptions - single source of truth
const (
	descSandboxStart  = "Start or resume a sandbox environment. Creates a new sandbox if none exists, resumes if paused, or returns current state if already running. Returns sandbox info including view_url for desktop access."
	descSandboxStop   = "Stop and delete the sandbox. All data in the sandbox will be lost. Use sandbox_pause instead if you want to preserve state."
	descSandboxPause  = "Pause the sandbox to save compute costs while preserving state. Paused sandboxes can be resumed within 30 days using sandbox_start."
	descSandboxExtend = "Extend the timeout of a running sandbox. Adds additional time to the current timeout. Maximum total runtime is 24 hours from start."
)

// NewSandboxManagement creates a new sandbox management handler.
// Returns nil if manager is nil (e.g., when using AIO provider).
func NewSandboxManagement(manager sandboxdomain.Manager) *SandboxManagement {
	if manager == nil {
		return nil
	}
	return &SandboxManagement{manager: manager}
}

// IsEnabled returns true if sandbox management is available.
func (m *SandboxManagement) IsEnabled() bool {
	return m != nil && m.manager != nil
}

// RegisterTools registers sandbox management tools with the MCP server.
func (m *SandboxManagement) RegisterTools(server *mcpsdk.Server) {
	if !m.IsEnabled() {
		return
	}

	m.registerStart(server)
	m.registerStop(server)
	m.registerPause(server)
	m.registerExtend(server)

	log.Info().Msg("Sandbox management tools registered (E2B)")
}

// GetManagementToolDefinitions returns the tool definitions for sandbox management.
// Used by MCPRoute to include these tools in tools/list response.
func (m *SandboxManagement) GetManagementToolDefinitions() []ToolDefinition {
	if !m.IsEnabled() {
		return nil
	}

	return []ToolDefinition{
		{
			Name:        "sandbox_start",
			Description: descSandboxStart,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"timeout": map[string]interface{}{
						"type":        "integer",
						"description": "Optional timeout in seconds (default: 1800, max: 86400)",
					},
				},
			},
		},
		{
			Name:        "sandbox_stop",
			Description: descSandboxStop,
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "sandbox_pause",
			Description: descSandboxPause,
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "sandbox_extend",
			Description: descSandboxExtend,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"additional_seconds": map[string]interface{}{
						"type":        "integer",
						"description": "Number of seconds to add to current timeout (required)",
					},
				},
				"required": []string{"additional_seconds"},
			},
		},
	}
}

// ToolDefinition is an alias for toolInfo for backwards compatibility.
// Both types represent tool definitions for MCP tools/list response.
type ToolDefinition = toolInfo

// IsSandboxManagementTool returns true if the tool name is a sandbox management tool.
func IsSandboxManagementTool(name string) bool {
	switch name {
	case "sandbox_start", "sandbox_stop", "sandbox_pause", "sandbox_extend":
		return true
	}
	return false
}

// --- Tool Implementations ---

type SandboxStartArgs struct {
	Timeout int `json:"timeout,omitempty"`
}

func (m *SandboxManagement) registerStart(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_start",
		Description: descSandboxStart,
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SandboxStartArgs) (*mcpsdk.CallToolResult, any, error) {
		startTime := time.Now()

		// Extract user context
		userID := getUserIDFromContext(ctx)
		conversationID := getConversationIDFromContext(ctx)

		if userID == "" {
			return nil, nil, fmt.Errorf("user_id is required (from JWT)")
		}
		if conversationID == "" {
			return nil, nil, fmt.Errorf("conversation_id is required (from X-Conversation-ID header)")
		}

		log.Info().
			Str("tool", "sandbox_start").
			Str("user_id", userID).
			Str("conversation_id", conversationID).
			Int("timeout", input.Timeout).
			Msg("Sandbox start requested")

		info, err := m.manager.Start(ctx, userID, conversationID, input.Timeout)

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("sandbox_start", "e2b", status, duration.Seconds())

		if err != nil {
			log.Error().Err(err).Str("tool", "sandbox_start").Msg("Sandbox start failed")
			return nil, nil, fmt.Errorf("sandbox start failed: %w", err)
		}

		output := map[string]any{
			"sandbox_id":  info.SandboxID,
			"status":      info.Status,
			"view_url":    info.ViewURL,
			"control_url": info.ControlURL,
			"started_at":  info.StartedAt.Format(time.RFC3339),
			"timeout_at":  info.TimeoutAt.Format(time.RFC3339),
		}

		outputJSON, _ := json.Marshal(output)
		log.Info().
			Str("tool", "sandbox_start").
			Str("sandbox_id", info.SandboxID).
			Str("status", info.Status).
			Msg("Sandbox started successfully")

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

type SandboxStopArgs struct{}

func (m *SandboxManagement) registerStop(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_stop",
		Description: descSandboxStop,
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SandboxStopArgs) (*mcpsdk.CallToolResult, any, error) {
		startTime := time.Now()

		userID := getUserIDFromContext(ctx)
		if userID == "" {
			return nil, nil, fmt.Errorf("user_id is required (from JWT)")
		}

		log.Info().
			Str("tool", "sandbox_stop").
			Str("user_id", userID).
			Msg("Sandbox stop requested")

		err := m.manager.Stop(ctx, userID)

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("sandbox_stop", "e2b", status, duration.Seconds())

		if err != nil {
			log.Error().Err(err).Str("tool", "sandbox_stop").Msg("Sandbox stop failed")
			return nil, nil, fmt.Errorf("sandbox stop failed: %w", err)
		}

		output := map[string]any{
			"status":  "stopped",
			"message": "Sandbox stopped and deleted successfully",
		}

		outputJSON, _ := json.Marshal(output)
		log.Info().Str("tool", "sandbox_stop").Msg("Sandbox stopped successfully")

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

type SandboxPauseArgs struct{}

func (m *SandboxManagement) registerPause(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_pause",
		Description: descSandboxPause,
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SandboxPauseArgs) (*mcpsdk.CallToolResult, any, error) {
		startTime := time.Now()

		userID := getUserIDFromContext(ctx)
		if userID == "" {
			return nil, nil, fmt.Errorf("user_id is required (from JWT)")
		}

		log.Info().
			Str("tool", "sandbox_pause").
			Str("user_id", userID).
			Msg("Sandbox pause requested")

		err := m.manager.Pause(ctx, userID)

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("sandbox_pause", "e2b", status, duration.Seconds())

		if err != nil {
			log.Error().Err(err).Str("tool", "sandbox_pause").Msg("Sandbox pause failed")
			return nil, nil, fmt.Errorf("sandbox pause failed: %w", err)
		}

		output := map[string]any{
			"status":  "paused",
			"message": "Sandbox paused. Resume within 30 days using sandbox_start.",
		}

		outputJSON, _ := json.Marshal(output)
		log.Info().Str("tool", "sandbox_pause").Msg("Sandbox paused successfully")

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

type SandboxExtendArgs struct {
	AdditionalSeconds int `json:"additional_seconds"`
}

func (m *SandboxManagement) registerExtend(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_extend",
		Description: descSandboxExtend,
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, input SandboxExtendArgs) (*mcpsdk.CallToolResult, any, error) {
		startTime := time.Now()

		userID := getUserIDFromContext(ctx)
		if userID == "" {
			return nil, nil, fmt.Errorf("user_id is required (from JWT)")
		}

		if input.AdditionalSeconds <= 0 {
			return nil, nil, fmt.Errorf("additional_seconds must be positive")
		}

		log.Info().
			Str("tool", "sandbox_extend").
			Str("user_id", userID).
			Int("additional_seconds", input.AdditionalSeconds).
			Msg("Sandbox extend requested")

		info, err := m.manager.Extend(ctx, userID, input.AdditionalSeconds)

		duration := time.Since(startTime)
		status := "success"
		if err != nil {
			status = "error"
		}
		metrics.RecordToolCall("sandbox_extend", "e2b", status, duration.Seconds())

		if err != nil {
			log.Error().Err(err).Str("tool", "sandbox_extend").Msg("Sandbox extend failed")
			return nil, nil, fmt.Errorf("sandbox extend failed: %w", err)
		}

		output := map[string]any{
			"status":     info.Status,
			"timeout_at": info.TimeoutAt.Format(time.RFC3339),
			"message":    fmt.Sprintf("Sandbox timeout extended by %d seconds", input.AdditionalSeconds),
		}

		outputJSON, _ := json.Marshal(output)
		log.Info().
			Str("tool", "sandbox_extend").
			Str("new_timeout_at", info.TimeoutAt.Format(time.RFC3339)).
			Msg("Sandbox extended successfully")

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(outputJSON)}},
		}, nil, nil
	})
}

// --- Helper Functions ---

// getUserIDFromContext extracts user_id from context (set by JWT middleware)
func getUserIDFromContext(ctx context.Context) string {
	if val := ctx.Value("user_id"); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getConversationIDFromContext extracts conversation_id from context (X-Conversation-ID header)
func getConversationIDFromContext(ctx context.Context) string {
	return GetConversationID(ctx)
}
