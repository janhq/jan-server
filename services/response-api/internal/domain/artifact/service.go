package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Service defines the interface for artifact business logic.
type Service interface {
	// Create creates a new artifact.
	Create(ctx context.Context, params CreateParams) (*Artifact, error)

	// CreateVersion creates a new version of an existing artifact.
	CreateVersion(ctx context.Context, artifactID string, params UpdateParams) (*Artifact, error)

	// GetByID retrieves an artifact by ID.
	GetByID(ctx context.Context, id string) (*Artifact, error)

	// GetLatestByResponseID gets the latest artifact for a response.
	GetLatestByResponseID(ctx context.Context, responseID string) (*Artifact, error)

	// GetLatestByPlanID gets the latest artifact for a plan.
	GetLatestByPlanID(ctx context.Context, planID string) (*Artifact, error)

	// GetVersions retrieves all versions of an artifact.
	GetVersions(ctx context.Context, artifactID string) ([]*Artifact, error)

	// UpdateMetadata updates artifact metadata.
	UpdateMetadata(ctx context.Context, id string, metadata json.RawMessage) error

	// SetRetention updates the retention policy.
	SetRetention(ctx context.Context, id string, policy RetentionPolicy, expiresAt *time.Time) error

	// Delete removes an artifact.
	Delete(ctx context.Context, id string) error

	// CleanupExpired removes expired artifacts.
	CleanupExpired(ctx context.Context) (int64, error)

	// List retrieves artifacts matching the filter.
	List(ctx context.Context, filter *Filter) ([]*Artifact, int64, error)
}

// CreateParams contains parameters for creating a new artifact.
type CreateParams struct {
	ResponseID      string
	PlanID          *string
	ContentType     ContentType
	MimeType        *string // Optional, defaults based on ContentType
	Title           string
	Content         *string // For inline content
	StoragePath     *string // For file-based content
	SizeBytes       int64
	RetentionPolicy RetentionPolicy
	Metadata        json.RawMessage
	ExpiresAt       *time.Time
}

// UpdateParams contains parameters for updating/versioning an artifact.
type UpdateParams struct {
	Title       *string
	Content     *string
	StoragePath *string
	SizeBytes   *int64
	Metadata    json.RawMessage
}

// DefaultService implements the Service interface.
type DefaultService struct {
	repo Repository
}

// NewService creates a new artifact service.
func NewService(repo Repository) Service {
	return &DefaultService{repo: repo}
}

// Create creates a new artifact.
func (s *DefaultService) Create(ctx context.Context, params CreateParams) (*Artifact, error) {
	// Validate content
	if params.Content == nil && params.StoragePath == nil {
		return nil, fmt.Errorf("artifact must have either content or storage_path")
	}

	// Determine MIME type
	mimeType := params.ContentType.MimeTypeFor()
	if params.MimeType != nil {
		mimeType = *params.MimeType
	}

	artifact := &Artifact{
		ResponseID:      params.ResponseID,
		PlanID:          params.PlanID,
		ContentType:     params.ContentType,
		MimeType:        mimeType,
		Title:           params.Title,
		Content:         params.Content,
		StoragePath:     params.StoragePath,
		SizeBytes:       params.SizeBytes,
		Version:         1,
		IsLatest:        true,
		RetentionPolicy: params.RetentionPolicy,
		Metadata:        params.Metadata,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
		ExpiresAt:       params.ExpiresAt,
	}

	if err := s.repo.Create(ctx, artifact); err != nil {
		return nil, err
	}

	return artifact, nil
}

// CreateVersion creates a new version of an existing artifact.
func (s *DefaultService) CreateVersion(ctx context.Context, artifactID string, params UpdateParams) (*Artifact, error) {
	existing, err := s.repo.FindByID(ctx, artifactID)
	if err != nil {
		return nil, err
	}

	// Create new version
	newArtifact := existing.CreateNextVersion()

	// Apply updates
	if params.Title != nil {
		newArtifact.Title = *params.Title
	}
	if params.Content != nil {
		newArtifact.Content = params.Content
	}
	if params.StoragePath != nil {
		newArtifact.StoragePath = params.StoragePath
	}
	if params.SizeBytes != nil {
		newArtifact.SizeBytes = *params.SizeBytes
	}
	if params.Metadata != nil {
		newArtifact.Metadata = params.Metadata
	}

	// Save new version
	if err := s.repo.Create(ctx, newArtifact); err != nil {
		return nil, err
	}

	// Mark old versions as not latest
	if err := s.repo.MarkOldVersionsNotLatest(ctx, newArtifact.ID, artifactID); err != nil {
		return nil, err
	}

	return newArtifact, nil
}

// GetByID retrieves an artifact by ID.
func (s *DefaultService) GetByID(ctx context.Context, id string) (*Artifact, error) {
	return s.repo.FindByID(ctx, id)
}

// GetLatestByResponseID gets the latest artifact for a response.
func (s *DefaultService) GetLatestByResponseID(ctx context.Context, responseID string) (*Artifact, error) {
	return s.repo.FindLatestByResponseID(ctx, responseID)
}

// GetLatestByPlanID gets the latest artifact for a plan.
func (s *DefaultService) GetLatestByPlanID(ctx context.Context, planID string) (*Artifact, error) {
	return s.repo.FindLatestByPlanID(ctx, planID)
}

// GetVersions retrieves all versions of an artifact.
func (s *DefaultService) GetVersions(ctx context.Context, artifactID string) ([]*Artifact, error) {
	return s.repo.ListVersions(ctx, artifactID)
}

// UpdateMetadata updates artifact metadata.
func (s *DefaultService) UpdateMetadata(ctx context.Context, id string, metadata json.RawMessage) error {
	artifact, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	artifact.Metadata = metadata
	artifact.UpdatedAt = time.Now().UTC()

	return s.repo.Update(ctx, artifact)
}

// SetRetention updates the retention policy.
func (s *DefaultService) SetRetention(ctx context.Context, id string, policy RetentionPolicy, expiresAt *time.Time) error {
	artifact, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	artifact.RetentionPolicy = policy
	artifact.ExpiresAt = expiresAt
	artifact.UpdatedAt = time.Now().UTC()

	return s.repo.Update(ctx, artifact)
}

// Delete removes an artifact.
func (s *DefaultService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// CleanupExpired removes expired artifacts.
func (s *DefaultService) CleanupExpired(ctx context.Context) (int64, error) {
	return s.repo.DeleteExpired(ctx)
}

// List retrieves artifacts matching the filter.
func (s *DefaultService) List(ctx context.Context, filter *Filter) ([]*Artifact, int64, error) {
	return s.repo.List(ctx, filter)
}
