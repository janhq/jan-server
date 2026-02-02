package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"

	"jan-server/services/mcp-tools/internal/infrastructure/llmapi"
)

// AgentProxyMCP handles the unified run_agent tool that routes to Response API.
type AgentProxyMCP struct {
	responseAPIURL string
	llmAPIURL      string
	httpClient     *http.Client
	llmClient      *llmapi.Client
	enabled        bool

	// Agent metadata cache
	cacheMu    sync.RWMutex
	agentCache []AgentMetadataCache
	cacheTime  time.Time
	cacheTTL   time.Duration

	// Model cache
	modelCacheMu   sync.RWMutex
	cachedModel    string
	modelCacheTime time.Time
}

var disabledAgentTypes = map[string]struct{}{}

// AgentMetadataCache holds cached agent metadata.
type AgentMetadataCache struct {
	Type              string   `json:"type"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Keywords          []string `json:"keywords"`
	Capabilities      []string `json:"capabilities"`
	OutputFormats     []string `json:"output_formats"`
	EstimatedDuration string   `json:"estimated_duration"`
	UseWhen           string   `json:"use_when"`
	Enabled           bool     `json:"enabled"`
}

// RunAgentArgs defines the input schema for the run_agent tool.
type RunAgentArgs struct {
	AgentType      string                 `json:"agent_type,omitempty"`
	Type           string                 `json:"type,omitempty"`
	Prompt         string                 `json:"prompt"`
	Model          string                 `json:"model,omitempty"`
	Options        map[string]interface{} `json:"options,omitempty"`
	Attachments    []AttachmentArg        `json:"attachments,omitempty"`
	ArtifactID     string                 `json:"artifact_id,omitempty"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	ToolCallID     string                 `json:"tool_call_id,omitempty"`
	RequestID      string                 `json:"request_id,omitempty"`
	UserID         string                 `json:"user_id,omitempty"`
}

// AttachmentArg represents a file attachment.
type AttachmentArg struct {
	FileID string `json:"file_id,omitempty"`
	URL    string `json:"url,omitempty"`
	Type   string `json:"type,omitempty"`
}

// NewAgentProxyMCP creates a new agent proxy MCP handler.
func NewAgentProxyMCP(responseAPIURL, llmAPIURL string, enabled bool) *AgentProxyMCP {
	if responseAPIURL == "" {
		return nil
	}

	return &AgentProxyMCP{
		responseAPIURL: strings.TrimSuffix(responseAPIURL, "/"),
		llmAPIURL:      strings.TrimSuffix(llmAPIURL, "/"),
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Longer timeout for agent operations
		},
		enabled:  enabled,
		cacheTTL: 5 * time.Minute,
	}
}

// SetLLMClient sets the LLM client for tool call tracking.
func (a *AgentProxyMCP) SetLLMClient(client *llmapi.Client) {
	if a != nil {
		a.llmClient = client
	}
}

// RegisterTools registers the unified run_agent tool with the MCP server.
func (a *AgentProxyMCP) RegisterTools(server *mcpsdk.Server) {
	if a == nil || !a.enabled {
		return
	}

	// Fetch available agents to build description
	agents := a.getAgentsWithCache()
	description := a.buildRunAgentDescription(agents)

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "run_agent",
		Description: description,
	}, a.handleRunAgent)

	log.Info().
		Str("url", a.responseAPIURL).
		Int("agents", len(agents)).
		Msg("Agent proxy run_agent tool registered")
}

