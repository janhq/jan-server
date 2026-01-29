-- Revert the unique constraint change

-- Drop the partial unique index
DROP INDEX IF EXISTS uq_projects_user_name_active;

-- Recreate the original constraint
ALTER TABLE projects ADD CONSTRAINT uq_projects_user_name UNIQUE (user_id, name);
