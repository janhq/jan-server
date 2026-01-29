-- Consolidated rollback for migration 000006
-- This reverses all changes: display/ordering fields, auto/thinking modes, reasoning config, and capability migration
-- Rollback order: capabilities -> reasoning_config -> auto/thinking modes -> display/ordering fields

-- ============================================================================
-- STEP 1: Restore capability columns to provider_models
-- ============================================================================

-- Add capability columns back to provider_models
ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS supports_images BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS supports_embeddings BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS supports_reasoning BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS supports_audio BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS supports_video BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS family VARCHAR(128);

-- Migrate capability data back from model_catalogs to provider_models
UPDATE llm_api.provider_models pm
SET 
    supports_images = COALESCE(mc.supports_images, false),
    supports_embeddings = COALESCE(mc.supports_embeddings, false),
    supports_reasoning = COALESCE(mc.supports_reasoning, false),
    supports_audio = COALESCE(mc.supports_audio, false),
    supports_video = COALESCE(mc.supports_video, false),
    family = mc.family
FROM llm_api.model_catalogs mc
WHERE pm.model_catalog_id = mc.id;

-- Drop capability columns from model_catalogs
DROP INDEX IF EXISTS llm_api.idx_model_catalogs_family;
DROP INDEX IF EXISTS llm_api.idx_model_catalogs_supports_video;
DROP INDEX IF EXISTS llm_api.idx_model_catalogs_supports_audio;
DROP INDEX IF EXISTS llm_api.idx_model_catalogs_supports_reasoning;
DROP INDEX IF EXISTS llm_api.idx_model_catalogs_supports_embeddings;
DROP INDEX IF EXISTS llm_api.idx_model_catalogs_supports_images;

ALTER TABLE llm_api.model_catalogs DROP COLUMN IF EXISTS family;
ALTER TABLE llm_api.model_catalogs DROP COLUMN IF EXISTS supports_video;
ALTER TABLE llm_api.model_catalogs DROP COLUMN IF EXISTS supports_audio;
ALTER TABLE llm_api.model_catalogs DROP COLUMN IF EXISTS supports_reasoning;
ALTER TABLE llm_api.model_catalogs DROP COLUMN IF EXISTS supports_embeddings;
ALTER TABLE llm_api.model_catalogs DROP COLUMN IF EXISTS supports_images;

-- ============================================================================
-- STEP 2: Rollback reasoning_config (split back into individual columns for compatibility)
-- ============================================================================

-- Recreate the individual reasoning columns
ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS reasoning_effort_levels JSONB;

ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS reasoning_default_effort VARCHAR(64) NOT NULL DEFAULT '';

ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS reasoning_max_tokens INTEGER;

ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS reasoning_price_multiplier DOUBLE PRECISION;

ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS reasoning_latency_hint_ms INTEGER;

ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS reasoning_mode_display JSONB;

-- Migrate data back from reasoning_config to individual columns
UPDATE llm_api.provider_models
SET 
    reasoning_effort_levels = reasoning_config->'effort_levels',
    reasoning_default_effort = COALESCE(reasoning_config->>'default_effort', ''),
    reasoning_max_tokens = (reasoning_config->>'max_tokens')::INTEGER,
    reasoning_price_multiplier = (reasoning_config->>'price_multiplier')::DOUBLE PRECISION,
    reasoning_latency_hint_ms = (reasoning_config->>'latency_hint_ms')::INTEGER,
    reasoning_mode_display = reasoning_config->'mode_display'
WHERE reasoning_config IS NOT NULL;

-- Drop the reasoning_config column
ALTER TABLE llm_api.provider_models DROP COLUMN IF EXISTS reasoning_config;

