import {
  AgentExecution,
  AgentExecutionContent,
  AgentExecutionHeader,
  AgentExecutionStep,
  getToolIcon,
} from "@janhq/interfaces/ai-elements/agent-execution";
import { useRightSidebarStore } from "@/stores/right-sidebar-store";
import type { StepWithResults } from "@/stores/right-sidebar-store";
import { useAgentExecution } from "@/stores/agent-execution-store";
import {
  convertTaskToStepWithResults,
  getStepLabel,
  getStepToolName,
} from "@/lib/step-output-parser";
import type { TaskResponse, StepResponse } from "@/services/response-api-service";

interface AgentExecutionPanelProps {
  conversationId?: string;
  toolState?: string; // Tool state from the message part (e.g., "call", "partial-call", "result")
}

const AgentExecutionPanel = ({ conversationId, toolState }: AgentExecutionPanelProps) => {
  const setAllSteps = useRightSidebarStore((state) => state.setAllSteps);
  const setCurrentStep = useRightSidebarStore((state) => state.setCurrentStep);
  const execution = useAgentExecution(conversationId);

  // If no execution data yet, show loading state based on tool state
  // This handles the case when the tool is being called but startExecution hasn't been called yet
  if (!execution) {
    // Show loading state for any tool state - either calling or just completed waiting for polling to start
    // We show the panel as long as there's a toolState (meaning we're rendering for a run_agent tool)
    if (toolState) {
      const isToolInProgress = !toolState.startsWith("output-");
      return (
        <AgentExecution defaultOpen={true} lastItem={true}>
          <AgentExecutionHeader status="active">
            {isToolInProgress ? "Starting Deep Research..." : "Deep Research in progress..."}
          </AgentExecutionHeader>
          <AgentExecutionContent>
            <p className="text-muted-foreground text-sm leading-relaxed ml-2">
              {isToolInProgress
                ? "Invoking research agent. Please wait..."
                : "Initializing research agent. Please wait while the plan is being created..."}
            </p>
          </AgentExecutionContent>
        </AgentExecution>
      );
    }
    // No toolState and no execution - shouldn't happen but return nothing
    return null;
  }

  // Show loading state while polling but no plan details yet
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

  // Map step status to UI status
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

  // Handle step click - populate right sidebar with step results
  const handleStepClick = (
    stepIndex: number,
    allTaskSteps: StepWithResults[],
  ) => {
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
            <AgentExecutionHeader
              status={mapStatus(task.status)}
            >
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
