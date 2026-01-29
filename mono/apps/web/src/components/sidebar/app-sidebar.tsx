import { NavChats } from "@/components/sidebar/nav-chat";
import { NavMain } from "@/components/sidebar/nav-main";
import { NavProjects } from "@/components/sidebar/nav-projects";
import {
  Sidebar,
  SidebarContent,
  SidebarHeader,
  SidebarRail,
  useSidebar,
  SidebarTrigger,
  SidebarFooter,
} from "@/components/sidebar/sidebar";
import { cn } from "@/lib/utils";
import { NavUser } from "@/components/sidebar/nav-user";
import { memo, useEffect, useState } from "react";
import { Jan } from "@janhq/interfaces/svgs/jan";
import { StaggeredAnimationProvider } from "@/hooks/useStaggeredFadeIn";
import { useProjects } from "@/stores/projects-store";
import { useConversations } from "@/stores/conversation-store";

export const AppSidebar = memo(function AppSidebar({
  ...props
}: React.ComponentProps<typeof Sidebar>) {
  const { state, isMobile, openMobile, variant } = useSidebar();
  const isOpen = state === "expanded";

  const [isReady, setIsReady] = useState(false);
  const getProjects = useProjects((state) => state.getProjects);
  const getConversations = useConversations((state) => state.getConversations);
  const projects = useProjects((state) => state.projects);
  const [isTransitionComplete, setIsTransitionComplete] = useState(
    state === "collapsed",
  );

  // Calculate animation indices:
  // NavMain: 0, 1, 2, 3 (4 items: New Chat, New Project, Connectors, Search)
  // NavProjects: starts at 4
  // NavChats: starts after projects (4 + 1 label + N projects, or 4 if no projects)
  const chatsStartIndex = projects.length > 0 ? 4 + 1 + projects.length : 4;

  useEffect(() => {
    if (state === "collapsed") {
      // Wait for the sidebar transition to complete (200ms as defined in sidebar.tsx)
      const timer = setTimeout(() => {
        setIsTransitionComplete(true);
      }, 200);
      return () => clearTimeout(timer);
    } else {
      const timer = setTimeout(() => {
        setIsTransitionComplete(false);
      }, 0);
      return () => clearTimeout(timer);
    }
  }, [state]);

  useEffect(() => {
    const loadData = async () => {
      await Promise.all([getProjects(), getConversations()]);
      // Wait for next frame to ensure store updates have propagated
      requestAnimationFrame(() => {
        setIsReady(true);
      });
    };
    loadData();
  }, [getProjects, getConversations]);

  return (
    <Sidebar className="border-r-0" {...props} variant={variant}>
      <StaggeredAnimationProvider ready={isReady}>
        <SidebarHeader className={cn("pt-3.5", !isOpen && "md:gap-y-4")}>
          <div
            className={cn(
              "flex items-center w-full pl-0.5",
              isOpen && "pl-2 mb-2 justify-between",
              openMobile && "pl-2 mb-2 justify-between",
              !isMobile && !isOpen && "justify-between",
            )}
          >
            <div
              className={cn(
                "flex gap-2 items-center transition-transform duration-200 ease-linear",
                state === "collapsed" &&
                  isTransitionComplete &&
                  "md:translate-x-[calc(50%-8px)]",
              )}
            >
              <Jan className="size-4 shrink-0 block md:hidden" />
              <span className="text-lg font-bold font-studio">Jan</span>
            </div>

            {(isOpen || openMobile) && (
              <SidebarTrigger className="text-muted-foreground hover:bg-foreground/10" />
            )}
            {!isOpen && !openMobile && !isMobile && !isTransitionComplete && (
              <div className="w-8 h-8 shrink-0" />
            )}
          </div>
          <NavMain />
        </SidebarHeader>
        <SidebarContent className="mask-b-from-95% mask-t-from-98%">
          <NavProjects startIndex={4} />
          <NavChats startIndex={chatsStartIndex} />
        </SidebarContent>
        <SidebarFooter className={cn(!isOpen && "pb-4")}>
          <NavUser />
        </SidebarFooter>
      </StaggeredAnimationProvider>
      <SidebarRail />
    </Sidebar>
  );
});
