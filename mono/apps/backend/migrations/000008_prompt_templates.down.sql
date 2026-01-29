-- Rollback: Drop prompt_templates table
SET search_path TO llm_api;

DROP TABLE IF EXISTS llm_api.prompt_templates;
