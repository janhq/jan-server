-- Admin MCP Tools Configuration
-- Migration 16: Create admin_mcp_tools table for dynamic MCP tool management
SET search_path TO llm_api;

CREATE TABLE IF NOT EXISTS llm_api.admin_mcp_tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    public_id VARCHAR(50) UNIQUE NOT NULL,
    tool_key VARCHAR(100) UNIQUE NOT NULL,      -- e.g., 'google_search', 'scrape'
    name VARCHAR(255) NOT NULL,                  -- Display name (read-only in admin UI)
    description TEXT NOT NULL,                   -- Tool description for LLM
    category VARCHAR(100) DEFAULT 'search',      -- 'search', 'scrape', 'code_execution', etc.
    is_active BOOLEAN DEFAULT true,
    metadata JSONB,                              -- Additional config (future extensibility)
    
    -- Content filtering for search results (regex patterns)
    disallowed_keywords TEXT[],                  -- Regex patterns to filter from search results (e.g., '(?i)menlo' for case-insensitive)
    
    -- Audit fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_by UUID
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_admin_mcp_tools_key ON llm_api.admin_mcp_tools(tool_key);
CREATE INDEX IF NOT EXISTS idx_admin_mcp_tools_active ON llm_api.admin_mcp_tools(is_active);
CREATE INDEX IF NOT EXISTS idx_admin_mcp_tools_category ON llm_api.admin_mcp_tools(category);

-- Comments
COMMENT ON TABLE llm_api.admin_mcp_tools IS 'Admin-configurable MCP tool definitions for dynamic tool management';
COMMENT ON COLUMN llm_api.admin_mcp_tools.tool_key IS 'Unique identifier matching the registered MCP tool name';
COMMENT ON COLUMN llm_api.admin_mcp_tools.disallowed_keywords IS 'Regex patterns that trigger result filtering in search responses (e.g., (?i)menlo for case-insensitive match)';

-- Seed default tools (matching serper_mcp.go)
-- Note: disallowed_keywords uses regex patterns (case-insensitive matching)
INSERT INTO llm_api.admin_mcp_tools (public_id, tool_key, name, description, category, disallowed_keywords)
VALUES 
(
    'mcp_google_search_001',
    'google_search',
    'google_search',
    'Perform web searches via the configured engines (Serper, SearXNG, or cached fallback) and fetch structured citations.',
    'search',
    ARRAY['(?i)menlo']::TEXT[]  -- Example: matches "menlo" or "Menlo" (case-insensitive regex)
),
(
    'mcp_scrape_001',
    'scrape',
    'scrape',
    'Scrape a webpage and retrieve the text with optional markdown formatting.',
    'scrape',
    ARRAY[]::TEXT[]
),
(
    'mcp_file_search_index_001',
    'file_search_index',
    'file_search_index',
    'Index arbitrary text into the lightweight vector store used for MCP automations.',
    'file_search',
    ARRAY[]::TEXT[]
),
(
    'mcp_file_search_query_001',
    'file_search_query',
    'file_search_query',
    'Run a semantic query against documents indexed via file_search_index.',
    'file_search',
    ARRAY[]::TEXT[]
);
