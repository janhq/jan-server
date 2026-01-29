package vectorstore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	baseURL    string
	httpClient *resty.Client
}

type IndexRequest struct {
	DocumentID string         `json:"document_id"`
	Text       string         `json:"text"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Tags       []string       `json:"tags,omitempty"`
}

type IndexResponse struct {
	Status     string `json:"status"`
	DocumentID string `json:"document_id"`
	TokenCount int    `json:"token_count"`
	IndexedAt  string `json:"indexed_at"`
}

type QueryRequest struct {
	Text        string   `json:"text"`
	TopK        int      `json:"top_k,omitempty"`
	DocumentIDs []string `json:"document_ids,omitempty"`
}

type QueryResult struct {
	DocumentID  string         `json:"document_id"`
	Score       float64        `json:"score"`
	TextPreview string         `json:"text_preview"`
	Metadata    map[string]any `json:"metadata"`
	Tags        []string       `json:"tags,omitempty"`
}

type QueryResponse struct {
	Query   string        `json:"query"`
	TopK    int           `json:"top_k"`
	Count   int           `json:"count"`
	Results []QueryResult `json:"results"`
}

func NewClient(baseURL string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return nil
	}
	httpClient := resty.New().
		SetBaseURL(baseURL).
		SetHeader("User-Agent", "Jan-MCP-VectorStore/1.0").
		SetTimeout(10 * time.Second)

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *Client) IsEnabled() bool {
	return c != nil && c.baseURL != ""
}

func (c *Client) IndexDocument(ctx context.Context, req IndexRequest) (*IndexResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("vector store client is not configured")
	}

	var resp IndexResponse
	httpResp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&resp).
		Post("/documents")
	if err != nil {
		return nil, fmt.Errorf("vector store index request failed: %w", err)
	}
	if httpResp.IsError() {
		return nil, fmt.Errorf("vector store index error (%d): %s", httpResp.StatusCode(), httpResp.String())
	}
	return &resp, nil
}

func (c *Client) Query(ctx context.Context, req QueryRequest) (*QueryResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("vector store client is not configured")
	}

	var resp QueryResponse
	httpResp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&resp).
		Post("/query")
	if err != nil {
		return nil, fmt.Errorf("vector store query request failed: %w", err)
	}
	if httpResp.IsError() {
		return nil, fmt.Errorf("vector store query error (%d): %s", httpResp.StatusCode(), httpResp.String())
	}
	return &resp, nil
}
