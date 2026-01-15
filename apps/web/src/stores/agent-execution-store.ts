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
  executions: Record<string, AgentExecution>;
  startExecution: (conversationId: string, responseId: string, onComplete?: (execution: AgentExecution) => void) => void;
  loadHistoricalExecution: (conversationId: string, responseId: string) => Promise<void>;
  updateProgress: (conversationId: string, progress: PlanProgressResponse) => void;
  updatePlanDetails: (conversationId: string, details: PlanDetailResponse) => void;
  setError: (conversationId: string, error: string) => void;
  clearExecution: (conversationId: string) => void;
  clearAllExecutions: () => void;
  getExecution: (conversationId: string) => AgentExecution | undefined;
  stopPolling: (conversationId: string) => void;
}

const pollingIntervals: Record<string, ReturnType<typeof setInterval>> = {};
const completionCallbacks: Record<string, (execution: AgentExecution) => void> = {};

export const useAgentExecutionStore = create<AgentExecutionState>((set, get) => ({
  executions: {},

  startExecution: (conversationId: string, responseId: string, onComplete?: (execution: AgentExecution) => void) => {
    if (onComplete) {
      completionCallbacks[conversationId] = onComplete;
    }

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

    startPolling(conversationId, responseId, set, get);
  },

  loadHistoricalExecution: async (conversationId: string, responseId: string) => {
    try {
      const details = await responseApiService.getPlanDetails(responseId);
      const isStillRunning = details.status === "in_progress" || details.status === "running";

      set((state) => ({
        executions: {
          ...state.executions,
          [conversationId]: {
            responseId,
            conversationId,
            status: details.status as ExecutionStatus,
            progress: details.progress,
            planDetails: details,
            isPolling: isStillRunning,
          },
        },
      }));

      if (isStillRunning) {
        startPolling(conversationId, responseId, set, get);
      }
    } catch {
      // Failed to load historical execution
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
    stopPollingInterval(conversationId);

    set((state) => {
      const { [conversationId]: _, ...rest } = state.executions;
      return { executions: rest };
    });
  },

  clearAllExecutions: () => {
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
  stopPollingInterval(conversationId);

  const poll = async () => {
    const execution = get().executions[conversationId];
    if (!execution || !execution.isPolling) {
      stopPollingInterval(conversationId);
      return;
    }

    try {
      const details = await responseApiService.getPlanDetails(responseId);
      get().updatePlanDetails(conversationId, details);

      const isComplete = ["completed", "failed", "cancelled"].includes(details.status);

      if (isComplete) {
        get().stopPolling(conversationId);

        const callback = completionCallbacks[conversationId];
        if (callback) {
          const updatedExecution = get().executions[conversationId];
          if (updatedExecution) {
            callback(updatedExecution);
          }
          delete completionCallbacks[conversationId];
        }
      }
    } catch {
      // Don't stop polling on transient errors
    }
  };

  poll();
  pollingIntervals[conversationId] = setInterval(poll, POLL_INTERVAL_MS);
}

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
