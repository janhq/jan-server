package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	domainsearch "jan-server/services/mcp-tools/internal/domain/search"
	"jan-server/services/mcp-tools/internal/infrastructure/llmapi"
	"jan-server/services/mcp-tools/internal/infrastructure/metrics"
	"jan-server/services/mcp-tools/internal/infrastructure/toolconfig"
	"jan-server/services/mcp-tools/internal/infrastructure/vectorstore"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

// SearchArgs defines the arguments for the google_search tool
type SearchArgs struct {
	Q               string   `json:"q"`
	DomainAllowList []string `json:"domain_allow_list,omitempty"`
	GL              *string  `json:"gl,omitempty"`
	HL              *string  `json:"hl,omitempty"`
	Location        *string  `json:"location,omitempty"`
	Num             *int     `json:"num,omitempty"`
	Tbs             *string  `json:"tbs,omitempty"`
	Page            *int     `json:"page,omitempty"`
	Autocorrect     *bool    `json:"autocorrect,omitempty"`
	LocationHint    *string  `json:"location_hint,omitempty"`
	OfflineMode     *bool    `json:"offline_mode,omitempty"`
	// Context passthrough (ignored by handler but allowed for validation)
	ToolCallID     string `json:"tool_call_id,omitempty"`
	RequestID      string `json:"request_id,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
}

// ImageSearchArgs defines the arguments for the image_search tool
type ImageSearchArgs struct {
	Q           string  `json:"q"`
	GL          *string `json:"gl,omitempty"`
	HL          *string `json:"hl,omitempty"`
	Num         *int    `json:"num,omitempty"`
	Autocorrect *bool   `json:"autocorrect,omitempty"`
	OfflineMode *bool   `json:"offline_mode,omitempty"`
	// Context passthrough
	ToolCallID     string `json:"tool_call_id,omitempty"`
	RequestID      string `json:"request_id,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
}

// ScrapeArgs defines the arguments for the scrape tool
type ScrapeArgs struct {
	Url             string `json:"url"`
	IncludeMarkdown *bool  `json:"includeMarkdown,omitempty"`
	OfflineMode     *bool  `json:"offline_mode,omitempty"`
	// Context passthrough
	ToolCallID     string `json:"tool_call_id,omitempty"`
	RequestID      string `json:"request_id,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
}

type FileSearchIndexArgs struct {
	DocumentID string         `json:"document_id"`
	Text       string         `json:"text"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Tags       []string       `json:"tags,omitempty"`
	// Context passthrough
	ToolCallID     string `json:"tool_call_id,omitempty"`
	RequestID      string `json:"request_id,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
}

type FileSearchQueryArgs struct {
	Query       string   `json:"query"`
	TopK        *int     `json:"top_k,omitempty"`
	DocumentIDs []string `json:"document_ids,omitempty"`
	// Context passthrough
	ToolCallID     string `json:"tool_call_id,omitempty"`
	RequestID      string `json:"request_id,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
}

type searchToolResult struct {
	Position    int    `json:"position"`
	Title       string `json:"title"`
	SourceURL   string `json:"source_url"`
	Snippet     string `json:"snippet"`
	CacheStatus string `json:"cache_status"`
	FetchedAt   string `json:"fetched_at"`
}

type searchToolPayload struct {
	Query       string                       `json:"query"`
	Engine      string                       `json:"engine"`
	Live        bool                         `json:"live"`
	CacheStatus string                       `json:"cache_status"`
	Metadata    map[string]any               `json:"metadata"`
	Results     []searchToolResult           `json:"results"`
	Citations   []string                     `json:"citations"`
	Raw         *domainsearch.SearchResponse `json:"raw,omitempty"`
}

type scrapeToolPayload struct {
	SourceURL   string         `json:"source_url"`
	Text        string         `json:"text"`
	TextPreview string         `json:"text_preview"`
	Metadata    map[string]any `json:"metadata"`
	CacheStatus string         `json:"cache_status"`
	FetchedAt   string         `json:"fetched_at"`
}

type imageSearchToolResult struct {
	Position        int    `json:"position"`
	Title           string `json:"title"`
	ImageURL        string `json:"image_url"`
	ImageWidth      int    `json:"image_width"`
	ImageHeight     int    `json:"image_height"`
	ThumbnailURL    string `json:"thumbnail_url"`
	ThumbnailWidth  int    `json:"thumbnail_width,omitempty"`
	ThumbnailHeight int    `json:"thumbnail_height,omitempty"`
	Source          string `json:"source"`
	Domain          string `json:"domain"`
	Link            string `json:"link"`
	Creator         string `json:"creator,omitempty"`
	Credit          string `json:"credit,omitempty"`
}

