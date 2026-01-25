-- Drop title/description from plan_steps
SET search_path TO response_api;

ALTER TABLE response_api.plan_steps
    DROP COLUMN description,
    DROP COLUMN title;
