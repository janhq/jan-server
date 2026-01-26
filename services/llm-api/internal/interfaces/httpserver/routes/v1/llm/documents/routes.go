package documents

import (
	"github.com/gin-gonic/gin"

	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/documenthandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/requests"
	"jan-server/services/llm-api/internal/interfaces/httpserver/responses"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

// DocumentRoute handles document-related routes
type DocumentRoute struct {
	handler     *documenthandler.DocumentHandler
	authHandler *authhandler.AuthHandler
}

// NewDocumentRoute creates a new document route
func NewDocumentRoute(handler *documenthandler.DocumentHandler, authHandler *authhandler.AuthHandler) *DocumentRoute {
	return &DocumentRoute{
		handler:     handler,
		authHandler: authHandler,
	}
}

// RegisterRoutes registers document routes
func (r *DocumentRoute) RegisterRoutes(rg *gin.RouterGroup) {
	// Document endpoints
	docs := rg.Group("/documents")
	docs.POST("/scan", r.authHandler.WithAppUserAuthChain(r.scanDocument)...)
	docs.GET("", r.authHandler.WithAppUserAuthChain(r.listDocuments)...)
	docs.GET("/:document_id", r.authHandler.WithAppUserAuthChain(r.getDocument)...)
	docs.GET("/:document_id/content", r.authHandler.WithAppUserAuthChain(r.getDocumentContent)...)

	// Project files endpoints (nested under projects)
	projectFiles := rg.Group("/projects/:project_id/files")
	projectFiles.POST("", r.authHandler.WithAppUserAuthChain(r.createProjectFile)...)
	projectFiles.GET("", r.authHandler.WithAppUserAuthChain(r.listProjectFiles)...)
	projectFiles.GET("/:file_id", r.authHandler.WithAppUserAuthChain(r.getProjectFile)...)
	projectFiles.DELETE("/:file_id", r.authHandler.WithAppUserAuthChain(r.deleteProjectFile)...)
	projectFiles.PATCH("/reorder", r.authHandler.WithAppUserAuthChain(r.reorderProjectFiles)...)
}

// scanDocument godoc
// @Summary Scan document for OCR
// @Description Upload and scan a document to extract text using OCR
// @Tags Documents API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body documenthandler.ScanDocumentRequest true "Scan document request"
// @Success 200 {object} documenthandler.ScanDocumentResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/documents/scan [post]
func (r *DocumentRoute) scanDocument(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "doc-scan-001")
		return
	}

	var req documenthandler.ScanDocumentRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "invalid request body", "doc-scan-002")
		return
	}

	// Get auth header to pass to media client
	authHeader := reqCtx.GetHeader("Authorization")

	response, err := r.handler.ScanDocument(ctx, user.ID, req, authHeader)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to scan document")
		return
	}

	reqCtx.JSON(200, response)
}

// listDocuments godoc
// @Summary List documents
// @Description List all scanned documents for the authenticated user
// @Tags Documents API
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Maximum number of documents to return"
// @Param after query string false "Return documents after the given ID"
// @Success 200 {object} documenthandler.DocumentListResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/documents [get]
func (r *DocumentRoute) listDocuments(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "doc-list-001")
		return
	}

	pagination, err := requests.GetCursorPaginationFromQuery(reqCtx, nil)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to process pagination")
		return
	}

	response, err := r.handler.ListDocuments(ctx, user.ID, pagination)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to list documents")
		return
	}

	reqCtx.JSON(200, response)
}

// getDocument godoc
// @Summary Get document
// @Description Get a single document by ID
// @Tags Documents API
// @Security BearerAuth
// @Produce json
// @Param document_id path string true "Document ID"
// @Success 200 {object} documenthandler.ScanDocumentResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/documents/{document_id} [get]
func (r *DocumentRoute) getDocument(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "doc-get-001")
		return
	}

	documentID := reqCtx.Param("document_id")

	response, err := r.handler.GetDocumentContent(ctx, user.ID, documentID)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to get document")
		return
	}

	reqCtx.JSON(200, response)
}

