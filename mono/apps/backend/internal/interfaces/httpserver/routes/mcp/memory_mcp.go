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

	"jan-server/mono/apps/backend/internal/infrastructure/llmapi"
	"jan-server/mono/apps/backend/internal/infrastructure/metrics"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

// MemoryRetrieveArgs defines the arguments for the memory_retrieve tool
type MemoryRetrieveArgs struct {
	Query            string   `json:"query"`
	UserID           *string  `json:"user_id,omitempty"`
	ProjectID        *string  `json:"project_id,omitempty"`
	ConversationID   *string  `json:"conversation_id,omitempty"`
	Scopes           []string `json:"scopes,omitempty"`
	MaxUserItems     *int     `json:"max_user_items,omitempty"`
	MaxProjectItems  *int     `json:"max_project_items,omitempty"`
	MaxEpisodicItems *int     `json:"max_episodic_items,omitempty"`
	MinSimilarity    *float32 `json:"min_similarity,omitempty"`
	// Context passthrough
	ToolCallID string `json:"tool_call_id,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
}

// memoryLoadRequest matches the memory-tools API structure
type memoryLoadRequest struct {
	UserID         string            `json:"user_id"`
	ProjectID      string            `json:"project_id,omitempty"`
	ConversationID string            `json:"conversation_id,omitempty"`
	Query          string            `json:"query"`
	Options        memoryLoadOptions `json:"options"`
}

type memoryLoadOptions struct {
	AugmentWithMemory bool    `json:"augment_with_memory"`
	MaxUserItems      int     `json:"max_user_items"`
	MaxProjectItems   int     `json:"max_project_items"`
	MaxEpisodicItems  int     `json:"max_episodic_items"`
	MinSimilarity     float32 `json:"min_similarity"`
}

// memoryLoadResponse matches the memory-tools API response
type memoryLoadResponse struct {
	CoreMemory     []memoryItem `json:"core_memory"`
	EpisodicMemory []memoryItem `json:"episodic_memory"`
	SemanticMemory []memoryItem `json:"semantic_memory"`
}

type memoryItem struct {
	ID             string                 `json:"id"`
	Scope          string                 `json:"scope,omitempty"`
	Key            string                 `json:"key,omitempty"`
	Text           string                 `json:"text"`
	Importance     string                 `json:"importance,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	RelevanceScore float64                `json:"relevance_score,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// memoryToolResult is the formatted result returned to the LLM
type memoryToolResult struct {
	Query            string       `json:"query"`
	TotalItems       int          `json:"total_items"`
	UserMemories     []memoryItem `json:"user_memories"`
	ProjectMemories  []memoryItem `json:"project_memories"`
	EpisodicMemories []memoryItem `json:"episodic_memories"`
	QueryTimeMS      int64        `json:"query_time_ms"`
	EstimatedTokens  int          `json:"estimated_tokens"`
}

// MemoryMCP handles MCP tool registration for memory retrieval.
type MemoryMCP struct {
	memoryToolsURL string
	httpClient     *http.Client
	llmClient      *llmapi.Client // LLM-API client for tool tracking
	enabled        bool
}

// NewMemoryMCP creates a new memory MCP handler.
func NewMemoryMCP(memoryToolsURL string, enabled bool) *MemoryMCP {
	return &MemoryMCP{
		memoryToolsURL: memoryToolsURL,
		enabled:        enabled,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetLLMClient sets the LLM-API client for tool call tracking
func (m *MemoryMCP) SetLLMClient(client *llmapi.Client) {
	m.llmClient = client
}

// RegisterTools registers memory tools with the MCP server
func (m *MemoryMCP) RegisterTools(server *mcp.Server) {
	if !m.enabled {
		log.Warn().Msg("memory_retrieve MCP tool disabled via config")
		return
	}
	if m.memoryToolsURL == "" {
		log.Warn().Msg("Memory tools URL not configured, skipping memory_retrieve tool registration")
		return
	}

	// Register memory_retrieve tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "memory_retrieve",
		Description: "READ-ONLY: Search and retrieve relevant user preferences, project facts, or conversation history from memory storage. This tool ONLY reads existing memories - it does NOT create, update, or sync memories. Use this to recall what you already know about the user or project context.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input MemoryRetrieveArgs) (*mcp.CallToolResult, memoryToolResult, error) {
		startTime := time.Now()
		callCtx := extractAllContext(req)

		// Check for tracking context from headers
		tracking, trackingEnabled := GetToolTracking(ctx)

		log.Info().
			Str("tool", "memory_retrieve").
			Str("tool_call_id", callCtx["tool_call_id"]).
			Str("request_id", callCtx["request_id"]).
			Str("conversation_id", callCtx["conversation_id"]).
			Str("user_id", callCtx["user_id"]).
			Bool("tracking_enabled", trackingEnabled).
			Msg("MCP tool call received")

		query := input.Query
		if strings.TrimSpace(query) == "" {
			log.Error().Str("tool", "memory_retrieve").Msg("missing required parameter 'query'")
			return nil, memoryToolResult{}, fmt.Errorf("query is required")
		}

		var userID string
		if ctxUserID, ok := ctx.Value("user_id").(string); ok && ctxUserID != "" {
			userID = ctxUserID
			log.Info().Str("user_id", userID).Str("query", query).Msg("[Memory MCP] Using user_id from JWT authentication")
		} else if input.UserID != nil && *input.UserID != "" {
			userID = *input.UserID
			log.Info().Str("user_id", userID).Str("query", query).Msg("[Memory MCP] Using user_id from parameter (no JWT)")
		} else {
			log.Error().Str("query", query).Msg("[Memory MCP] user_id is required but not provided")
			return nil, memoryToolResult{}, fmt.Errorf("user_id is required: provide it as a parameter or authenticate with JWT")
		}

		memReq := memoryLoadRequest{
			UserID: userID,
			Query:  query,
			Options: memoryLoadOptions{
				AugmentWithMemory: true,
				MaxUserItems:      3,
				MaxProjectItems:   5,
				MaxEpisodicItems:  3,
				MinSimilarity:     0.75,
			},
		}

		if input.ProjectID != nil && *input.ProjectID != "" {
			memReq.ProjectID = *input.ProjectID
			log.Info().Str("user_id", userID).Str("project_id", memReq.ProjectID).Msg("[Memory MCP] Including project_id filter")
		}
		if input.ConversationID != nil && *input.ConversationID != "" {
			memReq.ConversationID = *input.ConversationID
			log.Info().Str("user_id", userID).Str("conversation_id", memReq.ConversationID).Msg("[Memory MCP] Including conversation_id filter")
		}

		if input.MaxUserItems != nil && *input.MaxUserItems > 0 {
			maxUser := *input.MaxUserItems
			if maxUser > 10 {
				maxUser = 10
			}
			memReq.Options.MaxUserItems = maxUser
		}
		if input.MaxProjectItems != nil && *input.MaxProjectItems > 0 {
			maxProject := *input.MaxProjectItems
			if maxProject > 10 {
				maxProject = 10
			}
			memReq.Options.MaxProjectItems = maxProject
		}
		if input.MaxEpisodicItems != nil && *input.MaxEpisodicItems > 0 {
			maxEpisodic := *input.MaxEpisodicItems
			if maxEpisodic > 10 {
				maxEpisodic = 10
			}
			memReq.Options.MaxEpisodicItems = maxEpisodic
		}

		if input.MinSimilarity != nil && *input.MinSimilarity >= 0 && *input.MinSimilarity <= 1 {
			memReq.Options.MinSimilarity = *input.MinSimilarity
		}

		log.Info().
			Str("user_id", userID).
			Str("query", query).
			Str("project_id", memReq.ProjectID).
			Str("conversation_id", memReq.ConversationID).
			Int("max_user_items", memReq.Options.MaxUserItems).
			Int("max_project_items", memReq.Options.MaxProjectItems).
			Int("max_episodic_items", memReq.Options.MaxEpisodicItems).
			Float32("min_similarity", memReq.Options.MinSimilarity).
			Str("memory_url", m.memoryToolsURL).
			Msg("[Memory MCP] Calling memory-tools service")

		response, err := m.callMemoryLoad(ctx, memReq)
		if err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID).
				Str("query", query).
				Str("memory_url", m.memoryToolsURL).
				Msg("[Memory MCP] Failed to retrieve memories")
			// Record error metrics
			metrics.RecordToolCall("memory_retrieve", "memory-tools", "error", time.Since(startTime).Seconds())
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(`{"query":"%s","total_items":0,"user_memories":[],"project_memories":[],"episodic_memories":[],"error":"memory service unavailable"}`, query)}},
			}, memoryToolResult{}, nil
		}

		elapsed := time.Since(startTime).Milliseconds()

		result := memoryToolResult{
			Query:            query,
			UserMemories:     response.CoreMemory,
			ProjectMemories:  response.SemanticMemory,
			EpisodicMemories: response.EpisodicMemory,
			TotalItems:       len(response.CoreMemory) + len(response.SemanticMemory) + len(response.EpisodicMemory),
			QueryTimeMS:      elapsed,
			EstimatedTokens:  m.estimateTokens(response),
		}

		log.Info().
			Str("user_id", userID).
			Str("query", query).
			Int("user_memories", len(response.CoreMemory)).
			Int("project_memories", len(response.SemanticMemory)).
			Int("episodic_memories", len(response.EpisodicMemory)).
			Int("total_items", result.TotalItems).
			Int64("query_time_ms", elapsed).
			Int("estimated_tokens", result.EstimatedTokens).
			Msg("[Memory MCP] Successfully retrieved memories")

		// Update tool call result in LLM-API (async)
		if trackingEnabled && m.llmClient != nil {
			// Capture input for async goroutine
			inputCopy := input
			go func() {
				saveCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				// Marshal result for storage
				resultJSON, _ := json.Marshal(result)

				// Serialize arguments
				argsBytes, _ := json.Marshal(inputCopy)
				argsStr := string(argsBytes)

				saveResult := m.llmClient.UpdateToolCallResult(
					saveCtx,
					tracking.AuthToken,
					tracking.ConversationID,
					tracking.ToolCallID,
					"memory_retrieve",
					argsStr,
					"Jan MCP Server",
					string(resultJSON),
					nil, // no error
				)
				if saveResult.Error != nil {
					log.Warn().
						Err(saveResult.Error).
						Str("conversation_id", tracking.ConversationID).
						Str("tool_call_id", tracking.ToolCallID).
						Msg("[Memory MCP] Failed to update tool call result")
				} else {
					log.Debug().
						Str("conversation_id", tracking.ConversationID).
						Str("tool_call_id", tracking.ToolCallID).
						Msg("[Memory MCP] Updated tool call result")
				}
			}()
		}

		// Record metrics
		metrics.RecordToolCall("memory_retrieve", "memory-tools", "success", time.Since(startTime).Seconds())
		metrics.RecordToolTokens("memory_retrieve", "memory-tools", float64(result.EstimatedTokens))

		return nil, result, nil
	})

	log.Info().Str("url", m.memoryToolsURL).Msg("Registered memory_retrieve MCP tool")
}