// handleRunAgent processes the run_agent tool call.
func (a *AgentProxyMCP) handleRunAgent(ctx context.Context, req *mcpsdk.CallToolRequest, input RunAgentArgs) (*mcpsdk.CallToolResult, map[string]any, error) {
	startTime := time.Now()
	callCtx := extractAllContext(req)

	if input.AgentType == "" {
		input.AgentType = input.Type
	}

	if input.AgentType == "slide_creator" {
		if input.Options == nil {
			input.Options = map[string]interface{}{}
		}
		if _, ok := input.Options["num_slides"]; !ok {
			input.Options["num_slides"] = 5
		}
		if _, ok := input.Options["user_input"]; !ok {
			input.Options["user_input"] = input.Prompt
		}
		if _, ok := input.Options["topic"]; !ok {
			input.Options["topic"] = input.Prompt
		}
	}

	log.Info().
		Str("tool", "run_agent").
		Str("agent_type", input.AgentType).
		Str("model", input.Model).
		Str("prompt", truncateString(input.Prompt, 100)).
		Str("tool_call_id", callCtx["tool_call_id"]).
		Str("request_id", callCtx["request_id"]).
		Msg("run_agent requested")

	// Validate agent type
	if input.AgentType == "" {
		return &mcpsdk.CallToolResult{
			IsError: true,
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "agent_type is required"}},
		}, nil, nil
	}

	if input.Prompt == "" {
		return &mcpsdk.CallToolResult{
			IsError: true,
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "prompt is required"}},
		}, nil, nil
	}

	if isAgentDisabled(input.AgentType) {
		return &mcpsdk.CallToolResult{
			IsError: true,
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "agent_type is disabled"}},
		}, nil, nil
	}

	// Get tracking context
	tracking, _ := GetToolTracking(ctx)

	// Determine model to use
	model := input.Model
	if model == "" {
		// Fetch first available model from llm-api
		var err error
		model, err = a.getFirstAvailableModel(ctx, tracking.AuthToken)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get available model")
			return &mcpsdk.CallToolResult{
				IsError: true,
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "Failed to get available model: " + err.Error()}},
			}, nil, nil
		}
	}

	log.Info().
		Str("model", model).
		Str("agent_type", input.AgentType).
		Msg("Using model for agent execution")

	// Build the /v1/responses request payload
	// The Response API triggers agents via metadata.agent_type
	metadata := map[string]interface{}{
		"agent_type": input.AgentType,
	}

	// Merge additional options into metadata if provided
	if input.Options != nil {
		for k, v := range input.Options {
			metadata[k] = v
		}
	}
	if input.ArtifactID != "" {
		metadata["artifact_id"] = input.ArtifactID
	}

	payload := map[string]interface{}{
		"model":      model,
		"input":      input.Prompt,
		"metadata":   metadata,
		"store":      true, // Store response for tracking
		"background": true, // Run in background for long-running agents
	}

	// Pass conversation ID to response-api so artifacts link to the correct llm-api conversation.
	// Priority: 1) explicit input.ConversationID, 2) tracking.ConversationID from parent context
	// This is stored as parent_conversation_id in response-api for artifact linking (no lookup, just stored).
	conversationID := input.ConversationID
	if conversationID == "" && tracking.ConversationID != "" {
		conversationID = tracking.ConversationID
	}
	if conversationID != "" {
		payload["parent_conversation_id"] = conversationID
	}

	// Add user ID if provided
	if input.UserID != "" {
		payload["user"] = input.UserID
	}

	// Execute agent via Response API /v1/responses endpoint
	authToken := tracking.AuthToken
	if authToken == "" {
		// Try to get from context
		if token, ok := callCtx["auth_token"]; ok {
			authToken = token
		}
	}

	resp, err := a.callResponseAPI(ctx, "/v1/responses", payload, authToken)
	if err != nil {
		log.Error().
			Err(err).
			Str("agent_type", input.AgentType).
			Str("model", model).
			Msg("Failed to execute agent")

		return &mcpsdk.CallToolResult{
			IsError: true,
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "Agent execution failed: " + err.Error()}},
		}, nil, nil
	}

	duration := time.Since(startTime)
	log.Info().
		Str("tool", "run_agent").
		Str("agent_type", input.AgentType).
		Str("model", model).
		Dur("duration", duration).
		Msg("run_agent completed")

	// Format result for MCP tool response
	resultJSON, _ := json.Marshal(resp)
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(resultJSON)}},
	}, resp, nil
}

// getFirstAvailableModel fetches the first available model from llm-api.
func (a *AgentProxyMCP) getFirstAvailableModel(ctx context.Context, authToken string) (string, error) {
	// Check cache first
	a.modelCacheMu.RLock()
	if a.cachedModel != "" && time.Since(a.modelCacheTime) < a.cacheTTL {
		model := a.cachedModel
		a.modelCacheMu.RUnlock()
		return model, nil
	}
	a.modelCacheMu.RUnlock()

	// Fetch from llm-api
	if a.llmAPIURL == "" {
		return "", fmt.Errorf("llm-api URL not configured")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", a.llmAPIURL+"/v1/models", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Data) == 0 {
		return "", fmt.Errorf("no models available")
	}

	model := result.Data[0].ID

	// Cache the result
	a.modelCacheMu.Lock()
	a.cachedModel = model
	a.modelCacheTime = time.Now()
	a.modelCacheMu.Unlock()

	log.Info().Str("model", model).Msg("Fetched first available model from llm-api")

	return model, nil
}

// callResponseAPI makes an HTTP call to the Response API.
func (a *AgentProxyMCP) callResponseAPI(ctx context.Context, path string, payload map[string]interface{}, authToken string) (map[string]interface{}, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := a.responseAPIURL + path
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// getAgentsWithCache returns cached agents or fetches fresh data.
func (a *AgentProxyMCP) getAgentsWithCache() []AgentMetadataCache {
	a.cacheMu.RLock()
	if len(a.agentCache) > 0 && time.Since(a.cacheTime) < a.cacheTTL {
		agents := a.agentCache
		a.cacheMu.RUnlock()
		return agents
	}
	a.cacheMu.RUnlock()

	// Fetch fresh data
	agents, err := a.fetchAvailableAgents()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch agents from Response API, using defaults")
		return a.getDefaultAgents()
	}
	agents = filterDisabledAgents(agents)

	// Update cache
	a.cacheMu.Lock()
	a.agentCache = agents
	a.cacheTime = time.Now()
	a.cacheMu.Unlock()

	return agents
}

