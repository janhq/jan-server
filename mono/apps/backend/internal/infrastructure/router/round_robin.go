package router

import (
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"

	"jan-server/mono/apps/backend/internal/domain/model"
)

// RoundRobinRouter implements simple round-robin endpoint selection.
// Thread-safe with per-provider atomic counters.
type RoundRobinRouter struct {
	mu       sync.RWMutex
	counters map[string]*uint64
}

func NewRoundRobinRouter() *RoundRobinRouter {
	return &RoundRobinRouter{
		counters: make(map[string]*uint64),
	}
}

// NextEndpoint returns the next endpoint using round-robin selection.
func (r *RoundRobinRouter) NextEndpoint(providerID string, endpoints model.EndpointList) (string, error) {
	if len(endpoints) == 0 {
		return "", model.ErrNoEndpoints
	}

	if len(endpoints) == 1 {
		return endpoints[0].URL, nil
	}

	healthy := endpoints.GetHealthy()
	if len(healthy) == 0 {
		return endpoints[0].URL, model.ErrNoHealthyEndpoints
	}

	counter := r.getOrCreateCounter(providerID)
	idx := atomic.AddUint64(counter, 1) - 1
	selected := healthy[idx%uint64(len(healthy))]

	log.Info().
		Str("provider_id", providerID).
		Str("endpoint_url", selected.URL).
		Int("healthy_count", len(healthy)).
		Int("total_count", len(endpoints)).
		Uint64("round_robin_index", idx).
		Msg("selected endpoint for request")

	return selected.URL, nil
}

func (r *RoundRobinRouter) getOrCreateCounter(providerID string) *uint64 {
	r.mu.RLock()
	counter, ok := r.counters[providerID]
	r.mu.RUnlock()
	if ok {
		return counter
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if counter, ok = r.counters[providerID]; ok {
		return counter
	}
	var c uint64
	r.counters[providerID] = &c
	return &c
}

// Reset clears all counters.
func (r *RoundRobinRouter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters = make(map[string]*uint64)
}

// ResetProvider clears counter for a specific provider.
func (r *RoundRobinRouter) ResetProvider(providerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.counters, providerID)
}
