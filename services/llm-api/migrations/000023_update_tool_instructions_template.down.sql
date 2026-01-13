-- +goose Down
-- Revert tool_instructions template to previous version
UPDATE llm_api.prompt_templates
SET content = 'You have access to various tools. Always choose the best tool for the task.
When you need to search for information, use web search. When you need to execute code, use the code execution tool.
Tool usage must respect project instructions and system-level constraints at all times.',
    updated_at = NOW(),
    version = version - 1
WHERE template_key = 'tool_instructions'
  AND public_id = 'pt_tool_instructions_001';
