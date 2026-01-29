-- Consolidated migration: Add display/ordering, reasoning modes, advanced features, and move capabilities to model_catalogs
-- Merges migrations 000006, 000007, and 000008 into a single migration

-- ============================================================================
-- SECTION 1: Display Name and Model Ordering (from 000006)
-- ============================================================================

-- Add model_display_name column (user-friendly display name, defaults to model_public_id)
ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS model_display_name VARCHAR(255) NOT NULL DEFAULT '';

-- Add category column (for grouping models, e.g., "Chat", "Embedding", "Image")
ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS category VARCHAR(128) NOT NULL DEFAULT '';

-- Add category_order_number column (for sorting categories)
ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS category_order_number INTEGER NOT NULL DEFAULT 0;

-- Add model_order_number column (for sorting models within a category)
ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS model_order_number INTEGER NOT NULL DEFAULT 0;

-- Add indexes for efficient sorting and filtering
CREATE INDEX IF NOT EXISTS idx_provider_models_category 
    ON llm_api.provider_models(category);

CREATE INDEX IF NOT EXISTS idx_provider_models_category_order 
    ON llm_api.provider_models(category_order_number);

CREATE INDEX IF NOT EXISTS idx_provider_models_model_order 
    ON llm_api.provider_models(model_order_number);

-- Composite index for efficient category-based sorting
CREATE INDEX IF NOT EXISTS idx_provider_models_category_sorting 
    ON llm_api.provider_models(category_order_number, model_order_number);

-- Add comments for documentation
COMMENT ON COLUMN llm_api.provider_models.model_display_name IS 'User-friendly display name for the model (defaults to model_public_id if empty)';
COMMENT ON COLUMN llm_api.provider_models.category IS 'Category for grouping models (e.g., Chat, Embedding, Image)';
COMMENT ON COLUMN llm_api.provider_models.category_order_number IS 'Order number for sorting categories (lower numbers appear first)';
COMMENT ON COLUMN llm_api.provider_models.model_order_number IS 'Order number for sorting models within a category (lower numbers appear first)';

-- Remove duplicate display_name column if it exists
-- The model_display_name column already serves this purpose
ALTER TABLE llm_api.provider_models DROP COLUMN IF EXISTS display_name;

-- ============================================================================
-- SECTION 2: Auto/Thinking Modes and Advanced Features (from 000007)
-- ============================================================================

-- Add supports_auto_mode (provider-controlled auto mode selection)
ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS supports_auto_mode BOOLEAN NOT NULL DEFAULT false;

-- Add supports_thinking_mode (long-form reasoning / thinking modes)
ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS supports_thinking_mode BOOLEAN NOT NULL DEFAULT false;

-- Add default_conversation_mode (standard, auto, thinking)
ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS default_conversation_mode VARCHAR(64) NOT NULL DEFAULT 'standard';

-- Add provider_flags (provider-specific quirks and overrides)
ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS provider_flags JSONB;

-- Add indexes for filtering by reasoning capabilities
CREATE INDEX IF NOT EXISTS idx_provider_models_auto_mode 
    ON llm_api.provider_models(supports_auto_mode) WHERE supports_auto_mode = true;

CREATE INDEX IF NOT EXISTS idx_provider_models_thinking_mode 
    ON llm_api.provider_models(supports_thinking_mode) WHERE supports_thinking_mode = true;

-- Add comments for documentation
COMMENT ON COLUMN llm_api.provider_models.supports_auto_mode IS 'Whether the model supports automatic mode selection (provider chooses best mode)';
COMMENT ON COLUMN llm_api.provider_models.supports_thinking_mode IS 'Whether the model supports extended thinking/reasoning mode';
COMMENT ON COLUMN llm_api.provider_models.default_conversation_mode IS 'Default conversation mode: standard, auto, or thinking';
COMMENT ON COLUMN llm_api.provider_models.provider_flags IS 'Provider-specific flags and configuration overrides (JSONB)';

-- ============================================================================
-- SECTION 3: Reasoning Configuration (from 000008)
-- ============================================================================

-- Add reasoning_config column (consolidated reasoning metadata)
ALTER TABLE llm_api.provider_models
    ADD COLUMN IF NOT EXISTS reasoning_config JSONB;

-- Add comment for documentation
COMMENT ON COLUMN llm_api.provider_models.reasoning_config IS 'Reasoning configuration (effort_levels, default_effort, max_tokens, price_multiplier, latency_hint_ms, mode_display)';

-- ============================================================================
-- SECTION 4: Move Capabilities to Model Catalogs (Phase 3 Optimization)
-- ============================================================================

