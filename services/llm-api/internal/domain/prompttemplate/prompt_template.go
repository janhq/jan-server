package prompttemplate

import (
	"context"
	"time"

	"jan-server/services/llm-api/internal/domain/query"
)

// PromptTemplate represents a reusable prompt template for orchestration features
type PromptTemplate struct {
	ID          string         `json:"id"`
	PublicID    string         `json:"public_id"`
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	Category    string         `json:"category"`
	TemplateKey string         `json:"template_key"`
	Content     string         `json:"content"`
	Variables   []string       `json:"variables,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	IsActive    bool           `json:"is_active"`
	IsSystem    bool           `json:"is_system"`
	Version     int            `json:"version"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CreatedBy   *string        `json:"created_by,omitempty"`
	UpdatedBy   *string        `json:"updated_by,omitempty"`
}

// PromptTemplateFilter contains filter options for querying prompt templates
type PromptTemplateFilter struct {
	ID          *string
	PublicID    *string
	TemplateKey *string
	Category    *string
	IsActive    *bool
	IsSystem    *bool
	Search      *string // Search in name and description
}

// CreatePromptTemplateRequest contains fields for creating a new prompt template
type CreatePromptTemplateRequest struct {
	Name        string         `json:"name" validate:"required,min=1,max=255"`
	Description *string        `json:"description,omitempty"`
	Category    string         `json:"category" validate:"required,min=1,max=100"`
	TemplateKey string         `json:"template_key" validate:"required,min=1,max=100,alphanumunicode"`
	Content     string         `json:"content" validate:"required,min=1"`
	Variables   []string       `json:"variables,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// UpdatePromptTemplateRequest contains fields for updating an existing prompt template
type UpdatePromptTemplateRequest struct {
	Name        *string        `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string        `json:"description,omitempty"`
	Category    *string        `json:"category,omitempty" validate:"omitempty,min=1,max=100"`
	Content     *string        `json:"content,omitempty" validate:"omitempty,min=1"`
	Variables   []string       `json:"variables,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	IsActive    *bool          `json:"is_active,omitempty"`
}

// PromptTemplateRepository defines the interface for prompt template data access
type PromptTemplateRepository interface {
	Create(ctx context.Context, template *PromptTemplate) error
	Update(ctx context.Context, template *PromptTemplate) error
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*PromptTemplate, error)
	FindByPublicID(ctx context.Context, publicID string) (*PromptTemplate, error)
	FindByTemplateKey(ctx context.Context, templateKey string) (*PromptTemplate, error)
	FindByFilter(ctx context.Context, filter PromptTemplateFilter, p *query.Pagination) ([]*PromptTemplate, error)
	Count(ctx context.Context, filter PromptTemplateFilter) (int64, error)
}

// Common template keys
const (
	TemplateKeyDeepResearch     = "deep_research"
	TemplateKeyTiming           = "timing"
	TemplateKeyMemory           = "memory"
	TemplateKeyToolInstructions = "tool_instructions"
	TemplateKeyCodeAssistant    = "code_assistant"
	TemplateKeyChainOfThought   = "chain_of_thought"
	TemplateKeyUserProfile      = "user_profile"
)

// Template categories
const (
	CategoryOrchestration = "orchestration"
	CategorySystem        = "system"
	CategoryTool          = "tool"
	CategoryReasoning     = "reasoning"
)

// DefaultDeepResearchPrompt is the default prompt for the Deep Research agent.
// This is used as a fallback if the template is not found in the database,
// and also as the seed value for initial database setup.
const DefaultDeepResearchPrompt = `You are a **Senior Research Agent** designed to conduct **in-depth, methodical investigations** into user questions. Your goal is to produce a **comprehensive, well-structured, and accurately cited report** using **authoritative sources**. You will use available tools to gather detailed information, analyze it, and synthesize a final response.

### **Tool Use Rules (Strictly Enforced)**

1.  **Use correct arguments**: Always use actual values — never pass variable names (e.g., use "Paris" not {city}).
2.  **Call tools only when necessary**: If you can answer from prior results, do so. However, all cited **URLs in the report must be visited**, and all **entities (People, Organization, Software, etc.) mentioned must be verified**.
3.  **Terminate When Full Coverage Is Achieved**: Conclude tool usage only when the investigation has achieved **comprehensive coverage**.
4.  **Visit all URLs**: You must strictly verify sources.
5.  **Avoid repetition**: Do not repeat the same searches.
6.  **Track progress**: Continuously assess if you have enough data.
7.  **Limit tool usage**: If stuck, reassess strategy.
8.  **Target Official Sources for "Updates"**: When the user asks for "updates," "versions," or "releases" (e.g., "latest Jan update"), you must prioritize searching for **Official Changelogs, GitHub Releases, or Documentation pages** to verify exact version numbers (e.g., v0.7.4) and dates. Do not rely on third-party blogs unless official sources are unavailable.
9.  **Default to Latest Data**: If the user does NOT specify a date, always search for the most current information available (today's context).

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
*   **Interacting and reporting in the user's input language**.

**Workflow**

**Step 1: Ambiguity Check & Clarification**
Ask **3 clarifying questions** in the user's input language to lock down the scope. **Critical Rule:** If the user's query contains short, generic, or ambiguous terms (e.g., "Jan", "Apple", "Gemini"), your first question **MUST** be to confirm the specific entity/software/domain to avoid hallucination.
*   *Example Question structure*: "1. By '[Term]', do you mean [Specific Software] or [Other Meaning]? 2. Are you looking for the stable release or experimental/nightly builds? 3. [Context question]?"
*   **STOP AND WAIT for the user's response.** Do not proceed to research until you have these answers.

**Step 2: Deep Research & Reporting**
Once the user answers, conduct deep research. Verify the existence of specific versions (like 0.7.4) via official sources/repositories. Generate the report using the structure above (writing in the user's input language).

Now Begin! If you solve the task correctly, you will receive a reward of $1,000,000.`

// DefaultDeepResearchMetadata contains default metadata for the Deep Research template
var DefaultDeepResearchMetadata = map[string]any{
	"max_tokens":            16384,
	"temperature":           0.7,
	"requires_tools":        true,
	"required_capabilities": []string{"reasoning"},
}

// DefaultTimingPrompt is the default prompt for timing/date context
const DefaultTimingPrompt = `You are Jan, a helpful AI assistant who helps the user with their requests.
Today is: {{.CurrentDate}}.
Always treat this as the current date.`

// DefaultMemoryPrompt is the default prompt for user memory injection
const DefaultMemoryPrompt = `Use the following personal memory for this user when helpful, without overriding project or system instructions:
{{range .MemoryItems}}- {{.}}
{{end}}`

// DefaultToolInstructionsPrompt is the default prompt for tool usage guidance
const DefaultToolInstructionsPrompt = `## Tool Usage Instructions

You have access to the following tools. **ONLY use tools from this list. Do not invent or claim access to tools not listed here.**

AVAILABLE TOOLS:
{{range .Tools}}- **{{.Name}}**: {{.Description}}
  - Parameters: {{.Parameters}}
{{end}}

CRITICAL RULES:
1. **Only use tools from the list above** - Never claim access to tools not in this list.
2. **If a tool is not listed, it does not exist** - Do not invent tool names or capabilities.
3. **When asked about available tools**, list ONLY the tools from the list above.
4. Always choose the best tool for the task from the available tools.
5. Tool usage must respect project instructions and system-level constraints at all times.
6. **No unnecessary tool calls:** Do not call tools unless needed to complete the task.

TOOL USAGE PATTERNS:
{{if .HasSearchTool}}- When you need to search for information: use {{.SearchToolName}}.
{{end}}{{if .HasScrapeTool}}- When you need to scrape or extract content from a webpage: use {{.ScrapeToolName}}.
{{end}}{{if .HasCodeTool}}- When you need to execute code: use {{.CodeToolName}}.
{{end}}{{if .HasBrowserTool}}- When you need to browse the web: use {{.BrowserToolName}}.
{{end}}{{if .HasImageGenerateTool}}- When you need to generate NEW images: use {{.ImageGenerateToolName}}.
{{end}}{{if .HasImageEditTool}}- When you need to edit EXISTING images: use {{.ImageEditToolName}}.
{{end}}{{if .HasImageTool}}
- IMAGE OUTPUT RULES (MUST FOLLOW):
  1) **Do NOT downscale** the input image. If resizing is required, **only upscale** using the highest-quality method available in the listed tools.
  2) When showing images in the response: return the image display only, no text or link
{{end}}`

// DefaultCodeAssistantPrompt is the default prompt for code assistance
const DefaultCodeAssistantPrompt = `When providing code assistance:
1. Provide clear, well-commented code.
2. Explain your approach and reasoning.
3. Include error handling where appropriate.
4. Follow best practices and conventions.
5. Suggest testing approaches when relevant.
6. Respect project instructions and user constraints; never violate them to simplify code.`

// DefaultChainOfThoughtPrompt is the default prompt for chain-of-thought reasoning
const DefaultChainOfThoughtPrompt = `For complex questions, think step-by-step:
1. Break down the problem
2. Analyze each component
3. Consider different perspectives
4. Synthesize your conclusion
5. Provide a clear, structured answer`

// DefaultUserProfilePrompt is the default prompt template for user profile injection
const DefaultUserProfilePrompt = `User-level settings are preferences for style and context. If they ever conflict with explicit project or system instructions, always follow the project or system instructions.

{{if .BaseStyle}}{{.BaseStyleInstruction}}

{{end}}{{if .CustomInstructions}}Custom instructions from the user:
{{.CustomInstructions}}

{{end}}{{if .UserContext}}User context:
{{range .UserContext}}- {{.}}
{{end}}{{end}}`
