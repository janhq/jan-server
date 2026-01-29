-- Migration: 000011_conversation_shares
-- Purpose: Add conversation sharing (public links) feature
-- Allows users to generate public, unlisted links for conversations or single messages

-- Set search path
SET search_path TO llm_api;

-- ============================================================================
-- CONVERSATION SHARES TABLE
-- ============================================================================
CREATE TABLE llm_api.conversation_shares (
    id SERIAL PRIMARY KEY,
    
    -- Public identifiers
    public_id VARCHAR(64) NOT NULL,
    slug VARCHAR(30) NOT NULL, -- 22-char base62 + some padding, cryptographically random
    
    -- Relationships (cascade on delete)
    conversation_id INTEGER NOT NULL REFERENCES llm_api.conversations(id) ON DELETE CASCADE,
    owner_user_id INTEGER NOT NULL REFERENCES llm_api.users(id) ON DELETE CASCADE,
    
    -- Share scope
    item_public_id VARCHAR(64), -- Nullable: if set, single-message share; otherwise full conversation
    
    -- Display
    title VARCHAR(256),
    
    -- Visibility (unlisted only for now, expandable later)
    visibility VARCHAR(20) NOT NULL DEFAULT 'unlisted',
    
    -- Revocation
    revoked_at TIMESTAMPTZ,
    
    -- Analytics
    view_count INTEGER NOT NULL DEFAULT 0,
    last_viewed_at TIMESTAMPTZ,
    
    -- Snapshot versioning
    snapshot_version INTEGER NOT NULL DEFAULT 1,
    
    -- The sanitized snapshot payload (max 10MB enforced at application level)
    snapshot JSONB NOT NULL,
    
    -- Share options (e.g., include_images, include_context_messages)
    share_options JSONB,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    -- Constraints
    CONSTRAINT conversation_shares_public_id_unique UNIQUE (public_id),
    CONSTRAINT conversation_shares_slug_unique UNIQUE (slug),
    CONSTRAINT conversation_shares_visibility_check CHECK (visibility IN ('unlisted', 'private', 'public'))
);

-- ============================================================================
-- INDEXES
-- ============================================================================

-- Primary lookup: by slug for public access
CREATE INDEX idx_conversation_shares_slug ON llm_api.conversation_shares(slug) WHERE deleted_at IS NULL;

-- List shares by conversation
CREATE INDEX idx_conversation_shares_conversation_id ON llm_api.conversation_shares(conversation_id) WHERE deleted_at IS NULL;

-- List shares by owner
CREATE INDEX idx_conversation_shares_owner_user_id ON llm_api.conversation_shares(owner_user_id) WHERE deleted_at IS NULL;

-- Filter active (non-revoked) shares
CREATE INDEX idx_conversation_shares_active ON llm_api.conversation_shares(conversation_id, revoked_at) WHERE deleted_at IS NULL AND revoked_at IS NULL;

-- Soft delete
CREATE INDEX idx_conversation_shares_deleted_at ON llm_api.conversation_shares(deleted_at);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Auto-update updated_at timestamp
CREATE TRIGGER conversation_shares_updated_at
    BEFORE UPDATE ON llm_api.conversation_shares
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE llm_api.conversation_shares IS 'Stores public share links for conversations with sanitized snapshots';
COMMENT ON COLUMN llm_api.conversation_shares.slug IS 'Cryptographically random 22-char base62 identifier for public URLs';
COMMENT ON COLUMN llm_api.conversation_shares.snapshot IS 'Sanitized JSON payload containing only public-safe content (max 10MB)';
COMMENT ON COLUMN llm_api.conversation_shares.item_public_id IS 'If set, share is for a single message; includes preceding user prompt for context';
COMMENT ON COLUMN llm_api.conversation_shares.visibility IS 'Currently only unlisted is supported; public/private for future expansion';
