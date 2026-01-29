-- Add index on api_keys.hash for fast API key lookups
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(hash);
