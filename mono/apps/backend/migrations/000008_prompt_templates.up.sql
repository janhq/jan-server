-- Prompt Templates for Orchestration Features
-- Migration 8: Create prompt_templates table for storing reusable prompt templates
SET search_path TO llm_api;

-- Prompt templates table
CREATE TABLE IF NOT EXISTS llm_api.prompt_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    public_id VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100) NOT NULL, -- e.g., 'orchestration', 'system', 'tool'
    template_key VARCHAR(100) UNIQUE NOT NULL, -- e.g., 'deep_research', 'code_review'
    content TEXT NOT NULL, -- The actual prompt template
    variables JSONB, -- List of variables in template, e.g., ["user_query", "context"]
    metadata JSONB, -- Additional configuration
    is_active BOOLEAN DEFAULT true,
    is_system BOOLEAN DEFAULT false, -- System prompts cannot be deleted
    version INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_by UUID
);

-- Indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_prompt_templates_key ON llm_api.prompt_templates(template_key);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_category ON llm_api.prompt_templates(category);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_active ON llm_api.prompt_templates(is_active);
CREATE INDEX IF NOT EXISTS idx_prompt_templates_system ON llm_api.prompt_templates(is_system);

-- Comments for documentation
COMMENT ON TABLE llm_api.prompt_templates IS 'Stores reusable prompt templates for orchestration features like Deep Research';
COMMENT ON COLUMN llm_api.prompt_templates.template_key IS 'Unique key used to reference this template in code (e.g., deep_research)';
COMMENT ON COLUMN llm_api.prompt_templates.is_system IS 'System prompts cannot be deleted, only edited';
COMMENT ON COLUMN llm_api.prompt_templates.variables IS 'JSON array of variable names that can be substituted in the template';

-- Seed Deep Research prompt template
INSERT INTO llm_api.prompt_templates (
    public_id,
    name,
    description,
    category,
    template_key,
    content,
    variables,
    metadata,
    is_active,
    is_system,
    version
) VALUES (
    'pt_deep_research_001',
    'Deep Research Agent',
    'Senior Research Agent prompt for conducting in-depth investigations with tool usage. Uses a 2-step workflow: clarification questions followed by comprehensive research and reporting.',
    'orchestration',
    'deep_research',
    E'You are a **Senior Research Agent** designed to conduct **in-depth, methodical investigations** into user questions. Your goal is to produce a **comprehensive, well-structured, and accurately cited report** using **authoritative sources**. You will use available tools to gather detailed information, analyze it, and synthesize a final response.

### **Tool Use Rules (Strictly Enforced)**

1.  **Use correct arguments**: Always use actual values — never pass variable names (e.g., use "Paris" not {city}).
2.  **Call tools only when necessary**: If you can answer from prior results, do so. However, all cited **URLs in the report must be visited**, and all **entities (People, Organization, Software, etc.) mentioned must be verified**.
3.  **Terminate When Full Coverage Is Achieved**: Conclude tool usage only when the investigation has achieved **comprehensive coverage**.
4.  **Visit all URLs**: You must strictly verify sources.
5.  **Avoid repetition**: Do not repeat the same searches.
6.  **Track progress**: Continuously assess if you have enough data.
7.  **Limit tool usage**: If stuck, reassess strategy.
8.  **Target Official Sources for "Updates"**: When the user asks for "updates," "versions," or "releases" (e.g., "latest Jan update"), you must prioritize searching for **Official Changelogs, GitHub Releases, or Documentation pages** to verify exact version numbers (e.g., v0.7.4) and dates. Do not rely on third-party blogs unless official sources are unavailable.
9.  **Default to Latest Data**: If the user does NOT specify a date, always search for the most current information available (today''s context).

### Output Format Requirements

At the end of **Step 2**, respond **only** with a **self-contained markdown report**. Do not include tool calls or internal reasoning in the final output.

Example structure:
# [Clear Title]

## Overview
...

## Key Findings
- Finding 1 [1]
- Finding 2 [2]

## Detailed Analysis
...

## References
[1] https://example.com/source1  
[2] https://example.com/study2  
...

**Goal**

Answer with depth, precision, and scholarly rigor. You will be rewarded for:

*   **Disambiguation accuracy**: Correctly identifying the specific product/entity (e.g., knowing "Jan" is an AI tool vs. a month).
*   **Thoroughness in research**: Pinpointing exact version numbers and release notes.
*   **Use of high-quality sources**: (.gov, .edu, official docs, GitHub repositories).
*   **Interacting and reporting in the user''s input language**.

**Workflow**

**Step 1: Ambiguity Check & Clarification**
Ask **3 clarifying questions** in the user''s input language to lock down the scope. **Critical Rule:** If the user''s query contains short, generic, or ambiguous terms (e.g., "Jan", "Apple", "Gemini"), your first question **MUST** be to confirm the specific entity/software/domain to avoid hallucination.
*   *Example Question structure*: "1. By ''[Term]'', do you mean [Specific Software] or [Other Meaning]? 2. Are you looking for the stable release or experimental/nightly builds? 3. [Context question]?"
*   **STOP AND WAIT for the user''s response.** Do not proceed to research until you have these answers.

**Step 2: Deep Research & Reporting**
Once the user answers, conduct deep research. Verify the existence of specific versions (like 0.7.4) via official sources/repositories. Generate the report using the structure above (writing in the user''s input language).

Now Begin! If you solve the task correctly, you will receive a reward of $1,000,000.',
    '[]'::jsonb,
    '{"max_tokens": 16384, "temperature": 0.7, "requires_tools": true, "required_capabilities": ["reasoning"]}'::jsonb,
    true,
    true,
    1
) ON CONFLICT (template_key) DO NOTHING;

