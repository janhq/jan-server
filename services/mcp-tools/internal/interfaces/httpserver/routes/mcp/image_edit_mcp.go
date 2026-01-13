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

// ImageEditInput defines an image input for the edit_image tool.
type ImageEditInput struct {
	ID      *string `json:"id,omitempty"`
	URL     *string `json:"url,omitempty"`
	B64JSON *string `json:"b64_json,omitempty"`
}

// UnmarshalJSON supports string inputs (treated as URL, ID, or base64) or object inputs.
func (i *ImageEditInput) UnmarshalJSON(data []byte) error {
	if i == nil {
		return nil
	}
	var rawString string
	if err := json.Unmarshal(data, &rawString); err == nil {
		trimmed := strings.TrimSpace(rawString)
		if trimmed == "" {
			return nil
		}
		switch {
		case strings.HasPrefix(trimmed, "jan_"):
			i.ID = &trimmed
		case strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") || strings.HasPrefix(trimmed, "data:"):
			i.URL = &trimmed
		default:
			i.B64JSON = &trimmed
		}
		return nil
	}

	type Alias ImageEditInput
	var decoded Alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*i = ImageEditInput(decoded)
	return nil
}

// ImageEditArgs defines the arguments for the edit_image tool.
type ImageEditArgs struct {
	Prompt         string          `json:"prompt"`
	Image          *ImageEditInput `json:"image"`
	Mask           *ImageEditInput `json:"mask,omitempty"`
	Model          *string         `json:"model,omitempty"`
	Size           *string         `json:"size,omitempty"`
	N              *int            `json:"n,omitempty"`
	Strength       *float64        `json:"strength,omitempty"`
	Steps          *int            `json:"steps,omitempty"`
	Seed           *int            `json:"seed,omitempty"`
	CfgScale       *float64        `json:"cfg_scale,omitempty"`
	Sampler        *string         `json:"sampler,omitempty"`
	Scheduler      *string         `json:"scheduler,omitempty"`
	NegativePrompt *string         `json:"negative_prompt,omitempty"`
	User           *string         `json:"user,omitempty"`
	ConversationID *string         `json:"conversation_id,omitempty"`
	Store          *bool           `json:"store,omitempty"`
	// Context passthrough
	ToolCallID string `json:"tool_call_id,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
}

type ImageEditMCP struct {
	llmAPIBaseURL string
	httpClient    *http.Client
	enabled       bool
}

// NewImageEditMCP creates a new image edit MCP handler.
func NewImageEditMCP(llmAPIBaseURL string, enabled bool) *ImageEditMCP {
	return &ImageEditMCP{
		llmAPIBaseURL: strings.TrimRight(llmAPIBaseURL, "/"),
		enabled:       enabled,
		httpClient: &http.Client{
			Timeout: 600 * time.Second,
		},
	}
}

// RegisterTools registers the edit_image tool with the MCP server.
func (i *ImageEditMCP) RegisterTools(server *mcp.Server) {
	if i == nil {
		return
	}
	if !i.enabled {
		log.Warn().Msg("edit_image MCP tool disabled via config")
		return
	}
	if i.llmAPIBaseURL == "" {
		log.Warn().Msg("LLM_API_BASE_URL not configured; skipping edit_image tool registration")
		return
	}

	imageInputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Jan media ID (jan_*)",
			},
			"url": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Remote URL to the image",
			},
			"b64_json": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Base64-encoded image data (no data URL prefix)",
			},
		},
	}

	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "Edit instruction describing the desired changes",
			},
			"image": map[string]any{
				"type":        []string{"object", "string", "null"},
				"description": "Input image (id, url, b64_json, or URL string)",
				"properties":  imageInputSchema["properties"],
			},
			"mask": map[string]any{
				"type":        []string{"object", "string", "null"},
				"description": "Optional mask for inpainting (id, url, b64_json, or URL string)",
				"properties":  imageInputSchema["properties"],
			},
			"size": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Output size (original or WIDTHxHEIGHT)",
				"default":     "original",
			},
			"model": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Image edit model",
			},
			"n": map[string]any{
				"type":        []string{"integer", "null"},
				"description": "Number of images to generate (often only 1 supported)",
				"default":     1,
			},
			"strength": map[string]any{
				"type":        []string{"number", "null"},
				"description": "Edit strength (0.0-1.0)",
			},
			"steps": map[string]any{
				"type":        []string{"integer", "null"},
				"description": "Sampling steps",
			},
			"seed": map[string]any{
				"type":        []string{"integer", "null"},
				"description": "Random seed (-1 for random)",
			},
			"cfg_scale": map[string]any{
				"type":        []string{"number", "null"},
				"description": "Classifier-free guidance scale",
			},
			"sampler": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Sampling algorithm",
			},
			"scheduler": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Scheduler",
			},
			"negative_prompt": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Negative prompt (what to avoid)",
			},
			"user": map[string]any{
				"type":        []string{"string", "null"},
				"description": "End-user identifier for abuse monitoring",
			},
			"conversation_id": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Conversation ID to store image edit items",
			},
			"store": map[string]any{
				"type":        []string{"boolean", "null"},
				"description": "Whether to store the result in the conversation",
			},
		},
		"required": []string{"prompt", "image"},
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "edit_image",
		Description: "Edit images with a prompt via LLM API /v1/images/edits.",
		InputSchema: inputSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ImageEditArgs) (*mcp.CallToolResult, map[string]any, error) {
		startTime := time.Now()
		callCtx := extractAllContext(req)
		tracking, _ := GetToolTracking(ctx)

		log.Info().
			Str("tool", "edit_image").
			Str("tool_call_id", callCtx["tool_call_id"]).
			Str("request_id", callCtx["request_id"]).
			Str("conversation_id", callCtx["conversation_id"]).
			Str("user_id", callCtx["user_id"]).
			Msg("MCP tool call received")

		if strings.TrimSpace(input.Prompt) == "" {
			log.Error().Str("tool", "edit_image").Msg("missing required parameter 'prompt'")
			return nil, nil, fmt.Errorf("prompt is required")
		}
		if !hasImageInput(input.Image) {
			log.Error().Str("tool", "edit_image").Msg("missing required parameter 'image'")
			return nil, nil, fmt.Errorf("image is required")
		}

		payload := map[string]any{
			"prompt":          input.Prompt,
			"response_format": "url",
			"image":           buildImageInputPayload(input.Image),
		}
		if input.Mask != nil && hasImageInput(input.Mask) {
			payload["mask"] = buildImageInputPayload(input.Mask)
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
		if input.Strength != nil {
			payload["strength"] = *input.Strength
		}
		if input.Steps != nil {
			payload["steps"] = *input.Steps
		}
		if input.Seed != nil {
			payload["seed"] = *input.Seed
		}
		if input.CfgScale != nil {
			payload["cfg_scale"] = *input.CfgScale
		}
		if input.Sampler != nil {
			payload["sampler"] = *input.Sampler
		}
		if input.Scheduler != nil {
			payload["scheduler"] = *input.Scheduler
		}
		if input.NegativePrompt != nil {
			payload["negative_prompt"] = *input.NegativePrompt
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

		endpoint := fmt.Sprintf("%s/v1/images/edits", i.llmAPIBaseURL)
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
			metrics.RecordToolCall("edit_image", "llm-api", "error", time.Since(startTime).Seconds())
			return nil, nil, fmt.Errorf("failed to call LLM API: %w", err)
		}
		defer resp.Body.Close()

		respBytes, _ := io.ReadAll(resp.Body)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			metrics.RecordToolCall("edit_image", "llm-api", "error", time.Since(startTime).Seconds())
			return nil, nil, fmt.Errorf("llm-api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
		}

		var result map[string]any
		if err := json.Unmarshal(respBytes, &result); err != nil {
			result = map[string]any{
				"raw": string(respBytes),
			}
		}

		metrics.RecordToolCall("edit_image", "llm-api", "success", time.Since(startTime).Seconds())
		return nil, result, nil
	})

	log.Info().Str("base_url", i.llmAPIBaseURL).Msg("Registered edit_image MCP tool")
}

func hasImageInput(input *ImageEditInput) bool {
	if input == nil {
		return false
	}
	if input.ID != nil && strings.TrimSpace(*input.ID) != "" {
		return true
	}
	if input.URL != nil && strings.TrimSpace(*input.URL) != "" {
		return true
	}
	if input.B64JSON != nil && strings.TrimSpace(*input.B64JSON) != "" {
		return true
	}
	return false
}

func buildImageInputPayload(input *ImageEditInput) map[string]any {
	payload := map[string]any{}
	if input == nil {
		return payload
	}
	if input.ID != nil && strings.TrimSpace(*input.ID) != "" {
		payload["id"] = *input.ID
	}
	if input.URL != nil && strings.TrimSpace(*input.URL) != "" {
		payload["url"] = *input.URL
	}
	if input.B64JSON != nil && strings.TrimSpace(*input.B64JSON) != "" {
		payload["b64_json"] = *input.B64JSON
	}
	return payload
}
