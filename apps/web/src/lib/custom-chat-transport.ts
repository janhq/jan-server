import { type UIMessage } from "@ai-sdk/react";
import {
  convertToModelMessages,
  streamText,
  type ChatRequestOptions,
  type ChatTransport,
  type LanguageModel,
  type UIMessageChunk,
  type Tool,
  jsonSchema,
  type CoreMessage,
} from "ai";
import { mcpService } from "@/services/mcp-service";
import { CONTENT_TYPE, MESSAGE_ROLE } from "@/constants";

/**
 * Custom download function that returns null for all URLs.
 * This tells the AI SDK to pass URLs directly to the model
 * instead of downloading and converting to base64.
 *
 * This avoids CORS issues with presigned S3 URLs and reduces payload size.
 */
async function passUrlsDirectly(
  options: Array<{ url: URL; isUrlSupportedByModel: boolean }>,
): Promise<Array<{ data: Uint8Array; mediaType: string | undefined } | null>> {
  // Return null for all URLs to pass them directly to the model
  return options.map(() => null);
}

/**
 * Process file parts for the API:
 * - Images: Keep as 'file' type - the OpenAI-compatible provider will convert to image_url
 * - Documents: Convert to text parts with providerOptions override to emit { type: "file", file: {...} }
 *
 * Note: AI SDK's convertToModelMessages converts FileUIPart.url -> FilePart.data
 * So we need to check for both 'url' and 'data' fields.
 *
 * The backend will:
 * - Pass image_url parts directly to the LLM provider
 * - Convert file parts into [FILE_URL:...] placeholders and inject extracted text
 */
function processFileParts(messages: CoreMessage[]): CoreMessage[] {
  return messages.map((message) => {
    if (message.role === MESSAGE_ROLE.USER && Array.isArray(message.content)) {
      return {
        ...message,
        content: message.content
          .map((part) => {
            // Process file parts
            // AI SDK converts FileUIPart { url } to FilePart { data }
            if (part.type === "file") {
              const filePart = part as {
                type: "file";
                mediaType?: string;
                data?: string | Uint8Array; // AI SDK uses 'data' not 'url'
                url?: string; // Original field (may not be present after conversion)
                filename?: string;
              };
              const mediaType = filePart.mediaType || "";
              // Get URL from either 'data' (after AI SDK conversion) or 'url' (original)
              const fileUrl =
                typeof filePart.data === "string"
                  ? filePart.data
                  : filePart.url;

              // Images: Keep as file type - OpenAI provider will convert to image_url
              if (mediaType.startsWith("image/")) {
                // Return as-is, the provider handles conversion
                return part;
              }

              // Documents/files: Convert to text part with providerOptions override
              // so the request emits { type: "file", file: {...} }
              if (fileUrl) {
                return {
                  type: CONTENT_TYPE.TEXT,
                  // Non-empty to avoid AI SDK filtering out empty user text parts
                  text: "[file]",
                  providerOptions: {
                    openaiCompatible: {
                      type: "file",
                      file: {
                        url: fileUrl,
                        name: filePart.filename || "document",
                        mime_type: mediaType || "application/octet-stream",
                      },
                    },
                  },
                };
              }
              return null;
            }

            return part;
          })
          .filter((part): part is NonNullable<typeof part> => part !== null),
      };
    }
    return message;
  }) as CoreMessage[];
}

/**
 * Convert user text parts to input_text parts in the outgoing request.
 */
function applyInputTextOverrides(messages: CoreMessage[]): CoreMessage[] {
  return messages.map((message) => {
    if (message.role !== MESSAGE_ROLE.USER || !Array.isArray(message.content)) {
      return message;
    }

    return {
      ...message,
      content: message.content.map((part) => {
        if (part.type !== CONTENT_TYPE.TEXT) {
          return part;
        }

        const existingOptions =
          (part as { providerOptions?: Record<string, unknown> })
            .providerOptions ?? {};
        const existingOpenAI =
          (existingOptions.openaiCompatible as Record<string, unknown>) ?? {};

        if (existingOpenAI.type === "file") {
          return part;
        }

        return {
          ...part,
          providerOptions: {
            ...existingOptions,
            openaiCompatible: {
              ...existingOpenAI,
              type: "input_text",
              input_text: "text" in part ? part.text : "",
            },
          },
        };
      }),
    };
  }) as CoreMessage[];
}

