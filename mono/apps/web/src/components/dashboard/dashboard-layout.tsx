import { useEffect, useState } from "react";
import { useRouter, Link, useLocation } from "@tanstack/react-router";
import {
  BarChart3,
  Home,
  Key,
  Loader2,
  Menu,
  User,
  X,
} from "lucide-react";
import { Button } from "@janhq/interfaces/button";
import { SidebarTrigger } from "@/components/sidebar/sidebar";
import { ThemeToggle } from "@/components/themes/theme-toggle";
import { useAuth } from "@/stores/auth-store";
import { cn } from "@/lib/utils";

interface DashboardLayoutProps {
  children: React.ReactNode;
}

interface NavItem {
  title: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
}

const navItems: NavItem[] = [
  {
    title: "Profile",
    href: "/dashboard",
    icon: User,
  },
  {
    title: "Usage",
    href: "/dashboard/usage",
    icon: BarChart3,
  },
  {
    title: "API Keys",
    href: "/dashboard/api-keys",
    icon: Key,
  },
];

export function DashboardLayout({ children }: DashboardLayoutProps) {
  const router = useRouter();
  const location = useLocation();
  const isAuthenticated = useAuth((state) => state.isAuthenticated);
  const isGuest = useAuth((state) => state.isGuest);
  const [isChecking, setIsChecking] = useState(true);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  const pathname = location.pathname;

  useEffect(() => {
    async function verifyAccess() {
      if (!isAuthenticated || isGuest) {
        router.navigate({ to: "/" });
        return;
      }
      setIsChecking(false);
    }

    verifyAccess();
  }, [isAuthenticated, isGuest, router]);

  const isItemActive = (item: NavItem) => {
    if (item.href === "/dashboard") {
      return pathname === "/dashboard";
    }
    return pathname === item.href || pathname.startsWith(item.href + "/");
  };

  if (isChecking) {
    return (
      <div className="flex h-screen w-full bg-background overflow-hidden">
        <div className="flex items-center justify-center flex-1">
          <div className="text-center">
            <Loader2 className="w-8 h-8 animate-spin mx-auto mb-4 text-primary" />
            <p className="text-sm text-muted-foreground">Loading...</p>
          </div>
        </div>
      </div>
    );
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
          {mobileNavOpen ? (
            <X className="w-4 h-4" />
          ) : (
            <Menu className="w-4 h-4" />
          )}
        </Button>
      </div>

      {/* Dashboard sidebar */}
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
            <User className="w-5 h-5 text-primary" />
            <span className="font-semibold">Dashboard</span>
          </div>

          {/* Navigation */}
          <nav className="flex-1 overflow-y-auto p-4">
            {navItems.map((item) => {
              const Icon = item.icon;
              const isActive = isItemActive(item);

              return (
                <div key={item.href} className="mb-2">
                  <Link
                    to={item.href}
                    onClick={() => setMobileNavOpen(false)}
                    className={cn(
                      "flex w-full items-center gap-2 px-3 py-2 text-sm font-medium rounded-md transition-colors",
                      isActive
                        ? "bg-primary text-primary-foreground"
                        : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                    )}
                  >
                    <Icon className="w-4 h-4" />
                    <span className="flex-1 text-left">{item.title}</span>
                  </Link>
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

      {/* Main content */}
      <main className="flex-1 min-w-0 h-full overflow-y-auto bg-background">
        {/* Top bar with theme toggle */}
        <div className="sticky top-0 z-10 flex items-center justify-end h-14 px-4 bg-background border-b">
          <ThemeToggle />
        </div>
        <div className="h-full w-full max-w-4xl mx-auto px-6 py-8">
          {children}
        </div>
      </main>
    </div>
  );
}