type imageSearchToolPayload struct {
	Query       string                            `json:"query"`
	Engine      string                            `json:"engine"`
	Live        bool                              `json:"live"`
	CacheStatus string                            `json:"cache_status"`
	Metadata    map[string]any                    `json:"metadata"`
	Results     []imageSearchToolResult           `json:"results"`
	Raw         *domainsearch.ImageSearchResponse `json:"raw,omitempty"`
}

// SearchMCP handles MCP tool registration for search tooling.
type SearchMCP struct {
	searchService         *domainsearch.SearchService
	vectorStore           *vectorstore.Client
	llmClient             *llmapi.Client    // LLM-API client for tool tracking
	toolConfigCache       *toolconfig.Cache // Cache for dynamic tool configurations
	fileIndexMu           sync.Mutex
	fileIndex             map[string]FileSearchIndexArgs
	maxSnippetChars       int
	maxScrapePreviewChars int
	maxScrapeTextChars    int
	enableFileSearch      bool
}

// SearchMCPConfig contains configuration for SearchMCP.
type SearchMCPConfig struct {
	MaxSnippetChars       int
	MaxScrapePreviewChars int
	MaxScrapeTextChars    int
	EnableFileSearch      bool
}

// NewSearchMCP creates a new search MCP handler.
func NewSearchMCP(searchService *domainsearch.SearchService, vectorStore *vectorstore.Client, cfg SearchMCPConfig) *SearchMCP {
	// Apply defaults if not set
	maxSnippet := cfg.MaxSnippetChars
	if maxSnippet <= 0 {
		maxSnippet = 5000
	}
	maxPreview := cfg.MaxScrapePreviewChars
	if maxPreview <= 0 {
		maxPreview = 600
	}
	maxText := cfg.MaxScrapeTextChars
	if maxText <= 0 {
		maxText = 50000
	}

	return &SearchMCP{
		searchService:         searchService,
		vectorStore:           vectorStore,
		fileIndex:             make(map[string]FileSearchIndexArgs),
		maxSnippetChars:       maxSnippet,
		maxScrapePreviewChars: maxPreview,
		maxScrapeTextChars:    maxText,
		enableFileSearch:      cfg.EnableFileSearch,
	}
}

// SetLLMClient sets the LLM-API client for tool call tracking
func (s *SearchMCP) SetLLMClient(client *llmapi.Client) {
	s.llmClient = client
}

// SetToolConfigCache sets the tool config cache for dynamic descriptions
func (s *SearchMCP) SetToolConfigCache(cache *toolconfig.Cache) {
	s.toolConfigCache = cache
}

// Tool key constants (matching llm-api mcptool domain)
const (
	ToolKeyGoogleSearch    = "google_search"
	ToolKeyImageSearch     = "image_search"
	ToolKeyScrape          = "scrape"
	ToolKeyFileSearchIndex = "file_search_index"
	ToolKeyFileSearchQuery = "file_search_query"
)

// Default tool descriptions (fallback when cache is unavailable)
var defaultToolDescriptions = map[string]string{
	ToolKeyGoogleSearch:    "Perform web searches via the configured engines (Serper, Exa, Tavily, or SearXNG) and fetch structured citations.",
	ToolKeyImageSearch:     "Search for images using Serper API. Returns image URLs, dimensions, thumbnails, and source information.",
	ToolKeyScrape:          "Scrape a webpage and retrieve the text with optional markdown formatting using the configured providers.",
	ToolKeyFileSearchIndex: "Index arbitrary text into the lightweight vector store used for MCP automations.",
	ToolKeyFileSearchQuery: "Run a semantic query against documents indexed via file_search_index.",
}

// getToolDescription gets the description for a tool, using cached config if available
func (s *SearchMCP) getToolDescription(ctx context.Context, toolKey string) string {
	if s.toolConfigCache != nil {
		tool, err := s.toolConfigCache.GetToolByKey(ctx, toolKey)
		if err == nil && tool != nil {
			return tool.Config.Description
		}
	}
	// Fallback to default
	if desc, ok := defaultToolDescriptions[toolKey]; ok {
		return desc
	}
	return ""
}

// isToolActive checks if a tool is active (should be listed/callable)
func (s *SearchMCP) isToolActive(ctx context.Context, toolKey string) bool {
	if s.toolConfigCache != nil {
		tool, err := s.toolConfigCache.GetToolByKey(ctx, toolKey)
		if err == nil && tool != nil {
			return tool.Config.IsActive
		}
	}
	// Default to active if no config found
	return true
}

