package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"jan-server/services/mcp-tools/internal/infrastructure/metrics"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

// ImageGenerateArgs defines the arguments for the generate_image tool
type ImageGenerateArgs struct {
	Prompt            string   `json:"prompt"`
	Model             *string  `json:"model,omitempty"`
	Size              *string  `json:"size,omitempty"`
	N                 *int     `json:"n,omitempty"`
	NumInferenceSteps *int     `json:"num_inference_steps,omitempty"`
	CfgScale          *float64 `json:"cfg_scale,omitempty"`
	Quality           *string  `json:"quality,omitempty"`
	Style             *string  `json:"style,omitempty"`
	User              *string  `json:"user,omitempty"`
	ConversationID    *string  `json:"conversation_id,omitempty"`
	Store             *bool    `json:"store,omitempty"`
	// Context passthrough
	ToolCallID string `json:"tool_call_id,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
}

type ImageGenerateMCP struct {
	llmAPIBaseURL string
	httpClient    *http.Client
	enabled       bool
}

// NewImageGenerateMCP creates a new image generation MCP handler.
func NewImageGenerateMCP(llmAPIBaseURL string, enabled bool) *ImageGenerateMCP {
	return &ImageGenerateMCP{
		llmAPIBaseURL: strings.TrimRight(llmAPIBaseURL, "/"),
		enabled:       enabled,
		httpClient: &http.Client{
			Timeout: 600 * time.Second,
		},
	}
}

// RegisterTools registers the generate_image tool with the MCP server.
func (i *ImageGenerateMCP) RegisterTools(server *mcp.Server) {
	if i == nil {
		return
	}
	if !i.enabled {
		log.Warn().Msg("generate_image MCP tool disabled via config")
		return
	}
	if i.llmAPIBaseURL == "" {
		log.Warn().Msg("LLM_API_BASE_URL not configured; skipping generate_image tool registration")
		return
	}

	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "Text prompt to generate an image from",
			},
			"size": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Image size (e.g., 512x512, 1024x1024, 1792x1024, 1024x1792)",
				"default":     "1024x1024",
			},
			"model": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Image generation model (e.g., z-image, flux-dev)",
			},
			"n": map[string]any{
				"type":        []string{"integer", "null"},
				"description": "Number of images to generate",
				"default":     1,
			},
			"num_inference_steps": map[string]any{
				"type":        []string{"integer", "null"},
				"description": "Inference steps to run for generation quality",
				"default":     30,
			},
			"cfg_scale": map[string]any{
				"type":        []string{"number", "null"},
				"description": "Classifier-free guidance scale",
				"default":     4.0,
			},
			"quality": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Image quality (standard, hd)",
			},
			"style": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Image style (vivid, natural)",
			},
			"user": map[string]any{
				"type":        []string{"string", "null"},
				"description": "End-user identifier for abuse monitoring",
			},
			"conversation_id": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Conversation ID to store image generation items",
			},
			"store": map[string]any{
				"type":        []string{"boolean", "null"},
				"description": "Whether to store the result in the conversation",
			},
		},
		"required": []string{"prompt"},
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "generate_image",
		Description: "Generate images from a text prompt via LLM API /v1/images/generations.",
		InputSchema: inputSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ImageGenerateArgs) (*mcp.CallToolResult, map[string]any, error) {
		startTime := time.Now()
		callCtx := extractAllContext(req)
		tracking, _ := GetToolTracking(ctx)

		log.Info().
			Str("tool", "generate_image").
			Str("tool_call_id", callCtx["tool_call_id"]).
			Str("request_id", callCtx["request_id"]).
			Str("conversation_id", callCtx["conversation_id"]).
			Str("user_id", callCtx["user_id"]).
			Msg("MCP tool call received")

		if strings.TrimSpace(input.Prompt) == "" {
			log.Error().Str("tool", "generate_image").Msg("missing required parameter 'prompt'")
			return nil, nil, fmt.Errorf("prompt is required")
		}

		payload := map[string]any{
			"prompt":          input.Prompt,
			"response_format": "url",
		}
		if input.Model != nil {
			payload["model"] = *input.Model
		}
		if input.Size != nil {
			payload["size"] = *input.Size
		}
		if input.N != nil {
			payload["n"] = *input.N
		}
		if input.NumInferenceSteps != nil {
			payload["num_inference_steps"] = *input.NumInferenceSteps
		}
		if input.CfgScale != nil {
			payload["cfg_scale"] = *input.CfgScale
		}
		if input.Quality != nil {
			payload["quality"] = *input.Quality
		}
		if input.Style != nil {
			payload["style"] = *input.Style
		}
		if input.User != nil {
			payload["user"] = *input.User
		}
		if input.ConversationID != nil && strings.TrimSpace(*input.ConversationID) != "" {
			payload["conversation_id"] = *input.ConversationID
		} else if strings.TrimSpace(tracking.ConversationID) != "" {
			payload["conversation_id"] = tracking.ConversationID
		}
		if input.Store != nil {
			payload["store"] = *input.Store
		}

		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		endpoint := fmt.Sprintf("%s/v1/images/generations", i.llmAPIBaseURL)
		reqOut, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create request: %w", err)
		}
		reqOut.Header.Set("Content-Type", "application/json")
		if strings.TrimSpace(tracking.AuthToken) != "" {
			reqOut.Header.Set("Authorization", tracking.AuthToken)
		}

		resp, err := i.httpClient.Do(reqOut)
		if err != nil {
			metrics.RecordToolCall("generate_image", "llm-api", "error", time.Since(startTime).Seconds())
			return nil, nil, fmt.Errorf("failed to call LLM API: %w", err)
		}
		defer resp.Body.Close()

		respBytes, _ := io.ReadAll(resp.Body)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			metrics.RecordToolCall("generate_image", "llm-api", "error", time.Since(startTime).Seconds())
			return nil, nil, fmt.Errorf("llm-api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
		}

		var result map[string]any
		if err := json.Unmarshal(respBytes, &result); err != nil {
			// Return raw response if parsing fails
			result = map[string]any{
				"raw": string(respBytes),
			}
		}

		metrics.RecordToolCall("generate_image", "llm-api", "success", time.Since(startTime).Seconds())
		return nil, result, nil
	})

	log.Info().Str("base_url", i.llmAPIBaseURL).Msg("Registered generate_image MCP tool")
}
