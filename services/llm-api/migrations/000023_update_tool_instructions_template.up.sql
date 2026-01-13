-- +goose Up
-- Update tool_instructions template with image tool support and detailed instructions
UPDATE llm_api.prompt_templates
SET content = '## Tool Usage Instructions

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
{{end}}',
    updated_at = NOW(),
    version = version + 1
WHERE template_key = 'tool_instructions'
  AND public_id = 'pt_tool_instructions_001';