-- Add capability columns to model_catalogs
ALTER TABLE llm_api.model_catalogs
    ADD COLUMN IF NOT EXISTS supports_images BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE llm_api.model_catalogs
    ADD COLUMN IF NOT EXISTS supports_embeddings BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE llm_api.model_catalogs
    ADD COLUMN IF NOT EXISTS supports_reasoning BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE llm_api.model_catalogs
    ADD COLUMN IF NOT EXISTS supports_audio BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE llm_api.model_catalogs
    ADD COLUMN IF NOT EXISTS supports_video BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE llm_api.model_catalogs
    ADD COLUMN IF NOT EXISTS family VARCHAR(128);

-- Add indexes for efficient filtering
CREATE INDEX IF NOT EXISTS idx_model_catalogs_supports_images 
    ON llm_api.model_catalogs(supports_images) WHERE supports_images = true;

CREATE INDEX IF NOT EXISTS idx_model_catalogs_supports_embeddings 
    ON llm_api.model_catalogs(supports_embeddings) WHERE supports_embeddings = true;

CREATE INDEX IF NOT EXISTS idx_model_catalogs_supports_reasoning 
    ON llm_api.model_catalogs(supports_reasoning) WHERE supports_reasoning = true;

CREATE INDEX IF NOT EXISTS idx_model_catalogs_supports_audio 
    ON llm_api.model_catalogs(supports_audio) WHERE supports_audio = true;

CREATE INDEX IF NOT EXISTS idx_model_catalogs_supports_video 
    ON llm_api.model_catalogs(supports_video) WHERE supports_video = true;

CREATE INDEX IF NOT EXISTS idx_model_catalogs_family 
    ON llm_api.model_catalogs(family);

-- Migrate capability data from provider_models to model_catalogs
-- For each unique model_catalog_id, aggregate capabilities from provider_models
UPDATE llm_api.model_catalogs mc
SET 
    supports_images = COALESCE((
        SELECT bool_or(pm.supports_images)
        FROM llm_api.provider_models pm
        WHERE pm.model_catalog_id = mc.id
    ), false),
    supports_embeddings = COALESCE((
        SELECT bool_or(pm.supports_embeddings)
        FROM llm_api.provider_models pm
        WHERE pm.model_catalog_id = mc.id
    ), false),
    supports_reasoning = COALESCE((
        SELECT bool_or(pm.supports_reasoning)
        FROM llm_api.provider_models pm
        WHERE pm.model_catalog_id = mc.id
    ), false),
    supports_audio = COALESCE((
        SELECT bool_or(pm.supports_audio)
        FROM llm_api.provider_models pm
        WHERE pm.model_catalog_id = mc.id
    ), false),
    supports_video = COALESCE((
        SELECT bool_or(pm.supports_video)
        FROM llm_api.provider_models pm
        WHERE pm.model_catalog_id = mc.id
    ), false),
    family = (
        SELECT pm.family
        FROM llm_api.provider_models pm
        WHERE pm.model_catalog_id = mc.id
        LIMIT 1
    );

-- Add comments for documentation
COMMENT ON COLUMN llm_api.model_catalogs.supports_images IS 'Model supports image input (vision)';
COMMENT ON COLUMN llm_api.model_catalogs.supports_embeddings IS 'Model generates embeddings';
COMMENT ON COLUMN llm_api.model_catalogs.supports_reasoning IS 'Model supports extended reasoning modes';
COMMENT ON COLUMN llm_api.model_catalogs.supports_audio IS 'Model supports audio input/output';
COMMENT ON COLUMN llm_api.model_catalogs.supports_video IS 'Model supports video input/output';
COMMENT ON COLUMN llm_api.model_catalogs.family IS 'Model family (e.g., gpt-4, claude-3, llama-3)';

-- Remove capability columns from provider_models (capabilities now in model_catalogs)
ALTER TABLE llm_api.provider_models DROP COLUMN IF EXISTS supports_images;
ALTER TABLE llm_api.provider_models DROP COLUMN IF EXISTS supports_embeddings;
ALTER TABLE llm_api.provider_models DROP COLUMN IF EXISTS supports_reasoning;
ALTER TABLE llm_api.provider_models DROP COLUMN IF EXISTS supports_audio;
ALTER TABLE llm_api.provider_models DROP COLUMN IF EXISTS supports_video;
ALTER TABLE llm_api.provider_models DROP COLUMN IF EXISTS family;

-- ============================================================================
-- SECTION 5: Remove legacy llm_api.models table (no longer used)
-- ============================================================================
DROP TRIGGER IF EXISTS models_updated_at ON llm_api.models;
DROP TABLE IF EXISTS llm_api.models;

