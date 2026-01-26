import { useEffect, useState } from "react";
import { Link } from "@tanstack/react-router";
import {
  ArrowLeft,
  Box,
  Check,
  ChevronLeft,
  ChevronRight,
  ExternalLink,
  Loader2,
  MoreHorizontal,
  Pencil,
  Power,
  PowerOff,
  RefreshCw,
  Search,
  Square,
  CheckSquare,
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
import {
  providerModelService,
  providerManagementService,
} from "@/services/admin-service";

const CATEGORIES = [
  { value: "jan", label: "Jan" },
  { value: "legacy", label: "Legacy" },
];

interface ModelFormData {
  model_display_name: string;
  category: string;
  category_order_number: number | undefined;
  model_order_number: number | undefined;
  pricing_prompt: number | undefined;
  pricing_completion: number | undefined;
  context_length: number | undefined;
  max_completion_tokens: number | undefined;
  supports_images: boolean;
  supports_audio: boolean;
  supports_video: boolean;
  supports_reasoning: boolean;
  supports_embeddings: boolean;
  supports_tools: boolean;
  supports_browser: boolean;
  instruct_model_public_id: string;
  active: boolean;
}

const defaultFormData: ModelFormData = {
  model_display_name: "",
  category: "jan",
  category_order_number: undefined,
  model_order_number: undefined,
  pricing_prompt: undefined,
  pricing_completion: undefined,
  context_length: undefined,
  max_completion_tokens: undefined,
  supports_images: false,
  supports_audio: false,
  supports_video: false,
  supports_reasoning: false,
  supports_embeddings: false,
  supports_tools: false,
  supports_browser: false,
  instruct_model_public_id: "",
  active: true,
};

export function ProviderModelsManagement() {
  const [models, setModels] = useState<ProviderModel[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedProvider, setSelectedProvider] = useState<string>("");
  const [activeFilter, setActiveFilter] = useState<"all" | "active" | "inactive">("all");
  const [supportsImagesFilter, setSupportsImagesFilter] = useState(false);
  const [pagination, setPagination] = useState({
    page: 1,
    limit: 20,
    total: 0,
  });

  // Selection state for bulk operations
  const [selectedModels, setSelectedModels] = useState<Set<string>>(new Set());

  // Dialog states
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [modelToEdit, setModelToEdit] = useState<ProviderModel | null>(null);

  // Form state
  const [formData, setFormData] = useState<ModelFormData>(defaultFormData);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isBulkUpdating, setIsBulkUpdating] = useState(false);

  useEffect(() => {
    loadProviders();
  }, []);

  useEffect(() => {
    loadModels();
  }, [pagination.page, selectedProvider, activeFilter, supportsImagesFilter]);

  async function loadProviders() {
    try {
      const response = await providerManagementService.listProviders();
      setProviders(response.data || []);
    } catch (err) {
      console.error("Failed to load providers:", err);
    }
  }

  async function loadModels() {
    try {
      setIsLoading(true);
      setError(null);
      const params: Record<string, unknown> = {
        page: pagination.page,
        limit: pagination.limit,
      };
      if (selectedProvider) {
        params.provider_id = selectedProvider;
      }
      if (activeFilter !== "all") {
        params.active = activeFilter === "active";
      }
      if (supportsImagesFilter) {
        params.supports_images = true;
      }
      const response = await providerModelService.listProviderModels(params);
      setModels(response.data || []);
      setPagination((prev) => ({ ...prev, total: response.total || 0 }));
      // Clear selection when page changes
      setSelectedModels(new Set());
    } catch (err) {
      console.error("Failed to load models:", err);
      setError("Failed to load provider models");
    } finally {
      setIsLoading(false);
    }
  }

  async function handleToggleActive(model: ProviderModel) {
    try {
      await providerModelService.updateProviderModel(model.id, {
        active: !model.active,
      });
      await loadModels();
    } catch (err) {
      console.error("Failed to toggle model status:", err);
      alert("Failed to update model status");
    }
  }

  async function handleUpdate() {
    if (!modelToEdit) return;

    try {
      setIsSubmitting(true);

      const updateData: Partial<ProviderModel> = {
        model_display_name: formData.model_display_name,
        category: formData.category,
        category_order_number: formData.category_order_number,
        model_order_number: formData.model_order_number,
        pricing: {
          prompt: formData.pricing_prompt,
          completion: formData.pricing_completion,
        },
        token_limits: {
          context_length: formData.context_length,
          max_completion_tokens: formData.max_completion_tokens,
        },
        supports_images: formData.supports_images,
        supports_audio: formData.supports_audio,
        supports_video: formData.supports_video,
        supports_reasoning: formData.supports_reasoning,
        supports_embeddings: formData.supports_embeddings,
        supports_tools: formData.supports_tools,
        supports_browser: formData.supports_browser,
        instruct_model_public_id: formData.instruct_model_public_id || undefined,
        active: formData.active,
      };

      await providerModelService.updateProviderModel(modelToEdit.id, updateData);
      setEditDialogOpen(false);
      setModelToEdit(null);
      resetForm();
      await loadModels();
    } catch (err) {
      console.error("Failed to update model:", err);
      alert("Failed to update model");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleBulkActivate() {
    if (selectedModels.size === 0) return;

    try {
      setIsBulkUpdating(true);
      // Update each selected model
      const promises = Array.from(selectedModels).map((id) =>
        providerModelService.updateProviderModel(id, { active: true })
      );
      await Promise.all(promises);
      setSelectedModels(new Set());
      await loadModels();
    } catch (err) {
      console.error("Failed to bulk activate:", err);
      alert("Failed to activate some models");
    } finally {
      setIsBulkUpdating(false);
    }
  }

  async function handleBulkDeactivate() {
    if (selectedModels.size === 0) return;

    try {
      setIsBulkUpdating(true);
      const promises = Array.from(selectedModels).map((id) =>
        providerModelService.updateProviderModel(id, { active: false })
      );
      await Promise.all(promises);
      setSelectedModels(new Set());
      await loadModels();
    } catch (err) {
      console.error("Failed to bulk deactivate:", err);
      alert("Failed to deactivate some models");
    } finally {
      setIsBulkUpdating(false);
    }
  }

  function resetForm() {
    setFormData(defaultFormData);
  }

  function openEditDialog(model: ProviderModel) {
    setModelToEdit(model);
    setFormData({
      model_display_name: model.model_display_name,
      category: model.category || "jan",
      category_order_number: model.category_order_number,
      model_order_number: model.model_order_number,
      pricing_prompt: model.pricing?.prompt,
      pricing_completion: model.pricing?.completion,
      context_length: model.token_limits?.context_length,
      max_completion_tokens: model.token_limits?.max_completion_tokens,
      supports_images: model.supports_images || false,
      supports_audio: model.supports_audio || false,
      supports_video: model.supports_video || false,
      supports_reasoning: model.supports_reasoning || false,
      supports_embeddings: model.supports_embeddings || false,
      supports_tools: model.supports_tools || false,
      supports_browser: model.supports_browser || false,
      instruct_model_public_id: model.instruct_model_public_id || "",
      active: model.active,
    });
    setEditDialogOpen(true);
  }

  function toggleModelSelection(modelId: string) {
    const newSelected = new Set(selectedModels);
    if (newSelected.has(modelId)) {
      newSelected.delete(modelId);
    } else {
      newSelected.add(modelId);
    }
    setSelectedModels(newSelected);
  }

  function selectAllModels() {
    setSelectedModels(new Set(models.map((m) => m.id)));
  }

  function deselectAllModels() {
    setSelectedModels(new Set());
  }

  const filteredModels = models.filter(
    (m) =>
      m.model_display_name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      m.model_public_id?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      m.provider_vendor?.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const totalPages = Math.ceil(pagination.total / pagination.limit);

  if (isLoading && models.length === 0) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <Loader2 className="w-8 h-8 animate-spin mx-auto mb-4 text-primary" />
          <p className="text-sm text-muted-foreground">Loading models...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-6">
        <h3 className="text-lg font-semibold text-destructive mb-2">Error</h3>
        <p className="text-sm text-muted-foreground">{error}</p>
        <Button onClick={loadModels} variant="outline" className="mt-4">
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

      <div>
        <h1 className="text-3xl font-bold tracking-tight">Provider Models</h1>
        <p className="text-muted-foreground mt-2">
          Browse and manage individual models from all providers
        </p>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-4">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="Search models..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>

        <select
          value={selectedProvider}
          onChange={(e) => {
            setSelectedProvider(e.target.value);
            setPagination((prev) => ({ ...prev, page: 1 }));
          }}
          className="flex h-10 rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <option value="">All Providers</option>
          {providers.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>

        <div className="flex items-center gap-2 border rounded-md p-1">
          <Button
            variant={activeFilter === "all" ? "secondary" : "ghost"}
            size="sm"
            onClick={() => {
              setActiveFilter("all");
              setPagination((prev) => ({ ...prev, page: 1 }));
            }}
          >
            All
          </Button>
          <Button
            variant={activeFilter === "active" ? "secondary" : "ghost"}
            size="sm"
            onClick={() => {
              setActiveFilter("active");
              setPagination((prev) => ({ ...prev, page: 1 }));
            }}
          >
            Active
          </Button>
          <Button
            variant={activeFilter === "inactive" ? "secondary" : "ghost"}
            size="sm"
            onClick={() => {
              setActiveFilter("inactive");
              setPagination((prev) => ({ ...prev, page: 1 }));
            }}
          >
            Inactive
          </Button>
        </div>

        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            id="supportsImages"
            checked={supportsImagesFilter}
            onChange={(e) => {
              setSupportsImagesFilter(e.target.checked);
              setPagination((prev) => ({ ...prev, page: 1 }));
            }}
            className="rounded"
          />
          <label htmlFor="supportsImages" className="text-sm cursor-pointer">
            Vision Only
          </label>
        </div>

        <Button variant="outline" onClick={loadModels}>
          <RefreshCw className="w-4 h-4 mr-2" />
          Refresh
        </Button>

        {(searchQuery || selectedProvider || activeFilter !== "all" || supportsImagesFilter) && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              setSearchQuery("");
              setSelectedProvider("");
              setActiveFilter("all");
              setSupportsImagesFilter(false);
              setPagination((prev) => ({ ...prev, page: 1 }));
            }}
          >
            Clear Filters
          </Button>
        )}
      </div>

      {/* Bulk Actions */}
      {selectedModels.size > 0 && (
        <div className="flex items-center gap-4 p-3 bg-muted rounded-lg">
          <span className="text-sm font-medium">
            {selectedModels.size} model{selectedModels.size !== 1 ? "s" : ""} selected
          </span>
          <div className="flex gap-2">
            <Button
              size="sm"
              variant="outline"
              onClick={handleBulkActivate}
              disabled={isBulkUpdating}
            >
              {isBulkUpdating ? (
                <Loader2 className="w-4 h-4 animate-spin mr-2" />
              ) : (
                <Power className="w-4 h-4 mr-2" />
              )}
              Activate Selected
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={handleBulkDeactivate}
              disabled={isBulkUpdating}
            >
              {isBulkUpdating ? (
                <Loader2 className="w-4 h-4 animate-spin mr-2" />
              ) : (
                <PowerOff className="w-4 h-4 mr-2" />
              )}
              Deactivate Selected
            </Button>
            <Button size="sm" variant="ghost" onClick={deselectAllModels}>
              Deselect All
            </Button>
          </div>
        </div>
      )}

      <div className="flex items-center justify-between">
        <div className="text-sm text-muted-foreground">
          Showing {filteredModels.length} of {pagination.total} models
        </div>
        {models.length > 0 && selectedModels.size === 0 && (
          <Button size="sm" variant="ghost" onClick={selectAllModels}>
            <CheckSquare className="w-4 h-4 mr-2" />
            Select All
          </Button>
        )}
      </div>

      {/* Models Table */}
      <div className="border rounded-lg overflow-hidden">
        <table className="w-full">
          <thead className="bg-muted/50">
            <tr>
              <th className="w-10 p-4"></th>
              <th className="text-left p-4 font-medium">Model</th>
              <th className="text-left p-4 font-medium">Provider</th>
              <th className="text-left p-4 font-medium">Status</th>
              <th className="text-left p-4 font-medium">Category</th>
              <th className="text-left p-4 font-medium">Pricing</th>
              <th className="text-left p-4 font-medium">Context</th>
              <th className="text-right p-4 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {filteredModels.length === 0 ? (
              <tr>
                <td colSpan={8} className="text-center py-12 text-muted-foreground">
                  {searchQuery ? "No models match your search" : "No models found"}
                </td>
              </tr>
            ) : (
              filteredModels.map((model) => (
                <tr key={model.id} className="hover:bg-muted/30">
                  <td className="p-4">
                    <button
                      onClick={() => toggleModelSelection(model.id)}
                      className="p-1 hover:bg-accent rounded transition-colors"
                    >
                      {selectedModels.has(model.id) ? (
                        <CheckSquare className="w-4 h-4 text-primary" />
                      ) : (
                        <Square className="w-4 h-4 text-muted-foreground" />
                      )}
                    </button>
                  </td>
                  <td className="p-4">
                    <div className="flex items-center gap-3">
                      <div className="bg-green-100 dark:bg-green-900/20 p-2 rounded">
                        <Box className="w-4 h-4 text-green-600" />
                      </div>
                      <div>
                        <div className="font-medium">
                          <code className="text-sm bg-muted px-1.5 py-0.5 rounded">
                            {model.model_public_id}
                          </code>
                        </div>
                        {model.model_display_name && model.model_display_name !== model.model_public_id && (
                          <div className="text-sm text-muted-foreground mt-0.5">
                            {model.model_display_name}
                          </div>
                        )}
                        {model.model_catalog_id && (
                          <Link
                            to="/admin/models/catalogs"
                            className="text-xs text-primary hover:underline inline-flex items-center gap-1 mt-1"
                          >
                            View Catalog
                            <ExternalLink className="w-3 h-3" />
                          </Link>
                        )}
                      </div>
                    </div>
                  </td>
                  <td className="p-4 text-muted-foreground">
                    {model.provider_vendor || "Unknown"}
                  </td>
                  <td className="p-4">
                    <span
                      className={`px-2 py-0.5 text-xs rounded-full ${
                        model.active
                          ? "bg-green-100 text-green-700 dark:bg-green-900/20 dark:text-green-400"
                          : "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400"
                      }`}
                    >
                      {model.active ? "Active" : "Inactive"}
                    </span>
                  </td>
                  <td className="p-4">
                    <span className="px-2 py-0.5 text-xs rounded-full bg-blue-100 text-blue-700 dark:bg-blue-900/20 dark:text-blue-400">
                      {model.category || "-"}
                    </span>
                  </td>
                  <td className="p-4 text-sm text-muted-foreground">
                    {model.pricing?.prompt !== undefined ? (
                      <div>
                        <div>${model.pricing.prompt.toFixed(2)}/M in</div>
                        <div>${model.pricing.completion?.toFixed(2) || "?"}/M out</div>
                      </div>
                    ) : (
                      "-"
                    )}
                  </td>
                  <td className="p-4 text-muted-foreground">
                    {model.token_limits?.context_length
                      ? `${(model.token_limits.context_length / 1000).toFixed(0)}K`
                      : "-"}
                  </td>
                  <td className="p-4 text-right">
                    <DropDrawer>
                      <DropDrawerTrigger asChild>
                        <Button variant="ghost" size="sm">
                          <MoreHorizontal className="w-4 h-4" />
                        </Button>
                      </DropDrawerTrigger>
                      <DropDrawerContent className="w-48">
                        <DropDrawerItem onClick={() => openEditDialog(model)}>
                          <div className="flex gap-2 items-center">
                            <Pencil className="w-4 h-4" />
                            <span>Edit</span>
                          </div>
                        </DropDrawerItem>
                        <DropDrawerItem onClick={() => handleToggleActive(model)}>
                          <div className="flex gap-2 items-center">
                            {model.active ? (
                              <PowerOff className="w-4 h-4" />
                            ) : (
                              <Power className="w-4 h-4" />
                            )}
                            <span>{model.active ? "Deactivate" : "Activate"}</span>
                          </div>
                        </DropDrawerItem>
                      </DropDrawerContent>
                    </DropDrawer>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
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
              <ChevronLeft className="w-4 h-4 mr-1" />
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPagination((prev) => ({ ...prev, page: prev.page + 1 }))}
              disabled={pagination.page >= totalPages}
            >
              Next
              <ChevronRight className="w-4 h-4 ml-1" />
            </Button>
          </div>
        </div>
      )}

      {/* Edit Provider Model Dialog */}
      <Dialog
        open={editDialogOpen}
        onOpenChange={(open) => {
          if (!open) {
            setEditDialogOpen(false);
            setModelToEdit(null);
            resetForm();
          }
        }}
      >
        <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit Provider Model</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <label className="text-sm font-medium">Display Name</label>
              <Input
                value={formData.model_display_name}
                onChange={(e) =>
                  setFormData({ ...formData, model_display_name: e.target.value })
                }
                placeholder="Model Display Name"
              />
            </div>

            <div className="grid gap-2">
              <label className="text-sm font-medium">Category</label>
              <Input
                value={formData.category}
                onChange={(e) =>
                  setFormData({ ...formData, category: e.target.value })
                }
                list="category-options"
                placeholder="Select or type custom category"
              />
              <datalist id="category-options">
                {CATEGORIES.map((cat) => (
                  <option key={cat.value} value={cat.value} />
                ))}
              </datalist>
              <p className="text-xs text-muted-foreground">
                Select from predefined options or type a custom category
              </p>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <label className="text-sm font-medium">Category Order Number</label>
                <Input
                  type="number"
                  value={formData.category_order_number ?? ""}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      category_order_number: e.target.value ? parseInt(e.target.value) : undefined,
                    })
                  }
                  placeholder="0"
                />
                <p className="text-xs text-muted-foreground">
                  Sort order for the category
                </p>
              </div>
              <div className="grid gap-2">
                <label className="text-sm font-medium">Model Order Number</label>
                <Input
                  type="number"
                  value={formData.model_order_number ?? ""}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      model_order_number: e.target.value ? parseInt(e.target.value) : undefined,
                    })
                  }
                  placeholder="0"
                />
                <p className="text-xs text-muted-foreground">
                  Sort order within the category
                </p>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <label className="text-sm font-medium">Prompt Price</label>
                <Input
                  type="number"
                  step="0.000001"
                  value={formData.pricing_prompt ?? ""}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      pricing_prompt: e.target.value ? parseFloat(e.target.value) : undefined,
                    })
                  }
                  placeholder=""
                />
              </div>
              <div className="grid gap-2">
                <label className="text-sm font-medium">Completion Price</label>
                <Input
                  type="number"
                  step="0.000001"
                  value={formData.pricing_completion ?? ""}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      pricing_completion: e.target.value ? parseFloat(e.target.value) : undefined,
                    })
                  }
                  placeholder=""
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <label className="text-sm font-medium">Context Length</label>
                <Input
                  type="number"
                  value={formData.context_length ?? ""}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      context_length: e.target.value ? parseInt(e.target.value) : undefined,
                    })
                  }
                  placeholder=""
                />
              </div>
              <div className="grid gap-2">
                <label className="text-sm font-medium">Max Output Tokens</label>
                <Input
                  type="number"
                  value={formData.max_completion_tokens ?? ""}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      max_completion_tokens: e.target.value ? parseInt(e.target.value) : undefined,
                    })
                  }
                  placeholder=""
                />
              </div>
            </div>

            <div className="grid gap-2">
              <label className="text-sm font-medium">Capabilities</label>
              <div className="grid grid-cols-2 gap-2">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.supports_images}
                    onChange={(e) =>
                      setFormData({ ...formData, supports_images: e.target.checked })
                    }
                    className="rounded"
                  />
                  <span className="text-sm">Vision</span>
                </label>
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.supports_audio}
                    onChange={(e) =>
                      setFormData({ ...formData, supports_audio: e.target.checked })
                    }
                    className="rounded"
                  />
                  <span className="text-sm">Audio</span>
                </label>
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.supports_video}
                    onChange={(e) =>
                      setFormData({ ...formData, supports_video: e.target.checked })
                    }
                    className="rounded"
                  />
                  <span className="text-sm">Video</span>
                </label>
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.supports_reasoning}
                    onChange={(e) =>
                      setFormData({ ...formData, supports_reasoning: e.target.checked })
                    }
                    className="rounded"
                  />
                  <span className="text-sm">Reasoning</span>
                </label>
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.supports_embeddings}
                    onChange={(e) =>
                      setFormData({ ...formData, supports_embeddings: e.target.checked })
                    }
                    className="rounded"
                  />
                  <span className="text-sm">Embeddings</span>
                </label>
              </div>
            </div>

            <div className="flex items-center gap-2 pt-2">
              <input
                type="checkbox"
                id="active"
                checked={formData.active}
                onChange={(e) =>
                  setFormData({ ...formData, active: e.target.checked })
                }
                className="rounded"
              />
              <label htmlFor="active" className="text-sm font-medium">
                Active
              </label>
            </div>
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Cancel</Button>
            </DialogClose>
            <Button onClick={handleUpdate} disabled={isSubmitting}>
              {isSubmitting ? (
                <Loader2 className="w-4 h-4 animate-spin mr-2" />
              ) : null}
              Save Changes
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
