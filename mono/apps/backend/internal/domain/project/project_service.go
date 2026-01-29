package project

import (
	"context"

	"jan-server/mono/apps/backend/internal/domain/query"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
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
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "project validation failed", err, "f6a7b8c9-d0e1-4f2a-3b4c-5d6e7f8a9b0c")
	}

	// Check for duplicate name
	existingProject, err := s.repo.GetByNameAndUserID(ctx, proj.Name, proj.UserID)
	if err == nil && existingProject != nil {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerDomain,
			platformerrors.ErrorTypeConflict,
			"Project name already exists",
			nil,
			existingProject.PublicID,
		)
	}

	// Persist project
	if err := s.repo.Create(ctx, proj); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to create project")
	}

	return proj, nil
}

// GetProjectByPublicIDAndUserID retrieves a project by public ID and validates ownership (core function)
func (s *ProjectService) GetProjectByPublicIDAndUserID(ctx context.Context, publicID string, userID string) (*Project, error) {
	// Validate project ID format
	if err := s.validator.ValidateProjectID(publicID); err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "invalid project ID", err, "a7b8c9d0-e1f2-4a3b-4c5d-6e7f8a9b0c1d")
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
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "project validation failed", err, "b8c9d0e1-f2a3-4b4c-5d6e-7f8a9b0c1d2e")
	}

	// Check for duplicate name (but exclude self)
	existingProject, err := s.repo.GetByNameAndUserID(ctx, proj.Name, proj.UserID)
	if err == nil && existingProject != nil && existingProject.PublicID != proj.PublicID {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerDomain,
			platformerrors.ErrorTypeConflict,
			"Project name already exists",
			nil,
			existingProject.PublicID,
		)
	}

	// Persist changes
	if err := s.repo.Update(ctx, proj); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to update project")
	}

	return proj, nil
}

// DeleteProject deletes a project (core function - soft delete)
func (s *ProjectService) DeleteProject(ctx context.Context, publicID string, userID string) error {
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
func (s *ProjectService) ListProjectsByUserID(ctx context.Context, userID string, pagination *query.Pagination) ([]*Project, int64, error) {
	// Get projects
	projects, total, err := s.repo.ListByUserID(ctx, userID, pagination)
	if err != nil {
		return nil, 0, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to list projects")
	}

	return projects, total, nil
}
