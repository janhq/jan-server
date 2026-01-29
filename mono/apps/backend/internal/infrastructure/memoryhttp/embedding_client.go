package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// EmbeddingClient is an HTTP client for the BGE-M3 embedding service
type EmbeddingClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// EmbedRequest represents a request to the embedding service
type EmbedRequest struct {
	Inputs    interface{} `json:"inputs"` // string or []string
	Normalize bool        `json:"normalize"`
	Truncate  bool        `json:"truncate"`
}

// EmbedResponse represents the response from the embedding service
type EmbedResponse [][]float32

// ModelInfo represents model information
type ModelInfo struct {
	ModelID        string `json:"model_id"`
	MaxInputLength int    `json:"max_input_length"`
}

// NewEmbeddingClient creates a new embedding HTTP client
func NewEmbeddingClient(baseURL, apiKey string, timeout time.Duration) *EmbeddingClient {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &EmbeddingClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Embed generates embeddings for the given texts
func (c *EmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := EmbedRequest{
		Inputs:    texts,
		Normalize: true,
		Truncate:  true,
	}
	log.Info().
		Int("text_count", len(texts)).
		Str("endpoint", c.baseURL+"/embed").
		Msg("embedding request")

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embed", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Error().
			Int("status", resp.StatusCode).
			Str("endpoint", c.baseURL+"/embed").
			Msg("embedding request failed")
		return nil, fmt.Errorf("embedding service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var embeddings EmbedResponse
	if err := json.Unmarshal(bodyBytes, &embeddings); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	log.Info().
		Int("status", resp.StatusCode).
		Int("embeddings", len(embeddings)).
		Int("dimension", func() int {
			if len(embeddings) > 0 {
				return len(embeddings[0])
			}
			return 0
		}()).
		Msg("embedding response")

	return embeddings, nil
}

// EmbedSingle generates an embedding for a single text
func (c *EmbeddingClient) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := c.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return embeddings[0], nil
}

// Health checks the health of the embedding service
func (c *EmbeddingClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// Info retrieves model information
func (c *EmbeddingClient) Info(ctx context.Context) (*ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/info", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("info request failed with status %d", resp.StatusCode)
	}

	var info ModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &info, nil
}

// ValidateServer validates the embedding server
func (c *EmbeddingClient) ValidateServer(ctx context.Context) error {
	// Check health
	if err := c.Health(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// Check model info
	info, err := c.Info(ctx)
	if err != nil {
		return fmt.Errorf("info check failed: %w", err)
	}

	// Verify it's BGE-M3
	if info.ModelID != "BAAI/bge-m3" {
		log.Warn().Str("model", info.ModelID).Msg("Expected BGE-M3, got different model")
	}

	// Test embedding
	embeddings, err := c.Embed(ctx, []string{"test"})
	if err != nil {
		return fmt.Errorf("test embedding failed: %w", err)
	}

	if len(embeddings) == 0 || len(embeddings[0]) != 1024 {
		return fmt.Errorf("expected 1024 dimensions, got %d", len(embeddings[0]))
	}

	log.Info().
		Str("model", info.ModelID).
		Int("max_input_length", info.MaxInputLength).
		Msg("Embedding server validated")

	return nil
}
