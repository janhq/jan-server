-- Migration: 000010_add_mcp_tracking_indexes (DOWN)
-- Purpose: Remove MCP tool call tracking indexes

DROP INDEX CONCURRENTLY IF EXISTS llm_api.idx_conversation_items_conv_call_id;
DROP INDEX CONCURRENTLY IF EXISTS llm_api.idx_conversation_items_status;
