package noop

import (
	"context"
	"time"
)

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

// Tracker is a no-op implementation
// It silently discards all events, useful for testing or when analytics is disabled
type Tracker struct{}

// New creates a new no-op tracker
func New() *Tracker {
	return &Tracker{}
}

// Track does nothing and returns nil
func (t *Tracker) Track(ctx context.Context, event Event) error {
	return nil
}

// Identify does nothing and returns nil
func (t *Tracker) Identify(ctx context.Context, distinctID string, props UserProperties) error {
	return nil
}

// Flush does nothing and returns nil
func (t *Tracker) Flush(ctx context.Context) error {
	return nil
}

// Shutdown does nothing and returns nil
func (t *Tracker) Shutdown(ctx context.Context) error {
	return nil
}
