-- Remove parent_conversation_id from responses table
DROP INDEX IF EXISTS response_api.idx_responses_parent_conversation_id;
ALTER TABLE response_api.responses DROP COLUMN IF EXISTS parent_conversation_id;
