package connectorapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ConnectorType represents the type of connector.
type ConnectorType string

const (
	ConnectorTypeGitHub         ConnectorType = "github"
	ConnectorTypeGmail          ConnectorType = "gmail"
	ConnectorTypeGoogleDrive    ConnectorType = "google_drive"
	ConnectorTypeGoogleCalendar ConnectorType = "google_calendar"
)

// Client is a client for the llm-api connector service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new connector API client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TokenResponse is the response from the token endpoint.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

// GetAccessToken gets the decrypted access token for a connector.
// The authToken is the user's JWT for authentication.
func (c *Client) GetAccessToken(ctx context.Context, authToken string, connectorType ConnectorType) (*TokenResponse, error) {
	url := fmt.Sprintf("%s/v1/connectors/%s/token", c.baseURL, connectorType)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("connector not connected")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &tokenResp, nil
}

// StatusResponse is the response from the status endpoint.
type StatusResponse struct {
	Connected bool   `json:"connected"`
	Enabled   bool   `json:"enabled"`
	Status    string `json:"status,omitempty"`
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty"`
}

// GetStatus gets the connection status for a connector.
func (c *Client) GetStatus(ctx context.Context, authToken string, connectorType ConnectorType) (*StatusResponse, error) {
	url := fmt.Sprintf("%s/v1/connectors/%s/status", c.baseURL, connectorType)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var statusResp StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &statusResp, nil
}
