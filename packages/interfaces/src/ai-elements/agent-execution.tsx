/* eslint-disable react-refresh/only-export-components */
import { useControllableState } from "@radix-ui/react-use-controllable-state";

import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "../ui/collapsible";
import { cn } from "../lib/utils";
import {
  ChevronDownIcon,
  Loader2,
  CircleCheck,
  SearchIcon,
  GlobeIcon,
  CodeIcon,
  FileTextIcon,
  ImageIcon,
  BrainIcon,
  DatabaseIcon,
  TerminalIcon,
  DotIcon,
  type LucideIcon,
  Circle,
} from "lucide-react";
import type { ComponentProps, ReactNode } from "react";
import { createContext, memo, useContext, useMemo } from "react";

type AgentExecutionContextValue = {
  isOpen: boolean;
  setIsOpen: (open: boolean) => void;
};

const AgentExecutionContext = createContext<AgentExecutionContextValue | null>(
  null,
);

const useAgentExecution = () => {
  const context = useContext(AgentExecutionContext);
  if (!context) {
    throw new Error(
      "AgentExecution components must be used within AgentExecution",
    );
  }
  return context;
};

export type AgentExecutionProps = ComponentProps<"div"> & {
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  lastItem?: boolean;
};

export const AgentExecution = memo(
  ({
    className,
    open,
    defaultOpen = false,
    onOpenChange,
    lastItem = false,
    children,
    ...props
  }: AgentExecutionProps) => {
    const [isOpen, setIsOpen] = useControllableState({
      prop: open,
      defaultProp: defaultOpen,
      onChange: onOpenChange,
    });

    const agentExecutionContext = useMemo(
      () => ({ isOpen, setIsOpen }),
      [isOpen, setIsOpen],
    );

    return (
      <AgentExecutionContext.Provider value={agentExecutionContext}>
        <div
          className={cn("not-prose relative pb-6", className)}
          style={{ paddingRight: "24px" }}
          {...props}
        >
          {!lastItem && (
            <div className="absolute top-0 bottom-0 left-2 w-px bg-border transition-all" />
          )}
          {children}
        </div>
      </AgentExecutionContext.Provider>
    );
  },
);

export type AgentExecutionHeaderProps = ComponentProps<
  typeof CollapsibleTrigger
> & {
  icon?: LucideIcon;
  status?: "complete" | "active" | "pending";
};

export const AgentExecutionHeader = memo(
  ({
    className,
    children,
    status = "pending",
    ...props
  }: AgentExecutionHeaderProps) => {
    const { isOpen, setIsOpen } = useAgentExecution();
    const isPending = status === "pending";

    return (
      <Collapsible onOpenChange={setIsOpen} open={isOpen} className="w-full" >
        <CollapsibleTrigger
          className={cn(
            "flex w-full items-center gap-2 text-muted-foreground text-sm transition-colors hover:text-foreground",
            isPending && "cursor-not-allowed opacity-50",
            className,
          )}
          disabled={isPending}
          {...props}
        >
          <div className="bg-background z-20 relative flex shrink-0 items-center justify-center size-4">
            {status === "complete" && (
              <CircleCheck className="size-4 text-green-600 dark:text-green-500" />
            )}
            {status === "pending" && (
              <Circle className="size-4 text-muted-foreground fill-background" />
            )}
            {status === "active" && (
              <Loader2 className="size-4 animate-spin text-muted-foreground" />
            )}
          </div>

          <div
            className={cn(
              "text-left overflow-hidden text-ellipsis whitespace-nowrap",
              status === "active" &&
                "animate-pulse bg-linear-to-r from-foreground via-muted-foreground to-foreground bg-clip-text text-transparent",
            )}
          >
            {children ?? "Agent Execution"}
          </div>

          <ChevronDownIcon
            className={cn(
              "size-4 shrink-0 transition-transform",
              isOpen ? "rotate-180" : "rotate-0",
            )}
          />
        </CollapsibleTrigger>
      </Collapsible>
    );
  },
);

export type SearchResultItem = {
  type: "link" | "image" | "text" | "artifact";
  title?: string;
  description?: string;
  url?: string;
  imageUrl?: string;
  icon?: string;
  content?: string;
};

export type AgentExecutionStepProps = Omit<ComponentProps<"div">, "results"> & {
  icon?: LucideIcon;
  label: ReactNode;
  description?: ReactNode;
  status?: "complete" | "active" | "pending";
  searchResults?: SearchResultItem[];
  onStepClick?: (results: SearchResultItem[]) => void;
};

