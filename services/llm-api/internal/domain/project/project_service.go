package project

import (
	"context"

	"jan-server/services/llm-api/internal/domain/query"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

// ProjectService handles business logic for projects
type ProjectService struct {
	repo      ProjectRepository
	validator *ProjectValidator
}

// NewProjectService creates a new project service
func NewProjectService(repo ProjectRepository) *ProjectService {
	return &ProjectService{
		repo:      repo,
		validator: NewProjectValidator(nil), // Use default config
	}
}

// ===============================================
// Core CRUD Operations
// ===============================================

// CreateProject creates a project (core function - direct repository call)
func (s *ProjectService) CreateProject(ctx context.Context, proj *Project) (*Project, error) {
	// Validate project
	if err := s.validator.ValidateProject(proj); err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "project validation failed", err, "")
	}

	// Persist project
	if err := s.repo.Create(ctx, proj); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to create project")
	}

	return proj, nil
}

// GetProjectByPublicIDAndUserID retrieves a project by public ID and validates ownership (core function)
func (s *ProjectService) GetProjectByPublicIDAndUserID(ctx context.Context, publicID string, userID uint) (*Project, error) {
	// Validate project ID format
	if err := s.validator.ValidateProjectID(publicID); err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "invalid project ID", err, "")
	}

	// Retrieve project
	proj, err := s.repo.GetByPublicIDAndUserID(ctx, publicID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "project not found")
	}

	return proj, nil
}

// UpdateProject updates a project (core function - direct repository call)
func (s *ProjectService) UpdateProject(ctx context.Context, proj *Project) (*Project, error) {
	// Validate updated project
	if err := s.validator.ValidateProject(proj); err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "project validation failed", err, "")
	}

	// Persist changes
	if err := s.repo.Update(ctx, proj); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to update project")
	}

	return proj, nil
}

// DeleteProject deletes a project (core function - soft delete)
func (s *ProjectService) DeleteProject(ctx context.Context, publicID string, userID uint) error {
	// Verify ownership before deletion
	_, err := s.GetProjectByPublicIDAndUserID(ctx, publicID, userID)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, publicID); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to delete project")
	}
	return nil
}

// ListProjectsByUserID retrieves all projects for a user with pagination
func (s *ProjectService) ListProjectsByUserID(ctx context.Context, userID uint, pagination *query.Pagination) ([]*Project, int64, error) {
	// Get projects
	projects, total, err := s.repo.ListByUserID(ctx, userID, pagination)
	if err != nil {
		return nil, 0, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to list projects")
	}

	return projects, total, nil
}
