package posthog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/posthog/posthog-go"
)

// Version is the library version sent with events
var Version = "1.0.0"

// ErrMissingAPIKey is returned when PostHog is enabled but no API key is provided
var ErrMissingAPIKey = errors.New("posthog: API key is required")

// Config holds PostHog-specific configuration
type Config struct {
	Enabled       bool
	APIKey        string
	Host          string
	Debug         bool
	BatchSize     int
	FlushInterval time.Duration
}

// DefaultConfig returns a default PostHog configuration
func DefaultConfig() Config {
	return Config{
		Enabled:       false,
		Host:          "https://eu.posthog.com",
		BatchSize:     100,
		FlushInterval: 10 * time.Second,
	}
}

// Event represents a tracked event
type Event struct {
	Name       string
	DistinctID string
	Timestamp  time.Time
	Properties map[string]interface{}
}

// UserProperties for identify calls
type UserProperties struct {
	Email     string
	Name      string
	CreatedAt time.Time
	Plan      string
	Platform  string
	Custom    map[string]string
}

// Client wraps the PostHog client
type Client struct {
	client posthog.Client
	config Config
}

// New creates a new PostHog client
func New(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, ErrMissingAPIKey
	}

	client, err := posthog.NewWithConfig(
		cfg.APIKey,
		posthog.Config{
			Endpoint:  cfg.Host,
			BatchSize: cfg.BatchSize,
			Interval:  cfg.FlushInterval,
			Verbose:   cfg.Debug,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("posthog: failed to create client: %w", err)
	}

	return &Client{
		client: client,
		config: cfg,
	}, nil
}

// Track sends an event to PostHog
func (c *Client) Track(ctx context.Context, event Event) error {
	properties := posthog.NewProperties().
		Set("$lib", "jan-server").
		Set("$lib_version", Version)

	// Add all event properties
	for k, v := range event.Properties {
		properties.Set(k, v)
	}

	return c.client.Enqueue(posthog.Capture{
		DistinctId: event.DistinctID,
		Event:      event.Name,
		Properties: properties,
		Timestamp:  event.Timestamp,
	})
}

// Identify associates user properties with a distinct ID
func (c *Client) Identify(ctx context.Context, distinctID string, props UserProperties) error {
	properties := posthog.NewProperties()

	if props.Email != "" {
		properties.Set("email", props.Email)
	}
	if props.Name != "" {
		properties.Set("name", props.Name)
	}
	if !props.CreatedAt.IsZero() {
		properties.Set("created_at", props.CreatedAt)
	}
	if props.Plan != "" {
		properties.Set("plan", props.Plan)
	}
	if props.Platform != "" {
		properties.Set("platform", props.Platform)
	}

	// Add custom properties
	for k, v := range props.Custom {
		properties.Set(k, v)
	}

	return c.client.Enqueue(posthog.Identify{
		DistinctId: distinctID,
		Properties: properties,
	})
}

// Flush ensures all queued events are sent
func (c *Client) Flush(ctx context.Context) error {
	// PostHog client doesn't have a Flush method, Close flushes and closes
	// For flush without close, we rely on the batch interval
	return nil
}

// Shutdown gracefully closes the PostHog client
func (c *Client) Shutdown(ctx context.Context) error {
	return c.client.Close()
}

// Alias creates an alias for a distinct ID (useful for linking anonymous to identified users)
func (c *Client) Alias(ctx context.Context, distinctID, alias string) error {
	return c.client.Enqueue(posthog.Alias{
		DistinctId: distinctID,
		Alias:      alias,
	})
}

// GroupIdentify identifies a group
func (c *Client) GroupIdentify(ctx context.Context, groupType, groupKey string, properties map[string]interface{}) error {
	props := posthog.NewProperties()
	for k, v := range properties {
		props.Set(k, v)
	}

	return c.client.Enqueue(posthog.GroupIdentify{
		Type:       groupType,
		Key:        groupKey,
		Properties: props,
	})
}
