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
	handler     *projecthandler.ProjectHandler
	authHandler *authhandler.AuthHandler
}

func NewProjectRoute(handler *projecthandler.ProjectHandler, authHandler *authhandler.AuthHandler) *ProjectRoute {
	return &ProjectRoute{
		handler:     handler,
		authHandler: authHandler,
	}
}

// RegisterRoutes registers project routes
func (r *ProjectRoute) RegisterRoutes(rg *gin.RouterGroup) {
	projects := rg.Group("/projects")
	projects.POST("", r.authHandler.WithAppUserAuthChain(r.createProject)...)
	projects.GET("", r.authHandler.WithAppUserAuthChain(r.listProjects)...)
	projects.GET("/:project_id", r.authHandler.WithAppUserAuthChain(r.getProject)...)
	projects.PATCH("/:project_id", r.authHandler.WithAppUserAuthChain(r.updateProject)...)
	projects.DELETE("/:project_id", r.authHandler.WithAppUserAuthChain(r.deleteProject)...)
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
