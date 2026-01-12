-- ============================================================================
-- Rollback: Remove planned/actual params, artifacts, and citations
-- ============================================================================

SET search_path TO response_api;

-- Drop indexes
DROP INDEX IF EXISTS response_api.idx_plan_steps_params_mismatch;
DROP INDEX IF EXISTS response_api.idx_responses_artifacts_gin;
DROP INDEX IF EXISTS response_api.idx_responses_citations_gin;

-- Remove columns from plan_steps
ALTER TABLE response_api.plan_steps
    DROP COLUMN IF EXISTS planned_params,
    DROP COLUMN IF EXISTS actual_params;

-- Remove columns from responses
ALTER TABLE response_api.responses
    DROP COLUMN IF EXISTS artifacts,
    DROP COLUMN IF EXISTS citations;
