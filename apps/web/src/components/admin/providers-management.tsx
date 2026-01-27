import { useEffect, useState } from "react";
import { Link } from "@tanstack/react-router";
import {
  ArrowLeft,
  Check,
  Copy,
  Database,
  Loader2,
  MoreHorizontal,
  Pencil,
  Plus,
  Power,
  PowerOff,
  RefreshCw,
  Search,
  Trash2,
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
import { providerManagementService } from "@/services/admin-service";

const PROVIDER_TYPES = [
  { value: "openai", label: "OpenAI" },
  { value: "anthropic", label: "Anthropic" },
  { value: "google", label: "Google" },
  { value: "azure", label: "Azure" },
  { value: "openrouter", label: "OpenRouter" },
  { value: "ollama", label: "Ollama" },
  { value: "local", label: "Local" },
  { value: "docling", label: "Docling" },
  { value: "custom", label: "Custom" },
];

const CATEGORIES = [
  { value: "llm", label: "LLM" },
  { value: "image", label: "Image" },
  { value: "ocr", label: "OCR" },
];

interface ProviderFormData {
  name: string;
  vendor: string;
  category: string;
  base_url: string;
  endpoints: string;
  image_edit_path: string;
  api_key: string;
  description: string;
  active: boolean;
  default_provider_image_generate: boolean;
  default_provider_image_edit: boolean;
  auto_enable_new_models: boolean;
}

const defaultFormData: ProviderFormData = {
  name: "",
  vendor: "",
  category: "llm",
  base_url: "",
  endpoints: "",
  image_edit_path: "",
  api_key: "",
  description: "",
  active: true,
  default_provider_image_generate: false,
  default_provider_image_edit: false,
  auto_enable_new_models: false,
};

export function ProvidersManagement() {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState<string>("");
  const [statusFilter, setStatusFilter] = useState<"all" | "active" | "inactive">("all");
  const [syncingProvider, setSyncingProvider] = useState<string | null>(null);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  // Dialog states
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [syncDialogOpen, setSyncDialogOpen] = useState(false);
  const [providerToEdit, setProviderToEdit] = useState<Provider | null>(null);
  const [providerToDelete, setProviderToDelete] = useState<Provider | null>(null);
  const [providerToSync, setProviderToSync] = useState<Provider | null>(null);
  const [deleteConfirmText, setDeleteConfirmText] = useState("");

  // Form state
  const [formData, setFormData] = useState<ProviderFormData>(defaultFormData);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    loadProviders();
  }, []);

  async function loadProviders() {
    try {
      setIsLoading(true);
      setError(null);
      const response = await providerManagementService.listProviders();
      setProviders(response.data || []);
    } catch (err) {
      console.error("Failed to load providers:", err);
      setError("Failed to load providers");
    } finally {
      setIsLoading(false);
    }
  }

  async function handleSync() {
    if (!providerToSync) return;

    try {
      setSyncingProvider(providerToSync.id);
      const result = await providerManagementService.syncProviderModels(
        providerToSync.id,
        formData.auto_enable_new_models
      );
      setSyncDialogOpen(false);
      setProviderToSync(null);
      setFormData({ ...formData, auto_enable_new_models: false });
      alert(`Successfully synced ${result.synced_models_count} models`);
      await loadProviders();
    } catch (err) {
      console.error("Failed to sync provider:", err);
      alert("Failed to sync provider models");
    } finally {
      setSyncingProvider(null);
    }
  }

  async function handleToggleActive(provider: Provider) {
    try {
      await providerManagementService.updateProvider(provider.id, {
        active: !provider.active,
      });
      await loadProviders();
    } catch (err) {
      console.error("Failed to toggle provider status:", err);
      alert("Failed to update provider status");
    }
  }

  async function handleDelete() {
    if (!providerToDelete || deleteConfirmText !== "Approved") return;

    try {
      setIsSubmitting(true);
      await providerManagementService.deleteProvider(providerToDelete.id);
      setDeleteDialogOpen(false);
      setProviderToDelete(null);
      setDeleteConfirmText("");
      await loadProviders();
    } catch (err) {
      console.error("Failed to delete provider:", err);
      alert("Failed to delete provider");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleCreate() {
    if (!formData.name || !formData.vendor) return;

    try {
      setIsSubmitting(true);

      // Parse endpoints - supports JSON array, newline-separated, or comma-separated
      let endpoints: Endpoint[] = [];
      if (formData.endpoints.trim()) {
        try {
          // Try JSON parse first
          const parsed = JSON.parse(formData.endpoints);
          if (Array.isArray(parsed)) {
            endpoints = parsed;
          } else {
            endpoints = [{ url: formData.endpoints }];
          }
        } catch {
          // Fallback: newline or comma-separated URLs
          endpoints = formData.endpoints
            .split(/[\n,]/)
            .map((url) => url.trim())
            .filter((url) => url)
            .map((url) => ({ url }));
        }
      }

      const createData: Record<string, unknown> = {
        name: formData.name,
        vendor: formData.vendor,
        category: formData.category || undefined,
        active: formData.active,
        default_provider_image_generate: formData.default_provider_image_generate,
        default_provider_image_edit: formData.default_provider_image_edit,
      };

      if (formData.base_url) {
        createData.base_url = formData.base_url;
      }
      if (endpoints.length > 0) {
        createData.endpoints = endpoints;
      }
      if (formData.api_key) {
        createData.api_key = formData.api_key;
      }
      // Build metadata object with optional fields
      const metadata: Record<string, string> = {};
      if (formData.description) {
        metadata.description = formData.description;
      }
      if (formData.image_edit_path) {
        metadata.image_edit_path = formData.image_edit_path;
      }
      if (Object.keys(metadata).length > 0) {
        createData.metadata = metadata;
      }

      await providerManagementService.createProvider(createData as Parameters<typeof providerManagementService.createProvider>[0]);
      setCreateDialogOpen(false);
      resetForm();
      await loadProviders();
    } catch (err) {
      console.error("Failed to create provider:", err);
      alert("Failed to create provider");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleUpdate() {
    if (!providerToEdit || !formData.name) return;

    try {
      setIsSubmitting(true);

      // Parse endpoints - supports JSON array, newline-separated, or comma-separated
      let endpoints: Endpoint[] | undefined;
      if (formData.endpoints.trim()) {
        try {
          const parsed = JSON.parse(formData.endpoints);
          if (Array.isArray(parsed)) {
            endpoints = parsed;
          } else {
            endpoints = [{ url: formData.endpoints }];
          }
        } catch {
          endpoints = formData.endpoints
            .split(/[\n,]/)
            .map((url) => url.trim())
            .filter((url) => url)
            .map((url) => ({ url }));
        }
      }

      const updateData: Record<string, unknown> = {
        name: formData.name,
        vendor: formData.vendor,
        category: formData.category || undefined,
        active: formData.active,
        default_provider_image_generate: formData.default_provider_image_generate,
        default_provider_image_edit: formData.default_provider_image_edit,
      };

      if (formData.base_url) {
        updateData.base_url = formData.base_url;
      }
      if (endpoints && endpoints.length > 0) {
        updateData.endpoints = endpoints;
      }
      if (formData.api_key) {
        updateData.api_key = formData.api_key;
      }
      // Build metadata object with optional fields
      const metadata: Record<string, string> = {};
      if (formData.description) {
        metadata.description = formData.description;
      }
      if (formData.image_edit_path) {
        metadata.image_edit_path = formData.image_edit_path;
      }
      if (Object.keys(metadata).length > 0) {
        updateData.metadata = metadata;
      }

      await providerManagementService.updateProvider(
        providerToEdit.id,
        updateData as Parameters<typeof providerManagementService.updateProvider>[1]
      );
      setEditDialogOpen(false);
      setProviderToEdit(null);
      resetForm();
      await loadProviders();
    } catch (err) {
      console.error("Failed to update provider:", err);
      alert("Failed to update provider");
    } finally {
      setIsSubmitting(false);
    }
  }

  function resetForm() {
    setFormData(defaultFormData);
  }

  function openEditDialog(provider: Provider) {
    setProviderToEdit(provider);
    const metadata = provider.metadata as Record<string, string> | undefined;
    setFormData({
      name: provider.name,
      vendor: provider.vendor,
      category: provider.category || "llm",
      base_url: provider.base_url || "",
      endpoints: provider.endpoints
        ? provider.endpoints.map((e) => e.url).join("\n")
        : "",
      image_edit_path: metadata?.image_edit_path || "",
      api_key: "",
      description: metadata?.description || "",
      active: provider.active,
      default_provider_image_generate: provider.default_provider_image_generate || false,
      default_provider_image_edit: provider.default_provider_image_edit || false,
      auto_enable_new_models: false,
    });
    setEditDialogOpen(true);
  }

  function openSyncDialog(provider: Provider) {
    setProviderToSync(provider);
    setFormData({ ...formData, auto_enable_new_models: false });
    setSyncDialogOpen(true);
  }

  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text);
    setCopiedId(text);
    setTimeout(() => setCopiedId(null), 2000);
  }

  // Filter providers
  const filteredProviders = providers.filter((p) => {
    const matchesSearch =
      p.name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      p.vendor?.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesType = !typeFilter || p.vendor?.toLowerCase() === typeFilter.toLowerCase();
    const matchesStatus =
      statusFilter === "all" ||
      (statusFilter === "active" && p.active) ||
      (statusFilter === "inactive" && !p.active);

    return matchesSearch && matchesType && matchesStatus;
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <Loader2 className="w-8 h-8 animate-spin mx-auto mb-4 text-primary" />
          <p className="text-sm text-muted-foreground">Loading providers...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-6">
        <h3 className="text-lg font-semibold text-destructive mb-2">Error</h3>
        <p className="text-sm text-muted-foreground">{error}</p>
        <Button onClick={loadProviders} variant="outline" className="mt-4">
          Retry
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link to="/admin/models">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back to Models
          </Button>
        </Link>
      </div>

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Model Providers</h1>
          <p className="text-muted-foreground mt-2">
            Manage model providers and their configurations
          </p>
        </div>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="w-4 h-4 mr-2" />
          Add Provider
        </Button>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-4">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="Search providers..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>

        <select
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value)}
          className="flex h-10 rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="">All Types</option>
          {PROVIDER_TYPES.map((type) => (
            <option key={type.value} value={type.value}>
              {type.label}
            </option>
          ))}
        </select>

        <div className="flex items-center gap-2 border rounded-md p-1">
          <Button
            variant={statusFilter === "all" ? "secondary" : "ghost"}
            size="sm"
            onClick={() => setStatusFilter("all")}
          >
            All
          </Button>
          <Button
            variant={statusFilter === "active" ? "secondary" : "ghost"}
            size="sm"
            onClick={() => setStatusFilter("active")}
          >
            Active
          </Button>
          <Button
            variant={statusFilter === "inactive" ? "secondary" : "ghost"}
            size="sm"
            onClick={() => setStatusFilter("inactive")}
          >
            Inactive
          </Button>
        </div>

        <Button variant="outline" onClick={loadProviders}>
          <RefreshCw className="w-4 h-4 mr-2" />
          Refresh
        </Button>

        {(searchQuery || typeFilter || statusFilter !== "all") && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              setSearchQuery("");
              setTypeFilter("");
              setStatusFilter("all");
            }}
          >
            Clear Filters
          </Button>
        )}
      </div>

      <div className="text-sm text-muted-foreground">
        Showing {filteredProviders.length} of {providers.length} providers
      </div>

      {/* Provider Grid */}
      <div className="grid gap-4">
        {filteredProviders.length === 0 ? (
          <div className="text-center py-12 text-muted-foreground border rounded-lg">
            {searchQuery || typeFilter || statusFilter !== "all"
              ? "No providers match your filters"
              : "No providers configured"}
          </div>
        ) : (
          filteredProviders.map((provider) => (
            <div
              key={provider.id}
              className="bg-card rounded-lg border p-6 hover:shadow-sm transition-shadow"
            >
              <div className="flex items-start justify-between">
                <div className="flex items-start gap-4">
                  <div className="bg-purple-100 dark:bg-purple-900/20 p-3 rounded-lg">
                    <Database className="w-6 h-6 text-purple-600" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2 flex-wrap">
                      <h3 className="text-lg font-semibold">{provider.name}</h3>
                      <span
                        className={`px-2 py-0.5 text-xs rounded-full ${
                          provider.active
                            ? "bg-green-100 text-green-700 dark:bg-green-900/20 dark:text-green-400"
                            : "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400"
                        }`}
                      >
                        {provider.active ? "Active" : "Inactive"}
                      </span>
                      {provider.category && (
                        <span className="px-2 py-0.5 text-xs rounded-full bg-blue-100 text-blue-700 dark:bg-blue-900/20 dark:text-blue-400">
                          {provider.category.toUpperCase()}
                        </span>
                      )}
                      {provider.default_provider_image_generate && (
                        <span className="px-2 py-0.5 text-xs rounded-full bg-orange-100 text-orange-700 dark:bg-orange-900/20 dark:text-orange-400">
                          Default Image Gen
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground mt-1">
                      Vendor: {provider.vendor}
                    </p>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="text-xs text-muted-foreground font-mono">
                        ID: {provider.id}
                      </span>
                      <button
                        onClick={() => copyToClipboard(provider.id)}
                        className="p-0.5 hover:bg-accent rounded transition-colors"
                        title="Copy ID"
                      >
                        {copiedId === provider.id ? (
                          <Check className="w-3 h-3 text-green-500" />
                        ) : (
                          <Copy className="w-3 h-3 text-muted-foreground" />
                        )}
                      </button>
                    </div>
                    {provider.endpoints && provider.endpoints.length > 0 && (
                      <p className="text-xs text-muted-foreground mt-1">
                        Endpoints: {provider.endpoints.slice(0, 3).map((e) => e.url).join(", ")}
                        {provider.endpoints.length > 3 && ` +${provider.endpoints.length - 3} more`}
                      </p>
                    )}
                    {provider.model_count !== undefined && (
                      <p className="text-sm text-muted-foreground mt-1">
                        Models: {provider.model_active_count || 0} active / {provider.model_count} total
                      </p>
                    )}
                    {provider.updated_at && (
                      <p className="text-xs text-muted-foreground mt-1">
                        Updated: {new Date(provider.updated_at).toLocaleDateString()}
                      </p>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => openSyncDialog(provider)}
                    disabled={syncingProvider === provider.id}
                  >
                    {syncingProvider === provider.id ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                      <RefreshCw className="w-4 h-4" />
                    )}
                    <span className="ml-2">Sync</span>
                  </Button>
                  <DropDrawer>
                    <DropDrawerTrigger asChild>
                      <Button variant="ghost" size="sm">
                        <MoreHorizontal className="w-4 h-4" />
                      </Button>
                    </DropDrawerTrigger>
                    <DropDrawerContent className="w-48">
                      <DropDrawerItem onClick={() => openEditDialog(provider)}>
                        <div className="flex gap-2 items-center">
                          <Pencil className="w-4 h-4" />
                          <span>Edit</span>
                        </div>
                      </DropDrawerItem>
                      <DropDrawerItem onClick={() => handleToggleActive(provider)}>
                        <div className="flex gap-2 items-center">
                          {provider.active ? (
                            <PowerOff className="w-4 h-4" />
                          ) : (
                            <Power className="w-4 h-4" />
                          )}
                          <span>{provider.active ? "Deactivate" : "Activate"}</span>
                        </div>
                      </DropDrawerItem>
                      <DropDrawerItem
                        variant="destructive"
                        onClick={() => {
                          setProviderToDelete(provider);
                          setDeleteConfirmText("");
                          setDeleteDialogOpen(true);
                        }}
                      >
                        <div className="flex gap-2 items-center">
                          <Trash2 className="w-4 h-4" />
                          <span>Delete</span>
                        </div>
                      </DropDrawerItem>
                    </DropDrawerContent>
                  </DropDrawer>
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Create/Edit Provider Dialog */}
      <Dialog
        open={createDialogOpen || editDialogOpen}
        onOpenChange={(open) => {
          if (!open) {
            setCreateDialogOpen(false);
            setEditDialogOpen(false);
            setProviderToEdit(null);
            resetForm();
          }
        }}
      >
        <DialogContent className="sm:max-w-[700px] max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              {editDialogOpen ? "Edit Provider" : "Add Provider"}
            </DialogTitle>
            <DialogDescription>
              {editDialogOpen
                ? "Update the provider configuration."
                : "Add a new model provider to your configuration."}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <label className="text-sm font-medium">Provider Name</label>
              <Input
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
                placeholder="vLLM Provider"
              />
            </div>

            <div className="grid gap-2">
              <label className="text-sm font-medium">Vendor</label>
              <Input
                value={formData.vendor}
                onChange={(e) =>
                  setFormData({ ...formData, vendor: e.target.value })
                }
                placeholder="jan"
              />
            </div>

            <div className="grid gap-2">
              <label className="text-sm font-medium">Category</label>
              <select
                value={formData.category}
                onChange={(e) =>
                  setFormData({ ...formData, category: e.target.value })
                }
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              >
                {CATEGORIES.map((cat) => (
                  <option key={cat.value} value={cat.value}>
                    {cat.label}
                  </option>
                ))}
              </select>
            </div>

            <div className="grid gap-2">
              <label className="text-sm font-medium">Base URL</label>
              <Input
                value={formData.base_url}
                onChange={(e) =>
                  setFormData({ ...formData, base_url: e.target.value })
                }
                placeholder="https://inference.jan.ai/v1"
              />
              <p className="text-xs text-muted-foreground">
                Optional when endpoints are provided.
              </p>
            </div>

            <div className="grid gap-2">
              <label className="text-sm font-medium">
                Endpoints (one per line or JSON array)
              </label>
              <textarea
                value={formData.endpoints}
                onChange={(e) =>
                  setFormData({ ...formData, endpoints: e.target.value })
                }
                placeholder={`${formData.base_url || "https://api.example.com/v1"}\nhttps://backup.example.com/v1`}
                rows={3}
                className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono"
              />
              <p className="text-xs text-muted-foreground">
                Enter one endpoint per line, or use comma-separated values. Leave blank to use Base URL fallback.
              </p>
            </div>

            <div className="grid gap-2">
              <label className="text-sm font-medium">
                Image Edit Path (Optional)
              </label>
              <Input
                value={formData.image_edit_path}
                onChange={(e) =>
                  setFormData({ ...formData, image_edit_path: e.target.value })
                }
                placeholder="/v1/images/edits or full URL"
              />
              <p className="text-xs text-muted-foreground">
                Overrides the edit endpoint for this provider.
              </p>
            </div>

            <div className="grid gap-2">
              <label className="text-sm font-medium">API Key</label>
              <Input
                type="password"
                value={formData.api_key}
                onChange={(e) =>
                  setFormData({ ...formData, api_key: e.target.value })
                }
                placeholder={editDialogOpen ? "Leave empty to keep current" : "sk-..."}
              />
            </div>

            <div className="grid gap-2">
              <label className="text-sm font-medium">Description</label>
              <textarea
                value={formData.description}
                onChange={(e) =>
                  setFormData({ ...formData, description: e.target.value })
                }
                placeholder="Optional description for this provider"
                className="flex min-h-[60px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              />
            </div>

            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="active"
                    checked={formData.active}
                    onChange={(e) =>
                      setFormData({ ...formData, active: e.target.checked })
                    }
                    className="rounded"
                  />
                  <label htmlFor="active" className="text-sm">
                    Active
                  </label>
                </div>
                <span className="text-xs text-muted-foreground">
                  Enable this provider
                </span>
              </div>

              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="default_image_gen"
                    checked={formData.default_provider_image_generate}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        default_provider_image_generate: e.target.checked,
                      })
                    }
                    className="rounded"
                  />
                  <label htmlFor="default_image_gen" className="text-sm">
                    Default for Image Generate
                  </label>
                </div>
                <span className="text-xs text-muted-foreground">
                  Use as default for /v1/images/generations
                </span>
              </div>

              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="default_image_edit"
                    checked={formData.default_provider_image_edit}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        default_provider_image_edit: e.target.checked,
                      })
                    }
                    className="rounded"
                  />
                  <label htmlFor="default_image_edit" className="text-sm">
                    Default for Image Edit
                  </label>
                </div>
                <span className="text-xs text-muted-foreground">
                  Use as default for /v1/images/edits
                </span>
              </div>
            </div>

            {/* Provider ID (read-only, only shown when editing) */}
            {editDialogOpen && providerToEdit && (
              <div className="pt-4 border-t">
                <p className="text-sm text-muted-foreground">
                  <span className="font-medium">Provider ID:</span>{" "}
                  <span className="font-mono">{providerToEdit.id}</span>
                </p>
              </div>
            )}
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Cancel</Button>
            </DialogClose>
            <Button
              onClick={editDialogOpen ? handleUpdate : handleCreate}
              disabled={!formData.name || !formData.vendor || isSubmitting}
            >
              {isSubmitting ? (
                <Loader2 className="w-4 h-4 animate-spin mr-2" />
              ) : null}
              {editDialogOpen ? "Save Changes" : "Add Provider"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Sync Dialog */}
      <Dialog open={syncDialogOpen} onOpenChange={setSyncDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>Sync Provider Models</DialogTitle>
            <DialogDescription>
              Fetch and synchronize models from{" "}
              <span className="font-semibold">{providerToSync?.name}</span>.
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="auto_enable"
                checked={formData.auto_enable_new_models}
                onChange={(e) =>
                  setFormData({
                    ...formData,
                    auto_enable_new_models: e.target.checked,
                  })
                }
                className="rounded"
              />
              <label htmlFor="auto_enable" className="text-sm">
                Auto-enable newly synced models
              </label>
            </div>
            <p className="text-xs text-muted-foreground mt-2">
              When enabled, new models discovered during sync will be automatically activated.
            </p>
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Cancel</Button>
            </DialogClose>
            <Button
              onClick={handleSync}
              disabled={syncingProvider === providerToSync?.id}
            >
              {syncingProvider === providerToSync?.id ? (
                <Loader2 className="w-4 h-4 animate-spin mr-2" />
              ) : (
                <RefreshCw className="w-4 h-4 mr-2" />
              )}
              Sync Models
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>Delete Provider</DialogTitle>
            <DialogDescription>
              This will permanently delete{" "}
              <span className="font-semibold">{providerToDelete?.name}</span> and
              all {providerToDelete?.model_count || 0} associated models. This action
              cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <label className="text-sm font-medium">
              Type "Approved" to confirm deletion
            </label>
            <Input
              value={deleteConfirmText}
              onChange={(e) => setDeleteConfirmText(e.target.value)}
              placeholder="Approved"
              className="mt-2"
            />
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Cancel</Button>
            </DialogClose>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteConfirmText !== "Approved" || isSubmitting}
            >
              {isSubmitting ? (
                <Loader2 className="w-4 h-4 animate-spin mr-2" />
              ) : null}
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
