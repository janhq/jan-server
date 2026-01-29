-- Drop helper index first.
DROP INDEX IF EXISTS llm_api.idx_providers_multi_endpoint;

-- Remove endpoints column, legacy base_url remains.
ALTER TABLE llm_api.providers DROP COLUMN IF EXISTS endpoints;
