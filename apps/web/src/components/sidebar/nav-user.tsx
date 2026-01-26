import {
  ChevronsUpDown,
  LogOut,
  SettingsIcon,
  FlagIcon,
  LayoutDashboard,
  Shield,
  BookOpen,
} from "lucide-react";
import { useEffect } from "react";

import { Avatar, AvatarFallback, AvatarImage } from "@janhq/interfaces/avatar";

declare const VITE_REPORT_ISSUE_URL: string;
import {
  DropDrawer,
  DropDrawerContent,
  DropDrawerItem,
  DropDrawerLabel,
  DropDrawerSeparator,
  DropDrawerTrigger,
} from "@janhq/interfaces/dropdrawer";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/sidebar/sidebar";
import { useAuth } from "@/stores/auth-store";
import { useAdminStore } from "@/stores/admin-store";
import { useRouter, Link } from "@tanstack/react-router";
import { getInitialsAvatar } from "@/lib/utils";
import { URL_PARAM, SETTINGS_SECTION } from "@/constants";
import { cn } from "@janhq/interfaces/lib";

export function NavUser() {
  const user = useAuth((state) => state.user);
  const isGuest = useAuth((state) => state.isGuest);
  const logout = useAuth((state) => state.logout);
  const router = useRouter();
  const { state, setOpenMobile, isMobile } = useSidebar();

  // Admin status
  const isAdmin = useAdminStore((state) => state.isAdmin);
  const checkAdminStatus = useAdminStore((state) => state.checkAdminStatus);

  // Check if user is logged in (not guest and has user)
  const isLoggedIn = user && !isGuest;

  // Check admin status on mount
  useEffect(() => {
    if (isLoggedIn) {
      checkAdminStatus();
    }
  }, [isLoggedIn, checkAdminStatus]);

  const handleNavigation = () => {
    if (isMobile) {
      setOpenMobile(false);
    }
  };

  const handleOpenSettings = (section: string = SETTINGS_SECTION.GENERAL) => {
    const url = new URL(window.location.href);
    url.searchParams.set(URL_PARAM.SETTING, section);
    router.navigate({ to: url.pathname + url.search });
  };

  const isCollapsed = state === "collapsed";

  return (
    <SidebarMenu className={cn(isCollapsed && "md:items-center", "gap-1")}>
      {/* Dashboard Button - Only for logged in users */}
      {isLoggedIn && (
        <SidebarMenuItem>
          <SidebarMenuButton
            asChild
            tooltip="Dashboard"
            className="hover:bg-sidebar-accent"
          >
            <Link to="/dashboard" onClick={handleNavigation}>
              <LayoutDashboard className="size-4" />
              <span>Dashboard</span>
            </Link>
          </SidebarMenuButton>
        </SidebarMenuItem>
      )}

      {/* Administrator Button - Only show for admins */}
      {isLoggedIn && isAdmin && (
        <SidebarMenuItem>
          <SidebarMenuButton
            asChild
            tooltip="Administrator"
            className="hover:bg-sidebar-accent"
          >
            <Link to="/admin" onClick={handleNavigation}>
              <Shield className="size-4" />
              <span>Administrator</span>
            </Link>
          </SidebarMenuButton>
        </SidebarMenuItem>
      )}

      {/* Documentation Button - Always visible (public) */}
      <SidebarMenuItem>
        <SidebarMenuButton
          asChild
          tooltip="Documentation"
          className="hover:bg-sidebar-accent"
        >
          <Link to="/docs" onClick={handleNavigation}>
            <BookOpen className="size-4" />
            <span>Documentation</span>
          </Link>
        </SidebarMenuButton>
      </SidebarMenuItem>

      {/* Report Issue Button - Always visible (public) */}
      <SidebarMenuItem>
        <SidebarMenuButton
          asChild
          tooltip="Report Issue"
          className="hover:bg-sidebar-accent"
        >
          <a
            href={VITE_REPORT_ISSUE_URL}
            target="_blank"
            rel="noopener noreferrer"
          >
            <FlagIcon className="size-4" />
            <span>Report Issue</span>
          </a>
        </SidebarMenuButton>
      </SidebarMenuItem>

      {/* User Dropdown - Only for logged in users */}
      {isLoggedIn && user && (
        <SidebarMenuItem>
          <DropDrawer>
            <DropDrawerTrigger asChild>
              <SidebarMenuButton
                size="lg"
                className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
              >
                <Avatar className="h-8 w-8 rounded-full">
                  <AvatarImage src={user.avatar} alt={user.name} />
                  <AvatarFallback className="bg-primary text-background font-medium">
                    {getInitialsAvatar(user.name)}
                  </AvatarFallback>
                </Avatar>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-medium">{user.email || user.name}</span>
                </div>
                <ChevronsUpDown className="ml-auto size-4" />
              </SidebarMenuButton>
            </DropDrawerTrigger>
            <DropDrawerContent
              className="md:w-56"
              side={isCollapsed ? "right" : "top"}
              align="center"
              sideOffset={4}
            >
              <DropDrawerLabel className="lg:p-0 font-normal">
                <div className="flex items-center gap-2 px-3 py-1.5 text-left text-sm">
                  <div className="grid flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-medium">{user.name}</span>
                    <span className="truncate text-xs text-muted-foreground mt-1">
                      {user.email}
                    </span>
                  </div>
                </div>
              </DropDrawerLabel>
              <DropDrawerSeparator />
              <DropDrawerItem
                onClick={() => handleOpenSettings(SETTINGS_SECTION.GENERAL)}
              >
                <div className="flex gap-2 items-center justify-center">
                  <SettingsIcon className="text-muted-foreground" />
                  Settings
                </div>
              </DropDrawerItem>
              <DropDrawerSeparator />
              <DropDrawerItem
                onClick={async () => {
                  await logout();
                  router.navigate({
                    to: "/",
                    replace: true,
                  });
                }}
              >
                <div className="flex gap-2 items-center justify-center">
                  <LogOut className="text-muted-foreground ml-0.5" />
                  Log out
                </div>
              </DropDrawerItem>
            </DropDrawerContent>
          </DropDrawer>
        </SidebarMenuItem>
      )}
    </SidebarMenu>
  );
}
