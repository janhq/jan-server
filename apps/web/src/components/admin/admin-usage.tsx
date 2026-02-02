import { useEffect, useState } from "react";
import {
  Activity,
  ArrowDown,
  ArrowUp,
  ChevronLeft,
  ChevronRight,
  Loader2,
  Search,
  Users,
  Zap,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { adminUsageService } from "@/services/admin-service";

type DateRangePreset = "7d" | "14d" | "30d" | "90d";

const presets: { value: DateRangePreset; label: string }[] = [
  { value: "7d", label: "7 days" },
  { value: "14d", label: "14 days" },
  { value: "30d", label: "30 days" },
  { value: "90d", label: "90 days" },
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
    case "90d":
      startDate.setDate(startDate.getDate() - 90);
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

export function AdminUsage() {
  const [dateRange, setDateRange] = useState<DateRangePreset>("30d");
  const [platformUsage, setPlatformUsage] =
    useState<PlatformUsageResponse | null>(null);
  const [usersUsage, setUsersUsage] = useState<UserUsageDetail[]>([]);
  const [totalUsers, setTotalUsers] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [currentPage, setCurrentPage] = useState(0);
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);
  const [selectedUserUsage, setSelectedUserUsage] =
    useState<UserUsageDetail | null>(null);
  const [isLoadingUserDetail, setIsLoadingUserDetail] = useState(false);

  const pageSize = 10;

  useEffect(() => {
    async function fetchData() {
      setIsLoading(true);
      setError(null);

      try {
        const { startDate, endDate } = getDateRange(dateRange);

        // Fetch platform usage and users usage in parallel
        const [platformData, usersData] = await Promise.all([
          adminUsageService.getPlatformUsage(startDate, endDate),
          adminUsageService.getAllUsersUsage(
            startDate,
            endDate,
            pageSize,
            currentPage * pageSize
          ),
        ]);

        setPlatformUsage(platformData);
        setTotalUsers(usersData.total);
        setUsersUsage(usersData.users);
      } catch (err) {
        console.error("Failed to fetch usage data:", err);
        setError("Failed to load usage data");
      } finally {
        setIsLoading(false);
      }
    }

    fetchData();
  }, [dateRange, currentPage]);

  useEffect(() => {
    async function fetchUserDetail() {
      if (!selectedUserId) {
        setSelectedUserUsage(null);
        return;
      }

      setIsLoadingUserDetail(true);
      try {
        const { startDate, endDate } = getDateRange(dateRange);
        const userDetail = await adminUsageService.getUserUsage(
          selectedUserId,
          startDate,
          endDate
        );
        setSelectedUserUsage(userDetail);
      } catch (err) {
        console.error("Failed to fetch user usage detail:", err);
      } finally {
        setIsLoadingUserDetail(false);
      }
    }

    fetchUserDetail();
  }, [selectedUserId, dateRange]);

  const filteredUsers = usersUsage.filter((user) => {
    if (!searchQuery) return true;
    const query = searchQuery.toLowerCase();
    return (
      user.user_id.toLowerCase().includes(query) ||
      user.email?.toLowerCase().includes(query) ||
      user.username?.toLowerCase().includes(query)
    );
  });

  const totalPages = Math.ceil(totalUsers / pageSize);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <Loader2 className="w-8 h-8 animate-spin mx-auto mb-4 text-primary" />
          <p className="text-sm text-muted-foreground">
            Loading usage data...
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

  const stats = platformUsage
    ? [
        {
          title: "Total Tokens",
          value: formatNumber(platformUsage.total_usage.total_tokens),
          description: "Platform-wide usage",
          icon: Zap,
          color: "text-blue-600 dark:text-blue-400",
          bgColor: "bg-blue-100 dark:bg-blue-900/30",
        },
        {
          title: "Input Tokens",
          value: formatNumber(platformUsage.total_usage.total_prompt_tokens),
          description: "Prompt tokens",
          icon: ArrowUp,
          color: "text-purple-600 dark:text-purple-400",
          bgColor: "bg-purple-100 dark:bg-purple-900/30",
        },
        {
          title: "Output Tokens",
          value: formatNumber(
            platformUsage.total_usage.total_completion_tokens
          ),
          description: "Completion tokens",
          icon: ArrowDown,
          color: "text-green-600 dark:text-green-400",
          bgColor: "bg-green-100 dark:bg-green-900/30",
        },
        {
          title: "Total Requests",
          value: formatNumber(platformUsage.total_usage.request_count),
          description: "API requests",
          icon: Activity,
          color: "text-orange-600 dark:text-orange-400",
          bgColor: "bg-orange-100 dark:bg-orange-900/30",
        },
        {
          title: "Active Users",
          value: totalUsers.toString(),
          description: "Users with usage",
          icon: Users,
          color: "text-pink-600 dark:text-pink-400",
          bgColor: "bg-pink-100 dark:bg-pink-900/30",
        },
      ]
    : [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Usage Analytics</h1>
          <p className="text-muted-foreground mt-1">
            Monitor platform-wide token usage and activity
          </p>
        </div>
        <div className="flex gap-2">
          {presets.map((preset) => (
            <Button
              key={preset.value}
              variant={dateRange === preset.value ? "default" : "outline"}
              size="sm"
              onClick={() => {
                setDateRange(preset.value);
                setCurrentPage(0);
              }}
            >
              {preset.label}
            </Button>
          ))}
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
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

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Usage by Model */}
        {platformUsage?.by_model && platformUsage.by_model.length > 0 && (
          <div className="rounded-lg border bg-card">
            <div className="border-b p-4">
              <h2 className="font-semibold">Usage by Model</h2>
            </div>
            <div className="divide-y max-h-[300px] overflow-y-auto">
              {platformUsage.by_model.map((model, idx) => {
                const percentage =
                  platformUsage.total_usage.total_tokens > 0
                    ? (model.total_tokens /
                        platformUsage.total_usage.total_tokens) *
                      100
                    : 0;
                return (
                  <div
                    key={model.model || idx}
                    className="flex items-center justify-between p-4"
                  >
                    <div>
                      <p className="font-medium">{model.model || "Unknown"}</p>
                      <p className="text-sm text-muted-foreground">
                        {formatNumber(model.request_count)} requests
                      </p>
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
        {platformUsage?.by_provider && platformUsage.by_provider.length > 0 && (
          <div className="rounded-lg border bg-card">
            <div className="border-b p-4">
              <h2 className="font-semibold">Usage by Provider</h2>
            </div>
            <div className="divide-y max-h-[300px] overflow-y-auto">
              {platformUsage.by_provider.map((provider, idx) => {
                const percentage =
                  platformUsage.total_usage.total_tokens > 0
                    ? (provider.total_tokens /
                        platformUsage.total_usage.total_tokens) *
                      100
                    : 0;
                return (
                  <div
                    key={provider.provider || idx}
                    className="flex items-center justify-between p-4"
                  >
                    <div>
                      <p className="font-medium">
                        {provider.provider || "Unknown"}
                      </p>
                      <p className="text-sm text-muted-foreground">
                        {formatNumber(provider.request_count)} requests
                      </p>
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
      </div>

      {/* Top Users */}
      {platformUsage?.top_users && platformUsage.top_users.length > 0 && (
        <div className="rounded-lg border bg-card">
          <div className="border-b p-4">
            <h2 className="font-semibold">Top Users</h2>
          </div>
          <div className="divide-y max-h-[300px] overflow-y-auto">
            {platformUsage.top_users.map((user, idx) => (
              <div
                key={user.user_id || idx}
                className="flex items-center justify-between p-4"
              >
                <div>
                  <p className="font-medium">
                    {user.email || `User ${user.user_id}`}
                  </p>
                </div>
                <div className="text-right">
                  <p className="font-medium">
                    {formatNumber(user.total_tokens)} tokens
                  </p>
                  <p className="text-sm text-muted-foreground">
                    {formatNumber(user.request_count)} requests
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Users Usage Table */}
      <div className="rounded-lg border bg-card">
        <div className="border-b p-4">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <h2 className="font-semibold">Usage by User</h2>
            <div className="relative w-full sm:w-64">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Search users..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9"
              />
            </div>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="px-4 py-3 text-left text-sm font-medium text-muted-foreground">
                  User
                </th>
                <th className="px-4 py-3 text-right text-sm font-medium text-muted-foreground">
                  Total Tokens
                </th>
                <th className="px-4 py-3 text-right text-sm font-medium text-muted-foreground hidden md:table-cell">
                  Input
                </th>
                <th className="px-4 py-3 text-right text-sm font-medium text-muted-foreground hidden md:table-cell">
                  Output
                </th>
                <th className="px-4 py-3 text-right text-sm font-medium text-muted-foreground">
                  Requests
                </th>
                <th className="px-4 py-3 text-center text-sm font-medium text-muted-foreground">
                  Details
                </th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {filteredUsers.length === 0 ? (
                <tr>
                  <td
                    colSpan={6}
                    className="px-4 py-8 text-center text-muted-foreground"
                  >
                    No users found with usage data
                  </td>
                </tr>
              ) : (
                filteredUsers.map((user) => (
                  <tr
                    key={user.user_id}
                    className={`hover:bg-muted/50 ${
                      selectedUserId === user.user_id ? "bg-muted/50" : ""
                    }`}
                  >
                    <td className="px-4 py-3">
                      <div>
                        <p className="font-medium">
                          {user.email ||
                            user.username ||
                            `User ${user.user_id}`}
                        </p>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-right font-medium">
                      {formatNumber(user.total_tokens)}
                    </td>
                    <td className="px-4 py-3 text-right text-muted-foreground hidden md:table-cell">
                      {formatNumber(user.total_prompt_tokens)}
                    </td>
                    <td className="px-4 py-3 text-right text-muted-foreground hidden md:table-cell">
                      {formatNumber(user.total_completion_tokens)}
                    </td>
                    <td className="px-4 py-3 text-right text-muted-foreground">
                      {formatNumber(user.request_count)}
                    </td>
                    <td className="px-4 py-3 text-center">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() =>
                          setSelectedUserId(
                            selectedUserId === user.user_id
                              ? null
                              : user.user_id
                          )
                        }
                      >
                        {selectedUserId === user.user_id ? "Hide" : "View"}
                      </Button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between border-t px-4 py-3">
            <p className="text-sm text-muted-foreground">
              Showing {currentPage * pageSize + 1} to{" "}
              {Math.min((currentPage + 1) * pageSize, totalUsers)} of{" "}
              {totalUsers} users
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={currentPage === 0}
                onClick={() => setCurrentPage((p) => p - 1)}
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={currentPage >= totalPages - 1}
                onClick={() => setCurrentPage((p) => p + 1)}
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        )}
      </div>

      {/* Selected User Detail */}
      {selectedUserId && (
        <div className="rounded-lg border bg-card">
          <div className="border-b p-4">
            <h2 className="font-semibold">
              User Detail:{" "}
              {usersUsage.find((u) => u.user_id === selectedUserId)?.email ||
                usersUsage.find((u) => u.user_id === selectedUserId)?.username ||
                `User ${selectedUserId}`}
            </h2>
          </div>

          {isLoadingUserDetail ? (
            <div className="flex items-center justify-center p-8">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : selectedUserUsage ? (
            <div className="p-4 space-y-6">
              {/* User Stats */}
              <div className="grid gap-4 md:grid-cols-3">
                <div className="rounded-lg border p-4">
                  <p className="text-sm text-muted-foreground">Total Tokens</p>
                  <p className="text-2xl font-bold">
                    {formatNumber(selectedUserUsage.total_tokens)}
                  </p>
                </div>
                <div className="rounded-lg border p-4">
                  <p className="text-sm text-muted-foreground">Input Tokens</p>
                  <p className="text-2xl font-bold">
                    {formatNumber(selectedUserUsage.total_prompt_tokens)}
                  </p>
                </div>
                <div className="rounded-lg border p-4">
                  <p className="text-sm text-muted-foreground">Output Tokens</p>
                  <p className="text-2xl font-bold">
                    {formatNumber(selectedUserUsage.total_completion_tokens)}
                  </p>
                </div>
              </div>

              <div className="grid gap-6 lg:grid-cols-2">
                {/* User Model Breakdown */}
                {selectedUserUsage.by_model &&
                  selectedUserUsage.by_model.length > 0 && (
                    <div className="rounded-lg border">
                      <div className="border-b p-4">
                        <h3 className="font-medium">By Model</h3>
                      </div>
                      <div className="divide-y max-h-[250px] overflow-y-auto">
                        {selectedUserUsage.by_model.map((model, idx) => (
                          <div
                            key={model.model || idx}
                            className="flex items-center justify-between p-3"
                          >
                            <div>
                              <p className="font-medium text-sm">
                                {model.model || "Unknown"}
                              </p>
                              <p className="text-xs text-muted-foreground">
                                {formatNumber(model.request_count)} requests
                              </p>
                            </div>
                            <div className="text-right">
                              <p className="font-medium text-sm">
                                {formatNumber(model.total_tokens)}
                              </p>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                {/* User Provider Breakdown */}
                {selectedUserUsage.by_provider &&
                  selectedUserUsage.by_provider.length > 0 && (
                    <div className="rounded-lg border">
                      <div className="border-b p-4">
                        <h3 className="font-medium">By Provider</h3>
                      </div>
                      <div className="divide-y max-h-[250px] overflow-y-auto">
                        {selectedUserUsage.by_provider.map((provider, idx) => (
                          <div
                            key={provider.provider || idx}
                            className="flex items-center justify-between p-3"
                          >
                            <div>
                              <p className="font-medium text-sm">
                                {provider.provider || "Unknown"}
                              </p>
                              <p className="text-xs text-muted-foreground">
                                {formatNumber(provider.request_count)} requests
                              </p>
                            </div>
                            <div className="text-right">
                              <p className="font-medium text-sm">
                                {formatNumber(provider.total_tokens)}
                              </p>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
              </div>
            </div>
          ) : (
            <div className="p-8 text-center text-muted-foreground">
              No usage data available for this user
            </div>
          )}
        </div>
      )}
    </div>
  );
}
