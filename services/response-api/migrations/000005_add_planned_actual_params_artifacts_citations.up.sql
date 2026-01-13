-- ============================================================================
-- Migration: Add planned/actual params, artifacts, and citations for Response API fixes
-- Fixes: duplicate outputs, missing step results, input/output mismatches, artifact handling
-- ============================================================================

SET search_path TO response_api;

-- ============================================================================
-- FIX 3: Add planned_params and actual_params to plan_steps
-- Tracks both the original plan and what actually executed
-- ============================================================================
ALTER TABLE response_api.plan_steps
    ADD COLUMN IF NOT EXISTS planned_params JSONB,
    ADD COLUMN IF NOT EXISTS actual_params JSONB;

-- Migrate existing input_params to planned_params for backward compatibility
UPDATE response_api.plan_steps 
SET planned_params = input_params 
WHERE planned_params IS NULL AND input_params IS NOT NULL;

-- ============================================================================
-- FIX 4 & 5: Add artifacts and citations to responses
-- These are extracted from plan execution and surfaced in the response
-- ============================================================================
ALTER TABLE response_api.responses
    ADD COLUMN IF NOT EXISTS artifacts JSONB DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS citations JSONB DEFAULT '[]'::jsonb;

-- ============================================================================
-- INDEXES for new columns
-- ============================================================================

-- Index for finding steps that have different actual vs planned params
CREATE INDEX IF NOT EXISTS idx_plan_steps_params_mismatch 
    ON response_api.plan_steps ((actual_params IS NOT NULL AND actual_params != planned_params));

-- GIN index for searching within artifacts and citations
CREATE INDEX IF NOT EXISTS idx_responses_artifacts_gin 
    ON response_api.responses USING GIN (artifacts);

CREATE INDEX IF NOT EXISTS idx_responses_citations_gin 
    ON response_api.responses USING GIN (citations);

-- ============================================================================
-- COMMENTS for documentation
-- ============================================================================

COMMENT ON COLUMN response_api.plan_steps.planned_params IS 
    'Original parameters from the plan. May differ from actual_params if agent chose different tool/params.';

COMMENT ON COLUMN response_api.plan_steps.actual_params IS 
    'Actual parameters used during execution. Set when agent dynamically changes the tool or params.';

COMMENT ON COLUMN response_api.responses.artifacts IS 
    'Array of MediaArtifact objects uploaded to media-api. Contains jan_file_* IDs and download URLs.';

COMMENT ON COLUMN response_api.responses.citations IS 
    'Array of Citation objects extracted from search/research steps. Contains titles, URLs, and snippets.';