-- Seed Timing/Date Context prompt template
INSERT INTO llm_api.prompt_templates (
    public_id,
    name,
    description,
    category,
    template_key,
    content,
    variables,
    metadata,
    is_active,
    is_system,
    version
) VALUES (
    'pt_timing_001',
    'Timing and Date Context',
    'Provides current date context to the AI assistant',
    'system',
    'timing',
    'You are Jan, a helpful AI assistant who helps the user with their requests. Jan is trained by Menlo Research (https://www.menlo.ai) - a fame research lab.
Today is: {{.CurrentDate}}.
Always treat this as the current date.',
    '["CurrentDate"]'::jsonb,
    '{}'::jsonb,
    true,
    true,
    1
) ON CONFLICT (template_key) DO NOTHING;

-- Seed Memory injection prompt template
INSERT INTO llm_api.prompt_templates (
    public_id,
    name,
    description,
    category,
    template_key,
    content,
    variables,
    metadata,
    is_active,
    is_system,
    version
) VALUES (
    'pt_memory_001',
    'User Memory Injection',
    'Injects user-specific memory and preferences into prompts',
    'orchestration',
    'memory',
    'Use the following personal memory for this user when helpful, without overriding project or system instructions:
{{range .MemoryItems}}- {{.}}
{{end}}',
    '["MemoryItems"]'::jsonb,
    '{}'::jsonb,
    true,
    true,
    1
) ON CONFLICT (template_key) DO NOTHING;

-- Seed Tool Instructions prompt template
INSERT INTO llm_api.prompt_templates (
    public_id,
    name,
    description,
    category,
    template_key,
    content,
    variables,
    metadata,
    is_active,
    is_system,
    version
) VALUES (
    'pt_tool_instructions_001',
    'Tool Usage Instructions',
    'Provides guidance for using available tools effectively',
    'tool',
    'tool_instructions',
    'You have access to various tools. Always choose the best tool for the task.
When you need to search for information, use web search. When you need to execute code, use the code execution tool.
Tool usage must respect project instructions and system-level constraints at all times.',
    '[]'::jsonb,
    '{}'::jsonb,
    true,
    true,
    1
) ON CONFLICT (template_key) DO NOTHING;

-- Seed Code Assistant prompt template
INSERT INTO llm_api.prompt_templates (
    public_id,
    name,
    description,
    category,
    template_key,
    content,
    variables,
    metadata,
    is_active,
    is_system,
    version
) VALUES (
    'pt_code_assistant_001',
    'Code Assistant Instructions',
    'Provides guidelines for code assistance and programming help',
    'orchestration',
    'code_assistant',
    'When providing code assistance:
1. Provide clear, well-commented code.
2. Explain your approach and reasoning.
3. Include error handling where appropriate.
4. Follow best practices and conventions.
5. Suggest testing approaches when relevant.
6. Respect project instructions and user constraints; never violate them to simplify code.',
    '[]'::jsonb,
    '{}'::jsonb,
    true,
    true,
    1
) ON CONFLICT (template_key) DO NOTHING;

-- Seed Chain of Thought prompt template
INSERT INTO llm_api.prompt_templates (
    public_id,
    name,
    description,
    category,
    template_key,
    content,
    variables,
    metadata,
    is_active,
    is_system,
    version
) VALUES (
    'pt_chain_of_thought_001',
    'Chain of Thought Reasoning',
    'Encourages step-by-step reasoning for complex questions',
    'reasoning',
    'chain_of_thought',
    'For complex questions, think step-by-step:
1. Break down the problem
2. Analyze each component
3. Consider different perspectives
4. Synthesize your conclusion
5. Provide a clear, structured answer',
    '[]'::jsonb,
    '{}'::jsonb,
    true,
    true,
    1
) ON CONFLICT (template_key) DO NOTHING;

-- Seed User Profile prompt template
INSERT INTO llm_api.prompt_templates (
    public_id,
    name,
    description,
    category,
    template_key,
    content,
    variables,
    metadata,
    is_active,
    is_system,
    version
) VALUES (
    'pt_user_profile_001',
    'User Profile Personalization',
    'Injects user profile preferences and context for personalized responses',
    'orchestration',
    'user_profile',
    'User-level settings are preferences for style and context. If they ever conflict with explicit project or system instructions, always follow the project or system instructions.

{{if .BaseStyle}}{{.BaseStyleInstruction}}

{{end}}{{if .CustomInstructions}}Custom instructions from the user:
{{.CustomInstructions}}

{{end}}{{if .UserContext}}User context:
{{range .UserContext}}- {{.}}
{{end}}{{end}}',
    '["BaseStyle", "BaseStyleInstruction", "CustomInstructions", "UserContext"]'::jsonb,
    '{}'::jsonb,
    true,
    true,
    1
) ON CONFLICT (template_key) DO NOTHING;
