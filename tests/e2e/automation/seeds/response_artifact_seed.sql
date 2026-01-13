-- Seed data for response-artifact automation tests.
-- This script is idempotent and safe to re-run.

BEGIN;
CREATE SCHEMA IF NOT EXISTS response_api;
SET search_path TO response_api;

-- Ensure required tables exist (in case migrations were not applied)
CREATE TABLE IF NOT EXISTS response_api.plans (
    id SERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    response_id INTEGER NOT NULL REFERENCES response_api.responses(id),
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    progress DOUBLE PRECISION NOT NULL DEFAULT 0,
    agent_type VARCHAR(32),
    planning_config JSONB,
    estimated_steps INTEGER NOT NULL DEFAULT 0,
    completed_steps INTEGER NOT NULL DEFAULT 0,
    current_task_id INTEGER,
    final_artifact_id INTEGER,
    user_selection JSONB,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS response_api.plan_tasks (
    id SERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    plan_id INTEGER NOT NULL REFERENCES response_api.plans(id) ON DELETE CASCADE,
    sequence INTEGER NOT NULL DEFAULT 0,
    task_type VARCHAR(32),
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    title VARCHAR(256) NOT NULL,
    description TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS response_api.plan_steps (
    id SERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    task_id INTEGER NOT NULL REFERENCES response_api.plan_tasks(id) ON DELETE CASCADE,
    sequence INTEGER NOT NULL DEFAULT 0,
    action VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    input_params JSONB,
    output_data JSONB,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    error_message TEXT,
    error_severity VARCHAR(32),
    duration_ms BIGINT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS response_api.artifacts (
    id SERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    response_id INTEGER NOT NULL REFERENCES response_api.responses(id),
    plan_id INTEGER REFERENCES response_api.plans(id),
    content_type VARCHAR(32) NOT NULL,
    mime_type VARCHAR(128) NOT NULL,
    title VARCHAR(512) NOT NULL,
    content TEXT,
    storage_path VARCHAR(1024),
    size_bytes BIGINT NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    parent_id INTEGER REFERENCES response_api.artifacts(id),
    is_latest BOOLEAN NOT NULL DEFAULT true,
    retention_policy VARCHAR(32) NOT NULL DEFAULT 'session',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS response_api.plan_step_details (
    id SERIAL PRIMARY KEY,
    public_id VARCHAR(64) NOT NULL UNIQUE,
    step_id INTEGER NOT NULL REFERENCES response_api.plan_steps(id) ON DELETE CASCADE,
    detail_type VARCHAR(32) NOT NULL,
    conversation_item_id INTEGER REFERENCES response_api.conversation_items(id),
    tool_call_id VARCHAR(64),
    tool_execution_id INTEGER REFERENCES response_api.tool_executions(id),
    artifact_id INTEGER REFERENCES response_api.artifacts(id),
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cleanup previous seed data
DELETE FROM plan_step_details
WHERE step_id IN (
  SELECT id FROM plan_steps WHERE public_id IN ('step_artifact_test')
);

DELETE FROM plan_steps WHERE public_id IN ('step_artifact_test');
DELETE FROM plan_tasks WHERE public_id IN ('task_artifact_test');
DELETE FROM artifacts WHERE public_id IN ('art_artifact_test', 'art_artifact_test_v2');
DELETE FROM plans WHERE public_id IN ('plan_artifact_test');
DELETE FROM responses WHERE public_id IN ('resp_artifact_test');

-- Seed response
INSERT INTO responses (
  public_id,
  user_id,
  model,
  input,
  output,
  status,
  stream,
  background,
  store,
  object,
  created_at,
  updated_at
) VALUES (
  'resp_artifact_test',
  'seed-user',
  'seed-model',
  '{"type":"seed","text":"artifact seed"}'::jsonb,
  '{"type":"seed","text":"response output"}'::jsonb,
  'completed',
  false,
  false,
  true,
  'response',
  NOW(),
  NOW()
);

-- Seed plan
INSERT INTO plans (
  public_id,
  response_id,
  status,
  progress,
  agent_type,
  planning_config,
  estimated_steps,
  completed_steps,
  created_at,
  updated_at
)
SELECT
  'plan_artifact_test',
  r.id,
  'completed',
  100,
  'slide_generator',
  '{}'::jsonb,
  1,
  1,
  NOW(),
  NOW()
FROM responses r
WHERE r.public_id = 'resp_artifact_test';

-- Seed task
INSERT INTO plan_tasks (
  public_id,
  plan_id,
  sequence,
  task_type,
  status,
  title,
  description,
  created_at,
  updated_at,
  completed_at
)
SELECT
  'task_artifact_test',
  p.id,
  1,
  'generation',
  'completed',
  'Generate Slides',
  'Seeded task for artifact tests',
  NOW(),
  NOW(),
  NOW()
FROM plans p
WHERE p.public_id = 'plan_artifact_test';

-- Seed step
INSERT INTO plan_steps (
  public_id,
  task_id,
  sequence,
  action,
  status,
  input_params,
  output_data,
  retry_count,
  max_retries,
  duration_ms,
  started_at,
  completed_at
)
SELECT
  'step_artifact_test',
  t.id,
  1,
  'tool_call',
  'completed',
  '{"prompt":"seed generate"}'::jsonb,
  '{"result":"ok"}'::jsonb,
  0,
  3,
  120,
  NOW(),
  NOW()
FROM plan_tasks t
WHERE t.public_id = 'task_artifact_test';

-- Seed artifacts (version 1 and 2)
INSERT INTO artifacts (
  public_id,
  response_id,
  plan_id,
  content_type,
  mime_type,
  title,
  content,
  size_bytes,
  version,
  parent_id,
  is_latest,
  retention_policy,
  metadata,
  created_at,
  updated_at
)
SELECT
  'art_artifact_test',
  r.id,
  p.id,
  'slides',
  'application/json',
  'Seeded Slides v1',
  '{"slides":[{"title":"Seed Slide v1"}]}'::text,
  120,
  1,
  NULL,
  false,
  'session',
  '{}'::jsonb,
  NOW(),
  NOW()
FROM responses r
JOIN plans p ON p.public_id = 'plan_artifact_test'
WHERE r.public_id = 'resp_artifact_test';

INSERT INTO artifacts (
  public_id,
  response_id,
  plan_id,
  content_type,
  mime_type,
  title,
  content,
  size_bytes,
  version,
  parent_id,
  is_latest,
  retention_policy,
  metadata,
  created_at,
  updated_at
)
SELECT
  'art_artifact_test_v2',
  r.id,
  p.id,
  'slides',
  'application/json',
  'Seeded Slides v2',
  '{"slides":[{"title":"Seed Slide v2"}]}'::text,
  140,
  2,
  a.id,
  true,
  'session',
  '{}'::jsonb,
  NOW(),
  NOW()
FROM responses r
JOIN plans p ON p.public_id = 'plan_artifact_test'
JOIN artifacts a ON a.public_id = 'art_artifact_test'
WHERE r.public_id = 'resp_artifact_test';

-- Link plan to latest artifact and current task
UPDATE plans
SET
  current_task_id = (SELECT id FROM plan_tasks WHERE public_id = 'task_artifact_test'),
  final_artifact_id = (SELECT id FROM artifacts WHERE public_id = 'art_artifact_test_v2')
WHERE public_id = 'plan_artifact_test';

-- Step detail linking artifact
INSERT INTO plan_step_details (
  public_id,
  step_id,
  detail_type,
  artifact_id,
  metadata,
  created_at
)
SELECT
  'step_detail_artifact_test',
  s.id,
  'artifact',
  a.id,
  '{"note":"seeded artifact detail"}'::jsonb,
  NOW()
FROM plan_steps s
JOIN artifacts a ON a.public_id = 'art_artifact_test_v2'
WHERE s.public_id = 'step_artifact_test';

COMMIT;
