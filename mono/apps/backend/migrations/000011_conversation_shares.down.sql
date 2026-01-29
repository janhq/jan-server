-- Rollback: 000011_conversation_shares
-- Purpose: Remove conversation sharing feature

SET search_path TO llm_api;

-- Drop trigger
DROP TRIGGER IF EXISTS conversation_shares_updated_at ON llm_api.conversation_shares;

-- Drop indexes
DROP INDEX IF EXISTS llm_api.idx_conversation_shares_slug;
DROP INDEX IF EXISTS llm_api.idx_conversation_shares_conversation_id;
DROP INDEX IF EXISTS llm_api.idx_conversation_shares_owner_user_id;
DROP INDEX IF EXISTS llm_api.idx_conversation_shares_active;
DROP INDEX IF EXISTS llm_api.idx_conversation_shares_deleted_at;

-- Drop table
DROP TABLE IF EXISTS llm_api.conversation_shares;
