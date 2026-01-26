-- Store extracted document content from OCR processing
CREATE TABLE IF NOT EXISTS document_contents (
    id BIGSERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    media_object_id VARCHAR(40) NOT NULL,  -- jan_* ID from media-api
    user_id BIGINT NOT NULL,
    filename VARCHAR(512),
    mime_type VARCHAR(128),
    file_size BIGINT,
    processing_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    extracted_text TEXT,
    extraction_model VARCHAR(128),
    page_count INTEGER,
    word_count INTEGER,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_document_contents_media_object_id ON document_contents(media_object_id);
CREATE INDEX idx_document_contents_user_id ON document_contents(user_id);
CREATE INDEX idx_document_contents_status ON document_contents(processing_status);

COMMENT ON TABLE document_contents IS 'Stores extracted text content from document OCR processing';
COMMENT ON COLUMN document_contents.media_object_id IS 'jan_* ID referencing the file in media-api';
COMMENT ON COLUMN document_contents.processing_status IS 'Status: pending, processing, completed, failed';

-- Project files table for attaching documents to projects
CREATE TABLE IF NOT EXISTS project_files (
    id BIGSERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    document_content_id BIGINT REFERENCES document_contents(id),
    display_order INTEGER NOT NULL DEFAULT 0,
    created_by BIGINT NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_project_files_project_id ON project_files(project_id);
CREATE INDEX idx_project_files_deleted_at ON project_files(deleted_at);
CREATE INDEX idx_project_files_display_order ON project_files(project_id, display_order);

COMMENT ON TABLE project_files IS 'Files attached to projects for context injection';
COMMENT ON COLUMN project_files.display_order IS 'Order in which files are displayed and injected into prompts';
