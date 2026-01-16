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
  status: ExecutionStatus;
  progress: number;
  planDetails: PlanDetailResponse | null;
  error?: string;
  isPolling: boolean;
}

interface AgentExecutionState {
  executions: Record<string, AgentExecution>;
  startExecution: (responseId: string, onComplete?: (execution: AgentExecution) => void) => void;
  loadHistoricalExecution: (responseId: string) => Promise<void>;
  updateProgress: (responseId: string, progress: PlanProgressResponse) => void;
  updatePlanDetails: (responseId: string, details: PlanDetailResponse) => void;
  setError: (responseId: string, error: string) => void;
  clearExecution: (responseId: string) => void;
  clearAllExecutions: () => void;
  getExecution: (responseId: string) => AgentExecution | undefined;
  stopPolling: (responseId: string) => void;
}

const pollingIntervals: Record<string, ReturnType<typeof setInterval>> = {};
const completionCallbacks: Record<string, (execution: AgentExecution) => void> = {};

export const useAgentExecutionStore = create<AgentExecutionState>((set, get) => ({
  executions: {},

  startExecution: (responseId: string, onComplete?: (execution: AgentExecution) => void) => {
    if (onComplete) {
      completionCallbacks[responseId] = onComplete;
    }

    set((state) => ({
      executions: {
        ...state.executions,
        [responseId]: {
          responseId,
          status: "in_progress",
          progress: 0,
          planDetails: null,
          isPolling: true,
        },
      },
    }));

    startPolling(responseId, set, get);
  },

  loadHistoricalExecution: async (responseId: string) => {
    set((state) => ({
      executions: {
        ...state.executions,
        [responseId]: {
          responseId,
          status: "in_progress",
          progress: 0,
          planDetails: null,
          isPolling: true,
        },
      },
    }));

    try {
      const details = await responseApiService.getPlanDetails(responseId);
      const isStillRunning = details.status === "in_progress" || details.status === "running";

      set((state) => ({
        executions: {
          ...state.executions,
          [responseId]: {
            responseId,
            status: details.status as ExecutionStatus,
            progress: details.progress,
            planDetails: details,
            isPolling: isStillRunning,
          },
        },
      }));

      if (isStillRunning) {
        startPolling(responseId, set, get);
      }
    } catch {
      startPolling(responseId, set, get);
    }
  },

  updateProgress: (responseId: string, progress: PlanProgressResponse) => {
    set((state) => {
      const existing = state.executions[responseId];
      if (!existing) return state;

      return {
        executions: {
          ...state.executions,
          [responseId]: {
            ...existing,
            status: progress.status as ExecutionStatus,
            progress: progress.progress,
          },
        },
      };
    });
  },

  updatePlanDetails: (responseId: string, details: PlanDetailResponse) => {
    set((state) => {
      const existing = state.executions[responseId];
      if (!existing) return state;

      return {
        executions: {
          ...state.executions,
          [responseId]: {
            ...existing,
            status: details.status as ExecutionStatus,
            progress: details.progress,
            planDetails: details,
          },
        },
      };
    });
  },

  setError: (responseId: string, error: string) => {
    set((state) => {
      const existing = state.executions[responseId];
      if (!existing) return state;

      return {
        executions: {
          ...state.executions,
          [responseId]: {
            ...existing,
            status: "failed",
            error,
            isPolling: false,
          },
        },
      };
    });
  },

  clearExecution: (responseId: string) => {
    stopPollingInterval(responseId);

    set((state) => {
      const { [responseId]: _, ...rest } = state.executions;
      return { executions: rest };
    });
  },

  clearAllExecutions: () => {
    Object.keys(pollingIntervals).forEach(stopPollingInterval);
    set({ executions: {} });
  },

  getExecution: (responseId: string) => {
    return get().executions[responseId];
  },

  stopPolling: (responseId: string) => {
    stopPollingInterval(responseId);

    set((state) => {
      const existing = state.executions[responseId];
      if (!existing) return state;

      return {
        executions: {
          ...state.executions,
          [responseId]: {
            ...existing,
            isPolling: false,
          },
        },
      };
    });
  },
}));

function stopPollingInterval(responseId: string) {
  if (pollingIntervals[responseId]) {
    clearInterval(pollingIntervals[responseId]);
    delete pollingIntervals[responseId];
  }
}

function startPolling(
  responseId: string,
  _set: (fn: (state: AgentExecutionState) => Partial<AgentExecutionState>) => void,
  get: () => AgentExecutionState,
) {
  stopPollingInterval(responseId);

  const poll = async () => {
    const execution = get().executions[responseId];
    if (!execution || !execution.isPolling) {
      stopPollingInterval(responseId);
      return;
    }

    try {
      const details = await responseApiService.getPlanDetails(responseId);
      get().updatePlanDetails(responseId, details);

      const isComplete = ["completed", "failed", "cancelled"].includes(details.status);

      if (isComplete) {
        get().stopPolling(responseId);

        const callback = completionCallbacks[responseId];
        if (callback) {
          const updatedExecution = get().executions[responseId];
          if (updatedExecution) {
            callback(updatedExecution);
          }
          delete completionCallbacks[responseId];
        }
      }
    } catch {
      // Don't stop polling on transient errors
    }
  };

  poll();
  pollingIntervals[responseId] = setInterval(poll, POLL_INTERVAL_MS);
}

export const useAgentExecution = (responseId: string | undefined) => {
  return useAgentExecutionStore((state) =>
    responseId ? state.executions[responseId] : undefined,
  );
};

export const useIsAgentExecuting = (responseId: string | undefined) => {
  return useAgentExecutionStore((state) => {
    if (!responseId) return false;
    const execution = state.executions[responseId];
    return execution?.status === "in_progress" && execution?.isPolling;
  });
};
