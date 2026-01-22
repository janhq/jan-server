/* eslint-disable react-hooks/exhaustive-deps */
/* eslint-disable @typescript-eslint/no-explicit-any */
import ChatInput from "@/components/chat-input";
import { AppSidebar } from "@/components/sidebar/app-sidebar";
import { AppSidebarRight } from "@/components/sidebar/app-sidebar-right";
import { SidebarInset } from "@/components/sidebar/sidebar";
import { NavHeader } from "@/components/sidebar/nav-header";
import { useChat } from "@/hooks/use-chat";
import { janProvider } from "@/lib/api-client";
import {
  Conversation,
  ConversationContent,
  ConversationScrollButton,
} from "@janhq/interfaces/ai-elements/conversation";
import type { PromptInputMessage } from "@janhq/interfaces/ai-elements/prompt-input";
import { Loader, AlertTriangleIcon } from "lucide-react";
import { useModels } from "@/stores/models-store";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useConversations } from "@/stores/conversation-store";
import { mcpService } from "@/services/mcp-service";
import { useCapabilities } from "@/stores/capabilities-store";
import { useAgentExecutionStore } from "@/stores/agent-execution-store";
import { lastAssistantMessageIsCompleteWithToolCalls } from "ai";
import type { UIDataTypes, UIMessage, UITools } from "ai";
import { MessageItem } from "./message-item";
import {
  findPrecedingUserMessageIndex,
  findPrecedingAssistantMessageIndex,
  buildIdMapping,
  resolveMessageId,
  buildMessageContent,
} from "@/lib/message-utils";
import { convertToUIMessages } from "@/lib/utils";
import { useChatSessions } from "@/stores/chat-session-store";
import {
  PRIVATE_CHAT_SESSION_ID,
  TEMPORARY_CHAT_SESSION_ID,
  SCROLL_ANIMATION,
  MCP,
  TOOL_STATE,
  CHAT_STATUS,
  CONTENT_TYPE,
  SESSION_STORAGE_KEY,
  SESSION_STORAGE_PREFIX,
  MESSAGE_ROLE,
} from "@/constants";
import { ApiError } from "@/lib/api-client";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "@janhq/interfaces/sonner";
import { analytics } from "@/lib/analytics";
import { useAuth } from "@/stores/auth-store";

interface ThreadPageContentProps {
  conversationId?: string;
  isPrivateChat?: boolean;
}