// getToolConfig gets the full cached tool config
func (s *SearchMCP) getToolConfig(ctx context.Context, toolKey string) *toolconfig.CachedTool {
	if s.toolConfigCache != nil {
		tool, _ := s.toolConfigCache.GetToolByKey(ctx, toolKey)
		return tool
	}
	return nil
}

// RegisterTools registers search tools with the MCP server
// Note: Tool descriptions are fetched dynamically from cache when available
func (s *SearchMCP) RegisterTools(server *mcp.Server) {
	// Use background context for initial description fetch
	ctx := context.Background()

	// Register google_search tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        ToolKeyGoogleSearch,
		Description: s.getToolDescription(ctx, ToolKeyGoogleSearch),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SearchArgs) (*mcp.CallToolResult, searchToolPayload, error) {
		// Check if tool is active
		if !s.isToolActive(ctx, ToolKeyGoogleSearch) {
			disabledPayload := searchToolPayload{
				Query:       input.Q,
				Engine:      "disabled",
				Live:        false,
				CacheStatus: "disabled",
				Metadata: map[string]any{
					"error": "tool is disabled",
				},
				Results:   []searchToolResult{},
				Citations: []string{},
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "tool is disabled"}},
				IsError: true,
			}, disabledPayload, nil
		}

		startTime := time.Now()
		callCtx := extractAllContext(req)

		// Check for tracking context from headers
		tracking, trackingEnabled := GetToolTracking(ctx)

		log.Info().
			Str("tool", "google_search").
			Str("tool_call_id", callCtx["tool_call_id"]).
			Str("request_id", callCtx["request_id"]).
			Str("conversation_id", callCtx["conversation_id"]).
			Str("user_id", callCtx["user_id"]).
			Bool("tracking_enabled", trackingEnabled).
			Msg("MCP tool call received")

		log.Debug().
			Str("tool", "google_search").
			Str("query", input.Q).
			Interface("domain_allow_list", input.DomainAllowList).
			Interface("location_hint", input.LocationHint).
			Interface("offline_mode", input.OfflineMode).
			Interface("num", input.Num).
			Msg("google_search request details")

		searchReq := domainsearch.SearchRequest{
			Q: input.Q,
		}
		if len(input.DomainAllowList) > 0 {
			searchReq.DomainAllowList = input.DomainAllowList
		}

		if input.GL != nil {
			searchReq.GL = input.GL
		}
		if input.HL != nil {
			searchReq.HL = input.HL
		}
		if input.Location != nil {
			searchReq.Location = input.Location
		}
		if input.Num != nil && *input.Num > 0 {
			searchReq.Num = input.Num
		}
		if input.Page != nil && *input.Page > 0 {
			searchReq.Page = input.Page
		}
		if input.Tbs != nil && *input.Tbs != "" {
			val := domainsearch.TBSTimeRange(*input.Tbs)
			searchReq.TBS = &val
		}
		autocorrect := true
		if input.Autocorrect != nil {
			autocorrect = *input.Autocorrect
		}
		searchReq.Autocorrect = &autocorrect

		if input.LocationHint != nil {
			searchReq.LocationHint = input.LocationHint
		}
		if input.OfflineMode != nil {
			searchReq.OfflineMode = input.OfflineMode
		}

		var payload searchToolPayload
		var toolErr error

		searchResp, err := s.searchService.Search(ctx, searchReq)
		if err != nil {
			log.Warn().Err(err).Str("tool", "google_search").Str("query", searchReq.Q).Msg("search service failed")
			toolErr = err
			payload = searchToolPayload{
				Query:       searchReq.Q,
				Engine:      "error",
				Live:        false,
				CacheStatus: "error",
				Metadata: map[string]any{
					"error": toolErr.Error(),
				},
				Results:   []searchToolResult{},
				Citations: []string{},
			}
		} else {
			log.Debug().
				Str("tool", "google_search").
				Str("query", searchReq.Q).
				Int("result_count", len(searchResp.Organic)).
				Interface("engine", searchResp.SearchParameters["engine"]).
				Bool("live", searchResp.SearchParameters["live"] == true).
				Msg("google_search response received")
			payload = s.buildSearchPayload(searchReq.Q, searchReq, searchResp)
			// Apply disallowed keyword filtering
			payload = s.filterSearchResults(ctx, ToolKeyGoogleSearch, payload)
		}

		// If tracking is enabled, save result to LLM-API (single PATCH call)
		if trackingEnabled && s.llmClient != nil {
			// Capture input for async goroutine
			inputCopy := input
			go func() {
				saveCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				// Serialize the result
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

				// Update the in_progress item to completed
				result := s.llmClient.UpdateToolCallResult(
					saveCtx,
					tracking.AuthToken,
					tracking.ConversationID,
					tracking.ToolCallID,
					"google_search",
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

		if toolErr != nil {
			if payload.Metadata == nil {
				payload.Metadata = map[string]any{
					"error": toolErr.Error(),
				}
			}
			if payload.Results == nil {
				payload.Results = []searchToolResult{}
			}
			if payload.Citations == nil {
				payload.Citations = []string{}
			}
			metrics.RecordToolCall("google_search", "none", "error", time.Since(startTime).Seconds())
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: toolErr.Error()}},
				IsError: true,
			}, payload, nil
		}
		if payload.Metadata == nil {
			payload.Metadata = map[string]any{}
		}
		if payload.Results == nil {
			payload.Results = []searchToolResult{}
		}
		if payload.Citations == nil {
			payload.Citations = []string{}
		}

		// Estimate payload tokens for observability
		estimatedTokens := estimateTokensFromSearchPayload(payload)

		provider := payload.Engine
		if provider == "" {
			provider = "unknown"
		}
		metrics.RecordToolCall("google_search", provider, "success", time.Since(startTime).Seconds())
		if estimatedTokens > 0 {
			metrics.RecordToolTokens("google_search", provider, estimatedTokens)
		}

		return nil, payload, nil
	})

	// Register image_search tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        ToolKeyImageSearch,
		Description: s.getToolDescription(ctx, ToolKeyImageSearch),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ImageSearchArgs) (*mcp.CallToolResult, imageSearchToolPayload, error) {
		// Check if tool is active
		if !s.isToolActive(ctx, ToolKeyImageSearch) {
			disabledPayload := imageSearchToolPayload{
				Query:       input.Q,
				Engine:      "disabled",
				Live:        false,
				CacheStatus: "disabled",
				Metadata: map[string]any{
					"error": "tool is disabled",
				},
				Results: []imageSearchToolResult{},
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "tool is disabled"}},
				IsError: true,
			}, disabledPayload, nil
		}

		startTime := time.Now()
		callCtx := extractAllContext(req)

		// Check for tracking context from headers
		tracking, trackingEnabled := GetToolTracking(ctx)

		log.Info().
			Str("tool", "image_search").
			Str("tool_call_id", callCtx["tool_call_id"]).
			Str("request_id", callCtx["request_id"]).
			Str("conversation_id", callCtx["conversation_id"]).
			Str("user_id", callCtx["user_id"]).
			Bool("tracking_enabled", trackingEnabled).
			Msg("MCP tool call received")

		log.Debug().
			Str("tool", "image_search").
			Str("query", input.Q).
			Interface("gl", input.GL).
			Interface("hl", input.HL).
			Interface("num", input.Num).
			Interface("offline_mode", input.OfflineMode).
			Msg("image_search request details")

		imageSearchReq := domainsearch.ImageSearchRequest{
			Q: input.Q,
		}
		if input.GL != nil {
			imageSearchReq.GL = input.GL
		}
		if input.HL != nil {
			imageSearchReq.HL = input.HL
		}
		if input.Num != nil && *input.Num > 0 {
			imageSearchReq.Num = input.Num
		}
		autocorrect := true
		if input.Autocorrect != nil {
			autocorrect = *input.Autocorrect
		}
		imageSearchReq.Autocorrect = &autocorrect
		if input.OfflineMode != nil {
			imageSearchReq.OfflineMode = input.OfflineMode
		}

		var payload imageSearchToolPayload
		var toolErr error

		imageSearchResp, err := s.searchService.ImageSearch(ctx, imageSearchReq)
		if err != nil {
			log.Warn().Err(err).Str("tool", "image_search").Str("query", imageSearchReq.Q).Msg("image search service failed")
			toolErr = err
			payload = imageSearchToolPayload{
				Query:       imageSearchReq.Q,
				Engine:      "error",
				Live:        false,
				CacheStatus: "error",
				Metadata: map[string]any{
					"error": toolErr.Error(),
				},
				Results: []imageSearchToolResult{},
			}
		} else {
			log.Debug().
				Str("tool", "image_search").
				Str("query", imageSearchReq.Q).
				Int("result_count", len(imageSearchResp.Images)).
				Interface("engine", imageSearchResp.SearchParameters["engine"]).
				Msg("image_search response received")
			payload = s.buildImageSearchPayload(imageSearchReq.Q, imageSearchResp)
		}

		// If tracking is enabled, save result to LLM-API
		if trackingEnabled && s.llmClient != nil {
			inputCopy := input
			go func() {
				saveCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				outputBytes, _ := json.Marshal(payload)
				outputStr := string(outputBytes)

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
					"image_search",
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

		if toolErr != nil {
			if payload.Metadata == nil {
				payload.Metadata = map[string]any{
					"error": toolErr.Error(),
				}
			}
			if payload.Results == nil {
				payload.Results = []imageSearchToolResult{}
			}
			metrics.RecordToolCall("image_search", "none", "error", time.Since(startTime).Seconds())
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: toolErr.Error()}},
				IsError: true,
			}, payload, nil
		}
		if payload.Metadata == nil {
			payload.Metadata = map[string]any{}
		}
		if payload.Results == nil {
			payload.Results = []imageSearchToolResult{}
		}

		// Estimate payload tokens for observability
		estimatedTokens := estimateTokensFromImageSearchPayload(payload)

		provider := payload.Engine
		if provider == "" {
			provider = "unknown"
		}
		metrics.RecordToolCall("image_search", provider, "success", time.Since(startTime).Seconds())
		if estimatedTokens > 0 {
			metrics.RecordToolTokens("image_search", provider, estimatedTokens)
		}

		return nil, payload, nil
	})

	// Register scrape tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        ToolKeyScrape,
		Description: s.getToolDescription(ctx, ToolKeyScrape),
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ScrapeArgs) (*mcp.CallToolResult, scrapeToolPayload, error) {
		// Check if tool is active
		if !s.isToolActive(ctx, ToolKeyScrape) {
			disabledPayload := scrapeToolPayload{
				SourceURL:   input.Url,
				Text:        "",
				TextPreview: "",
				Metadata: map[string]any{
					"error": "tool is disabled",
				},
				CacheStatus: "disabled",
				FetchedAt:   time.Now().UTC().Format(time.RFC3339),
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "tool is disabled"}},
				IsError: true,
			}, disabledPayload, nil
		}

		startTime := time.Now()
		callCtx := extractAllContext(req)

		// Check for tracking context from headers
		tracking, trackingEnabled := GetToolTracking(ctx)

		log.Info().
			Str("tool", "scrape").
			Str("tool_call_id", callCtx["tool_call_id"]).
			Str("request_id", callCtx["request_id"]).
			Str("conversation_id", callCtx["conversation_id"]).
			Str("user_id", callCtx["user_id"]).
			Bool("tracking_enabled", trackingEnabled).
			Msg("MCP tool call received")

		log.Debug().
			Str("tool", "scrape").
			Str("url", input.Url).
			Interface("include_markdown", input.IncludeMarkdown).
			Interface("offline_mode", input.OfflineMode).
			Msg("scrape request details")

		scrapeReq := domainsearch.FetchWebpageRequest{
			Url: input.Url,
		}

		if input.IncludeMarkdown != nil && *input.IncludeMarkdown {
			scrapeReq.IncludeMarkdown = input.IncludeMarkdown
		}
		if input.OfflineMode != nil {
			scrapeReq.OfflineMode = input.OfflineMode
		}

		var payload scrapeToolPayload
		var toolErr error

		scrapeResp, err := s.searchService.FetchWebpage(ctx, scrapeReq)
		if err != nil {
			log.Warn().Err(err).Str("tool", "scrape").Str("url", scrapeReq.Url).Msg("scrape service failed")
			toolErr = err
			payload = scrapeToolPayload{
				SourceURL:   scrapeReq.Url,
				Text:        "",
				TextPreview: "",
				Metadata: map[string]any{
					"error": toolErr.Error(),
				},
				CacheStatus: "error",
				FetchedAt:   time.Now().UTC().Format(time.RFC3339),
			}
		} else {
			log.Debug().
				Str("tool", "scrape").
				Str("url", scrapeReq.Url).
				Str("status", scrapeResp.Status).
				Int("text_length", len(scrapeResp.Text)).
				Interface("metadata", scrapeResp.Metadata).
				Msg("scrape response received")
			payload = s.buildScrapePayload(scrapeReq.Url, scrapeResp)
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
					"scrape",
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

		if toolErr != nil {
			if payload.Metadata == nil {
				payload.Metadata = map[string]any{
					"error": toolErr.Error(),
				}
			}
			metrics.RecordToolCall("scrape", "none", "error", time.Since(startTime).Seconds())
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: toolErr.Error()}},
				IsError: true,
			}, payload, nil
		}
		if payload.Metadata == nil {
			payload.Metadata = map[string]any{}
		}

		// Estimate payload tokens for observability
		scrapeTokens := estimateTokensFromScrapePayload(payload)

		provider := "unknown"
		if payload.Metadata != nil {
			if val, ok := payload.Metadata["provider"].(string); ok && val != "" {
				provider = val
			} else if val, ok := payload.Metadata["fallback_mode"].(bool); ok && val {
				provider = "direct-http"
			}
		}
		metrics.RecordToolCall("scrape", provider, "success", time.Since(startTime).Seconds())
		if scrapeTokens > 0 {
			metrics.RecordToolTokens("scrape", provider, scrapeTokens)
		}

		return nil, payload, nil
	})

	// file_search_index and file_search_query tools (conditionally enabled)
	if !s.enableFileSearch {
		log.Warn().Msg("file_search_index and file_search_query MCP tools disabled via config")
	} else {
		mcp.AddTool(server, &mcp.Tool{
			Name:        ToolKeyFileSearchIndex,
			Description: s.getToolDescription(ctx, ToolKeyFileSearchIndex),
		}, func(ctx context.Context, req *mcp.CallToolRequest, input FileSearchIndexArgs) (*mcp.CallToolResult, map[string]any, error) {
			// Check if tool is active
			if !s.isToolActive(ctx, ToolKeyFileSearchIndex) {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: "tool is disabled"}},
					IsError: true,
				}, nil, nil
			}

			startTime := time.Now()
			callCtx := extractAllContext(req)
			log.Info().
				Str("tool", ToolKeyFileSearchIndex).
				Str("tool_call_id", callCtx["tool_call_id"]).
				Str("request_id", callCtx["request_id"]).
				Str("conversation_id", callCtx["conversation_id"]).
				Str("user_id", callCtx["user_id"]).
				Msg("MCP tool call received")

			status := "success"
			var tokens float64

			if s.vectorStore != nil && s.vectorStore.IsEnabled() {
				resp, err := s.vectorStore.IndexDocument(ctx, vectorstore.IndexRequest{
					DocumentID: input.DocumentID,
					Text:       input.Text,
					Metadata:   input.Metadata,
					Tags:       input.Tags,
				})
				if err == nil {
					tokens = float64(resp.TokenCount)
					metrics.RecordToolCall("file_search_index", "vectorstore", status, time.Since(startTime).Seconds())
					if tokens > 0 {
						metrics.RecordToolTokens("file_search_index", "vectorstore", tokens)
					}
					return nil, map[string]any{
						"document_id": resp.DocumentID,
						"status":      resp.Status,
						"indexed_at":  resp.IndexedAt,
						"token_count": resp.TokenCount,
					}, nil
				}
				log.Warn().Err(err).Str("tool", "file_search_index").Msg("vector store index failed; falling back to stub")
				status = "error"
			}

			s.fileIndexMu.Lock()
			s.fileIndex[input.DocumentID] = input
			s.fileIndexMu.Unlock()

			tokens = float64(len(input.Text)) / 4
			metrics.RecordToolCall("file_search_index", "vectorstore", status, time.Since(startTime).Seconds())
			if tokens > 0 {
				metrics.RecordToolTokens("file_search_index", "vectorstore", tokens)
			}

			return nil, map[string]any{
				"document_id": input.DocumentID,
				"status":      "indexed",
				"indexed_at":  time.Now().UTC().Format(time.RFC3339),
				"token_count": len(input.Text),
			}, nil
		})

		mcp.AddTool(server, &mcp.Tool{
			Name:        ToolKeyFileSearchQuery,
			Description: s.getToolDescription(ctx, ToolKeyFileSearchQuery),
		}, func(ctx context.Context, req *mcp.CallToolRequest, input FileSearchQueryArgs) (*mcp.CallToolResult, map[string]any, error) {
			// Check if tool is active
			if !s.isToolActive(ctx, ToolKeyFileSearchQuery) {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: "tool is disabled"}},
					IsError: true,
				}, nil, nil
			}

			startTime := time.Now()
			callCtx := extractAllContext(req)
			log.Info().
				Str("tool", "file_search_query").
				Str("tool_call_id", callCtx["tool_call_id"]).
				Str("request_id", callCtx["request_id"]).
				Str("conversation_id", callCtx["conversation_id"]).
				Str("user_id", callCtx["user_id"]).
				Msg("MCP tool call received")

			status := "success"
			var tokens float64

			topK := 5
			if input.TopK != nil && *input.TopK > 0 {
				topK = *input.TopK
			}
			if topK > 20 {
				topK = 20
			}

			if s.vectorStore != nil && s.vectorStore.IsEnabled() {
				resp, err := s.vectorStore.Query(ctx, vectorstore.QueryRequest{
					Text:        input.Query,
					TopK:        topK,
					DocumentIDs: input.DocumentIDs,
				})
				if err == nil {
					for _, r := range resp.Results {
						tokens += float64(len(r.TextPreview)) / 4
					}
					metrics.RecordToolCall("file_search_query", "vectorstore", status, time.Since(startTime).Seconds())
					if tokens > 0 {
						metrics.RecordToolTokens("file_search_query", "vectorstore", tokens)
					}
					return nil, map[string]any{
						"query":   resp.Query,
						"top_k":   resp.TopK,
						"count":   resp.Count,
						"results": resp.Results,
					}, nil
				}
				log.Warn().Err(err).Str("tool", "file_search_query").Msg("vector store query failed; falling back to stub")
				status = "error"
			}

			s.fileIndexMu.Lock()
			defer s.fileIndexMu.Unlock()
			results := make([]map[string]any, 0)
			for docID, doc := range s.fileIndex {
				if len(input.DocumentIDs) > 0 {
					match := false
					for _, allowed := range input.DocumentIDs {
						if allowed == docID {
							match = true
							break
						}
					}
					if !match {
						continue
					}
				}
				preview := truncateSnippet(doc.Text, 200)
				results = append(results, map[string]any{
					"document_id":  docID,
					"text_preview": preview,
					"score":        1.0,
					"metadata":     doc.Metadata,
					"tags":         doc.Tags,
				})
				if len(results) >= topK {
					break
				}
			}

			for _, r := range results {
				if preview, ok := r["text_preview"].(string); ok {
					tokens += float64(len(preview)) / 4
				}
			}
			metrics.RecordToolCall("file_search_query", "vectorstore", status, time.Since(startTime).Seconds())
			if tokens > 0 {
				metrics.RecordToolTokens("file_search_query", "vectorstore", tokens)
			}

			return nil, map[string]any{
				"query":   input.Query,
				"top_k":   topK,
				"count":   len(results),
				"results": results,
			}, nil
		})
	} // end if enableFileSearch
}

