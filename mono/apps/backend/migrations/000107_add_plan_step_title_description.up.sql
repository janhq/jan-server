-- Add title/description to plan_steps
SET search_path TO response_api;

ALTER TABLE response_api.plan_steps
    ADD COLUMN title VARCHAR(256) NOT NULL DEFAULT '',
    ADD COLUMN description TEXT;
