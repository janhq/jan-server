-- Add is_system column to api_keys table
-- System keys are created by internal services (e.g., Response API) and should not be shown in user's key list

ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT FALSE;

-- Create index for efficient filtering of system keys
CREATE INDEX IF NOT EXISTS idx_api_keys_is_system ON api_keys (is_system);

-- Create composite index for efficient lookup of active system keys per user
CREATE INDEX IF NOT EXISTS idx_api_keys_user_system_active ON api_keys (user_id, is_system, expires_at)
WHERE revoked_at IS NULL;
