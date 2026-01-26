package documenthandler

import (
	"context"
	"strconv"
	"strings"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/domain/document"
	domainmodel "jan-server/services/llm-api/internal/domain/model"
	"jan-server/services/llm-api/internal/domain/project"
	"jan-server/services/llm-api/internal/domain/query"
	"jan-server/services/llm-api/internal/infrastructure/inference"
	"jan-server/services/llm-api/internal/infrastructure/mediaclient"
	"jan-server/services/llm-api/internal/utils/idgen"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

// DocumentHandler handles document-related HTTP operations
type DocumentHandler struct {
	documentService    *document.DocumentService
	projectFileService *document.ProjectFileService
	projectService     *project.ProjectService
	providerService    *domainmodel.ProviderService
	ocrService         *inference.DocumentOCRService
	mediaClient        *mediaclient.Client
	config             *config.Config
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(
	documentService *document.DocumentService,
	projectFileService *document.ProjectFileService,
	projectService *project.ProjectService,
	providerService *domainmodel.ProviderService,
	ocrService *inference.DocumentOCRService,
	mediaClient *mediaclient.Client,
	cfg *config.Config,
) *DocumentHandler {
	return &DocumentHandler{
		documentService:    documentService,
		projectFileService: projectFileService,
		projectService:     projectService,
		providerService:    providerService,
		ocrService:         ocrService,
		mediaClient:        mediaClient,
		config:             cfg,
	}
}

// ScanDocumentRequest represents a request to scan a document
type ScanDocumentRequest struct {
	MediaObjectID string `json:"media_object_id" binding:"required"`
	Filename      string `json:"filename,omitempty"`
}

// ScanDocumentResponse represents the response from scanning a document
type ScanDocumentResponse struct {
	ID               string                    `json:"id"`
	Object           string                    `json:"object"`
	MediaObjectID    string                    `json:"media_object_id"`
	Filename         string                    `json:"filename,omitempty"`
	MimeType         string                    `json:"mime_type,omitempty"`
	ProcessingStatus document.ProcessingStatus `json:"processing_status"`
	ExtractedText    string                    `json:"extracted_text,omitempty"`
	PageCount        *int                      `json:"page_count,omitempty"`
	WordCount        *int                      `json:"word_count,omitempty"`
	ErrorMessage     *string                   `json:"error_message,omitempty"`
}

// ScanDocument performs OCR on a document
func (h *DocumentHandler) ScanDocument(ctx context.Context, userID uint, req ScanDocumentRequest, authHeader string) (*ScanDocumentResponse, error) {
	// Check if OCR is enabled
	if !h.documentService.IsOCREnabled() {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation,
			"document OCR is not enabled", nil, "doc-ocr-disabled")
	}

	// Check if document already exists for this media object
	existingDoc, err := h.documentService.GetDocumentContentByMediaObjectID(ctx, req.MediaObjectID, userID)
	if err == nil && existingDoc != nil {
		// Return existing document if already processed
		if existingDoc.ProcessingStatus == document.ProcessingStatusCompleted {
			return h.toScanResponse(existingDoc), nil
		}
	}

	// Resolve media object to get file URL and metadata
	mediaInfo, err := h.mediaClient.Resolve(ctx, req.MediaObjectID, authHeader)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to resolve media object")
	}

	// Validate mime type
	mimeType := mediaInfo.ContentType
	if !h.documentService.IsSupportedMimeType(mimeType) {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation,
			"unsupported document type: "+mimeType, nil, "doc-unsupported-type")
	}

	// Generate public ID for new document
	publicID, err := idgen.GenerateSecureID("doc", 16)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to generate document ID")
	}

	filename := req.Filename
	if filename == "" {
		filename = mediaInfo.Filename
	}

	// Create document content record
	doc := document.NewDocumentContent(publicID, userID, req.MediaObjectID, filename, mimeType, mediaInfo.Size)
	doc.MarkProcessing()

	doc, err = h.documentService.CreateDocumentContent(ctx, doc)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to create document content")
	}

	// Get OCR provider
	provider, err := h.getOCRProvider(ctx)
	if err != nil {
		doc.MarkFailed("no OCR provider available")
		h.documentService.UpdateDocumentContent(ctx, doc)
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get OCR provider")
	}

	// Fetch file data from media URL
	fileData, _, err := h.ocrService.FetchFileFromURL(ctx, mediaInfo.URL)
	if err != nil {
		doc.MarkFailed("failed to fetch file from media service")
		h.documentService.UpdateDocumentContent(ctx, doc)
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to fetch document file")
	}

	// Call OCR service
	ocrResp, err := h.ocrService.ScanWithFileData(ctx, provider, fileData, mimeType, filename)
	if err != nil {
		doc.MarkFailed(err.Error())
		h.documentService.UpdateDocumentContent(ctx, doc)
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "OCR processing failed")
	}

	// Update document with extracted text
	doc.MarkCompleted(ocrResp.Text, ocrResp.Model, ocrResp.PageCount, ocrResp.WordCount)
	doc, err = h.documentService.UpdateDocumentContent(ctx, doc)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to update document content")
	}

	return h.toScanResponse(doc), nil
}

