import { fetchJsonWithAuth } from "@/lib/api-client";

declare const JAN_API_BASE_URL: string;

// Types matching the response-api server responses
export interface PlanProgressResponse {
  plan_id: string;
  status: string;
  progress: number;
  estimated_steps: number;
  completed_steps: number;
  failed_steps: number;
  current_task?: TaskProgressResponse;
}

export interface TaskProgressResponse {
  task_id: string;
  title: string;
  status: string;
}

export interface StepResponse {
  id: string;
  object: string;
  task_id: string;
  sequence: number;
  action: string;
  status: string;
  retry_count: number;
  max_retries: number;
  error?: string;
  error_severity?: string;
  duration_ms?: number;
  planned_params?: Record<string, unknown>;
  actual_params?: Record<string, unknown>;
  output_data?: Record<string, unknown>;
  started_at?: number;
  completed_at?: number;
}

export interface TaskResponse {
  id: string;
  object: string;
  plan_id: string;
  sequence: number;
  task_type: string;
  status: string;
  title: string;
  description?: string;
  error?: string;
  steps?: StepResponse[];
  created_at: number;
  updated_at: number;
  completed_at?: number;
}

export interface PlanDetailResponse {
  id: string;
  object: string;
  response_id: string;
  status: string;
  progress: number;
  agent_type: string;
  estimated_steps: number;
  completed_steps: number;
  current_task_id?: string;
  final_artifact_id?: string;
  error?: string;
  created_at: number;
  updated_at: number;
  completed_at?: number;
  tasks: TaskResponse[];
}

// Response API routes go through Kong at /responses prefix
// which strips the prefix and forwards to response-api service
const RESPONSE_API_PREFIX = `${JAN_API_BASE_URL}responses/`;

export const responseApiService = {
  /**
   * Get plan progress for polling during active execution
   */
  getPlanProgress: async (responseId: string): Promise<PlanProgressResponse> => {
    return fetchJsonWithAuth<PlanProgressResponse>(
      `${RESPONSE_API_PREFIX}v1/responses/${responseId}/plan/progress`,
    );
  },

  /**
   * Get full plan details with tasks and steps
   */
  getPlanDetails: async (responseId: string): Promise<PlanDetailResponse> => {
    return fetchJsonWithAuth<PlanDetailResponse>(
      `${RESPONSE_API_PREFIX}v1/responses/${responseId}/plan/details`,
    );
  },

  /**
   * Cancel a plan execution
   */
  cancelPlan: async (
    responseId: string,
    reason?: string,
  ): Promise<{ id: string; status: string }> => {
    return fetchJsonWithAuth<{ id: string; status: string }>(
      `${RESPONSE_API_PREFIX}v1/responses/${responseId}/plan/cancel`,
      {
        method: "POST",
        body: JSON.stringify({ reason }),
      },
    );
  },
};
