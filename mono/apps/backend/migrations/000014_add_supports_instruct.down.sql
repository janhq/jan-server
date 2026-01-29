-- Rollback Migration 000014: Remove supports_instruct and instruct_model_id

-- Drop index first
DROP INDEX IF EXISTS llm_api.idx_provider_models_instruct_model_id;

-- Remove instruct_model_id column from provider_models table
ALTER TABLE llm_api.provider_models DROP COLUMN IF EXISTS instruct_model_id;

-- Drop index first
DROP INDEX IF EXISTS llm_api.idx_model_catalogs_supports_instruct;

-- Remove supports_instruct column from model_catalogs table
ALTER TABLE llm_api.model_catalogs DROP COLUMN IF EXISTS supports_instruct;
