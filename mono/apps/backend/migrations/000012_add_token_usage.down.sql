-- Migration: 000011_add_token_usage (DOWN)
-- Purpose: Remove token usage tracking tables

-- Drop trigger first
DROP TRIGGER IF EXISTS trigger_update_token_usage_daily ON llm_api.token_usage;

-- Drop function
DROP FUNCTION IF EXISTS llm_api.update_token_usage_daily();

-- Drop tables
DROP TABLE IF EXISTS llm_api.token_usage_daily;
DROP TABLE IF EXISTS llm_api.token_usage;
