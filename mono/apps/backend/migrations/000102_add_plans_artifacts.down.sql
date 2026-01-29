-- ============================================================================
-- Rollback: Remove Plans, Artifacts, and Idempotency Keys
-- ============================================================================

SET search_path TO response_api;

-- Remove columns from conversation_items
DROP INDEX IF EXISTS idx_conv_items_parent;
DROP INDEX IF EXISTS idx_conv_items_root;
DROP INDEX IF EXISTS idx_conv_items_path;
DROP INDEX IF EXISTS idx_conv_items_tool_call;
DROP INDEX IF EXISTS idx_conv_items_tool_name;

ALTER TABLE response_api.conversation_items
    DROP COLUMN IF EXISTS parent_item_id,
    DROP COLUMN IF EXISTS root_item_id,
    DROP COLUMN IF EXISTS depth,
    DROP COLUMN IF EXISTS path,
    DROP COLUMN IF EXISTS tool_name,
    DROP COLUMN IF EXISTS tool_call_id,
    DROP COLUMN IF EXISTS tool_arguments,
    DROP COLUMN IF EXISTS tool_result,
    DROP COLUMN IF EXISTS error_code,
    DROP COLUMN IF EXISTS error_message,
    DROP COLUMN IF EXISTS error_details,
    DROP COLUMN IF EXISTS latency_ms,
    DROP COLUMN IF EXISTS tokens_used,
    DROP COLUMN IF EXISTS provider;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS response_api.plan_step_details;
DROP TABLE IF EXISTS response_api.artifacts;
DROP TABLE IF EXISTS response_api.plan_steps;
DROP TABLE IF EXISTS response_api.plan_tasks;
DROP TABLE IF EXISTS response_api.plans;
DROP TABLE IF EXISTS response_api.idempotency_keys;
