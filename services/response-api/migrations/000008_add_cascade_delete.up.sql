-- Add ON DELETE CASCADE to responses.conversation_id
-- When a conversation is deleted, all its responses are deleted

ALTER TABLE response_api.responses
DROP CONSTRAINT IF EXISTS responses_conversation_id_fkey;

ALTER TABLE response_api.responses
ADD CONSTRAINT responses_conversation_id_fkey
FOREIGN KEY (conversation_id) REFERENCES response_api.conversations(id) ON DELETE CASCADE;

-- Add ON DELETE CASCADE to artifacts.response_id
-- When a response is deleted, all its artifacts are deleted

ALTER TABLE response_api.artifacts
DROP CONSTRAINT IF EXISTS artifacts_response_id_fkey;

ALTER TABLE response_api.artifacts
ADD CONSTRAINT artifacts_response_id_fkey
FOREIGN KEY (response_id) REFERENCES response_api.responses(id) ON DELETE CASCADE;

-- Add ON DELETE CASCADE to plans.response_id
-- When a response is deleted, all its plans are deleted

ALTER TABLE response_api.plans
DROP CONSTRAINT IF EXISTS plans_response_id_fkey;

ALTER TABLE response_api.plans
ADD CONSTRAINT plans_response_id_fkey
FOREIGN KEY (response_id) REFERENCES response_api.responses(id) ON DELETE CASCADE;
