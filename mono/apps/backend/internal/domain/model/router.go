package model

import "errors"

var (
	ErrNoEndpoints        = errors.New("no endpoints available")
	ErrNoHealthyEndpoints = errors.New("no healthy endpoints available")
)

// EndpointRouter selects the next endpoint for a request.
//
// Implementations must be safe for concurrent use by multiple goroutines.
// Calls to NextEndpoint and Reset may occur concurrently without causing
// data races or corrupting internal state.
type EndpointRouter interface {
	// NextEndpoint returns the next endpoint URL for the given provider.
	//
	// The endpoints slice must not be mutated by the implementation.
	//
	// Error semantics:
	//   - ErrNoEndpoints: the provided endpoints slice is empty
	//   - ErrNoHealthyEndpoints: none of the endpoints are healthy (may return fallback URL)
	//
	// Thread-safe: may be called from multiple goroutines concurrently.
	NextEndpoint(providerID string, endpoints EndpointList) (string, error)

	// Reset clears any internal router state (counters, health cache, etc.).
	// Thread-safe: may be called concurrently with NextEndpoint.
	Reset()
}
