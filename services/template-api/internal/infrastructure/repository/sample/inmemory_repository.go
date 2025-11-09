package sample

import (
	"context"
	"errors"
	"sync"

	domain "jan-server/services/template-api/internal/domain/sample"
)

// InMemoryRepository is a thread-safe repository useful for demos/tests.
type InMemoryRepository struct {
	mu      sync.RWMutex
	entries []domain.Sample
}

// NewInMemoryRepository seeds demo data.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		entries: []domain.Sample{
			{ID: "sample-1", Message: "Hello from the repository layer"},
			{ID: "sample-2", Message: "Customize this implementation for real data sources"},
		},
	}
}

// FetchLatest returns the most recent entry.
func (r *InMemoryRepository) FetchLatest(ctx context.Context) (domain.Sample, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.entries) == 0 {
		return domain.Sample{}, errors.New("no sample data available")
	}
	return r.entries[len(r.entries)-1], nil
}

// Store adds a new entry (optional helper if you extend the sample).
func (r *InMemoryRepository) Store(ctx context.Context, sample domain.Sample) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, sample)
}
