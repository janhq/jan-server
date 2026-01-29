import { useEffect, useState } from "react";
import { Link } from "@tanstack/react-router";
import { ArrowRight, Box, Database, Layers, Loader2 } from "lucide-react";
import {
  providerManagementService,
  providerModelService,
  modelCatalogService,
} from "@/services/admin-service";

interface ModelStats {
  totalProviders: number;
  activeProviders: number;
  totalModels: number;
  activeModels: number;
  totalCatalogs: number;
}

export function ModelsOverview() {
  const [stats, setStats] = useState<ModelStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadStats() {
      try {
        setIsLoading(true);

        const [providersResponse, modelsResponse, catalogsResponse] =
          await Promise.all([
            providerManagementService.listProviders(),
            providerModelService.listProviderModels({ limit: 1 }),
            modelCatalogService.listModelCatalogs({ limit: 1 }),
          ]);

        const activeModelsResponse = await providerModelService.listProviderModels({
          active: true,
          limit: 1,
        });

        const providersList = providersResponse.data || [];
        const activeProviders = providersList.filter((p) => p.active).length;

        setStats({
          totalProviders: providersResponse.total || providersList.length || 0,
          activeProviders: activeProviders,
          totalModels: modelsResponse.total || 0,
          activeModels: activeModelsResponse.total || 0,
          totalCatalogs: catalogsResponse.total || 0,
        });
      } catch (err) {
        console.error("Failed to load model stats:", err);
        setError("Failed to load model statistics");
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
          <p className="text-sm text-muted-foreground">
            Loading model statistics...
          </p>
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

  const sections = [
    {
      title: "Model Providers",
      description: "Manage model providers and sync their available models",
      href: "/admin/models/providers",
      icon: Database,
      stats: [
        { label: "Total Providers", value: stats?.totalProviders || 0 },
        { label: "Active", value: stats?.activeProviders || 0 },
      ],
      color: "text-purple-600",
      bgColor: "bg-purple-100 dark:bg-purple-900/20",
    },
    {
      title: "Provider Models",
      description: "Browse and configure individual models from all providers",
      href: "/admin/models/provider-models",
      icon: Box,
      stats: [
        { label: "Total Models", value: stats?.totalModels || 0 },
        { label: "Active", value: stats?.activeModels || 0 },
      ],
      color: "text-green-600",
      bgColor: "bg-green-100 dark:bg-green-900/20",
    },
    {
      title: "Model Catalogs",
      description: "View and manage model catalog entries and capabilities",
      href: "/admin/models/catalogs",
      icon: Layers,
      stats: [{ label: "Total Catalogs", value: stats?.totalCatalogs || 0 }],
      color: "text-blue-600",
      bgColor: "bg-blue-100 dark:bg-blue-900/20",
    },
  ];

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Model Management</h1>
        <p className="text-muted-foreground mt-2">
          Manage model providers, individual models, and catalog entries
        </p>
      </div>

      {/* Stats Overview */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-card rounded-lg border p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="bg-purple-100 dark:bg-purple-900/20 p-3 rounded-lg">
              <Database className="w-6 h-6 text-purple-600" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Providers</p>
              <p className="text-2xl font-bold">{stats?.totalProviders || 0}</p>
            </div>
          </div>
          <div className="text-sm text-muted-foreground">
            {stats?.activeProviders || 0} active
          </div>
        </div>

        <div className="bg-card rounded-lg border p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="bg-green-100 dark:bg-green-900/20 p-3 rounded-lg">
              <Box className="w-6 h-6 text-green-600" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Models</p>
              <p className="text-2xl font-bold">{stats?.totalModels || 0}</p>
            </div>
          </div>
          <div className="text-sm text-muted-foreground">
            {stats?.activeModels || 0} active
          </div>
        </div>

        <div className="bg-card rounded-lg border p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="bg-blue-100 dark:bg-blue-900/20 p-3 rounded-lg">
              <Layers className="w-6 h-6 text-blue-600" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Catalogs</p>
              <p className="text-2xl font-bold">{stats?.totalCatalogs || 0}</p>
            </div>
          </div>
          <div className="text-sm text-muted-foreground">
            Model catalog entries
          </div>
        </div>
      </div>

      {/* Management Sections */}
      <div>
        <h2 className="text-xl font-semibold mb-4">Management Sections</h2>
        <div className="space-y-4">
          {sections.map((section) => {
            const Icon = section.icon;
            return (
              <Link key={section.href} to={section.href} className="block group">
                <div className="bg-card rounded-lg border p-6 hover:shadow-md hover:border-primary/50 transition-all">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-3 mb-2">
                        <div className={`${section.bgColor} p-2 rounded-lg`}>
                          <Icon className={`w-5 h-5 ${section.color}`} />
                        </div>
                        <h3 className="text-lg font-semibold group-hover:text-primary transition-colors">
                          {section.title}
                        </h3>
                      </div>
                      <p className="text-sm text-muted-foreground mb-4">
                        {section.description}
                      </p>
                      <div className="flex gap-4">
                        {section.stats.map((stat) => (
                          <div key={stat.label} className="flex items-center gap-2">
                            <div className="text-2xl font-bold">{stat.value}</div>
                            <div className="text-xs text-muted-foreground">
                              {stat.label}
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                    <div className="ml-4">
                      <ArrowRight className="w-5 h-5 text-muted-foreground group-hover:text-primary group-hover:translate-x-1 transition-all" />
                    </div>
                  </div>
                </div>
              </Link>
            );
          })}
        </div>
      </div>
    </div>
  );
}
