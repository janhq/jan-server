package llmprovider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"jan-server/services/response-api/internal/domain/llm"
)

// Client implements the llm.Provider interface.
type Client struct {
	httpClient *resty.Client
	baseURL    string
}

// NewClient creates a Resty-backed client.
func NewClient(baseURL string) *Client {
	return &Client{
		httpClient: resty.New().
			SetBaseURL(baseURL).
			SetHeader("Content-Type", "application/json").
			SetTimeout(75 * time.Second),
		baseURL: baseURL,
	}
}

// CreateChatCompletion calls llm-api /v1/chat/completions.
func (c *Client) CreateChatCompletion(ctx context.Context, req llm.ChatCompletionRequest) (*llm.ChatCompletionResponse, error) {
	var completion llm.ChatCompletionResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&completion).
		Post("/v1/chat/completions")
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("llm api error: %s", resp.String())
	}
	return &completion, nil
}

// CreateChatCompletionStream calls llm-api /v1/chat/completions with streaming enabled.
func (c *Client) CreateChatCompletionStream(ctx context.Context, req llm.ChatCompletionRequest) (llm.Stream, error) {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	httpClient := &http.Client{Timeout: 120 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("llm api error: %d %s", resp.StatusCode, string(body))
	}

	return &sseStream{
		resp:   resp,
		reader: bufio.NewReader(resp.Body),
	}, nil
}

// Ensure interface compliance.
var _ llm.Provider = (*Client)(nil)

// sseStream implements llm.Stream backed by http.Response body with SSE parsing.
type sseStream struct {
	resp   *http.Response
	reader *bufio.Reader
}

func (s *sseStream) Recv() (*llm.ChatCompletionDelta, error) {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil, io.EOF
			}
			return nil, fmt.Errorf("read line: %w", err)
		}

		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Look for data: prefix
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream termination
		if data == "[DONE]" {
			return nil, io.EOF
		}

		// Parse the JSON delta
		var delta llm.ChatCompletionDelta
		if err := json.Unmarshal([]byte(data), &delta); err != nil {
			// Skip malformed chunks
			continue
		}

		return &delta, nil
	}
}

func (s *sseStream) Close() error {
	if s.resp != nil && s.resp.Body != nil {
		return s.resp.Body.Close()
	}
	return nil
}
