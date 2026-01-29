-- Rollback Migration 000009: Remove supports_tools and OpenAI-compatible fields

-- Part 1: Remove OpenAI-compatible fields from conversation_items table
-- Drop indexes first
DROP INDEX IF EXISTS llm_api.idx_conversation_items_approval_request_id;
DROP INDEX IF EXISTS llm_api.idx_conversation_items_server_label;
DROP INDEX IF EXISTS llm_api.idx_conversation_items_call_id;

-- Drop columns
ALTER TABLE llm_api.conversation_items
DROP COLUMN IF EXISTS operation,
DROP COLUMN IF EXISTS shell_outputs,
DROP COLUMN IF EXISTS max_output_length,
DROP COLUMN IF EXISTS commands,
DROP COLUMN IF EXISTS reason,
DROP COLUMN IF EXISTS approve,
DROP COLUMN IF EXISTS acknowledged_safety_checks,
DROP COLUMN IF EXISTS pending_safety_checks,
DROP COLUMN IF EXISTS tools,
DROP COLUMN IF EXISTS action,
DROP COLUMN IF EXISTS error,
DROP COLUMN IF EXISTS output,
DROP COLUMN IF EXISTS arguments,
DROP COLUMN IF EXISTS approval_request_id,
DROP COLUMN IF EXISTS server_label,
DROP COLUMN IF EXISTS call_id;

-- Part 2: Remove supports_tools from model_catalogs
-- Drop the supports_tools index
DROP INDEX IF EXISTS llm_api.idx_model_catalogs_supports_tools;

-- Drop the supports_tools column
ALTER TABLE llm_api.model_catalogs DROP COLUMN IF EXISTS supports_tools;
