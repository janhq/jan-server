-- Rollback Admin MCP Tools Configuration
SET search_path TO llm_api;

DROP TABLE IF EXISTS llm_api.admin_mcp_tools;
