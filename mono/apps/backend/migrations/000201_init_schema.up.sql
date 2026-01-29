-- Create schema
CREATE SCHEMA IF NOT EXISTS media_api;

-- Set search path to media_api schema
SET search_path TO media_api;

-- ============================================================================
-- MEDIA OBJECTS
-- ============================================================================
CREATE TABLE media_api.media_objects (
    id VARCHAR(40) PRIMARY KEY,
    storage_provider VARCHAR(32) NOT NULL,
    storage_key VARCHAR(255) NOT NULL,
    mime_type VARCHAR(64) NOT NULL,
    bytes BIGINT NOT NULL,
    sha256 CHAR(64) NOT NULL,
    created_by VARCHAR(64),
    retention_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_media_objects_sha256 ON media_api.media_objects(sha256);
CREATE INDEX idx_media_objects_created_by ON media_api.media_objects(created_by);
CREATE INDEX idx_media_objects_created_at ON media_api.media_objects(created_at);
CREATE INDEX idx_media_objects_retention_until ON media_api.media_objects(retention_until) WHERE retention_until IS NOT NULL;
