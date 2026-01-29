-- Migration: 000024_add_token_usage_5min (rollback)
-- Purpose: Remove 5-minute bucket aggregation table

DROP TRIGGER IF EXISTS trigger_update_token_usage_5min ON llm_api.token_usage;
DROP FUNCTION IF EXISTS llm_api.update_token_usage_5min();
DROP TABLE IF EXISTS llm_api.token_usage_5min;
