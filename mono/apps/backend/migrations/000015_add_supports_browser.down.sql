-- Rollback Migration 000015: Remove supports_browser column

-- Drop index first
DROP INDEX IF EXISTS llm_api.idx_model_catalogs_supports_browser;

-- Remove supports_browser column from model_catalogs table
ALTER TABLE llm_api.model_catalogs DROP COLUMN IF EXISTS supports_browser;
