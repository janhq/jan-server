package document

import (
	"context"
	"mime"
	"strings"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/domain/query"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

// DocumentService handles business logic for document OCR operations
type DocumentService struct {
	contentRepo DocumentContentRepository
	fileRepo    ProjectFileRepository
	config      *config.Config
}

// NewDocumentService creates a new document service
func NewDocumentService(
	contentRepo DocumentContentRepository,
	fileRepo ProjectFileRepository,
	cfg *config.Config,
) *DocumentService {
	return &DocumentService{
		contentRepo: contentRepo,
		fileRepo:    fileRepo,
		config:      cfg,
	}
}

// CreateDocumentContent creates a new document content record
func (s *DocumentService) CreateDocumentContent(ctx context.Context, doc *DocumentContent) (*DocumentContent, error) {
	// Validate mime type
	if !s.IsSupportedMimeType(doc.MimeType) {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerDomain,
			platformerrors.ErrorTypeValidation,
			"unsupported document type: "+doc.MimeType,
			nil,
			"doc-unsupported-type",
		)
	}

	// Validate file size
	if doc.FileSize > s.config.DocumentMaxBytes {
		return nil, platformerrors.NewError(
			ctx,
			platformerrors.LayerDomain,
			platformerrors.ErrorTypeValidation,
			"file size exceeds maximum allowed",
			nil,
			"doc-size-exceeded",
		)
	}

	if err := s.contentRepo.Create(ctx, doc); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to create document content")
	}

	return doc, nil
}

// GetDocumentContentByPublicID retrieves a document content by public ID
func (s *DocumentService) GetDocumentContentByPublicID(ctx context.Context, publicID string) (*DocumentContent, error) {
	doc, err := s.contentRepo.GetByPublicID(ctx, publicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "document content not found")
	}
	return doc, nil
}

// GetDocumentContentByMediaObjectID retrieves a document content by media object ID
func (s *DocumentService) GetDocumentContentByMediaObjectID(ctx context.Context, mediaObjectID string, userID uint) (*DocumentContent, error) {
	doc, err := s.contentRepo.GetByMediaObjectID(ctx, mediaObjectID, userID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "document content not found")
	}
	return doc, nil
}

// UpdateDocumentContent updates a document content record
func (s *DocumentService) UpdateDocumentContent(ctx context.Context, doc *DocumentContent) (*DocumentContent, error) {
	if err := s.contentRepo.Update(ctx, doc); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to update document content")
	}
	return doc, nil
}

// ListDocumentContents lists document contents with filtering and pagination
func (s *DocumentService) ListDocumentContents(ctx context.Context, userID uint, pagination *query.Pagination) ([]*DocumentContent, int64, error) {
	filter := DocumentContentFilter{UserID: &userID}
	docs, total, err := s.contentRepo.List(ctx, filter, pagination)
	if err != nil {
		return nil, 0, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to list document contents")
	}
	return docs, total, nil
}

// IsSupportedMimeType checks if the given mime type is supported for OCR
func (s *DocumentService) IsSupportedMimeType(mimeType string) bool {
	normalized := strings.TrimSpace(mimeType)
	if parsed, _, err := mime.ParseMediaType(normalized); err == nil && parsed != "" {
		normalized = parsed
	}

	supportedTypes := strings.Split(s.config.DocumentSupportedTypes, ",")
	for _, supported := range supportedTypes {
		if strings.TrimSpace(supported) == normalized {
			return true
		}
	}
	return false
}

// IsOCREnabled returns whether document OCR is enabled
func (s *DocumentService) IsOCREnabled() bool {
	return s.config.DocumentOCREnabled
}

// ProjectFileService handles business logic for project files
type ProjectFileService struct {
	fileRepo    ProjectFileRepository
	contentRepo DocumentContentRepository
}

// NewProjectFileService creates a new project file service
func NewProjectFileService(
	fileRepo ProjectFileRepository,
	contentRepo DocumentContentRepository,
) *ProjectFileService {
	return &ProjectFileService{
		fileRepo:    fileRepo,
		contentRepo: contentRepo,
	}
}

// CreateProjectFile creates a new project file
func (s *ProjectFileService) CreateProjectFile(ctx context.Context, file *ProjectFile) (*ProjectFile, error) {
	if err := s.fileRepo.Create(ctx, file); err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to create project file")
	}
	return file, nil
}

// GetProjectFile retrieves a project file by public ID
func (s *ProjectFileService) GetProjectFile(ctx context.Context, publicID string) (*ProjectFile, error) {
	file, err := s.fileRepo.GetByPublicID(ctx, publicID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "project file not found")
	}
	return file, nil
}

// GetProjectFileByPublicIDAndProjectID retrieves a project file with ownership validation
func (s *ProjectFileService) GetProjectFileByPublicIDAndProjectID(ctx context.Context, publicID string, projectID uint) (*ProjectFile, error) {
	file, err := s.fileRepo.GetByPublicIDAndProjectID(ctx, publicID, projectID)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "project file not found")
	}
	return file, nil
}

// ListProjectFiles lists all files for a project with their document content
func (s *ProjectFileService) ListProjectFiles(ctx context.Context, projectID uint, pagination *query.Pagination) ([]*ProjectFile, int64, error) {
	files, total, err := s.fileRepo.ListByProjectID(ctx, projectID, pagination)
	if err != nil {
		return nil, 0, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to list project files")
	}
	return files, total, nil
}

// DeleteProjectFile soft deletes a project file
func (s *ProjectFileService) DeleteProjectFile(ctx context.Context, publicID string) error {
	if err := s.fileRepo.Delete(ctx, publicID); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to delete project file")
	}
	return nil
}

// ReorderProjectFiles updates the display order of project files
func (s *ProjectFileService) ReorderProjectFiles(ctx context.Context, projectID uint, fileOrders map[string]int) error {
	if err := s.fileRepo.UpdateDisplayOrders(ctx, projectID, fileOrders); err != nil {
		return platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to reorder project files")
	}
	return nil
}

// GetProjectFilesWithContent retrieves all project files with their extracted content
// This is used for injecting project file content into prompts
func (s *ProjectFileService) GetProjectFilesWithContent(ctx context.Context, projectID uint) ([]*ProjectFile, error) {
	files, _, err := s.fileRepo.ListByProjectID(ctx, projectID, nil)
	if err != nil {
		return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "failed to get project files")
	}

	return files, nil
}
