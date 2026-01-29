-- Remove project_public_id column from conversations table
DROP INDEX IF EXISTS llm_api.idx_conversations_project_public_id;

ALTER TABLE llm_api.conversations
    DROP COLUMN IF EXISTS project_public_id;