export function ThreadPageContent({
  conversationId,
  isPrivateChat = false,
}: ThreadPageContentProps) {
  const selectedModel = useModels((state) => state.selectedModel);
  const models = useModels((state) => state.models);
  const setSelectedModel = useModels((state) => state.setSelectedModel);
  const getConversation = useConversations((state) => state.getConversation);
  const isAuthenticated = useAuth((state) => state.isAuthenticated);
  const initialMessageSentRef = useRef(false);
  const reasoningContainerRef = useRef<HTMLDivElement>(null);
  const messageStartTimeRef = useRef<number | null>(null);
  const deepResearchEnabled = useCapabilities(
    (state) => state.deepResearchEnabled,
  );
  const searchEnabled = useCapabilities((state) => state.searchEnabled);
  const agentModeEnabled = useCapabilities((state) => state.agentModeEnabled);
  const agentModeAvailable = useCapabilities(
    (state) => state.agentModeAvailable,
  );
  const enableThinking = useCapabilities((state) => state.reasoningEnabled);
  const imageGenerationEnabled = useCapabilities(
    (state) => state.imageGenerationEnabled,
  );
  const agentModeActive = agentModeEnabled && agentModeAvailable;

  const getCurrentMode = useCallback(() => {
    if (agentModeActive) return "agent";
    if (deepResearchEnabled) return "deep_research";
    if (searchEnabled) return "search";
    if (enableThinking) return "reasoning";
    if (imageGenerationEnabled) return "create_image";
    return "normal";
  }, [
    agentModeActive,
    deepResearchEnabled,
    searchEnabled,
    enableThinking,
    imageGenerationEnabled,
  ]);
  const [conversationTitle, setConversationTitle] = useState<string>("");
  const navigate = useNavigate();
  const hasRedirectedRef = useRef(false);

  const handleConversationNotFound = useCallback(() => {
    if (hasRedirectedRef.current) return;
    hasRedirectedRef.current = true;
    toast.error("Conversation not found.");
    navigate({ to: "/" });
  }, [navigate]);

  const provider = useMemo(
    () =>
      janProvider(
        conversationId,
        deepResearchEnabled,
        isPrivateChat,
        enableThinking,
        imageGenerationEnabled,
        agentModeActive,
      ),
    [
      conversationId,
      deepResearchEnabled,
      isPrivateChat,
      enableThinking,
      imageGenerationEnabled,
      agentModeActive,
    ],
  );

  const getUIMessages = useConversations((state) => state.getUIMessages);
  const loadMoreItems = useConversations((state) => state.loadMoreItems);
  const itemsCursor = useConversations((state) => state.itemsCursor);
  const createItems = useConversations((state) => state.createItems);
  const moveConversationToTop = useConversations(
    (state) => state.moveConversationToTop,
  );
  const [loadingMoreItems, setLoadingMoreItems] = useState(false);
  const fetchingMessagesRef = useRef(false);
  const getSessionData = useChatSessions((state) => state.getSessionData);

  const chatSessionId =
    conversationId ??
    (isPrivateChat ? PRIVATE_CHAT_SESSION_ID : TEMPORARY_CHAT_SESSION_ID);
  // sessionData is a mutable ref-like object - direct mutations don't trigger re-renders (intentional)
  const sessionData = getSessionData(chatSessionId);

  // AbortController for cancelling tool calls (kept as ref since it's a signal, not session data)
  const toolCallAbortController = useRef<AbortController | null>(null);

  // Helper to get current messages for this session
  const getCurrentMessages = useCallback(() => {
    const state = useChatSessions.getState();
    const session = state.sessions[chatSessionId];
    return (
      session?.chat.messages ?? state.getSessionData(chatSessionId).messages
    );
  }, [chatSessionId]);

  // Persist user message to server and map temp ID to backend ID
  const createUserMessageItem = useCallback(
    async (message: PromptInputMessage): Promise<string | null> => {
      if (!conversationId || isPrivateChat) return null;

      const content = buildMessageContent(message);
      if (content.length === 0) return null;

      // Capture index before async operation - message will be added here after sendMessage
      const messageIndex = getCurrentMessages().length;

      // Skip if message at this index already has a backend ID or mapping
      const existingMsg = getCurrentMessages()[messageIndex];
      if (existingMsg) {
        if (
          existingMsg.id.startsWith("msg_") ||
          sessionData.idMap.has(existingMsg.id)
        ) {
          return sessionData.idMap.get(existingMsg.id) ?? existingMsg.id;
        }
      }

      try {
        const response = await createItems(conversationId, [
          {
            role: MESSAGE_ROLE.USER,
            type: "message",
            content,
          },
        ]);
        const backendId = response.data?.[0]?.id ?? null;

        // Map temp ID to backend ID after sendMessage adds the message
        if (backendId) {
          setTimeout(() => {
            const msg = getCurrentMessages()[messageIndex];
            if (msg) {
              sessionData.idMap.set(msg.id, backendId);
            }
          }, 0);
        }

        return backendId;
      } catch (error) {
        console.error("Failed to create user message item:", error);
        return null;
      }
    },
    [
      conversationId,
      isPrivateChat,
      createItems,
      getCurrentMessages,
      sessionData,
    ],
  );

  // Check if we should follow up with tool calls (respects abort signal)
  const followUpMessage = ({
    messages,
  }: {
    messages: UIMessage<unknown, UIDataTypes, UITools>[];
  }) => {
    if (
      !toolCallAbortController.current ||
      toolCallAbortController.current?.signal.aborted
    ) {
      return false;
    }
    return lastAssistantMessageIsCompleteWithToolCalls({ messages });
  };

  const {
    messages,
    status,
    sendMessage,
    regenerate,
    setMessages,
    error,
    addToolOutput,
    stop,
  } = useChat(provider(selectedModel?.id), {
    experimental_throttle: 50,
    sessionId: chatSessionId,
    sessionTitle: conversationTitle || undefined,
    onFinish: ({ message, isAbort }) => {
      // Note: These values are captured at Chat creation time, which is correct
      // because onFinish fires for the Chat that started the stream, not the current conversation
      initialMessageSentRef.current = false;
      const hadToolCalls = sessionData.tools.length > 0;

      // Create a new AbortController for tool calls
      toolCallAbortController.current = new AbortController();
      const signal = toolCallAbortController.current.signal;

      // Check whether this is a valid message otherwise continue
      const needFollowUp =
        !hadToolCalls &&
        !isAbort &&
        message?.parts.some((e) => e.type === CONTENT_TYPE.REASONING) &&
        !message?.parts.some(
          (e) => e.type === CONTENT_TYPE.TEXT && e.text.length > 0,
        );

      let ranAgent = false;
      Promise.all(
        sessionData.tools.map(async (toolCall: any) => {
          // Check if already aborted before starting
          if (signal.aborted) {
            return;
          }

          // Prepare arguments - inject model for agent tools if not already specified
          let toolArguments = toolCall.input as Record<string, unknown>;
          if (toolCall.toolName === "run_agent") {
            ranAgent = true; // Mark that we ran an agent - always set regardless of model injection
            if (!toolArguments.model && selectedModel?.id) {
              toolArguments = {
                ...toolArguments,
                model: selectedModel.id,
              };
            }
          }

          // Check if already aborted before making the call
          if (signal.aborted) {
            return;
          }

          const result = await mcpService.callTool(
            {
              toolName: toolCall.toolName,
              serverName: MCP.SERVER_NAME,
              arguments: toolArguments,
            },
            {
              conversationId,
              toolCallId: toolCall.toolCallId,
              signal, // Pass abort signal to tool call
            },
          );

          if (result.error) {
            addToolOutput({
              state: TOOL_STATE.OUTPUT_ERROR,
              tool: toolCall.toolName,
              toolCallId: toolCall.toolCallId,
              errorText: `Error: ${result.error}`,
            });
          } else {
            if (toolCall.toolName === "run_agent") {
              toolCallAbortController.current?.abort();

              try {
                const content = result.content;
                if (Array.isArray(content) && content.length > 0) {
                  const textContent = content.find((c) => c.type === "text");
                  if (textContent && textContent.text) {
                    const agentResult = JSON.parse(textContent.text);
                    const responseId = agentResult.id || agentResult.response_id;
                    if (responseId) {
                      const inProgressOutput = [{ type: "text", text: JSON.stringify({
                        status: "in_progress",
                        response_id: responseId,
                        message: "Deep research started"
                      })}];

                      addToolOutput({
                        tool: toolCall.toolName,
                        toolCallId: toolCall.toolCallId,
                        output: inProgressOutput,
                      });

                      // Persist tool result to backend so it's available on reload
                      if (conversationId && !isPrivateChat) {
                        createItems(conversationId, [
                          {
                            type: "message",
                            role: MESSAGE_ROLE.TOOL,
                            content: [{
                              type: "tool_result",
                              tool_call_id: toolCall.toolCallId,
                              tool_result: JSON.stringify({
                                status: "in_progress",
                                response_id: responseId,
                                message: "Deep research started"
                              }),
                            // eslint-disable-next-line @typescript-eslint/no-explicit-any
                            } as any],
                          },
                        ]).catch((err) => {
                          console.error("Failed to persist agent tool result:", err);
                        });
                      }

                      startExecution(responseId, (execution) => {
                        let finalReport = "";
                        if (execution.status === "completed" && execution.planDetails) {
                          const reportTask = execution.planDetails.tasks.find(
                            (t) => t.title === "Report" || t.task_type === "generation"
                          );
                          if (reportTask?.steps) {
                            const llmStep = reportTask.steps.find((s) => s.action === "llm_call");
                            if (llmStep?.output_data) {
                              const outputData = llmStep.output_data as Record<string, unknown>;
                              finalReport = (outputData.content as string) || "";
                            }
                          }
                        }

                        const agentOutput = finalReport
                          ? [{ type: "text", text: JSON.stringify({
                              status: execution.status,
                              response_id: responseId,
                              report: finalReport,
                              message: "Deep research completed. Here is the comprehensive analysis:"
                            })}]
                          : [{ type: "text", text: JSON.stringify({
                              status: execution.status,
                              response_id: responseId,
                              error: execution.error || "Agent execution failed or was cancelled"
                            })}];

                        addToolOutput({
                          tool: toolCall.toolName,
                          toolCallId: toolCall.toolCallId,
                          output: agentOutput,
                        });

                        // Don't trigger another LLM call after agent completion
                        // The agent's output (with download URLs) is already in the tool result
                        // Calling sendMessage() here causes the model to generate
                        // unnecessary follow-up messages after the plan is completed
                      });

                      return;
                    }
                  }
                }
              } catch (parseError) {
                console.error("Failed to parse run_agent result:", parseError);
              }

              // If we couldn't parse the response, add the original output
              addToolOutput({
                tool: toolCall.toolName,
                toolCallId: toolCall.toolCallId,
                output: result.content,
              });
            } else {
              addToolOutput({
                tool: toolCall.toolName,
                toolCallId: toolCall.toolCallId,
                output: result.content,
              });
            }
          }
        }),
      )
        .then(() => {
          if (ranAgent) {
            return;
          }

          if (needFollowUp) {
            sendMessage();
          } else if (conversationId && !isPrivateChat && !hadToolCalls) {
            // Build ID mapping without updating state to avoid scroll jump
            getUIMessages(conversationId)
              .then((result) => {
                buildIdMapping(
                  getCurrentMessages(),
                  result.messages,
                  sessionData.idMap,
                );
              })
              .catch(console.error);
          }
        })
        .catch((error) => {
          // Ignore abort errors
          if (error.name !== "AbortError") {
            console.error("Tool call error:", error);
          }
        })
        .finally(() => {
          sessionData.tools = [];
          toolCallAbortController.current = null;
        });
    },
    sendAutomaticallyWhen: followUpMessage,
    onToolCall: ({ toolCall }) => {
      sessionData.tools.push(toolCall);
      return;
    },
  });

  // Keep ref in sync for use in onFinish closure
  useEffect(() => {
    sessionData.messages = messages;
  }, [messages, sessionData]);

  const regenerateMessage = useConversations(
    (state) => state.regenerateMessage,
  );

  const executeRegenerate = useCallback(
    async (realId: string, userIndex: number) => {
      if (!conversationId) return;

      const response = await regenerateMessage(conversationId, realId);

      if (response.branch_created && response.branch) {
        const currentMessages = getCurrentMessages();
        const truncatedMessages = currentMessages.slice(0, userIndex + 1);
        setMessages(truncatedMessages);

        setTimeout(() => regenerate(), 0);

        getUIMessages(conversationId, response.branch)
          .then((result) => {
            buildIdMapping(
              truncatedMessages,
              result.messages,
              sessionData.idMap,
            );
          })
          .catch(console.error);
      }
    },
    [
      conversationId,
      regenerateMessage,
      getCurrentMessages,
      setMessages,
      regenerate,
      getUIMessages,
      sessionData.idMap,
    ],
  );

  const handleRegenerateUserMessage = useCallback(
    async (messageId: string, messageIndex: number) => {
      const currentMessages = getCurrentMessages();
      const realId = resolveMessageId(messageId, sessionData.idMap);
      const hasMappedId = realId !== messageId;

      // Case 1: Has mapped backend ID - use it directly
      if (hasMappedId) {
        await executeRegenerate(realId, messageIndex);
        return;
      }

      // Case 2: No mapped ID - check if messageId IS a backend ID (from reload)
      if (messageId.startsWith("msg_")) {
        await executeRegenerate(messageId, messageIndex);
        return;
      }

      // Case 3: Temp ID with no mapping - try previous assistant message
      const assistantIndex = findPrecedingAssistantMessageIndex(
        currentMessages,
        messageIndex,
      );
      if (assistantIndex !== -1) {
        const assistantId = currentMessages[assistantIndex].id;
        const assistantRealId = resolveMessageId(
          assistantId,
          sessionData.idMap,
        );
        const assistantHasBackendId =
          assistantRealId !== assistantId || assistantId.startsWith("msg_");

        if (assistantHasBackendId) {
          const assistantBackendId =
            assistantRealId !== assistantId ? assistantRealId : assistantId;
          await executeRegenerate(assistantBackendId, messageIndex);
          return;
        }
      }

      // Case 4: No backend IDs anywhere
      // If user message is first message - regenerate locally
      if (messageIndex === 0) {
        const truncatedMessages = currentMessages.slice(0, messageIndex + 1);
        setMessages(truncatedMessages);
        setTimeout(() => regenerate(), 0);
        return;
      }

      // Case 5: Not first message but no backend IDs - abort (user might need to reload)
    },
    [
      getCurrentMessages,
      sessionData.idMap,
      executeRegenerate,
      setMessages,
      regenerate,
    ],
  );

  const handleRegenerateAssistantMessage = useCallback(
    async (messageId: string, messageIndex: number) => {
      const currentMessages = getCurrentMessages();
      const realId = resolveMessageId(messageId, sessionData.idMap);
      const userIndex = findPrecedingUserMessageIndex(
        currentMessages,
        messageIndex,
      );

      if (userIndex === -1) {
        // No user message found -> cant regenerate -> abort
        return;
      }

      // Check if assistant message has a valid backend ID
      const hasBackendId = realId !== messageId || messageId.startsWith("msg_");

      if (hasBackendId) {
        // Use assistant's backend ID
        await executeRegenerate(realId, userIndex);
        return;
      }

      // No backend ID for assistant (e.g., stopped mid-stream)
      // Try using the preceding user message's backend ID
      const userMessage = currentMessages[userIndex];
      const userRealId = resolveMessageId(userMessage.id, sessionData.idMap);
      const userHasBackendId =
        userRealId !== userMessage.id || userMessage.id.startsWith("msg_");

      if (userHasBackendId) {
        const userBackendId =
          userRealId !== userMessage.id ? userRealId : userMessage.id;
        await executeRegenerate(userBackendId, userIndex);
      }

      // No backend IDs - abort (cannot regenerate without server state)
    },
    [getCurrentMessages, sessionData.idMap, executeRegenerate],
  );

  const handleRegenerate = useCallback(
    async (messageId: string) => {
      if (!conversationId) return;
      try {
        const currentMessages = getCurrentMessages();
        const messageIndex = currentMessages.findIndex(
          (m) => m.id === messageId,
        );
        if (messageIndex === -1) return;

        const message = currentMessages[messageIndex];

        if (message.role === MESSAGE_ROLE.USER) {
          await handleRegenerateUserMessage(messageId, messageIndex);
        } else if (message.role === MESSAGE_ROLE.ASSISTANT) {
          await handleRegenerateAssistantMessage(messageId, messageIndex);
        }
      } catch (error) {
        console.error("Failed to regenerate:", error);
      }
    },
    [
      conversationId,
      getCurrentMessages,
      handleRegenerateUserMessage,
      handleRegenerateAssistantMessage,
    ],
  );

  const appendImageUrlsToPrompt = useCallback(
    (message: PromptInputMessage, enabled: boolean): PromptInputMessage => {
      if (!enabled) return message;
      const urls = message.files
        .map((file) => file.url)
        .filter(
          (url): url is string => typeof url === "string" && url.trim() !== "",
        );
      if (urls.length === 0) return message;

      const baseText = (message.text || "").trim();
      const wrappedUrls = urls
        .map((url) => `<attached_url>${url}</attached_url>`)
        .join("\n");
      const appended = baseText ? `${baseText}\n\n${wrappedUrls}` : wrappedUrls;

      return {
        ...message,
        text: appended,
      };
    },
    [],
  );

  const handleSubmit = useCallback(
    async (message?: PromptInputMessage) => {
      // Get the current session to check its status directly
      const currentSession = useChatSessions.getState().sessions[chatSessionId];
      const currentStatus = currentSession?.status ?? status;

      if (
        message &&
        currentStatus !== CHAT_STATUS.STREAMING &&
        currentStatus !== CHAT_STATUS.SUBMITTED
      ) {
        sessionData.tools = [];

        const withImageUrls = appendImageUrlsToPrompt(
          message,
          imageGenerationEnabled,
        );

        messageStartTimeRef.current = Date.now();

        analytics.capture("message_sent", {
          conversation_id: conversationId || null,
          model: selectedModel?.id || null,
          mode: getCurrentMode(),
          has_attachments: (withImageUrls.files?.length || 0) > 0,
          attachment_count: withImageUrls.files?.length || 0,
          message_length: withImageUrls.text?.length || 0,
          user_status: analytics.getUserStatus(isAuthenticated),
        });

        // Normal message flow

        // Persist to server (fire-and-forget, ID mapping handled in onFinish)
        createUserMessageItem(withImageUrls);

        sendMessage({
          text: withImageUrls.text || "Sent with attachments",
          files: withImageUrls.files,
        });
        // Move conversation to top when a new message is sent
        if (conversationId && !isPrivateChat) {
          moveConversationToTop(conversationId);
        }
      } else if (
        currentStatus === CHAT_STATUS.STREAMING ||
        currentStatus === CHAT_STATUS.SUBMITTED
      ) {
        const stoppedAfterMs = messageStartTimeRef.current
          ? Date.now() - messageStartTimeRef.current
          : null;
        analytics.capture("message_stopped", {
          conversation_id: conversationId || null,
          model: selectedModel?.id || null,
          mode: getCurrentMode(),
          stopped_after_ms: stoppedAfterMs,
          user_status: analytics.getUserStatus(isAuthenticated),
        });
        messageStartTimeRef.current = null;

        stop();
      } else {
        // Stop pending tool calls when user clicks stop (not streaming but tools are running)
        if (toolCallAbortController.current) {
          toolCallAbortController.current.abort();
          toolCallAbortController.current = null;
          sessionData.tools = [];
        }
      }
    },
    [
      chatSessionId,
      sendMessage,
      sessionData,
      status,
      stop,
      conversationId,
      isPrivateChat,
      moveConversationToTop,
      setMessages,
      createUserMessageItem,
      appendImageUrlsToPrompt,
      imageGenerationEnabled,
      selectedModel,
      getCurrentMode,
      isAuthenticated,
    ],
  );

  // Load conversation metadata (only for persistent conversations)
  useEffect(() => {
    if (conversationId && !isPrivateChat && models.length > 0) {
      getConversation(conversationId)
        .then((conversation) => {
          // Store conversation title for share dialog
          setConversationTitle(conversation.title);

          // Load model from metadata
          const modelId = conversation.metadata?.model_id;
          if (modelId) {
            const model = models.find((m) => m.id === modelId);
            if (model && model.id !== selectedModel?.id) {
              setSelectedModel(model);
            }
          }
        })
        .catch((error) => {
          if (error instanceof ApiError && error.status === 404) {
            handleConversationNotFound();
            return;
          }
          console.error("Failed to load conversation:", error);
        });
    }
  }, [conversationId, models.length, isPrivateChat]);

  // Agent execution store actions
  const startExecution = useAgentExecutionStore(
    (state) => state.startExecution,
  );
  const clearExecution = useAgentExecutionStore(
    (state) => state.clearExecution,
  );

  // Reset state when conversation changes
  useEffect(() => {
    initialMessageSentRef.current = false;
  }, [conversationId]);

  useEffect(() => {
    return () => {
      if (conversationId) {
        clearExecution(conversationId);
      }
    };
  }, [conversationId, clearExecution]);

  useEffect(() => {
    const initialMessageKey = isPrivateChat
      ? SESSION_STORAGE_KEY.INITIAL_MESSAGE_TEMPORARY
      : `${SESSION_STORAGE_PREFIX.INITIAL_MESSAGE}${conversationId}`;

    const storedMessage = sessionStorage.getItem(initialMessageKey);

    if (storedMessage && (isPrivateChat || conversationId)) {
      try {
        const message: PromptInputMessage = JSON.parse(storedMessage);
        // Clear the stored message
        sessionStorage.removeItem(initialMessageKey);
        // Mark as sent to prevent duplicate sends
        initialMessageSentRef.current = true;
        sessionData.tools = [];

        // Preload cached items if any
        const initialItemsKey = `${SESSION_STORAGE_PREFIX.INITIAL_ITEMS}${conversationId}`;
        const cachedItems = sessionStorage.getItem(initialItemsKey);
        if (cachedItems) {
          const items = JSON.parse(cachedItems) as any[];
          setMessages(convertToUIMessages(items));
          sessionStorage.removeItem(initialItemsKey);
        }

        const withImageUrls = appendImageUrlsToPrompt(
          message,
          imageGenerationEnabled,
        );

        // Persist to server (fire-and-forget, ID mapping handled in onFinish)
        createUserMessageItem(withImageUrls);

        sendMessage({
          text: withImageUrls.text,
          files: withImageUrls.files,
        });
        // Move conversation to top when initial message is sent
        if (conversationId && !isPrivateChat) {
          moveConversationToTop(conversationId);
        }
      } catch (error) {
        console.error("Failed to parse initial message:", error);
      }
    }
  }, [
    conversationId,
    isPrivateChat,
    sendMessage,
    sessionData,
    setMessages,
    moveConversationToTop,
    createUserMessageItem,
    appendImageUrlsToPrompt,
    imageGenerationEnabled,
  ]);

  // Fetch messages for old conversations (only for persistent conversations)
  useEffect(() => {
    if (
      conversationId &&
      !isPrivateChat &&
      !initialMessageSentRef.current &&
      !fetchingMessagesRef.current
    ) {
      // Check if session already has messages (e.g., returning to a streaming conversation)
      const existingSession =
        useChatSessions.getState().sessions[chatSessionId];
      if (
        existingSession?.chat.messages.length > 0 ||
        existingSession?.isStreaming
      ) {
        // Don't overwrite existing messages - session already has data
        return;
      }

      fetchingMessagesRef.current = true;
      // Clear messages first, then fetch (like ChatGPT)
      setMessages([]);
      getUIMessages(conversationId)
        .then((result) => {
          // Double-check session state hasn't changed during async fetch
          const currentSession =
            useChatSessions.getState().sessions[chatSessionId];
          if (currentSession?.isStreaming) {
            return; // Don't overwrite if streaming started
          }
          if (!initialMessageSentRef.current) setMessages(result.messages);
        })
        .catch((error) => {
          if (error instanceof ApiError && error.status === 404) {
            handleConversationNotFound();
            return;
          }
          console.error("Failed to load conversation items:", error);
        })
        .finally(() => {
          fetchingMessagesRef.current = false;
        });
    }
  }, [conversationId, isPrivateChat, chatSessionId]);

  // Auto-scroll reasoning container to bottom during streaming
  useEffect(() => {
    if (status === CHAT_STATUS.STREAMING && reasoningContainerRef.current) {
      reasoningContainerRef.current.scrollTop =
        reasoningContainerRef.current.scrollHeight;
    }
  }, [status, messages]);

  const topSentinelRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLElement | null>(null);

  const handleLoadMoreItems = useCallback(async () => {
    if (!conversationId || isPrivateChat || loadingMoreItems || !itemsCursor) {
      return;
    }
    const scrollContainer = scrollContainerRef.current;
    const previousScrollHeight = scrollContainer?.scrollHeight ?? 0;

    setLoadingMoreItems(true);
    try {
      const result = await loadMoreItems(conversationId);
      if (result.messages.length > 0) {
        setMessages((prev) => [...result.messages, ...prev]);
        requestAnimationFrame(() => {
          if (scrollContainer) {
            const newScrollHeight = scrollContainer.scrollHeight;
            scrollContainer.scrollTop = newScrollHeight - previousScrollHeight;
          }
        });
      }
    } finally {
      setLoadingMoreItems(false);
    }
  }, [
    conversationId,
    isPrivateChat,
    loadingMoreItems,
    itemsCursor,
    loadMoreItems,
    setMessages,
  ]);

  useEffect(() => {
    if (!conversationId || isPrivateChat) return;

    const sentinel = topSentinelRef.current;
    if (!sentinel) return;

    let scrollContainer: HTMLElement | null = sentinel.parentElement;
    while (scrollContainer) {
      const overflow = getComputedStyle(scrollContainer).overflow;
      if (overflow === "auto" || overflow === "scroll") {
        break;
      }
      scrollContainer = scrollContainer.parentElement;
    }
    scrollContainerRef.current = scrollContainer;

    const observer = new IntersectionObserver(
      (entries) => {
        const entry = entries[0];
        if (entry.isIntersecting) {
          handleLoadMoreItems();
        }
      },
      {
        root: scrollContainer,
        threshold: 0.1,
        rootMargin: "100px",
      },
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [conversationId, isPrivateChat, handleLoadMoreItems]);

  return (
    <>
      <AppSidebar />
      <SidebarInset>
        <NavHeader
          conversationId={conversationId}
          conversationTitle={conversationTitle}
        />
        <div className="flex flex-1 flex-col h-full overflow-hidden max-h-[calc(100vh-56px)] w-full ">
          {/* Messages Area */}
          <div className="flex-1 relative">
            <Conversation
              className="absolute inset-0 text-start"
              mass={SCROLL_ANIMATION.MASS}
              damping={SCROLL_ANIMATION.DAMPING}
              stiffness={SCROLL_ANIMATION.STIFFNESS}
            >
              <ConversationContent className="max-w-3xl mx-auto">
                {conversationId && !isPrivateChat && (
                  <div ref={topSentinelRef} className="h-1" />
                )}
                {loadingMoreItems && (
                  <div className="flex justify-center py-4">
                    <Loader className="animate-spin" />
                  </div>
                )}
                {messages.map((message, messageIndex) => (
                  <MessageItem
                    key={message.id}
                    message={message}
                    isFirstMessage={messageIndex === 0}
                    isLastMessage={messageIndex === messages.length - 1}
                    status={status}
                    reasoningContainerRef={reasoningContainerRef}
                    onRegenerate={conversationId ? handleRegenerate : undefined}
                    conversationId={conversationId}
                  />
                ))}
                {status === CHAT_STATUS.SUBMITTED && (
                  <Loader className="animate-spin" />
                )}
                {error && (
                  <div className="size-full mb-4 flex justify-center items-start">
                    <div className="text-center text-sm bg-muted rounded-full text-muted-foreground py-3 px-6 flex items-start gap-2 max-w-2xl">
                      <AlertTriangleIcon className="text-yellow-500 shrink-0 mt-0.5" />
                      <span className="text-left">
                        {error.message || "Something seems to have gone wrong."}
                      </span>
                    </div>
                  </div>
                )}
              </ConversationContent>
              <ConversationScrollButton />
            </Conversation>
          </div>

          {/* Chat Input - Fixed at bottom */}
          <div className="px-4 py-4 max-w-3xl mx-auto w-full">
            <ChatInput
              submit={handleSubmit}
              status={
                sessionData.tools.length > 0 ? CHAT_STATUS.STREAMING : status
              }
              conversationId={conversationId}
            />
          </div>
        </div>
      </SidebarInset>
      <AppSidebarRight />
    </>
  );
}