func (s *SearchMCP) buildSearchPayload(query string, req domainsearch.SearchRequest, resp *domainsearch.SearchResponse) searchToolPayload {
	now := time.Now().UTC().Format(time.RFC3339)

	metadata := map[string]any{}
	if resp != nil && resp.SearchParameters != nil {
		metadata = resp.SearchParameters
	}

	engine := stringFromMap(metadata, "engine")
	if engine == "" {
		engine = "serper"
	}

	live := true
	if resp != nil && resp.SearchParameters != nil {
		if val, ok := resp.SearchParameters["live"].(bool); ok {
			live = val
		}
	}

	cacheStatus := "live"
	if !live {
		if reason := stringFromMap(metadata, "reason"); reason != "" {
			cacheStatus = reason
		} else {
			cacheStatus = "fallback"
		}
	}

	results := make([]searchToolResult, 0)
	citations := make([]string, 0)

	if resp != nil {
		for idx, item := range resp.Organic {
			sourceURL := stringFromMap(item, "link")
			snippet := firstNonEmpty(
				stringFromMap(item, "snippet"),
				stringFromMap(item, "description"),
			)
			if snippet == "" {
				snippet = "No snippet returned by upstream engine."
			}

			results = append(results, searchToolResult{
				Position:    idx + 1,
				Title:       stringFromMap(item, "title"),
				SourceURL:   sourceURL,
				Snippet:     truncateSnippet(snippet, s.maxSnippetChars),
				CacheStatus: cacheStatus,
				FetchedAt:   now,
			})

			if sourceURL != "" {
				citations = append(citations, sourceURL)
			}
		}
	}

	payload := searchToolPayload{
		Query:       query,
		Engine:      engine,
		Live:        live,
		CacheStatus: cacheStatus,
		Metadata:    metadata,
		Results:     results,
		Citations:   citations,
		Raw:         resp,
	}

	return payload
}

