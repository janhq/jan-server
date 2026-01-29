-- Add endpoints column to store multiple URLs as JSON array.
-- NULL/empty falls back to base_url for backward compatibility.
ALTER TABLE llm_api.providers
ADD COLUMN IF NOT EXISTS endpoints JSONB DEFAULT NULL;

COMMENT ON COLUMN llm_api.providers.endpoints IS
'JSON array of endpoint objects: [{"url":"...", "weight":1, "healthy":true, "priority":0}]. Falls back to base_url if NULL or empty.';

-- Backfill existing base_url into endpoints for consistency.
UPDATE llm_api.providers
SET endpoints = jsonb_build_array(
        jsonb_build_object(
            'url', base_url,
            'weight', 1,
            'healthy', true,
            'priority', 0
        )
    )
WHERE base_url IS NOT NULL
  AND base_url != ''
  AND endpoints IS NULL;

-- Optional index to quickly find providers with multiple endpoints.
CREATE INDEX IF NOT EXISTS idx_providers_multi_endpoint
    ON llm_api.providers ((jsonb_array_length(endpoints) > 1))
    WHERE endpoints IS NOT NULL;
