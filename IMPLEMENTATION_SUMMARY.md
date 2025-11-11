# Project Feature Implementation Summary

## Completed ✅

1. **Domain Layer** - `services/llm-api/internal/domain/project/`
   - ✅ `project.go` - Project entity, repository interface, NewProject factory
   - ✅ `project_validation.go` - Validation logic (name ≤ 120 chars, instruction ≤ 32k)
   - ✅ `project_service.go` - Business logic service

2. **Database Schema** - `services/llm-api/internal/infrastructure/database/dbschema/`
   - ✅ `project.go` - GORM schema with EtoD/DtoE converters, auto-migration registered
   - ✅ `conversation.go` - Updated with ProjectID, InstructionVersion, EffectiveInstructionSnapshot

3. **Repository Layer** - `services/llm-api/internal/infrastructure/database/repository/projectrepo/`
   - ✅ `project_repository.go` - CRUD operations with soft-delete, pagination support

4. **HTTP DTOs**
   - ✅ `requests/projectreq/requests.go` - CreateProjectRequest, UpdateProjectRequest
   - ✅ `responses/projectres/responses.go` - ProjectResponse, ProjectListResponse, ProjectDeletedResponse

5. **Conversation Domain Updates**
   - ✅ Added ProjectID, InstructionVersion, EffectiveInstructionSnapshot fields
   - ✅ Updated NewConversation to initialize instruction fields

## Remaining Work 🚧

### 1. Project HTTP Handler
**File**: `services/llm-api/internal/interfaces/httpserver/handlers/projecthandler/project_handler.go`

```go
package projecthandler

import (
	"context"
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
	if requestedLimit != nil && len(projects) > *requestedLimit {
		hasMore = true
		projects = projects[:*requestedLimit]
	}

	return projectres.NewProjectListResponse(projects, hasMore, total), nil
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
```

### 2. Project Routes
**File**: `services/llm-api/internal/interfaces/httpserver/routes/v1/llm/projects/routes.go`

```go
package projects

import (
	"github.com/gin-gonic/gin"

	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/projecthandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/requests"
	"jan-server/services/llm-api/internal/interfaces/httpserver/requests/projectreq"
	"jan-server/services/llm-api/internal/interfaces/httpserver/responses"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

type ProjectRoute struct {
	handler *projecthandler.ProjectHandler
}

func NewProjectRoute(handler *projecthandler.ProjectHandler) *ProjectRoute {
	return &ProjectRoute{handler: handler}
}

// RegisterRoutes registers project routes
func (r *ProjectRoute) RegisterRoutes(rg *gin.RouterGroup) {
	projects := rg.Group("/projects")
	{
		projects.POST("", r.createProject)
		projects.GET("", r.listProjects)
		projects.GET("/:project_id", r.getProject)
		projects.PATCH("/:project_id", r.updateProject)
		projects.DELETE("/:project_id", r.deleteProject)
	}
}

// createProject godoc
// @Summary Create project
// @Description Create a new project for grouping conversations
// @Tags Projects API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body projectreq.CreateProjectRequest true "Create project request"
// @Success 201 {object} projectres.ProjectResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/llm/projects [post]
func (r *ProjectRoute) createProject(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "proj-create-001")
		return
	}

	var req projectreq.CreateProjectRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "invalid request body", "proj-create-002")
		return
	}

	response, err := r.handler.CreateProject(ctx, user.ID, req)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to create project")
		return
	}

	reqCtx.JSON(201, response)
}

// listProjects godoc
// @Summary List projects
// @Description List all projects for the authenticated user
// @Tags Projects API
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Maximum number of projects to return"
// @Param after query string false "Return projects after the given numeric ID"
// @Param order query string false "Sort order (asc or desc)"
// @Success 200 {object} projectres.ProjectListResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/llm/projects [get]
func (r *ProjectRoute) listProjects(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "proj-list-001")
		return
	}

	pagination, err := requests.GetCursorPaginationFromQuery(reqCtx, nil)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to process pagination")
		return
	}

	response, err := r.handler.ListProjects(ctx, user.ID, pagination)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to list projects")
		return
	}

	reqCtx.JSON(200, response)
}

// getProject godoc
// @Summary Get project
// @Description Get a single project by ID
// @Tags Projects API
// @Security BearerAuth
// @Produce json
// @Param project_id path string true "Project ID"
// @Success 200 {object} projectres.ProjectResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/llm/projects/{project_id} [get]
func (r *ProjectRoute) getProject(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "proj-get-001")
		return
	}

	projectID := reqCtx.Param("project_id")

	response, err := r.handler.GetProject(ctx, user.ID, projectID)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to get project")
		return
	}

	reqCtx.JSON(200, response)
}

// updateProject godoc
// @Summary Update project
// @Description Update project name, instruction, or archived status
// @Tags Projects API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project_id path string true "Project ID"
// @Param request body projectreq.UpdateProjectRequest true "Update request"
// @Success 200 {object} projectres.ProjectResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/llm/projects/{project_id} [patch]
func (r *ProjectRoute) updateProject(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "proj-update-001")
		return
	}

	projectID := reqCtx.Param("project_id")

	var req projectreq.UpdateProjectRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "invalid request body", "proj-update-002")
		return
	}

	response, err := r.handler.UpdateProject(ctx, user.ID, projectID, req)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to update project")
		return
	}

	reqCtx.JSON(200, response)
}

// deleteProject godoc
// @Summary Delete project
// @Description Soft-delete a project
// @Tags Projects API
// @Security BearerAuth
// @Produce json
// @Param project_id path string true "Project ID"
// @Success 200 {object} projectres.ProjectDeletedResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/llm/projects/{project_id} [delete]
func (r *ProjectRoute) deleteProject(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "proj-delete-001")
		return
	}

	projectID := reqCtx.Param("project_id")

	response, err := r.handler.DeleteProject(ctx, user.ID, projectID)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to delete project")
		return
	}

	reqCtx.JSON(200, response)
}
```

