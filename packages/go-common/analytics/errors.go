package analytics

import "errors"

var (
	// ErrMissingPostHogAPIKey is returned when PostHog is enabled but no API key is provided
	ErrMissingPostHogAPIKey = errors.New("analytics: PostHog enabled but POSTHOG_API_KEY is not set")

	// ErrTrackerNotInitialized is returned when tracking is attempted before initialization
	ErrTrackerNotInitialized = errors.New("analytics: tracker not initialized")

	// ErrInvalidEvent is returned when an event is malformed
	ErrInvalidEvent = errors.New("analytics: invalid event")

	// ErrMissingDistinctID is returned when an event is missing a distinct ID
	ErrMissingDistinctID = errors.New("analytics: missing distinct_id")
)