-- Restore comments for reasoning columns
COMMENT ON COLUMN llm_api.provider_models.reasoning_effort_levels IS 'Available reasoning effort levels (e.g., ["low","medium","high"])';
COMMENT ON COLUMN llm_api.provider_models.reasoning_default_effort IS 'Default reasoning effort level';
COMMENT ON COLUMN llm_api.provider_models.reasoning_max_tokens IS 'Maximum tokens for reasoning/thinking output';
COMMENT ON COLUMN llm_api.provider_models.reasoning_price_multiplier IS 'Price multiplier for reasoning mode vs standard mode';
COMMENT ON COLUMN llm_api.provider_models.reasoning_latency_hint_ms IS 'Estimated additional latency in milliseconds for reasoning mode';
COMMENT ON COLUMN llm_api.provider_models.reasoning_mode_display IS 'UI display options for reasoning modes with hints';

-- ============================================================================
-- STEP 3: Rollback auto/thinking mode columns
-- ============================================================================

-- Drop indexes from auto/thinking modes
DROP INDEX IF EXISTS llm_api.idx_provider_models_thinking_mode;
DROP INDEX IF EXISTS llm_api.idx_provider_models_auto_mode;

-- Drop all columns added in 000007 (in reverse order)
ALTER TABLE llm_api.provider_models
    DROP COLUMN IF EXISTS provider_flags;

ALTER TABLE llm_api.provider_models
    DROP COLUMN IF EXISTS reasoning_mode_display;

ALTER TABLE llm_api.provider_models
    DROP COLUMN IF EXISTS reasoning_latency_hint_ms;

ALTER TABLE llm_api.provider_models
    DROP COLUMN IF EXISTS reasoning_price_multiplier;

ALTER TABLE llm_api.provider_models
    DROP COLUMN IF EXISTS reasoning_max_tokens;

ALTER TABLE llm_api.provider_models
    DROP COLUMN IF EXISTS reasoning_default_effort;

ALTER TABLE llm_api.provider_models
    DROP COLUMN IF EXISTS reasoning_effort_levels;

ALTER TABLE llm_api.provider_models
    DROP COLUMN IF EXISTS default_conversation_mode;

ALTER TABLE llm_api.provider_models
    DROP COLUMN IF EXISTS supports_thinking_mode;

ALTER TABLE llm_api.provider_models
    DROP COLUMN IF EXISTS supports_auto_mode;

-- ============================================================================
-- STEP 4: Rollback display/ordering columns and restore display_name
-- ============================================================================

-- Drop indexes from display/ordering fields
DROP INDEX IF EXISTS llm_api.idx_provider_models_category_sorting;
DROP INDEX IF EXISTS llm_api.idx_provider_models_model_order;
DROP INDEX IF EXISTS llm_api.idx_provider_models_category_order;
DROP INDEX IF EXISTS llm_api.idx_provider_models_category;

-- Restore legacy display_name column before dropping model_display_name
-- Copy data from model_display_name to display_name for data preservation
ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS display_name VARCHAR(255) NOT NULL DEFAULT '';

-- Copy model_display_name to display_name if model_display_name exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'llm_api'
          AND table_name = 'provider_models'
          AND column_name = 'model_display_name'
    ) THEN
        UPDATE llm_api.provider_models
        SET display_name = COALESCE(NULLIF(model_display_name, ''), model_public_id)
        WHERE display_name = '' OR display_name IS NULL;
    END IF;
END $$;

COMMENT ON COLUMN llm_api.provider_models.display_name IS 'Legacy display name column (restored for rollback compatibility)';

-- Drop the new columns added in 000006
ALTER TABLE llm_api.provider_models
    DROP COLUMN IF EXISTS model_order_number;

ALTER TABLE llm_api.provider_models
    DROP COLUMN IF EXISTS category_order_number;

ALTER TABLE llm_api.provider_models
    DROP COLUMN IF EXISTS category;

ALTER TABLE llm_api.provider_models
    DROP COLUMN IF EXISTS model_display_name;

-- ============================================================================
-- STEP 5: Restore legacy llm_api.models table and trigger (removed in 000006 up)
-- ============================================================================
CREATE TABLE IF NOT EXISTS llm_api.models (
    id VARCHAR(255) PRIMARY KEY,
    provider VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    family VARCHAR(255),
    capabilities JSONB,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER IF NOT EXISTS models_updated_at
    BEFORE UPDATE ON llm_api.models
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