### 3. Update Conversation Requests for Project ID
**File**: `services/llm-api/internal/interfaces/httpserver/requests/conversation/conversation.go`

Add to CreateConversationRequest:
```go
ProjectID *string `json:"project_id,omitempty"`
```

Add to UpdateConversationRequest:
```go
ProjectID *string `json:"project_id,omitempty"`
```

### 4. Update Conversation Handler
Update `conversation_handler.go` CreateConversation to handle project_id

### 5. Wire Providers

**Update**: `services/llm-api/internal/domain/provider.go`
```go
import "jan-server/services/llm-api/internal/domain/project"

var ServiceProvider = wire.NewSet(
	// ... existing services ...
	project.NewProjectService,
)
```

**Update**: `services/llm-api/internal/infrastructure/database/provider.go`
```go
import "jan-server/services/llm-api/internal/infrastructure/database/repository/projectrepo"

var InfrastructureProvider = wire.NewSet(
	// ... existing repos ...
	projectrepo.NewProjectGormRepository,
	wire.Bind(new(project.ProjectRepository), new(*projectrepo.ProjectGormRepository)),
)
```

**Update**: `services/llm-api/internal/interfaces/httpserver/routes/provider.go`
```go
import (
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/projecthandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1/llm/projects"
)

var RouteProvider = wire.NewSet(
	// ... existing handlers/routes ...
	projecthandler.NewProjectHandler,
	projects.NewProjectRoute,
)
```

**Update**: `services/llm-api/internal/interfaces/httpserver/routes/v1/llm/routes.go`
Register project routes in LLM group

### 6. Database Migration
**File**: `services/llm-api/migrations/000X_create_projects.up.sql`

```sql
-- Create projects table
CREATE TABLE IF NOT EXISTS projects (
    id BIGSERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    instruction TEXT,
    favorite BOOLEAN NOT NULL DEFAULT false,
    archived_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_projects_user_name UNIQUE (user_id, name)
);

CREATE INDEX idx_projects_user_id ON projects(user_id);
CREATE INDEX idx_projects_user_updated_at ON projects(user_id, updated_at DESC);
CREATE INDEX idx_projects_deleted_at ON projects(deleted_at);
CREATE INDEX idx_projects_archived_at ON projects(archived_at);

-- Update conversations table
ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS project_id BIGINT REFERENCES projects(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS instruction_version INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS effective_instruction_snapshot TEXT;

CREATE INDEX IF NOT EXISTS idx_conversations_project_updated_at 
    ON conversations(project_id, updated_at DESC);

COMMENT ON COLUMN conversations.project_id IS 'Optional project grouping';
COMMENT ON COLUMN conversations.instruction_version IS 'Version of project instruction when conversation was created';
COMMENT ON COLUMN conversations.effective_instruction_snapshot IS 'Snapshot of merged instruction for reproducibility';
```

**File**: `services/llm-api/migrations/000X_create_projects.down.sql`

```sql
-- Remove indexes
DROP INDEX IF EXISTS idx_conversations_project_updated_at;
DROP INDEX IF EXISTS idx_projects_archived_at;
DROP INDEX IF EXISTS idx_projects_deleted_at;
DROP INDEX IF EXISTS idx_projects_user_updated_at;
DROP INDEX IF EXISTS idx_projects_user_id;

-- Remove columns from conversations
ALTER TABLE conversations
    DROP COLUMN IF EXISTS effective_instruction_snapshot,
    DROP COLUMN IF EXISTS instruction_version,
    DROP COLUMN IF EXISTS project_id;

-- Drop projects table
DROP TABLE IF EXISTS projects;
```

## Next Steps

1. Create the handler file
2. Create the routes file  
3. Update conversation requests/handler
4. Update Wire providers (domain, infrastructure, routes)
5. Create migration files
6. Run `make wire` to generate dependency injection code
7. Run migration to create tables
8. Run `make doc` to update Swagger
9. Test with Postman collection

## Testing Required

- Create project
- List projects with pagination
- Get single project
- Update project (name, instruction, archived, favorite)
- Delete project (soft-delete)
- Create conversation with project_id
- List conversations by project_id
- Move conversation between projects
- Delete project cascades to conversations (set project_id to NULL)
