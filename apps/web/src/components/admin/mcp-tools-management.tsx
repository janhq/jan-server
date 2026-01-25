import { useEffect, useState } from "react";
import {
  Wrench,
  Loader2,
  RefreshCw,
  Search,
  MoreHorizontal,
  Pencil,
  Power,
  PowerOff,
  ChevronDown,
  ChevronRight,
  AlertTriangle,
  Copy,
  Check,
  Filter,
  Info,
  Shield,
} from "lucide-react";
import { Button } from "@janhq/interfaces/button";
import { Input } from "@janhq/interfaces/input";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@janhq/interfaces/dialog";
import {
  DropDrawer,
  DropDrawerContent,
  DropDrawerItem,
  DropDrawerTrigger,
} from "@janhq/interfaces/dropdrawer";
import { mcpToolService } from "@/services/admin-service";
import { cn } from "@/lib/utils";

type EditTabType = "general" | "restrictions";

export function MCPToolsManagement() {
  const [tools, setTools] = useState<MCPTool[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string>("");
  const [selectedStatus, setSelectedStatus] = useState<string>("");
  const [expandedTool, setExpandedTool] = useState<string | null>(null);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  // Dialog states
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [toolToEdit, setToolToEdit] = useState<MCPTool | null>(null);
  const [activeEditTab, setActiveEditTab] = useState<EditTabType>("general");

  // Form state
  const [formData, setFormData] = useState<UpdateMCPToolRequest>({
    description: "",
    category: "",
    is_active: true,
    disallowed_keywords: [],
  });
  const [keywordsInput, setKeywordsInput] = useState("");

  const [pagination, setPagination] = useState({
    page: 1,
    limit: 20,
    total: 0,
  });

  useEffect(() => {
    loadTools();
  }, [pagination.page, selectedCategory, selectedStatus]);

  async function loadTools() {
    try {
      setIsLoading(true);
      setError(null);
      const params: Record<string, unknown> = {
        page: pagination.page,
        limit: pagination.limit,
      };
      if (selectedCategory) {
        params.category = selectedCategory;
      }
      if (selectedStatus === "active") {
        params.is_active = true;
      } else if (selectedStatus === "inactive") {
        params.is_active = false;
      }
      const response = await mcpToolService.listMCPTools(params);
      setTools(response.data || []);
      setPagination((prev) => ({ ...prev, total: response.total || 0 }));
    } catch (err) {
      console.error("Failed to load tools:", err);
      setError("Failed to load MCP tools");
    } finally {
      setIsLoading(false);
    }
  }

  async function handleToggleActive(tool: MCPTool) {
    try {
      await mcpToolService.updateMCPTool(tool.public_id, {
        is_active: !tool.is_active,
      });
      await loadTools();
    } catch (err) {
      console.error("Failed to toggle tool status:", err);
    }
  }

  async function handleUpdate() {
    if (!toolToEdit) return;
    try {
      // Parse keywords from newline-separated input
      const keywords = keywordsInput
        .split("\n")
        .map((k) => k.trim())
        .filter((k) => k.length > 0);

      await mcpToolService.updateMCPTool(toolToEdit.public_id, {
        description: formData.description,
        category: formData.category,
        is_active: formData.is_active,
        disallowed_keywords: keywords,
      });
      setEditDialogOpen(false);
      setToolToEdit(null);
      resetForm();
      await loadTools();
    } catch (err) {
      console.error("Failed to update tool:", err);
    }
  }

  function resetForm() {
    setFormData({
      description: "",
      category: "",
      is_active: true,
      disallowed_keywords: [],
    });
    setKeywordsInput("");
    setActiveEditTab("general");
  }

  function openEditDialog(tool: MCPTool) {
    setToolToEdit(tool);
    setFormData({
      description: tool.description || "",
      category: tool.category || "",
      is_active: tool.is_active,
      disallowed_keywords: tool.disallowed_keywords || [],
    });
    // Convert to newline-separated for regex patterns support
    setKeywordsInput((tool.disallowed_keywords || []).join("\n"));
    setActiveEditTab("general");
    setEditDialogOpen(true);
  }

  function copyToClipboard(text: string, toolId: string) {
    navigator.clipboard.writeText(text);
    setCopiedId(toolId);
    setTimeout(() => setCopiedId(null), 2000);
  }

  const categories = [...new Set(tools.map((t) => t.category).filter(Boolean))].sort();

  const filteredTools = tools.filter(
    (t) =>
      t.name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      t.tool_key?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      t.description?.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Stats
  const activeCount = tools.filter((t) => t.is_active).length;
  const restrictedCount = tools.filter(
    (t) => t.disallowed_keywords && t.disallowed_keywords.length > 0
  ).length;

  const totalPages = Math.ceil(pagination.total / pagination.limit);

  if (isLoading && tools.length === 0) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <Loader2 className="w-8 h-8 animate-spin mx-auto mb-4 text-primary" />
          <p className="text-sm text-muted-foreground">Loading MCP tools...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-6">
        <h3 className="text-lg font-semibold text-destructive mb-2">Error</h3>
        <p className="text-sm text-muted-foreground">{error}</p>
        <Button onClick={loadTools} variant="outline" className="mt-4">
          Retry
        </Button>
      </div>
    );
  }

  const editTabs: { id: EditTabType; label: string; icon: React.ElementType }[] = [
    { id: "general", label: "General", icon: Info },
    { id: "restrictions", label: "Restrictions", icon: Shield },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">MCP Tools</h1>
        <p className="text-muted-foreground mt-2">
          Manage Model Context Protocol tools and their configurations
        </p>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-3 gap-4">
        <div className="bg-card rounded-lg border p-4">
          <div className="text-sm text-muted-foreground">Total Tools</div>
          <div className="text-2xl font-bold">{pagination.total}</div>
        </div>
        <div className="bg-card rounded-lg border p-4">
          <div className="text-sm text-muted-foreground">Active</div>
          <div className="text-2xl font-bold text-green-600">{activeCount}</div>
        </div>
        <div className="bg-card rounded-lg border p-4">
          <div className="text-sm text-muted-foreground">With Restrictions</div>
          <div className="text-2xl font-bold text-yellow-600">{restrictedCount}</div>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-4">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="Search tools..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>

        <select
          value={selectedCategory}
          onChange={(e) => {
            setSelectedCategory(e.target.value);
            setPagination((prev) => ({ ...prev, page: 1 }));
          }}
          className="flex h-10 rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="">All Categories</option>
          {categories.map((cat) => (
            <option key={cat} value={cat}>
              {cat}
            </option>
          ))}
        </select>

        <select
          value={selectedStatus}
          onChange={(e) => {
            setSelectedStatus(e.target.value);
            setPagination((prev) => ({ ...prev, page: 1 }));
          }}
          className="flex h-10 rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="">All Status</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
        </select>

        <Button variant="outline" onClick={loadTools}>
          <RefreshCw className="w-4 h-4 mr-2" />
          Refresh
        </Button>
      </div>

      <div className="text-sm text-muted-foreground">
        Showing {filteredTools.length} of {pagination.total} tools
      </div>

      <div className="space-y-3">
        {filteredTools.length === 0 ? (
          <div className="text-center py-12 text-muted-foreground border rounded-lg">
            {searchQuery ? "No tools match your search" : "No MCP tools found"}
          </div>
        ) : (
          filteredTools.map((tool) => (
            <div
              key={tool.id}
              className="bg-card rounded-lg border overflow-hidden"
            >
              <div className="p-4 flex items-start justify-between">
                <div
                  className="flex items-start gap-3 flex-1 cursor-pointer"
                  onClick={() =>
                    setExpandedTool(expandedTool === tool.id ? null : tool.id)
                  }
                >
                  <div className="bg-cyan-100 dark:bg-cyan-900/20 p-2 rounded mt-1">
                    <Wrench className="w-5 h-5 text-cyan-600" />
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2 flex-wrap">
                      <h3 className="font-semibold">{tool.name}</h3>
                      <span
                        className={`px-2 py-0.5 text-xs rounded-full ${
                          tool.is_active
                            ? "bg-green-100 text-green-700 dark:bg-green-900/20 dark:text-green-400"
                            : "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400"
                        }`}
                      >
                        {tool.is_active ? "Active" : "Inactive"}
                      </span>
                      {tool.disallowed_keywords && tool.disallowed_keywords.length > 0 && (
                        <span className="px-2 py-0.5 text-xs bg-yellow-100 text-yellow-700 dark:bg-yellow-900/20 dark:text-yellow-400 rounded-full flex items-center gap-1">
                          <AlertTriangle className="w-3 h-3" />
                          {tool.disallowed_keywords.length} restriction{tool.disallowed_keywords.length !== 1 ? "s" : ""}
                        </span>
                      )}
                      {tool.category && (
                        <span className="px-2 py-0.5 text-xs bg-blue-100 text-blue-700 dark:bg-blue-900/20 dark:text-blue-400 rounded">
                          {tool.category}
                        </span>
                      )}
                    </div>
                    <div className="text-sm text-muted-foreground mt-1 flex items-center gap-2">
                      <span className="font-mono text-xs">{tool.tool_key}</span>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          copyToClipboard(tool.public_id, tool.id);
                        }}
                        className="text-muted-foreground hover:text-foreground"
                        title="Copy ID"
                      >
                        {copiedId === tool.id ? (
                          <Check className="w-3 h-3 text-green-500" />
                        ) : (
                          <Copy className="w-3 h-3" />
                        )}
                      </button>
                    </div>
                    {tool.description && (
                      <p className="text-sm text-muted-foreground mt-1 line-clamp-2">
                        {tool.description}
                      </p>
                    )}
                  </div>
                  <div className="flex items-center">
                    {expandedTool === tool.id ? (
                      <ChevronDown className="w-5 h-5 text-muted-foreground" />
                    ) : (
                      <ChevronRight className="w-5 h-5 text-muted-foreground" />
                    )}
                  </div>
                </div>
                <div className="ml-4">
                  <DropDrawer>
                    <DropDrawerTrigger asChild>
                      <Button variant="ghost" size="sm">
                        <MoreHorizontal className="w-4 h-4" />
                      </Button>
                    </DropDrawerTrigger>
                    <DropDrawerContent className="w-48">
                      <DropDrawerItem onClick={() => openEditDialog(tool)}>
                        <div className="flex gap-2 items-center">
                          <Pencil className="w-4 h-4" />
                          <span>Edit</span>
                        </div>
                      </DropDrawerItem>
                      <DropDrawerItem onClick={() => handleToggleActive(tool)}>
                        <div className="flex gap-2 items-center">
                          {tool.is_active ? (
                            <PowerOff className="w-4 h-4" />
                          ) : (
                            <Power className="w-4 h-4" />
                          )}
                          <span>{tool.is_active ? "Deactivate" : "Activate"}</span>
                        </div>
                      </DropDrawerItem>
                      <DropDrawerItem
                        onClick={() => copyToClipboard(tool.public_id, tool.id)}
                      >
                        <div className="flex gap-2 items-center">
                          <Copy className="w-4 h-4" />
                          <span>Copy ID</span>
                        </div>
                      </DropDrawerItem>
                    </DropDrawerContent>
                  </DropDrawer>
                </div>
              </div>

              {expandedTool === tool.id && (
                <div className="border-t p-4 bg-muted/20">
                  <div className="space-y-4">
                    {tool.description && (
                      <div>
                        <div className="text-xs text-muted-foreground uppercase tracking-wider mb-2">
                          Description
                        </div>
                        <p className="text-sm">{tool.description}</p>
                      </div>
                    )}

                    {tool.disallowed_keywords && tool.disallowed_keywords.length > 0 && (
                      <div>
                        <div className="text-xs text-muted-foreground uppercase tracking-wider mb-2 flex items-center gap-2">
                          <Shield className="w-3 h-3" />
                          Disallowed Keywords / Patterns
                        </div>
                        <div className="space-y-1">
                          {tool.disallowed_keywords.map((keyword, idx) => (
                            <div
                              key={idx}
                              className="flex items-center gap-2"
                            >
                              <span className="px-2 py-1 text-xs bg-red-100 text-red-700 dark:bg-red-900/20 dark:text-red-400 rounded font-mono">
                                {keyword}
                              </span>
                              {keyword.includes("*") || keyword.includes("^") || keyword.includes("$") || keyword.includes("\\") ? (
                                <span className="text-xs text-muted-foreground">(regex pattern)</span>
                              ) : null}
                            </div>
                          ))}
                        </div>
                        <p className="text-xs text-muted-foreground mt-2">
                          Tool execution will be blocked if input matches these keywords or patterns
                        </p>
                      </div>
                    )}

                    {tool.metadata && Object.keys(tool.metadata).length > 0 && (
                      <div>
                        <div className="text-xs text-muted-foreground uppercase tracking-wider mb-2">
                          Metadata
                        </div>
                        <pre className="text-sm bg-muted p-3 rounded-md overflow-x-auto font-mono">
                          {JSON.stringify(tool.metadata, null, 2)}
                        </pre>
                      </div>
                    )}

                    <div className="grid grid-cols-2 gap-4 text-sm pt-2 border-t">
                      <div>
                        <span className="text-muted-foreground">Created: </span>
                        {new Date(tool.created_at).toLocaleDateString()}
                      </div>
                      <div>
                        <span className="text-muted-foreground">Updated: </span>
                        {new Date(tool.updated_at).toLocaleDateString()}
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          ))
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            Page {pagination.page} of {totalPages}
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPagination((prev) => ({ ...prev, page: prev.page - 1 }))}
              disabled={pagination.page <= 1}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPagination((prev) => ({ ...prev, page: prev.page + 1 }))}
              disabled={pagination.page >= totalPages}
            >
              Next
            </Button>
          </div>
        </div>
      )}

      {/* Edit Dialog with Tabs */}
      <Dialog
        open={editDialogOpen}
        onOpenChange={(open) => {
          if (!open) {
            setEditDialogOpen(false);
            setToolToEdit(null);
            resetForm();
          }
        }}
      >
        <DialogContent className="sm:max-w-[650px] max-h-[90vh] overflow-hidden flex flex-col">
          <DialogHeader>
            <DialogTitle>Edit MCP Tool</DialogTitle>
            <DialogDescription>
              Update the tool configuration. Tool key and name cannot be changed.
            </DialogDescription>
          </DialogHeader>

          {/* Tabs */}
          <div className="border-b">
            <nav className="flex space-x-4 px-1" aria-label="Tabs">
              {editTabs.map((tab) => {
                const Icon = tab.icon;
                return (
                  <button
                    key={tab.id}
                    onClick={() => setActiveEditTab(tab.id)}
                    className={cn(
                      "flex items-center gap-2 py-2 px-3 text-sm font-medium border-b-2 -mb-px transition-colors",
                      activeEditTab === tab.id
                        ? "border-primary text-primary"
                        : "border-transparent text-muted-foreground hover:text-foreground hover:border-muted-foreground/50"
                    )}
                  >
                    <Icon className="w-4 h-4" />
                    {tab.label}
                  </button>
                );
              })}
            </nav>
          </div>

          <div className="flex-1 overflow-y-auto py-4">
            {/* General Tab */}
            {activeEditTab === "general" && (
              <div className="grid gap-4">
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Tool Key</label>
                  <Input value={toolToEdit?.tool_key || ""} disabled className="font-mono" />
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Name</label>
                  <Input value={toolToEdit?.name || ""} disabled />
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Category</label>
                  <Input
                    value={formData.category}
                    onChange={(e) => setFormData({ ...formData, category: e.target.value })}
                    placeholder="search, code, utility, etc."
                  />
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Description</label>
                  <textarea
                    value={formData.description}
                    onChange={(e) =>
                      setFormData({ ...formData, description: e.target.value })
                    }
                    placeholder="Brief description of the tool functionality"
                    className="flex min-h-[100px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  />
                </div>
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="is_active"
                    checked={formData.is_active}
                    onChange={(e) =>
                      setFormData({ ...formData, is_active: e.target.checked })
                    }
                    className="rounded"
                  />
                  <label htmlFor="is_active" className="text-sm">
                    Active (tool can be used in conversations)
                  </label>
                </div>
              </div>
            )}

            {/* Restrictions Tab */}
            {activeEditTab === "restrictions" && (
              <div className="grid gap-4">
                <div className="p-4 bg-yellow-50 dark:bg-yellow-900/10 border border-yellow-200 dark:border-yellow-900/30 rounded-lg">
                  <div className="flex items-start gap-3">
                    <AlertTriangle className="w-5 h-5 text-yellow-600 shrink-0 mt-0.5" />
                    <div>
                      <div className="font-medium text-yellow-800 dark:text-yellow-400">
                        Content Restrictions
                      </div>
                      <p className="text-sm text-yellow-700 dark:text-yellow-500 mt-1">
                        Disallowed keywords prevent the tool from being executed when the input contains matching content.
                        This helps prevent misuse of the tool.
                      </p>
                    </div>
                  </div>
                </div>

                <div className="grid gap-2">
                  <label className="text-sm font-medium">
                    Disallowed Keywords / Patterns (one per line)
                  </label>
                  <textarea
                    value={keywordsInput}
                    onChange={(e) => setKeywordsInput(e.target.value)}
                    placeholder="password
secret
api_key
\b(SELECT|INSERT|UPDATE|DELETE)\b.*FROM
<script>.*</script>"
                    className="flex min-h-[200px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono"
                  />
                  <p className="text-xs text-muted-foreground">
                    Enter one keyword or regex pattern per line. Regex patterns will be matched against tool inputs.
                  </p>
                </div>

                <div className="p-3 bg-muted rounded-lg">
                  <div className="text-xs text-muted-foreground uppercase tracking-wider mb-2">
                    Pattern Examples
                  </div>
                  <div className="space-y-2 text-sm">
                    <div className="flex gap-2">
                      <code className="px-1.5 py-0.5 bg-background rounded text-xs">password</code>
                      <span className="text-muted-foreground">- Simple keyword match</span>
                    </div>
                    <div className="flex gap-2">
                      <code className="px-1.5 py-0.5 bg-background rounded text-xs">\bapi[_-]?key\b</code>
                      <span className="text-muted-foreground">- Regex for api_key, api-key, apikey</span>
                    </div>
                    <div className="flex gap-2">
                      <code className="px-1.5 py-0.5 bg-background rounded text-xs">{"<script>.*</script>"}</code>
                      <span className="text-muted-foreground">- Block script tags</span>
                    </div>
                  </div>
                </div>

                {keywordsInput && (
                  <div className="p-3 bg-muted rounded-lg">
                    <div className="text-xs text-muted-foreground uppercase tracking-wider mb-2">
                      Current Restrictions ({keywordsInput.split("\n").filter(k => k.trim()).length})
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {keywordsInput
                        .split("\n")
                        .filter((k) => k.trim())
                        .map((keyword, idx) => (
                          <span
                            key={idx}
                            className="px-2 py-1 text-xs bg-red-100 text-red-700 dark:bg-red-900/20 dark:text-red-400 rounded font-mono"
                          >
                            {keyword.trim()}
                          </span>
                        ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>

          <DialogFooter className="border-t pt-4">
            <DialogClose asChild>
              <Button variant="outline">Cancel</Button>
            </DialogClose>
            <Button onClick={handleUpdate}>Update</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
