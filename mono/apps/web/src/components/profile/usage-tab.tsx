import { useEffect, useState } from "react";
import { Activity, ArrowDown, ArrowUp, Loader2, Zap } from "lucide-react";
import { usageService } from "@/services/admin-service";
import { Button } from "@/components/ui/button";

type DateRangePreset = "7d" | "14d" | "30d";

const presets: { value: DateRangePreset; label: string }[] = [
  { value: "7d", label: "Last 7 days" },
  { value: "14d", label: "Last 14 days" },
  { value: "30d", label: "Last 30 days" },
];

function getDateRange(preset: DateRangePreset): {
  startDate: string;
  endDate: string;
} {
  const endDate = new Date();
  const startDate = new Date();

  switch (preset) {
    case "7d":
      startDate.setDate(startDate.getDate() - 7);
      break;
    case "14d":
      startDate.setDate(startDate.getDate() - 14);
      break;
    case "30d":
      startDate.setDate(startDate.getDate() - 30);
      break;
  }

  return {
    startDate: startDate.toISOString().split("T")[0],
    endDate: endDate.toISOString().split("T")[0],
  };
}

function formatNumber(num: number): string {
  if (num >= 1_000_000) {
    return (num / 1_000_000).toFixed(1) + "M";
  }
  if (num >= 1_000) {
    return (num / 1_000).toFixed(1) + "K";
  }
  return num.toLocaleString();
}

export function UsageTab() {
  const [dateRange, setDateRange] = useState<DateRangePreset>("7d");
  const [usage, setUsage] = useState<UsageResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchUsage() {
      setIsLoading(true);
      setError(null);

      try {
        const { startDate, endDate } = getDateRange(dateRange);
        const data = await usageService.getMyUsage(startDate, endDate);
        setUsage(data);
      } catch (err) {
        console.error("Failed to fetch usage:", err);
        setError("Failed to load usage data");
      } finally {
        setIsLoading(false);
      }
    }

    fetchUsage();
  }, [dateRange]);

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-center text-destructive">
        {error}
      </div>
    );
  }

  const stats = usage
    ? [
        {
          title: "Total Tokens",
          value: formatNumber(usage.total_usage.total_tokens),
          description: "Total tokens used",
          icon: Zap,
          color: "text-blue-600 dark:text-blue-400",
          bgColor: "bg-blue-100 dark:bg-blue-900/30",
        },
        {
          title: "Input Tokens",
          value: formatNumber(usage.total_usage.total_prompt_tokens),
          description: "Prompt tokens",
          icon: ArrowUp,
          color: "text-purple-600 dark:text-purple-400",
          bgColor: "bg-purple-100 dark:bg-purple-900/30",
        },
        {
          title: "Output Tokens",
          value: formatNumber(usage.total_usage.total_completion_tokens),
          description: "Completion tokens",
          icon: ArrowDown,
          color: "text-green-600 dark:text-green-400",
          bgColor: "bg-green-100 dark:bg-green-900/30",
        },
        {
          title: "Requests",
          value: formatNumber(usage.total_usage.request_count),
          description: "Total API requests",
          icon: Activity,
          color: "text-orange-600 dark:text-orange-400",
          bgColor: "bg-orange-100 dark:bg-orange-900/30",
        },
      ]
    : [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <h3 className="text-lg font-medium">Token Usage</h3>
          <p className="text-sm text-muted-foreground">
            Monitor your API token usage over time
          </p>
        </div>
        <div className="flex gap-2">
          {presets.map((preset) => (
            <Button
              key={preset.value}
              variant={dateRange === preset.value ? "default" : "outline"}
              size="sm"
              onClick={() => setDateRange(preset.value)}
            >
              {preset.label}
            </Button>
          ))}
        </div>
      </div>

      {!usage || usage.total_usage.total_tokens === 0 ? (
        <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">
          No usage data available for this period
        </div>
      ) : (
        <>
          {/* Stats Cards */}
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {stats.map((stat) => (
              <div
                key={stat.title}
                className="rounded-lg border bg-card p-4 shadow-sm"
              >
                <div className="flex items-center justify-between">
                  <p className="text-sm font-medium text-muted-foreground">
                    {stat.title}
                  </p>
                  <div className={`rounded-full p-2 ${stat.bgColor}`}>
                    <stat.icon className={`h-4 w-4 ${stat.color}`} />
                  </div>
                </div>
                <div className="mt-2">
                  <p className="text-2xl font-bold">{stat.value}</p>
                  <p className="text-xs text-muted-foreground">
                    {stat.description}
                  </p>
                </div>
              </div>
            ))}
          </div>

          {/* Usage by Model */}
          {usage.by_model && usage.by_model.length > 0 && (
            <div className="rounded-lg border bg-card">
              <div className="border-b p-4">
                <h4 className="font-medium">Usage by Model</h4>
              </div>
              <div className="divide-y">
                {usage.by_model.map((model, idx) => {
                  const percentage =
                    usage.total_usage.total_tokens > 0
                      ? (model.total_tokens / usage.total_usage.total_tokens) *
                        100
                      : 0;
                  return (
                    <div
                      key={model.model || idx}
                      className="flex items-center justify-between p-4"
                    >
                      <div className="flex items-center gap-3">
                        <div className="flex-1">
                          <p className="font-medium">
                            {model.model || "Unknown"}
                          </p>
                          <p className="text-sm text-muted-foreground">
                            {formatNumber(model.request_count)} requests
                          </p>
                        </div>
                      </div>
                      <div className="text-right">
                        <p className="font-medium">
                          {formatNumber(model.total_tokens)} tokens
                        </p>
                        <p className="text-sm text-muted-foreground">
                          {percentage.toFixed(1)}%
                        </p>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {/* Usage by Provider */}
          {usage.by_provider && usage.by_provider.length > 0 && (
            <div className="rounded-lg border bg-card">
              <div className="border-b p-4">
                <h4 className="font-medium">Usage by Provider</h4>
              </div>
              <div className="divide-y">
                {usage.by_provider.map((provider, idx) => {
                  const percentage =
                    usage.total_usage.total_tokens > 0
                      ? (provider.total_tokens /
                          usage.total_usage.total_tokens) *
                        100
                      : 0;
                  return (
                    <div
                      key={provider.provider || idx}
                      className="flex items-center justify-between p-4"
                    >
                      <div className="flex items-center gap-3">
                        <div className="flex-1">
                          <p className="font-medium">
                            {provider.provider || "Unknown"}
                          </p>
                          <p className="text-sm text-muted-foreground">
                            {formatNumber(provider.request_count)} requests
                          </p>
                        </div>
                      </div>
                      <div className="text-right">
                        <p className="font-medium">
                          {formatNumber(provider.total_tokens)} tokens
                        </p>
                        <p className="text-sm text-muted-foreground">
                          {percentage.toFixed(1)}%
                        </p>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
