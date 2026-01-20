package search

import "context"

// SearchClient defines the search operations required by the domain layer
type SearchClient interface {
	Search(ctx context.Context, query SearchRequest) (*SearchResponse, error)
	FetchWebpage(ctx context.Context, query FetchWebpageRequest) (*FetchWebpageResponse, error)
	ImageSearch(ctx context.Context, query ImageSearchRequest) (*ImageSearchResponse, error)
}

// SearchService orchestrates MCP operations across pluggable search engines while remaining transport-agnostic.
type SearchService struct {
	client SearchClient
}

// NewSearchService creates a new search service.
func NewSearchService(client SearchClient) *SearchService {
	return &SearchService{
		client: client,
	}
}

// Search performs a web search using Serper API
func (s *SearchService) Search(ctx context.Context, query SearchRequest) (*SearchResponse, error) {
	return s.client.Search(ctx, query)
}

// FetchWebpage scrapes a webpage using Serper API
func (s *SearchService) FetchWebpage(ctx context.Context, query FetchWebpageRequest) (*FetchWebpageResponse, error) {
	return s.client.FetchWebpage(ctx, query)
}

// ImageSearch performs an image search using Serper API
func (s *SearchService) ImageSearch(ctx context.Context, query ImageSearchRequest) (*ImageSearchResponse, error) {
	return s.client.ImageSearch(ctx, query)
}
