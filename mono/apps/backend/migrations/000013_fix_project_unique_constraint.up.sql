-- Fix unique constraint on projects to exclude soft-deleted records
-- This allows creating a project with the same name after the original is deleted

-- Drop the existing constraint
ALTER TABLE projects DROP CONSTRAINT IF EXISTS uq_projects_user_name;

-- Create a partial unique index that only applies to non-deleted records
CREATE UNIQUE INDEX uq_projects_user_name_active 
    ON projects (user_id, name) 
    WHERE deleted_at IS NULL;

COMMENT ON INDEX uq_projects_user_name_active IS 'Ensures unique project names per user, excluding soft-deleted projects';
