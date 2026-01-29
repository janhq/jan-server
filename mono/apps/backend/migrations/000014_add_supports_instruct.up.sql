-- Migration 000014: Add supports_instruct and instruct_model_id for instruct backup feature
-- supports_instruct on model_catalogs: indicates the model can use an instruct backup
-- instruct_model_id on provider_models: references the provider_model to use when enable_thinking=false

-- Add supports_instruct column to model_catalogs table
ALTER TABLE llm_api.model_catalogs 
    ADD COLUMN IF NOT EXISTS supports_instruct BOOLEAN NOT NULL DEFAULT false;

-- Create index for filtering models by instruct support
CREATE INDEX IF NOT EXISTS idx_model_catalogs_supports_instruct 
    ON llm_api.model_catalogs(supports_instruct) WHERE supports_instruct = true;

-- Add comment for documentation
COMMENT ON COLUMN llm_api.model_catalogs.supports_instruct IS 'Model can use an instruct backup (shows backup dropdown in admin)';

-- Add instruct_model_id column to provider_models table for instruct model fallback
ALTER TABLE llm_api.provider_models 
    ADD COLUMN IF NOT EXISTS instruct_model_id BIGINT REFERENCES llm_api.provider_models(id) ON DELETE SET NULL;

-- Create index for the foreign key
CREATE INDEX IF NOT EXISTS idx_provider_models_instruct_model_id 
    ON llm_api.provider_models(instruct_model_id) WHERE instruct_model_id IS NOT NULL;

-- Add comment for documentation
COMMENT ON COLUMN llm_api.provider_models.instruct_model_id IS 'Reference to provider_model to use when enable_thinking=false';
