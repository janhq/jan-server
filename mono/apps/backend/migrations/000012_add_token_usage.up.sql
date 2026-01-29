-- Migration: 000011_add_token_usage
-- Purpose: Add token usage tracking tables for LLM usage analytics and billing

-- Main token usage table - records each LLM request
CREATE TABLE IF NOT EXISTS llm_api.token_usage (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255),
    conversation_id VARCHAR(255),
    model VARCHAR(255) NOT NULL,
    provider VARCHAR(255) NOT NULL,
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    estimated_cost_usd DECIMAL(10, 6),
    request_id VARCHAR(255),
    stream BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_token_usage_user_id ON llm_api.token_usage(user_id);
CREATE INDEX IF NOT EXISTS idx_token_usage_project_id ON llm_api.token_usage(project_id);
CREATE INDEX IF NOT EXISTS idx_token_usage_created_at ON llm_api.token_usage(created_at);
CREATE INDEX IF NOT EXISTS idx_token_usage_model ON llm_api.token_usage(model);
CREATE INDEX IF NOT EXISTS idx_token_usage_provider ON llm_api.token_usage(provider);
CREATE INDEX IF NOT EXISTS idx_token_usage_user_created ON llm_api.token_usage(user_id, created_at);

-- Aggregated daily usage for faster analytics queries
CREATE TABLE IF NOT EXISTS llm_api.token_usage_daily (
    id BIGSERIAL PRIMARY KEY,
    usage_date DATE NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL DEFAULT '',
    model VARCHAR(255) NOT NULL,
    provider VARCHAR(255) NOT NULL,
    total_prompt_tokens BIGINT NOT NULL DEFAULT 0,
    total_completion_tokens BIGINT NOT NULL DEFAULT 0,
    total_tokens BIGINT NOT NULL DEFAULT 0,
    request_count INTEGER NOT NULL DEFAULT 0,
    estimated_cost_usd DECIMAL(12, 6),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Unique index for daily aggregates (project_id defaults to empty string)
CREATE UNIQUE INDEX IF NOT EXISTS uk_token_usage_daily 
ON llm_api.token_usage_daily(usage_date, user_id, project_id, model, provider);

-- Indexes for daily aggregates
CREATE INDEX IF NOT EXISTS idx_token_usage_daily_date ON llm_api.token_usage_daily(usage_date);
CREATE INDEX IF NOT EXISTS idx_token_usage_daily_user_id ON llm_api.token_usage_daily(user_id);
CREATE INDEX IF NOT EXISTS idx_token_usage_daily_user_date ON llm_api.token_usage_daily(user_id, usage_date);

-- Function to update daily aggregates automatically
CREATE OR REPLACE FUNCTION llm_api.update_token_usage_daily()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO llm_api.token_usage_daily (
        usage_date, user_id, project_id, model, provider,
        total_prompt_tokens, total_completion_tokens, total_tokens,
        request_count, estimated_cost_usd, updated_at
    )
    VALUES (
        DATE(NEW.created_at),
        NEW.user_id,
        COALESCE(NEW.project_id, ''),
        NEW.model,
        NEW.provider,
        NEW.prompt_tokens,
        NEW.completion_tokens,
        NEW.total_tokens,
        1,
        COALESCE(NEW.estimated_cost_usd, 0),
        NOW()
    )
    ON CONFLICT (usage_date, user_id, project_id, model, provider)
    DO UPDATE SET
        total_prompt_tokens = llm_api.token_usage_daily.total_prompt_tokens + EXCLUDED.total_prompt_tokens,
        total_completion_tokens = llm_api.token_usage_daily.total_completion_tokens + EXCLUDED.total_completion_tokens,
        total_tokens = llm_api.token_usage_daily.total_tokens + EXCLUDED.total_tokens,
        request_count = llm_api.token_usage_daily.request_count + 1,
        estimated_cost_usd = llm_api.token_usage_daily.estimated_cost_usd + EXCLUDED.estimated_cost_usd,
        updated_at = NOW();
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update daily aggregates
DROP TRIGGER IF EXISTS trigger_update_token_usage_daily ON llm_api.token_usage;
CREATE TRIGGER trigger_update_token_usage_daily
    AFTER INSERT ON llm_api.token_usage
    FOR EACH ROW
    EXECUTE FUNCTION llm_api.update_token_usage_daily();

-- Comments
COMMENT ON TABLE llm_api.token_usage IS 'Records individual LLM request token usage for billing and analytics';
COMMENT ON TABLE llm_api.token_usage_daily IS 'Daily aggregated token usage for efficient analytics queries';
COMMENT ON FUNCTION llm_api.update_token_usage_daily() IS 'Trigger function to automatically aggregate token usage into daily table';
