package sample

import "context"

// Repository exposes data access for Sample entities.
type Repository interface {
	FetchLatest(ctx context.Context) (Sample, error)
}
