package kong

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Client implements a thin wrapper around the Kong Admin API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     zerolog.Logger
}

// Consumer represents a Kong consumer entity.
type Consumer struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	CustomID string `json:"custom_id"`
}

// KeyAuthCredential represents a key-auth credential in Kong.
type KeyAuthCredential struct {
	ID        string   `json:"id"`
	Key       string   `json:"key"`
	CreatedAt int64    `json:"created_at"`
	Tags      []string `json:"tags"`
}

// NewClient constructs a Kong Admin API client.
func NewClient(baseURL string, httpClient *http.Client, logger zerolog.Logger) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
		logger:     logger.With().Str("component", "kong-admin-client").Logger(),
	}
}

// EnsureConsumer fetches or creates a consumer with the provided identifiers.
func (c *Client) EnsureConsumer(ctx context.Context, username, customID string, tags []string) (*Consumer, error) {
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if consumer, err := c.getConsumer(ctx, username); err == nil {
		return consumer, nil
	} else if !isNotFound(err) {
		return nil, err
	}
	payload := map[string]any{
		"username":  username,
		"custom_id": customID,
	}
	if len(tags) > 0 {
		payload["tags"] = tags
	}
	var resp Consumer
	if err := c.do(ctx, http.MethodPost, "/consumers", payload, &resp); err != nil {
		if isConflict(err) {
			return c.getConsumer(ctx, username)
		}
		return nil, err
	}
	return &resp, nil
}

// CreateKeyCredential registers a key-auth credential for the given consumer username.
func (c *Client) CreateKeyCredential(ctx context.Context, username, key string, tags []string) (*KeyAuthCredential, error) {
	if username == "" {
		return nil, fmt.Errorf("consumer username is required")
	}
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	endpoint := fmt.Sprintf("/consumers/%s/key-auth", username)
	payload := map[string]any{"key": key}
	if len(tags) > 0 {
		payload["tags"] = tags
	}

	var resp KeyAuthCredential
	if err := c.do(ctx, http.MethodPost, endpoint, payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteKeyCredential removes a key-auth credential by its ID.
func (c *Client) DeleteKeyCredential(ctx context.Context, credentialID string) error {
	if credentialID == "" {
		return fmt.Errorf("credential id is required")
	}
	endpoint := fmt.Sprintf("/key-auth/%s", credentialID)
	return c.do(ctx, http.MethodDelete, endpoint, nil, nil)
}

func (c *Client) getConsumer(ctx context.Context, username string) (*Consumer, error) {
	endpoint := fmt.Sprintf("/consumers/%s", username)
	var resp Consumer
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) do(ctx context.Context, method, path string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		buf, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
		body = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		data, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return &Error{
			Code:    res.StatusCode,
			Message: strings.TrimSpace(string(data)),
		}
	}

	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// Error represents an HTTP error returned by Kong Admin API.
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("kong admin api error: %d", e.Code)
	}
	return fmt.Sprintf("kong admin api error: %d %s", e.Code, e.Message)
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := err.(*Error); ok && apiErr.Code == http.StatusNotFound {
		return true
	}
	return false
}

func isConflict(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := err.(*Error); ok && apiErr.Code == http.StatusConflict {
		return true
	}
	return false
}