func (s *SearchMCP) buildScrapePayload(url string, resp *domainsearch.FetchWebpageResponse) scrapeToolPayload {
	metadata := map[string]any{}
	if resp != nil && resp.Metadata != nil {
		metadata = resp.Metadata
	}

	cacheStatus := "live"
	if metadata != nil {
		if val, ok := metadata["fallback_mode"].(bool); ok && val {
			cacheStatus = "fallback"
		}
	}

	text := ""
	if resp != nil {
		text = resp.Text
		// Truncate full text if it exceeds the max limit
		if len([]rune(text)) > s.maxScrapeTextChars {
			text = truncateSnippet(text, s.maxScrapeTextChars)
		}
	}

	return scrapeToolPayload{
		SourceURL:   url,
		Text:        text,
		TextPreview: truncateSnippet(text, s.maxScrapePreviewChars),
		Metadata:    metadata,
		CacheStatus: cacheStatus,
		FetchedAt:   time.Now().UTC().Format(time.RFC3339),
	}
}

func estimateTokensFromSearchPayload(payload searchToolPayload) float64 {
	charCount := len(payload.Query)
	for _, result := range payload.Results {
		charCount += len(result.Title)
		charCount += len(result.Snippet)
		charCount += len(result.SourceURL)
	}
	for _, cite := range payload.Citations {
		charCount += len(cite)
	}
	return float64(charCount) / 4
}

