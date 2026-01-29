-- Drop indexes first.
DROP INDEX IF EXISTS llm_api.idx_providers_active_category;
DROP INDEX IF EXISTS llm_api.idx_providers_category;

-- Remove category column.
ALTER TABLE llm_api.providers DROP COLUMN IF EXISTS category;
