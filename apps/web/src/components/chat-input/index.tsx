/* eslint-disable react-hooks/exhaustive-deps */
import {
  PromptInput,
  PromptInputActionAddImages,
  PromptInputActionAddFiles,
  PromptInputActionMenu,
  PromptInputActionMenuContent,
  PromptInputActionMenuTrigger,
  PromptInputAttachment,
  PromptInputAttachments,
  PromptInputBody,
  PromptInputButton,
  PromptInputFooter,
  type PromptInputMessage,
  PromptInputProvider,
  // PromptInputSpeechButton,
  PromptInputSubmit,
  PromptInputTextarea,
  PromptInputTools,
  usePromptInputController,
} from "@janhq/interfaces/ai-elements/prompt-input";

import React, { memo, useRef, useEffect, useState, useMemo } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useModels } from "@/stores/models-store";
import { useConversations } from "@/stores/conversation-store";
import { useCapabilities } from "@/stores/capabilities-store";
import { useProjects } from "@/stores/projects-store";
import { useIsMobile } from "@janhq/interfaces/hooks/use-mobile";
import { useIsMobileDevice } from "@janhq/interfaces/hooks/use-is-mobile-device";
import { toast } from "@janhq/interfaces/sonner";
import {
  BotIcon,
  FolderIcon,
  GlobeIcon,
  ImageIcon,
  LightbulbIcon,
  TelescopeIcon,
  Settings2,
  X,
} from "lucide-react";
import { isChromeBrowser } from "@janhq/mcp-web-client";

import { BorderAnimate } from "@janhq/interfaces/border-animate";
import { cn } from "@/lib/utils";
import type { ChatStatus } from "ai";
import { SettingChatInput } from "./setting-chat-input";
import { ProjectsChatInput } from "./projects-chat-input";
import { Button } from "@janhq/interfaces/button";
import { usePrivateChat } from "@/stores/private-chat-store";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@janhq/interfaces/tooltip";
import { useBrowserConnection } from "@/stores/browser-connection-store";
import { useProfile } from "@/stores/profile-store";
import {
  CHAT_STATUS,
  CONNECTION_STATE,
  SESSION_STORAGE_KEY,
  SESSION_STORAGE_PREFIX,
} from "@/constants";
import {
  blobUrlToDataUrl,
  uploadMedia,
  createJanMediaUrl,
} from "@/services/media-upload-service";
import {
  isDocumentType,
  scanDocument,
} from "@/services/document-upload-service";
import { mcpService } from "@/services/mcp-service";
import type { UploadServiceConfig } from "@janhq/interfaces/ai-elements/prompt-input";

// File type constants
const ACCEPTED_IMAGE_TYPES = ["image/jpeg", "image/jpg", "image/png"];
const ACCEPTED_DOCUMENT_TYPES = [
  "application/pdf",
  "application/msword",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
  "text/plain",
  "text/markdown",
  "text/html",
  "application/rtf",
];
const ACCEPTED_FILE_TYPES = [
  ...ACCEPTED_IMAGE_TYPES,
  ...ACCEPTED_DOCUMENT_TYPES,
].join(",");

// File limits
const MAX_IMAGES = 10;
const MAX_FILE_SIZE = 50 * 1024 * 1024; // 50MB for documents (images have smaller practical limits)

/**
 * Generates a meaningful title from message text.
 * Returns 'New Conversation' if the text contains only special characters, URLs, or is empty.
 */
function generateThreadTitle(text: string): string {
  const trimmed = text.trim();

  // Check if the text is only a URL (or multiple URLs)
  // This regex matches common URL patterns
  const urlPattern =
    /^(https?:\/\/|www\.)[^\s]+$|^([^\s]+\.(com|org|net|io|co|edu|gov|app|dev|ai|me|info|biz|tv|cc|xyz|tech|online|site|store|blog|cloud|pro|link|page|space|live|world|zone|digital|agency|email|network|social|media|video|music|news|shop|design|studio|work|team|group|chat|help|support|docs|api|cdn|assets|static|img|images|files|data|db|admin|dashboard|portal|app|mobile|web|server|service|system|platform|solution|software|tool|tools|product|project|projects|code|dev|test|stage|prod|demo|beta|alpha|v1|v2|v3)[^\s]*)$/i;

  const isOnlyUrl = urlPattern.test(trimmed);

  // Check if the text contains any word characters (letters or numbers, including Unicode)
  const hasWordCharacters = /[\p{L}\p{N}]/u.test(trimmed);

  if (!hasWordCharacters || isOnlyUrl) {
    return "New Conversation";
  }

  // Truncate long titles to keep them manageable
  const maxLength = 100;
  if (trimmed.length > maxLength) {
    return trimmed.substring(0, maxLength).trim() + "...";
  }

  return trimmed;
}

