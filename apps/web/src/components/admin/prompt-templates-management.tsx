import { useEffect, useState } from "react";
import {
  FileText,
  Loader2,
  Plus,
  RefreshCw,
  Search,
  MoreHorizontal,
  Pencil,
  Trash2,
  Copy,
  Power,
  PowerOff,
  ChevronDown,
  ChevronRight,
  Eye,
  Play,
  X,
  Code,
  Variable,
  Settings,
  Info,
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
import { promptTemplateService } from "@/services/admin-service";
import { cn } from "@/lib/utils";

type TabType = "basic" | "content" | "variables" | "metadata";

interface VariableInput {
  name: string;
  value: string;
}

export function PromptTemplatesManagement() {
  const [templates, setTemplates] = useState<PromptTemplate[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string>("");
  const [selectedStatus, setSelectedStatus] = useState<string>("");
  const [expandedTemplate, setExpandedTemplate] = useState<string | null>(null);

  // Dialog states
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [templateToDelete, setTemplateToDelete] = useState<PromptTemplate | null>(null);
  const [deleteConfirmText, setDeleteConfirmText] = useState("");
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [templateToEdit, setTemplateToEdit] = useState<PromptTemplate | null>(null);
  const [activeTab, setActiveTab] = useState<TabType>("basic");

  // Preview dialog states
  const [previewDialogOpen, setPreviewDialogOpen] = useState(false);
  const [templateToPreview, setTemplateToPreview] = useState<PromptTemplate | null>(null);
  const [previewVariables, setPreviewVariables] = useState<VariableInput[]>([]);
  const [renderedPreview, setRenderedPreview] = useState<string>("");

  // Form state
  const [formData, setFormData] = useState<CreatePromptTemplateRequest>({
    name: "",
    description: "",
    category: "general",
    template_key: "",
    content: "",
    variables: [],
    is_active: true,
  });

  // Extended form data for metadata
  const [metadata, setMetadata] = useState<{
    version: string;
    author: string;
    tags: string[];
    notes: string;
  }>({
    version: "1.0",
    author: "",
    tags: [],
    notes: "",
  });

  // Variables management
  const [variablesInput, setVariablesInput] = useState<string>("");
  const [variableDefaults, setVariableDefaults] = useState<Record<string, string>>({});

  const [pagination, setPagination] = useState({
    page: 1,
    limit: 20,
    total: 0,
  });

  useEffect(() => {
    loadTemplates();
  }, [pagination.page, selectedCategory, selectedStatus]);

  async function loadTemplates() {
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
      const response = await promptTemplateService.listPromptTemplates(params);
      setTemplates(response.data || []);
      setPagination((prev) => ({ ...prev, total: response.total || 0 }));
    } catch (err) {
      console.error("Failed to load templates:", err);
      setError("Failed to load prompt templates");
    } finally {
      setIsLoading(false);
    }
  }

  async function handleToggleActive(template: PromptTemplate) {
    try {
      await promptTemplateService.updatePromptTemplate(template.public_id, {
        is_active: !template.is_active,
      });
      await loadTemplates();
    } catch (err) {
      console.error("Failed to toggle template status:", err);
    }
  }

  async function handleDelete() {
    if (!templateToDelete || deleteConfirmText !== "Approved") return;
    try {
      await promptTemplateService.deletePromptTemplate(templateToDelete.public_id);
      setDeleteDialogOpen(false);
      setTemplateToDelete(null);
      setDeleteConfirmText("");
      await loadTemplates();
    } catch (err) {
      console.error("Failed to delete template:", err);
    }
  }

  async function handleDuplicate(template: PromptTemplate) {
    try {
      await promptTemplateService.duplicatePromptTemplate(template.public_id, {
        new_name: `${template.name} (Copy)`,
      });
      await loadTemplates();
    } catch (err) {
      console.error("Failed to duplicate template:", err);
    }
  }

  async function handleCreate() {
    try {
      const data = {
        ...formData,
        variables: parseVariables(variablesInput),
        metadata: {
          ...metadata,
          variable_defaults: variableDefaults,
        },
      };
      await promptTemplateService.createPromptTemplate(data);
      setCreateDialogOpen(false);
      resetForm();
      await loadTemplates();
    } catch (err) {
      console.error("Failed to create template:", err);
    }
  }

  async function handleUpdate() {
    if (!templateToEdit) return;
    try {
      const data = {
        name: formData.name,
        description: formData.description,
        category: formData.category,
        content: formData.content,
        variables: parseVariables(variablesInput),
        is_active: formData.is_active,
        metadata: {
          ...metadata,
          variable_defaults: variableDefaults,
        },
      };
      await promptTemplateService.updatePromptTemplate(templateToEdit.public_id, data);
      setEditDialogOpen(false);
      setTemplateToEdit(null);
      resetForm();
      await loadTemplates();
    } catch (err) {
      console.error("Failed to update template:", err);
    }
  }

  function parseVariables(input: string): string[] {
    return input
      .split("\n")
      .map((v) => v.trim())
      .filter((v) => v.length > 0);
  }

  function resetForm() {
    setFormData({
      name: "",
      description: "",
      category: "general",
      template_key: "",
      content: "",
      variables: [],
      is_active: true,
    });
    setMetadata({
      version: "1.0",
      author: "",
      tags: [],
      notes: "",
    });
    setVariablesInput("");
    setVariableDefaults({});
    setActiveTab("basic");
  }

  function openEditDialog(template: PromptTemplate) {
    setTemplateToEdit(template);
    setFormData({
      name: template.name,
      description: template.description || "",
      category: template.category,
      template_key: template.template_key,
      content: template.content,
      variables: template.variables || [],
      is_active: template.is_active,
    });
    setVariablesInput((template.variables || []).join("\n"));

    // Load metadata if available
    const templateMetadata = (template as any).metadata || {};
    setMetadata({
      version: templateMetadata.version || "1.0",
      author: templateMetadata.author || "",
      tags: templateMetadata.tags || [],
      notes: templateMetadata.notes || "",
    });
    setVariableDefaults(templateMetadata.variable_defaults || {});

    setActiveTab("basic");
    setEditDialogOpen(true);
  }

  function openPreviewDialog(template: PromptTemplate) {
    setTemplateToPreview(template);
    const vars = (template.variables || []).map((v) => ({
      name: v,
      value: "",
    }));
    setPreviewVariables(vars);
    setRenderedPreview(template.content);
    setPreviewDialogOpen(true);
  }

  function updatePreview() {
    if (!templateToPreview) return;

    let rendered = templateToPreview.content;
    previewVariables.forEach((v) => {
      const regex = new RegExp(`\\{\\{\\s*${v.name}\\s*\\}\\}`, "g");
      rendered = rendered.replace(regex, v.value || `{{${v.name}}}`);
    });
    setRenderedPreview(rendered);
  }

  useEffect(() => {
    if (previewDialogOpen) {
      updatePreview();
    }
  }, [previewVariables, previewDialogOpen]);

  function extractVariablesFromContent(content: string): string[] {
    const regex = /\{\{\s*(\w+)\s*\}\}/g;
    const variables = new Set<string>();
    let match;
    while ((match = regex.exec(content)) !== null) {
      variables.add(match[1]);
    }
    return Array.from(variables);
  }

  function handleContentChange(content: string) {
    setFormData({ ...formData, content });
    // Auto-extract variables from content
    const extracted = extractVariablesFromContent(content);
    const existing = parseVariables(variablesInput);
    const merged = Array.from(new Set([...existing, ...extracted]));
    setVariablesInput(merged.join("\n"));
  }

  const categories = [...new Set(templates.map((t) => t.category))].sort();

  const filteredTemplates = templates.filter((t) => {
    const matchesSearch =
      t.name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      t.template_key?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      t.description?.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesSearch;
  });

  const totalPages = Math.ceil(pagination.total / pagination.limit);

  if (isLoading && templates.length === 0) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <Loader2 className="w-8 h-8 animate-spin mx-auto mb-4 text-primary" />
          <p className="text-sm text-muted-foreground">Loading templates...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-6">
        <h3 className="text-lg font-semibold text-destructive mb-2">Error</h3>
        <p className="text-sm text-muted-foreground">{error}</p>
        <Button onClick={loadTemplates} variant="outline" className="mt-4">
          Retry
        </Button>
      </div>
    );
  }

  const tabs: { id: TabType; label: string; icon: React.ElementType }[] = [
    { id: "basic", label: "Basic", icon: Info },
    { id: "content", label: "Content", icon: Code },
    { id: "variables", label: "Variables", icon: Variable },
    { id: "metadata", label: "Metadata", icon: Settings },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Prompt Templates</h1>
          <p className="text-muted-foreground mt-2">
            Manage system and custom prompt templates
          </p>
        </div>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="w-4 h-4 mr-2" />
          Create Template
        </Button>
      </div>

      <div className="flex flex-wrap items-center gap-4">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="Search templates..."
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

        <Button variant="outline" onClick={loadTemplates}>
          <RefreshCw className="w-4 h-4 mr-2" />
          Refresh
        </Button>
      </div>

      <div className="text-sm text-muted-foreground">
        Showing {filteredTemplates.length} of {pagination.total} templates
      </div>

      <div className="space-y-3">
        {filteredTemplates.length === 0 ? (
          <div className="text-center py-12 text-muted-foreground border rounded-lg">
            {searchQuery ? "No templates match your search" : "No templates found"}
          </div>
        ) : (
          filteredTemplates.map((template) => (
            <div
              key={template.id}
              className="bg-card rounded-lg border overflow-hidden"
            >
              <div className="p-4 flex items-start justify-between">
                <div
                  className="flex items-start gap-3 flex-1 cursor-pointer"
                  onClick={() =>
                    setExpandedTemplate(
                      expandedTemplate === template.id ? null : template.id
                    )
                  }
                >
                  <div className="bg-orange-100 dark:bg-orange-900/20 p-2 rounded mt-1">
                    <FileText className="w-5 h-5 text-orange-600" />
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2 flex-wrap">
                      <h3 className="font-semibold">{template.name}</h3>
                      {template.is_system && (
                        <span className="px-2 py-0.5 text-xs bg-blue-100 text-blue-700 dark:bg-blue-900/20 dark:text-blue-400 rounded">
                          System
                        </span>
                      )}
                      <span
                        className={`px-2 py-0.5 text-xs rounded-full ${
                          template.is_active
                            ? "bg-green-100 text-green-700 dark:bg-green-900/20 dark:text-green-400"
                            : "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400"
                        }`}
                      >
                        {template.is_active ? "Active" : "Inactive"}
                      </span>
                      {template.variables && template.variables.length > 0 && (
                        <span className="px-2 py-0.5 text-xs bg-purple-100 text-purple-700 dark:bg-purple-900/20 dark:text-purple-400 rounded">
                          {template.variables.length} variable{template.variables.length !== 1 ? "s" : ""}
                        </span>
                      )}
                    </div>
                    <div className="text-sm text-muted-foreground mt-1">
                      Key: <span className="font-mono text-xs">{template.template_key}</span> | Category: {template.category}
                    </div>
                    {template.description && (
                      <p className="text-sm text-muted-foreground mt-1 line-clamp-1">
                        {template.description}
                      </p>
                    )}
                  </div>
                  <div className="flex items-center">
                    {expandedTemplate === template.id ? (
                      <ChevronDown className="w-5 h-5 text-muted-foreground" />
                    ) : (
                      <ChevronRight className="w-5 h-5 text-muted-foreground" />
                    )}
                  </div>
                </div>
                <div className="ml-4 flex items-center gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => openPreviewDialog(template)}
                    title="Preview template"
                  >
                    <Eye className="w-4 h-4" />
                  </Button>
                  <DropDrawer>
                    <DropDrawerTrigger asChild>
                      <Button variant="ghost" size="sm">
                        <MoreHorizontal className="w-4 h-4" />
                      </Button>
                    </DropDrawerTrigger>
                    <DropDrawerContent className="w-48">
                      <DropDrawerItem onClick={() => openEditDialog(template)}>
                        <div className="flex gap-2 items-center">
                          <Pencil className="w-4 h-4" />
                          <span>Edit</span>
                        </div>
                      </DropDrawerItem>
                      <DropDrawerItem onClick={() => openPreviewDialog(template)}>
                        <div className="flex gap-2 items-center">
                          <Eye className="w-4 h-4" />
                          <span>Preview</span>
                        </div>
                      </DropDrawerItem>
                      <DropDrawerItem onClick={() => handleDuplicate(template)}>
                        <div className="flex gap-2 items-center">
                          <Copy className="w-4 h-4" />
                          <span>Duplicate</span>
                        </div>
                      </DropDrawerItem>
                      <DropDrawerItem onClick={() => handleToggleActive(template)}>
                        <div className="flex gap-2 items-center">
                          {template.is_active ? (
                            <PowerOff className="w-4 h-4" />
                          ) : (
                            <Power className="w-4 h-4" />
                          )}
                          <span>{template.is_active ? "Deactivate" : "Activate"}</span>
                        </div>
                      </DropDrawerItem>
                      <DropDrawerItem
                        onClick={() => {
                          navigator.clipboard.writeText(template.public_id);
                        }}
                      >
                        <div className="flex gap-2 items-center">
                          <Copy className="w-4 h-4" />
                          <span>Copy ID</span>
                        </div>
                      </DropDrawerItem>
                      {!template.is_system && (
                        <DropDrawerItem
                          variant="destructive"
                          onClick={() => {
                            setTemplateToDelete(template);
                            setDeleteConfirmText("");
                            setDeleteDialogOpen(true);
                          }}
                        >
                          <div className="flex gap-2 items-center">
                            <Trash2 className="w-4 h-4" />
                            <span>Delete</span>
                          </div>
                        </DropDrawerItem>
                      )}
                    </DropDrawerContent>
                  </DropDrawer>
                </div>
              </div>

              {expandedTemplate === template.id && (
                <div className="border-t p-4 bg-muted/20">
                  <div className="space-y-4">
                    {template.variables && template.variables.length > 0 && (
                      <div>
                        <div className="text-xs text-muted-foreground uppercase tracking-wider mb-2">
                          Variables
                        </div>
                        <div className="flex flex-wrap gap-2">
                          {template.variables.map((v) => (
                            <span
                              key={v}
                              className="px-2 py-1 text-xs bg-purple-100 text-purple-700 dark:bg-purple-900/20 dark:text-purple-400 rounded font-mono"
                            >
                              {"{{"}{v}{"}}"}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}
                    <div>
                      <div className="flex items-center justify-between mb-2">
                        <div className="text-xs text-muted-foreground uppercase tracking-wider">
                          Content
                        </div>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => openPreviewDialog(template)}
                        >
                          <Play className="w-3 h-3 mr-1" />
                          Test with variables
                        </Button>
                      </div>
                      <pre className="text-sm bg-muted p-3 rounded-md overflow-x-auto whitespace-pre-wrap max-h-[200px] overflow-y-auto font-mono">
                        {template.content}
                      </pre>
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

      {/* Create/Edit Dialog with Tabs */}
      <Dialog
        open={createDialogOpen || editDialogOpen}
        onOpenChange={(open) => {
          if (!open) {
            setCreateDialogOpen(false);
            setEditDialogOpen(false);
            setTemplateToEdit(null);
            resetForm();
          }
        }}
      >
        <DialogContent className="sm:max-w-[700px] max-h-[90vh] overflow-hidden flex flex-col">
          <DialogHeader>
            <DialogTitle>
              {editDialogOpen ? "Edit Template" : "Create Template"}
            </DialogTitle>
            <DialogDescription>
              {editDialogOpen
                ? "Update the prompt template configuration."
                : "Create a new prompt template for your system."}
            </DialogDescription>
          </DialogHeader>

          {/* Tabs */}
          <div className="border-b">
            <nav className="flex space-x-4 px-1" aria-label="Tabs">
              {tabs.map((tab) => {
                const Icon = tab.icon;
                return (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id)}
                    className={cn(
                      "flex items-center gap-2 py-2 px-3 text-sm font-medium border-b-2 -mb-px transition-colors",
                      activeTab === tab.id
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
            {/* Basic Tab */}
            {activeTab === "basic" && (
              <div className="grid gap-4">
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Name *</label>
                  <Input
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="Template name"
                  />
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Template Key *</label>
                  <Input
                    value={formData.template_key}
                    onChange={(e) =>
                      setFormData({ ...formData, template_key: e.target.value })
                    }
                    placeholder="template_key"
                    disabled={editDialogOpen}
                    className="font-mono"
                  />
                  <p className="text-xs text-muted-foreground">
                    Unique identifier for the template (snake_case recommended)
                  </p>
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Category</label>
                  <Input
                    value={formData.category}
                    onChange={(e) => setFormData({ ...formData, category: e.target.value })}
                    placeholder="general"
                  />
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Description</label>
                  <textarea
                    value={formData.description}
                    onChange={(e) =>
                      setFormData({ ...formData, description: e.target.value })
                    }
                    placeholder="Brief description of the template purpose and usage"
                    className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
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
                    Active (template can be used in the system)
                  </label>
                </div>
              </div>
            )}

            {/* Content Tab */}
            {activeTab === "content" && (
              <div className="grid gap-4">
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Template Content *</label>
                  <textarea
                    value={formData.content}
                    onChange={(e) => handleContentChange(e.target.value)}
                    placeholder="Enter the prompt template content...

Use {{variable_name}} syntax for dynamic variables.

Example:
You are a helpful assistant named {{assistant_name}}.
Please help the user with {{task_description}}."
                    className="flex min-h-[300px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono"
                  />
                  <p className="text-xs text-muted-foreground">
                    Use {"{{variable}}"} syntax for dynamic variables. Variables will be auto-extracted.
                  </p>
                </div>
                {formData.content && (
                  <div className="p-3 bg-muted rounded-lg">
                    <div className="text-xs text-muted-foreground uppercase tracking-wider mb-2">
                      Detected Variables
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {extractVariablesFromContent(formData.content).length > 0 ? (
                        extractVariablesFromContent(formData.content).map((v) => (
                          <span
                            key={v}
                            className="px-2 py-1 text-xs bg-purple-100 text-purple-700 dark:bg-purple-900/20 dark:text-purple-400 rounded font-mono"
                          >
                            {"{{"}{v}{"}}"}
                          </span>
                        ))
                      ) : (
                        <span className="text-xs text-muted-foreground">No variables detected</span>
                      )}
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* Variables Tab */}
            {activeTab === "variables" && (
              <div className="grid gap-4">
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Variables (one per line)</label>
                  <textarea
                    value={variablesInput}
                    onChange={(e) => setVariablesInput(e.target.value)}
                    placeholder="variable_name
another_variable
user_name"
                    className="flex min-h-[120px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono"
                  />
                  <p className="text-xs text-muted-foreground">
                    List all template variables, one per line. These are automatically detected from content but can be manually adjusted.
                  </p>
                </div>

                {parseVariables(variablesInput).length > 0 && (
                  <div className="grid gap-4">
                    <div className="text-sm font-medium">Variable Defaults (optional)</div>
                    <p className="text-xs text-muted-foreground -mt-2">
                      Set default values for variables when none are provided
                    </p>
                    {parseVariables(variablesInput).map((varName) => (
                      <div key={varName} className="grid grid-cols-2 gap-4 items-center">
                        <div className="text-sm font-mono text-muted-foreground">
                          {"{{"}{varName}{"}}"}
                        </div>
                        <Input
                          value={variableDefaults[varName] || ""}
                          onChange={(e) =>
                            setVariableDefaults({
                              ...variableDefaults,
                              [varName]: e.target.value,
                            })
                          }
                          placeholder="Default value"
                          className="text-sm"
                        />
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Metadata Tab */}
            {activeTab === "metadata" && (
              <div className="grid gap-4">
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Version</label>
                  <Input
                    value={metadata.version}
                    onChange={(e) => setMetadata({ ...metadata, version: e.target.value })}
                    placeholder="1.0"
                  />
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Author</label>
                  <Input
                    value={metadata.author}
                    onChange={(e) => setMetadata({ ...metadata, author: e.target.value })}
                    placeholder="Author name or email"
                  />
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Tags (comma-separated)</label>
                  <Input
                    value={metadata.tags.join(", ")}
                    onChange={(e) =>
                      setMetadata({
                        ...metadata,
                        tags: e.target.value.split(",").map((t) => t.trim()).filter(Boolean),
                      })
                    }
                    placeholder="chat, assistant, customer-support"
                  />
                </div>
                <div className="grid gap-2">
                  <label className="text-sm font-medium">Notes</label>
                  <textarea
                    value={metadata.notes}
                    onChange={(e) => setMetadata({ ...metadata, notes: e.target.value })}
                    placeholder="Additional notes about this template, usage guidelines, changelog, etc."
                    className="flex min-h-[100px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  />
                </div>
              </div>
            )}
          </div>

          <DialogFooter className="border-t pt-4">
            <DialogClose asChild>
              <Button variant="outline">Cancel</Button>
            </DialogClose>
            <Button
              onClick={editDialogOpen ? handleUpdate : handleCreate}
              disabled={!formData.name || !formData.template_key || !formData.content}
            >
              {editDialogOpen ? "Update" : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Preview Dialog */}
      <Dialog open={previewDialogOpen} onOpenChange={setPreviewDialogOpen}>
        <DialogContent className="sm:max-w-[800px] max-h-[90vh] overflow-hidden flex flex-col">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Eye className="w-5 h-5" />
              Preview: {templateToPreview?.name}
            </DialogTitle>
            <DialogDescription>
              Test the template with sample variable values
            </DialogDescription>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto">
            <div className="grid md:grid-cols-2 gap-4">
              {/* Variables Input */}
              {previewVariables.length > 0 && (
                <div className="space-y-4">
                  <div className="text-sm font-medium">Variables</div>
                  {previewVariables.map((v, idx) => (
                    <div key={v.name} className="grid gap-2">
                      <label className="text-xs font-mono text-muted-foreground">
                        {"{{"}{v.name}{"}}"}
                      </label>
                      <Input
                        value={v.value}
                        onChange={(e) => {
                          const updated = [...previewVariables];
                          updated[idx] = { ...v, value: e.target.value };
                          setPreviewVariables(updated);
                        }}
                        placeholder={`Enter ${v.name}...`}
                      />
                    </div>
                  ))}
                </div>
              )}

              {/* Rendered Preview */}
              <div className={cn("space-y-2", previewVariables.length === 0 && "md:col-span-2")}>
                <div className="text-sm font-medium flex items-center justify-between">
                  <span>Rendered Output</span>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => navigator.clipboard.writeText(renderedPreview)}
                  >
                    <Copy className="w-3 h-3 mr-1" />
                    Copy
                  </Button>
                </div>
                <pre className="text-sm bg-muted p-4 rounded-lg overflow-auto whitespace-pre-wrap min-h-[200px] max-h-[400px] font-mono border">
                  {renderedPreview}
                </pre>
              </div>
            </div>
          </div>

          <DialogFooter className="border-t pt-4">
            <Button variant="outline" onClick={() => setPreviewDialogOpen(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>Delete Template</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete{" "}
              <span className="font-semibold">{templateToDelete?.name}</span>?
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <label className="text-sm font-medium">
              Type "Approved" to confirm deletion
            </label>
            <Input
              value={deleteConfirmText}
              onChange={(e) => setDeleteConfirmText(e.target.value)}
              placeholder="Type 'Approved' to confirm"
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
              disabled={deleteConfirmText !== "Approved"}
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