export const AgentExecutionStep = memo(
  ({
    className,
    icon: Icon = DotIcon,
    label,
    description,
    status = "complete",
    searchResults = [],
    onStepClick,
    children,
    ...props
  }: AgentExecutionStepProps) => {
    const statusStyles = {
      complete: "text-muted-foreground",
      active: "text-muted-foreground",
      pending: "text-muted-foreground/50",
    };

    const isClickable = searchResults.length > 0 && status !== "pending";

    const handleClick = () => {
      if (isClickable && onStepClick) {
        onStepClick(searchResults);
      }
    };

    return (
      <div
        className={cn(
          "flex gap-2 text-sm relative overflow-hidden",
          statusStyles[status],
          "fade-in-0 slide-in-from-top-2 animate-in border py-1 px-2 rounded-md bg-secondary/50",
          isClickable ? "cursor-pointer hover:bg-secondary/80" : 'inline-flex mr-2',
          status === "pending" && "cursor-not-allowed",
          className,
        )}
        onClick={handleClick}
        {...props}
      >
        {status === "active" && (
          <div
            className="absolute inset-0 bg-linear-to-r from-transparent via-background to-transparent"
            style={{
              animation: "shimmer 5s ease-in-out infinite",
              backgroundSize: "200% 100%",
            }}
          />
        )}
        <div className="relative flex shrink-0 items-center justify-center z-10">
          <Icon className="size-3" />
        </div>
        <div className="flex-1 space-y-2 overflow-hidden z-10">
          <div
            className={cn(
              status === "active" &&
                "bg-linear-to-r from-foreground via-muted-foreground to-foreground/50 bg-clip-text text-transparent",
              status === "active" && "animate-pulse",
            )}
          >
            <span className="line-clamp-1">{label}</span>
          </div>
          {description && (
            <div className="text-muted-foreground text-xs">{description}</div>
          )}
          {children}
        </div>
      </div>
    );
  },
);

export type AgentExecutionContentProps = ComponentProps<
  typeof CollapsibleContent
>;

export const AgentExecutionContent = memo(
  ({ className, children, ...props }: AgentExecutionContentProps) => {
    const { isOpen } = useAgentExecution();

    return (
      <Collapsible open={isOpen} className="w-full">
        <CollapsibleContent
          className={cn(
            "mt-2 space-y-3 w-full ml-6",
            "data-[state=closed]:fade-out-0 data-[state=closed]:slide-out-to-top-2 data-[state=open]:slide-in-from-top-2 text-popover-foreground outline-none data-[state=closed]:animate-out data-[state=open]:animate-in",
            className,
          )}
          {...props}
        >
          {children}
        </CollapsibleContent>
      </Collapsible>
    );
  },
);

export type AgentExecutionImageProps = ComponentProps<"div"> & {
  caption?: string;
};

export const AgentExecutionImage = memo(
  ({ className, children, caption, ...props }: AgentExecutionImageProps) => (
    <div className={cn("mt-2 space-y-2", className)} {...props}>
      <div className="relative flex max-h-88 items-center justify-center overflow-hidden rounded-lg bg-muted p-3">
        {children}
      </div>
      {caption && <p className="text-muted-foreground text-xs">{caption}</p>}
    </div>
  ),
);

export const getToolIcon = (name: string): LucideIcon | undefined => {
  const nameLower = name.toLowerCase();

  // Search related
  if (nameLower.includes("search") || nameLower.includes("query")) {
    return SearchIcon;
  }

  // Web browsing
  if (
    nameLower.includes("browse") ||
    nameLower.includes("web") ||
    nameLower.includes("url") ||
    nameLower.includes("fetch")
  ) {
    return GlobeIcon;
  }

  // Code related
  if (
    nameLower.includes("code") ||
    nameLower.includes("program") ||
    nameLower.includes("script")
  ) {
    return CodeIcon;
  }

  // Terminal/command execution
  if (
    nameLower.includes("terminal") ||
    nameLower.includes("command") ||
    nameLower.includes("exec") ||
    nameLower.includes("bash") ||
    nameLower.includes("shell")
  ) {
    return TerminalIcon;
  }

  // File operations
  if (
    nameLower.includes("file") ||
    nameLower.includes("read") ||
    nameLower.includes("write")
  ) {
    return FileTextIcon;
  }

  // Image related
  if (
    nameLower.includes("image") ||
    nameLower.includes("photo") ||
    nameLower.includes("picture")
  ) {
    return ImageIcon;
  }

  // Database operations
  if (
    nameLower.includes("database") ||
    nameLower.includes("sql") ||
    nameLower.includes("query")
  ) {
    return DatabaseIcon;
  }

  // AI/Reasoning
  if (
    nameLower.includes("think") ||
    nameLower.includes("reason") ||
    nameLower.includes("plan") ||
    nameLower.includes("ai")
  ) {
    return BrainIcon;
  }

  // Default: no icon
  return undefined;
};

AgentExecution.displayName = "AgentExecution";
AgentExecutionHeader.displayName = "AgentExecutionHeader";
AgentExecutionStep.displayName = "AgentExecutionStep";
AgentExecutionContent.displayName = "AgentExecutionContent";
AgentExecutionImage.displayName = "AgentExecutionImage";
