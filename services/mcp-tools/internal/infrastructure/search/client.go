package search

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	domainsearch "jan-server/services/mcp-tools/internal/domain/search"
	"jan-server/services/mcp-tools/internal/infrastructure/metrics"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"
)

const (
	serperSearchEndpoint         = "https://google.serper.dev/search"
	serperImageSearchEndpoint    = "https://google.serper.dev/images"
	serperScrapeEndpoint         = "https://scrape.serper.dev"
	exaSearchEndpointDefault     = "https://api.exa.ai/search"
	exaContentsEndpointDefault   = "https://api.exa.ai/contents"
	tavilySearchEndpointDefault  = "https://api.tavily.com/search"
	tavilyExtractEndpointDefault = "https://api.tavily.com/extract"
	searxngSearchPath            = "/search"
)

// Engine represents the configured backend for search operations.
type Engine string

const (
	// EngineSerper routes search requests to the hosted Serper API.
	EngineSerper Engine = "serper"
	// EngineExa routes search requests to the Exa API.
	EngineExa Engine = "exa"
	// EngineTavily routes search requests to the Tavily API.
	EngineTavily Engine = "tavily"
	// EngineSearxng routes search requests to a local SearXNG instance.
	EngineSearxng Engine = "searxng"
)

// ClientConfig captures the knobs exposed to operators for the search client.
type ClientConfig struct {
	Engine         Engine
	SerperAPIKey   string
	SerperEnabled  bool
	SearxngURL     string
	SearxngEnabled bool
	DomainFilters  []string
	LocationHint   string
	OfflineMode    bool

	ExaAPIKey   string
	ExaEnabled  bool
	ExaEndpoint string
	ExaTimeout  time.Duration

	TavilyAPIKey   string
	TavilyEnabled  bool
	TavilyEndpoint string
	TavilyTimeout  time.Duration

	// Circuit Breaker Settings
	CBEnabled          bool
	CBFailureThreshold int
	CBSuccessThreshold int
	CBTimeout          time.Duration
	CBMaxHalfOpen      int

	// HTTP Client Settings
	HTTPTimeout     time.Duration
	ScrapeTimeout   time.Duration // Separate timeout for scrape operations (typically longer)
	MaxConnsPerHost int
	MaxIdleConns    int
	IdleConnTimeout time.Duration

	// Retry Settings
	RetryMaxAttempts   int
	RetryInitialDelay  time.Duration
	RetryMaxDelay      time.Duration
	RetryBackoffFactor float64
}

// SearchClient implements domainsearch.SearchClient with pluggable backends.
type SearchClient struct {
	cfg            ClientConfig
	serperClient   *resty.Client
	exaClient      *resty.Client
	tavilyClient   *resty.Client
	scrapeClient   *resty.Client // Separate client for scrape with longer timeout
	fallbackClient *resty.Client
	searxClient    *resty.Client
	retryConfig    RetryConfig
	serperCB       *CircuitBreaker
	exaCB          *CircuitBreaker
	tavilyCB       *CircuitBreaker
	searxCB        *CircuitBreaker
}

var _ domainsearch.SearchClient = (*SearchClient)(nil)

// NewSearchClient wires HTTP clients for each supported backend.
func NewSearchClient(cfg ClientConfig) *SearchClient {
	engine := Engine(strings.ToLower(string(cfg.Engine)))
	if engine == "" {
		engine = EngineSerper
	}
	cfg.Engine = engine

	if strings.TrimSpace(cfg.ExaEndpoint) == "" {
		cfg.ExaEndpoint = exaSearchEndpointDefault
	}
	if strings.TrimSpace(cfg.TavilyEndpoint) == "" {
		cfg.TavilyEndpoint = tavilySearchEndpointDefault
	}

	// Set default HTTP timeout if not configured
	httpTimeout := 15 * time.Second
	if cfg.HTTPTimeout > 0 {
		httpTimeout = cfg.HTTPTimeout
	}

	// Configure HTTP transport with connection pooling
	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 100 // match Go default
	}
	maxConnsPerHost := cfg.MaxConnsPerHost
	if maxConnsPerHost == 0 {
		maxConnsPerHost = 50 // match Go default
	}
	idleConnTimeout := cfg.IdleConnTimeout
	if idleConnTimeout == 0 {
		idleConnTimeout = 90 * time.Second // match Go default
	}
	transport := &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: 20,
		MaxConnsPerHost:     maxConnsPerHost,
		IdleConnTimeout:     idleConnTimeout,
		DisableKeepAlives:   false,
		ForceAttemptHTTP2:   true,
	}

	serperHTTP := resty.New().
		SetHeader("User-Agent", "Jan-MCP-Tools/1.0").
		SetTimeout(httpTimeout).
		SetRetryCount(0).
		SetTransport(transport)

	exaTimeout := httpTimeout
	if cfg.ExaTimeout > 0 {
		exaTimeout = cfg.ExaTimeout
	}
	exaHTTP := resty.New().
		SetHeader("User-Agent", "Jan-MCP-Tools/1.0").
		SetTimeout(exaTimeout).
		SetRetryCount(0).
		SetTransport(transport)

	tavilyTimeout := httpTimeout
	if cfg.TavilyTimeout > 0 {
		tavilyTimeout = cfg.TavilyTimeout
	}
	tavilyHTTP := resty.New().
		SetHeader("User-Agent", "Jan-MCP-Tools/1.0").
		SetTimeout(tavilyTimeout).
		SetRetryCount(0).
		SetTransport(transport)

	// Scrape client with longer timeout (default 30s if not configured)
	scrapeTimeout := cfg.ScrapeTimeout
	if scrapeTimeout == 0 {
		scrapeTimeout = 30 * time.Second
	}
	scrapeHTTP := resty.New().
		SetHeader("User-Agent", "Jan-MCP-Tools/1.0").
		SetTimeout(scrapeTimeout).
		SetRetryCount(0).
		SetTransport(transport)

	// Fallback client with browser-like headers to avoid basic bot detection
	fallbackHTTP := resty.New().
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36").
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8").
		SetHeader("Accept-Language", "en-US,en;q=0.5").
		SetHeader("Accept-Encoding", "gzip, deflate").
		SetHeader("Connection", "keep-alive").
		SetHeader("Upgrade-Insecure-Requests", "1").
		SetTimeout(15 * time.Second).
		SetRetryCount(0)

	searxHTTP := resty.New().
		SetHeader("User-Agent", "Jan-MCP-Tools/1.0").
		SetTimeout(httpTimeout).
		SetRetryCount(0).
		SetTransport(transport)

	baseURL := strings.TrimSuffix(cfg.SearxngURL, "/")
	if baseURL != "" {
		searxHTTP.SetBaseURL(baseURL)
	}

	// Build retry config from ClientConfig
	retryConfig := DefaultRetryConfig()
	if cfg.RetryMaxAttempts > 0 {
		retryConfig.MaxAttempts = cfg.RetryMaxAttempts
	}
	if cfg.RetryInitialDelay > 0 {
		retryConfig.InitialDelay = cfg.RetryInitialDelay
	}
	if cfg.RetryBackoffFactor > 0 {
		retryConfig.BackoffFactor = cfg.RetryBackoffFactor
	}

	// Build circuit breaker config from ClientConfig
	cbConfig := DefaultCircuitBreakerConfig()
	cbConfig.Enabled = cfg.CBEnabled
	if cfg.CBFailureThreshold > 0 {
		cbConfig.FailureThreshold = cfg.CBFailureThreshold
	}
	if cfg.CBSuccessThreshold > 0 {
		cbConfig.SuccessThreshold = cfg.CBSuccessThreshold
	}
	if cfg.CBTimeout > 0 {
		cbConfig.Timeout = cfg.CBTimeout
	}
	if cfg.CBMaxHalfOpen > 0 {
		cbConfig.MaxHalfOpenCalls = cfg.CBMaxHalfOpen
	}

	return &SearchClient{
		cfg:            cfg,
		serperClient:   serperHTTP,
		exaClient:      exaHTTP,
		tavilyClient:   tavilyHTTP,
		scrapeClient:   scrapeHTTP,
		fallbackClient: fallbackHTTP,
		searxClient:    searxHTTP,
		retryConfig:    retryConfig,
		serperCB:       NewCircuitBreaker(cbConfig),
		exaCB:          NewCircuitBreaker(cbConfig),
		tavilyCB:       NewCircuitBreaker(cbConfig),
		searxCB:        NewCircuitBreaker(cbConfig),
	}
}

