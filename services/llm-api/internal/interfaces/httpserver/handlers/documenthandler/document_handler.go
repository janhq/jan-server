package documenthandler

import (
	"context"
	"fmt"
	"mime"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

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
	FileSize         int64                     `json:"file_size,omitempty"`
	ProcessingStatus document.ProcessingStatus `json:"processing_status"`
	ExtractedText    string                    `json:"extracted_text,omitempty"`
	PageCount        *int                      `json:"page_count,omitempty"`
	WordCount        *int                      `json:"word_count,omitempty"`
	ErrorMessage     *string                   `json:"error_message,omitempty"`
}

// ScanDocument performs OCR on a document
func (h *DocumentHandler) ScanDocument(ctx context.Context, userID uint, req ScanDocumentRequest, authHeader string) (*ScanDocumentResponse, error) {
	startTime := time.Now()
	requestID := ""
	if rid, ok := ctx.Value("request_id").(string); ok {
		requestID = rid
	}

	log.Info().
		Uint("user_id", userID).
		Str("media_object_id", req.MediaObjectID).
		Str("filename", req.Filename).
		Str("request_id", requestID).
		Msg("[DocumentHandler] OCR scan requested")

	// Check if OCR is enabled
	if !h.documentService.IsOCREnabled() {
		log.Warn().
			Uint("user_id", userID).
			Str("media_object_id", req.MediaObjectID).
			Str("request_id", requestID).
			Msg("[DocumentHandler] OCR is disabled")
		return nil, platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation,
			"document OCR is not enabled", nil, "doc-ocr-disabled")
	}

	// Check if document already exists for this media object
	existingDoc, err := h.documentService.GetDocumentContentByMediaObjectID(ctx, req.MediaObjectID, userID)
	if err == nil && existingDoc != nil {
		// Return existing document if already processed
		if existingDoc.ProcessingStatus == document.ProcessingStatusCompleted {
			log.Info().
				Uint("user_id", userID).
				Str("document_id", existingDoc.PublicID).
				Str("media_object_id", req.MediaObjectID).
				Str("request_id", requestID).
				Int("word_count", safeIntPtr(existingDoc.WordCount)).
				Int("page_count", safeIntPtr(existingDoc.PageCount)).
				Msg("[DocumentHandler] OCR already completed, returning cached result")
			return h.toScanResponse(existingDoc), nil
		}
	}

	// Resolve media object to get file URL and metadata
	mediaInfo, err := h.mediaClient.Resolve(ctx, req.MediaObjectID, authHeader)
	if err != nil {
		log.Error().
			Err(err).
			Uint("user_id", userID).
			Str("media_object_id", req.MediaObjectID).
			Str("request_id", requestID).
			Msg("[DocumentHandler] Failed to resolve media object")
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to resolve media object")
	}

	log.Debug().
		Uint("user_id", userID).
		Str("media_object_id", req.MediaObjectID).
		Str("media_id", mediaInfo.ID).
		Str("media_filename", mediaInfo.Filename).
		Str("media_content_type", mediaInfo.ContentType).
		Int64("media_size", mediaInfo.Size).
		Str("request_id", requestID).
		Msg("[DocumentHandler] Resolved media object")

	// Validate mime type
	mimeType := mediaInfo.ContentType
	if !h.documentService.IsSupportedMimeType(mimeType) {
		log.Warn().
			Uint("user_id", userID).
			Str("media_object_id", req.MediaObjectID).
			Str("mime_type", mimeType).
			Str("request_id", requestID).
			Msg("[DocumentHandler] Unsupported document type for OCR")
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
		log.Error().
			Err(err).
			Uint("user_id", userID).
			Str("document_id", publicID).
			Str("media_object_id", req.MediaObjectID).
			Str("request_id", requestID).
			Msg("[DocumentHandler] Failed to create document content")
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to create document content")
	}

	log.Info().
		Uint("user_id", userID).
		Str("document_id", doc.PublicID).
		Str("media_object_id", req.MediaObjectID).
		Str("mime_type", mimeType).
		Str("request_id", requestID).
		Msg("[DocumentHandler] Document content created, starting OCR")

	// Fetch file data from media URL (rewrite localhost for in-cluster access)
	mediaURL := mediaInfo.URL
	if mediaURL != "" {
		if parsedURL, err := url.Parse(mediaURL); err == nil {
			host := strings.ToLower(parsedURL.Hostname())
			if host == "localhost" || host == "127.0.0.1" {
				if resolveURL, err := url.Parse(h.config.MediaResolveURL); err == nil && resolveURL.Host != "" {
					base := (&url.URL{Scheme: resolveURL.Scheme, Host: resolveURL.Host}).String()
					mediaURL = fmt.Sprintf("%s/api/media/%s", strings.TrimSuffix(base, "/"), mediaInfo.ID)
					log.Debug().
						Str("media_host", host).
						Str("resolved_host", resolveURL.Host).
						Str("request_id", requestID).
						Msg("[DocumentHandler] Rewrote media URL for in-cluster access")
				}
			}
		}
	}

	// Fetch file data from media URL
	fileFetchStart := time.Now()
	fileData, fetchedContentType, err := h.ocrService.FetchFileFromURL(ctx, mediaURL)
	if err != nil {
		doc.MarkFailed("failed to fetch file from media service")
		h.documentService.UpdateDocumentContent(ctx, doc)
		log.Error().
			Err(err).
			Uint("user_id", userID).
			Str("document_id", doc.PublicID).
			Str("media_object_id", req.MediaObjectID).
			Str("request_id", requestID).
			Msg("[DocumentHandler] Failed to fetch file from media URL")
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to fetch document file")
	}

	log.Info().
		Uint("user_id", userID).
		Str("document_id", doc.PublicID).
		Str("media_object_id", req.MediaObjectID).
		Int("data_size", len(fileData)).
		Str("fetched_content_type", fetchedContentType).
		Dur("duration", time.Since(fileFetchStart)).
		Str("request_id", requestID).
		Msg("[DocumentHandler] Fetched file data from media service")

	normalizedMimeType := mimeType
	if parsed, _, err := mime.ParseMediaType(mimeType); err == nil && parsed != "" {
		normalizedMimeType = parsed
	}

	// Fallback: for text/* types, skip external OCR and store raw text
	if strings.HasPrefix(normalizedMimeType, "text/") {
		text := string(fileData)
		wordCount := len(strings.Fields(text))
		pageCount := 1
		doc.MarkCompleted(text, "raw-text", pageCount, wordCount)
		doc, err = h.documentService.UpdateDocumentContent(ctx, doc)
		if err != nil {
			log.Error().
				Err(err).
				Uint("user_id", userID).
				Str("document_id", doc.PublicID).
				Str("request_id", requestID).
				Msg("[DocumentHandler] Failed to update document content after text fallback")
			return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to update document content")
		}
		log.Info().
			Uint("user_id", userID).
			Str("document_id", doc.PublicID).
			Str("mime_type", normalizedMimeType).
			Int("word_count", wordCount).
			Int("page_count", pageCount).
			Dur("duration", time.Since(startTime)).
			Str("request_id", requestID).
			Msg("[DocumentHandler] Stored raw text without external OCR")
		return h.toScanResponse(doc), nil
	}

	// Get OCR provider for non-text formats
	provider, err := h.getOCRProvider(ctx)
	if err != nil {
		doc.MarkFailed("no OCR provider available")
		h.documentService.UpdateDocumentContent(ctx, doc)
		log.Warn().
			Err(err).
			Uint("user_id", userID).
			Str("document_id", doc.PublicID).
			Str("request_id", requestID).
			Msg("[DocumentHandler] No active OCR provider available")
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to get OCR provider")
	}

	log.Debug().
		Str("provider_id", provider.PublicID).
		Str("provider_name", provider.DisplayName).
		Str("provider_kind", string(provider.Kind)).
		Str("provider_category", string(provider.Category)).
		Str("request_id", requestID).
		Msg("[DocumentHandler] Using OCR provider")

	// Call OCR service
	ocrStart := time.Now()
	ocrResp, err := h.ocrService.ScanWithFileData(ctx, provider, fileData, mimeType, filename)
	if err != nil {
		doc.MarkFailed(err.Error())
		h.documentService.UpdateDocumentContent(ctx, doc)
		log.Error().
			Err(err).
			Uint("user_id", userID).
			Str("document_id", doc.PublicID).
			Str("provider_id", provider.PublicID).
			Str("request_id", requestID).
			Msg("[DocumentHandler] OCR processing failed")
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "OCR processing failed")
	}

	// Update document with extracted text
	doc.MarkCompleted(ocrResp.Text, ocrResp.Model, ocrResp.PageCount, ocrResp.WordCount)
	doc, err = h.documentService.UpdateDocumentContent(ctx, doc)
	if err != nil {
		log.Error().
			Err(err).
			Uint("user_id", userID).
			Str("document_id", doc.PublicID).
			Str("request_id", requestID).
			Msg("[DocumentHandler] Failed to update document content after OCR")
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to update document content")
	}

	log.Info().
		Uint("user_id", userID).
		Str("document_id", doc.PublicID).
		Str("provider_id", provider.PublicID).
		Str("model", ocrResp.Model).
		Int("word_count", ocrResp.WordCount).
		Int("page_count", ocrResp.PageCount).
		Dur("provider_duration", time.Since(ocrStart)).
		Dur("duration", time.Since(startTime)).
		Str("request_id", requestID).
		Msg("[DocumentHandler] OCR completed successfully")

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
	ID              string                `json:"id"`
	Object          string                `json:"object"`
	DisplayOrder    int                   `json:"display_order"`
	DocumentContent *ScanDocumentResponse `json:"document_content,omitempty"`
	CreatedAt       int64                 `json:"created_at"`
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
	existingFiles, _, err := h.projectFileService.ListProjectFiles(ctx, proj.ID, nil)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerHandler, err, "failed to load project files")
	}
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
	// Prefer new OCR category, fall back to legacy docling category.
	categories := []domainmodel.ProviderCategory{
		domainmodel.ProviderCategoryOCR,
		domainmodel.ProviderCategoryDocling,
	}

	for _, category := range categories {
		providers, _, err := h.providerService.GetProvidersByCategory(ctx, category)
		if err != nil || len(providers) == 0 {
			continue
		}
		for _, p := range providers {
			if p.Active {
				return p, nil
			}
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
		FileSize:         doc.FileSize,
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
func safeIntPtr(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
