-- Model-Specific Prompt Templates
-- Migration 17: Create junction table for model-specific prompt template assignments
-- This allows each model_catalog entry to have its own prompt templates that override global defaults

SET search_path TO llm_api;

-- Remove unique constraint on template_key to allow duplicate templates for model-specific prompts
ALTER TABLE llm_api.prompt_templates DROP CONSTRAINT IF EXISTS prompt_templates_template_key_key;

-- Update comment to reflect the change
COMMENT ON COLUMN llm_api.prompt_templates.template_key IS 'Key used to reference this template in code (e.g., deep_research). Multiple templates can share the same key for model-specific variants.';

-- Junction table for model-specific prompt template assignments
CREATE TABLE IF NOT EXISTS llm_api.model_prompt_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_catalog_public_id VARCHAR(64) NOT NULL REFERENCES llm_api.model_catalogs(public_id) ON DELETE CASCADE,
    template_key VARCHAR(100) NOT NULL,  -- e.g., 'deep_research', 'timing'
    prompt_template_id UUID NOT NULL REFERENCES llm_api.prompt_templates(id) ON DELETE CASCADE,
    priority INTEGER DEFAULT 0,           -- Higher = more priority (for future multi-template support)
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    
    -- Unique constraint: one template per key per model
    CONSTRAINT uq_model_template_key UNIQUE (model_catalog_public_id, template_key)
);

-- Indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_model_prompt_templates_model ON llm_api.model_prompt_templates(model_catalog_public_id);
CREATE INDEX IF NOT EXISTS idx_model_prompt_templates_key ON llm_api.model_prompt_templates(template_key);
CREATE INDEX IF NOT EXISTS idx_model_prompt_templates_active ON llm_api.model_prompt_templates(is_active);
CREATE INDEX IF NOT EXISTS idx_model_prompt_templates_template ON llm_api.model_prompt_templates(prompt_template_id);

-- Comments for documentation
COMMENT ON TABLE llm_api.model_prompt_templates IS 'Assigns specific prompt templates to model catalogs, overriding global defaults';
COMMENT ON COLUMN llm_api.model_prompt_templates.template_key IS 'The template key being overridden (e.g., deep_research, timing)';
COMMENT ON COLUMN llm_api.model_prompt_templates.priority IS 'Higher priority templates are used first when multiple exist';
COMMENT ON COLUMN llm_api.model_prompt_templates.is_active IS 'Whether this template assignment is active';