// Search fans out to the configured backend while preserving offline + fallback behaviour.
func (c *SearchClient) Search(ctx context.Context, query domainsearch.SearchRequest) (*domainsearch.SearchResponse, error) {
	query = c.enrichQuery(query)
	offline := c.resolveOfflineMode(query.OfflineMode)

	log.Debug().
		Str("operation", "search").
		Str("query", query.Q).
		Bool("offline_mode", offline).
		Bool("serper_enabled", c.cfg.SerperEnabled && c.hasSerperAPIKey()).
		Bool("exa_enabled", c.cfg.ExaEnabled && c.hasExaAPIKey()).
		Bool("tavily_enabled", c.cfg.TavilyEnabled && c.hasTavilyAPIKey()).
		Bool("searxng_enabled", c.cfg.SearxngEnabled && c.hasSearxngURL()).
		Msg("search client starting provider chain")

	if offline {
		return nil, fmt.Errorf("search unavailable: offline mode is enabled")
	}

	var lastErr error
	providersTried := make([]string, 0, 4)

	if c.cfg.SerperEnabled && c.hasSerperAPIKey() {
		providersTried = append(providersTried, "serper")
		log.Debug().Str("provider", "serper").Str("query", query.Q).Msg("trying search provider")
		if res, err := c.searchViaSerper(ctx, query); err == nil {
			log.Info().Str("engine", "serper").Str("query", query.Q).Int("result_count", len(res.Organic)).Msg("search completed using engine")
			return res, nil
		} else {
			lastErr = err
			log.Warn().Err(err).Msg("Serper search failed, trying next provider")
		}
	}
	if !c.cfg.SerperEnabled || !c.hasSerperAPIKey() {
		log.Debug().Bool("enabled", c.cfg.SerperEnabled).Bool("has_key", c.hasSerperAPIKey()).Msg("Skipping Serper search provider")
	}

	if c.cfg.TavilyEnabled && c.hasTavilyAPIKey() {
		providersTried = append(providersTried, "tavily")
		log.Debug().Str("provider", "tavily").Str("query", query.Q).Msg("trying search provider")
		if res, err := c.searchViaTavily(ctx, query); err == nil {
			log.Info().Str("engine", "tavily").Str("query", query.Q).Int("result_count", len(res.Organic)).Msg("search completed using engine")
			return res, nil
		} else {
			lastErr = err
			log.Warn().Err(err).Msg("Tavily search failed, trying next provider")
		}
	}
	if !c.cfg.TavilyEnabled || !c.hasTavilyAPIKey() {
		log.Debug().Bool("enabled", c.cfg.TavilyEnabled).Bool("has_key", c.hasTavilyAPIKey()).Msg("Skipping Tavily search provider")
	}

	if c.cfg.ExaEnabled && c.hasExaAPIKey() {
		providersTried = append(providersTried, "exa")
		log.Debug().Str("provider", "exa").Str("query", query.Q).Msg("trying search provider")
		if res, err := c.searchViaExa(ctx, query); err == nil {
			log.Info().Str("engine", "exa").Str("query", query.Q).Int("result_count", len(res.Organic)).Msg("search completed using engine")
			return res, nil
		} else {
			lastErr = err
			log.Warn().Err(err).Msg("Exa search failed, trying next provider")
		}
	}
	if !c.cfg.ExaEnabled || !c.hasExaAPIKey() {
		log.Debug().Bool("enabled", c.cfg.ExaEnabled).Bool("has_key", c.hasExaAPIKey()).Msg("Skipping Exa search provider")
	}

	if c.cfg.SearxngEnabled && c.hasSearxngURL() {
		providersTried = append(providersTried, "searxng")
		log.Debug().Str("provider", "searxng").Str("query", query.Q).Msg("trying search provider")
		if res, err := c.searchViaSearxng(ctx, query); err == nil {
			log.Info().Str("engine", "searxng").Str("query", query.Q).Int("result_count", len(res.Organic)).Msg("search completed using engine")
			return res, nil
		} else {
			lastErr = err
			log.Warn().Err(err).Msg("SearXNG search failed")
		}
	}
	if !c.cfg.SearxngEnabled || !c.hasSearxngURL() {
		log.Debug().Bool("enabled", c.cfg.SearxngEnabled).Bool("has_url", c.hasSearxngURL()).Msg("Skipping SearXNG search provider")
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all search providers failed (tried: %v): %w", strings.Join(providersTried, ", "), lastErr)
	}
	return nil, fmt.Errorf("search unavailable: no providers enabled")
}

