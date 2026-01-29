-- Create schema
CREATE SCHEMA IF NOT EXISTS response_api;

-- Set search path to response_api schema
SET search_path TO response_api;

-- ============================================================================
-- CONVERSATIONS
-- ============================================================================
CREATE TABLE response_api.conversations (
    id SERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    user_id VARCHAR(64),
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conversations_user_id ON response_api.conversations(user_id);
CREATE INDEX idx_conversations_created_at ON response_api.conversations(created_at);

-- ============================================================================
-- RESPONSES
-- ============================================================================
CREATE TABLE response_api.responses (
    id SERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    user_id VARCHAR(64),
    model VARCHAR(128),
    system_prompt TEXT,
    input JSONB,
    output JSONB,
    status VARCHAR(32) NOT NULL,
    stream BOOLEAN NOT NULL DEFAULT false,
    background BOOLEAN NOT NULL DEFAULT false,
    store BOOLEAN NOT NULL DEFAULT false,
    api_key TEXT,
    metadata JSONB,
    usage JSONB,
    error JSONB,
    conversation_id INTEGER REFERENCES response_api.conversations(id),
    previous_response_id VARCHAR(64),
    object VARCHAR(32) NOT NULL DEFAULT 'response',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    queued_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ
);

CREATE INDEX idx_responses_user_id ON response_api.responses(user_id);
CREATE INDEX idx_responses_status ON response_api.responses(status);
CREATE INDEX idx_responses_conversation_id ON response_api.responses(conversation_id);
CREATE INDEX idx_responses_created_at ON response_api.responses(created_at);
CREATE INDEX idx_responses_background_status ON response_api.responses(background, status) WHERE background = true;
CREATE INDEX idx_responses_queued_at ON response_api.responses(queued_at) WHERE status = 'queued';

-- ============================================================================
-- CONVERSATION ITEMS
-- ============================================================================
CREATE TABLE response_api.conversation_items (
    id SERIAL PRIMARY KEY,
    conversation_id INTEGER NOT NULL REFERENCES response_api.conversations(id) ON DELETE CASCADE,
    role VARCHAR(32) NOT NULL,
    status VARCHAR(32),
    content JSONB,
    sequence INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conversation_items_conversation_id ON response_api.conversation_items(conversation_id);
CREATE INDEX idx_conversation_items_sequence ON response_api.conversation_items(conversation_id, sequence);

-- ============================================================================
-- TOOL EXECUTIONS
-- ============================================================================
CREATE TABLE response_api.tool_executions (
    id SERIAL PRIMARY KEY,
    response_id INTEGER NOT NULL REFERENCES response_api.responses(id) ON DELETE CASCADE,
    call_id VARCHAR(64),
    tool_name VARCHAR(128) NOT NULL,
    arguments JSONB,
    result JSONB,
    status VARCHAR(32) NOT NULL,
    error_message TEXT,
    execution_order INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tool_executions_response_id ON response_api.tool_executions(response_id);
CREATE INDEX idx_tool_executions_execution_order ON response_api.tool_executions(response_id, execution_order);
CREATE INDEX idx_tool_executions_status ON response_api.tool_executions(status);
