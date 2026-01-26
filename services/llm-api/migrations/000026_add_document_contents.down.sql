-- Remove indexes
DROP INDEX IF EXISTS idx_project_files_display_order;
DROP INDEX IF EXISTS idx_project_files_deleted_at;
DROP INDEX IF EXISTS idx_project_files_project_id;

-- Drop project_files table
DROP TABLE IF EXISTS project_files;

-- Remove indexes
DROP INDEX IF EXISTS idx_document_contents_status;
DROP INDEX IF EXISTS idx_document_contents_user_id;
DROP INDEX IF EXISTS idx_document_contents_media_object_id;

-- Drop document_contents table
DROP TABLE IF EXISTS document_contents;
