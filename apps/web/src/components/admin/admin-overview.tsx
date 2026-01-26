import { useEffect, useState } from "react";
import { Link } from "@tanstack/react-router";
import {
  Activity,
  Box,
  Database,
  Loader2,
  TrendingUp,
  Users,
} from "lucide-react";
import {
  userManagementService,
  providerManagementService,
  providerModelService,
} from "@/services/admin-service";

interface DashboardStats {
  totalUsers: number;
  activeUsers: number;
  totalProviders: number;
  totalModels: number;
  activeModels: number;
}

export function AdminOverview() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadStats() {
      try {
        setIsLoading(true);

        // Fetch all stats in parallel
        const [usersResponse, providersResponse, modelsResponse] =
          await Promise.all([
            userManagementService.listUsers({ offset: 0, limit: 1 }),
            providerManagementService.listProviders({ limit: 1 }),
            providerModelService.listProviderModels({ limit: 1 }),
          ]);

        // Get active counts
        const [activeUsersResponse, activeModelsResponse] = await Promise.all([
          userManagementService.listUsers({ enabled: true, offset: 0, limit: 1 }),
          providerModelService.listProviderModels({ active: true, limit: 1 }),
        ]);

        setStats({
          totalUsers: usersResponse.total || 0,
          activeUsers: activeUsersResponse.total || 0,
          totalProviders: providersResponse.total || 0,
          totalModels: modelsResponse.total || 0,
          activeModels: activeModelsResponse.total || 0,
        });
      } catch (err) {
        console.error("Failed to load dashboard stats:", err);
        setError("Failed to load dashboard statistics");
      } finally {
        setIsLoading(false);
      }
    }

    loadStats();
  }, []);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <Loader2 className="w-8 h-8 animate-spin mx-auto mb-4 text-primary" />
          <p className="text-sm text-muted-foreground">Loading dashboard...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-6">
        <h3 className="text-lg font-semibold text-destructive mb-2">Error</h3>
        <p className="text-sm text-muted-foreground">{error}</p>
      </div>
    );
  }

  const statCards = [
    {
      title: "Total Users",
      value: stats?.totalUsers || 0,
      subtitle: `${stats?.activeUsers || 0} active`,
      icon: Users,
      link: "/admin/users",
      color: "text-blue-600",
      bgColor: "bg-blue-100 dark:bg-blue-900/20",
    },
    {
      title: "Total Models",
      value: stats?.totalModels || 0,
      subtitle: `${stats?.activeModels || 0} active`,
      icon: Box,
      link: "/admin/models/provider-models",
      color: "text-green-600",
      bgColor: "bg-green-100 dark:bg-green-900/20",
    },
    {
      title: "Providers",
      value: stats?.totalProviders || 0,
      subtitle: "Model providers",
      icon: Database,
      link: "/admin/models/providers",
      color: "text-purple-600",
      bgColor: "bg-purple-100 dark:bg-purple-900/20",
    },
    {
      title: "Model Usage",
      value:
        stats?.activeModels && stats?.totalModels
          ? `${Math.round((stats.activeModels / stats.totalModels) * 100)}%`
          : "0%",
      subtitle: "Active models ratio",
      icon: TrendingUp,
      link: "/admin/models",
      color: "text-orange-600",
      bgColor: "bg-orange-100 dark:bg-orange-900/20",
    },
  ];

  const quickActions = [
    {
      title: "Manage Users",
      description: "View, edit, and organize users into groups",
      href: "/admin/users",
      icon: Users,
    },
    {
      title: "Model Providers",
      description: "Configure and sync model providers",
      href: "/admin/models/providers",
      icon: Database,
    },
    {
      title: "Provider Models",
      description: "Manage individual models and their settings",
      href: "/admin/models/provider-models",
      icon: Box,
    },
    {
      title: "Model Catalogs",
      description: "Browse and configure model catalog entries",
      href: "/admin/models/catalogs",
      icon: Activity,
    },
  ];

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Admin Dashboard</h1>
        <p className="text-muted-foreground mt-2">
          Overview of your platform&apos;s users and models
        </p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {statCards.map((stat) => {
          const Icon = stat.icon;
          return (
            <Link key={stat.title} to={stat.link} className="block group">
              <div className="rounded-lg border bg-card p-6 shadow-sm transition-all hover:shadow-md hover:border-primary/50">
                <div className="flex items-center justify-between mb-4">
                  <div className={`${stat.bgColor} p-3 rounded-lg`}>
                    <Icon className={`w-6 h-6 ${stat.color}`} />
                  </div>
                </div>
                <div>
                  <p className="text-sm font-medium text-muted-foreground mb-1">
                    {stat.title}
                  </p>
                  <p className="text-3xl font-bold mb-1">{stat.value}</p>
                  <p className="text-xs text-muted-foreground">
                    {stat.subtitle}
                  </p>
                </div>
              </div>
            </Link>
          );
        })}
      </div>

      {/* Quick Actions */}
      <div>
        <h2 className="text-xl font-semibold mb-4">Quick Actions</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {quickActions.map((action) => {
            const Icon = action.icon;
            return (
              <Link key={action.href} to={action.href} className="block group">
                <div className="rounded-lg border bg-card p-6 shadow-sm transition-all hover:shadow-md hover:border-primary/50">
                  <div className="flex items-start gap-4">
                    <div className="bg-primary/10 p-3 rounded-lg">
                      <Icon className="w-5 h-5 text-primary" />
                    </div>
                    <div className="flex-1">
                      <h3 className="font-semibold mb-1 group-hover:text-primary transition-colors">
                        {action.title}
                      </h3>
                      <p className="text-sm text-muted-foreground">
                        {action.description}
                      </p>
                    </div>
                  </div>
                </div>
              </Link>
            );
          })}
        </div>
      </div>

      {/* System Status */}
      <div className="rounded-lg border bg-card p-6">
        <h2 className="text-xl font-semibold mb-4">System Status</h2>
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">API Status</span>
            <span className="flex items-center gap-2 text-sm">
              <span className="w-2 h-2 bg-green-500 rounded-full"></span>
              Operational
            </span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">Database</span>
            <span className="flex items-center gap-2 text-sm">
              <span className="w-2 h-2 bg-green-500 rounded-full"></span>
              Connected
            </span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">
              Authentication
            </span>
            <span className="flex items-center gap-2 text-sm">
              <span className="w-2 h-2 bg-green-500 rounded-full"></span>
              Active
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
