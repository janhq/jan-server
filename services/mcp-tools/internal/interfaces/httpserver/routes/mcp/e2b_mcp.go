package mcp

import (
	"context"
	"fmt"
	"strings"

	"jan-server/services/mcp-tools/internal/infrastructure/e2b"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

// e2bDesktopTools are all hardcoded static tools from e2b-api
// These are shown when user has conversation context
var e2bDesktopTools = map[string]bool{
	"e2b_desktop_shell":        true,
	"e2b_desktop_file_read":    true,
	"e2b_desktop_file_write":   true,
	"e2b_desktop_file_edit":    true,
	"e2b_desktop_file_list":    true,
	"e2b_desktop_screenshot":   true,
	"e2b_desktop_click":        true,
	"e2b_desktop_type":         true,
	"e2b_desktop_code_execute": true,
	"e2b_desktop_packages":     true,
}

// e2bSandboxTools are lifecycle management tools
// These are always shown when user is authenticated
var e2bSandboxTools = map[string]bool{
	"e2b_sandbox_start":  true,
	"e2b_sandbox_stop":   true,
	"e2b_sandbox_pause":  true,
	"e2b_sandbox_extend": true,
}

// e2bStaticTools combines all static tool types for lookup
var e2bStaticTools = func() map[string]bool {
	m := make(map[string]bool)
	for k := range e2bDesktopTools {
		m[k] = true
	}
	for k := range e2bSandboxTools {
		m[k] = true
	}
	return m
}()

// e2bSandboxToolDefs are sandbox lifecycle management tools
// These work without a running sandbox and are always available with auth
var e2bSandboxToolDefs = []Tool{
	{
		Name:        "e2b_sandbox_start",
		Description: "Start or resume E2B sandbox for the user. Creates workspace for the conversation. Returns VNC URLs for frontend display.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"timeout": map[string]any{
					"type":        "integer",
					"description": "Sandbox timeout in seconds (default: 1800, max: 86400)",
				},
				"conversation_id": map[string]any{
					"type":        "string",
					"description": "Conversation ID to create workspace (required)",
				},
			},
			"required": []string{"conversation_id"},
		},
	},
	{
		Name:        "e2b_sandbox_pause",
		Description: "Pause the user's running sandbox. Can be resumed within 30 days.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	{
		Name:        "e2b_sandbox_stop",
		Description: "Stop and delete the user's sandbox completely.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	{
		Name:        "e2b_sandbox_extend",
		Description: "Extend the user's sandbox timeout.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"additional_seconds": map[string]any{
					"type":        "integer",
					"description": "Additional seconds to add (60-86400)",
				},
			},
			"required": []string{"additional_seconds"},
		},
	},
}

// E2BMCP handles E2B Sandbox tools via proxy to e2b-api.
// Tools are NOT registered statically - they are fetched dynamically per-request.
type E2BMCP struct {
	client  *e2b.Client
	enabled bool
}

// NewE2BMCP creates a new E2B MCP handler
func NewE2BMCP(client *e2b.Client, _ *e2b.WorkspaceResolver, enabled bool) *E2BMCP {
	if client == nil || !client.IsEnabled() {
		return nil
	}
	return &E2BMCP{client: client, enabled: enabled}
}

// IsEnabled returns whether E2B is enabled and ready
func (e *E2BMCP) IsEnabled() bool {
	return e != nil && e.enabled && e.client != nil && e.client.IsEnabled()
}

// GetClient returns the E2B client
func (e *E2BMCP) GetClient() *e2b.Client {
	if e == nil {
		return nil
	}
	return e.client
}

// SetLLMClient is a no-op (kept for interface compatibility)
func (e *E2BMCP) SetLLMClient(_ interface{}) {}

// RegisterTools is a no-op - E2B tools are fetched dynamically per-request
func (e *E2BMCP) RegisterTools(_ *mcpsdk.Server) {
	if e == nil || !e.enabled {
		return
	}
	log.Info().
		Str("url", e.client.BaseURL()).
		Msg("E2B tools will be fetched dynamically (no static registration)")
}

