package projecthandler

import (
	"context"
	"strconv"
	"strings"
	"time"

	"jan-server/services/llm-api/internal/domain/project"
	"jan-server/services/llm-api/internal/domain/query"
	"jan-server/services/llm-api/internal/interfaces/httpserver/requests/projectreq"
	"jan-server/services/llm-api/internal/interfaces/httpserver/responses/projectres"
	"jan-server/services/llm-api/internal/utils/idgen"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

type ProjectHandler struct {
	projectService *project.ProjectService
}

func NewProjectHandler(projectService *project.ProjectService) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
	}
}

// CreateProject creates a new project
func (h *ProjectHandler) CreateProject(
	ctx context.Context,
	userID uint,
	req projectreq.CreateProjectRequest,
) (*projectres.ProjectResponse, error) {
	// Trim and validate input
	req.Name = strings.TrimSpace(req.Name)
	if req.Instruction != nil {
		trimmed := strings.TrimSpace(*req.Instruction)
		req.Instruction = &trimmed
	}

	// Generate public ID
	publicID, err := idgen.GenerateSecureID("proj", 16)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to generate project ID")
	}

	// Create project entity
	proj := project.NewProject(publicID, userID, req.Name, req.Instruction)

	// Persist project
	proj, err = h.projectService.CreateProject(ctx, proj)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to create project")
	}

	return projectres.NewProjectResponse(proj), nil
}

// GetProject retrieves a single project
func (h *ProjectHandler) GetProject(
	ctx context.Context,
	userID uint,
	projectID string,
) (*projectres.ProjectResponse, error) {
	proj, err := h.projectService.GetProjectByPublicIDAndUserID(ctx, projectID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get project")
	}

	return projectres.NewProjectResponse(proj), nil
}

// ListProjects lists all projects for a user
func (h *ProjectHandler) ListProjects(
	ctx context.Context,
	userID uint,
	pagination *query.Pagination,
) (*projectres.ProjectListResponse, error) {
	// Fetch limit+1 to determine hasMore
	var requestedLimit *int
	if pagination != nil && pagination.Limit != nil {
		requestedLimit = pagination.Limit
		extraLimit := *pagination.Limit + 1
		pagination.Limit = &extraLimit
	}

	projects, total, err := h.projectService.ListProjectsByUserID(ctx, userID, pagination)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to list projects")
	}

	// Calculate hasMore
	hasMore := false
	var nextCursor *string
	if requestedLimit != nil && len(projects) > *requestedLimit {
		hasMore = true
		lastIndex := *requestedLimit - 1
		cursorValue := strconv.FormatUint(uint64(projects[lastIndex].ID), 10)
		nextCursor = &cursorValue
		projects = projects[:*requestedLimit]
	}

	return projectres.NewProjectListResponse(projects, hasMore, nextCursor, total), nil
}

// UpdateProject updates a project
func (h *ProjectHandler) UpdateProject(
	ctx context.Context,
	userID uint,
	projectID string,
	req projectreq.UpdateProjectRequest,
) (*projectres.ProjectResponse, error) {
	// Get existing project
	proj, err := h.projectService.GetProjectByPublicIDAndUserID(ctx, projectID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get project")
	}

	// Update fields
	if req.Name != nil {
		proj.Name = strings.TrimSpace(*req.Name)
	}
	if req.Instruction != nil {
		trimmed := strings.TrimSpace(*req.Instruction)
		proj.Instruction = &trimmed
	}
	if req.Favorite != nil {
		proj.Favorite = *req.Favorite
	}
	if req.Archived != nil {
		if *req.Archived {
			now := time.Now()
			proj.ArchivedAt = &now
		} else {
			proj.ArchivedAt = nil
		}
	}

	proj.UpdatedAt = time.Now()

	// Persist changes
	proj, err = h.projectService.UpdateProject(ctx, proj)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to update project")
	}

	return projectres.NewProjectResponse(proj), nil
}

// DeleteProject deletes a project
func (h *ProjectHandler) DeleteProject(
	ctx context.Context,
	userID uint,
	projectID string,
) (*projectres.ProjectDeletedResponse, error) {
	err := h.projectService.DeleteProject(ctx, projectID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to delete project")
	}

	return projectres.NewProjectDeletedResponse(projectID), nil
}
