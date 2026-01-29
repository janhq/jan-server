package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"jan-server/mono/apps/backend/internal/infrastructure/logger"
)

// Client handles communication with the memory-tools service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new memory client with the provided base URL and timeout.
func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// LoadRequest represents a memory load request.
type LoadRequest struct {
	UserID         string      `json:"user_id"`
	ProjectID      string      `json:"project_id,omitempty"`
	ConversationID string      `json:"conversation_id,omitempty"`
	Query          string      `json:"query"`
	Options        LoadOptions `json:"options"`
}

// LoadOptions contains options for memory loading.
type LoadOptions struct {
	MaxUserItems     int     `json:"max_user_items"`
	MaxProjectItems  int     `json:"max_project_items"`
	MaxEpisodicItems int     `json:"max_episodic_items"`
	MinSimilarity    float32 `json:"min_similarity"`
}

// LoadResponse contains loaded memories.
type LoadResponse struct {
	CoreMemory     []UserMemoryItem `json:"core_memory"`
	EpisodicMemory []EpisodicEvent  `json:"episodic_memory"`
	SemanticMemory []ProjectFact    `json:"semantic_memory"`
}

// UserMemoryItem represents a user memory item.
type UserMemoryItem struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Scope      string    `json:"scope"`
	Text       string    `json:"text"`
	Score      int       `json:"score"`
	Similarity float32   `json:"similarity"`
	CreatedAt  time.Time `json:"created_at"`
}

// ProjectFact represents a project fact.
type ProjectFact struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	Kind       string    `json:"kind"`
	Title      string    `json:"title"`
	Text       string    `json:"text"`
	Confidence float32   `json:"confidence"`
	Similarity float32   `json:"similarity"`
	CreatedAt  time.Time `json:"created_at"`
}

// EpisodicEvent represents an episodic event.
type EpisodicEvent struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Time       time.Time `json:"time"`
	Text       string    `json:"text"`
	Kind       string    `json:"kind"`
	Similarity float32   `json:"similarity"`
}

// Load retrieves relevant memories.
func (c *Client) Load(ctx context.Context, req LoadRequest) (*LoadResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	log := logger.GetLogger()
	log.Info().
		Str("base_url", c.baseURL).
		Str("user_id", req.UserID).
		Str("project_id", req.ProjectID).
		Str("conversation_id", req.ConversationID).
		Msg("memory load request")

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/memory/load", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Warn().Int("status", resp.StatusCode).Msg("memory load failed")
		return nil, fmt.Errorf("memory load failed with status %d: %s", resp.StatusCode, string(body))
	}

	var loadResp LoadResponse
	if err := json.Unmarshal(body, &loadResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	log.Info().
		Int("status", resp.StatusCode).
		Int("core_memory", len(loadResp.CoreMemory)).
		Int("semantic_memory", len(loadResp.SemanticMemory)).
		Int("episodic_memory", len(loadResp.EpisodicMemory)).
		Msg("memory load response")

	return &loadResp, nil
}

// ObserveRequest represents a memory observe request.
type ObserveRequest struct {
	UserID         string             `json:"user_id"`
	ProjectID      string             `json:"project_id,omitempty"`
	ConversationID string             `json:"conversation_id"`
	Messages       []ConversationItem `json:"messages"`
}

// ConversationItem represents a message.
type ConversationItem struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Observe stores conversation for memory extraction.
func (c *Client) Observe(ctx context.Context, req ObserveRequest) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	log := logger.GetLogger()
	log.Info().
		Str("base_url", c.baseURL).
		Str("user_id", req.UserID).
		Str("project_id", req.ProjectID).
		Str("conversation_id", req.ConversationID).
		Int("message_count", len(req.Messages)).
		Msg("memory observe request")

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/memory/observe", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warn().Int("status", resp.StatusCode).Msg("memory observe failed")
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("memory observe failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Info().Int("status", resp.StatusCode).Msg("memory observe response")

	return nil
}

// Health checks the health of memory-tools service.
func (c *Client) Health(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/healthz", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}