// getDocumentContent godoc
// @Summary Get document text content
// @Description Get only the extracted text of a document
// @Tags Documents API
// @Security BearerAuth
// @Produce json
// @Param document_id path string true "Document ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/documents/{document_id}/content [get]
func (r *DocumentRoute) getDocumentContent(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "doc-content-001")
		return
	}

	documentID := reqCtx.Param("document_id")

	text, err := r.handler.GetDocumentText(ctx, user.ID, documentID)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to get document content")
		return
	}

	reqCtx.JSON(200, gin.H{"text": text})
}

// createProjectFile godoc
// @Summary Add file to project
// @Description Upload and attach a file to a project
// @Tags Project Files API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project_id path string true "Project ID"
// @Param request body documenthandler.CreateProjectFileRequest true "Create project file request"
// @Success 201 {object} documenthandler.ProjectFileResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/projects/{project_id}/files [post]
func (r *DocumentRoute) createProjectFile(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "pf-create-001")
		return
	}

	projectID := reqCtx.Param("project_id")

	var req documenthandler.CreateProjectFileRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "invalid request body", "pf-create-002")
		return
	}

	// Get auth header to pass to media client
	authHeader := reqCtx.GetHeader("Authorization")

	response, err := r.handler.CreateProjectFile(ctx, user.ID, projectID, req, authHeader)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to create project file")
		return
	}

	reqCtx.JSON(201, response)
}

// listProjectFiles godoc
// @Summary List project files
// @Description List all files attached to a project
// @Tags Project Files API
// @Security BearerAuth
// @Produce json
// @Param project_id path string true "Project ID"
// @Param limit query int false "Maximum number of files to return"
// @Param after query string false "Return files after the given ID"
// @Success 200 {object} documenthandler.ProjectFileListResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/projects/{project_id}/files [get]
func (r *DocumentRoute) listProjectFiles(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "pf-list-001")
		return
	}

	projectID := reqCtx.Param("project_id")

	pagination, err := requests.GetCursorPaginationFromQuery(reqCtx, nil)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to process pagination")
		return
	}

	response, err := r.handler.ListProjectFiles(ctx, user.ID, projectID, pagination)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to list project files")
		return
	}

	reqCtx.JSON(200, response)
}

// getProjectFile godoc
// @Summary Get project file
// @Description Get a single file from a project
// @Tags Project Files API
// @Security BearerAuth
// @Produce json
// @Param project_id path string true "Project ID"
// @Param file_id path string true "File ID"
// @Success 200 {object} documenthandler.ProjectFileResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/projects/{project_id}/files/{file_id} [get]
func (r *DocumentRoute) getProjectFile(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "pf-get-001")
		return
	}

	projectID := reqCtx.Param("project_id")
	fileID := reqCtx.Param("file_id")

	response, err := r.handler.GetProjectFile(ctx, user.ID, projectID, fileID)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to get project file")
		return
	}

	reqCtx.JSON(200, response)
}

// deleteProjectFile godoc
// @Summary Delete project file
// @Description Remove a file from a project
// @Tags Project Files API
// @Security BearerAuth
// @Produce json
// @Param project_id path string true "Project ID"
// @Param file_id path string true "File ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/projects/{project_id}/files/{file_id} [delete]
func (r *DocumentRoute) deleteProjectFile(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "pf-delete-001")
		return
	}

	projectID := reqCtx.Param("project_id")
	fileID := reqCtx.Param("file_id")

	err := r.handler.DeleteProjectFile(ctx, user.ID, projectID, fileID)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to delete project file")
		return
	}

	reqCtx.JSON(200, gin.H{
		"id":      fileID,
		"object":  "project_file.deleted",
		"deleted": true,
	})
}

// reorderProjectFiles godoc
// @Summary Reorder project files
// @Description Update the display order of files in a project
// @Tags Project Files API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project_id path string true "Project ID"
// @Param request body documenthandler.ReorderProjectFilesRequest true "Reorder request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/projects/{project_id}/files/reorder [patch]
func (r *DocumentRoute) reorderProjectFiles(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "pf-reorder-001")
		return
	}

	projectID := reqCtx.Param("project_id")

	var req documenthandler.ReorderProjectFilesRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "invalid request body", "pf-reorder-002")
		return
	}

	err := r.handler.ReorderProjectFiles(ctx, user.ID, projectID, req)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to reorder project files")
		return
	}

	reqCtx.JSON(200, gin.H{"success": true})
}
