-- Seed edit_image MCP tool into admin_mcp_tools
SET search_path TO llm_api;

INSERT INTO llm_api.admin_mcp_tools (
    public_id,
    tool_key,
    name,
    description,
    category,
    is_active,
    disallowed_keywords
)
VALUES (
    'mcp_edit_image_001',
    'edit_image',
    'edit_image',
    'Edit images with a prompt and input image (params: prompt, image, mask, size, strength, steps, seed, cfg_scale). Always keep the image url from tool_result, do not change or reformat as it is presigned url',
    'image_edit',
    true,
    ARRAY[]::TEXT[]
)
ON CONFLICT (tool_key) DO NOTHING;
