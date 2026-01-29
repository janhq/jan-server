-- Drop triggers first
DROP TRIGGER IF EXISTS api_keys_updated_at ON llm_api.api_keys;
DROP TRIGGER IF EXISTS conversation_items_updated_at ON llm_api.conversation_items;
DROP TRIGGER IF EXISTS conversation_branches_updated_at ON llm_api.conversation_branches;
DROP TRIGGER IF EXISTS conversations_updated_at ON llm_api.conversations;
DROP TRIGGER IF EXISTS models_updated_at ON llm_api.models;
DROP TRIGGER IF EXISTS provider_models_updated_at ON llm_api.provider_models;
DROP TRIGGER IF EXISTS model_catalogs_updated_at ON llm_api.model_catalogs;
DROP TRIGGER IF EXISTS providers_updated_at ON llm_api.providers;
DROP TRIGGER IF EXISTS users_updated_at ON llm_api.users;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS llm_api.api_keys;
DROP TABLE IF EXISTS llm_api.conversation_items;
DROP TABLE IF EXISTS llm_api.conversation_branches;
DROP TABLE IF EXISTS llm_api.conversations;
DROP TABLE IF EXISTS llm_api.models;
DROP TABLE IF EXISTS llm_api.provider_models;
DROP TABLE IF EXISTS llm_api.model_catalogs;
DROP TABLE IF EXISTS llm_api.providers;
DROP TABLE IF EXISTS llm_api.users;

-- Drop schema
DROP SCHEMA IF EXISTS llm_api CASCADE;

