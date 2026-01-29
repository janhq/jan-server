package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"jan-server/mono/apps/backend/internal/infrastructure/llmapi"
	"jan-server/mono/apps/backend/internal/infrastructure/metrics"
	"jan-server/mono/apps/backend/internal/infrastructure/sandboxfusion"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

type SandboxFusionArgs struct {
	Code      string  `json:"code"`
	Language  *string `json:"language,omitempty"`
	SessionID *string `json:"session_id,omitempty"`
	Approved  *bool   `json:"approved,omitempty"`
	// Context passthrough
	ToolCallID     string `json:"tool_call_id,omitempty"`
	RequestID      string `json:"request_id,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
}

type SandboxFusionMCP struct {
	client          *sandboxfusion.Client
	llmClient       *llmapi.Client // LLM-API client for tool tracking
	requireApproval bool
	enabled         bool
}

func NewSandboxFusionMCP(client *sandboxfusion.Client, requireApproval bool, enabled bool) *SandboxFusionMCP {
	if client == nil {
		return nil
	}
	return &SandboxFusionMCP{
		client:          client,
		requireApproval: requireApproval,
		enabled:         enabled,
	}
}

// SetLLMClient sets the LLM-API client for tool call tracking
func (s *SandboxFusionMCP) SetLLMClient(client *llmapi.Client) {
	s.llmClient = client
}

func (s *SandboxFusionMCP) RegisterTools(server *mcp.Server) {
	if s == nil {
		return
	}
	if !s.enabled {
		log.Warn().Msg("python_exec MCP tool disabled via config")
		return
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "python_exec",
		Description: "Execute trusted code inside SandboxFusion and return stdout/stderr/artifacts.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SandboxFusionArgs) (*mcp.CallToolResult, map[string]any, error) {
		startTime := time.Now()
		callCtx := extractAllContext(req)

		// Check for tracking context from headers
		tracking, trackingEnabled := GetToolTracking(ctx)

		log.Info().
			Str("tool", "python_exec").
			Str("tool_call_id", callCtx["tool_call_id"]).
			Str("request_id", callCtx["request_id"]).
			Str("conversation_id", callCtx["conversation_id"]).
			Str("user_id", callCtx["user_id"]).
			Bool("tracking_enabled", trackingEnabled).
			Msg("MCP tool call received")

		if s.requireApproval {
			if input.Approved == nil || !*input.Approved {
				log.Warn().Str("tool", "python_exec").Msg("execution requires approval but not granted")
				return nil, nil, fmt.Errorf("sandboxfusion execution requires approval; set the `approved` argument to true")
			}
		}

		runReq := sandboxfusion.RunCodeRequest{
			Code: input.Code,
		}

		if input.Language != nil && *input.Language != "" {
			runReq.Language = *input.Language
		}
		if input.SessionID != nil && *input.SessionID != "" {
			runReq.SessionID = *input.SessionID
		}

		var payload map[string]any
		var toolErr error
		var estimatedTokens float64

		if s.client != nil {
			resp, err := s.client.RunCode(ctx, runReq)
			if err == nil {
				payload = map[string]any{
					"stdout":      resp.Stdout,
					"stderr":      resp.Stderr,
					"duration_ms": resp.Duration,
					"session_id":  resp.SessionID,
					"artifacts":   resp.Artifacts,
					"error":       resp.Error,
				}
				estimatedTokens = estimateTokensFromStrings(resp.Stdout, resp.Stderr)
			} else {
				log.Warn().Err(err).Str("tool", "python_exec").Str("language", runReq.Language).Msg("sandboxfusion execution failed; using fallback stub")
				toolErr = err
				payload = map[string]any{
					"stdout":      "hello from sandbox (stub)",
					"stderr":      "",
					"duration_ms": 0,
					"session_id":  runReq.SessionID,
					"artifacts":   []string{},
					"error":       "",
				}
				estimatedTokens = estimateTokensFromStrings(
					"hello from sandbox (stub)",
					"",
				)
			}
		} else {
			payload = map[string]any{
				"stdout":      "hello from sandbox (stub)",
				"stderr":      "",
				"duration_ms": 0,
				"session_id":  runReq.SessionID,
				"artifacts":   []string{},
				"error":       "",
			}
			estimatedTokens = estimateTokensFromStrings("hello from sandbox (stub)", "")
		}

		// If tracking is enabled, save result to LLM-API
		if trackingEnabled && s.llmClient != nil {
			// Capture input for async goroutine
			inputCopy := input
			go func() {
				saveCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				outputBytes, _ := json.Marshal(payload)
				outputStr := string(outputBytes)

				// Serialize arguments
				argsBytes, _ := json.Marshal(inputCopy)
				argsStr := string(argsBytes)

				var errStr *string
				if toolErr != nil {
					e := toolErr.Error()
					errStr = &e
				}

				result := s.llmClient.UpdateToolCallResult(
					saveCtx,
					tracking.AuthToken,
					tracking.ConversationID,
					tracking.ToolCallID,
					"python_exec",
					argsStr,
					"Jan MCP Server",
					outputStr,
					errStr,
				)

				if !result.Success && result.Error != nil {
					log.Error().
						Err(result.Error).
						Str("call_id", tracking.ToolCallID).
						Str("conv_id", tracking.ConversationID).
						Int64("duration_ms", time.Since(startTime).Milliseconds()).
						Msg("Failed to update tool result in LLM-API")
				}
			}()
		}

		// Record metrics
		status := "success"
		if toolErr != nil {
			status = "error"
		}
		metrics.RecordToolCall("python_exec", "sandboxfusion", status, time.Since(startTime).Seconds())
		if estimatedTokens > 0 {
			metrics.RecordToolTokens("python_exec", "sandboxfusion", estimatedTokens)
		}

		return nil, payload, nil
	})
}

func estimateTokensFromStrings(parts ...string) float64 {
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	return float64(total) / 4
}
