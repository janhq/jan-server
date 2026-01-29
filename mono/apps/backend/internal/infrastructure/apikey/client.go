package apikey

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

// Client handles API key generation via LLM-API.
type Client struct {
	httpClient *resty.Client
	baseURL    string
}

// NewClient creates a new API key client.
func NewClient(baseURL string) *Client {
	return &Client{
		httpClient: resty.New().
			SetBaseURL(baseURL).
			SetHeader("Content-Type", "application/json").
			SetTimeout(30 * time.Second),
		baseURL: baseURL,
	}
}

// CreateRequest is the request body for creating an API key.
type CreateRequest struct {
	Name      string         `json:"name"`
	ExpiresIn *time.Duration `json:"expires_in,omitempty"` // Duration in nanoseconds
}

// CreateResponse is the response from creating an API key.
type CreateResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	Suffix    string    `json:"suffix"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Status    string    `json:"status"`
	Key       string    `json:"key"` // The actual API key (only returned on creation)
}

// CreateTemporaryKey generates a temporary API key for service-to-service calls.
// It uses the provided bearer token to authenticate with LLM-API.
// The key expires after the specified TTL (default: 1 hour).
func (c *Client) CreateTemporaryKey(ctx context.Context, bearerToken string, ttl time.Duration) (*CreateResponse, error) {
	if bearerToken == "" {
		return nil, fmt.Errorf("bearer token is required to create temporary API key")
	}

	if ttl == 0 {
		ttl = time.Hour // Default 1 hour expiry
	}

	// Generate a unique name for the temporary key
	keyName := fmt.Sprintf("response-api-temp-%d", time.Now().UnixNano())

	req := CreateRequest{
		Name:      keyName,
		ExpiresIn: &ttl,
	}

	var result CreateResponse
	var errResp map[string]interface{}

	// Normalize bearer token
	authHeader := bearerToken
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		authHeader = "Bearer " + authHeader
	}

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Authorization", authHeader).
		SetHeader("X-System-Key", "true"). // Mark as system key (not shown in user's key list)
		SetBody(req).
		SetResult(&result).
		SetError(&errResp).
		Post("/auth/api-keys")

	if err != nil {
		log.Error().Err(err).Msg("Failed to call LLM-API for API key creation")
		return nil, fmt.Errorf("failed to create temporary API key: %w", err)
	}

	if resp.IsError() {
		log.Error().
			Int("status", resp.StatusCode()).
			Interface("error", errResp).
			Msg("LLM-API returned error for API key creation")
		return nil, fmt.Errorf("LLM-API error: %d - %v", resp.StatusCode(), errResp)
	}

	if result.Key == "" {
		return nil, fmt.Errorf("LLM-API did not return API key")
	}

	log.Info().
		Str("key_id", result.ID).
		Str("key_name", result.Name).
		Time("expires_at", result.ExpiresAt).
		Msg("Created temporary API key for response-api")

	return &result, nil
}

// DeleteKey revokes an API key by ID.
func (c *Client) DeleteKey(ctx context.Context, bearerToken string, keyID string) error {
	if bearerToken == "" || keyID == "" {
		return nil // Nothing to delete
	}

	authHeader := bearerToken
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		authHeader = "Bearer " + authHeader
	}

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Authorization", authHeader).
		Delete("/auth/api-keys/" + keyID)

	if err != nil {
		log.Warn().Err(err).Str("key_id", keyID).Msg("Failed to delete temporary API key")
		return err
	}

	if resp.IsError() && resp.StatusCode() != 404 {
		log.Warn().
			Int("status", resp.StatusCode()).
			Str("key_id", keyID).
			Msg("LLM-API returned error when deleting API key")
	}

	return nil
}

// Provider interface for API key operations.
type Provider interface {
	// CreateTemporaryKey generates a temporary API key using the user's bearer token.
	CreateTemporaryKey(ctx context.Context, bearerToken string, ttl time.Duration) (*CreateResponse, error)
	// DeleteKey revokes an API key by ID.
	DeleteKey(ctx context.Context, bearerToken string, keyID string) error
}

// Ensure Client implements Provider
var _ Provider = (*Client)(nil)
