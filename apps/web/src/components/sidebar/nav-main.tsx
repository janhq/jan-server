import { MessageCirclePlusIcon, FolderPenIcon, Search, Plug, LayoutGrid } from "lucide-react";
import { useRouter } from "@tanstack/react-router";
import * as React from "react";

import { SidebarMenu, useSidebar } from "@/components/sidebar/sidebar";
import { AnimatedMenuItem, type NavMainItem } from "@/components/sidebar/items";
import { URL_PARAM, URL_PARAM_VALUE } from "@/constants";
import { cn } from "@janhq/interfaces/lib";
import { useProfile } from "@/stores/profile-store";

export function NavMain() {
  const router = useRouter();
  const preferences = useProfile((state) => state.preferences);
  const hideConnectors = preferences?.preferences?.hide_connectors ?? false;
  const hideArtifacts = preferences?.preferences?.hide_artifacts ?? false;

  const handleNewProject = () => {
    const url = new URL(window.location.href);
    url.searchParams.set(URL_PARAM.PROJECTS, URL_PARAM_VALUE.CREATE);
    router.navigate({ to: url.pathname + url.search });
  };

  const handleSearch = () => {
    const url = new URL(window.location.href);
    url.searchParams.set(URL_PARAM.SEARCH, URL_PARAM_VALUE.OPEN);
    router.navigate({ to: url.pathname + url.search });
  };

  const navMain: NavMainItem[] = [
    {
      title: "New Chat",
      url: "/",
      icon: MessageCirclePlusIcon,
      isActive: false,
    },
    {
      title: "New Project",
      url: "#",
      icon: FolderPenIcon,
      onClick: handleNewProject,
    },
  ];

  if (!hideArtifacts) {
    navMain.push({
      title: "Artifacts",
      url: "/artifacts",
      icon: LayoutGrid,
    });
  }

  if (!hideConnectors) {
    navMain.push({
      title: "Connectors",
      url: "/connectors",
      icon: Plug,
    });
  }

  navMain.push({
    title: "Search",
    url: "#",
    icon: Search,
    onClick: handleSearch,
  });

  const { isMobile, setOpenMobile, state } = useSidebar();
  const [isTransitionComplete, setIsTransitionComplete] = React.useState(
    state === "collapsed",
  );

  React.useEffect(() => {
    if (state === "collapsed") {
      // Wait for the sidebar transition to complete (200ms as defined in sidebar.tsx)
      const timer = setTimeout(() => {
        setIsTransitionComplete(true);
      }, 200);
      return () => clearTimeout(timer);
    } else {
      setIsTransitionComplete(false);
    }
  }, [state]);

  return (
    <SidebarMenu
      className={cn(
        state === "collapsed" &&
          "[&>li]:transition-transform [&>li]:duration-200 [&>li]:ease-linear",
        state === "collapsed" &&
          isTransitionComplete &&
          "md:[&>li]:translate-x-[calc(50%-1rem)]",
      )}
    >
      {navMain.map((item, index) => (
        <AnimatedMenuItem
          key={item.title}
          item={item}
          isMobile={isMobile}
          setOpenMobile={setOpenMobile}
          index={index}
        />
      ))}
    </SidebarMenu>
  );
}