// FetchWebpage scrapes a webpage either via Serper's scrape API or a fallback HTTP fetcher.
// Returns a response with status indicating success/failure - graceful degradation instead of errors.
func (c *SearchClient) FetchWebpage(ctx context.Context, query domainsearch.FetchWebpageRequest) (*domainsearch.FetchWebpageResponse, error) {
	offline := c.resolveOfflineMode(query.OfflineMode)

	log.Debug().
		Str("operation", "scrape").
		Str("url", query.Url).
		Bool("offline_mode", offline).
		Bool("serper_enabled", c.cfg.SerperEnabled && c.hasSerperAPIKey()).
		Bool("exa_enabled", c.cfg.ExaEnabled && c.hasExaAPIKey()).
		Bool("tavily_enabled", c.cfg.TavilyEnabled && c.hasTavilyAPIKey()).
		Msg("scrape client starting provider chain")

	if offline {
		return nil, fmt.Errorf("scrape unavailable: offline mode is enabled")
	}

	var lastErr error
	providersTried := make([]string, 0, 4)

	if c.cfg.SerperEnabled && c.hasSerperAPIKey() {
		providersTried = append(providersTried, "serper")
		log.Debug().Str("provider", "serper").Str("url", query.Url).Msg("trying scrape provider")
		if res, err := c.fetchViaSerper(ctx, query); err == nil {
			log.Info().Str("engine", "serper").Str("url", query.Url).Int("text_length", len(res.Text)).Msg("scrape completed using engine")
			res.Status = "success"
			return res, nil
		} else {
			lastErr = err
			log.Debug().Err(err).Str("provider", "serper").Msg("scrape provider failed, trying next")
		}
	}

	if c.cfg.ExaEnabled && c.hasExaAPIKey() {
		providersTried = append(providersTried, "exa")
		log.Debug().Str("provider", "exa").Str("url", query.Url).Msg("trying scrape provider")
		if res, err := c.fetchViaExa(ctx, query); err == nil {
			log.Info().Str("engine", "exa").Str("url", query.Url).Int("text_length", len(res.Text)).Msg("scrape completed using engine")
			res.Status = "success"
			return res, nil
		} else {
			lastErr = err
			log.Debug().Err(err).Str("provider", "exa").Msg("scrape provider failed, trying next")
		}
	}

	if c.cfg.TavilyEnabled && c.hasTavilyAPIKey() {
		providersTried = append(providersTried, "tavily")
		log.Debug().Str("provider", "tavily").Str("url", query.Url).Msg("trying scrape provider")
		if res, err := c.fetchViaTavily(ctx, query); err == nil {
			log.Info().Str("engine", "tavily").Str("url", query.Url).Int("text_length", len(res.Text)).Msg("scrape completed using engine")
			res.Status = "success"
			return res, nil
		} else {
			lastErr = err
			log.Debug().Err(err).Str("provider", "tavily").Msg("scrape provider failed, trying next")
		}
	}

	providersTried = append(providersTried, "direct-http")
	log.Debug().Str("provider", "direct-http").Str("url", query.Url).Msg("trying scrape provider")
	if res, err := c.fetchFallback(ctx, query); err == nil {
		log.Info().Str("engine", "direct-http").Str("url", query.Url).Int("text_length", len(res.Text)).Msg("scrape completed using engine")
		res.Status = "success"
		return res, nil
	} else {
		lastErr = err
		log.Debug().Err(err).Str("provider", "direct-http").Msg("scrape provider failed")
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all scrape methods failed (tried: %v): %w", strings.Join(providersTried, ", "), lastErr)
	}
	return nil, fmt.Errorf("scrape unavailable: no providers enabled")
}

func (c *SearchClient) enrichQuery(query domainsearch.SearchRequest) domainsearch.SearchRequest {
	mergedDomains := c.mergeDomains(query.DomainAllowList)
	if len(mergedDomains) > 0 {
		query.DomainAllowList = mergedDomains
		query.Q = applyDomainFilter(query.Q, mergedDomains)
	}

	if query.LocationHint == nil && strings.TrimSpace(c.cfg.LocationHint) != "" {
		hint := c.cfg.LocationHint
		query.LocationHint = &hint
	}

	return query
}

func (c *SearchClient) mergeDomains(custom []string) []string {
	var merged []string
	seen := map[string]struct{}{}

	appendDomain := func(values []string) {
		for _, val := range values {
			domain := sanitizeDomain(val)
			if domain == "" {
				continue
			}
			if _, exists := seen[domain]; exists {
				continue
			}
			seen[domain] = struct{}{}
			merged = append(merged, domain)
		}
	}

	appendDomain(c.cfg.DomainFilters)
	appendDomain(custom)

	return merged
}

func (c *SearchClient) resolveOfflineMode(override *bool) bool {
	if override != nil {
		return *override
	}
	return c.cfg.OfflineMode
}

func (c *SearchClient) searchViaSerper(ctx context.Context, query domainsearch.SearchRequest) (*domainsearch.SearchResponse, error) {
	// Check circuit breaker
	if c.serperCB.GetState() == StateOpen {
		log.Error().Str("service", "serper").Msg("serper circuit breaker is open, skipping")
		return nil, fmt.Errorf("serper circuit breaker is open")
	}

	startTime := time.Now()
	status := "success"
	defer func() {
		metrics.RecordProviderRequest("search", "serper", status)
		metrics.RecordExternalProviderLatency("serper", time.Since(startTime).Seconds())
	}()

	body := map[string]any{
		"q": query.Q,
	}
	if query.GL != nil {
		body["gl"] = *query.GL
	}
	if query.HL != nil {
		body["hl"] = *query.HL
	}
	if query.Location != nil {
		body["location"] = *query.Location
	} else if query.LocationHint != nil {
		body["location"] = *query.LocationHint
	}
	if query.Num != nil {
		body["num"] = *query.Num
	}
	if query.Page != nil {
		body["page"] = *query.Page
	}
	if query.Autocorrect != nil {
		body["autocorrect"] = *query.Autocorrect
	}
	if query.TBS != nil {
		body["tbs"] = string(*query.TBS)
	}

	var result *domainsearch.SearchResponse
	var opErr error

	// Retry with exponential backoff
	resultPtr, err := WithRetry(ctx, c.retryConfig, "serper_search", func() (*domainsearch.SearchResponse, error) {
		var res domainsearch.SearchResponse
		resp, err := c.serperClient.R().
			SetContext(ctx).
			SetHeader("X-API-KEY", c.cfg.SerperAPIKey).
			SetHeader("Content-Type", "application/json").
			SetBody(body).
			SetResult(&res).
			Post(serperSearchEndpoint)

		if err != nil {
			log.Error().Err(err).Str("service", "serper").Str("endpoint", serperSearchEndpoint).Msg("failed to query Serper search API")
			return nil, fmt.Errorf("failed to query Serper search API: %w", err)
		}

		if resp.IsError() {
			log.Error().Int("status", resp.StatusCode()).Str("service", "serper").Str("response", resp.String()).Msg("Serper search API error")
			return nil, fmt.Errorf("Serper search API error (status %d): %s", resp.StatusCode(), resp.String())
		}

		return &res, nil
	})

	opErr = err
	if opErr == nil {
		result = resultPtr
		// Validate response
		if validationErr := ValidateSearchResponse(result, 0); validationErr != nil {
			log.Warn().Err(validationErr).Msg("serper search returned invalid response")
			opErr = fmt.Errorf("serper search invalid response: %w", validationErr)
		}
	}

	// Update circuit breaker
	c.serperCB.recordResult("serper_search", opErr)

	if opErr != nil {
		status = "error"
		log.Error().Err(opErr).Str("service", "serper").Str("operation", "search").Msg("serper search failed after retries")
		return nil, opErr
	}

	if result.SearchParameters == nil {
		result.SearchParameters = map[string]any{}
	}
	result.SearchParameters["engine"] = "serper"
	result.SearchParameters["live"] = true
	result.SearchParameters["domain_allow_list"] = query.DomainAllowList
	if query.LocationHint != nil {
		result.SearchParameters["location_hint"] = *query.LocationHint
	}

	return result, nil
}

