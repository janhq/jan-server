-- Rollback merged migrations: 7
SET search_path TO llm_api;

-- Migration 10 rollback: Remove requires_feature_flag column from model_catalogs
DROP INDEX IF EXISTS llm_api.idx_model_catalogs_requires_feature_flag;
ALTER TABLE llm_api.model_catalogs DROP COLUMN IF EXISTS requires_feature_flag;

-- Migration 9 rollback: Remove description column
ALTER TABLE llm_api.model_catalogs DROP COLUMN IF EXISTS description;

-- Migration 8 rollback: Remove display name and context length
ALTER TABLE llm_api.model_catalogs DROP COLUMN IF EXISTS model_display_name;
ALTER TABLE llm_api.model_catalogs DROP COLUMN IF EXISTS context_length;

-- Migration 7 rollback: Remove experimental, audit logs, and feature flags
DROP INDEX IF EXISTS llm_api.idx_model_catalogs_experimental;
ALTER TABLE llm_api.model_catalogs DROP COLUMN IF EXISTS experimental;

DROP INDEX IF EXISTS llm_api.idx_audit_logs_resource_type;
DROP INDEX IF EXISTS llm_api.idx_audit_logs_admin_user_id;
DROP INDEX IF EXISTS llm_api.idx_audit_logs_created_at;
DROP TABLE IF EXISTS llm_api.audit_logs;

DROP INDEX IF EXISTS llm_api.idx_feature_flags_category;
DROP INDEX IF EXISTS llm_api.idx_feature_flags_key;
DROP TABLE IF EXISTS llm_api.feature_flags;
