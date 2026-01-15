import { NavActions } from "@/components/sidebar/nav-actions";
import { ModelSelector } from "@/components/sidebar/model-selector";
import { SidebarTrigger, useSidebar } from "@/components/sidebar/sidebar";
import { ThemeToggle } from "../themes/theme-toggle";
import { memo } from "react";
import { Button } from "@janhq/interfaces/button";
import { PanelRightIcon } from "lucide-react";
import { useRightSidebarStore } from "@/stores/right-sidebar-store";

interface NavHeaderProps {
  conversationId?: string;
  conversationTitle?: string;
}

export const NavHeader = memo(function NavHeader({
  conversationId,
  conversationTitle,
}: NavHeaderProps = {}) {
  const { state, isMobile } = useSidebar();
  const toggleRightSidebar = useRightSidebarStore((state) => state.toggleSidebar);
  const rightSidebarOpen = useRightSidebarStore((state) => state.isOpen);

  return (
    <header className="flex h-14 shrink-0 items-center gap-2 justify-between">
      <div className="flex flex-1 items-center gap-2 px-3">
        {(isMobile || state === "collapsed") && (
          <SidebarTrigger className="text-muted-foreground" />
        )}
        <ModelSelector />
      </div>
      <div className="ml-auto px-3 flex items-center gap-2">
        <ThemeToggle />
        <NavActions
          conversationId={conversationId}
          conversationTitle={conversationTitle}
        />
        <Button
          variant="ghost"
          size="icon"
          className="size-7 text-muted-foreground"
          onClick={toggleRightSidebar}
          title={rightSidebarOpen ? "Close panel" : "Open panel"}
        >
          <PanelRightIcon className="size-4" />
        </Button>
      </div>
    </header>
  );
});
