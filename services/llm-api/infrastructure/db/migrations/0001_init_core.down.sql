DROP TRIGGER IF EXISTS models_updated_at ON models;
DROP TRIGGER IF EXISTS conversations_updated_at ON conversations;
DROP FUNCTION IF EXISTS update_updated_at_column;
DROP TABLE IF EXISTS models;
DROP TABLE IF EXISTS idempotency;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
