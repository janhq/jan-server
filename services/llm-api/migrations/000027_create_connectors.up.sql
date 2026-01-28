-- Connectors: OAuth connections to external services (GitHub, Gmail, Google Drive, Google Calendar)
CREATE SCHEMA IF NOT EXISTS llm_api;

-- Connector types enum
DO $$ BEGIN
    CREATE TYPE llm_api.connector_type AS ENUM ('github', 'gmail', 'google_drive', 'google_calendar');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- User connector connections
CREATE TABLE IF NOT EXISTS llm_api.connector_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL,
    connector_type llm_api.connector_type NOT NULL,

    -- OAuth tokens (encrypted with AES-256-GCM)
    access_token_encrypted TEXT NOT NULL,
    refresh_token_encrypted TEXT,
    encryption_key_id VARCHAR(32) NOT NULL DEFAULT 'v1',
    token_type VARCHAR(32) DEFAULT 'Bearer',
    expires_at TIMESTAMPTZ,

    -- Provider-specific user info
    provider_user_id VARCHAR(255),
    provider_username VARCHAR(255),
    provider_email VARCHAR(255),
    provider_avatar_url TEXT,

    -- Scopes granted
    scopes TEXT[],

    -- Status
    is_connected BOOLEAN NOT NULL DEFAULT true,
    last_sync_at TIMESTAMPTZ,
    last_error TEXT,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure one connection per connector type per user
    UNIQUE(user_id, connector_type)
);

-- Indexes for quick lookups
CREATE INDEX IF NOT EXISTS idx_connector_connections_user_type
    ON llm_api.connector_connections(user_id, connector_type);
CREATE INDEX IF NOT EXISTS idx_connector_connections_connected
    ON llm_api.connector_connections(is_connected) WHERE is_connected = true;
CREATE INDEX IF NOT EXISTS idx_connector_connections_expires
    ON llm_api.connector_connections(expires_at) WHERE expires_at IS NOT NULL;

-- Connector OAuth states (for CSRF protection during OAuth flow)
CREATE TABLE IF NOT EXISTS llm_api.connector_oauth_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL,
    connector_type llm_api.connector_type NOT NULL,
    state VARCHAR(128) NOT NULL UNIQUE,
    state_hash VARCHAR(64) NOT NULL, -- HMAC hash for verification
    code_verifier VARCHAR(128), -- For PKCE
    redirect_uri TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auto-cleanup expired states
CREATE INDEX IF NOT EXISTS idx_connector_oauth_states_expires
    ON llm_api.connector_oauth_states(expires_at);
CREATE INDEX IF NOT EXISTS idx_connector_oauth_states_state
    ON llm_api.connector_oauth_states(state);

-- Audit log for connector operations (security tracking)
CREATE TABLE IF NOT EXISTS llm_api.connector_audit_log (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    connector_type llm_api.connector_type NOT NULL,
    action VARCHAR(64) NOT NULL, -- 'connect', 'disconnect', 'refresh', 'api_call', 'write_operation'
    tool_name VARCHAR(128),
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    ip_address INET,
    user_agent TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_connector_audit_log_user
    ON llm_api.connector_audit_log(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_connector_audit_log_action
    ON llm_api.connector_audit_log(action, created_at DESC);

-- Comments
COMMENT ON TABLE llm_api.connector_connections IS 'OAuth connections to external services (GitHub, Gmail, Drive, Calendar)';
COMMENT ON COLUMN llm_api.connector_connections.access_token_encrypted IS 'AES-256-GCM encrypted access token';
COMMENT ON COLUMN llm_api.connector_connections.refresh_token_encrypted IS 'AES-256-GCM encrypted refresh token';
COMMENT ON COLUMN llm_api.connector_connections.encryption_key_id IS 'Key ID for token encryption (supports key rotation)';
COMMENT ON TABLE llm_api.connector_oauth_states IS 'Temporary OAuth state storage for CSRF protection';
COMMENT ON TABLE llm_api.connector_audit_log IS 'Security audit log for connector operations';