// Component to handle resetting input when conversation changes
// IMPORTANT: This must be defined outside ChatInput to prevent remounting on every render
const InputResetHandler = ({
  textareaRef,
  isMobile,
  conversationId,
}: {
  textareaRef: React.RefObject<HTMLTextAreaElement | null>;
  isMobile: boolean;
  conversationId?: string;
}) => {
  const controller = usePromptInputController();

  useEffect(() => {
    // Reset input and attachments when conversationId changes
    controller.textInput.clear();
    controller.attachments.clear();

    // Restore focus after clearing (desktop only)
    if (textareaRef.current && !isMobile) {
      textareaRef.current.focus();
    }
  }, [conversationId]);

  return null;
};

// Component to handle document scanning after upload completes
// Scans uploaded documents to extract text via OCR
const DocumentScanHandler = () => {
  const controller = usePromptInputController();
  const scanningRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    const files = controller.attachments.files;

    // Find files that are documents, uploaded successfully, but not yet scanned
    const documentsToScan = files.filter((file) => {
      const mimeType = file.mediaType || "";
      return (
        isDocumentType(mimeType) &&
        file.uploadStatus === "completed" &&
        file.mediaId &&
        !file.isDocument && // Not yet marked as document (scanning not started)
        !scanningRef.current.has(file.id)
      );
    });

    if (documentsToScan.length === 0) return;

    // Scan each document
    documentsToScan.forEach((file) => {
      scanningRef.current.add(file.id);

      // Mark as document and set scanning status
      controller.attachments.updateFile(file.id, {
        isDocument: true,
        documentStatus: "processing",
      });

      // Trigger document scan
      (async () => {
        try {
          // Extract media object ID from the URL
          const mediaId = file.mediaId || "";
          const mediaObjectId = extractMediaObjectId(mediaId);

          const scanResult = await scanDocument(
            mediaObjectId,
            file.filename || "document",
          );

          // Update file with scan results
          controller.attachments.updateFile(file.id, {
            documentId: scanResult.id,
            extractedText: scanResult.extracted_text,
            pageCount: scanResult.page_count,
            wordCount: scanResult.word_count,
            documentStatus: scanResult.processing_status === "completed" ? "completed" : "failed",
            documentError: scanResult.error_message,
          });
        } catch (error) {
          console.error("Document scan failed:", error);
          controller.attachments.updateFile(file.id, {
            documentStatus: "failed",
            documentError: error instanceof Error ? error.message : "Scan failed",
          });
        } finally {
          scanningRef.current.delete(file.id);
        }
      })();
    });
  }, [controller.attachments]);

  return null;
};

/**
 * Extract the media object ID (jan_*) from a media URL
 */
function extractMediaObjectId(url: string): string {
  // Try to find jan_* pattern in the URL
  const match = url.match(/jan_[a-zA-Z0-9]+/);
  if (match) {
    return match[0];
  }
  // If no jan_* found, the URL itself might be the ID
  return url;
}