// GetSandboxTools returns only sandbox lifecycle tools (e2b_sandbox_*)
// Context fields (conversation_id, user_id) are stripped from schemas
// since mcp-tools auto-injects them when executing tools.
func (e *E2BMCP) GetSandboxTools() []Tool {
	if !e.IsEnabled() {
		return nil
	}
	// Return tools with context fields stripped from schemas
	var tools []Tool
	for _, t := range e2bSandboxToolDefs {
		tools = append(tools, Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: stripContextFields(t.InputSchema),
		})
	}
	return tools
}

// GetToolsForUser fetches all tools for a user with conversation context.
// This includes:
// - e2b_sandbox_* (lifecycle tools)
// - e2b_desktop_* (workspace tools from e2b-api static)
// - e2b_* (browser tools + dynamic tools from sandbox mcp-proxy)
//
// Context fields (conversation_id, user_id) are stripped from all tool schemas
// since mcp-tools auto-injects them when executing tools.
func (e *E2BMCP) GetToolsForUser(ctx context.Context, userID string, conversationID string) ([]Tool, error) {
	if !e.IsEnabled() {
		return nil, fmt.Errorf("E2B not enabled")
	}

	var tools []Tool

	// Always include sandbox lifecycle tools (with context fields stripped)
	for _, t := range e2bSandboxToolDefs {
		tools = append(tools, Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: stripContextFields(t.InputSchema),
		})
	}

	// Try to fetch tools from e2b-api (static + dynamic from sandbox mcp-proxy)
	resp, err := e.client.CallUserMCP(ctx, userID, "tools/list", nil)
	if err != nil {
		log.Debug().Err(err).Str("user_id", userID).Msg("Could not fetch tools from sandbox (sandbox may not exist)")
	} else if resp.Error != nil {
		log.Debug().
			Interface("code", resp.Error["code"]).
			Interface("msg", resp.Error["message"]).
			Str("user_id", userID).
			Msg("MCP error fetching tools (sandbox may not be running)")
	} else {
		// Parse result.tools from e2b-api
		if resultMap, ok := resp.Result.(map[string]interface{}); ok {
			if toolsRaw, ok := resultMap["tools"].([]interface{}); ok {
				for _, t := range toolsRaw {
					toolMap, ok := t.(map[string]interface{})
					if !ok {
						continue
					}

					name, _ := toolMap["name"].(string)
					if name == "" {
						continue
					}

					// Add e2b_ prefix to dynamic tools from mcp-proxy (those without prefix)
					// This ensures all sandbox tools have consistent naming for agents
					if !strings.HasPrefix(name, "e2b_") {
						name = "e2b_" + name
					}

					desc, _ := toolMap["description"].(string)
					schema, _ := toolMap["inputSchema"].(map[string]interface{})

					// Strip context fields from schema
					tools = append(tools, Tool{
						Name:        name,
						Description: desc,
						InputSchema: stripContextFields(schema),
					})
				}
			}
		}
	}

	log.Debug().
		Str("user_id", userID).
		Str("conversation_id", conversationID).
		Int("total_tools", len(tools)).
		Msg("E2B tools for user with conversation context")

	return tools, nil
}