// fetchAvailableAgents gets agent list from Response API.
func (a *AgentProxyMCP) fetchAvailableAgents() ([]AgentMetadataCache, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := a.responseAPIURL + "/v1/agents"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	var result struct {
		Agents []AgentMetadataCache `json:"agents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Agents, nil
}

// buildRunAgentDescription creates a descriptive string for LLM.
func (a *AgentProxyMCP) buildRunAgentDescription(agents []AgentMetadataCache) string {
	var sb strings.Builder
	sb.WriteString("Run a specialized agent to perform complex multi-step tasks. ")
	sb.WriteString("Each agent creates a plan with visible nested tool calls and produces artifacts.\n\n")
	sb.WriteString("Parameters:\n")
	sb.WriteString("- type (required): The type of agent to run (alias: agent_type)\n")
	sb.WriteString("- prompt (required): The task description for the agent\n")
	sb.WriteString("- model (optional): The model to use. If not provided, uses the first available model\n")
	sb.WriteString("- options (optional): Agent-specific options (e.g., research_depth, num_slides, format)\n")
	sb.WriteString("\nIf type is slide_creator, require and provide in options:\n")
	sb.WriteString("- topic: summarized topic for the deck\n")
	sb.WriteString("- tone: template tone (must be one of the predefined tones)\n")
	sb.WriteString("- num_slides: target number of slides (default: 5)\n")
	sb.WriteString("- user_input: original user prompt (must match user request)\n")
	sb.WriteString("Available slide_creator tones:\n- ")
	sb.WriteString(strings.Join(slideCreatorTones, "\n- "))
	sb.WriteString("\n")
	sb.WriteString("Available agents:\n")

	for _, agent := range agents {
		if agent.Enabled {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", agent.Type, agent.Description))
		}
	}

	return sb.String()
}

var slideCreatorTones = []string{
	"Corporate Consulting",
	"Creative Studio",
	"Data Analyst",
	"Editorial Serif",
	"Education Friendly",
	"Finance Professional",
	"Government/Public Sector",
	"Gradient Modern",
	"Healthcare Calm",
	"Luxury Elegant",
	"Marketing Vibrant",
	"Minimal Clean",
	"Monochrome",
	"Neon Cyberpunk",
	"Playful Pastel",
	"Retro",
	"Sports/Energy",
	"Startup Bold",
	"Sustainability Earthy",
	"Tech Dark",
}

func isSlideCreatorTone(tone string) bool {
	for _, candidate := range slideCreatorTones {
		if tone == candidate {
			return true
		}
	}
	return false
}

func isAgentDisabled(agentType string) bool {
	_, disabled := disabledAgentTypes[strings.TrimSpace(agentType)]
	return disabled
}

func filterDisabledAgents(agents []AgentMetadataCache) []AgentMetadataCache {
	if len(agents) == 0 {
		return agents
	}
	filtered := make([]AgentMetadataCache, 0, len(agents))
	for _, agent := range agents {
		if isAgentDisabled(agent.Type) {
			continue
		}
		filtered = append(filtered, agent)
	}
	return filtered
}

// getEnabledAgentTypes returns a list of enabled agent type strings.
func (a *AgentProxyMCP) getEnabledAgentTypes(agents []AgentMetadataCache) []string {
	types := make([]string, 0, len(agents))
	for _, agent := range agents {
		if agent.Enabled {
			types = append(types, agent.Type)
		}
	}
	return types
}

// getDefaultAgents returns default agent list if API is unavailable.
func (a *AgentProxyMCP) getDefaultAgents() []AgentMetadataCache {
	return []AgentMetadataCache{
		{
			Type:              "deep_research",
			Name:              "Deep Research Agent",
			Description:       "Performs comprehensive web research with source verification and cross-referencing",
			Keywords:          []string{"research", "analyze", "investigate", "study"},
			Capabilities:      []string{"web_search", "scrape", "cross_reference", "report_generation"},
			OutputFormats:     []string{"markdown", "pdf", "json"},
			EstimatedDuration: "2-10 minutes",
			UseWhen:           "User wants to research, investigate, analyze, or learn about a topic in depth",
			Enabled:           true,
		},
		{
			Type:              "slide_creator",
			Name:              "Slide Creator Agent",
			Description:       "Builds HTML-based slide decks and exports them to editable PPTX with research-backed content",
			Keywords:          []string{"slides", "presentation", "powerpoint", "deck", "pitch"},
			Capabilities:      []string{"research", "outline", "html_slide_generation", "template_selection", "pptx_export"},
			OutputFormats:     []string{"pptx", "html"},
			EstimatedDuration: "3-15 minutes",
			UseWhen:           "User wants an editable PPTX generated from HTML slide layouts with template control",
			Enabled:           true,
		},
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
