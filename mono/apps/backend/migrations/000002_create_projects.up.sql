-- Create projects table
CREATE TABLE IF NOT EXISTS projects (
    id BIGSERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    instruction TEXT,
    favorite BOOLEAN NOT NULL DEFAULT false,
    archived_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_projects_user_name UNIQUE (user_id, name)
);

CREATE INDEX idx_projects_user_id ON projects(user_id);
CREATE INDEX idx_projects_user_updated_at ON projects(user_id, updated_at DESC);
CREATE INDEX idx_projects_deleted_at ON projects(deleted_at);
CREATE INDEX idx_projects_archived_at ON projects(archived_at);

-- Update conversations table
ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS project_id BIGINT REFERENCES projects(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS instruction_version INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS effective_instruction_snapshot TEXT;

CREATE INDEX IF NOT EXISTS idx_conversations_project_updated_at 
    ON conversations(project_id, updated_at DESC);

COMMENT ON TABLE projects IS 'Projects for grouping conversations and inheriting instructions';
COMMENT ON COLUMN conversations.project_id IS 'Optional project grouping';
COMMENT ON COLUMN conversations.instruction_version IS 'Version of project instruction when conversation was created';
COMMENT ON COLUMN conversations.effective_instruction_snapshot IS 'Snapshot of merged instruction for reproducibility';
