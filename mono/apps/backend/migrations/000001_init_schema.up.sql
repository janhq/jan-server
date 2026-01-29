CREATE SCHEMA IF NOT EXISTS llm_api;

-- Set search path to llm_api schema
SET search_path TO llm_api;

-- ============================================================================
-- USERS
-- ============================================================================
CREATE TABLE llm_api.users (
    id SERIAL PRIMARY KEY,
    auth_provider VARCHAR(50) NOT NULL DEFAULT 'keycloak',
    issuer VARCHAR(255) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    username VARCHAR(150),
    email VARCHAR(320),
    name VARCHAR(255),
    picture VARCHAR(512),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT ux_users_issuer_subject UNIQUE (issuer, subject)
);

CREATE INDEX idx_users_deleted_at ON llm_api.users(deleted_at);

-- ============================================================================
-- PROVIDERS
-- ============================================================================
CREATE TABLE llm_api.providers (
    id SERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    kind VARCHAR(64) NOT NULL,
    base_url VARCHAR(512),
    encrypted_api_key TEXT,
    api_key_hint VARCHAR(128),
    is_moderated BOOLEAN NOT NULL DEFAULT false,
    active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB,
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT providers_public_id_unique UNIQUE (public_id)
);

CREATE INDEX idx_providers_kind ON llm_api.providers(kind);
CREATE INDEX idx_providers_is_moderated ON llm_api.providers(is_moderated);
CREATE INDEX idx_providers_active ON llm_api.providers(active);
CREATE INDEX idx_providers_active_kind ON llm_api.providers(active, kind);
CREATE INDEX idx_providers_deleted_at ON llm_api.providers(deleted_at);

-- ============================================================================
-- MODEL CATALOG
-- ============================================================================
CREATE TABLE llm_api.model_catalogs (
    id SERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL,
    supported_parameters JSONB NOT NULL,
    architecture JSONB NOT NULL,
    tags JSONB,
    notes TEXT,
    is_moderated BOOLEAN,
    active BOOLEAN DEFAULT true,
    status VARCHAR(32) NOT NULL DEFAULT 'init',
    extras JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT model_catalogs_public_id_unique UNIQUE (public_id)
);

CREATE INDEX idx_model_catalogs_is_moderated ON llm_api.model_catalogs(is_moderated);
CREATE INDEX idx_model_catalogs_active ON llm_api.model_catalogs(active);
CREATE INDEX idx_model_catalogs_status ON llm_api.model_catalogs(status);
CREATE INDEX idx_model_catalogs_status_active ON llm_api.model_catalogs(status, active);
CREATE INDEX idx_model_catalogs_deleted_at ON llm_api.model_catalogs(deleted_at);

-- ============================================================================
-- PROVIDER MODELS
-- ============================================================================
CREATE TABLE llm_api.provider_models (
    id SERIAL PRIMARY KEY,
    provider_id INTEGER NOT NULL,
    public_id VARCHAR(64) NOT NULL,
    kind VARCHAR(64) NOT NULL,
    model_catalog_id INTEGER,
    model_public_id VARCHAR(128) NOT NULL,
    provider_original_model_id VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    pricing JSONB NOT NULL,
    token_limits JSONB,
    family VARCHAR(128),
    supports_images BOOLEAN NOT NULL DEFAULT false,
    supports_embeddings BOOLEAN NOT NULL DEFAULT false,
    supports_reasoning BOOLEAN NOT NULL DEFAULT false,
    supports_audio BOOLEAN NOT NULL DEFAULT false,
    supports_video BOOLEAN NOT NULL DEFAULT false,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT provider_models_public_id_unique UNIQUE (public_id),
    CONSTRAINT ux_provider_model_public_id UNIQUE (provider_id, model_public_id),
    CONSTRAINT fk_provider_models_provider FOREIGN KEY (provider_id) REFERENCES llm_api.providers(id) ON DELETE CASCADE,
    CONSTRAINT fk_provider_models_model_catalog FOREIGN KEY (model_catalog_id) REFERENCES llm_api.model_catalogs(id) ON DELETE SET NULL
);

CREATE INDEX idx_provider_models_provider_id ON llm_api.provider_models(provider_id);
CREATE INDEX idx_provider_models_kind ON llm_api.provider_models(kind);
CREATE INDEX idx_provider_models_model_catalog_id ON llm_api.provider_models(model_catalog_id);
CREATE INDEX idx_provider_models_model_public_id ON llm_api.provider_models(model_public_id);
CREATE INDEX idx_provider_models_active ON llm_api.provider_models(active);
CREATE INDEX idx_provider_models_provider_active ON llm_api.provider_models(provider_id, active);
CREATE INDEX idx_provider_models_catalog_active ON llm_api.provider_models(model_catalog_id, active);
CREATE INDEX idx_provider_models_deleted_at ON llm_api.provider_models(deleted_at);

