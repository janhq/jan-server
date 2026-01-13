-- +goose Down
-- Remove columns added to conversation_items

SET search_path TO response_api;

-- Drop indexes first
DROP INDEX IF EXISTS response_api.idx_conversation_items_public_id;
DROP INDEX IF EXISTS response_api.idx_item_conversation_branch;
DROP INDEX IF EXISTS response_api.idx_item_conversation_sequence;
DROP INDEX IF EXISTS response_api.idx_conversation_items_call_id;
DROP INDEX IF EXISTS response_api.idx_conversation_items_server_label;
DROP INDEX IF EXISTS response_api.idx_conversation_items_approval_request_id;
DROP INDEX IF EXISTS response_api.idx_conversation_items_response_id;

-- Drop columns
ALTER TABLE response_api.conversation_items 
DROP COLUMN IF EXISTS public_id,
DROP COLUMN IF EXISTS object,
DROP COLUMN IF EXISTS branch,
DROP COLUMN IF EXISTS type,
DROP COLUMN IF EXISTS incomplete_at,
DROP COLUMN IF EXISTS incomplete_details,
DROP COLUMN IF EXISTS completed_at,
DROP COLUMN IF EXISTS response_id,
DROP COLUMN IF EXISTS rating,
DROP COLUMN IF EXISTS rated_at,
DROP COLUMN IF EXISTS rating_comment,
DROP COLUMN IF EXISTS call_id,
DROP COLUMN IF EXISTS name,
DROP COLUMN IF EXISTS server_label,
DROP COLUMN IF EXISTS approval_request_id,
DROP COLUMN IF EXISTS arguments,
DROP COLUMN IF EXISTS output,
DROP COLUMN IF EXISTS error,
DROP COLUMN IF EXISTS action,
DROP COLUMN IF EXISTS tools,
DROP COLUMN IF EXISTS pending_safety_checks,
DROP COLUMN IF EXISTS acknowledged_safety_checks,
DROP COLUMN IF EXISTS approve,
DROP COLUMN IF EXISTS reason,
DROP COLUMN IF EXISTS commands,
DROP COLUMN IF EXISTS max_output_length,
DROP COLUMN IF EXISTS shell_outputs,
DROP COLUMN IF EXISTS operation;
