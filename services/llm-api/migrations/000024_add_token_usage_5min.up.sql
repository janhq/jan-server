-- Migration: 000024_add_token_usage_5min
-- Purpose: Add 5-minute bucket aggregation table for detailed usage charts

-- 5-minute bucket aggregation table
CREATE TABLE IF NOT EXISTS llm_api.token_usage_5min (
    id BIGSERIAL PRIMARY KEY,
    bucket_time TIMESTAMP WITH TIME ZONE NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL DEFAULT '',
    model VARCHAR(255) NOT NULL,
    provider VARCHAR(255) NOT NULL,
    total_prompt_tokens BIGINT NOT NULL DEFAULT 0,
    total_completion_tokens BIGINT NOT NULL DEFAULT 0,
    total_tokens BIGINT NOT NULL DEFAULT 0,
    request_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Unique constraint for upsert
CREATE UNIQUE INDEX IF NOT EXISTS uk_token_usage_5min
    ON llm_api.token_usage_5min(bucket_time, user_id, project_id, model, provider);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_token_usage_5min_user_bucket
    ON llm_api.token_usage_5min(user_id, bucket_time);
CREATE INDEX IF NOT EXISTS idx_token_usage_5min_bucket
    ON llm_api.token_usage_5min(bucket_time);

-- Function to update 5-minute bucket aggregates automatically
CREATE OR REPLACE FUNCTION llm_api.update_token_usage_5min()
RETURNS TRIGGER AS $$
DECLARE
    bucket TIMESTAMP WITH TIME ZONE;
BEGIN
    -- Calculate 5-minute bucket: floor to nearest 5-minute interval
    bucket := date_trunc('hour', NEW.created_at) +
              (floor(extract(minute from NEW.created_at) / 5) * interval '5 minutes');

    INSERT INTO llm_api.token_usage_5min (
        bucket_time, user_id, project_id, model, provider,
        total_prompt_tokens, total_completion_tokens, total_tokens,
        request_count, updated_at
    )
    VALUES (
        bucket,
        NEW.user_id,
        COALESCE(NEW.project_id, ''),
        NEW.model,
        NEW.provider,
        NEW.prompt_tokens,
        NEW.completion_tokens,
        NEW.total_tokens,
        1,
        NOW()
    )
    ON CONFLICT (bucket_time, user_id, project_id, model, provider)
    DO UPDATE SET
        total_prompt_tokens = llm_api.token_usage_5min.total_prompt_tokens + EXCLUDED.total_prompt_tokens,
        total_completion_tokens = llm_api.token_usage_5min.total_completion_tokens + EXCLUDED.total_completion_tokens,
        total_tokens = llm_api.token_usage_5min.total_tokens + EXCLUDED.total_tokens,
        request_count = llm_api.token_usage_5min.request_count + 1,
        updated_at = NOW();

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update 5-minute aggregates
DROP TRIGGER IF EXISTS trigger_update_token_usage_5min ON llm_api.token_usage;
CREATE TRIGGER trigger_update_token_usage_5min
    AFTER INSERT ON llm_api.token_usage
    FOR EACH ROW
    EXECUTE FUNCTION llm_api.update_token_usage_5min();

-- Comments
COMMENT ON TABLE llm_api.token_usage_5min IS '5-minute bucket aggregated token usage for detailed charts';
COMMENT ON FUNCTION llm_api.update_token_usage_5min() IS 'Trigger function to automatically aggregate token usage into 5-minute buckets';