const ChatInput = ({
  initialConversation = false,
  status,
  projectId,
  conversationId,
  submit,
}: {
  initialConversation?: boolean;
  projectId?: string;
  conversationId?: string;
  status?: ChatStatus;
  submit?: (message?: PromptInputMessage) => void;
}) => {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const navigate = useNavigate();
  const isPrivateChat = usePrivateChat((state) => state.isPrivateChat);

  const browserConnectionState = useBrowserConnection(
    (state) => state.connectionState,
  );
  const isMobile = useIsMobile();
  const isMobileDevice = useIsMobileDevice();
  const isChromiumBrowser = useMemo(() => isChromeBrowser(), []);

  // Auto-focus on chat input when component mounts (desktop only)
  useEffect(() => {
    if (textareaRef.current && !isMobile) {
      textareaRef.current.focus();
    }
  }, [isMobile]);

  // Auto-focus when assistant finishes responding (desktop only)
  useEffect(() => {
    // Focus when status changes from 'streaming' to idle (assistant finished)
    if (status !== CHAT_STATUS.STREAMING && textareaRef.current && !isMobile) {
      textareaRef.current.focus();
    }
  }, [status, isMobile]);

  const selectedModel = useModels((state) => state.selectedModel);
  const modelDetail = useModels((state) => state.modelDetail);

  const createConversation = useConversations(
    (state) => state.createConversation,
  );

  const { projects } = useProjects();
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(
    null,
  );

  const searchEnabled = useCapabilities((state) => state.searchEnabled);
  const deepResearchEnabled = useCapabilities(
    (state) => state.deepResearchEnabled,
  );
  const browserEnabled = useCapabilities((state) => state.browserEnabled);
  const reasoningEnabled = useCapabilities((state) => state.reasoningEnabled);
  const imageGenerationEnabled = useCapabilities(
    (state) => state.imageGenerationEnabled,
  );
  const toggleSearch = useCapabilities((state) => state.toggleSearch);
  const toggleDeepResearch = useCapabilities(
    (state) => state.toggleDeepResearch,
  );
  const toggleBrowser = useCapabilities((state) => state.toggleBrowser);
  const toggleInstruct = useCapabilities((state) => state.toggleReasoning);
  const toggleImageGeneration = useCapabilities(
    (state) => state.toggleImageGeneration,
  );
  const agentModeEnabled = useCapabilities((state) => state.agentModeEnabled);
  const agentModeAvailable = useCapabilities(
    (state) => state.agentModeAvailable,
  );
  const toggleAgentMode = useCapabilities((state) => state.toggleAgentMode);
  const setAgentModeAvailable = useCapabilities(
    (state) => state.setAgentModeAvailable,
  );
  const setAgentModeEnabled = useCapabilities(
    (state) => state.setAgentModeEnabled,
  );
  const hydrateCapabilities = useCapabilities((state) => state.hydrate);

  // Typewriter animation for placeholder
  const DEFAULT_PLACEHOLDER = "Ask Jan ...";
  const AGENT_PLACEHOLDER = "Let me take care of it ✨";
  const [placeholder, setPlaceholder] = useState(
    agentModeEnabled ? AGENT_PLACEHOLDER : DEFAULT_PLACEHOLDER
  );

  useEffect(() => {
    const targetText = agentModeEnabled ? AGENT_PLACEHOLDER : DEFAULT_PLACEHOLDER;

    let deleteInterval: ReturnType<typeof setInterval> | null = null;
    let typeInterval: ReturnType<typeof setInterval> | null = null;
    let cancelled = false;

    const startAnimation = () => {
      const currentText = placeholder;
      let index = currentText.length;

      // Delete animation
      deleteInterval = setInterval(() => {
        if (cancelled) return;
        index--;
        if (index < 0) {
          if (deleteInterval) clearInterval(deleteInterval);
          // Type animation
          let typeIndex = 0;
          typeInterval = setInterval(() => {
            if (cancelled) return;
            typeIndex++;
            setPlaceholder(targetText.slice(0, typeIndex));
            if (typeIndex >= targetText.length) {
              if (typeInterval) clearInterval(typeInterval);
            }
          }, 30);
        } else {
          setPlaceholder(currentText.slice(0, index));
        }
      }, 20);
    };

    if (placeholder !== targetText) {
      startAnimation();
    }

    return () => {
      cancelled = true;
      if (deleteInterval) clearInterval(deleteInterval);
      if (typeInterval) clearInterval(typeInterval);
      // Immediately set to target when cancelled (rapid toggle)
      setPlaceholder(targetText);
    };
  }, [agentModeEnabled]);

  const setSearchEnabled = useCapabilities((state) => state.setSearchEnabled);
  const setDeepResearchEnabled = useCapabilities(
    (state) => state.setDeepResearchEnabled,
  );
  const setBrowserEnabled = useCapabilities((state) => state.setBrowserEnabled);
  const setReasoningEnabled = useCapabilities(
    (state) => state.setReasoningEnabled,
  );
  const setImageGenerationEnabled = useCapabilities(
    (state) => state.setImageGenerationEnabled,
  );

  const fetchPreferences = useProfile((state) => state.fetchPreferences);
  const fetchSettings = useProfile((state) => state.fetchSettings);
  const pref = useProfile((state) => state.preferences);
  const settings = useProfile((state) => state.settings);

  const isSupportTools = modelDetail.supports_tools;
  const isSupportReasoning = modelDetail.supports_reasoning;
  const isSupportDeepResearch = isSupportTools && isSupportReasoning;
  const isSupportInstruct = modelDetail.supports_instruct;
  const isBrowserSupported = isChromiumBrowser && modelDetail.supports_browser;
  const shouldShowBrowserUI = isBrowserSupported && !isMobileDevice;
  const isSupportImageGeneration =
    settings?.server_capabilities?.image_generation_enabled ?? false;
  const isSupportImages = modelDetail.supports_images;

  // Auto-disable capabilities when model doesn't support them
  useEffect(() => {
    // Only run if we have a valid model loaded
    if (!modelDetail.id) return;

    if (!isSupportTools && searchEnabled) {
      setSearchEnabled(false);
    }
  }, [isSupportTools, searchEnabled, setSearchEnabled, modelDetail.id]);

  useEffect(() => {
    // Only run if we have a valid model loaded
    if (!modelDetail.id) return;

    if (!isSupportDeepResearch && deepResearchEnabled) {
      setDeepResearchEnabled(false);
    }
  }, [
    isSupportDeepResearch,
    deepResearchEnabled,
    setDeepResearchEnabled,
    modelDetail.id,
  ]);

  // Auto-disable browser capability when disconnected
  useEffect(() => {
    if (
      browserConnectionState === CONNECTION_STATE.DISCONNECTED &&
      browserEnabled
    ) {
      setBrowserEnabled(false);
    }
  }, [browserConnectionState, browserEnabled, setBrowserEnabled]);

  // Disable browser capability on unsupported environments
  useEffect(() => {
    if (!shouldShowBrowserUI && browserEnabled) {
      setBrowserEnabled(false);
    }
  }, [shouldShowBrowserUI, browserEnabled, setBrowserEnabled]);

  useEffect(() => {
    fetchPreferences();
    fetchSettings();
  }, []);

  useEffect(() => {
    let isActive = true;

    const fetchAgentAvailability = async () => {
      try {
        const toolsResponse = await mcpService.getTools();
        if (!isActive) {
          return;
        }
        const hasAgentTool = toolsResponse.data.some(
          (tool) => tool.name === "run_agent",
        );
        setAgentModeAvailable(hasAgentTool);
      } catch (error) {
        console.error(
          "Failed to fetch MCP tools for agent availability:",
          error,
        );
        if (!isActive) {
          return;
        }
        setAgentModeAvailable(false);
      }
    };

    fetchAgentAvailability();

    return () => {
      isActive = false;
    };
  }, [setAgentModeAvailable]);

  useEffect(() => {
    if (pref) {
      hydrateCapabilities(pref.preferences);
    }
  }, [pref, hydrateCapabilities]);

  useEffect(() => {
    if (!agentModeAvailable && agentModeEnabled) {
      setAgentModeEnabled(false);
    }
  }, [agentModeAvailable, agentModeEnabled, setAgentModeEnabled]);

  // Auto-disable image generation when server doesn't support it
  useEffect(() => {
    if (!isSupportImageGeneration && imageGenerationEnabled) {
      setImageGenerationEnabled(false);
    }
  }, [
    isSupportImageGeneration,
    imageGenerationEnabled,
    setImageGenerationEnabled,
  ]);

  const handleError = (err: {
    code: "max_files" | "max_file_size" | "accept" | "max_images";
    message: string;
  }) => {
    toast.error(err.message);
  };

  // Upload service configuration
  const uploadService: UploadServiceConfig = useMemo(
    () => ({
      blobUrlToDataUrl,
      uploadMedia,
      createJanMediaUrl,
    }),
    [],
  );

  const handleSubmit = (message: PromptInputMessage) => {
    const trimmedText = message.text.trim();
    const hasText = Boolean(trimmedText);
    const hasAttachments = Boolean(message.files?.length);

    if (!(hasText || hasAttachments)) {
      submit?.();
      return;
    }

    const normalizedMessage: PromptInputMessage = {
      ...message,
      text: trimmedText,
    };

    if (!selectedModel) {
      toast.warning("Please select a model to start chatting.");
      return;
    }

    if (initialConversation) {
      if (isPrivateChat) {
        sessionStorage.setItem(
          SESSION_STORAGE_KEY.INITIAL_MESSAGE_TEMPORARY,
          JSON.stringify(normalizedMessage),
        );
        navigate({
          to: "/threads/temporary",
        });

        return;
      }

      const conversationPayload: CreateConversationPayload = {
        title: generateThreadTitle(normalizedMessage.text),
        ...(projectId && { project_id: String(projectId) }),
        ...(selectedProjectId && { project_id: selectedProjectId }),
        metadata: {
          model_id: selectedModel.id,
          model_provider: selectedModel.owned_by,
          is_favorite: "false",
        },
      };

      createConversation(conversationPayload)
        .then((conversation) => {
          // Store the initial message in sessionStorage for the new conversation
          sessionStorage.setItem(
            `${SESSION_STORAGE_PREFIX.INITIAL_MESSAGE}${conversation.id}`,
            JSON.stringify(normalizedMessage),
          );

          // Clear selected project after creating conversation
          setSelectedProjectId(null);

          // Redirect to the conversation detail page
          navigate({
            to: "/threads/$conversationId",
            params: { conversationId: conversation.id },
          });

          return;
        })
        .catch((error) => {
          console.error("Failed to create initial conversation:", error);
        });
    } else {
      submit?.(normalizedMessage);
    }
  };

  return (
    <div>
      <div
        className={cn(
          "w-full relative rounded-3xl p-[1.5px]",
          !initialConversation &&
            (status === CHAT_STATUS.STREAMING ||
              status === CHAT_STATUS.SUBMITTED) &&
            "overflow-hidden outline-0",
        )}
      >
        <PromptInputProvider
          maxImages={MAX_IMAGES}
          maxFileSize={MAX_FILE_SIZE}
          accept={ACCEPTED_FILE_TYPES}
          onError={handleError}
          uploadService={uploadService}
          userId={conversationId || projectId || "anonymous"}
        >
          <InputResetHandler
            textareaRef={textareaRef}
            isMobile={isMobile}
            conversationId={conversationId}
          />
          <DocumentScanHandler />
          <PromptInput
            accept={ACCEPTED_FILE_TYPES}
            globalDrop
            multiple
            userId={conversationId || projectId || "anonymous"}
            onError={handleError}
            onSubmit={handleSubmit}
            className="rounded-3xl relative z-40 bg-background"
          >
            <PromptInputAttachments>
              {(attachment) => <PromptInputAttachment data={attachment} />}
            </PromptInputAttachments>
            <PromptInputBody>
              <PromptInputTextarea
                ref={textareaRef}
                placeholder={placeholder}
                disabled={
                  status === CHAT_STATUS.STREAMING ||
                  status === CHAT_STATUS.SUBMITTED
                }
              />
            </PromptInputBody>
            <PromptInputFooter>
              <PromptInputTools>
                {(isSupportImages ||
                  (initialConversation && !projectId && !isPrivateChat)) && (
                  <PromptInputActionMenu>
                    <PromptInputActionMenuTrigger
                      className="rounded-full"
                      variant="secondary"
                    />
                    <PromptInputActionMenuContent className="lg:w-56">
                        {isSupportImages && (
                      <PromptInputActionAddImages label="Add images" />
                      )}
                      <PromptInputActionAddFiles label="Add files" />
                      {initialConversation && !projectId && !isPrivateChat && (
                        <ProjectsChatInput
                          currentProjectId={selectedProjectId || undefined}
                          onProjectSelect={(projectId) => {
                            setSelectedProjectId(projectId);
                          }}
                        />
                      )}
                    </PromptInputActionMenuContent>
                  </PromptInputActionMenu>
                )}

                <SettingChatInput
                  searchEnabled={searchEnabled}
                  deepResearchEnabled={deepResearchEnabled}
                  browserEnabled={browserEnabled}
                  reasoningEnabled={reasoningEnabled}
                  imageGenerationEnabled={imageGenerationEnabled}
                  disablePreferences={status === CHAT_STATUS.STREAMING}
                  toggleSearch={() => {
                    toggleSearch();
                    setBrowserEnabled(false);
                    setImageGenerationEnabled(false);
                  }}
                  toggleDeepResearch={() => {
                    toggleDeepResearch();
                    setBrowserEnabled(false);
                    setImageGenerationEnabled(false);
                  }}
                  toggleBrowser={() => {
                    toggleBrowser();
                    setDeepResearchEnabled(false);
                    setSearchEnabled(false);
                    setImageGenerationEnabled(false);
                  }}
                  toggleImageGeneration={() => {
                    toggleImageGeneration();
                    // Disable all other capabilities when image generation is enabled
                    if (!imageGenerationEnabled) {
                      setSearchEnabled(false);
                      setDeepResearchEnabled(false);
                      setBrowserEnabled(false);
                      setReasoningEnabled(false);
                    }
                  }}
                  isBrowserSupported={isBrowserSupported}
                  toggleInstruct={() => {
                    toggleInstruct();
                    setImageGenerationEnabled(false);
                  }}
                  isSupportTools={isSupportTools}
                  isSupportDeepResearch={isSupportDeepResearch}
                  isSupportReasoningToggle={isSupportInstruct}
                  isSupportImageGeneration={isSupportImageGeneration}
                >
                  <Button
                    className="rounded-full mx-1 size-8"
                    variant="secondary"
                    size="icon"
                  >
                    <Settings2 className="size-4 text-muted-foreground" />
                  </Button>
                </SettingChatInput>
                {!imageGenerationEnabled && agentModeAvailable && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="secondary"
                        className={cn(
                          "rounded-full size-8 transition-all duration-300 ease-in-out overflow-hidden group",
                          agentModeEnabled && "w-22 px-3 gap-1.5"
                        )}
                        disabled={status === CHAT_STATUS.STREAMING}
                        onClick={toggleAgentMode}
                      >
                        <BotIcon
                          className={cn(
                            "size-4 shrink-0 text-muted-foreground ml-2",
                            agentModeEnabled && "text-primary group-hover:hidden ml-0"
                          )}
                        />
                        {agentModeEnabled && (
                          <X className="size-4 shrink-0 text-primary hidden group-hover:block" />
                        )}
                        <span
                          className={cn(
                            "text-sm font-medium transition-all duration-300 ease-in-out whitespace-nowrap overflow-hidden",
                            agentModeEnabled
                              ? "opacity-100 max-w-12.5 text-primary"
                              : "opacity-0 max-w-0"
                          )}
                        >
                          Agent
                        </span>
                      </Button>
                    </TooltipTrigger>
                    {!agentModeEnabled && (
                      <TooltipContent>Agent Mode</TooltipContent>
                    )}
                  </Tooltip>
                )}
                {isSupportInstruct &&
                  reasoningEnabled &&
                  !deepResearchEnabled &&
                  !imageGenerationEnabled &&
                  !agentModeEnabled && (
                    <PromptInputButton
                      variant="outline"
                      className="rounded-full group transition-all bg-primary/10 hover:bg-primary/10 border-0"
                      disabled={status === CHAT_STATUS.STREAMING}
                      onClick={() => {
                        toggleInstruct();
                        setImageGenerationEnabled(false);
                      }}
                    >
                      <LightbulbIcon className="text-primary size-4 group-hover:hidden" />
                      <X className="text-primary size-4 hidden group-hover:block" />
                      <span className="text-primary">Think</span>
                    </PromptInputButton>
                  )}
                {searchEnabled &&
                  !deepResearchEnabled &&
                  !imageGenerationEnabled &&
                  !agentModeEnabled && (
                    <PromptInputButton
                      variant="outline"
                      className="rounded-full group transition-all bg-primary/10 hover:bg-primary/10 border-0"
                      disabled={status === CHAT_STATUS.STREAMING}
                      onClick={toggleSearch}
                    >
                      <GlobeIcon className="text-primary size-4 group-hover:hidden" />
                      <X className="text-primary size-4 hidden group-hover:block" />
                      <span className="text-primary">Search</span>
                    </PromptInputButton>
                  )}
                {deepResearchEnabled && !imageGenerationEnabled && !agentModeEnabled && (
                  <PromptInputButton
                    variant="outline"
                    className="rounded-full group transition-all bg-primary/10 hover:bg-primary/10 border-0"
                    disabled={status === CHAT_STATUS.STREAMING}
                    onClick={toggleDeepResearch}
                  >
                    <TelescopeIcon className="text-primary size-4 group-hover:hidden" />
                    <X className="text-primary size-4 hidden group-hover:block" />
                    <span className="text-primary">Deep Research</span>
                  </PromptInputButton>
                )}
                {browserEnabled &&
                  shouldShowBrowserUI &&
                  !imageGenerationEnabled &&
                  !agentModeEnabled && (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <PromptInputButton
                          variant="outline"
                          className="rounded-full group transition-all bg-primary/10 hover:bg-primary/10 border-0"
                          disabled={status === CHAT_STATUS.STREAMING}
                          onClick={toggleBrowser}
                        >
                          <div className="size-4 flex items-center justify-center group-hover:hidden">
                            {browserConnectionState ===
                              CONNECTION_STATE.ERROR && (
                              <div className="size-3 bg-red-400 rounded-full" />
                            )}
                            {browserConnectionState ===
                              CONNECTION_STATE.CONNECTING && (
                              <div className="size-3 animate-pulse bg-blue-400 rounded-full" />
                            )}
                            {browserConnectionState ===
                              CONNECTION_STATE.CONNECTED && (
                              <div className="size-3 bg-green-400 rounded-full" />
                            )}
                          </div>
                          <X className="text-primary size-4 hidden group-hover:block" />
                          <span className="text-primary">Browse</span>
                        </PromptInputButton>
                      </TooltipTrigger>
                      <TooltipContent>
                        {browserConnectionState === CONNECTION_STATE.ERROR && (
                          <p>Connection error</p>
                        )}
                        {browserConnectionState ===
                          CONNECTION_STATE.CONNECTING && <p>Connecting...</p>}
                        {browserConnectionState ===
                          CONNECTION_STATE.CONNECTED && <p>Ready to use</p>}
                      </TooltipContent>
                    </Tooltip>
                  )}
                {selectedProjectId &&
                  !isPrivateChat &&
                  !imageGenerationEnabled &&
                  !agentModeEnabled && (
                    <PromptInputButton
                      variant="outline"
                      className="rounded-full group transition-all bg-primary/10 hover:bg-primary/10 border-0"
                      disabled={status === CHAT_STATUS.STREAMING}
                      onClick={() => setSelectedProjectId(null)}
                    >
                      <FolderIcon className="text-primary size-4 group-hover:hidden" />
                      <X className="text-primary size-4 hidden group-hover:block" />
                      <span className="text-primary">
                        {projects.find((p) => p.id === selectedProjectId)
                          ?.name || "Project"}
                      </span>
                    </PromptInputButton>
                  )}
                {isSupportImageGeneration && imageGenerationEnabled && (
                  <PromptInputButton
                    variant="outline"
                    className="rounded-full group transition-all bg-primary/10 hover:bg-primary/10 border-0"
                    disabled={status === CHAT_STATUS.STREAMING}
                    onClick={() => {
                      toggleImageGeneration();
                      // Re-enable other capabilities when image generation is toggled off
                    }}
                  >
                    <ImageIcon className="text-primary size-4 group-hover:hidden" />
                    <X className="text-primary size-4 hidden group-hover:block" />
                    <span className="text-primary">Create Images</span>
                  </PromptInputButton>
                )}
              </PromptInputTools>
              <div className="flex items-center gap-2">
                {/* Disabled speech input button */}
                {/* <PromptInputSpeechButton textareaRef={textareaRef} /> */}
                <PromptInputSubmit
                  status={status}
                  className="rounded-full"
                  variant={
                    status === CHAT_STATUS.STREAMING ||
                    status === CHAT_STATUS.SUBMITTED
                      ? "destructive"
                      : "default"
                  }
                />
              </div>
            </PromptInputFooter>
          </PromptInput>
          {(status === CHAT_STATUS.STREAMING ||
            status === CHAT_STATUS.SUBMITTED) && (
            <div className="absolute inset-0 ">
              <BorderAnimate rx="10%" ry="10%">
                <div
                  className={cn(
                    "h-100 w-100 bg-[radial-gradient(var(--primary),transparent_50%)]",
                  )}
                />
              </BorderAnimate>
            </div>
          )}
        </PromptInputProvider>
        {status !== CHAT_STATUS.STREAMING &&
          agentModeEnabled &&
          agentModeAvailable && (
        <div className="absolute inset-0 scale-90 opacity-50 dark:opacity-25 blur-xl transition-all duration-100">
          <div className="bg-linear-to-r/increasing animate-hue-rotate absolute inset-x-0 bottom-0 top-6 from-pink-300 to-purple-300" />
        </div>
      )}
      </div>
      {conversationId && (
        <div className="mt-2 text-xs text-muted-foreground text-center">
          <p>Jan can make mistakes. Please double check responses.</p>
        </div>
      )}
    </div>
  );
};

export default memo(ChatInput);
