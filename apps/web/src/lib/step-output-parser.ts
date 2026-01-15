import type { StepResponse } from "@/services/response-api-service";
import type { SearchResultItem, StepWithResults } from "@/stores/right-sidebar-store";

/**
 * Parse step output_data to extract search results based on the action type.
 * The response-api stores different data structures for different tools.
 */
export function parseStepOutputToResults(step: StepResponse): SearchResultItem[] {
  if (!step.output_data) return [];

  const output = step.output_data as Record<string, unknown>;

  switch (step.action) {
    case "tool_call":
      return parseToolCallOutput(step, output);
    case "llm_call":
      return parseLLMOutput(output);
    case "file_operation":
      return parseFileOperationOutput(output);
    default:
      return parseGenericOutput(output);
  }
}

/**
 * Parse output from tool_call steps (search, browse, etc.)
 * The response-api stores MCP tool responses in output_data with format:
 * { content: [{ type: "text", text: "<JSON string>" }], tool_name: "...", is_error: false }
 */
function parseToolCallOutput(step: StepResponse, output: Record<string, unknown>): SearchResultItem[] {
  const results: SearchResultItem[] = [];

  // Get tool name from output or params
  const toolName = (output.tool_name as string) || "";
  const stepAny = step as unknown as Record<string, unknown>;
  const inputParams = (stepAny.input_params as Record<string, unknown>) ||
                      step.actual_params ||
                      step.planned_params ||
                      {};
  const effectiveToolName = toolName || (inputParams.tool as string) || "";

  // Extract content from MCP response format (content is an array with text items)
  let parsedContent: Record<string, unknown> | null = null;
  if (Array.isArray(output.content)) {
    const textItem = (output.content as Array<Record<string, unknown>>).find(
      (c) => c.type === "text"
    );
    if (textItem && typeof textItem.text === "string") {
      try {
        parsedContent = JSON.parse(textItem.text);
      } catch {
        // Not JSON, use text as-is
        parsedContent = { text: textItem.text };
      }
    }
  }

  // Handle search tool results (google_search, brave_search, etc.)
  if (effectiveToolName.includes("search")) {
    const searchData = parsedContent || output;
    // Try different result locations based on search provider format
    const searchResults =
      (searchData.results as Array<Record<string, unknown>>) ||
      ((searchData.raw as Record<string, unknown>)?.organic as Array<Record<string, unknown>>) ||
      [];

    for (const result of searchResults) {
      results.push({
        type: "link",
        title: (result.title as string) || "",
        description: (result.snippet as string) || (result.description as string) || "",
        url: (result.source_url as string) || (result.link as string) || (result.url as string) || "",
      });
    }
  }

  // Handle browse/fetch tool results
  if (effectiveToolName.includes("browse") || effectiveToolName.includes("fetch") || effectiveToolName.includes("scrape")) {
    const browseData = parsedContent || output;
    const content = (browseData.content as string) || (browseData.text as string) || (browseData.markdown as string);
    const url = (browseData.url as string) || (inputParams.url as string);
    if (content || url) {
      results.push({
        type: "link",
        title: (browseData.title as string) || url || "Web Content",
        description: typeof content === "string" ? content.substring(0, 300) : "",
        url: url || "",
      });
    }
  }

  // Handle image results
  const imageData = parsedContent || output;
  const images = (imageData.images as Array<Record<string, unknown>>) || [];
  for (const img of images) {
    results.push({
      type: "image",
      title: (img.title as string) || "",
      description: (img.alt as string) || (img.description as string) || "",
      imageUrl: (img.url as string) || (img.src as string) || "",
    });
  }

  // If no specific parsing worked, show formatted text content
  if (results.length === 0 && parsedContent) {
    // For search results that weren't parsed, show the query info
    if (parsedContent.query) {
      results.push({
        type: "text",
        title: `Search: ${parsedContent.query}`,
        content: `Found ${(parsedContent.results as Array<unknown>)?.length || 0} results`,
      });
    } else if (typeof parsedContent.text === "string") {
      results.push({
        type: "text",
        title: effectiveToolName || "Tool Output",
        content: parsedContent.text.substring(0, 500) + (parsedContent.text.length > 500 ? "..." : ""),
      });
    }
  }

  return results;
}