/**
 * Filter out base64 image data from tool messages to reduce payload size.
 * Browser MCP tools return screenshots as base64 which are very large.
 * We replace them with a placeholder to avoid sending huge payloads to the API.
 */
function filterBase64FromMessages(messages: CoreMessage[]): CoreMessage[] {
  const base64Pattern = /data:image\/[^;]+;base64,[A-Za-z0-9+/=]{100,}/g;

  return messages.map((message) => {
    if (message.role === MESSAGE_ROLE.TOOL) {
      // Tool messages have content as array of ToolResultPart
      if (Array.isArray(message.content)) {
        return {
          ...message,
          content: message.content.map((part) => {
            // ToolResultPart has type 'tool-result' with a result property
            // The result can contain text with embedded base64
            if ("result" in part && part.result) {
              const resultStr =
                typeof part.result === "string"
                  ? part.result
                  : JSON.stringify(part.result);

              if (base64Pattern.test(resultStr)) {
                return {
                  ...part,
                  result: resultStr.replace(
                    base64Pattern,
                    "[base64 image data removed]",
                  ),
                };
              }
            }
            return part;
          }),
        };
      }
    }

    // Handle user messages with image parts
    if (message.role === MESSAGE_ROLE.USER && Array.isArray(message.content)) {
      return {
        ...message,
        content: message.content.map((part) => {
          // Check for image parts with base64 data
          if (part.type === "image") {
            const imageData = "image" in part ? part.image : null;
            if (
              typeof imageData === "string" &&
              (imageData.startsWith("data:") || imageData.length > 10000)
            ) {
              // Replace with placeholder text
              return {
                type: CONTENT_TYPE.TEXT,
                text: `[Image: screenshot captured]`,
              };
            }
          }
          // Check for text parts with embedded base64
          if (
            part.type === CONTENT_TYPE.TEXT &&
            "text" in part &&
            typeof part.text === "string"
          ) {
            if (base64Pattern.test(part.text)) {
              return {
                ...part,
                text: part.text.replace(
                  base64Pattern,
                  "[base64 image data removed]",
                ),
              };
            }
          }
          return part;
        }),
      };
    }

    return message;
  });
}

export class CustomChatTransport implements ChatTransport<UIMessage> {
  private model: LanguageModel;
  private tools: Record<string, Tool> = {};
  private enableBrowse = false;
  private enableImageTools = false;
  private enableAgentMode = false;

  constructor(
    model: LanguageModel,
    _enabledSearch?: boolean, // Reserved for future use
    enableBrowse?: boolean,
    enableImageTools?: boolean,
    enableAgentMode?: boolean,
  ) {
    this.model = model;
    this.enableBrowse = enableBrowse ?? false;
    this.enableImageTools = enableImageTools ?? false;
    this.enableAgentMode = enableAgentMode ?? false;
    this.initializeTools();
  }

  updateModel(model: LanguageModel) {
    this.model = model;
  }

  updateSearchEnabled(_enabledSearch: boolean) {
    // Reserved for future use
  }
  updateBrowseEnabled(enableBrowse: boolean) {
    this.enableBrowse = enableBrowse;
  }
  updateImageToolsEnabled(enableImageTools: boolean) {
    this.enableImageTools = enableImageTools;
  }

  updateAgentModeEnabled(enableAgentMode: boolean) {
    this.enableAgentMode = enableAgentMode;
  }

