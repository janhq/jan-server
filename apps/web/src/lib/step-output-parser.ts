import type { StepResponse } from "@/services/response-api-service";
import type { ArtifactItem, SearchResultItem, StepWithResults } from "@/stores/right-sidebar-store";

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

  const dataBank = extractDataBank(output);
  if (dataBank) {
    results.push({
      type: "text",
      title: "Data Bank",
      content: formatDataBankMarkdown(dataBank),
    });
    return results;
  }

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
    const id = artifact.id as string || "";
    const filename = artifact.filename as string || "artifact";
    const contentType = artifact.type as string || "file";
    const mimeType = artifact.content_type as string || "application/octet-stream";
    const size = artifact.size as number || 0;
    const downloadUrl = artifact.download_url as string;
    const createdAt = artifact.created_at as string;
    const resolvedUrl = resolveArtifactUrl(downloadUrl);
    const rawSlidesImages = artifact.slides_images as Array<Record<string, unknown>> | undefined;
    const slidesImages = rawSlidesImages?.map((img, idx) => ({
      id: (img.id as string) || String(idx + 1),
      thumb: img.thumb as string || img.url as string || "",
    })).filter(img => img.thumb);

    results.push({
      type: "artifact",
      title: filename,
      description: `${contentType} artifact created`,
      url: resolvedUrl || "",
      downloadUrl: resolvedUrl || "",
      filename,
      requiresAuth: true,
      artifact: {
        id,
        filename,
        contentType,
        mimeType,
        size,
        downloadUrl: resolvedUrl || "",
        createdAt,
        slidesImages: slidesImages && slidesImages.length > 0 ? slidesImages : undefined,
      },
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
  const stepTitle = typeof step.title === "string" ? step.title.trim() : "";
  const stepDescription =
    typeof step.description === "string" ? step.description.trim() : "";
  const paramsDescription =
    typeof paramsObj.description === "string" ? (paramsObj.description as string).trim() : "";
  const preferredLabel = stepTitle || stepDescription || paramsDescription;
  if (preferredLabel) {
    return preferredLabel.length > 80
      ? `${preferredLabel.substring(0, 80)}...`
      : preferredLabel;
  }

  switch (step.action) {
    case "tool_call": {
      const toolName = (paramsObj.tool as string) || (paramsObj.tool_name as string) || "Tool";
      const query = (paramsObj.q as string) || (paramsObj.query as string) || (paramsObj.input as string) || (paramsObj.url as string) || "";
      return query ? `${toolName}: ${query.substring(0, 50)}${query.length > 50 ? "..." : ""}` : toolName;
    }

    case "llm_call":
      if (paramsObj.action === "plan_and_template" && paramsDescription) return paramsDescription;
      if (paramsObj.action === "generate_single_slide" && paramsDescription) return paramsDescription;
      return "Draft Notes";
    
    case "artifact_create":
      return "Save Output";
    case "file_operation": {
      const op = paramsObj.operation as string || "file";
      const filename = paramsObj.filename as string || "";
      return filename ? `${op}: ${filename}` : op;
    }
    case "transform":
      return "Process Content";
    default:
      return step.action || "Step";
  }
}

type DataBankFact = {
  claim?: string;
  value?: string | number;
  unit?: string;
  sourceUrl?: string;
  date?: string;
};

type DataBankSeries = {
  name?: string;
  values?: Array<number | string>;
};

type DataBankDataset = {
  id?: string;
  kind?: string;
  data?: {
    labels?: string[];
    series?: DataBankSeries[];
  };
  sourceNote?: string;
};

type DataBank = {
  facts?: DataBankFact[];
  datasets?: DataBankDataset[];
};

function extractDataBank(output: Record<string, unknown>): DataBank | null {
  const data = output.data as Record<string, unknown> | undefined;
  if (data && (Array.isArray(data.facts) || Array.isArray(data.datasets))) {
    return data as DataBank;
  }

  const content = output.content;
  if (typeof content === "string" && content.trim()) {
    try {
      const parsed = JSON.parse(content) as DataBank;
      if (parsed && (Array.isArray(parsed.facts) || Array.isArray(parsed.datasets))) {
        return parsed;
      }
    } catch {
      return null;
    }
  }

  return null;
}

function formatDataBankMarkdown(data: DataBank): string {
  const lines: string[] = [];

  if (Array.isArray(data.facts) && data.facts.length > 0) {
    lines.push("### Facts");
    const facts = data.facts.slice(0, 12);
    for (const fact of facts) {
      const claim = fact.claim?.trim() || "Fact";
      const valueParts: string[] = [];
      if (fact.value !== undefined && fact.value !== null && `${fact.value}`.trim()) {
        valueParts.push(`${fact.value}`);
      }
      if (fact.unit && fact.unit.trim()) {
        valueParts.push(fact.unit.trim());
      }
      const valueText = valueParts.length > 0 ? ` — ${valueParts.join(" ")}` : "";
      const dateText = fact.date && fact.date.trim() ? ` (${fact.date.trim()})` : "";
      const sourceText = fact.sourceUrl && fact.sourceUrl.trim()
        ? ` ([source](${fact.sourceUrl.trim()}))`
        : "";
      lines.push(`- **${claim}**${valueText}${dateText}${sourceText}`);
    }
  }

  if (Array.isArray(data.datasets) && data.datasets.length > 0) {
    if (lines.length > 0) {
      lines.push("");
    }
    lines.push("### Datasets");
    const datasets = data.datasets.slice(0, 6);
    for (let i = 0; i < datasets.length; i++) {
      const dataset = datasets[i];
      const title = dataset.id?.trim() || `Dataset ${i + 1}`;
      lines.push(`#### ${title}`);
      if (dataset.sourceNote && dataset.sourceNote.trim()) {
        lines.push(`*${dataset.sourceNote.trim()}*`);
      }
      const table = buildDatasetTable(dataset);
      if (table) {
        lines.push(table);
      }
      if (i < datasets.length - 1) {
        lines.push("");
      }
    }
  }

  if (lines.length === 0) {
    return "No facts or datasets available.";
  }

  return lines.join("\n");
}

function buildDatasetTable(dataset: DataBankDataset): string | null {
  const labels = dataset.data?.labels ?? [];
  const series = dataset.data?.series ?? [];
  if (!Array.isArray(labels) || !Array.isArray(series) || labels.length === 0 || series.length === 0) {
    return null;
  }

  const maxRows = 10;
  const rowLabels = labels.slice(0, maxRows);
  const header = ["Label", ...series.map((s, idx) => s.name?.trim() || `Series ${idx + 1}`)];
  const rows = rowLabels.map((label, rowIndex) => {
    const row: string[] = [label];
    for (const s of series) {
      const value = s.values?.[rowIndex];
      row.push(value === undefined || value === null || value === "" ? "-" : String(value));
    }
    return row;
  });

  return toMarkdownTable(header, rows);
}

function toMarkdownTable(headers: string[], rows: string[][]): string {
  const escapeCell = (value: string) => value.replace(/\|/g, "\\|");
  const headerLine = `| ${headers.map(escapeCell).join(" | ")} |`;
  const separatorLine = `| ${headers.map(() => "---").join(" | ")} |`;
  const rowLines = rows.map((row) => `| ${row.map(escapeCell).join(" | ")} |`);
  return [headerLine, separatorLine, ...rowLines].join("\n");
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

export function extractArtifactsFromTasks(
  tasks: { steps?: StepResponse[] }[],
): ArtifactItem[] {
  const artifacts: ArtifactItem[] = [];

  for (const task of tasks) {
    if (!task.steps) continue;

    for (const step of task.steps) {
      if (step.action === "artifact_create" && step.output_data) {
        const output = step.output_data as Record<string, unknown>;
        const artifact = output.artifact as Record<string, unknown>;

        if (artifact) {
          const id = artifact.id as string;
          const downloadUrl = artifact.download_url as string;
          const resolvedUrl = resolveArtifactUrl(downloadUrl);

          // Extract slides images if available
          const rawSlidesImages = artifact.slides_images as Array<Record<string, unknown>> | undefined;
          const slidesImages = rawSlidesImages?.map((img, idx) => ({
            id: (img.id as string) || String(idx + 1),
            thumb: img.thumb as string || img.url as string || "",
          })).filter(img => img.thumb);

          artifacts.push({
            id: id || "",
            filename: artifact.filename as string || "artifact",
            contentType: artifact.type as string || "file",
            mimeType: artifact.content_type as string || "application/octet-stream",
            size: artifact.size as number || 0,
            downloadUrl: resolvedUrl || "",
            createdAt: artifact.created_at as string,
            slidesImages: slidesImages && slidesImages.length > 0 ? slidesImages : undefined,
          });
        }
      }
    }
  }

  return artifacts;
}
