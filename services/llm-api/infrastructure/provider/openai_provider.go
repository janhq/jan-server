package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const (
	chatCompletionsPath = "/chat/completions"
	modelsPath          = "/models"
)

// OpenAIProvider proxies requests to OpenAI compatible backends such as vLLM.
type OpenAIProvider struct {
	name       string
	baseURL    *url.URL
	headers    map[string]string
	httpClient *http.Client
	logger     zerolog.Logger
}

// OpenAIOption configures an OpenAIProvider.
type OpenAIOption func(*OpenAIProvider)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) OpenAIOption {
	return func(p *OpenAIProvider) {
		p.httpClient = client
	}
}

// NewOpenAIProvider constructs a new provider instance.
func NewOpenAIProvider(name string, baseURL string, headers map[string]string, logger zerolog.Logger, opts ...OpenAIOption) (*OpenAIProvider, error) {
	if name == "" {
		return nil, fmt.Errorf("provider name is required")
	}
	if baseURL == "" {
		return nil, fmt.Errorf("provider baseURL is required")
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}

	provider := &OpenAIProvider{
		name:    name,
		baseURL: u,
		headers: headers,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}

	for _, opt := range opts {
		opt(provider)
	}

	return provider, nil
}

// Name returns provider's configured name.
func (p *OpenAIProvider) Name() string {
	return p.name
}

// Supports indicates whether a model is compatible with this provider.
func (p *OpenAIProvider) Supports(model ModelConfig) bool {
	// For now assume all configured models are supported.
	return true
}

func (p *OpenAIProvider) composeURL(path string) string {
	return strings.TrimRight(p.baseURL.String(), "/") + "/v1" + path
}

func (p *OpenAIProvider) applyHeaders(req *http.Request, principalHeaders map[string]string) {
	for k, v := range p.headers {
		req.Header.Set(k, v)
	}
	for k, v := range principalHeaders {
		if v == "" {
			continue
		}
		req.Header.Set(k, v)
	}
}

// ChatCompletions proxies a non-streaming chat completion request.
func (p *OpenAIProvider) ChatCompletions(ctx context.Context, req ChatCompletionRequest, principalHeaders map[string]string) (*ChatCompletionResponse, error) {
	req.Stream = false
	resp, err := p.performRequest(ctx, req, principalHeaders)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read provider response: %w", err)
	}
	return &ChatCompletionResponse{
		Body:       body,
		StatusCode: resp.StatusCode,
		Headers:    collectHeaders(resp.Header),
	}, nil
}

// ChatCompletionsStream proxies a streaming chat completion request by returning a StreamResponse.
func (p *OpenAIProvider) ChatCompletionsStream(ctx context.Context, req ChatCompletionRequest, principalHeaders map[string]string) (StreamResponse, error) {
	req.Stream = true
	resp, err := p.performRequest(ctx, req, principalHeaders)
	if err != nil {
		return nil, err
	}

	stream := &httpStreamResponse{resp: resp}
	return stream, nil
}

func (p *OpenAIProvider) performRequest(ctx context.Context, req ChatCompletionRequest, principalHeaders map[string]string) (*http.Response, error) {
	payload := map[string]any{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   req.Stream,
	}

	if req.Temperature != nil {
		payload["temperature"] = req.Temperature
	}
	if req.TopP != nil {
		payload["top_p"] = req.TopP
	}
	if req.MaxTokens != nil {
		payload["max_tokens"] = req.MaxTokens
	}
	for k, v := range req.Extras {
		payload[k] = v
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal provider payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.composeURL(chatCompletionsPath), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create provider request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	p.applyHeaders(httpReq, principalHeaders)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute provider request: %w", err)
	}

	return resp, nil
}

// ListModels fetches models from the provider.
func (p *OpenAIProvider) ListModels(ctx context.Context) ([]RemoteModel, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.composeURL(modelsPath), nil)
	if err != nil {
		return nil, fmt.Errorf("create list models request: %w", err)
	}
	p.applyHeaders(httpReq, nil)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("provider list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("provider list models status=%d body=%s", resp.StatusCode, body)
	}

	var payload struct {
		Data []struct {
			ID    string `json:"id"`
			Owned string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode provider list models: %w", err)
	}

	out := make([]RemoteModel, 0, len(payload.Data))
	for _, item := range payload.Data {
		out = append(out, RemoteModel{
			ID:           item.ID,
			DisplayName:  item.ID,
			Family:       item.Owned,
			Capabilities: []string{"chat"},
		})
	}
	return out, nil
}

// HealthCheck verifies the provider responds to /models.
func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
	_, err := p.ListModels(ctx)
	return err
}

type httpStreamResponse struct {
	resp *http.Response
}

func (h *httpStreamResponse) Stream(ctx context.Context, cb func(data []byte) error) error {
	reader := bufio.NewReader(h.resp.Body)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			if err := cb(line); err != nil {
				return err
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func (h *httpStreamResponse) Close() error {
	if h.resp == nil || h.resp.Body == nil {
		return nil
	}
	return h.resp.Body.Close()
}

func (h *httpStreamResponse) Headers() map[string]string {
	return collectHeaders(h.resp.Header)
}

func (h *httpStreamResponse) StatusCode() int {
	return h.resp.StatusCode
}

func collectHeaders(header http.Header) map[string]string {
	out := make(map[string]string, len(header))
	for k, v := range header {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}
