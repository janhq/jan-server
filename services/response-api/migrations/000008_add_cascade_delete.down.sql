-- Revert responses.conversation_id to NO ACTION (default)

ALTER TABLE response_api.responses
DROP CONSTRAINT IF EXISTS responses_conversation_id_fkey;

ALTER TABLE response_api.responses
ADD CONSTRAINT responses_conversation_id_fkey
FOREIGN KEY (conversation_id) REFERENCES response_api.conversations(id);

-- Revert artifacts.response_id to NO ACTION (default)

ALTER TABLE response_api.artifacts
DROP CONSTRAINT IF EXISTS artifacts_response_id_fkey;

ALTER TABLE response_api.artifacts
ADD CONSTRAINT artifacts_response_id_fkey
FOREIGN KEY (response_id) REFERENCES response_api.responses(id);

-- Revert plans.response_id to NO ACTION (default)

ALTER TABLE response_api.plans
DROP CONSTRAINT IF EXISTS plans_response_id_fkey;

ALTER TABLE response_api.plans
ADD CONSTRAINT plans_response_id_fkey
FOREIGN KEY (response_id) REFERENCES response_api.responses(id);