-- ============================================================================
-- MODELS (Legacy table)
-- ============================================================================
CREATE TABLE llm_api.models (
    id VARCHAR(255) PRIMARY KEY,
    provider VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    family VARCHAR(255),
    capabilities JSONB,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- CONVERSATIONS
-- ============================================================================
CREATE TABLE llm_api.conversations (
    id SERIAL PRIMARY KEY,
    public_id VARCHAR(50) NOT NULL,
    object VARCHAR(50) NOT NULL DEFAULT 'conversation',
    title VARCHAR(256),
    user_id INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    active_branch VARCHAR(50) NOT NULL DEFAULT 'MAIN',
    referrer VARCHAR(100),
    metadata JSONB,
    is_private BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT conversations_public_id_unique UNIQUE (public_id),
    CONSTRAINT fk_conversations_user FOREIGN KEY (user_id) REFERENCES llm_api.users(id)
);

CREATE INDEX idx_conversations_user_id_referrer ON llm_api.conversations(user_id, referrer);
CREATE INDEX idx_conversations_user_id_status ON llm_api.conversations(user_id, status);
CREATE INDEX idx_conversations_deleted_at ON llm_api.conversations(deleted_at);

-- ============================================================================
-- CONVERSATION BRANCHES
-- ============================================================================
CREATE TABLE llm_api.conversation_branches (
    id SERIAL PRIMARY KEY,
    conversation_id INTEGER NOT NULL,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    parent_branch VARCHAR(50),
    forked_at TIMESTAMPTZ,
    forked_from_item_id VARCHAR(50),
    item_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT idx_conversation_branch_name UNIQUE (conversation_id, name),
    CONSTRAINT fk_conversation_branches_conversation FOREIGN KEY (conversation_id) REFERENCES llm_api.conversations(id) ON DELETE CASCADE
);

CREATE INDEX idx_conversation_branches_deleted_at ON llm_api.conversation_branches(deleted_at);

-- ============================================================================
-- CONVERSATION ITEMS
-- ============================================================================
CREATE TABLE llm_api.conversation_items (
    id SERIAL PRIMARY KEY,
    conversation_id INTEGER NOT NULL,
    public_id VARCHAR(50) NOT NULL,
    object VARCHAR(50) NOT NULL DEFAULT 'conversation.item',
    branch VARCHAR(50) NOT NULL DEFAULT 'MAIN',
    sequence_number INTEGER NOT NULL,
    type VARCHAR(50) NOT NULL,
    role VARCHAR(20),
    content JSONB,
    status VARCHAR(20),
    incomplete_at TIMESTAMPTZ,
    incomplete_details JSONB,
    completed_at TIMESTAMPTZ,
    response_id INTEGER,
    rating VARCHAR(10),
    rated_at TIMESTAMPTZ,
    rating_comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT conversation_items_public_id_unique UNIQUE (public_id),
    CONSTRAINT fk_conversation_items_conversation FOREIGN KEY (conversation_id) REFERENCES llm_api.conversations(id) ON DELETE CASCADE
);

CREATE INDEX idx_conversation_items_conversation_id_branch ON llm_api.conversation_items(conversation_id, branch);
CREATE INDEX idx_conversation_items_conversation_id_sequence ON llm_api.conversation_items(conversation_id, sequence_number);
CREATE INDEX idx_conversation_items_response_id ON llm_api.conversation_items(response_id);
CREATE INDEX idx_conversation_items_deleted_at ON llm_api.conversation_items(deleted_at);

-- ============================================================================
-- API KEYS
-- ============================================================================
CREATE TABLE llm_api.api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INTEGER NOT NULL,
    name VARCHAR(128) NOT NULL,
    prefix VARCHAR(32) NOT NULL,
    suffix VARCHAR(8) NOT NULL,
    hash VARCHAR(128) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_api_keys_user FOREIGN KEY (user_id) REFERENCES llm_api.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_api_keys_user_id ON llm_api.api_keys(user_id);
CREATE INDEX idx_api_keys_expires_at ON llm_api.api_keys(expires_at);
CREATE INDEX idx_api_keys_prefix ON llm_api.api_keys(prefix);
CREATE INDEX idx_api_keys_user_id_revoked_at ON llm_api.api_keys(user_id, revoked_at);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at triggers
CREATE TRIGGER users_updated_at
    BEFORE UPDATE ON llm_api.users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER providers_updated_at
    BEFORE UPDATE ON llm_api.providers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER model_catalogs_updated_at
    BEFORE UPDATE ON llm_api.model_catalogs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER provider_models_updated_at
    BEFORE UPDATE ON llm_api.provider_models
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER models_updated_at
    BEFORE UPDATE ON llm_api.models
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER conversations_updated_at
    BEFORE UPDATE ON llm_api.conversations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER conversation_branches_updated_at
    BEFORE UPDATE ON llm_api.conversation_branches
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER conversation_items_updated_at
    BEFORE UPDATE ON llm_api.conversation_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER api_keys_updated_at
    BEFORE UPDATE ON llm_api.api_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
