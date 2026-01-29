-- +goose Up
-- Add missing columns to conversations table to match Go entity

SET search_path TO response_api;

-- Add missing columns
ALTER TABLE response_api.conversations 
ADD COLUMN IF NOT EXISTS object VARCHAR(50) NOT NULL DEFAULT 'conversation',
ADD COLUMN IF NOT EXISTS title VARCHAR(256),
ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active',
ADD COLUMN IF NOT EXISTS active_branch VARCHAR(50) NOT NULL DEFAULT 'MAIN',
ADD COLUMN IF NOT EXISTS referrer VARCHAR(100),
ADD COLUMN IF NOT EXISTS is_private BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS project_id INTEGER,
ADD COLUMN IF NOT EXISTS project_public_id VARCHAR(64),
ADD COLUMN IF NOT EXISTS instruction_version INTEGER NOT NULL DEFAULT 1,
ADD COLUMN IF NOT EXISTS effective_instruction_snapshot TEXT;

-- Create indexes for new columns
CREATE INDEX IF NOT EXISTS idx_conversation_user_referrer ON response_api.conversations(user_id, referrer);
CREATE INDEX IF NOT EXISTS idx_conversation_user_status ON response_api.conversations(user_id, status);
CREATE INDEX IF NOT EXISTS idx_conversations_project_updated_at ON response_api.conversations(project_id, updated_at);
CREATE INDEX IF NOT EXISTS idx_conversations_project_public_id ON response_api.conversations(project_public_id);

-- Create conversation_branches table if not exists
CREATE TABLE IF NOT EXISTS response_api.conversation_branches (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    conversation_id INTEGER NOT NULL REFERENCES response_api.conversations(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    parent_branch VARCHAR(50),
    forked_at TIMESTAMPTZ,
    forked_from_item_id VARCHAR(50),
    item_count INTEGER DEFAULT 0,
    UNIQUE(conversation_id, name)
);

CREATE INDEX IF NOT EXISTS idx_conversation_branch_name ON response_api.conversation_branches(conversation_id, name);

-- Add missing columns to conversation_items if needed
ALTER TABLE response_api.conversation_items 
ADD COLUMN IF NOT EXISTS sequence_number INTEGER,
ADD COLUMN IF NOT EXISTS legacy_content JSONB;

-- Update sequence_number to match sequence for existing rows (batch update for production safety)
-- This is safe because it only updates NULL values
UPDATE response_api.conversation_items 
SET sequence_number = sequence 
WHERE sequence_number IS NULL 
  AND id IN (SELECT id FROM response_api.conversation_items WHERE sequence_number IS NULL LIMIT 10000);

-- +goose StatementEnd
