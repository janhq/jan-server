package document

import (
	"context"
	"time"

	"jan-server/services/llm-api/internal/domain/query"
)

// ProcessingStatus represents the status of document OCR processing
type ProcessingStatus string

const (
	ProcessingStatusPending    ProcessingStatus = "pending"
	ProcessingStatusProcessing ProcessingStatus = "processing"
	ProcessingStatusCompleted  ProcessingStatus = "completed"
	ProcessingStatusFailed     ProcessingStatus = "failed"
)

// DocumentContent represents extracted content from a document via OCR
type DocumentContent struct {
	ID               uint             `json:"-"`
	PublicID         string           `json:"id"`
	Object           string           `json:"object"` // Always "document_content"
	MediaObjectID    string           `json:"media_object_id"`
	UserID           uint             `json:"-"`
	Filename         string           `json:"filename,omitempty"`
	MimeType         string           `json:"mime_type,omitempty"`
	FileSize         int64            `json:"file_size,omitempty"`
	ProcessingStatus ProcessingStatus `json:"processing_status"`
	ExtractedText    string           `json:"extracted_text,omitempty"`
	ExtractionModel  string           `json:"extraction_model,omitempty"`
	PageCount        *int             `json:"page_count,omitempty"`
	WordCount        *int             `json:"word_count,omitempty"`
	ErrorMessage     *string          `json:"error_message,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// ProjectFile represents a file attached to a project
type ProjectFile struct {
	ID                uint              `json:"-"`
	PublicID          string            `json:"id"`
	Object            string            `json:"object"` // Always "project_file"
	ProjectID         uint              `json:"-"`
	DocumentContentID *uint             `json:"-"`
	DocumentContent   *DocumentContent  `json:"document_content,omitempty"`
	DisplayOrder      int               `json:"display_order"`
	CreatedBy         uint              `json:"-"`
	DeletedAt         *time.Time        `json:"-"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// DocumentContentFilter is used to filter document content queries
type DocumentContentFilter struct {
	ID               *uint
	PublicID         *string
	MediaObjectID    *string
	UserID           *uint
	ProcessingStatus *ProcessingStatus
}

// ProjectFileFilter is used to filter project file queries
type ProjectFileFilter struct {
	ID                *uint
	PublicID          *string
	ProjectID         *uint
	DocumentContentID *uint
	IncludeDeleted    bool
}

// DocumentContentRepository defines the interface for document content persistence
type DocumentContentRepository interface {
	Create(ctx context.Context, doc *DocumentContent) error
	GetByPublicID(ctx context.Context, publicID string) (*DocumentContent, error)
	GetByMediaObjectID(ctx context.Context, mediaObjectID string, userID uint) (*DocumentContent, error)
	Update(ctx context.Context, doc *DocumentContent) error
	Delete(ctx context.Context, publicID string) error
	List(ctx context.Context, filter DocumentContentFilter, pagination *query.Pagination) ([]*DocumentContent, int64, error)
}

// ProjectFileRepository defines the interface for project file persistence
type ProjectFileRepository interface {
	Create(ctx context.Context, file *ProjectFile) error
	GetByPublicID(ctx context.Context, publicID string) (*ProjectFile, error)
	GetByPublicIDAndProjectID(ctx context.Context, publicID string, projectID uint) (*ProjectFile, error)
	Update(ctx context.Context, file *ProjectFile) error
	Delete(ctx context.Context, publicID string) error
	ListByProjectID(ctx context.Context, projectID uint, pagination *query.Pagination) ([]*ProjectFile, int64, error)
	UpdateDisplayOrders(ctx context.Context, projectID uint, fileOrders map[string]int) error
}

// NewDocumentContent creates a new DocumentContent with the given parameters
func NewDocumentContent(publicID string, userID uint, mediaObjectID, filename, mimeType string, fileSize int64) *DocumentContent {
	now := time.Now()

	return &DocumentContent{
		PublicID:         publicID,
		Object:           "document_content",
		MediaObjectID:    mediaObjectID,
		UserID:           userID,
		Filename:         filename,
		MimeType:         mimeType,
		FileSize:         fileSize,
		ProcessingStatus: ProcessingStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// NewProjectFile creates a new ProjectFile with the given parameters
func NewProjectFile(publicID string, projectID, documentContentID, createdBy uint, displayOrder int) *ProjectFile {
	now := time.Now()

	return &ProjectFile{
		PublicID:          publicID,
		Object:            "project_file",
		ProjectID:         projectID,
		DocumentContentID: &documentContentID,
		DisplayOrder:      displayOrder,
		CreatedBy:         createdBy,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// MarkProcessing updates the document status to processing
func (d *DocumentContent) MarkProcessing() {
	d.ProcessingStatus = ProcessingStatusProcessing
	d.UpdatedAt = time.Now()
}

// MarkCompleted updates the document with extracted text and completion status
func (d *DocumentContent) MarkCompleted(text, model string, pageCount, wordCount int) {
	d.ProcessingStatus = ProcessingStatusCompleted
	d.ExtractedText = text
	d.ExtractionModel = model
	d.PageCount = &pageCount
	d.WordCount = &wordCount
	d.UpdatedAt = time.Now()
}

// MarkFailed updates the document with failure status and error message
func (d *DocumentContent) MarkFailed(errorMsg string) {
	d.ProcessingStatus = ProcessingStatusFailed
	d.ErrorMessage = &errorMsg
	d.UpdatedAt = time.Now()
}
