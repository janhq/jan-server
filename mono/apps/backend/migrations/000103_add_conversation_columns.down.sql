-- +goose Down
-- Remove columns added in 000003

SET search_path TO response_api;

-- Drop indexes first
DROP INDEX IF EXISTS response_api.idx_conversation_user_referrer;
DROP INDEX IF EXISTS response_api.idx_conversation_user_status;
DROP INDEX IF EXISTS response_api.idx_conversations_project_updated_at;
DROP INDEX IF EXISTS response_api.idx_conversations_project_public_id;
DROP INDEX IF EXISTS response_api.idx_conversation_branch_name;

-- Drop conversation_branches table
DROP TABLE IF EXISTS response_api.conversation_branches;

-- Remove columns from conversation_items
ALTER TABLE response_api.conversation_items 
DROP COLUMN IF EXISTS sequence_number,
DROP COLUMN IF EXISTS legacy_content;

-- Remove columns from conversations (in reverse order)
ALTER TABLE response_api.conversations 
DROP COLUMN IF EXISTS effective_instruction_snapshot,
DROP COLUMN IF EXISTS instruction_version,
DROP COLUMN IF EXISTS project_public_id,
DROP COLUMN IF EXISTS project_id,
DROP COLUMN IF EXISTS is_private,
DROP COLUMN IF EXISTS referrer,
DROP COLUMN IF EXISTS active_branch,
DROP COLUMN IF EXISTS status,
DROP COLUMN IF EXISTS title,
DROP COLUMN IF EXISTS object;
