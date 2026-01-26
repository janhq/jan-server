import { useEffect, useState } from "react";
import { Link } from "@tanstack/react-router";
import {
  ArrowLeft,
  ChevronLeft,
  ChevronRight,
  Flag,
  Loader2,
  MoreHorizontal,
  Pencil,
  Plus,
  Search,
  Trash2,
  X,
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
import { userManagementService } from "@/services/admin-service";

export function FeatureFlagsManagement() {
  const [featureFlags, setFeatureFlags] = useState<FeatureFlag[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");

  // Dialog states
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [groupFlagsDialogOpen, setGroupFlagsDialogOpen] = useState(false);
  const [flagToEdit, setFlagToEdit] = useState<FeatureFlag | null>(null);
  const [flagToDelete, setFlagToDelete] = useState<FeatureFlag | null>(null);
  const [selectedGroup, setSelectedGroup] = useState<Group | null>(null);

  // Form state
  const [formData, setFormData] = useState({
    key: "",
    name: "",
    description: "",
  });

  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    loadData();
  }, []);

  async function loadData() {
    try {
      setIsLoading(true);
      setError(null);
      const [flagsResponse, groupsResponse] = await Promise.all([
        userManagementService.listFeatureFlags(),
        userManagementService.listGroups(),
      ]);
      setFeatureFlags(flagsResponse.data || []);
      setGroups(groupsResponse.data || []);
    } catch (err) {
      console.error("Failed to load data:", err);
      setError("Failed to load feature flags");
    } finally {
      setIsLoading(false);
    }
  }

  async function handleCreate() {
    if (!formData.key || !formData.name) return;

    try {
      setIsSubmitting(true);
      await userManagementService.createFeatureFlag({
        key: formData.key,
        name: formData.name,
        description: formData.description || undefined,
      });
      setCreateDialogOpen(false);
      resetForm();
      await loadData();
    } catch (err) {
      console.error("Failed to create feature flag:", err);
      alert("Failed to create feature flag");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleUpdate() {
    if (!flagToEdit || !formData.name) return;

    try {
      setIsSubmitting(true);
      await userManagementService.updateFeatureFlag(flagToEdit.id, {
        name: formData.name,
        description: formData.description || undefined,
      });
      setEditDialogOpen(false);
      setFlagToEdit(null);
      resetForm();
      await loadData();
    } catch (err) {
      console.error("Failed to update feature flag:", err);
      alert("Failed to update feature flag");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleDelete() {
    if (!flagToDelete) return;

    try {
      setIsSubmitting(true);
      await userManagementService.deleteFeatureFlag(flagToDelete.id);
      setDeleteDialogOpen(false);
      setFlagToDelete(null);
      await loadData();
    } catch (err) {
      console.error("Failed to delete feature flag:", err);
      alert("Failed to delete feature flag");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleToggleGroupFlag(group: Group, flagKey: string, enable: boolean) {
    try {
      const newFlags = enable
        ? [...(group.feature_flags || []), flagKey]
        : (group.feature_flags || []).filter((f) => f !== flagKey);

      await userManagementService.setGroupFeatureFlags(group.id, newFlags);
      await loadData();

      // Update selected group state
      const updatedGroups = groups.map((g) =>
        g.id === group.id ? { ...g, feature_flags: newFlags } : g
      );
      setGroups(updatedGroups);
      if (selectedGroup?.id === group.id) {
        setSelectedGroup({ ...selectedGroup, feature_flags: newFlags });
      }
    } catch (err) {
      console.error("Failed to update group feature flag:", err);
      alert("Failed to update group feature flag");
    }
  }

  function resetForm() {
    setFormData({
      key: "",
      name: "",
      description: "",
    });
  }

  function openEditDialog(flag: FeatureFlag) {
    setFlagToEdit(flag);
    setFormData({
      key: flag.key,
      name: flag.name,
      description: flag.description || "",
    });
    setEditDialogOpen(true);
  }

  function openGroupFlagsDialog(group: Group) {
    setSelectedGroup(group);
    setGroupFlagsDialogOpen(true);
  }

  const filteredFlags = featureFlags.filter(
    (f) =>
      f.key?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      f.name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      f.description?.toLowerCase().includes(searchQuery.toLowerCase())
  );

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <Loader2 className="w-8 h-8 animate-spin mx-auto mb-4 text-primary" />
          <p className="text-sm text-muted-foreground">Loading feature flags...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-6">
        <h3 className="text-lg font-semibold text-destructive mb-2">Error</h3>
        <p className="text-sm text-muted-foreground">{error}</p>
        <Button onClick={loadData} variant="outline" className="mt-4">
          Retry
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link to="/admin/users">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back to Users
          </Button>
        </Link>
      </div>

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <Flag className="w-8 h-8" />
            Feature Flags
          </h1>
          <p className="text-muted-foreground mt-2">
            Manage feature flags for gradual rollout and access control
          </p>
        </div>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="w-4 h-4 mr-2" />
          Create Flag
        </Button>
      </div>

      {/* Search */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="Search flags..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      {/* Feature Flags Table */}
      <div className="bg-card rounded-lg border">
        {filteredFlags.length === 0 ? (
          <div className="text-center py-12">
            <Flag className="w-12 h-12 mx-auto text-muted-foreground mb-4" />
            <p className="text-muted-foreground">
              {searchQuery ? "No flags match your search" : "No feature flags found"}
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-muted/50">
                <tr>
                  <th className="text-left px-4 py-3 text-sm font-medium">Key</th>
                  <th className="text-left px-4 py-3 text-sm font-medium">Name</th>
                  <th className="text-left px-4 py-3 text-sm font-medium">Description</th>
                  <th className="text-left px-4 py-3 text-sm font-medium">Groups Using</th>
                  <th className="text-left px-4 py-3 text-sm font-medium">Created</th>
                  <th className="text-right px-4 py-3 text-sm font-medium">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {filteredFlags.map((flag) => {
                  const groupsUsingFlag = groups.filter((g) =>
                    g.feature_flags?.includes(flag.key)
                  );

                  return (
                    <tr key={flag.id} className="hover:bg-muted/30 transition-colors">
                      <td className="px-4 py-3">
                        <code className="text-sm bg-muted px-2 py-1 rounded">
                          {flag.key}
                        </code>
                      </td>
                      <td className="px-4 py-3">
                        <span className="font-medium text-sm">{flag.name}</span>
                      </td>
                      <td className="px-4 py-3 text-sm text-muted-foreground max-w-xs truncate">
                        {flag.description || "-"}
                      </td>
                      <td className="px-4 py-3">
                        {groupsUsingFlag.length > 0 ? (
                          <span className="inline-flex items-center gap-1 px-2 py-1 rounded-md bg-primary/10 text-primary text-xs font-medium">
                            {groupsUsingFlag.length} group{groupsUsingFlag.length !== 1 ? "s" : ""}
                          </span>
                        ) : (
                          <span className="text-xs text-muted-foreground">No groups</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-sm text-muted-foreground">
                        {flag.created_at
                          ? new Date(flag.created_at).toLocaleDateString()
                          : "-"}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <DropDrawer>
                          <DropDrawerTrigger asChild>
                            <Button variant="ghost" size="sm">
                              <MoreHorizontal className="w-4 h-4" />
                            </Button>
                          </DropDrawerTrigger>
                          <DropDrawerContent className="w-48">
                            <DropDrawerItem onClick={() => openEditDialog(flag)}>
                              <div className="flex gap-2 items-center">
                                <Pencil className="w-4 h-4" />
                                <span>Edit</span>
                              </div>
                            </DropDrawerItem>
                            <DropDrawerItem
                              variant="destructive"
                              onClick={() => {
                                setFlagToDelete(flag);
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
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Group Assignment Section */}
      <div className="space-y-4">
        <h2 className="text-xl font-semibold">Assign to Groups</h2>
        <p className="text-sm text-muted-foreground">
          Select a group to manage its feature flag assignments
        </p>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {groups.map((group) => (
            <div
              key={group.id}
              className="bg-card rounded-lg border p-4 hover:shadow-sm transition-shadow cursor-pointer"
              onClick={() => openGroupFlagsDialog(group)}
            >
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="font-medium">{group.name}</h3>
                  <p className="text-sm text-muted-foreground">
                    {group.feature_flags?.length || 0} flags assigned
                  </p>
                </div>
                <ChevronRight className="w-5 h-5 text-muted-foreground" />
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Create Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>Create Feature Flag</DialogTitle>
            <DialogDescription>
              Create a new feature flag for gradual rollout or access control.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <label className="text-sm font-medium">Key *</label>
              <Input
                value={formData.key}
                onChange={(e) =>
                  setFormData({
                    ...formData,
                    key: e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, "_"),
                  })
                }
                placeholder="my_feature_flag"
              />
              <p className="text-xs text-muted-foreground">
                Unique identifier (lowercase, underscores allowed)
              </p>
            </div>
            <div className="grid gap-2">
              <label className="text-sm font-medium">Name *</label>
              <Input
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                placeholder="My Feature Flag"
              />
            </div>
            <div className="grid gap-2">
              <label className="text-sm font-medium">Description</label>
              <textarea
                value={formData.description}
                onChange={(e) =>
                  setFormData({ ...formData, description: e.target.value })
                }
                placeholder="Describe what this flag controls..."
                className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              />
            </div>
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Cancel</Button>
            </DialogClose>
            <Button
              onClick={handleCreate}
              disabled={!formData.key || !formData.name || isSubmitting}
            >
              {isSubmitting ? (
                <Loader2 className="w-4 h-4 animate-spin mr-2" />
              ) : null}
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog
        open={editDialogOpen}
        onOpenChange={(open) => {
          if (!open) {
            setEditDialogOpen(false);
            setFlagToEdit(null);
            resetForm();
          }
        }}
      >
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>Edit Feature Flag</DialogTitle>
            <DialogDescription>
              Update the feature flag details. The key cannot be changed.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <label className="text-sm font-medium">Key</label>
              <Input value={formData.key} disabled />
            </div>
            <div className="grid gap-2">
              <label className="text-sm font-medium">Name *</label>
              <Input
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                placeholder="My Feature Flag"
              />
            </div>
            <div className="grid gap-2">
              <label className="text-sm font-medium">Description</label>
              <textarea
                value={formData.description}
                onChange={(e) =>
                  setFormData({ ...formData, description: e.target.value })
                }
                placeholder="Describe what this flag controls..."
                className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              />
            </div>
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Cancel</Button>
            </DialogClose>
            <Button onClick={handleUpdate} disabled={!formData.name || isSubmitting}>
              {isSubmitting ? (
                <Loader2 className="w-4 h-4 animate-spin mr-2" />
              ) : null}
              Update
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>Delete Feature Flag</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete the feature flag{" "}
              <span className="font-semibold">{flagToDelete?.name}</span>? This will
              remove it from all groups and cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Cancel</Button>
            </DialogClose>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={isSubmitting}
            >
              {isSubmitting ? (
                <Loader2 className="w-4 h-4 animate-spin mr-2" />
              ) : null}
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Group Feature Flags Dialog */}
      <Dialog
        open={groupFlagsDialogOpen}
        onOpenChange={(open) => {
          if (!open) {
            setGroupFlagsDialogOpen(false);
            setSelectedGroup(null);
          }
        }}
      >
        <DialogContent className="sm:max-w-[600px] max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Manage Group Feature Flags</DialogTitle>
            <DialogDescription>
              Configure which feature flags are enabled for{" "}
              <span className="font-semibold">{selectedGroup?.name}</span>
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            {featureFlags.length === 0 ? (
              <div className="text-center py-8 border rounded-md border-dashed">
                <Flag className="w-8 h-8 mx-auto text-muted-foreground mb-2" />
                <p className="text-sm text-muted-foreground">
                  No feature flags available. Create one first.
                </p>
              </div>
            ) : (
              <div className="space-y-2">
                {featureFlags.map((flag) => {
                  const isEnabled = selectedGroup?.feature_flags?.includes(flag.key);

                  return (
                    <label
                      key={flag.id}
                      className="flex items-start gap-3 p-3 border rounded-md hover:bg-muted/50 cursor-pointer transition-colors"
                    >
                      <input
                        type="checkbox"
                        checked={isEnabled}
                        onChange={(e) => {
                          if (selectedGroup) {
                            handleToggleGroupFlag(
                              selectedGroup,
                              flag.key,
                              e.target.checked
                            );
                          }
                        }}
                        className="mt-0.5 rounded border-gray-300"
                      />
                      <div className="flex-1">
                        <div className="font-medium text-sm">{flag.name}</div>
                        <div className="text-xs text-muted-foreground font-mono">
                          {flag.key}
                        </div>
                        {flag.description && (
                          <div className="text-xs text-muted-foreground mt-1">
                            {flag.description}
                          </div>
                        )}
                      </div>
                    </label>
                  );
                })}
              </div>
            )}
          </div>
          <DialogFooter>
            <Button onClick={() => setGroupFlagsDialogOpen(false)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
