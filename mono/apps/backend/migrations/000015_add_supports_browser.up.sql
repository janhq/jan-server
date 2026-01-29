-- Migration 000015: Add supports_browser column to model_catalogs
-- supports_browser indicates the model supports browser/web browsing functionality

-- Add supports_browser column to model_catalogs table
ALTER TABLE llm_api.model_catalogs 
    ADD COLUMN IF NOT EXISTS supports_browser BOOLEAN NOT NULL DEFAULT false;

-- Create index for filtering models by browser support
CREATE INDEX IF NOT EXISTS idx_model_catalogs_supports_browser 
    ON llm_api.model_catalogs(supports_browser) WHERE supports_browser = true;

-- Add comment for documentation
COMMENT ON COLUMN llm_api.model_catalogs.supports_browser IS 'Model supports browser/web browsing functionality';
