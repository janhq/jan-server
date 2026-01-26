package document

// ScanRequest represents a request to scan a document for OCR
type ScanRequest struct {
	MediaObjectID string `json:"media_object_id" binding:"required"`
	Filename      string `json:"filename,omitempty"`
}

// ScanResponse represents the response from a document scan operation
type ScanResponse struct {
	ID               string           `json:"id"`
	MediaObjectID    string           `json:"media_object_id"`
	Filename         string           `json:"filename,omitempty"`
	MimeType         string           `json:"mime_type,omitempty"`
	ProcessingStatus ProcessingStatus `json:"processing_status"`
	ExtractedText    string           `json:"extracted_text,omitempty"`
	PageCount        *int             `json:"page_count,omitempty"`
	WordCount        *int             `json:"word_count,omitempty"`
	ErrorMessage     *string          `json:"error_message,omitempty"`
}

// DocumentContentResponse converts a DocumentContent to a response DTO
func DocumentContentResponse(doc *DocumentContent) *ScanResponse {
	if doc == nil {
		return nil
	}
	return &ScanResponse{
		ID:               doc.PublicID,
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

// CreateProjectFileRequest represents a request to attach a file to a project
type CreateProjectFileRequest struct {
	MediaObjectID string `json:"media_object_id" binding:"required"`
	Filename      string `json:"filename,omitempty"`
}

// ReorderProjectFilesRequest represents a request to reorder project files
type ReorderProjectFilesRequest struct {
	FileOrders map[string]int `json:"file_orders" binding:"required"`
}

// ProjectFileResponse represents a project file in API responses
type ProjectFileResponse struct {
	ID              string                  `json:"id"`
	Object          string                  `json:"object"`
	ProjectID       string                  `json:"project_id,omitempty"`
	DisplayOrder    int                     `json:"display_order"`
	DocumentContent *DocumentContentSummary `json:"document_content,omitempty"`
	CreatedAt       int64                   `json:"created_at"`
}

// DocumentContentSummary is a lightweight representation of document content
type DocumentContentSummary struct {
	ID               string           `json:"id"`
	Filename         string           `json:"filename,omitempty"`
	MimeType         string           `json:"mime_type,omitempty"`
	FileSize         int64            `json:"file_size,omitempty"`
	ProcessingStatus ProcessingStatus `json:"processing_status"`
	PageCount        *int             `json:"page_count,omitempty"`
	WordCount        *int             `json:"word_count,omitempty"`
}

// ToProjectFileResponse converts a ProjectFile to a response DTO
func ToProjectFileResponse(file *ProjectFile, projectPublicID string) *ProjectFileResponse {
	if file == nil {
		return nil
	}

	resp := &ProjectFileResponse{
		ID:           file.PublicID,
		Object:       "project_file",
		ProjectID:    projectPublicID,
		DisplayOrder: file.DisplayOrder,
		CreatedAt:    file.CreatedAt.Unix(),
	}

	if file.DocumentContent != nil {
		resp.DocumentContent = &DocumentContentSummary{
			ID:               file.DocumentContent.PublicID,
			Filename:         file.DocumentContent.Filename,
			MimeType:         file.DocumentContent.MimeType,
			FileSize:         file.DocumentContent.FileSize,
			ProcessingStatus: file.DocumentContent.ProcessingStatus,
			PageCount:        file.DocumentContent.PageCount,
			WordCount:        file.DocumentContent.WordCount,
		}
	}

	return resp
}
