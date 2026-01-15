import { create } from "zustand";
import {
  responseApiService,
  type PlanDetailResponse,
  type PlanProgressResponse,
} from "@/services/response-api-service";

const POLL_INTERVAL_MS = 2000;

export type ExecutionStatus = "pending" | "in_progress" | "completed" | "failed" | "cancelled";

export interface AgentExecution {
  responseId: string;
  conversationId: string;
  status: ExecutionStatus;
  progress: number;
  planDetails: PlanDetailResponse | null;
  error?: string;
  isPolling: boolean;
}

interface AgentExecutionState {
  // Map of conversationId -> AgentExecution
  executions: Record<string, AgentExecution>;

  // Actions
  startExecution: (conversationId: string, responseId: string, onComplete?: (execution: AgentExecution) => void) => void;
  loadHistoricalExecution: (conversationId: string, responseId: string) => Promise<void>;
  loadConversationExecutions: (conversationId: string) => Promise<void>;
  updateProgress: (conversationId: string, progress: PlanProgressResponse) => void;
  updatePlanDetails: (conversationId: string, details: PlanDetailResponse) => void;
  setError: (conversationId: string, error: string) => void;
  clearExecution: (conversationId: string) => void;
  clearAllExecutions: () => void;
  getExecution: (conversationId: string) => AgentExecution | undefined;
  stopPolling: (conversationId: string) => void;
}

// Track polling intervals outside of store state
const pollingIntervals: Record<string, ReturnType<typeof setInterval>> = {};

// Callbacks for when execution completes - used to trigger follow-up in chat
const completionCallbacks: Record<string, (execution: AgentExecution) => void> = {};

export const useAgentExecutionStore = create<AgentExecutionState>((set, get) => ({
  executions: {},

  startExecution: (conversationId: string, responseId: string, onComplete?: (execution: AgentExecution) => void) => {
    // Store the completion callback if provided
    if (onComplete) {
      completionCallbacks[conversationId] = onComplete;
    }

    // Create initial execution state
    set((state) => ({
      executions: {
        ...state.executions,
        [conversationId]: {
          responseId,
          conversationId,
          status: "in_progress",
          progress: 0,
          planDetails: null,
          isPolling: true,
        },
      },
    }));

    // Start polling for progress
    startPolling(conversationId, responseId, set, get);
  },

  loadHistoricalExecution: async (conversationId: string, responseId: string) => {
    try {
      const details = await responseApiService.getPlanDetails(responseId);

      set((state) => ({
        executions: {
          ...state.executions,
          [conversationId]: {
            responseId,
            conversationId,
            status: details.status as ExecutionStatus,
            progress: details.progress,
            planDetails: details,
            isPolling: false,
          },
        },
      }));
    } catch (error) {
      console.error("Failed to load historical execution:", error);
    }
  },

  loadConversationExecutions: async (conversationId: string) => {
    try {
      const responses = await responseApiService.getConversationResponses(conversationId);

      // Find responses with plans (agent executions)
      for (const resp of responses.data) {
        if (resp.has_plan) {
          // Load the plan details for this response
          await get().loadHistoricalExecution(conversationId, resp.id);
          // For now, only load the first/most recent execution per conversation
          break;
        }
      }
    } catch (error) {
      console.error("Failed to load conversation executions:", error);
    }
  },

  updateProgress: (conversationId: string, progress: PlanProgressResponse) => {
    set((state) => {
      const existing = state.executions[conversationId];
      if (!existing) return state;

      return {
        executions: {
          ...state.executions,
          [conversationId]: {
            ...existing,
            status: progress.status as ExecutionStatus,
            progress: progress.progress,
          },
        },
      };
    });
  },

  updatePlanDetails: (conversationId: string, details: PlanDetailResponse) => {
    set((state) => {
      const existing = state.executions[conversationId];
      if (!existing) return state;

      return {
        executions: {
          ...state.executions,
          [conversationId]: {
            ...existing,
            status: details.status as ExecutionStatus,
            progress: details.progress,
            planDetails: details,
          },
        },
      };
    });
  },

  setError: (conversationId: string, error: string) => {
    set((state) => {
      const existing = state.executions[conversationId];
      if (!existing) return state;

      return {
        executions: {
          ...state.executions,
          [conversationId]: {
            ...existing,
            status: "failed",
            error,
            isPolling: false,
          },
        },
      };
    });
  },

  clearExecution: (conversationId: string) => {
    // Stop polling if active
    stopPollingInterval(conversationId);

    set((state) => {
      const { [conversationId]: _, ...rest } = state.executions;
      return { executions: rest };
    });
  },

  clearAllExecutions: () => {
    // Stop all polling
    Object.keys(pollingIntervals).forEach(stopPollingInterval);

    set({ executions: {} });
  },

  getExecution: (conversationId: string) => {
    return get().executions[conversationId];
  },

  stopPolling: (conversationId: string) => {
    stopPollingInterval(conversationId);

    set((state) => {
      const existing = state.executions[conversationId];
      if (!existing) return state;

      return {
        executions: {
          ...state.executions,
          [conversationId]: {
            ...existing,
            isPolling: false,
          },
        },
      };
    });
  },
}));

// Helper functions for polling management

function stopPollingInterval(conversationId: string) {
  if (pollingIntervals[conversationId]) {
    clearInterval(pollingIntervals[conversationId]);
    delete pollingIntervals[conversationId];
  }
}

function startPolling(
  conversationId: string,
  responseId: string,
  _set: (fn: (state: AgentExecutionState) => Partial<AgentExecutionState>) => void,
  get: () => AgentExecutionState,
) {
  // Clear any existing polling for this conversation
  stopPollingInterval(conversationId);

  const poll = async () => {
    const execution = get().executions[conversationId];
    if (!execution || !execution.isPolling) {
      stopPollingInterval(conversationId);
      return;
    }

    try {
      // Always fetch full details to get tasks and steps for real-time display
      const details = await responseApiService.getPlanDetails(responseId);
      get().updatePlanDetails(conversationId, details);

      // Check if execution is complete
      const isComplete = ["completed", "failed", "cancelled"].includes(details.status);

      if (isComplete) {
        get().stopPolling(conversationId);

        // Call completion callback if registered
        const callback = completionCallbacks[conversationId];
        if (callback) {
          const updatedExecution = get().executions[conversationId];
          if (updatedExecution) {
            callback(updatedExecution);
          }
          delete completionCallbacks[conversationId];
        }
      }
    } catch (error) {
      console.error("Polling error:", error);
      // Don't stop polling on transient errors, but log them
    }
  };

  // Initial poll
  poll();

  // Set up interval
  pollingIntervals[conversationId] = setInterval(poll, POLL_INTERVAL_MS);
}

// Selector hooks for components
export const useAgentExecution = (conversationId: string | undefined) => {
  return useAgentExecutionStore((state) =>
    conversationId ? state.executions[conversationId] : undefined,
  );
};

export const useIsAgentExecuting = (conversationId: string | undefined) => {
  return useAgentExecutionStore((state) => {
    if (!conversationId) return false;
    const execution = state.executions[conversationId];
    return execution?.status === "in_progress" && execution?.isPolling;
  });
};
