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
			SetTimeout(900 * time.Second),
		baseURL: baseURL,
	}
}

// CreateChatCompletion calls llm-api /v1/chat/completions.
func (c *Client) CreateChatCompletion(ctx context.Context, req llm.ChatCompletionRequest) (*llm.ChatCompletionResponse, error) {
	// Convert to API-compatible format with string content
	apiReq := convertToAPIRequest(req)

	var completion llm.ChatCompletionResponse
	request := c.httpClient.R().
		SetContext(ctx).
		SetBody(apiReq).
		SetResult(&completion)

	setAuthHeaders(ctx, func(key, value string) {
		request.SetHeader(key, value)
	})

	resp, err := request.Post("/v1/chat/completions")
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
	setAuthHeaders(ctx, httpReq.Header.Set)

	httpClient := &http.Client{Timeout: 900 * time.Second}
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

// convertToAPIRequest converts domain types to API-compatible format.
// This ensures Content is always a string as expected by LLM API.
func convertToAPIRequest(req llm.ChatCompletionRequest) map[string]interface{} {
	// Convert messages with string content
	messages := make([]map[string]interface{}, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.GetContentAsString(), // Convert content to string
		}

		if len(msg.ToolCalls) > 0 {
			messages[i]["tool_calls"] = msg.ToolCalls
		}

		if msg.ToolCallID != nil {
			messages[i]["tool_call_id"] = *msg.ToolCallID
		}
	}

	apiReq := map[string]interface{}{
		"model":    req.Model,
		"messages": messages,
		"stream":   req.Stream,
	}

	if len(req.Tools) > 0 {
		apiReq["tools"] = req.Tools
	}
	if req.ToolChoice != nil {
		apiReq["tool_choice"] = req.ToolChoice
	}
	if req.Temperature != nil {
		apiReq["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		apiReq["max_tokens"] = *req.MaxTokens
	}

	return apiReq
}

func setAuthHeaders(ctx context.Context, setHeader func(string, string)) {
	token := strings.TrimSpace(llm.AuthTokenFromContext(ctx))
	if token == "" {
		return
	}
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		setHeader("Authorization", token)
		return
	}
	setHeader("X-API-Key", token)
}

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