func (c *SearchClient) searchViaExa(ctx context.Context, query domainsearch.SearchRequest) (*domainsearch.SearchResponse, error) {
	if c.exaCB.GetState() == StateOpen {
		log.Error().Str("service", "exa").Msg("exa circuit breaker is open, skipping")
		return nil, fmt.Errorf("exa circuit breaker is open")
	}

	startTime := time.Now()
	status := "success"
	defer func() {
		metrics.RecordProviderRequest("search", "exa", status)
		metrics.RecordExternalProviderLatency("exa", time.Since(startTime).Seconds())
	}()

	numResults := 10
	if query.Num != nil && *query.Num > 0 {
		numResults = *query.Num
	}

	log.Info().
		Str("service", "exa").
		Str("operation", "search").
		Str("endpoint", c.cfg.ExaEndpoint).
		Str("query", query.Q).
		Int("num_results", numResults).
		Int("domain_allow_list_count", len(query.DomainAllowList)).
		Msg("Exa search request")

	body := domainsearch.ExaSearchRequest{
		Query:          query.Q,
		NumResults:     numResults,
		IncludeDomains: query.DomainAllowList,
		UseAutoprompt:  true,
		Type:           "neural",
		Contents: &domainsearch.ExaContents{
			Text:          true,
			Highlights:    true,
			MaxCharacters: 400,
		},
	}

	var opErr error
	resultPtr, err := WithRetry(ctx, c.retryConfig, "exa_search", func() (*exaSearchResponse, error) {
		var res exaSearchResponse
		resp, err := c.exaClient.R().
			SetContext(ctx).
			SetHeader("Authorization", "Bearer "+c.cfg.ExaAPIKey).
			SetHeader("Content-Type", "application/json").
			SetBody(body).
			SetResult(&res).
			Post(c.cfg.ExaEndpoint)

		if err != nil {
			log.Error().Err(err).Str("service", "exa").Str("endpoint", c.cfg.ExaEndpoint).Msg("failed to query Exa search API")
			return nil, fmt.Errorf("failed to query Exa search API: %w", err)
		}
		if resp.IsError() {
			log.Error().Int("status", resp.StatusCode()).Str("service", "exa").Str("response", resp.String()).Msg("Exa search API error")
			return nil, fmt.Errorf("Exa search API error (status %d): %s", resp.StatusCode(), resp.String())
		}

		return &res, nil
	})

	opErr = err
	if opErr == nil {
		searchResp := &domainsearch.SearchResponse{
			SearchParameters: map[string]any{
				"engine":            "exa",
				"q":                 query.Q,
				"live":              true,
				"domain_allow_list": query.DomainAllowList,
			},
			Organic: make([]map[string]any, 0, len(resultPtr.Results)),
		}
		if query.LocationHint != nil {
			searchResp.SearchParameters["location_hint"] = *query.LocationHint
		}

		for _, item := range resultPtr.Results {
			snippet := firstNonEmpty(item.Text, item.Summary, strings.Join(item.Highlights, " "))
			searchResp.Organic = append(searchResp.Organic, map[string]any{
				"title":          item.Title,
				"link":           item.URL,
				"snippet":        snippet,
				"source":         "exa",
				"published_date": item.PublishedDate,
				"author":         item.Author,
				"score":          item.Score,
			})
		}

		log.Info().
			Str("service", "exa").
			Str("operation", "search").
			Int("result_count", len(searchResp.Organic)).
			Msg("Exa search response received")

		if validationErr := ValidateSearchResponse(searchResp, 0); validationErr != nil {
			log.Warn().Err(validationErr).Msg("exa search returned invalid response")
			opErr = fmt.Errorf("exa search invalid response: %w", validationErr)
		} else {
			c.exaCB.recordResult("exa_search", nil)
			return searchResp, nil
		}
	}

	c.exaCB.recordResult("exa_search", opErr)
	if opErr != nil {
		status = "error"
		log.Error().Err(opErr).Str("service", "exa").Str("operation", "search").Msg("exa search failed after retries")
		return nil, opErr
	}

	return nil, fmt.Errorf("exa search failed")
}

func (c *SearchClient) searchViaTavily(ctx context.Context, query domainsearch.SearchRequest) (*domainsearch.SearchResponse, error) {
	if c.tavilyCB.GetState() == StateOpen {
		log.Error().Str("service", "tavily").Msg("tavily circuit breaker is open, skipping")
		return nil, fmt.Errorf("tavily circuit breaker is open")
	}

	startTime := time.Now()
	status := "success"
	defer func() {
		metrics.RecordProviderRequest("search", "tavily", status)
		metrics.RecordExternalProviderLatency("tavily", time.Since(startTime).Seconds())
	}()

	maxResults := 10
	if query.Num != nil && *query.Num > 0 {
		maxResults = *query.Num
	}

	body := domainsearch.TavilySearchRequest{
		Query:             query.Q,
		SearchDepth:       "basic",
		MaxResults:        maxResults,
		IncludeDomains:    query.DomainAllowList,
		IncludeAnswer:     false,
		IncludeRawContent: false,
	}

	var opErr error
	resultPtr, err := WithRetry(ctx, c.retryConfig, "tavily_search", func() (*tavilySearchResponse, error) {
		var res tavilySearchResponse
		resp, err := c.tavilyClient.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetBody(map[string]any{
				"api_key":             c.cfg.TavilyAPIKey,
				"query":               body.Query,
				"search_depth":        body.SearchDepth,
				"max_results":         body.MaxResults,
				"include_domains":     body.IncludeDomains,
				"exclude_domains":     body.ExcludeDomains,
				"include_answer":      body.IncludeAnswer,
				"include_raw_content": body.IncludeRawContent,
			}).
			SetResult(&res).
			Post(c.cfg.TavilyEndpoint)

		if err != nil {
			log.Error().Err(err).Str("service", "tavily").Str("endpoint", c.cfg.TavilyEndpoint).Msg("failed to query Tavily search API")
			return nil, fmt.Errorf("failed to query Tavily search API: %w", err)
		}
		if resp.IsError() {
			log.Error().Int("status", resp.StatusCode()).Str("service", "tavily").Str("response", resp.String()).Msg("Tavily search API error")
			return nil, fmt.Errorf("Tavily search API error (status %d): %s", resp.StatusCode(), resp.String())
		}

		return &res, nil
	})

	opErr = err
	if opErr == nil {
		searchResp := &domainsearch.SearchResponse{
			SearchParameters: map[string]any{
				"engine":            "tavily",
				"q":                 query.Q,
				"live":              true,
				"domain_allow_list": query.DomainAllowList,
			},
			Organic: make([]map[string]any, 0, len(resultPtr.Results)),
		}
		if query.LocationHint != nil {
			searchResp.SearchParameters["location_hint"] = *query.LocationHint
		}

		for _, item := range resultPtr.Results {
			snippet := firstNonEmpty(item.Content, item.RawContent)
			searchResp.Organic = append(searchResp.Organic, map[string]any{
				"title":          item.Title,
				"link":           item.URL,
				"snippet":        snippet,
				"source":         "tavily",
				"published_date": item.PublishedDate,
				"score":          item.Score,
			})
		}

		if validationErr := ValidateSearchResponse(searchResp, 0); validationErr != nil {
			log.Warn().Err(validationErr).Msg("tavily search returned invalid response")
			opErr = fmt.Errorf("tavily search invalid response: %w", validationErr)
		} else {
			c.tavilyCB.recordResult("tavily_search", nil)
			return searchResp, nil
		}
	}

	c.tavilyCB.recordResult("tavily_search", opErr)
	if opErr != nil {
		status = "error"
		log.Error().Err(opErr).Str("service", "tavily").Str("operation", "search").Msg("tavily search failed after retries")
		return nil, opErr
	}

	return nil, fmt.Errorf("tavily search failed")
}

