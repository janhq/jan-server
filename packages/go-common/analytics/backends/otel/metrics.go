package otel

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	meterName = "jan-analytics"

	// Property keys (duplicated to avoid circular import)
	propModel       = "model"
	propMode        = "mode"
	propPlatform    = "platform"
	propEnvironment = "environment"
	propUserStatus  = "user_status"
	propStatus      = "status"
	propProvider    = "provider"
	propToolName    = "tool_name"
	propAgentType   = "agent_type"
	propLatencyMs   = "latency_ms"
	propDurationMs  = "duration_ms"
	propTTFRMs      = "ttfr_ms"
	propTokensTotal = "tokens_total"
)

// Config holds OTel-specific configuration
type Config struct {
	Enabled     bool
	Endpoint    string
	Headers     string
	MetricsPort int
}

// DefaultConfig returns a default OTel configuration
func DefaultConfig() Config {
	return Config{
		Enabled:     false,
		MetricsPort: 8080,
	}
}

// Event represents a tracked event
type Event struct {
	Name       string
	DistinctID string
	Timestamp  time.Time
	Properties map[string]interface{}
}

// UserProperties for identify calls (not used in OTel but needed for interface)
type UserProperties struct {
	Email     string
	Name      string
	CreatedAt time.Time
	Plan      string
	Platform  string
	Custom    map[string]string
}

// MetricsBackend translates analytics events to OTel metrics
type MetricsBackend struct {
	meter       metric.Meter
	eventCounts metric.Int64Counter
	durations   metric.Float64Histogram
	tokens      metric.Int64Counter
	ttfr        metric.Float64Histogram
}

// New creates a new OTel metrics backend
func New(cfg Config) (*MetricsBackend, error) {
	meter := otel.Meter(meterName)

	eventCounts, err := meter.Int64Counter("jan.analytics.events",
		metric.WithDescription("Analytics events by type"),
		metric.WithUnit("{event}"))
	if err != nil {
		return nil, err
	}

	durations, err := meter.Float64Histogram("jan.analytics.duration_ms",
		metric.WithDescription("Event durations in milliseconds"),
		metric.WithUnit("ms"))
	if err != nil {
		return nil, err
	}

	tokens, err := meter.Int64Counter("jan.analytics.tokens",
		metric.WithDescription("Token usage"),
		metric.WithUnit("{token}"))
	if err != nil {
		return nil, err
	}

	ttfr, err := meter.Float64Histogram("jan.analytics.ttfr_ms",
		metric.WithDescription("Time to first response in milliseconds"),
		metric.WithUnit("ms"))
	if err != nil {
		return nil, err
	}

	return &MetricsBackend{
		meter:       meter,
		eventCounts: eventCounts,
		durations:   durations,
		tokens:      tokens,
		ttfr:        ttfr,
	}, nil
}

// Track converts an analytics event to OTel metrics
func (b *MetricsBackend) Track(ctx context.Context, event Event) error {
	attrs := b.extractAttributes(event)

	// Record event count
	b.eventCounts.Add(ctx, 1, metric.WithAttributes(attrs...))

	// Record duration if present
	if dur, ok := getInt64(event.Properties, propLatencyMs); ok {
		b.durations.Record(ctx, float64(dur), metric.WithAttributes(attrs...))
	}
	if dur, ok := getInt64(event.Properties, propDurationMs); ok {
		b.durations.Record(ctx, float64(dur), metric.WithAttributes(attrs...))
	}

	// Record TTFR if present
	if ttfr, ok := getInt64(event.Properties, propTTFRMs); ok {
		b.ttfr.Record(ctx, float64(ttfr), metric.WithAttributes(attrs...))
	}

	// Record tokens if present
	if tokens, ok := getInt64(event.Properties, propTokensTotal); ok {
		b.tokens.Add(ctx, tokens, metric.WithAttributes(attrs...))
	}

	return nil
}

// Identify is a no-op for OTel (user identification is not a metrics concept)
func (b *MetricsBackend) Identify(ctx context.Context, distinctID string, props UserProperties) error {
	// OTel doesn't have user identification - no-op
	return nil
}

// Flush is handled by the meter provider
func (b *MetricsBackend) Flush(ctx context.Context) error {
	return nil
}

// Shutdown is handled by the meter provider
func (b *MetricsBackend) Shutdown(ctx context.Context) error {
	return nil
}

// extractAttributes extracts common attributes from an event
func (b *MetricsBackend) extractAttributes(event Event) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("event", event.Name),
	}

	// Extract common attributes from properties
	if model, ok := getString(event.Properties, propModel); ok {
		attrs = append(attrs, attribute.String("model", model))
	}
	if mode, ok := getString(event.Properties, propMode); ok {
		attrs = append(attrs, attribute.String("mode", mode))
	}
	if platform, ok := getString(event.Properties, propPlatform); ok {
		attrs = append(attrs, attribute.String("platform", platform))
	}
	if environment, ok := getString(event.Properties, propEnvironment); ok {
		attrs = append(attrs, attribute.String("environment", environment))
	}
	if userStatus, ok := getString(event.Properties, propUserStatus); ok {
		attrs = append(attrs, attribute.String("user_status", userStatus))
	}
	if status, ok := getString(event.Properties, propStatus); ok {
		attrs = append(attrs, attribute.String("status", status))
	}
	if provider, ok := getString(event.Properties, propProvider); ok {
		attrs = append(attrs, attribute.String("provider", provider))
	}
	if toolName, ok := getString(event.Properties, propToolName); ok {
		attrs = append(attrs, attribute.String("tool_name", toolName))
	}
	if agentType, ok := getString(event.Properties, propAgentType); ok {
		attrs = append(attrs, attribute.String("agent_type", agentType))
	}

	return attrs
}

// getString safely extracts a string from properties
func getString(props map[string]interface{}, key string) (string, bool) {
	if props == nil {
		return "", false
	}
	if v, ok := props[key]; ok {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return "", false
}

// getInt64 safely extracts an int64 from properties
func getInt64(props map[string]interface{}, key string) (int64, bool) {
	if props == nil {
		return 0, false
	}
	if v, ok := props[key]; ok {
		switch val := v.(type) {
		case int64:
			return val, true
		case int:
			return int64(val), true
		case int32:
			return int64(val), true
		case float64:
			return int64(val), true
		}
	}
	return 0, false
}
