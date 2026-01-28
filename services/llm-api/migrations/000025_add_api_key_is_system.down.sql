-- Remove is_system column from api_keys table

DROP INDEX IF EXISTS idx_api_keys_user_system_active;
DROP INDEX IF EXISTS idx_api_keys_is_system;
ALTER TABLE api_keys DROP COLUMN IF EXISTS is_system;
