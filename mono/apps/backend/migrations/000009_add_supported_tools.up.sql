-- Migration 000009: Add supports_tools and OpenAI-compatible conversation item fields
-- Part 1: Add supports_tools column to model_catalogs table
-- This field indicates whether a model supports tool/function calling
-- Default is true for all models
ALTER TABLE llm_api.model_catalogs 
    ADD COLUMN IF NOT EXISTS supports_tools BOOLEAN NOT NULL DEFAULT true;

-- Create index for filtering models by tool support
CREATE INDEX IF NOT EXISTS idx_model_catalogs_supports_tools 
    ON llm_api.model_catalogs(supports_tools) WHERE supports_tools = true;

-- Add comment for documentation
COMMENT ON COLUMN llm_api.model_catalogs.supports_tools IS 'Model supports tool/function calling capabilities';

-- Part 2: Add OpenAI-compatible fields to conversation_items table
-- Purpose: Add fields for OpenAI-compatible conversation item types (MCP, shell, computer use, etc.)

-- Add new columns for OpenAI-specific item types
ALTER TABLE llm_api.conversation_items
ADD COLUMN IF NOT EXISTS call_id VARCHAR(50),
ADD COLUMN IF NOT EXISTS server_label VARCHAR(255),
ADD COLUMN IF NOT EXISTS approval_request_id VARCHAR(50),
ADD COLUMN IF NOT EXISTS arguments TEXT,
ADD COLUMN IF NOT EXISTS output TEXT,
ADD COLUMN IF NOT EXISTS error TEXT,
ADD COLUMN IF NOT EXISTS action JSONB,
ADD COLUMN IF NOT EXISTS tools JSONB,
ADD COLUMN IF NOT EXISTS pending_safety_checks JSONB,
ADD COLUMN IF NOT EXISTS acknowledged_safety_checks JSONB,
ADD COLUMN IF NOT EXISTS approve BOOLEAN,
ADD COLUMN IF NOT EXISTS reason TEXT,
ADD COLUMN IF NOT EXISTS commands JSONB,
ADD COLUMN IF NOT EXISTS max_output_length BIGINT,
ADD COLUMN IF NOT EXISTS shell_outputs JSONB,
ADD COLUMN IF NOT EXISTS operation JSONB;

-- Add indexes for frequently queried fields
CREATE INDEX IF NOT EXISTS idx_conversation_items_call_id ON llm_api.conversation_items(call_id) WHERE call_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_conversation_items_server_label ON llm_api.conversation_items(server_label) WHERE server_label IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_conversation_items_approval_request_id ON llm_api.conversation_items(approval_request_id) WHERE approval_request_id IS NOT NULL;

-- Add comments to document the new fields
COMMENT ON COLUMN llm_api.conversation_items.call_id IS 'Tool call ID for function/tool calls (OpenAI-compatible)';
COMMENT ON COLUMN llm_api.conversation_items.server_label IS 'MCP server label for MCP tool calls';
COMMENT ON COLUMN llm_api.conversation_items.approval_request_id IS 'MCP approval request ID';
COMMENT ON COLUMN llm_api.conversation_items.arguments IS 'Tool call arguments (JSON string)';
COMMENT ON COLUMN llm_api.conversation_items.output IS 'Tool call output';
COMMENT ON COLUMN llm_api.conversation_items.error IS 'Tool call error message';
COMMENT ON COLUMN llm_api.conversation_items.action IS 'Action details for computer use or shell calls (JSONB)';
COMMENT ON COLUMN llm_api.conversation_items.tools IS 'List of MCP tools (JSONB array)';
COMMENT ON COLUMN llm_api.conversation_items.pending_safety_checks IS 'Pending safety checks for computer use (JSONB array)';
COMMENT ON COLUMN llm_api.conversation_items.acknowledged_safety_checks IS 'Acknowledged safety checks (JSONB array)';
COMMENT ON COLUMN llm_api.conversation_items.approve IS 'Approval decision for MCP approval response';
COMMENT ON COLUMN llm_api.conversation_items.reason IS 'Reason for MCP approval decision';
COMMENT ON COLUMN llm_api.conversation_items.commands IS 'Shell commands (JSONB array)';
COMMENT ON COLUMN llm_api.conversation_items.max_output_length IS 'Maximum output length for shell calls';
COMMENT ON COLUMN llm_api.conversation_items.shell_outputs IS 'Shell command outputs (JSONB array)';
COMMENT ON COLUMN llm_api.conversation_items.operation IS 'Patch operation details (JSONB)';