func estimateTokensFromScrapePayload(payload scrapeToolPayload) float64 {
	return float64(len(payload.Text)+len(payload.TextPreview)) / 4
}

func (s *SearchMCP) buildImageSearchPayload(query string, resp *domainsearch.ImageSearchResponse) imageSearchToolPayload {
	metadata := map[string]any{}
	if resp != nil && resp.SearchParameters != nil {
		metadata = resp.SearchParameters
	}

	engine := "serper"
	if resp != nil && resp.SearchParameters != nil {
		if val, ok := resp.SearchParameters["engine"].(string); ok && val != "" {
			engine = val
		}
	}

	live := true
	if resp != nil && resp.SearchParameters != nil {
		if val, ok := resp.SearchParameters["live"].(bool); ok {
			live = val
		}
	}

	cacheStatus := "live"
	if !live {
		cacheStatus = "fallback"
	}

	results := make([]imageSearchToolResult, 0)
	if resp != nil {
		for _, img := range resp.Images {
			results = append(results, imageSearchToolResult{
				Position:        img.Position,
				Title:           img.Title,
				ImageURL:        img.ImageURL,
				ImageWidth:      img.ImageWidth,
				ImageHeight:     img.ImageHeight,
				ThumbnailURL:    img.ThumbnailURL,
				ThumbnailWidth:  img.ThumbnailWidth,
				ThumbnailHeight: img.ThumbnailHeight,
				Source:          img.Source,
				Domain:          img.Domain,
				Link:            img.Link,
				Creator:         img.Creator,
				Credit:          img.Credit,
			})
		}
	}

	return imageSearchToolPayload{
		Query:       query,
		Engine:      engine,
		Live:        live,
		CacheStatus: cacheStatus,
		Metadata:    metadata,
		Results:     results,
		Raw:         resp,
	}
}

