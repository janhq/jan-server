-- Add category column to distinguish LLM providers from image generation providers.
-- Default to 'llm' for backward compatibility with existing providers.
ALTER TABLE llm_api.providers
ADD COLUMN IF NOT EXISTS category VARCHAR(20) NOT NULL DEFAULT 'llm';

COMMENT ON COLUMN llm_api.providers.category IS
'Provider category: "llm" for language models, "image" for image generation. Defaults to "llm".';

-- Index for efficient filtering by category.
CREATE INDEX IF NOT EXISTS idx_providers_category
    ON llm_api.providers (category);

-- Composite index for common query pattern: active providers by category.
CREATE INDEX IF NOT EXISTS idx_providers_active_category
    ON llm_api.providers (active, category)
    WHERE active = true;