func (c *SearchClient) searchViaSearxng(ctx context.Context, query domainsearch.SearchRequest) (*domainsearch.SearchResponse, error) {
	if c.searxClient == nil {
		return nil, fmt.Errorf("searxng client not configured")
	}

	// Check circuit breaker
	if c.searxCB.GetState() == StateOpen {
		log.Error().Str("service", "searxng").Msg("searxng circuit breaker is open, skipping")
		return nil, fmt.Errorf("searxng circuit breaker is open")
	}

	startTime := time.Now()
	status := "success"
	defer func() {
		metrics.RecordProviderRequest("search", "searxng", status)
		metrics.RecordExternalProviderLatency("searxng", time.Since(startTime).Seconds())
	}()

	var opErr error

	// Retry with exponential backoff
	resultPtr, err := WithRetry(ctx, c.retryConfig, "searxng_search", func() (*searxngResponse, error) {
		req := c.searxClient.R().
			SetContext(ctx).
			SetQueryParam("q", query.Q).
			SetQueryParam("format", "json").
			SetQueryParam("safesearch", "1")

		if query.HL != nil {
			req.SetQueryParam("language", *query.HL)
		}
		if query.Page != nil && *query.Page > 1 {
			req.SetQueryParam("p", strconv.Itoa(*query.Page))
		}
		if query.Num != nil && *query.Num > 0 {
			req.SetQueryParam("num", strconv.Itoa(*query.Num))
		}
		if query.TBS != nil {
			if mapped := mapTBSToSearxng(*query.TBS); mapped != "" {
				req.SetQueryParam("time_range", mapped)
			}
		}

		var result searxngResponse
		resp, err := req.SetResult(&result).Get(searxngSearchPath)
		if err != nil {
			log.Error().Err(err).Str("service", "searxng").Str("url", c.cfg.SearxngURL).Msg("failed to query SearXNG API")
			return nil, fmt.Errorf("failed to query SearXNG API: %w", err)
		}
		if resp.IsError() {
			log.Error().Int("status", resp.StatusCode()).Str("service", "searxng").Str("response", resp.String()).Msg("SearXNG API error")
			return nil, fmt.Errorf("SearXNG API error (status %d): %s", resp.StatusCode(), resp.String())
		}

		return &result, nil
	})

	opErr = err
	if opErr != nil {
		c.searxCB.recordResult("searxng_search", opErr)
		log.Error().Err(opErr).Str("service", "searxng").Str("operation", "search").Msg("searxng search failed after retries")
		return nil, opErr
	}

	result := *resultPtr

	limit := 10
	if query.Num != nil && *query.Num > 0 {
		limit = *query.Num
	}

	results := make([]map[string]any, 0, len(result.Results))
	for idx, item := range result.Results {
		if idx >= limit {
			break
		}
		results = append(results, map[string]any{
			"title":       item.Title,
			"link":        item.URL,
			"description": strings.TrimSpace(item.Content),
			"source":      "searxng",
			"engine":      item.Engine,
		})
	}

	searchMetadata := map[string]any{
		"engine":            "searxng",
		"q":                 query.Q,
		"live":              true,
		"domain_allow_list": query.DomainAllowList,
	}
	if query.LocationHint != nil {
		searchMetadata["location_hint"] = *query.LocationHint
	}

	searchResp := &domainsearch.SearchResponse{
		SearchParameters: searchMetadata,
		Organic:          results,
	}

	// Validate response
	if validationErr := ValidateSearchResponse(searchResp, 0); validationErr != nil {
		log.Warn().Err(validationErr).Msg("searxng search returned invalid response")
		opErr = fmt.Errorf("searxng search invalid response: %w", validationErr)
	}

	// Update circuit breaker
	c.searxCB.recordResult("searxng_search", opErr)

	if opErr != nil {
		status = "error"
		log.Error().Err(opErr).Str("service", "searxng").Str("operation", "search").Msg("searxng search failed after retries")
		return nil, opErr
	}

	return searchResp, nil
}

func mapTBSToSearxng(t domainsearch.TBSTimeRange) string {
	switch t {
	case domainsearch.TBSPastHour:
		return "day"
	case domainsearch.TBSPastDay:
		return "day"
	case domainsearch.TBSPastWeek:
		return "week"
	case domainsearch.TBSPastMonth:
		return "month"
	case domainsearch.TBSPastYear:
		return "year"
	default:
		return ""
	}
}

