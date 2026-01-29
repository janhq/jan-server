-- Drop tables in reverse order (respecting foreign key constraints)
DROP TABLE IF EXISTS response_api.tool_executions CASCADE;
DROP TABLE IF EXISTS response_api.conversation_items CASCADE;
DROP TABLE IF EXISTS response_api.responses CASCADE;
DROP TABLE IF EXISTS response_api.conversations CASCADE;

-- Drop schema
DROP SCHEMA IF EXISTS response_api CASCADE;
