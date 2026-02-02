import { useEffect, useState } from "react";
import { useRouter, Link, useLocation } from "@tanstack/react-router";
import {
  BarChart3,
  Box,
  ChevronRight,
  FileText,
  Home,
  LayoutDashboard,
  Loader2,
  Menu,
  Shield,
  Users,
  Wrench,
  X,
} from "lucide-react";
import { Button } from "@janhq/interfaces/button";
import { SidebarTrigger } from "@/components/sidebar/sidebar";
import { ThemeToggle } from "@/components/themes/theme-toggle";
import { useAuth } from "@/stores/auth-store";
import { useAdminStore } from "@/stores/admin-store";
import { cn } from "@/lib/utils";

interface AdminLayoutProps {
  children: React.ReactNode;
}

interface NavItem {
  title: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  children?: { title: string; href: string }[];
}

const navItems: NavItem[] = [
  {
    title: "Overview",
    href: "/admin",
    icon: LayoutDashboard,
  },
  {
    title: "Usage Analytics",
    href: "/admin/usage",
    icon: BarChart3,
  },
  {
    title: "User Management",
    href: "/admin/users",
    icon: Users,
    children: [
      { title: "Users", href: "/admin/users" },
      { title: "Feature Flags", href: "/admin/users/feature-flags" },
    ],
  },
  {
    title: "Model Management",
    href: "/admin/models",
    icon: Box,
    children: [
      { title: "Overview", href: "/admin/models" },
      { title: "Providers", href: "/admin/models/providers" },
      { title: "Provider Models", href: "/admin/models/provider-models" },
      { title: "Model Catalogs", href: "/admin/models/catalogs" },
    ],
  },
  {
    title: "Prompt Templates",
    href: "/admin/prompt-templates",
    icon: FileText,
  },
  {
    title: "MCP Tools",
    href: "/admin/mcp-tools",
    icon: Wrench,
  },
];

