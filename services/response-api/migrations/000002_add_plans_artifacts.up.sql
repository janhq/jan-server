-- ============================================================================
-- Migration: Add Plans, Artifacts, and Idempotency Keys for Agent Responses
-- ============================================================================

SET search_path TO response_api;

-- ============================================================================
-- PLANS
-- ============================================================================
CREATE TABLE response_api.plans (
    id SERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    response_id INTEGER NOT NULL REFERENCES response_api.responses(id),
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    progress DOUBLE PRECISION NOT NULL DEFAULT 0,
    agent_type VARCHAR(32),
    planning_config JSONB,
    estimated_steps INTEGER NOT NULL DEFAULT 0,
    completed_steps INTEGER NOT NULL DEFAULT 0,
    current_task_id INTEGER,
    final_artifact_id INTEGER,
    user_selection JSONB,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_plans_response ON response_api.plans(response_id);
CREATE INDEX idx_plans_status ON response_api.plans(status);
CREATE INDEX idx_plans_agent_type ON response_api.plans(agent_type);

-- ============================================================================
-- PLAN_TASKS
-- ============================================================================
CREATE TABLE response_api.plan_tasks (
    id SERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    plan_id INTEGER NOT NULL REFERENCES response_api.plans(id) ON DELETE CASCADE,
    sequence INTEGER NOT NULL DEFAULT 0,
    task_type VARCHAR(32),
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    title VARCHAR(256) NOT NULL,
    description TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_plan_tasks_plan ON response_api.plan_tasks(plan_id);
CREATE INDEX idx_plan_tasks_status ON response_api.plan_tasks(status);
CREATE INDEX idx_plan_tasks_sequence ON response_api.plan_tasks(plan_id, sequence);

-- ============================================================================
-- PLAN_STEPS
-- ============================================================================
CREATE TABLE response_api.plan_steps (
    id SERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    task_id INTEGER NOT NULL REFERENCES response_api.plan_tasks(id) ON DELETE CASCADE,
    sequence INTEGER NOT NULL DEFAULT 0,
    action VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    input_params JSONB,
    output_data JSONB,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    error_message TEXT,
    error_severity VARCHAR(32),
    duration_ms BIGINT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_plan_steps_task ON response_api.plan_steps(task_id);
CREATE INDEX idx_plan_steps_status ON response_api.plan_steps(status);
CREATE INDEX idx_plan_steps_sequence ON response_api.plan_steps(task_id, sequence);
CREATE INDEX idx_plan_steps_action ON response_api.plan_steps(action);

-- ============================================================================
-- ARTIFACTS
-- ============================================================================
CREATE TABLE response_api.artifacts (
    id SERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    response_id INTEGER NOT NULL REFERENCES response_api.responses(id),
    plan_id INTEGER REFERENCES response_api.plans(id),
    content_type VARCHAR(32) NOT NULL,
    mime_type VARCHAR(128) NOT NULL,
    title VARCHAR(512) NOT NULL,
    content TEXT,
    storage_path VARCHAR(1024),
    size_bytes BIGINT NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    parent_id INTEGER REFERENCES response_api.artifacts(id),
    is_latest BOOLEAN NOT NULL DEFAULT true,
    retention_policy VARCHAR(32) NOT NULL DEFAULT 'session',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX idx_artifacts_response ON response_api.artifacts(response_id);
CREATE INDEX idx_artifacts_plan ON response_api.artifacts(plan_id);
CREATE INDEX idx_artifacts_parent ON response_api.artifacts(parent_id);
CREATE INDEX idx_artifacts_latest ON response_api.artifacts(is_latest);
CREATE INDEX idx_artifacts_expires ON response_api.artifacts(expires_at);

-- ============================================================================
-- PLAN_STEP_DETAILS
-- ============================================================================
CREATE TABLE response_api.plan_step_details (
    id SERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    step_id INTEGER NOT NULL REFERENCES response_api.plan_steps(id) ON DELETE CASCADE,
    detail_type VARCHAR(32) NOT NULL,
    conversation_item_id INTEGER REFERENCES response_api.conversation_items(id),
    tool_call_id VARCHAR(64),
    tool_execution_id INTEGER REFERENCES response_api.tool_executions(id),
    artifact_id INTEGER REFERENCES response_api.artifacts(id),
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plan_step_details_step ON response_api.plan_step_details(step_id);
CREATE INDEX idx_plan_step_details_type ON response_api.plan_step_details(detail_type);
CREATE INDEX idx_plan_step_details_conv_item ON response_api.plan_step_details(conversation_item_id);

-- ============================================================================
-- IDEMPOTENCY_KEYS
-- ============================================================================
CREATE TABLE response_api.idempotency_keys (
    id SERIAL PRIMARY KEY,
    key VARCHAR(128) NOT NULL UNIQUE,
    user_id VARCHAR(64) NOT NULL,
    request_hash VARCHAR(64),
    response_id INTEGER REFERENCES response_api.responses(id),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_idempotency_expires ON response_api.idempotency_keys(expires_at);
CREATE INDEX idx_idempotency_user ON response_api.idempotency_keys(user_id);

-- ============================================================================
-- ALTER CONVERSATION_ITEMS - Add tree structure and tool fields
-- ============================================================================
ALTER TABLE response_api.conversation_items 
    ADD COLUMN IF NOT EXISTS parent_item_id INTEGER REFERENCES response_api.conversation_items(id),
    ADD COLUMN IF NOT EXISTS root_item_id INTEGER REFERENCES response_api.conversation_items(id),
    ADD COLUMN IF NOT EXISTS depth INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS path TEXT;

ALTER TABLE response_api.conversation_items
    ADD COLUMN IF NOT EXISTS tool_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS tool_call_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS tool_arguments JSONB,
    ADD COLUMN IF NOT EXISTS tool_result JSONB;

ALTER TABLE response_api.conversation_items
    ADD COLUMN IF NOT EXISTS error_code VARCHAR(50),
    ADD COLUMN IF NOT EXISTS error_message TEXT,
    ADD COLUMN IF NOT EXISTS error_details JSONB;

ALTER TABLE response_api.conversation_items
    ADD COLUMN IF NOT EXISTS latency_ms INTEGER,
    ADD COLUMN IF NOT EXISTS tokens_used INTEGER;

ALTER TABLE response_api.conversation_items
    ADD COLUMN IF NOT EXISTS provider VARCHAR(50);

CREATE INDEX IF NOT EXISTS idx_conv_items_parent ON response_api.conversation_items(parent_item_id);
CREATE INDEX IF NOT EXISTS idx_conv_items_root ON response_api.conversation_items(root_item_id);
CREATE INDEX IF NOT EXISTS idx_conv_items_path ON response_api.conversation_items(path);
CREATE INDEX IF NOT EXISTS idx_conv_items_tool_call ON response_api.conversation_items(tool_call_id);
CREATE INDEX IF NOT EXISTS idx_conv_items_tool_name ON response_api.conversation_items(tool_name);
