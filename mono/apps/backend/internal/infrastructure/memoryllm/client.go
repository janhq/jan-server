package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"jan-server/mono/apps/backend/internal/domain/memory"
	"github.com/rs/zerolog/log"
)

// Client implements LLM client for internal service calls
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// ChatCompletionRequest represents a request to the LLM API
type ChatCompletionRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	Temperature    float32         `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ResponseFormat specifies the response format
type ResponseFormat struct {
	Type string `json:"type"` // "json_object" or "text"
}

// ChatCompletionResponse represents the LLM API response
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewClient creates a new LLM client
func NewClient(baseURL, apiKey string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Complete implements memory.LLMClient interface
func (c *Client) Complete(ctx context.Context, prompt string, options memory.LLMOptions) (string, error) {
	// Build request
	req := ChatCompletionRequest{
		Model: options.Model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a helpful assistant that provides structured, accurate responses.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: options.Temperature,
		MaxTokens:   options.MaxTokens,
	}

	// Set response format if JSON requested
	if options.ResponseFormat == "json" {
		req.ResponseFormat = &ResponseFormat{Type: "json_object"}
	}

	// Marshal request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Execute request
	log.Debug().
		Str("model", options.Model).
		Str("endpoint", c.baseURL).
		Msg("Calling LLM API")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	content := chatResp.Choices[0].Message.Content

	log.Info().
		Str("model", chatResp.Model).
		Int("prompt_tokens", chatResp.Usage.PromptTokens).
		Int("completion_tokens", chatResp.Usage.CompletionTokens).
		Msg("LLM completion successful")

	return content, nil
}

// Health checks the health of the LLM service
func (c *Client) Health(ctx context.Context) error {
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