export function AdminLayout({ children }: AdminLayoutProps) {
  const router = useRouter();
  const location = useLocation();
  const isAuthenticated = useAuth((state) => state.isAuthenticated);
  const isGuest = useAuth((state) => state.isGuest);
  const { isAdmin, isLoading, checkAdminStatus } = useAdminStore();
  const [isChecking, setIsChecking] = useState(true);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const [expandedSections, setExpandedSections] = useState<string[]>([]);

  const pathname = location.pathname;

  // Auto-expand sections based on current path
  useEffect(() => {
    const sectionsToExpand: string[] = [];
    navItems.forEach((item) => {
      if (item.children) {
        const isChildActive = item.children.some((child) => pathname === child.href);
        if (isChildActive || pathname.startsWith(item.href + "/")) {
          sectionsToExpand.push(item.title);
        }
      }
    });
    setExpandedSections(sectionsToExpand);
  }, [pathname]);

  useEffect(() => {
    async function verifyAccess() {
      if (!isAuthenticated || isGuest) {
        router.navigate({ to: "/" });
        return;
      }

      setIsChecking(true);
      // Force check on each admin route navigation for security
      const adminStatus = await checkAdminStatus(true);
      setIsChecking(false);

      if (!adminStatus) {
        console.warn("Non-admin user attempted to access admin panel");
        router.navigate({ to: "/" });
      }
    }

    verifyAccess();
  }, [isAuthenticated, isGuest, checkAdminStatus, router, pathname]);

  const toggleSection = (title: string) => {
    setExpandedSections((prev) =>
      prev.includes(title)
        ? prev.filter((t) => t !== title)
        : [...prev, title]
    );
  };

  const isItemActive = (item: NavItem) => {
    // For Overview (/admin), only highlight if exactly on /admin
    if (item.href === "/admin") {
      return pathname === "/admin";
    }
    // For items with children, check if any child is active
    if (item.children) {
      return item.children.some((child) => pathname === child.href);
    }
    // For other items, check exact match or starts with
    return pathname === item.href || pathname.startsWith(item.href + "/");
  };

  if (isChecking || isLoading || isAdmin === null) {
    return (
      <div className="flex h-screen w-full bg-background overflow-hidden">
        <div className="flex items-center justify-center flex-1">
          <div className="text-center">
            <Loader2 className="w-8 h-8 animate-spin mx-auto mb-4 text-primary" />
            <p className="text-sm text-muted-foreground">
              Verifying admin access...
            </p>
          </div>
        </div>
      </div>
    );
  }

  if (!isAdmin) {
    return null;
  }

  return (
    <div className="flex h-screen w-full bg-background overflow-hidden">
      {/* Mobile nav toggle */}
      <div className="fixed top-4 left-4 z-50 md:hidden">
        <Button
          variant="outline"
          size="icon"
          onClick={() => setMobileNavOpen(!mobileNavOpen)}
        >
          {mobileNavOpen ? <X className="w-4 h-4" /> : <Menu className="w-4 h-4" />}
        </Button>
      </div>

      {/* Admin sidebar */}
      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-40 w-64 bg-card border-r transform transition-transform duration-200 ease-in-out md:relative md:translate-x-0 flex-shrink-0",
          mobileNavOpen ? "translate-x-0" : "-translate-x-full"
        )}
      >
        <div className="flex flex-col h-full">
          {/* Header */}
          <div className="flex items-center gap-2 p-4 border-b">
            <SidebarTrigger className="mr-2" />
            <Shield className="w-5 h-5 text-primary" />
            <span className="font-semibold">Admin Panel</span>
          </div>

          {/* Navigation */}
          <nav className="flex-1 overflow-y-auto p-4">
            {navItems.map((item) => {
              const Icon = item.icon;
              const isActive = isItemActive(item);
              const isExpanded = expandedSections.includes(item.title);
              const hasChildren = item.children && item.children.length > 0;

              return (
                <div key={item.href} className="mb-2">
                  <button
                    onClick={() => {
                      if (hasChildren) {
                        toggleSection(item.title);
                      } else {
                        router.navigate({ to: item.href });
                        setMobileNavOpen(false);
                      }
                    }}
                    className={cn(
                      "flex w-full items-center gap-2 px-3 py-2 text-sm font-medium rounded-md transition-colors",
                      isActive
                        ? "bg-primary text-primary-foreground"
                        : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                    )}
                  >
                    <Icon className="w-4 h-4" />
                    <span className="flex-1 text-left">{item.title}</span>
                    {hasChildren && (
                      <ChevronRight
                        className={cn(
                          "w-4 h-4 transition-transform",
                          isExpanded && "rotate-90"
                        )}
                      />
                    )}
                  </button>
                  {hasChildren && isExpanded && (
                    <div className="ml-4 mt-1 space-y-1 border-l pl-3">
                      {item.children?.map((child) => {
                        const isChildActive = pathname === child.href;
                        return (
                          <Link
                            key={child.href}
                            to={child.href}
                            onClick={() => setMobileNavOpen(false)}
                            className={cn(
                              "block px-3 py-1.5 text-sm rounded-md transition-colors",
                              isChildActive
                                ? "text-primary bg-primary/10 font-medium"
                                : "text-muted-foreground hover:text-foreground hover:bg-muted"
                            )}
                          >
                            {child.title}
                          </Link>
                        );
                      })}
                    </div>
                  )}
                </div>
              );
            })}
          </nav>

          {/* Footer */}
          <div className="p-4 border-t">
            <Link
              to="/"
              className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
            >
              <Home className="w-4 h-4" />
              Back to Chat
            </Link>
          </div>
        </div>
      </aside>

      {/* Mobile overlay */}
      {mobileNavOpen && (
        <div
          className="fixed inset-0 z-30 bg-black/50 md:hidden"
          onClick={() => setMobileNavOpen(false)}
        />
      )}

      {/* Main content - fills remaining space */}
      <main className="flex-1 min-w-0 h-full overflow-y-auto bg-background">
        {/* Top bar with theme toggle */}
        <div className="sticky top-0 z-10 flex items-center justify-end h-14 px-4 bg-background border-b">
          <ThemeToggle />
        </div>
        <div className="h-full w-full max-w-7xl mx-auto px-6 py-8">
          {children}
        </div>
      </main>
    </div>
  );
}
