-- Drop connector tables
DROP TABLE IF EXISTS llm_api.connector_audit_log;
DROP TABLE IF EXISTS llm_api.connector_oauth_states;
DROP TABLE IF EXISTS llm_api.connector_connections;

-- Drop enum type
DROP TYPE IF EXISTS llm_api.connector_type;