// ExecuteTool executes an E2B tool.
// - Sandbox tools (e2b_sandbox_*): handled directly via e2b-api REST
// - Desktop/Browser tools: proxy to e2b-api MCP
// - Dynamic tools: remove e2b_ prefix before sending to e2b-api MCP
func (e *E2BMCP) ExecuteTool(ctx context.Context, userID, toolName string, args map[string]interface{}) (map[string]interface{}, error) {
	if !e.IsEnabled() {
		return nil, fmt.Errorf("E2B not enabled")
	}

	// Handle sandbox lifecycle tools directly (not via MCP proxy)
	if IsSandboxTool(toolName) {
		return e.executeSandboxTool(ctx, userID, toolName, args)
	}

	// Determine actual tool name to send to e2b-api
	actualToolName := toolName
	if !e2bStaticTools[toolName] && strings.HasPrefix(toolName, "e2b_") {
		// Dynamic tool - remove prefix before sending to e2b-api
		actualToolName = strings.TrimPrefix(toolName, "e2b_")
		log.Debug().
			Str("original", toolName).
			Str("actual", actualToolName).
			Msg("Dynamic tool: removing e2b_ prefix")
	}

	log.Info().
		Str("tool", toolName).
		Str("actual_tool", actualToolName).
		Str("user_id", userID).
		Msg("Executing E2B tool")

	resp, err := e.client.CallUserToolMCP(ctx, userID, actualToolName, args)
	if err != nil {
		return nil, fmt.Errorf("e2b-api call failed: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("tool error: code=%v msg=%v", resp.Error["code"], resp.Error["message"])
	}

	if resultMap, ok := resp.Result.(map[string]interface{}); ok {
		return resultMap, nil
	}

	return map[string]interface{}{"result": resp.Result}, nil
}

// IsE2BTool checks if a tool name is an E2B tool (starts with e2b_)
func IsE2BTool(toolName string) bool {
	return strings.HasPrefix(toolName, "e2b_")
}

// IsSandboxTool checks if a tool is a sandbox lifecycle management tool (e2b_sandbox_*)
func IsSandboxTool(toolName string) bool {
	return e2bSandboxTools[toolName]
}

// IsDesktopTool checks if a tool is a desktop tool (e2b_desktop_*)
func IsDesktopTool(toolName string) bool {
	return e2bDesktopTools[toolName]
}

// stripContextFields removes conversation_id and user_id from tool schema
// since mcp-tools auto-injects these fields when executing tools.
// This simplifies the client-facing API - agents don't need to pass these.
func stripContextFields(schema map[string]any) map[string]any {
	if schema == nil {
		return schema
	}

	// Deep copy to avoid modifying original
	result := make(map[string]any)
	for k, v := range schema {
		result[k] = v
	}

	// Remove from properties
	if props, ok := result["properties"].(map[string]any); ok {
		newProps := make(map[string]any)
		for k, v := range props {
			if k != "conversation_id" && k != "user_id" {
				newProps[k] = v
			}
		}
		result["properties"] = newProps
	}

	// Remove from required array ([]any type from JSON parsing)
	if required, ok := result["required"].([]any); ok {
		newRequired := make([]any, 0) // Initialize as empty array, not nil
		for _, r := range required {
			if rStr, ok := r.(string); ok {
				if rStr != "conversation_id" && rStr != "user_id" {
					newRequired = append(newRequired, r)
				}
			}
		}
		result["required"] = newRequired
	}

	// Handle []string type (from static tool definitions)
	if required, ok := result["required"].([]string); ok {
		newRequired := make([]string, 0) // Initialize as empty array, not nil
		for _, r := range required {
			if r != "conversation_id" && r != "user_id" {
				newRequired = append(newRequired, r)
			}
		}
		result["required"] = newRequired
	} else if req, exists := result["required"]; !exists || req == nil {
		// Ensure required is always present as empty array for MCP compliance
		result["required"] = make([]string, 0)
	}

	return result
}

// executeSandboxTool handles sandbox lifecycle operations internally.
func (e *E2BMCP) executeSandboxTool(ctx context.Context, userID, toolName string, args map[string]any) (map[string]any, error) {
	if !e.IsEnabled() {
		return nil, fmt.Errorf("E2B not enabled")
	}

	// Validate userID is required for all sandbox tools
	if userID == "" {
		return nil, fmt.Errorf("user_id is required for %s", toolName)
	}

	log.Info().
		Str("tool", toolName).
		Str("user_id", userID).
		Msg("Executing E2B sandbox tool")

	switch toolName {
	case "e2b_sandbox_start":
		return e.startSandbox(ctx, userID, args)
	case "e2b_sandbox_pause":
		return e.pauseSandbox(ctx, userID)
	case "e2b_sandbox_stop":
		return e.stopSandbox(ctx, userID)
	case "e2b_sandbox_extend":
		return e.extendSandbox(ctx, userID, args)
	default:
		return nil, fmt.Errorf("unknown sandbox tool: %s", toolName)
	}
}

func (e *E2BMCP) startSandbox(ctx context.Context, userID string, args map[string]any) (map[string]any, error) {
	// Validate required conversation_id
	convID, ok := args["conversation_id"].(string)
	if !ok || convID == "" {
		return nil, fmt.Errorf("conversation_id is required for e2b_start_sandbox")
	}

	timeout := 1800 // default 30 min
	if t, ok := args["timeout"].(float64); ok {
		timeout = int(t)
	}

	// Start sandbox
	record, err := e.client.StartSandboxForUser(ctx, userID, timeout)
	if err != nil {
		return nil, fmt.Errorf("start sandbox: %w", err)
	}

	result := map[string]any{
		"sandbox_id": record.PublicID,
		"status":     record.Status,
	}

	// Add all available fields from sandbox record
	if record.ViewURL != nil {
		result["view_url"] = *record.ViewURL
	}
	if record.ControlURL != nil {
		result["control_url"] = *record.ControlURL
	}
	if record.SandboxURL != nil {
		result["sandbox_url"] = *record.SandboxURL
	}
	if record.StartedAt != nil {
		result["started_at"] = *record.StartedAt
	}
	if record.TimeoutAt != nil {
		result["timeout_at"] = *record.TimeoutAt
	}
	if record.E2BSandboxID != nil {
		result["e2b_sandbox_id"] = *record.E2BSandboxID
	}
	if record.ErrorMessage != nil {
		result["error_message"] = *record.ErrorMessage
	}

	// Create workspace for conversation
	workspace, err := e.client.GetOrCreateWorkspace(ctx, userID, convID)
	if err == nil {
		result["workspace_path"] = workspace.Workspace.WorkspacePath
	} else {
		log.Warn().Err(err).Str("conversation_id", convID).Msg("Failed to create workspace")
	}

	log.Info().
		Str("sandbox_id", record.PublicID).
		Str("status", record.Status).
		Str("conversation_id", convID).
		Msg("Sandbox started")

	return result, nil
}

func (e *E2BMCP) pauseSandbox(ctx context.Context, userID string) (map[string]any, error) {
	record, err := e.client.PauseSandbox(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("pause sandbox: %w", err)
	}

	result := map[string]any{
		"sandbox_id": record.PublicID,
		"status":     record.Status,
	}

	if record.PausedAt != nil {
		result["paused_at"] = *record.PausedAt
	}
	if record.PauseExpiresAt != nil {
		result["pause_expires_at"] = *record.PauseExpiresAt
	}

	log.Info().
		Str("sandbox_id", record.PublicID).
		Str("status", record.Status).
		Msg("Sandbox paused")

	return result, nil
}

func (e *E2BMCP) stopSandbox(ctx context.Context, userID string) (map[string]any, error) {
	record, err := e.client.StopSandboxForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("stop sandbox: %w", err)
	}

	result := map[string]any{
		"sandbox_id": record.PublicID,
		"status":     record.Status,
		"message":    "Sandbox stopped successfully",
	}

	log.Info().
		Str("sandbox_id", record.PublicID).
		Str("status", record.Status).
		Msg("Sandbox stopped")

	return result, nil
}

func (e *E2BMCP) extendSandbox(ctx context.Context, userID string, args map[string]any) (map[string]any, error) {
	additionalSeconds, ok := args["additional_seconds"].(float64)
	if !ok {
		return nil, fmt.Errorf("additional_seconds is required")
	}

	record, err := e.client.ExtendTimeout(ctx, userID, int(additionalSeconds))
	if err != nil {
		return nil, fmt.Errorf("extend sandbox: %w", err)
	}

	result := map[string]any{
		"sandbox_id": record.PublicID,
		"status":     record.Status,
	}

	if record.TimeoutAt != nil {
		result["timeout_at"] = *record.TimeoutAt
	}

	log.Info().
		Str("sandbox_id", record.PublicID).
		Int("additional_seconds", int(additionalSeconds)).
		Msg("Sandbox timeout extended")

	return result, nil
}