// GetDocumentContent retrieves a document content by ID
func (h *DocumentHandler) GetDocumentContent(ctx context.Context, userID uint, publicID string) (*ScanDocumentResponse, error) {
	doc, err := h.documentService.GetDocumentContentByPublicID(ctx, publicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "document not found")
	}

	// Verify ownership
	if doc.UserID != userID {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound,
			"document not found", nil, "doc-not-found")
	}

	return h.toScanResponse(doc), nil
}

// GetDocumentText retrieves just the extracted text of a document
func (h *DocumentHandler) GetDocumentText(ctx context.Context, userID uint, publicID string) (string, error) {
	doc, err := h.documentService.GetDocumentContentByPublicID(ctx, publicID)
	if err != nil {
		return "", platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "document not found")
	}

	// Verify ownership
	if doc.UserID != userID {
		return "", platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound,
			"document not found", nil, "doc-not-found")
	}

	return doc.ExtractedText, nil
}

// ListDocuments lists all documents for a user
func (h *DocumentHandler) ListDocuments(ctx context.Context, userID uint, pagination *query.Pagination) (*DocumentListResponse, error) {
	docs, total, err := h.documentService.ListDocumentContents(ctx, userID, pagination)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to list documents")
	}

	items := make([]*ScanDocumentResponse, len(docs))
	for i, doc := range docs {
		items[i] = h.toScanResponse(doc)
	}

	return &DocumentListResponse{
		Object:  "list",
		Data:    items,
		HasMore: false, // Simplified for now
		Total:   total,
	}, nil
}

// DocumentListResponse represents a list of documents
type DocumentListResponse struct {
	Object  string                  `json:"object"`
	Data    []*ScanDocumentResponse `json:"data"`
	HasMore bool                    `json:"has_more"`
	Total   int64                   `json:"total"`
}

// Project file handlers

// CreateProjectFileRequest represents a request to add a file to a project
type CreateProjectFileRequest struct {
	MediaObjectID string `json:"media_object_id" binding:"required"`
	Filename      string `json:"filename,omitempty"`
}

// ProjectFileResponse represents a project file in API responses
type ProjectFileResponse struct {
	ID              string               `json:"id"`
	Object          string               `json:"object"`
	DisplayOrder    int                  `json:"display_order"`
	DocumentContent *ScanDocumentResponse `json:"document_content,omitempty"`
	CreatedAt       int64                `json:"created_at"`
}

// CreateProjectFile adds a file to a project
func (h *DocumentHandler) CreateProjectFile(ctx context.Context, userID uint, projectPublicID string, req CreateProjectFileRequest, authHeader string) (*ProjectFileResponse, error) {
	// Verify project ownership
	proj, err := h.projectService.GetProjectByPublicIDAndUserID(ctx, projectPublicID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "project not found")
	}

	// First scan the document (or get existing)
	scanResp, err := h.ScanDocument(ctx, userID, ScanDocumentRequest{
		MediaObjectID: req.MediaObjectID,
		Filename:      req.Filename,
	}, authHeader)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to process document")
	}

	// Get the document content
	doc, err := h.documentService.GetDocumentContentByPublicID(ctx, scanResp.ID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get document content")
	}

	// Generate public ID for project file
	filePublicID, err := idgen.GenerateSecureID("pf", 16)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to generate file ID")
	}

	// Get current file count for display order
	existingFiles, _, _ := h.projectFileService.ListProjectFiles(ctx, proj.ID, nil)
	displayOrder := len(existingFiles)

	// Create project file
	projectFile := document.NewProjectFile(filePublicID, proj.ID, doc.ID, userID, displayOrder)
	projectFile, err = h.projectFileService.CreateProjectFile(ctx, projectFile)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to create project file")
	}

	return &ProjectFileResponse{
		ID:              projectFile.PublicID,
		Object:          "project_file",
		DisplayOrder:    projectFile.DisplayOrder,
		DocumentContent: scanResp,
		CreatedAt:       projectFile.CreatedAt.Unix(),
	}, nil
}

