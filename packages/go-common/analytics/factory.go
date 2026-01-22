package analytics

import (
	"context"
	"fmt"

	"github.com/janhq/jan-server/packages/go-common/analytics/backends/otel"
	"github.com/janhq/jan-server/packages/go-common/analytics/backends/posthog"
)

// NewTracker creates a new analytics tracker based on configuration
// It returns a MultiTracker that fans out to all enabled backends
func NewTracker(cfg Config, sanitizer *Sanitizer) (Tracker, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	mt := &MultiTracker{
		backends:  make([]Tracker, 0),
		sanitizer: sanitizer,
		config:    cfg,
	}

	// If analytics is disabled globally, use no-op
	if !cfg.Enabled {
		mt.backends = append(mt.backends, NewNoopTracker())
		return mt, nil
	}

	// Initialize PostHog backend if enabled
	if cfg.PostHog.Enabled {
		phCfg := posthog.Config{
			Enabled:       cfg.PostHog.Enabled,
			APIKey:        cfg.PostHog.APIKey,
			Host:          cfg.PostHog.Host,
			Debug:         cfg.PostHog.Debug,
			BatchSize:     cfg.PostHog.BatchSize,
			FlushInterval: cfg.PostHog.FlushInterval,
		}
		phClient, err := posthog.New(phCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create PostHog client: %w", err)
		}
		mt.backends = append(mt.backends, &posthogAdapter{client: phClient})
	}

	// Initialize OTel backend if enabled
	if cfg.OTel.Enabled {
		otelCfg := otel.Config{
			Enabled:     cfg.OTel.Enabled,
			Endpoint:    cfg.OTel.Endpoint,
			Headers:     cfg.OTel.Headers,
			MetricsPort: cfg.OTel.MetricsPort,
		}
		otelBackend, err := otel.New(otelCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTel backend: %w", err)
		}
		mt.backends = append(mt.backends, &otelAdapter{backend: otelBackend})
	}

	// If no backends were enabled, use no-op
	if len(mt.backends) == 0 {
		mt.backends = append(mt.backends, NewNoopTracker())
	}

	return mt, nil
}

// MustNewTracker creates a new tracker or panics on error
func MustNewTracker(cfg Config, sanitizer *Sanitizer) Tracker {
	tracker, err := NewTracker(cfg, sanitizer)
	if err != nil {
		panic(fmt.Sprintf("failed to create analytics tracker: %v", err))
	}
	return tracker
}

// posthogAdapter adapts posthog.Client to analytics.Tracker
type posthogAdapter struct {
	client *posthog.Client
}

func (a *posthogAdapter) Track(ctx context.Context, event Event) error {
	phEvent := posthog.Event{
		Name:       string(event.Name),
		DistinctID: event.DistinctID,
		Timestamp:  event.Timestamp,
		Properties: event.Properties,
	}
	return a.client.Track(ctx, phEvent)
}

func (a *posthogAdapter) Identify(ctx context.Context, distinctID string, props UserProperties) error {
	phProps := posthog.UserProperties{
		Email:     props.Email,
		Name:      props.Name,
		CreatedAt: props.CreatedAt,
		Plan:      props.Plan,
		Platform:  props.Platform,
		Custom:    props.Custom,
	}
	return a.client.Identify(ctx, distinctID, phProps)
}

func (a *posthogAdapter) Flush(ctx context.Context) error {
	return a.client.Flush(ctx)
}

func (a *posthogAdapter) Shutdown(ctx context.Context) error {
	return a.client.Shutdown(ctx)
}

// otelAdapter adapts otel.MetricsBackend to analytics.Tracker
type otelAdapter struct {
	backend *otel.MetricsBackend
}

func (a *otelAdapter) Track(ctx context.Context, event Event) error {
	otelEvent := otel.Event{
		Name:       string(event.Name),
		DistinctID: event.DistinctID,
		Timestamp:  event.Timestamp,
		Properties: event.Properties,
	}
	return a.backend.Track(ctx, otelEvent)
}

func (a *otelAdapter) Identify(ctx context.Context, distinctID string, props UserProperties) error {
	otelProps := otel.UserProperties{
		Email:     props.Email,
		Name:      props.Name,
		CreatedAt: props.CreatedAt,
		Plan:      props.Plan,
		Platform:  props.Platform,
		Custom:    props.Custom,
	}
	return a.backend.Identify(ctx, distinctID, otelProps)
}

func (a *otelAdapter) Flush(ctx context.Context) error {
	return a.backend.Flush(ctx)
}

func (a *otelAdapter) Shutdown(ctx context.Context) error {
	return a.backend.Shutdown(ctx)
}

// Ensure adapters implement Tracker
var _ Tracker = (*posthogAdapter)(nil)
var _ Tracker = (*otelAdapter)(nil)
