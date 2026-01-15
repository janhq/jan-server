import { useEffect } from "react";
import {
  AgentExecution,
  AgentExecutionContent,
  AgentExecutionHeader,
  AgentExecutionStep,
  getToolIcon,
} from "@janhq/interfaces/ai-elements/agent-execution";
import { useRightSidebarStore } from "@/stores/right-sidebar-store";
import type { StepWithResults } from "@/stores/right-sidebar-store";
import { useAgentExecution, useAgentExecutionStore } from "@/stores/agent-execution-store";
import {
  convertTaskToStepWithResults,
  getStepLabel,
  getStepToolName,
} from "@/lib/step-output-parser";
import type { TaskResponse, StepResponse } from "@/services/response-api-service";

interface AgentExecutionPanelProps {
  conversationId?: string;
  toolState?: string;
  responseId?: string;
}

const AgentExecutionPanel = ({ conversationId, toolState, responseId }: AgentExecutionPanelProps) => {
  const setAllSteps = useRightSidebarStore((state) => state.setAllSteps);
  const setCurrentStep = useRightSidebarStore((state) => state.setCurrentStep);
  const loadHistoricalExecution = useAgentExecutionStore((state) => state.loadHistoricalExecution);
  const execution = useAgentExecution(conversationId);

  useEffect(() => {
    if (conversationId && responseId && !execution) {
      loadHistoricalExecution(conversationId, responseId);
    }
  }, [conversationId, responseId, execution, loadHistoricalExecution]);

  if (!execution) {
    if (toolState || responseId) {
      const isToolInProgress = toolState && !toolState.startsWith("output-");
      return (
        <AgentExecution defaultOpen={true} lastItem={true}>
          <AgentExecutionHeader status="active">
            {isToolInProgress ? "Starting Deep Research..." : "Loading Deep Research..."}
          </AgentExecutionHeader>
          <AgentExecutionContent>
            <p className="text-muted-foreground text-sm leading-relaxed ml-2">
              {isToolInProgress
                ? "Invoking research agent. Please wait..."
                : "Loading research data..."}
            </p>
          </AgentExecutionContent>
        </AgentExecution>
      );
    }
    return null;
  }

  if (!execution.planDetails) {
    if (execution.isPolling) {
      return (
        <AgentExecution defaultOpen={true} lastItem={true}>
          <AgentExecutionHeader status="active">
            Deep Research in progress...
          </AgentExecutionHeader>
          <AgentExecutionContent>
            <p className="text-muted-foreground text-sm leading-relaxed ml-2">
              Initializing research agent. Please wait while the plan is being created...
            </p>
          </AgentExecutionContent>
        </AgentExecution>
      );
    }
    return null;
  }

  const tasks = execution.planDetails.tasks;

  const mapStatus = (status: string): "complete" | "active" | "pending" => {
    switch (status) {
      case "completed":
        return "complete";
      case "in_progress":
      case "running":
        return "active";
      default:
        return "pending";
    }
  };

  const handleStepClick = (stepIndex: number, allTaskSteps: StepWithResults[]) => {
    setAllSteps(allTaskSteps);
    setCurrentStep(stepIndex);
  };

  return (
    <>
      {tasks.map((task: TaskResponse, taskIndex: number) => {
        const isLastItem = taskIndex === tasks.length - 1;
        const shouldDefaultOpen = task.status !== "pending";
        const allTaskSteps = convertTaskToStepWithResults(task);

        return (
          <AgentExecution
            key={task.id}
            defaultOpen={shouldDefaultOpen}
            lastItem={isLastItem}
          >
            <AgentExecutionHeader status={mapStatus(task.status)}>
              {task.title}
            </AgentExecutionHeader>
            <AgentExecutionContent>
              {task.description && (
                <p className="text-muted-foreground text-sm leading-relaxed ml-2">
                  {task.description}
                </p>
              )}
              {task.steps?.map((step: StepResponse, stepIndex: number) => (
                <AgentExecutionStep
                  key={step.id}
                  icon={getToolIcon(getStepToolName(step))}
                  label={getStepLabel(step)}
                  status={mapStatus(step.status)}
                  searchResults={allTaskSteps[stepIndex]?.results || []}
                  onStepClick={() => handleStepClick(stepIndex, allTaskSteps)}
                />
              ))}
            </AgentExecutionContent>
          </AgentExecution>
        );
      })}
    </>
  );
};

export default AgentExecutionPanel;
