package search

// TBSTimeRange defines time-based search filters for Serper API
type TBSTimeRange string

const (
	TBSAny       TBSTimeRange = ""
	TBSPastHour  TBSTimeRange = "qdr:h"
	TBSPastDay   TBSTimeRange = "qdr:d"
	TBSPastWeek  TBSTimeRange = "qdr:w"
	TBSPastMonth TBSTimeRange = "qdr:m"
	TBSPastYear  TBSTimeRange = "qdr:y"
)

// SearchRequest represents a search query to Serper API
type SearchRequest struct {
	Q               string        `json:"q"`
	GL              *string       `json:"gl,omitempty"`                // Region code (ISO 3166-1 alpha-2)
	HL              *string       `json:"hl,omitempty"`                // Language code (ISO 639-1)
	Location        *string       `json:"location,omitempty"`          // Location for search results
	LocationHint    *string       `json:"location_hint,omitempty"`     // Soft location preference (country/region/timezone)
	Num             *int          `json:"num,omitempty"`               // Number of results (default: 10)
	Page            *int          `json:"page,omitempty"`              // Page number (default: 1)
	Autocorrect     *bool         `json:"autocorrect,omitempty"`       // Enable autocorrect
	TBS             *TBSTimeRange `json:"tbs,omitempty"`               // Time-based search filter
	DomainAllowList []string      `json:"domain_allow_list,omitempty"` // Restrict results to these domains
	OfflineMode     *bool         `json:"offline_mode,omitempty"`      // Force cached/offline behaviour
}

// SearchResponse contains search results from Serper API
type SearchResponse struct {
	SearchParameters map[string]any   `json:"searchParameters"`
	Organic          []map[string]any `json:"organic"`
	KnowledgeGraph   map[string]any   `json:"knowledgeGraph,omitempty"`
	Images           []map[string]any `json:"images,omitempty"`
	News             []map[string]any `json:"news,omitempty"`
	AnswerBox        map[string]any   `json:"answerBox,omitempty"`
}

// ExaContents specifies what content to retrieve from Exa API
type ExaContents struct {
	Text          bool `json:"text,omitempty"`          // Include page text content
	Highlights    bool `json:"highlights,omitempty"`    // Include highlighted snippets
	Summary       bool `json:"summary,omitempty"`       // Include AI-generated summary
	MaxCharacters int  `json:"maxCharacters,omitempty"` // Limit text length
}

// ExaSearchRequest for Exa API
type ExaSearchRequest struct {
	Query          string       `json:"query"`
	NumResults     int          `json:"num_results,omitempty"`
	IncludeDomains []string     `json:"include_domains,omitempty"`
	ExcludeDomains []string     `json:"exclude_domains,omitempty"`
	StartPublished string       `json:"start_published_date,omitempty"`
	UseAutoprompt  bool         `json:"use_autoprompt,omitempty"`
	Type           string       `json:"type,omitempty"` // "keyword" or "neural"
	Contents       *ExaContents `json:"contents,omitempty"`
}

// TavilySearchRequest for Tavily API
type TavilySearchRequest struct {
	Query             string   `json:"query"`
	SearchDepth       string   `json:"search_depth,omitempty"` // "basic" or "advanced"
	MaxResults        int      `json:"max_results,omitempty"`
	IncludeDomains    []string `json:"include_domains,omitempty"`
	ExcludeDomains    []string `json:"exclude_domains,omitempty"`
	IncludeAnswer     bool     `json:"include_answer,omitempty"`
	IncludeRawContent bool     `json:"include_raw_content,omitempty"`
}

// FetchWebpageRequest represents a webpage scraping request
type FetchWebpageRequest struct {
	Url             string `json:"url"`
	IncludeMarkdown *bool  `json:"includeMarkdown,omitempty"`
	OfflineMode     *bool  `json:"offline_mode,omitempty"`
}

// FetchWebpageResponse contains scraped webpage content
type FetchWebpageResponse struct {
	Text     string         `json:"text"`
	Metadata map[string]any `json:"metadata"`
	Status   string         `json:"status,omitempty"` // "success", "partial", or "failed"
	Error    string         `json:"error,omitempty"`  // Error message if scrape failed
}

// ImageSearchRequest represents an image search query to Serper API
type ImageSearchRequest struct {
	Q           string  `json:"q"`
	GL          *string `json:"gl,omitempty"`          // Region code (ISO 3166-1 alpha-2)
	HL          *string `json:"hl,omitempty"`          // Language code (ISO 639-1)
	Num         *int    `json:"num,omitempty"`         // Number of results (default: 10, max: 100)
	Autocorrect *bool   `json:"autocorrect,omitempty"` // Enable autocorrect
	OfflineMode *bool   `json:"offline_mode,omitempty"`
}

// ImageSearchResult represents a single image result from the search
type ImageSearchResult struct {
	Title           string `json:"title"`
	ImageURL        string `json:"imageUrl"`
	ImageWidth      int    `json:"imageWidth"`
	ImageHeight     int    `json:"imageHeight"`
	ThumbnailURL    string `json:"thumbnailUrl"`
	ThumbnailWidth  int    `json:"thumbnailWidth,omitempty"`
	ThumbnailHeight int    `json:"thumbnailHeight,omitempty"`
	Source          string `json:"source"`
	Domain          string `json:"domain"`
	Link            string `json:"link"`
	GoogleURL       string `json:"googleUrl,omitempty"`
	Creator         string `json:"creator,omitempty"`
	Credit          string `json:"credit,omitempty"`
	Position        int    `json:"position"`
}

// ImageSearchResponse contains image search results from Serper API
type ImageSearchResponse struct {
	SearchParameters map[string]any      `json:"searchParameters"`
	Images           []ImageSearchResult `json:"images"`
}