  /**
   * Initialize MCP tools and convert them to AI SDK format
   * Search tools (google_search, scrape) are always included regardless of user toggle
   * AIO tools (aio_* prefix) are always filtered out from user-facing chat
   */
  private async initializeTools() {
    try {
      // Fetch all tools from MCP service
      const servers = this.enableAgentMode
        ? (["default"] as string[])
        : this.enableImageTools
          ? undefined
          : ([
              "search", // Always include search tools
              this.enableBrowse ? "browse" : null,
            ].filter(Boolean) as string[]);
      const toolsResponse = await mcpService.getTools(servers);
        const allowedImageTools = new Set(["generate_image", "edit_image"]);
        const blockedTools = new Set(["image_search"]);

      // Filter tools based on mode
        const filteredTools = toolsResponse.data.filter((tool) => {
          if (blockedTools.has(tool.name)) {
            return false;
          }
          // Always filter out AIO tools (prefix: aio_) - these are for internal agent use
          if (tool.name.startsWith("aio_")) {
            return false;
        }

        // Agent mode: only allow run_agent tool
        if (this.enableAgentMode) {
          return tool.name === "run_agent";
        }

        // Normal mode: filter out run_agent
        if (tool.name === "run_agent") {
          return false;
        }

        // Image tool filtering
        return this.enableImageTools
          ? allowedImageTools.has(tool.name)
          : !allowedImageTools.has(tool.name);
      });

      // Convert MCP tools to AI SDK CoreTool format
      this.tools = filteredTools.reduce(
        (acc, tool) => {
          acc[tool.name] = {
            description: tool.description,
            inputSchema: jsonSchema(tool.inputSchema),
            name: tool.name,
          };
          return acc;
        },
        {} as Record<string, Tool>,
      );

      console.log(
        `Initialized ${Object.keys(this.tools).length} tools${this.enableAgentMode ? " (agent mode)" : " (AIO/agent tools filtered)"}`,
      );
    } catch (error) {
      console.error("Failed to initialize MCP tools:", error);
      this.tools = {};
    }
  }

  async sendMessages(
    options: {
      chatId: string;
      messages: UIMessage[];
      abortSignal: AbortSignal | undefined;
    } & {
      trigger: "submit-message" | "regenerate-message";
      messageId: string | undefined;
    } & ChatRequestOptions,
  ): Promise<ReadableStream<UIMessageChunk>> {
    await this.initializeTools();

    // Convert UI messages to model messages
    const modelMessages = convertToModelMessages(options.messages);

    // Filter out base64 data from tool results
    const filteredMessages = filterBase64FromMessages(modelMessages);

    // Process file parts: images stay as-is, documents become file parts
    const processedMessages = processFileParts(filteredMessages);
    const normalizedMessages = applyInputTextOverrides(processedMessages);

    // Always include tools if we have any loaded
    // Search tools (google_search, scrape) are always sent regardless of user toggle
    const hasTools = Object.keys(this.tools).length > 0;
    const result = streamText({
      model: this.model,
      messages: normalizedMessages,
      abortSignal: options.abortSignal,
      // Pass URLs directly to the model instead of downloading
      // This avoids CORS issues with presigned S3 URLs
      experimental_download: passUrlsDirectly,
      tools: hasTools ? this.tools : undefined,
      toolChoice: "auto",
    });

    return result.toUIMessageStream({
      onError: (error) => {
        // Note: By default, the AI SDK will return "An error occurred",
        // which is intentionally vague in case the error contains sensitive information like API keys.
        // If you want to provide more detailed error messages, keep the code below. Otherwise, remove this whole onError callback.
        if (error == null) {
          return "Unknown error";
        }
        if (typeof error === "string") {
          return error;
        }
        if (error instanceof Error) {
          return error.message;
        }
        return JSON.stringify(error);
      },
    });
  }

  async reconnectToStream(
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    _options: {
      chatId: string;
    } & ChatRequestOptions,
  ): Promise<ReadableStream<UIMessageChunk> | null> {
    // This function normally handles reconnecting to a stream on the backend, e.g. /api/chat
    // Since this project has no backend, we can't reconnect to a stream, so this is intentionally no-op.
    return null;
  }
}
