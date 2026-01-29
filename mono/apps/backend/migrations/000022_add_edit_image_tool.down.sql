SET search_path TO llm_api;

DELETE FROM llm_api.admin_mcp_tools
WHERE tool_key = 'edit_image';
