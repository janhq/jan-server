type SearchResultItem = {
  type: "link" | "image" | "text";
  title?: string;
  description?: string;
  url?: string;
  imageUrl?: string;
  icon?: string;
  content?: string;
};

type ToolItem = {
  name: string;
  title: string;
  content: string;
  results: SearchResultItem[];
  status: string;
};

type Plan = {
  plan_id: number;
  title: string;
  description: string;
  status?: string;
  items?: ToolItem[];
};

const dummy: Plan[] = [
  {
    plan_id: 1,
    title:
      "Research and gather information on AI's impact on the future of work",
    description:
      "I'm gathering key research on AI's impact on jobs, focusing on automation, displacement, new roles, and reskilling to prepare an engaging, well-structured presentation.",
    status: "complete",
    items: [
      {
        name: "Search",
        title:
          "impact of AI on future of work automation job displacement new jobs reskilling",
        content: "",
        results: [
          {
            type: "link",
            title: "AI and the Future of Work: Navigating Job Displacement ...",
            description:
              "AI and automation have significantly reduced the need for manual labor in repetitive tasks such as assembly, quality control, and inventory management. AI- ...",
            url: "https://www.weforum.org",
          },
          {
            type: "link",
            title: "Ways AI Impacts the Job Market and Employment Trends",
            description:
              "AI is transforming recruitment and workforce management by optimizing key processes. One of its most significant benefits is improving hiring efficiency.",
            url: "https://www.mckinsey.com",
            icon: "💼",
          },
          {
            type: "link",
            title: "How will Artificial Intelligence Affect Jobs 2026-2030",
            description:
              "Oct 20, 2025 — PwC estimates that by the mid-2030s, up to 30% of jobs could be automatable, with slightly more men being affected in the long run as autonomous ...",
            url: "https://www.brookings.edu",
            icon: "📊",
          },
        ],
        status: "complete",
      },
      {
        name: "Browse",
        title: "Reading World Economic Forum Future of Jobs Report 2024",
        content: "",
        results: [
          {
            type: "link",
            title: "The Future of Jobs Report 2025",
            description:
              "Technological change, geoeconomic fragmentation, economic uncertainty, demographic shifts and the green transition are among the major drivers expected to shape the global labour market by 2030.",
            url: "https://www.weforum.org/publications/the-future-of-jobs-report-2025",
            icon: "🌐",
          },
          {
            type: "image",
            title: "Future of Jobs Report 2025 Cover",
            description:
              "The official report cover showing key statistics and trends",
            imageUrl:
              "https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=800&q=80",
          },
        ],
        status: "complete",
      },
      {
        name: "File",
        title: "Creating file ai_future_work_research.md",
        content: "",
        results: [
          {
            type: "text",
            title: "",
            content: `# Research Findings: Impact of AI on the Future of Work (2025-2030)

## 1. Job Displacement and Creation (WEF 2025 Report)
- **Structural Transformation:** 22% of today's total jobs will be affected by structural labor-market transformation between 2025 and 2030.
- **Job Creation:** 14% of today's total employment (approx. 170 million jobs) will be new roles.
- **Job Displacement:** 8% (approx. 92 million jobs) will be displaced.
- **Net Growth:** Net growth of 7% (approx. 78 million jobs) in total employment.

## 2. Automation Trends
- **PwC Estimate:** Up to 30% of jobs could be automatable by the mid-2030s.
- **Business Adoption:** 78% of organizations reported using AI in 2024 (up from 55% in 2023).`,
          },
        ],
        status: "complete",
      },
      {
        name: "Bash",
        title: "Running npm install chart.js for visualization",
        content: "",
        results: [
          {
            type: "text",
            title: "Command Output",
            content: `$ npm install chart.js
added 1 package, and audited 245 packages in 3s
found 0 vulnerabilities`,
          },
        ],
        status: "complete",
      },
    ],
  },
  {
    plan_id: 2,
    title:
      "Prepare detailed slide content outline with structure and key points",
    description:
      "AI will displace 8% of jobs but create 14%, mainly in tech and green roles. Significant reskilling is needed; 59% of the workforce may require training by 2030.",
    status: "active",
    // items: [
    //   {
    //     name: "File",
    //     title: "Generating infographic for automation statistics",
    //     content: "",
    //     links: [],
    //     status: "pending",
    //   },
    //   {
    //     name: "File Write",
    //     title: "Saving presentation slides to presentation.pptx",
    //     content: "",
    //     links: [],
    //     status: "pending",
    //   },
    // ],
  },
  {
    plan_id: 3,
    title: "Create visual presentation slides and finalize the deck",
    description:
      "Awaiting completion of previous steps before generating the final presentation slides with visualizations and formatting.",
    status: "pending",
    // items: [
    //   {
    //     name: "File",
    //     title: "Generating infographic for automation statistics",
    //     content: "",
    //     links: [],
    //     status: "pending",
    //   },
    //   {
    //     name: "File Write",
    //     title: "Saving presentation slides to presentation.pptx",
    //     content: "",
    //     links: [],
    //     status: "pending",
    //   },
    // ],
  },
];

import {
  AgentExecution,
  AgentExecutionContent,
  AgentExecutionHeader,
  AgentExecutionStep,
  getToolIcon,
} from "@janhq/interfaces/ai-elements/agent-execution";
import { useRightSidebarStore } from "@/stores/right-sidebar-store";
import type { StepWithResults } from "@/stores/right-sidebar-store";

const AgentExecutionDemo = () => {
  const setAllSteps = useRightSidebarStore((state) => state.setAllSteps);
  const setCurrentStep = useRightSidebarStore((state) => state.setCurrentStep);

  // Convert plan items to StepWithResults format
  const convertPlanToSteps = (plan: Plan): StepWithResults[] => {
    return (
      plan.items?.map((item) => ({
        stepName: item.name,
        stepTitle: item.title,
        results: item.results,
      })) || []
    );
  };

  const handleStepClick = (
    stepIndex: number,
    allPlanSteps: StepWithResults[]
  ) => {
    setAllSteps(allPlanSteps);
    setCurrentStep(stepIndex);
  };

  return (
    <>
      {dummy.map((plan, planIndex) => {
        const isLastItem = planIndex === dummy.length - 1;
        const shouldDefaultOpen = plan.status !== "pending";
        const allPlanSteps = convertPlanToSteps(plan);

        return (
          <AgentExecution
            key={plan.plan_id}
            defaultOpen={shouldDefaultOpen}
            lastItem={isLastItem}
          >
            <AgentExecutionHeader
              status={plan.status as "complete" | "active" | "pending"}
            >
              {plan.title}
            </AgentExecutionHeader>
            <AgentExecutionContent>
              {plan.description && (
                <p className="text-muted-foreground text-sm leading-relaxed ml-2">
                  {plan.description}
                </p>
              )}
              {plan.items?.map((item: ToolItem, stepIndex: number) => (
                <AgentExecutionStep
                  key={stepIndex}
                  icon={getToolIcon(item.name)}
                  label={item.title}
                  status={item.status as "complete" | "active" | "pending"}
                  searchResults={item.results}
                  onStepClick={() => handleStepClick(stepIndex, allPlanSteps)}
                />
              ))}
            </AgentExecutionContent>
          </AgentExecution>
        );
      })}
    </>
  );
};

export default AgentExecutionDemo;
