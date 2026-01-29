import { useState } from "react";
import { Link, useLocation } from "@tanstack/react-router";
import {
  Book,
  ChevronRight,
  Menu,
  X,
  Home,
  Code,
  Settings,
  PlayCircle,
  Server,
} from "lucide-react";
import { Button } from "@janhq/interfaces/button";
import { cn } from "@/lib/utils";
import { SidebarTrigger } from "@/components/sidebar/sidebar";

interface NavItem {
  title: string;
  href: string;
  icon?: React.ComponentType<{ className?: string }>;
  children?: NavItem[];
}

const docsNavigation: NavItem[] = [
  {
    title: "Getting Started",
    href: "/docs",
    icon: Home,
    children: [
      { title: "Introduction", href: "/docs" },
      { title: "Quickstart", href: "/docs/quickstart" },
      { title: "Roadmap", href: "/docs/roadmap" },
    ],
  },
  {
    title: "API Reference",
    href: "/docs/api",
    icon: Code,
    children: [
      { title: "Authentication", href: "/docs/api/authentication" },
      { title: "Chat Completions", href: "/docs/api/chat-completions" },
      { title: "Conversations", href: "/docs/api/conversations" },
      { title: "Messages", href: "/docs/api/messages" },
      { title: "Models", href: "/docs/api/models" },
      { title: "Media", href: "/docs/api/media" },
    ],
  },
  {
    title: "Architecture",
    href: "/docs/architecture",
    icon: Server,
    children: [
      { title: "Overview", href: "/docs/architecture" },
      { title: "Services", href: "/docs/architecture/services" },
      { title: "Data Flow", href: "/docs/architecture/data-flow" },
      { title: "Security", href: "/docs/architecture/security" },
    ],
  },
  {
    title: "Guides",
    href: "/docs/guides",
    icon: PlayCircle,
    children: [
      { title: "Development", href: "/docs/guides/development" },
      { title: "Testing", href: "/docs/guides/testing" },
      { title: "Deployment", href: "/docs/guides/deployment" },
      { title: "MCP Tools", href: "/docs/guides/mcp-tools" },
    ],
  },
  {
    title: "Configuration",
    href: "/docs/configuration",
    icon: Settings,
    children: [
      { title: "Environment", href: "/docs/configuration/environment" },
      { title: "Providers", href: "/docs/configuration/providers" },
      { title: "Models", href: "/docs/configuration/models" },
    ],
  },
];

interface DocsLayoutProps {
  children: React.ReactNode;
}

export function DocsLayout({ children }: DocsLayoutProps) {
  const location = useLocation();
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const [expandedSections, setExpandedSections] = useState<string[]>([
    "Getting Started",
    "API Reference",
  ]);

  const toggleSection = (title: string) => {
    setExpandedSections((prev) =>
      prev.includes(title)
        ? prev.filter((t) => t !== title)
        : [...prev, title]
    );
  };

  const isActive = (href: string) => location.pathname === href;
  const isParentActive = (item: NavItem) => {
    if (isActive(item.href)) return true;
    return item.children?.some((child) => isActive(child.href)) ?? false;
  };

  const NavSection = ({ item }: { item: NavItem }) => {
    const Icon = item.icon;
    const isExpanded = expandedSections.includes(item.title);
    const hasChildren = item.children && item.children.length > 0;
    const active = isParentActive(item);

    return (
      <div className="mb-2">
        <button
          onClick={() => hasChildren && toggleSection(item.title)}
          className={cn(
            "flex w-full items-center gap-2 px-3 py-2 text-sm font-medium rounded-md transition-colors",
            active
              ? "text-primary bg-primary/10"
              : "text-muted-foreground hover:text-foreground hover:bg-muted"
          )}
        >
          {Icon && <Icon className="w-4 h-4" />}
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
            {item.children?.map((child) => (
              <Link
                key={child.href}
                to={child.href}
                onClick={() => setMobileNavOpen(false)}
                className={cn(
                  "block px-3 py-1.5 text-sm rounded-md transition-colors",
                  isActive(child.href)
                    ? "text-primary bg-primary/10 font-medium"
                    : "text-muted-foreground hover:text-foreground hover:bg-muted"
                )}
              >
                {child.title}
              </Link>
            ))}
          </div>
        )}
      </div>
    );
  };

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

      {/* Docs sidebar */}
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
            <Book className="w-5 h-5 text-primary" />
            <span className="font-semibold">Documentation</span>
          </div>

          {/* Navigation */}
          <nav className="flex-1 overflow-y-auto p-4">
            {docsNavigation.map((item) => (
              <NavSection key={item.title} item={item} />
            ))}
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
        <div className="h-full w-full max-w-4xl mx-auto px-6 py-8 md:py-12 md:px-8">
          {children}
        </div>
      </main>
    </div>
  );
}
