-- Add model column to plans table
ALTER TABLE plans ADD COLUMN IF NOT EXISTS model VARCHAR(128) DEFAULT '';
