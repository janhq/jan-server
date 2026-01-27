-- Ensure llm_api schema exists (migrate from public if needed)
CREATE SCHEMA IF NOT EXISTS llm_api;

-- Store extracted document content from OCR processing
CREATE TABLE IF NOT EXISTS llm_api.document_contents (
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

CREATE INDEX IF NOT EXISTS idx_document_contents_media_object_id ON llm_api.document_contents(media_object_id);
CREATE INDEX IF NOT EXISTS idx_document_contents_user_id ON llm_api.document_contents(user_id);
CREATE INDEX IF NOT EXISTS idx_document_contents_status ON llm_api.document_contents(processing_status);

COMMENT ON TABLE llm_api.document_contents IS 'Stores extracted text content from document OCR processing';
COMMENT ON COLUMN llm_api.document_contents.media_object_id IS 'jan_* ID referencing the file in media-api';
COMMENT ON COLUMN llm_api.document_contents.processing_status IS 'Status: pending, processing, completed, failed';

-- Project files table for attaching documents to projects
CREATE TABLE IF NOT EXISTS llm_api.project_files (
    id BIGSERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    project_id BIGINT NOT NULL REFERENCES llm_api.projects(id) ON DELETE CASCADE,
    document_content_id BIGINT REFERENCES llm_api.document_contents(id),
    display_order INTEGER NOT NULL DEFAULT 0,
    created_by BIGINT NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_project_files_project_id ON llm_api.project_files(project_id);
CREATE INDEX IF NOT EXISTS idx_project_files_deleted_at ON llm_api.project_files(deleted_at);
CREATE INDEX IF NOT EXISTS idx_project_files_display_order ON llm_api.project_files(project_id, display_order);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'document_contents'
    ) THEN
        INSERT INTO llm_api.document_contents (
            id, public_id, media_object_id, user_id, filename, mime_type, file_size,
            processing_status, extracted_text, extraction_model, page_count, word_count,
            error_message, created_at, updated_at
        )
        SELECT
            id, public_id, media_object_id, user_id, filename, mime_type, file_size,
            processing_status, extracted_text, extraction_model, page_count, word_count,
            error_message, created_at, updated_at
        FROM public.document_contents
        ON CONFLICT DO NOTHING;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'project_files'
    ) THEN
        INSERT INTO llm_api.project_files (
            id, public_id, project_id, document_content_id, display_order,
            created_by, deleted_at, created_at, updated_at
        )
        SELECT
            id, public_id, project_id, document_content_id, display_order,
            created_by, deleted_at, created_at, updated_at
        FROM public.project_files
        ON CONFLICT DO NOTHING;
    END IF;
END $$;

COMMENT ON TABLE llm_api.project_files IS 'Files attached to projects for context injection';
COMMENT ON COLUMN llm_api.project_files.display_order IS 'Order in which files are displayed and injected into prompts';
