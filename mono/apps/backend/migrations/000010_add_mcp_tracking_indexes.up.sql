-- Migration: 000010_add_mcp_tracking_indexes
-- Purpose: Add composite index for MCP tool call tracking
-- This improves query performance when looking up items by call_id within a conversation

-- Add composite index on (conversation_id, call_id) for efficient lookup
-- This is used by the PATCH endpoint to update MCP tool call results
CREATE INDEX IF NOT EXISTS idx_conversation_items_conv_call_id 
ON llm_api.conversation_items(conversation_id, call_id) 
WHERE call_id IS NOT NULL;

-- Add index on status for filtering pending/in_progress tool calls
CREATE INDEX IF NOT EXISTS idx_conversation_items_status 
ON llm_api.conversation_items(status) 
WHERE status IN ('in_progress', 'incomplete');

-- Comment on indexes
COMMENT ON INDEX llm_api.idx_conversation_items_conv_call_id IS 'Composite index for efficient MCP tool call lookup by conversation and call_id';
COMMENT ON INDEX llm_api.idx_conversation_items_status IS 'Partial index for filtering pending/in-progress tool calls';
