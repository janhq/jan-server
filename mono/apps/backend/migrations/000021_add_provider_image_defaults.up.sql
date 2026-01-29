-- Add default provider flags for image generation and image edits.
ALTER TABLE llm_api.providers
    ADD COLUMN IF NOT EXISTS default_provider_image_generate BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS default_provider_image_edit BOOLEAN NOT NULL DEFAULT false;
