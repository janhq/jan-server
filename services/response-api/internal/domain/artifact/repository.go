package artifact

import "context"

// Repository defines the interface for artifact persistence.
type Repository interface {
	// Create persists a new artifact.
	Create(ctx context.Context, artifact *Artifact) error

	// Update updates an existing artifact.
	Update(ctx context.Context, artifact *Artifact) error

	// FindByID retrieves an artifact by ID.
	FindByID(ctx context.Context, id string) (*Artifact, error)

	// FindLatestByResponseID finds the latest artifact for a response.
	FindLatestByResponseID(ctx context.Context, responseID string) (*Artifact, error)

	// FindLatestByPlanID finds the latest artifact for a plan.
	FindLatestByPlanID(ctx context.Context, planID string) (*Artifact, error)

	// List retrieves artifacts matching the filter.
	List(ctx context.Context, filter *Filter) ([]*Artifact, int64, error)

	// ListVersions retrieves all versions of an artifact.
	ListVersions(ctx context.Context, artifactID string) ([]*Artifact, error)

	// Delete removes an artifact.
	Delete(ctx context.Context, id string) error

	// DeleteExpired removes all expired artifacts.
	DeleteExpired(ctx context.Context) (int64, error)

	// MarkOldVersionsNotLatest marks old versions as not latest when a new version is created.
	MarkOldVersionsNotLatest(ctx context.Context, newVersionID string, parentID string) error
}
