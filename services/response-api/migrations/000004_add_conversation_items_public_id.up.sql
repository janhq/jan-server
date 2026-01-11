-- +goose Up
-- Add missing public_id column to conversation_items table

SET search_path TO response_api;

-- Add public_id column to conversation_items
ALTER TABLE response_api.conversation_items 
ADD COLUMN IF NOT EXISTS public_id VARCHAR(50);

-- Add other missing columns from the entity
ALTER TABLE response_api.conversation_items 
ADD COLUMN IF NOT EXISTS object VARCHAR(50) NOT NULL DEFAULT 'conversation.item',
ADD COLUMN IF NOT EXISTS branch VARCHAR(50) NOT NULL DEFAULT 'MAIN',
ADD COLUMN IF NOT EXISTS type VARCHAR(50),
ADD COLUMN IF NOT EXISTS incomplete_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS incomplete_details JSONB,
ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS response_id INTEGER REFERENCES response_api.responses(id),
ADD COLUMN IF NOT EXISTS rating VARCHAR(10),
ADD COLUMN IF NOT EXISTS rated_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS rating_comment TEXT,
ADD COLUMN IF NOT EXISTS call_id VARCHAR(50),
ADD COLUMN IF NOT EXISTS name VARCHAR(255),
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

-- Create unique index on public_id (only for non-empty values since existing rows won't have it)
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_items_public_id ON response_api.conversation_items(public_id) WHERE public_id IS NOT NULL AND public_id != '';

-- Create other indexes for commonly queried columns
CREATE INDEX IF NOT EXISTS idx_item_conversation_branch ON response_api.conversation_items(conversation_id, branch);
CREATE INDEX IF NOT EXISTS idx_item_conversation_sequence ON response_api.conversation_items(conversation_id, sequence);
CREATE INDEX IF NOT EXISTS idx_conversation_items_call_id ON response_api.conversation_items(call_id);
CREATE INDEX IF NOT EXISTS idx_conversation_items_server_label ON response_api.conversation_items(server_label);
CREATE INDEX IF NOT EXISTS idx_conversation_items_approval_request_id ON response_api.conversation_items(approval_request_id);
CREATE INDEX IF NOT EXISTS idx_conversation_items_response_id ON response_api.conversation_items(response_id);
