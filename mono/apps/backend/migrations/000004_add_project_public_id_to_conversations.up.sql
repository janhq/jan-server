-- Add project_public_id column to conversations table
ALTER TABLE llm_api.conversations
    ADD COLUMN IF NOT EXISTS project_public_id VARCHAR(64);

-- Add index for project_public_id
CREATE INDEX IF NOT EXISTS idx_conversations_project_public_id 
    ON llm_api.conversations(project_public_id);

COMMENT ON COLUMN llm_api.conversations.project_public_id IS 'Public ID of the project this conversation belongs to';