/**
 * Parse output from LLM call steps
 */
function parseLLMOutput(output: Record<string, unknown>): SearchResultItem[] {
  const results: SearchResultItem[] = [];

  const content = output.content as string || output.text as string || output.response as string;
  if (content) {
    results.push({
      type: "text",
      title: "LLM Response",
      content: content,
    });
  }

  return results;
}

/**
 * Parse output from file operation steps
 */
function parseFileOperationOutput(output: Record<string, unknown>): SearchResultItem[] {
  const results: SearchResultItem[] = [];

  const content = output.content as string || output.data as string;
  const filename = output.filename as string || output.path as string;

  if (content) {
    results.push({
      type: "text",
      title: filename || "File Content",
      content: content,
    });
  }

  return results;
}

/**
 * Parse generic output as fallback
 */
function parseGenericOutput(output: Record<string, unknown>): SearchResultItem[] {
  const results: SearchResultItem[] = [];

  // Try to extract any text content
  if (typeof output.text === "string" && output.text) {
    results.push({
      type: "text",
      content: output.text,
    });
  }

  if (typeof output.content === "string" && output.content) {
    results.push({
      type: "text",
      content: output.content,
    });
  }

  return results;
}

/**
 * Get a human-readable label for a step based on its action and parameters
 */
export function getStepLabel(step: StepResponse): string {
  // Check input_params first (from response-api), then fall back to others
  const stepAny = step as unknown as Record<string, unknown>;
  const inputParams = (stepAny.input_params as Record<string, unknown>) || {};
  const params = step.actual_params || step.planned_params || inputParams;
  const paramsObj = params as Record<string, unknown>;

  switch (step.action) {
    case "tool_call": {
      // Tool name can be in different locations depending on the format
      const toolName = (paramsObj.tool as string) || (paramsObj.tool_name as string) || "Tool";
      // Query/search term can be in q, query, input, or url
      const query = (paramsObj.q as string) || (paramsObj.query as string) || (paramsObj.input as string) || (paramsObj.url as string) || "";
      const description = paramsObj.description as string;

      // Use description if available (more human-readable)
      if (description) {
        return description.substring(0, 60) + (description.length > 60 ? "..." : "");
      }
      return query ? `${toolName}: ${query.substring(0, 50)}${query.length > 50 ? "..." : ""}` : toolName;
    }
    case "llm_call":
      return "AI Analysis";
    case "artifact_create":
      return "Creating Report";
    case "file_operation": {
      const op = paramsObj.operation as string || "file";
      const filename = paramsObj.filename as string || "";
      return filename ? `${op}: ${filename}` : op;
    }
    default:
      return step.action || "Step";
  }
}

/**
 * Get the tool name/icon identifier for a step
 */
export function getStepToolName(step: StepResponse): string {
  // Check input_params first (from response-api), then fall back to others
  const stepAny = step as unknown as Record<string, unknown>;
  const inputParams = (stepAny.input_params as Record<string, unknown>) || {};
  const params = step.actual_params || step.planned_params || inputParams;
  const paramsObj = params as Record<string, unknown>;

  if (step.action === "tool_call") {
    const toolName = (paramsObj.tool as string) || (paramsObj.tool_name as string) || "";
    // Normalize tool names to match existing icon mappings
    if (toolName.includes("search") || toolName.includes("google") || toolName.includes("brave")) return "Search";
    if (toolName.includes("browse") || toolName.includes("fetch") || toolName.includes("scrape")) return "Browse";
    if (toolName.includes("file") || toolName.includes("write")) return "File";
    if (toolName.includes("bash") || toolName.includes("exec")) return "Bash";
    return toolName;
  }

  if (step.action === "llm_call") return "AI";
  if (step.action === "artifact_create") return "File";
  if (step.action === "file_operation") return "File";

  return step.action || "Step";
}

/**
 * Convert a TaskResponse with steps to StepWithResults format for the right sidebar
 */
export function convertTaskToStepWithResults(
  task: { steps?: StepResponse[] },
): StepWithResults[] {
  if (!task.steps) return [];

  return task.steps.map((step) => ({
    stepName: getStepToolName(step),
    stepTitle: getStepLabel(step),
    results: parseStepOutputToResults(step),
  }));
}
