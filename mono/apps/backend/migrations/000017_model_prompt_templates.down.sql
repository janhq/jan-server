-- Rollback Model-Specific Prompt Templates
-- Migration 17 Down: Drop model_prompt_templates table and restore unique constraint

SET search_path TO llm_api;

DROP TABLE IF EXISTS llm_api.model_prompt_templates;

-- Re-add the unique constraint on template_key
ALTER TABLE llm_api.prompt_templates ADD CONSTRAINT prompt_templates_template_key_key UNIQUE (template_key);

-- Restore original comment
COMMENT ON COLUMN llm_api.prompt_templates.template_key IS 'Unique key used to reference this template in code (e.g., deep_research)';
