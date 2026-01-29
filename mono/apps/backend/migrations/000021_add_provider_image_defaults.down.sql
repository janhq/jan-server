ALTER TABLE llm_api.providers
    DROP COLUMN IF EXISTS default_provider_image_edit,
    DROP COLUMN IF EXISTS default_provider_image_generate;
