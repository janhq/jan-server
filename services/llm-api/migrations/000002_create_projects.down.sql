-- Remove indexes
DROP INDEX IF EXISTS idx_conversations_project_updated_at;
DROP INDEX IF EXISTS idx_projects_archived_at;
DROP INDEX IF EXISTS idx_projects_deleted_at;
DROP INDEX IF EXISTS idx_projects_user_updated_at;
DROP INDEX IF EXISTS idx_projects_user_id;

-- Remove columns from conversations
ALTER TABLE conversations
    DROP COLUMN IF EXISTS effective_instruction_snapshot,
    DROP COLUMN IF EXISTS instruction_version,
    DROP COLUMN IF EXISTS project_id;

-- Drop projects table
DROP TABLE IF EXISTS projects;