// ListProjectFiles lists all files for a project
func (h *DocumentHandler) ListProjectFiles(ctx context.Context, userID uint, projectPublicID string, pagination *query.Pagination) (*ProjectFileListResponse, error) {
	// Verify project ownership
	proj, err := h.projectService.GetProjectByPublicIDAndUserID(ctx, projectPublicID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "project not found")
	}

	files, total, err := h.projectFileService.ListProjectFiles(ctx, proj.ID, pagination)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to list project files")
	}

	items := make([]*ProjectFileResponse, len(files))
	for i, f := range files {
		items[i] = h.toProjectFileResponse(f)
	}

	return &ProjectFileListResponse{
		Object:  "list",
		Data:    items,
		HasMore: false,
		Total:   total,
	}, nil
}

// ProjectFileListResponse represents a list of project files
type ProjectFileListResponse struct {
	Object  string                 `json:"object"`
	Data    []*ProjectFileResponse `json:"data"`
	HasMore bool                   `json:"has_more"`
	Total   int64                  `json:"total"`
}

// GetProjectFile retrieves a single project file
func (h *DocumentHandler) GetProjectFile(ctx context.Context, userID uint, projectPublicID, filePublicID string) (*ProjectFileResponse, error) {
	// Verify project ownership
	proj, err := h.projectService.GetProjectByPublicIDAndUserID(ctx, projectPublicID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "project not found")
	}

	file, err := h.projectFileService.GetProjectFileByPublicIDAndProjectID(ctx, filePublicID, proj.ID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "project file not found")
	}

	return h.toProjectFileResponse(file), nil
}

// DeleteProjectFile removes a file from a project
func (h *DocumentHandler) DeleteProjectFile(ctx context.Context, userID uint, projectPublicID, filePublicID string) error {
	// Verify project ownership
	proj, err := h.projectService.GetProjectByPublicIDAndUserID(ctx, projectPublicID, userID)
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "project not found")
	}

	// Verify file belongs to project
	_, err = h.projectFileService.GetProjectFileByPublicIDAndProjectID(ctx, filePublicID, proj.ID)
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "project file not found")
	}

	return h.projectFileService.DeleteProjectFile(ctx, filePublicID)
}

// ReorderProjectFilesRequest represents a request to reorder project files
type ReorderProjectFilesRequest struct {
	FileOrders map[string]int `json:"file_orders" binding:"required"`
}

// ReorderProjectFiles updates the display order of project files
func (h *DocumentHandler) ReorderProjectFiles(ctx context.Context, userID uint, projectPublicID string, req ReorderProjectFilesRequest) error {
	// Verify project ownership
	proj, err := h.projectService.GetProjectByPublicIDAndUserID(ctx, projectPublicID, userID)
	if err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "project not found")
	}

	return h.projectFileService.ReorderProjectFiles(ctx, proj.ID, req.FileOrders)
}

// Helper methods

func (h *DocumentHandler) getOCRProvider(ctx context.Context) (*domainmodel.Provider, error) {
	// Get provider that supports document OCR (category = "docling" or similar)
	providers, _, err := h.providerService.GetProvidersByCategory(ctx, domainmodel.ProviderCategoryDocling)
	if err != nil || len(providers) == 0 {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound,
			"no document OCR provider available", nil, "no-ocr-provider")
	}

	// Return first active provider
	for _, p := range providers {
		if p.Active {
			return p, nil
		}
	}

	return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeNotFound,
		"no active document OCR provider", nil, "no-active-ocr-provider")
}

func (h *DocumentHandler) toScanResponse(doc *document.DocumentContent) *ScanDocumentResponse {
	return &ScanDocumentResponse{
		ID:               doc.PublicID,
		Object:           "document_content",
		MediaObjectID:    doc.MediaObjectID,
		Filename:         doc.Filename,
		MimeType:         doc.MimeType,
		ProcessingStatus: doc.ProcessingStatus,
		ExtractedText:    doc.ExtractedText,
		PageCount:        doc.PageCount,
		WordCount:        doc.WordCount,
		ErrorMessage:     doc.ErrorMessage,
	}
}

func (h *DocumentHandler) toProjectFileResponse(file *document.ProjectFile) *ProjectFileResponse {
	resp := &ProjectFileResponse{
		ID:           file.PublicID,
		Object:       "project_file",
		DisplayOrder: file.DisplayOrder,
		CreatedAt:    file.CreatedAt.Unix(),
	}

	if file.DocumentContent != nil {
		resp.DocumentContent = h.toScanResponse(file.DocumentContent)
	}

	return resp
}

// Helper to convert string to int for pagination cursor
func parseUintCursor(s string) *uint {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil
	}
	u := uint(v)
	return &u
}

// Trim string utility
func trimString(s string) string {
	return strings.TrimSpace(s)
}
