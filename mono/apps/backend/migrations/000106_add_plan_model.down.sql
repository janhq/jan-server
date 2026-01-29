-- Remove model column from plans table
ALTER TABLE plans DROP COLUMN IF EXISTS model;