func (c *SearchClient) fetchViaSerper(ctx context.Context, query domainsearch.FetchWebpageRequest) (*domainsearch.FetchWebpageResponse, error) {
	// Check circuit breaker
	if c.serperCB.GetState() == StateOpen {
		log.Error().Str("service", "serper").Str("operation", "scrape").Msg("serper circuit breaker is open for scraping, using fallback")
		return nil, fmt.Errorf("serper circuit breaker is open")
	}

	startTime := time.Now()
	status := "success"
	defer func() {
		metrics.RecordProviderRequest("scrape", "serper", status)
		metrics.RecordExternalProviderLatency("serper", time.Since(startTime).Seconds())
	}()

	body := map[string]any{
		"url": query.Url,
	}
	if query.IncludeMarkdown != nil {
		body["includeMarkdown"] = *query.IncludeMarkdown
	}

	var opErr error

	// Retry with exponential backoff - use dedicated scrape client with longer timeout
	result, err := WithRetry(ctx, c.retryConfig, "serper_scrape", func() (*domainsearch.FetchWebpageResponse, error) {
		var res domainsearch.FetchWebpageResponse
		resp, err := c.scrapeClient.R().
			SetContext(ctx).
			SetHeader("X-API-KEY", c.cfg.SerperAPIKey).
			SetHeader("Content-Type", "application/json").
			SetBody(body).
			SetResult(&res).
			Post(serperScrapeEndpoint)

		if err != nil {
			log.Error().Err(err).Str("service", "serper").Str("endpoint", serperScrapeEndpoint).Str("url", query.Url).Msg("failed to query Serper scrape API")
			return nil, fmt.Errorf("failed to query Serper scrape API: %w", err)
		}

		if resp.IsError() {
			log.Error().Int("status", resp.StatusCode()).Str("service", "serper").Str("url", query.Url).Str("response", resp.String()).Msg("Serper scrape API error")
			return nil, fmt.Errorf("Serper scrape API error (status %d): %s", resp.StatusCode(), resp.String())
		}

		return &res, nil
	})

	opErr = err
	if opErr == nil {
		// Validate response (minimum 50 chars for meaningful content)
		if validationErr := ValidateFetchResponse(result, 50); validationErr != nil {
			log.Warn().Err(validationErr).Msg("serper scrape returned invalid response")
			opErr = fmt.Errorf("serper scrape invalid response: %w", validationErr)
		}
	}

	// Update circuit breaker
	c.serperCB.recordResult("serper_scrape", opErr)

	if opErr != nil {
		status = "error"
		log.Error().Err(opErr).Str("service", "serper").Str("operation", "scrape").Str("url", query.Url).Msg("serper scrape failed after retries")
		return nil, opErr
	}

	return result, nil
}

func (c *SearchClient) fetchViaExa(ctx context.Context, query domainsearch.FetchWebpageRequest) (*domainsearch.FetchWebpageResponse, error) {
	if c.exaCB.GetState() == StateOpen {
		log.Error().Str("service", "exa").Str("operation", "scrape").Msg("exa circuit breaker is open for scraping")
		return nil, fmt.Errorf("exa circuit breaker is open")
	}

	startTime := time.Now()
	status := "success"
	defer func() {
		metrics.RecordProviderRequest("scrape", "exa", status)
		metrics.RecordExternalProviderLatency("exa", time.Since(startTime).Seconds())
	}()

	log.Info().
		Str("service", "exa").
		Str("operation", "scrape").
		Str("endpoint", c.exaContentsEndpoint()).
		Str("url", query.Url).
		Msg("Exa scrape request")

	body := map[string]any{
		"ids":  []string{query.Url},
		"text": true,
	}

	var opErr error
	resultPtr, err := WithRetry(ctx, c.retryConfig, "exa_contents", func() (*exaContentsResponse, error) {
		var res exaContentsResponse
		resp, err := c.exaClient.R().
			SetContext(ctx).
			SetHeader("Authorization", "Bearer "+c.cfg.ExaAPIKey).
			SetHeader("Content-Type", "application/json").
			SetBody(body).
			SetResult(&res).
			Post(c.exaContentsEndpoint())

		if err != nil {
			log.Error().Err(err).Str("service", "exa").Str("endpoint", c.exaContentsEndpoint()).Str("url", query.Url).Msg("failed to query Exa contents API")
			return nil, fmt.Errorf("failed to query Exa contents API: %w", err)
		}
		if resp.IsError() {
			log.Error().Int("status", resp.StatusCode()).Str("service", "exa").Str("url", query.Url).Str("response", resp.String()).Msg("Exa contents API error")
			return nil, fmt.Errorf("Exa contents API error (status %d): %s", resp.StatusCode(), resp.String())
		}

		return &res, nil
	})

	opErr = err
	if opErr == nil {
		text := ""
		if len(resultPtr.Results) > 0 {
			text = strings.TrimSpace(resultPtr.Results[0].Text)
		}
		resp := &domainsearch.FetchWebpageResponse{
			Text: text,
			Metadata: map[string]any{
				"source":   query.Url,
				"provider": "exa",
			},
		}

		log.Info().
			Str("service", "exa").
			Str("operation", "scrape").
			Int("text_length", len(resp.Text)).
			Msg("Exa scrape response received")

		if validationErr := ValidateFetchResponse(resp, 50); validationErr != nil {
			log.Warn().Err(validationErr).Msg("exa scrape returned invalid response")
			opErr = fmt.Errorf("exa scrape invalid response: %w", validationErr)
		} else {
			c.exaCB.recordResult("exa_contents", nil)
			return resp, nil
		}
	}

	c.exaCB.recordResult("exa_contents", opErr)
	if opErr != nil {
		status = "error"
		log.Error().Err(opErr).Str("service", "exa").Str("operation", "scrape").Str("url", query.Url).Msg("exa scrape failed after retries")
		return nil, opErr
	}

	return nil, fmt.Errorf("exa scrape failed")
}

func (c *SearchClient) fetchViaTavily(ctx context.Context, query domainsearch.FetchWebpageRequest) (*domainsearch.FetchWebpageResponse, error) {
	if c.tavilyCB.GetState() == StateOpen {
		log.Error().Str("service", "tavily").Str("operation", "scrape").Msg("tavily circuit breaker is open for scraping")
		return nil, fmt.Errorf("tavily circuit breaker is open")
	}

	startTime := time.Now()
	status := "success"
	defer func() {
		metrics.RecordProviderRequest("scrape", "tavily", status)
		metrics.RecordExternalProviderLatency("tavily", time.Since(startTime).Seconds())
	}()

	body := map[string]any{
		"api_key": c.cfg.TavilyAPIKey,
		"urls":    []string{query.Url},
	}

	var opErr error
	resultPtr, err := WithRetry(ctx, c.retryConfig, "tavily_extract", func() (*tavilyExtractResponse, error) {
		var res tavilyExtractResponse
		resp, err := c.tavilyClient.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetBody(body).
			SetResult(&res).
			Post(c.tavilyExtractEndpoint())

		if err != nil {
			log.Error().Err(err).Str("service", "tavily").Str("endpoint", c.tavilyExtractEndpoint()).Str("url", query.Url).Msg("failed to query Tavily extract API")
			return nil, fmt.Errorf("failed to query Tavily extract API: %w", err)
		}
		if resp.IsError() {
			log.Error().Int("status", resp.StatusCode()).Str("service", "tavily").Str("url", query.Url).Str("response", resp.String()).Msg("Tavily extract API error")
			return nil, fmt.Errorf("Tavily extract API error (status %d): %s", resp.StatusCode(), resp.String())
		}

		return &res, nil
	})

	opErr = err
	if opErr == nil {
		text := ""
		if len(resultPtr.Results) > 0 {
			text = firstNonEmpty(resultPtr.Results[0].RawContent, resultPtr.Results[0].Content)
		}
		resp := &domainsearch.FetchWebpageResponse{
			Text: text,
			Metadata: map[string]any{
				"source":   query.Url,
				"provider": "tavily",
			},
		}

		if validationErr := ValidateFetchResponse(resp, 50); validationErr != nil {
			log.Warn().Err(validationErr).Msg("tavily scrape returned invalid response")
			opErr = fmt.Errorf("tavily scrape invalid response: %w", validationErr)
		} else {
			c.tavilyCB.recordResult("tavily_extract", nil)
			return resp, nil
		}
	}

	c.tavilyCB.recordResult("tavily_extract", opErr)
	if opErr != nil {
		status = "error"
		log.Error().Err(opErr).Str("service", "tavily").Str("operation", "scrape").Str("url", query.Url).Msg("tavily scrape failed after retries")
		return nil, opErr
	}

	return nil, fmt.Errorf("tavily scrape failed")
}

