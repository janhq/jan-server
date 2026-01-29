-- Add parent_conversation_id to responses table
-- This stores the llm-api conversation ID that spawned this agent response
-- Used for linking artifacts back to the correct user conversation
ALTER TABLE response_api.responses ADD COLUMN IF NOT EXISTS parent_conversation_id VARCHAR(64);
CREATE INDEX IF NOT EXISTS idx_responses_parent_conversation_id ON response_api.responses(parent_conversation_id);
