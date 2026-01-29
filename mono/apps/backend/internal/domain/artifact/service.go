package artifact

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
)

var (
	ErrArtifactNotFound = errors.New("artifact not found")
	ErrUnauthorized     = errors.New("unauthorized")
)

// Repository defines the interface for artifact data operations.
type Repository interface {
	Create(ctx context.Context, artifact *Artifact) error
	GetByID(ctx context.Context, id string) (*Artifact, error)
	GetByShareToken(ctx context.Context, token string) (*Artifact, error)
	Update(ctx context.Context, artifact *Artifact) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter ListArtifactsFilter) ([]*Artifact, int64, error)

	// Version operations
	CreateVersion(ctx context.Context, version *ArtifactVersion) error
	ListVersions(ctx context.Context, artifactID string) ([]*ArtifactVersion, error)
	GetVersion(ctx context.Context, artifactID string, version int) (*ArtifactVersion, error)
}

// Service handles artifact-related business logic.
type Service struct {
	repo Repository
}

// NewService creates a new artifact service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new artifact.
func (s *Service) Create(ctx context.Context, userID string, req CreateArtifactRequest) (*Artifact, error) {
	artifact := &Artifact{
		UserID:         userID,
		ConversationID: req.ConversationID,
		ResponseID:     req.ResponseID,
		Title:          req.Title,
		Description:    req.Description,
		Type:           req.Type,
		Language:       req.Language,
		Content:        req.Content,
		Version:        1,
		Metadata:       req.Metadata,
		IsPublic:       req.IsPublic,
	}

	if artifact.Title == "" {
		artifact.Title = "Untitled"
	}
	if artifact.Type == "" {
		artifact.Type = "code"
	}

	if err := s.repo.Create(ctx, artifact); err != nil {
		return nil, err
	}

	return artifact, nil
}

// GetByID retrieves an artifact by ID.
func (s *Service) GetByID(ctx context.Context, userID, id string) (*Artifact, error) {
	artifact, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrArtifactNotFound
	}

	// Check ownership or public access
	if artifact.UserID != userID && !artifact.IsPublic {
		return nil, ErrUnauthorized
	}

	return artifact, nil
}

// Update updates an artifact and creates a version backup.
func (s *Service) Update(ctx context.Context, userID, id string, req UpdateArtifactRequest) (*Artifact, error) {
	artifact, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrArtifactNotFound
	}

	if artifact.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Create version backup if content is changing
	if req.Content != nil && *req.Content != artifact.Content {
		version := &ArtifactVersion{
			ArtifactID: artifact.ID,
			Version:    artifact.Version,
			Content:    artifact.Content,
		}
		if err := s.repo.CreateVersion(ctx, version); err != nil {
			return nil, err
		}
		artifact.Version++
		artifact.Content = *req.Content
	}

	if req.Title != nil {
		artifact.Title = *req.Title
	}
	if req.Description != nil {
		artifact.Description = *req.Description
	}
	if req.Language != nil {
		artifact.Language = *req.Language
	}
	if req.IsPublic != nil {
		artifact.IsPublic = *req.IsPublic
	}
	if req.Metadata != nil {
		artifact.Metadata = req.Metadata
	}

	if err := s.repo.Update(ctx, artifact); err != nil {
		return nil, err
	}

	return artifact, nil
}

// Delete deletes an artifact.
func (s *Service) Delete(ctx context.Context, userID, id string) error {
	artifact, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrArtifactNotFound
	}

	if artifact.UserID != userID {
		return ErrUnauthorized
	}

	return s.repo.Delete(ctx, id)
}

// List lists artifacts for a user.
func (s *Service) List(ctx context.Context, filter ListArtifactsFilter) ([]*Artifact, int64, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	return s.repo.List(ctx, filter)
}

// ListVersions lists all versions of an artifact.
func (s *Service) ListVersions(ctx context.Context, userID, id string) ([]*ArtifactVersion, error) {
	artifact, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrArtifactNotFound
	}

	if artifact.UserID != userID {
		return nil, ErrUnauthorized
	}

	return s.repo.ListVersions(ctx, id)
}

// GetVersion retrieves a specific version of an artifact.
func (s *Service) GetVersion(ctx context.Context, userID, id string, version int) (*ArtifactVersion, error) {
	artifact, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrArtifactNotFound
	}

	if artifact.UserID != userID {
		return nil, ErrUnauthorized
	}

	return s.repo.GetVersion(ctx, id, version)
}

// Share generates a share token for an artifact.
func (s *Service) Share(ctx context.Context, userID, id string) (*Artifact, error) {
	artifact, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrArtifactNotFound
	}

	if artifact.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Generate share token
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)

	artifact.ShareToken = &token
	artifact.IsPublic = true

	if err := s.repo.Update(ctx, artifact); err != nil {
		return nil, err
	}

	return artifact, nil
}

// GetByShareToken retrieves an artifact by its share token.
func (s *Service) GetByShareToken(ctx context.Context, token string) (*Artifact, error) {
	return s.repo.GetByShareToken(ctx, token)
}