func estimateTokensFromImageSearchPayload(payload imageSearchToolPayload) float64 {
	charCount := len(payload.Query)
	for _, result := range payload.Results {
		charCount += len(result.Title)
		charCount += len(result.ImageURL)
		charCount += len(result.ThumbnailURL)
		charCount += len(result.Source)
		charCount += len(result.Domain)
		charCount += len(result.Link)
	}
	return float64(charCount) / 4
}

func stringFromMap(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func truncateSnippet(text string, maxLen int) string {
	trimmed := strings.TrimSpace(text)
	runes := []rune(trimmed)
	if len(runes) <= maxLen {
		return trimmed
	}
	return string(runes[:maxLen]) + "…"
}

func extractMapArgument(args map[string]any, key string) map[string]any {
	if args == nil {
		return nil
	}
	raw, ok := args[key]
	if !ok {
		return nil
	}
	if cast, ok := raw.(map[string]any); ok {
		return cast
	}
	return nil
}

// filterSearchResults applies disallowed keyword filtering to search results
func (s *SearchMCP) filterSearchResults(ctx context.Context, toolKey string, payload searchToolPayload) searchToolPayload {
	toolConfig := s.getToolConfig(ctx, toolKey)
	if toolConfig == nil || len(toolConfig.CompiledFilters) == 0 {
		return payload
	}

	filteredResults := make([]searchToolResult, 0, len(payload.Results))
	filteredCitations := make([]string, 0, len(payload.Citations))
	removedCount := 0
	filteredURLs := make([]string, 0)

	for _, result := range payload.Results {
		// Check if any field matches disallowed keywords
		contentToCheck := result.Title + " " + result.Snippet + " " + result.SourceURL
		if toolConfig.MatchesDisallowedKeyword(contentToCheck) {
			removedCount++
			filteredURLs = append(filteredURLs, result.SourceURL)
			log.Info().
				Str("tool_key", toolKey).
				Str("source_url", result.SourceURL).
				Str("title", result.Title).
				Msg("Filtered search result due to disallowed keyword")
			continue
		}
		// Re-number the position
		result.Position = len(filteredResults) + 1
		filteredResults = append(filteredResults, result)
		if result.SourceURL != "" {
			filteredCitations = append(filteredCitations, result.SourceURL)
		}
	}

	if removedCount > 0 {
		log.Info().
			Str("tool_key", toolKey).
			Int("removed", removedCount).
			Int("remaining", len(filteredResults)).
			Strs("filtered_urls", filteredURLs).
			Msg("Filtered search results due to disallowed keywords")
	}

	payload.Results = filteredResults
	payload.Citations = filteredCitations
	return payload
}