func (c *SearchClient) fetchFallback(ctx context.Context, query domainsearch.FetchWebpageRequest) (*domainsearch.FetchWebpageResponse, error) {
	// Retry fallback fetch with shorter retry config
	shortRetry := c.retryConfig
	shortRetry.MaxAttempts = 2

	startTime := time.Now()
	status := "success"
	defer func() {
		metrics.RecordProviderRequest("scrape", "direct-http", status)
		metrics.RecordExternalProviderLatency("direct-http", time.Since(startTime).Seconds())
	}()

	result, err := WithRetry(ctx, shortRetry, "fallback_fetch", func() (*domainsearch.FetchWebpageResponse, error) {
		resp, err := c.fallbackClient.R().
			SetContext(ctx).
			SetHeader("User-Agent", "Jan-MCP-Tools-Fallback/1.0").
			Get(query.Url)
		if err != nil {
			log.Error().Err(err).Str("service", "fallback").Str("url", query.Url).Msg("fallback fetch failed")
			return nil, fmt.Errorf("fallback fetch failed: %w", err)
		}
		if resp.IsError() {
			log.Error().Int("status", resp.StatusCode()).Str("service", "fallback").Str("url", query.Url).Msg("fallback fetch HTTP error")
			return nil, fmt.Errorf("fallback fetch HTTP %d: %s", resp.StatusCode(), resp.Status())
		}

		bodyBytes := resp.Body()
		text := extractVisibleText(bodyBytes)
		if text == "" {
			text = string(bodyBytes)
		}

		metadata := map[string]any{
			"source":        query.Url,
			"contentType":   resp.Header().Get("Content-Type"),
			"fallback_mode": true,
		}

		return &domainsearch.FetchWebpageResponse{
			Text:     text,
			Metadata: metadata,
		}, nil
	})

	if err != nil {
		status = "error"
		log.Error().Err(err).Str("service", "fallback").Str("operation", "fetch").Str("url", query.Url).Msg("fallback fetch failed after retries")
		return nil, err
	}

	// Validate response
	if validationErr := ValidateFetchResponse(result, 50); validationErr != nil {
		log.Warn().Err(validationErr).Msg("fallback fetch returned invalid response")
		status = "error"
		return nil, fmt.Errorf("fallback fetch invalid response: %w", validationErr)
	}

	return result, nil
}

func (c *SearchClient) hasSerperAPIKey() bool {
	return strings.TrimSpace(c.cfg.SerperAPIKey) != ""
}

func (c *SearchClient) hasExaAPIKey() bool {
	return strings.TrimSpace(c.cfg.ExaAPIKey) != ""
}

func (c *SearchClient) hasTavilyAPIKey() bool {
	return strings.TrimSpace(c.cfg.TavilyAPIKey) != ""
}

func (c *SearchClient) hasSearxngURL() bool {
	return strings.TrimSpace(c.cfg.SearxngURL) != ""
}

func (c *SearchClient) exaContentsEndpoint() string {
	endpoint := strings.TrimSpace(c.cfg.ExaEndpoint)
	if endpoint == "" {
		return exaContentsEndpointDefault
	}
	if strings.Contains(endpoint, "/search") {
		return strings.Replace(endpoint, "/search", "/contents", 1)
	}
	return exaContentsEndpointDefault
}

func (c *SearchClient) tavilyExtractEndpoint() string {
	endpoint := strings.TrimSpace(c.cfg.TavilyEndpoint)
	if endpoint == "" {
		return tavilyExtractEndpointDefault
	}
	if strings.Contains(endpoint, "/search") {
		return strings.Replace(endpoint, "/search", "/extract", 1)
	}
	return tavilyExtractEndpointDefault
}

// --- Helper types + functions ---

type searxngResponse struct {
	Query            string           `json:"query"`
	NumberOfResults  int              `json:"number_of_results"`
	Results          []searxngResult  `json:"results"`
	Corrections      []string         `json:"corrections"`
	UnresponsiveList []string         `json:"unresponsive_engines"`
	Answers          []string         `json:"answers"`
	Info             []map[string]any `json:"infoboxes"`
}

type searxngResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Engine  string `json:"engine"`
}

type exaSearchResponse struct {
	Results []exaSearchResult `json:"results"`
}

type exaSearchResult struct {
	Title         string   `json:"title"`
	URL           string   `json:"url"`
	PublishedDate string   `json:"publishedDate"`
	Author        string   `json:"author"`
	Score         float64  `json:"score"`
	Text          string   `json:"text"`
	Highlights    []string `json:"highlights"`
	Summary       string   `json:"summary"`
}

type exaContentsResponse struct {
	Results []exaContentResult `json:"results"`
}

type exaContentResult struct {
	URL  string `json:"url"`
	Text string `json:"text"`
}

type tavilySearchResponse struct {
	Query   string               `json:"query"`
	Answer  string               `json:"answer"`
	Results []tavilySearchResult `json:"results"`
}

type tavilySearchResult struct {
	Title         string  `json:"title"`
	URL           string  `json:"url"`
	Content       string  `json:"content"`
	RawContent    string  `json:"raw_content"`
	Score         float64 `json:"score"`
	PublishedDate string  `json:"published_date"`
}

type tavilyExtractResponse struct {
	Results []tavilyExtractResult `json:"results"`
}

type tavilyExtractResult struct {
	URL        string `json:"url"`
	Content    string `json:"content"`
	RawContent string `json:"raw_content"`
}