// callMemoryLoad calls the memory-tools /v1/memory/load endpoint
func (m *MemoryMCP) callMemoryLoad(ctx context.Context, req memoryLoadRequest) (*memoryLoadResponse, error) {
	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		log.Error().Err(err).Str("user_id", req.UserID).Msg("[Memory MCP] Failed to marshal request")
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log the raw request being sent
	log.Debug().
		Str("user_id", req.UserID).
		Str("url", m.memoryToolsURL+"/v1/memory/load").
		Str("request_body", string(reqBody)).
		Msg("[Memory MCP] Sending HTTP request to memory service")

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, m.memoryToolsURL+"/v1/memory/load", bytes.NewReader(reqBody))
	if err != nil {
		log.Error().Err(err).Str("user_id", req.UserID).Str("url", m.memoryToolsURL).Msg("[Memory MCP] Failed to create HTTP request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	httpResp, err := m.httpClient.Do(httpReq)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", req.UserID).
			Str("url", m.memoryToolsURL+"/v1/memory/load").
			Msg("[Memory MCP] HTTP request failed - connection error")
		return nil, fmt.Errorf("failed to call memory service: %w", err)
	}
	defer httpResp.Body.Close()

	// Check status
	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		log.Error().
			Str("user_id", req.UserID).
			Int("status_code", httpResp.StatusCode).
			Str("response_body", string(body)).
			Str("url", m.memoryToolsURL+"/v1/memory/load").
			Msg("[Memory MCP] Memory service returned non-OK status")
		return nil, fmt.Errorf("memory service returned status %d: %s", httpResp.StatusCode, string(body))
	}

	// Read response body for logging
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		log.Error().Err(err).Str("user_id", req.UserID).Msg("[Memory MCP] Failed to read response body")
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	log.Debug().
		Str("user_id", req.UserID).
		Str("response_body", string(respBody)).
		Msg("[Memory MCP] Received response from memory service")

	// Parse response
	var response memoryLoadResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		log.Error().
			Err(err).
			Str("user_id", req.UserID).
			Str("response_body", string(respBody)).
			Msg("[Memory MCP] Failed to decode response JSON")
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Info().
		Str("user_id", req.UserID).
		Int("core_memory_count", len(response.CoreMemory)).
		Int("semantic_memory_count", len(response.SemanticMemory)).
		Int("episodic_memory_count", len(response.EpisodicMemory)).
		Msg("[Memory MCP] Successfully parsed memory service response")

	return &response, nil
}

// estimateTokens provides a rough estimate of token count for the response
func (m *MemoryMCP) estimateTokens(response *memoryLoadResponse) int {
	// Rough estimate: 1 token ≈ 4 characters
	totalChars := 0

	for _, item := range response.CoreMemory {
		totalChars += len(item.Text)
	}
	for _, item := range response.SemanticMemory {
		totalChars += len(item.Text)
	}
	for _, item := range response.EpisodicMemory {
		totalChars += len(item.Text)
	}

	return totalChars / 4
}
