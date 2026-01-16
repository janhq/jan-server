import type { StepResponse } from "@/services/response-api-service";
import type { SearchResultItem, StepWithResults } from "@/stores/right-sidebar-store";

declare const JAN_API_BASE_URL: string;

export function parseStepOutputToResults(step: StepResponse): SearchResultItem[] {
  if (!step.output_data) return [];

  const output = step.output_data as Record<string, unknown>;

  switch (step.action) {
    case "tool_call":
      return parseToolCallOutput(step, output);
    case "llm_call":
      return parseLLMOutput(output);
    case "artifact_create":
      return parseArtifactOutput(output);
    case "file_operation":
      return parseFileOperationOutput(output);
    default:
      return parseGenericOutput(output);
  }
}

function parseToolCallOutput(step: StepResponse, output: Record<string, unknown>): SearchResultItem[] {
  const results: SearchResultItem[] = [];

  const toolName = (output.tool_name as string) || "";
  const stepAny = step as unknown as Record<string, unknown>;
  const inputParams = (stepAny.input_params as Record<string, unknown>) ||
                      step.actual_params ||
                      step.planned_params ||
                      {};
  const effectiveToolName = toolName || (inputParams.tool as string) || "";

  let parsedContent: Record<string, unknown> | null = null;
  if (Array.isArray(output.content)) {
    const textItem = (output.content as Array<Record<string, unknown>>).find(
      (c) => c.type === "text"
    );
    if (textItem && typeof textItem.text === "string") {
      try {
        parsedContent = JSON.parse(textItem.text);
      } catch {
        parsedContent = { text: textItem.text };
      }
    }
  }

  if (effectiveToolName.includes("search")) {
    const searchData = parsedContent || output;
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

  if (results.length === 0 && parsedContent) {
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

function parseLLMOutput(output: Record<string, unknown>): SearchResultItem[] {
  const results: SearchResultItem[] = [];

  const content = output.content as string || output.text as string || output.response as string;
  if (content) {
    results.push({
      type: "text",
      title: "AI Analysis",
      content: content,
    });
  }

  return results;
}

function parseArtifactOutput(output: Record<string, unknown>): SearchResultItem[] {
  const results: SearchResultItem[] = [];

  const artifact = output.artifact as Record<string, unknown>;
  if (artifact) {
    const filename = artifact.filename as string || "artifact";
    const type = artifact.type as string || "file";
    const downloadUrl = artifact.download_url as string;
    const resolvedUrl = resolveArtifactUrl(downloadUrl);

    results.push({
      type: "link",
      title: filename,
      description: `${type} artifact created`,
      url: resolvedUrl || "",
    });
  }

  return results;
}

function resolveArtifactUrl(url: string | undefined): string {
  if (!url) return "";
  if (/^https?:\/\//i.test(url)) return url;
  const base = (JAN_API_BASE_URL || "").trim();
  if (!base) return url;
  const normalizedBase = base.endsWith("/") ? base : `${base}/`;
  const normalizedUrl = url.startsWith("/") ? url.slice(1) : url;
  return `${normalizedBase}${normalizedUrl}`;
}

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

function parseGenericOutput(output: Record<string, unknown>): SearchResultItem[] {
  const results: SearchResultItem[] = [];

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

export function getStepLabel(step: StepResponse): string {
  const stepAny = step as unknown as Record<string, unknown>;
  const inputParams = (stepAny.input_params as Record<string, unknown>) || {};
  const params = step.actual_params || step.planned_params || inputParams;
  const paramsObj = params as Record<string, unknown>;

  switch (step.action) {
    case "tool_call": {
      const toolName = (paramsObj.tool as string) || (paramsObj.tool_name as string) || "Tool";
      const query = (paramsObj.q as string) || (paramsObj.query as string) || (paramsObj.input as string) || (paramsObj.url as string) || "";
      const description = paramsObj.description as string;

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

export function getStepToolName(step: StepResponse): string {
  const stepAny = step as unknown as Record<string, unknown>;
  const inputParams = (stepAny.input_params as Record<string, unknown>) || {};
  const params = step.actual_params || step.planned_params || inputParams;
  const paramsObj = params as Record<string, unknown>;

  if (step.action === "tool_call") {
    const toolName = (paramsObj.tool as string) || (paramsObj.tool_name as string) || "";
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
