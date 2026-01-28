package dbschema

import (
	"time"

	"jan-server/services/llm-api/internal/domain/document"
	"jan-server/services/llm-api/internal/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(DocumentContent{})
	database.RegisterSchemaForAutoMigrate(ProjectFile{})
}

// DocumentContent represents the database schema for document OCR content
type DocumentContent struct {
	ID               uint       `gorm:"primarykey"`
	PublicID         string     `gorm:"uniqueIndex;size:64;not null"`
	MediaObjectID    string     `gorm:"index;size:40;not null"`
	UserID           uint       `gorm:"index;not null"`
	Filename         *string    `gorm:"size:512"`
	MimeType         *string    `gorm:"size:128"`
	FileSize         *int64
	ProcessingStatus string     `gorm:"index;size:32;not null;default:'pending'"`
	ExtractedText    *string    `gorm:"type:text"`
	ExtractionModel  *string    `gorm:"size:128"`
	PageCount        *int
	WordCount        *int
	ErrorMessage     *string    `gorm:"type:text"`
	CreatedAt        time.Time  `gorm:"not null"`
	UpdatedAt        time.Time  `gorm:"not null"`
}

// TableName specifies the table name for DocumentContent
func (DocumentContent) TableName() string {
	return "llm_api.document_contents"
}

// EtoD converts database schema to domain document content (Entity to Domain)
func (d *DocumentContent) EtoD() *document.DocumentContent {
	doc := &document.DocumentContent{
		ID:               d.ID,
		PublicID:         d.PublicID,
		Object:           "document_content",
		MediaObjectID:    d.MediaObjectID,
		UserID:           d.UserID,
		ProcessingStatus: document.ProcessingStatus(d.ProcessingStatus),
		PageCount:        d.PageCount,
		WordCount:        d.WordCount,
		ErrorMessage:     d.ErrorMessage,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}

	if d.Filename != nil {
		doc.Filename = *d.Filename
	}
	if d.MimeType != nil {
		doc.MimeType = *d.MimeType
	}
	if d.FileSize != nil {
		doc.FileSize = *d.FileSize
	}
	if d.ExtractedText != nil {
		doc.ExtractedText = *d.ExtractedText
	}
	if d.ExtractionModel != nil {
		doc.ExtractionModel = *d.ExtractionModel
	}

	return doc
}

// NewSchemaDocumentContent creates a database schema from domain document content
func NewSchemaDocumentContent(d *document.DocumentContent) *DocumentContent {
	doc := &DocumentContent{
		ID:               d.ID,
		PublicID:         d.PublicID,
		MediaObjectID:    d.MediaObjectID,
		UserID:           d.UserID,
		ProcessingStatus: string(d.ProcessingStatus),
		PageCount:        d.PageCount,
		WordCount:        d.WordCount,
		ErrorMessage:     d.ErrorMessage,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}

	if d.Filename != "" {
		doc.Filename = &d.Filename
	}
	if d.MimeType != "" {
		doc.MimeType = &d.MimeType
	}
	if d.FileSize > 0 {
		doc.FileSize = &d.FileSize
	}
	if d.ExtractedText != "" {
		doc.ExtractedText = &d.ExtractedText
	}
	if d.ExtractionModel != "" {
		doc.ExtractionModel = &d.ExtractionModel
	}

	return doc
}

// ProjectFile represents the database schema for project files
type ProjectFile struct {
	ID                uint       `gorm:"primarykey"`
	PublicID          string     `gorm:"uniqueIndex;size:64;not null"`
	ProjectID         uint       `gorm:"index;not null"`
	DocumentContentID *uint      `gorm:"index"`
	DisplayOrder      int        `gorm:"not null;default:0"`
	CreatedBy         uint       `gorm:"not null"`
	DeletedAt         *time.Time `gorm:"index"`
	CreatedAt         time.Time  `gorm:"not null"`
	UpdatedAt         time.Time  `gorm:"not null"`

	// Relations
	DocumentContent *DocumentContent `gorm:"foreignKey:DocumentContentID"`
}

// TableName specifies the table name for ProjectFile
func (ProjectFile) TableName() string {
	return "llm_api.project_files"
}

// EtoD converts database schema to domain project file (Entity to Domain)
func (p *ProjectFile) EtoD() *document.ProjectFile {
	file := &document.ProjectFile{
		ID:                p.ID,
		PublicID:          p.PublicID,
		Object:            "project_file",
		ProjectID:         p.ProjectID,
		DocumentContentID: p.DocumentContentID,
		DisplayOrder:      p.DisplayOrder,
		CreatedBy:         p.CreatedBy,
		DeletedAt:         p.DeletedAt,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}

	if p.DocumentContent != nil {
		file.DocumentContent = p.DocumentContent.EtoD()
	}

	return file
}

// NewSchemaProjectFile creates a database schema from domain project file
func NewSchemaProjectFile(f *document.ProjectFile) *ProjectFile {
	return &ProjectFile{
		ID:                f.ID,
		PublicID:          f.PublicID,
		ProjectID:         f.ProjectID,
		DocumentContentID: f.DocumentContentID,
		DisplayOrder:      f.DisplayOrder,
		CreatedBy:         f.CreatedBy,
		DeletedAt:         f.DeletedAt,
		CreatedAt:         f.CreatedAt,
		UpdatedAt:         f.UpdatedAt,
	}
}
