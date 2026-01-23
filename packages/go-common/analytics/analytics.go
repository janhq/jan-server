package analytics

import (
	"context"
	"errors"
	"sync"
)

// Tracker is the unified analytics interface
type Tracker interface {
	// Track records an event with properties
	Track(ctx context.Context, event Event) error

	// Identify associates user properties with a distinct ID
	Identify(ctx context.Context, distinctID string, props UserProperties) error

	// Flush ensures all queued events are sent
	Flush(ctx context.Context) error

	// Shutdown gracefully closes all backends
	Shutdown(ctx context.Context) error
}

// MultiTracker fans out events to multiple backends
type MultiTracker struct {
	backends  []Tracker
	sanitizer *Sanitizer
	config    Config
	mu        sync.RWMutex
}

// NewMultiTracker creates a new multi-backend tracker
func NewMultiTracker(cfg Config, sanitizer *Sanitizer) (*MultiTracker, error) {
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

	// Initialize enabled backends (will be added in subsequent implementations)
	// For now, we'll add them via RegisterBackend or return no-op if none configured

	if len(mt.backends) == 0 {
		mt.backends = append(mt.backends, NewNoopTracker())
	}

	return mt, nil
}

// RegisterBackend adds a backend to the multi-tracker
func (m *MultiTracker) RegisterBackend(backend Tracker) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove no-op if we're adding a real backend
	if len(m.backends) == 1 {
		if _, isNoop := m.backends[0].(*NoopTracker); isNoop {
			m.backends = m.backends[:0]
		}
	}

	m.backends = append(m.backends, backend)
}

// Track sends an event to all backends
func (m *MultiTracker) Track(ctx context.Context, event Event) error {
	if err := event.Validate(); err != nil {
		return err
	}

	m.mu.RLock()
	backends := m.backends
	m.mu.RUnlock()

	// Sanitize event properties if sanitizer is available
	if m.sanitizer != nil {
		event = m.sanitizeEvent(event)
	}

	// Add common properties
	event = m.enrichEvent(event)

	var errs []error
	for _, b := range backends {
		if err := b.Track(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// Identify sends user identification to all backends
func (m *MultiTracker) Identify(ctx context.Context, distinctID string, props UserProperties) error {
	if distinctID == "" {
		return ErrMissingDistinctID
	}

	m.mu.RLock()
	backends := m.backends
	m.mu.RUnlock()

	var errs []error
	for _, b := range backends {
		if err := b.Identify(ctx, distinctID, props); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// Flush flushes all backends
func (m *MultiTracker) Flush(ctx context.Context) error {
	m.mu.RLock()
	backends := m.backends
	m.mu.RUnlock()

	var errs []error
	for _, b := range backends {
		if err := b.Flush(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// Shutdown shuts down all backends
func (m *MultiTracker) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for _, b := range m.backends {
		if err := b.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	m.backends = nil
	return errors.Join(errs...)
}

// sanitizeEvent applies PII sanitization to event properties
func (m *MultiTracker) sanitizeEvent(event Event) Event {
	if event.Properties == nil {
		return event
	}

	sanitized := make(map[string]interface{}, len(event.Properties))
	for k, v := range event.Properties {
		switch val := v.(type) {
		case string:
			// Sanitize string values that might contain PII
			if isPIISensitiveKey(k) {
				sanitized[k] = m.sanitizer.SanitizePrompt(val)
			} else {
				sanitized[k] = val
			}
		default:
			sanitized[k] = v
		}
	}

	event.Properties = sanitized
	return event
}

// enrichEvent adds common properties to the event
func (m *MultiTracker) enrichEvent(event Event) Event {
	if event.Properties == nil {
		event.Properties = make(map[string]interface{})
	}

	// Add environment if not already set
	if _, ok := event.Properties[PropEnvironment]; !ok {
		event.Properties[PropEnvironment] = m.config.Environment
	}

	return event
}

// isPIISensitiveKey checks if a property key might contain PII
func isPIISensitiveKey(key string) bool {
	sensitiveKeys := map[string]bool{
		"email":        true,
		"name":         true,
		"user_name":    true,
		"phone":        true,
		"address":      true,
		"ip":           true,
		"ip_address":   true,
		"content":      true,
		"message":      true,
		"prompt":       true,
		"response":     true,
		"error_message": true,
	}
	return sensitiveKeys[key]
}

// NoopTracker is a no-op implementation of Tracker
type NoopTracker struct{}

// NewNoopTracker creates a new no-op tracker
func NewNoopTracker() *NoopTracker {
	return &NoopTracker{}
}

// Track does nothing
func (n *NoopTracker) Track(ctx context.Context, event Event) error {
	return nil
}

// Identify does nothing
func (n *NoopTracker) Identify(ctx context.Context, distinctID string, props UserProperties) error {
	return nil
}

// Flush does nothing
func (n *NoopTracker) Flush(ctx context.Context) error {
	return nil
}

// Shutdown does nothing
func (n *NoopTracker) Shutdown(ctx context.Context) error {
	return nil
}

// Ensure NoopTracker implements Tracker
var _ Tracker = (*NoopTracker)(nil)
var _ Tracker = (*MultiTracker)(nil)