func sanitizeDomain(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	if idx := strings.Index(value, "#"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	if fields := strings.Fields(value); len(fields) > 0 {
		value = fields[0]
	}
	if value == "" {
		return ""
	}
	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Host == "" {
			return ""
		}
		value = parsed.Host
	}
	if slash := strings.Index(value, "/"); slash >= 0 {
		value = value[:slash]
	}
	value = strings.TrimPrefix(value, "www.")
	value = strings.Trim(value, ".-")
	if value == "" {
		return ""
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-' {
			continue
		}
		return ""
	}
	return value
}

func applyDomainFilter(query string, domains []string) string {
	if len(domains) == 0 {
		return query
	}

	var filters []string
	for _, domain := range domains {
		if domain == "" {
			continue
		}
		filters = append(filters, fmt.Sprintf("site:%s", domain))
	}
	if len(filters) == 0 {
		return query
	}

	filterExpr := strings.Join(filters, " OR ")
	query = strings.TrimSpace(query)
	if query == "" {
		return filterExpr
	}
	return fmt.Sprintf("(%s) (%s)", query, filterExpr)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func extractVisibleText(htmlBytes []byte) string {
	doc, err := html.Parse(strings.NewReader(string(htmlBytes)))
	if err != nil {
		return ""
	}

	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			val := strings.TrimSpace(n.Data)
			if val != "" {
				if builder.Len() > 0 {
					builder.WriteString(" ")
				}
				builder.WriteString(val)
			}
		}
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return builder.String()
}

// ImageSearch performs an image search using the Serper API.
func (c *SearchClient) ImageSearch(ctx context.Context, query domainsearch.ImageSearchRequest) (*domainsearch.ImageSearchResponse, error) {
	offline := c.resolveOfflineMode(query.OfflineMode)

	log.Debug().
		Str("operation", "image_search").
		Str("query", query.Q).
		Bool("offline_mode", offline).
		Bool("serper_enabled", c.cfg.SerperEnabled && c.hasSerperAPIKey()).
		Msg("image search client starting")

	if offline {
		return nil, fmt.Errorf("image search unavailable: offline mode is enabled")
	}

	// Currently only Serper supports image search
	if !c.cfg.SerperEnabled || !c.hasSerperAPIKey() {
		return nil, fmt.Errorf("image search unavailable: Serper provider not enabled or missing API key")
	}

	return c.imageSearchViaSerper(ctx, query)
}

// imageSearchViaSerper performs image search using the Serper Images API.
func (c *SearchClient) imageSearchViaSerper(ctx context.Context, query domainsearch.ImageSearchRequest) (*domainsearch.ImageSearchResponse, error) {
	// Check circuit breaker
	if c.serperCB.GetState() == StateOpen {
		log.Error().Str("service", "serper").Msg("serper circuit breaker is open, skipping image search")
		return nil, fmt.Errorf("serper circuit breaker is open")
	}

	startTime := time.Now()
	status := "success"
	defer func() {
		metrics.RecordProviderRequest("image_search", "serper", status)
		metrics.RecordExternalProviderLatency("serper", time.Since(startTime).Seconds())
	}()

	body := map[string]any{
		"q": query.Q,
	}
	if query.GL != nil {
		body["gl"] = *query.GL
	}
	if query.HL != nil {
		body["hl"] = *query.HL
	}
	if query.Num != nil && *query.Num > 0 {
		body["num"] = *query.Num
	}
	if query.Autocorrect != nil {
		body["autocorrect"] = *query.Autocorrect
	}

	var opErr error

	// Retry with exponential backoff
	resultPtr, err := WithRetry(ctx, c.retryConfig, "serper_image_search", func() (*domainsearch.ImageSearchResponse, error) {
		// Use map for initial parsing since Serper returns images as []map[string]any
		var rawResp struct {
			SearchParameters map[string]any   `json:"searchParameters"`
			Images           []map[string]any `json:"images"`
		}

		resp, err := c.serperClient.R().
			SetContext(ctx).
			SetHeader("X-API-KEY", c.cfg.SerperAPIKey).
			SetHeader("Content-Type", "application/json").
			SetBody(body).
			SetResult(&rawResp).
			Post(serperImageSearchEndpoint)

		if err != nil {
			log.Error().Err(err).Str("service", "serper").Str("endpoint", serperImageSearchEndpoint).Msg("failed to query Serper image search API")
			return nil, fmt.Errorf("failed to query Serper image search API: %w", err)
		}

		if resp.IsError() {
			log.Error().Int("status", resp.StatusCode()).Str("service", "serper").Str("response", resp.String()).Msg("Serper image search API error")
			return nil, fmt.Errorf("Serper image search API error (status %d): %s", resp.StatusCode(), resp.String())
		}

		// Convert raw response to typed response
		result := &domainsearch.ImageSearchResponse{
			SearchParameters: rawResp.SearchParameters,
			Images:           make([]domainsearch.ImageSearchResult, 0, len(rawResp.Images)),
		}

		for _, img := range rawResp.Images {
			result.Images = append(result.Images, domainsearch.ImageSearchResult{
				Title:           getStringFromMap(img, "title"),
				ImageURL:        getStringFromMap(img, "imageUrl"),
				ImageWidth:      getIntFromMap(img, "imageWidth"),
				ImageHeight:     getIntFromMap(img, "imageHeight"),
				ThumbnailURL:    getStringFromMap(img, "thumbnailUrl"),
				ThumbnailWidth:  getIntFromMap(img, "thumbnailWidth"),
				ThumbnailHeight: getIntFromMap(img, "thumbnailHeight"),
				Source:          getStringFromMap(img, "source"),
				Domain:          getStringFromMap(img, "domain"),
				Link:            getStringFromMap(img, "link"),
				GoogleURL:       getStringFromMap(img, "googleUrl"),
				Creator:         getStringFromMap(img, "creator"),
				Credit:          getStringFromMap(img, "credit"),
				Position:        getIntFromMap(img, "position"),
			})
		}

		return result, nil
	})

	opErr = err

	// Update circuit breaker
	c.serperCB.recordResult("serper_image_search", opErr)

	if opErr != nil {
		status = "error"
		log.Error().Err(opErr).Str("service", "serper").Str("operation", "image_search").Msg("serper image search failed after retries")
		return nil, opErr
	}

	if resultPtr.SearchParameters == nil {
		resultPtr.SearchParameters = map[string]any{}
	}
	resultPtr.SearchParameters["engine"] = "serper"
	resultPtr.SearchParameters["type"] = "images"
	resultPtr.SearchParameters["live"] = true

	log.Info().
		Str("engine", "serper").
		Str("query", query.Q).
		Int("result_count", len(resultPtr.Images)).
		Msg("image search completed")

	return resultPtr, nil
}

// getStringFromMap safely extracts a string from a map
func getStringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getIntFromMap safely extracts an int from a map
func getIntFromMap(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return 0
}
