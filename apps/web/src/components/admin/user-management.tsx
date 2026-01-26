import { useEffect, useState } from "react";
import { Link } from "@tanstack/react-router";
import {
  ChevronLeft,
  ChevronRight,
  Flag,
  Loader2,
  MoreVertical,
  Plus,
  Search,
  Settings,
  Shield,
  ShieldCheck,
  Tag,
  Trash2,
  UserCheck,
  Users,
  UserX,
  X,
} from "lucide-react";
import { userManagementService } from "@/services/admin-service";
import { Button } from "@/components/ui/button";

export function UserManagement() {
  const [users, setUsers] = useState<UserProfile[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [featureFlags, setFeatureFlags] = useState<FeatureFlag[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [enabledFilter, setEnabledFilter] = useState<boolean | undefined>(
    undefined,
  );
  const [hideGuestUsers, setHideGuestUsers] = useState(true);
  const [page, setPage] = useState(0);
  const [totalUsers, setTotalUsers] = useState(0);
  const [selectedUser, setSelectedUser] = useState<UserProfile | null>(null);
  const [showGroupModal, setShowGroupModal] = useState(false);
  const [showManageGroupsModal, setShowManageGroupsModal] = useState(false);
  const [showFeatureFlagsModal, setShowFeatureFlagsModal] = useState(false);
  const [selectedGroupForFlags, setSelectedGroupForFlags] =
    useState<Group | null>(null);
  const [newGroupName, setNewGroupName] = useState("");
  const [selectedGroupId, setSelectedGroupId] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [openMenuUserId, setOpenMenuUserId] = useState<string | null>(null);

  const max = 20;

  useEffect(() => {
    loadGroups();
    loadFeatureFlags();
  }, []);

  useEffect(() => {
    loadUsers();
  }, [page, searchQuery, enabledFilter, hideGuestUsers]);

  async function loadGroups() {
    try {
      const response = await userManagementService.listGroups();
      setGroups(response.data || []);
    } catch (err) {
      console.error("Failed to load groups:", err);
    }
  }

  async function loadFeatureFlags() {
    try {
      const response = await userManagementService.listFeatureFlags();
      setFeatureFlags(response.data || []);
    } catch (err) {
      console.error("Failed to load feature flags:", err);
    }
  }

  function findGroupByRef(groupRef: string | Group): Group | undefined {
    if (typeof groupRef !== "string") return groupRef;
    return groups.find(
      (g) => g.id === groupRef || g.name === groupRef || g.path === groupRef,
    );
  }

  function isUserInGroup(userGroupRef: string | Group, group: Group): boolean {
    if (typeof userGroupRef === "string") {
      return (
        userGroupRef === group.id ||
        userGroupRef === group.name ||
        userGroupRef === group.path
      );
    }
    return userGroupRef.id === group.id;
  }

  function getUserFeatureFlags(user: UserProfile): string[] {
    if (!user.groups || user.groups.length === 0) return [];

    const flagSet = new Set<string>();
    user.groups.forEach((group) => {
      const groupObj = findGroupByRef(group);
      if (groupObj?.feature_flags) {
        groupObj.feature_flags.forEach((flag) => flagSet.add(flag));
      }
    });

    return Array.from(flagSet);
  }

  async function loadUsers() {
    try {
      setIsLoading(true);
      setError(null);

      const response = await userManagementService.listUsers({
        offset: page * max,
        limit: max,
        search: searchQuery || undefined,
        enabled: enabledFilter,
        exclude_guests: hideGuestUsers,
      });

      setUsers(response.data || []);
      setTotalUsers(response.total || response.data?.length || 0);
    } catch (err) {
      console.error("Failed to load users:", err);
      setError("Failed to load users. Please try again.");
    } finally {
      setIsLoading(false);
    }
  }

  async function handleToggleEnabled(userId: string, currentlyEnabled: boolean) {
    if (
      !confirm(
        `Are you sure you want to ${currentlyEnabled ? "deactivate" : "activate"} this user?`,
      )
    ) {
      return;
    }

    try {
      if (currentlyEnabled) {
        await userManagementService.deactivateUser(userId);
      } else {
        await userManagementService.activateUser(userId);
      }
      loadUsers();
    } catch (err) {
      console.error("Failed to update user status:", err);
      alert("Failed to update user status");
    }
  }

  async function handleAddUserToGroup() {
    if (!selectedUser || !selectedGroupId) return;

    try {
      setIsSubmitting(true);
      await userManagementService.addUserToGroup(selectedUser.id, selectedGroupId);

      const response = await userManagementService.listUsers({
        offset: 0,
        limit: 100,
        search: selectedUser.email || selectedUser.username,
      });
      const updatedUser = response.data?.find((u) => u.id === selectedUser.id);
      if (updatedUser) {
        setSelectedUser(updatedUser);
      }
      setSelectedGroupId("");
      loadUsers();
    } catch (err) {
      console.error("Failed to add user to group:", err);
      alert("Failed to add user to group");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleRemoveUserFromGroup(groupId: string) {
    if (!selectedUser) return;

    try {
      await userManagementService.removeUserFromGroup(selectedUser.id, groupId);

      const response = await userManagementService.listUsers({
        offset: 0,
        limit: 100,
        search: selectedUser.email || selectedUser.username,
      });
      const updatedUser = response.data?.find((u) => u.id === selectedUser.id);
      if (updatedUser) {
        setSelectedUser(updatedUser);
      }
      loadUsers();
    } catch (err) {
      console.error("Failed to remove user from group:", err);
      alert("Failed to remove user from group");
    }
  }

  async function handleCreateGroup(e: React.FormEvent) {
    e.preventDefault();
    if (!newGroupName.trim()) return;

    try {
      setIsSubmitting(true);
      await userManagementService.createGroup(newGroupName);
      setNewGroupName("");
      loadGroups();
    } catch (err) {
      console.error("Failed to create group:", err);
      alert("Failed to create group");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleDeleteGroup(groupId: string) {
    if (!confirm("Are you sure you want to delete this group?")) return;

    try {
      await userManagementService.deleteGroup(groupId);
      loadGroups();
    } catch (err) {
      console.error("Failed to delete group:", err);
      alert("Failed to delete group");
    }
  }

  async function handleAssignAdminRole(userId: string) {
    if (!confirm("Are you sure you want to assign admin role to this user?")) {
      return;
    }

    try {
      await userManagementService.assignAdminRole(userId);
      loadUsers();
    } catch (err) {
      console.error("Failed to assign admin role:", err);
      alert("Failed to assign admin role");
    }
  }

  async function handleToggleGroupFlag(group: Group, flagKey: string, enable: boolean) {
    try {
      const newFlags = enable
        ? [...(group.feature_flags || []), flagKey]
        : (group.feature_flags || []).filter((f) => f !== flagKey);

      await userManagementService.setGroupFeatureFlags(group.id, newFlags);
      await loadGroups();
      await loadUsers();

      const updatedGroup = groups.find((g) => g.id === group.id);
      if (updatedGroup) {
        setSelectedGroupForFlags(updatedGroup);
      }
    } catch (err) {
      console.error("Failed to update feature flag:", err);
      alert("Failed to update feature flag");
    }
  }

  const totalPages = Math.max(1, Math.ceil(totalUsers / max));
  const userGroups = selectedUser?.groups || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <Users className="w-8 h-8" />
            User Management
          </h1>
          <p className="text-muted-foreground mt-2">
            Manage users, permissions, and groups
          </p>
        </div>
        <div className="flex gap-2">
          <Link to="/admin/users/feature-flags">
            <Button variant="outline">
              <Flag className="w-4 h-4 mr-2" />
              Feature Flags
            </Button>
          </Link>
          <Button variant="outline" onClick={() => setShowManageGroupsModal(true)}>
            <Settings className="w-4 h-4 mr-2" />
            Manage Groups
          </Button>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-card rounded-lg border p-4">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <input
              type="text"
              placeholder="Search by email or username..."
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                setPage(0);
              }}
              className="w-full pl-9 pr-3 py-2 border rounded-md bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary"
            />
          </div>

          <select
            value={
              enabledFilter === undefined
                ? "all"
                : enabledFilter
                  ? "enabled"
                  : "disabled"
            }
            onChange={(e) => {
              setEnabledFilter(
                e.target.value === "all"
                  ? undefined
                  : e.target.value === "enabled",
              );
              setPage(0);
            }}
            className="px-3 py-2 border rounded-md bg-background text-sm focus:outline-none focus:ring-2 focus:ring-primary"
          >
            <option value="all">All Users</option>
            <option value="enabled">Enabled Only</option>
            <option value="disabled">Disabled Only</option>
          </select>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="hideGuestUsers"
              checked={hideGuestUsers}
              onChange={(e) => {
                setHideGuestUsers(e.target.checked);
                setPage(0);
              }}
              className="rounded border-gray-300 cursor-pointer"
            />
            <label
              htmlFor="hideGuestUsers"
              className="text-sm cursor-pointer select-none"
            >
              Hide Guest Users
            </label>
          </div>
        </div>

        {(searchQuery || enabledFilter !== undefined || !hideGuestUsers) && (
          <div className="mt-3">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setSearchQuery("");
                setEnabledFilter(undefined);
                setHideGuestUsers(true);
                setPage(0);
              }}
            >
              Clear Filters
            </Button>
          </div>
        )}
      </div>

      {/* Error */}
      {error && (
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
          <p className="text-sm text-destructive">{error}</p>
        </div>
      )}

      {/* Users Table */}
      <div className="bg-card rounded-lg border">
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="w-6 h-6 animate-spin text-primary" />
          </div>
        ) : users.length === 0 ? (
          <div className="text-center py-12">
            <Users className="w-12 h-12 mx-auto text-muted-foreground mb-4" />
            <p className="text-muted-foreground">No users found</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-muted/50">
                <tr>
                  <th className="text-left px-4 py-3 text-sm font-medium">
                    User
                  </th>
                  <th className="text-left px-4 py-3 text-sm font-medium">
                    Email
                  </th>
                  <th className="text-left px-4 py-3 text-sm font-medium">
                    Status
                  </th>
                  <th className="text-left px-4 py-3 text-sm font-medium">
                    Groups
                  </th>
                  <th className="text-left px-4 py-3 text-sm font-medium">
                    Feature Flags
                  </th>
                  <th className="text-left px-4 py-3 text-sm font-medium">
                    Role
                  </th>
                  <th className="text-right px-4 py-3 text-sm font-medium">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {users.map((user) => {
                  const isEnabled = user.enabled !== false;
                  const displayName =
                    user.first_name && user.last_name
                      ? `${user.first_name} ${user.last_name}`
                      : user.name || user.username || "Unknown";

                  return (
                    <tr key={user.id} className="hover:bg-muted/30 transition-colors">
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-3">
                          {user.picture ? (
                            <img
                              src={user.picture}
                              alt={displayName}
                              className="w-8 h-8 rounded-full"
                            />
                          ) : (
                            <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
                              <Users className="w-4 h-4 text-primary" />
                            </div>
                          )}
                          <span className="font-medium text-sm">
                            {displayName}
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-sm text-muted-foreground">
                        {user.email || "N/A"}
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium ${
                            isEnabled
                              ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                              : "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400"
                          }`}
                        >
                          <span
                            className={`w-1.5 h-1.5 rounded-full ${isEnabled ? "bg-green-500" : "bg-gray-500"}`}
                          />
                          {isEnabled ? "Enabled" : "Disabled"}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap gap-1">
                          {user.groups && user.groups.length > 0 ? (
                            user.groups.slice(0, 2).map((group) => (
                              <span
                                key={typeof group === "string" ? group : group.id}
                                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-primary/10 text-primary text-xs"
                              >
                                <Tag className="w-3 h-3" />
                                {typeof group === "string" ? group : group.name}
                              </span>
                            ))
                          ) : (
                            <span className="text-xs text-muted-foreground">
                              No groups
                            </span>
                          )}
                          {user.groups && user.groups.length > 2 && (
                            <span className="text-xs text-muted-foreground">
                              +{user.groups.length - 2}
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        {(() => {
                          const userFlags = getUserFeatureFlags(user);
                          return userFlags.length > 0 ? (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400 text-xs">
                              <Flag className="w-3 h-3" />
                              {userFlags.length} flag{userFlags.length !== 1 ? "s" : ""}
                            </span>
                          ) : (
                            <span className="text-xs text-muted-foreground">
                              No flags
                            </span>
                          );
                        })()}
                      </td>
                      <td className="px-4 py-3">
                        {user.is_admin || user.role === "admin" ? (
                          <span className="inline-flex items-center gap-1 px-2 py-1 rounded-md bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400 text-xs font-medium">
                            <Shield className="w-3 h-3" />
                            Admin
                          </span>
                        ) : (
                          <span className="text-xs text-muted-foreground">
                            User
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-3 relative">
                        <button
                          onClick={() =>
                            setOpenMenuUserId(
                              openMenuUserId === user.id ? null : user.id,
                            )
                          }
                          className="p-1 hover:bg-accent rounded transition-colors"
                        >
                          <MoreVertical className="w-4 h-4" />
                        </button>

                        {openMenuUserId === user.id && (
                          <>
                            <div
                              className="fixed inset-0 z-40"
                              onClick={() => setOpenMenuUserId(null)}
                            />
                            <div className="absolute right-0 top-full mt-1 w-48 bg-card border border-border rounded-md shadow-lg z-50">
                              <button
                                onClick={() => {
                                  setSelectedUser(user);
                                  setShowGroupModal(true);
                                  setOpenMenuUserId(null);
                                }}
                                className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent transition-colors text-left"
                              >
                                <Tag className="w-4 h-4" />
                                Manage Groups
                              </button>
                              <button
                                onClick={() => {
                                  setSelectedUser(user);
                                  setShowFeatureFlagsModal(true);
                                  setOpenMenuUserId(null);
                                }}
                                className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent transition-colors text-left"
                              >
                                <Settings className="w-4 h-4" />
                                View Flags
                              </button>
                              <button
                                onClick={() => {
                                  handleToggleEnabled(user.id, isEnabled);
                                  setOpenMenuUserId(null);
                                }}
                                className={`w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent transition-colors text-left ${
                                  isEnabled ? "text-destructive" : "text-green-600"
                                }`}
                              >
                                {isEnabled ? (
                                  <>
                                    <UserX className="w-4 h-4" />
                                    Deactivate
                                  </>
                                ) : (
                                  <>
                                    <UserCheck className="w-4 h-4" />
                                    Activate
                                  </>
                                )}
                              </button>
                              {!(user.is_admin || user.role === "admin") && (
                                <button
                                  onClick={() => {
                                    handleAssignAdminRole(user.id);
                                    setOpenMenuUserId(null);
                                  }}
                                  className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-accent transition-colors text-left text-orange-600"
                                >
                                  <ShieldCheck className="w-4 h-4" />
                                  Assign Admin Role
                                </button>
                              )}
                            </div>
                          </>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            Showing {page * max + 1} to {Math.min((page + 1) * max, totalUsers)}{" "}
            of {totalUsers} users
          </p>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage(Math.max(0, page - 1))}
              disabled={page === 0}
              className="p-2 border rounded-md hover:bg-accent disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>
            <span className="text-sm">
              Page {page + 1} of {totalPages}
            </span>
            <button
              onClick={() => setPage(Math.min(totalPages - 1, page + 1))}
              disabled={page >= totalPages - 1}
              className="p-2 border rounded-md hover:bg-accent disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      )}

      {/* Group Management Modal */}
      {showGroupModal && selectedUser && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-card rounded-lg border max-w-md w-full p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold">Manage Groups</h2>
              <button
                onClick={() => {
                  setShowGroupModal(false);
                  setSelectedUser(null);
                  setSelectedGroupId("");
                }}
                className="p-1 hover:bg-accent rounded-md"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="space-y-4">
              <p className="text-sm text-muted-foreground">
                User:{" "}
                <span className="font-medium text-foreground">
                  {selectedUser.username || selectedUser.email}
                </span>
              </p>

              {/* Current Groups */}
              <div className="space-y-2">
                <label className="block text-sm font-medium">
                  Current Groups
                </label>
                {userGroups.length > 0 ? (
                  <div className="space-y-2">
                    {userGroups.map((group) => {
                      const groupObj = findGroupByRef(group);
                      if (!groupObj) return null;

                      return (
                        <div
                          key={groupObj.id}
                          className="flex items-center justify-between p-2 border rounded-md"
                        >
                          <span className="text-sm flex items-center gap-2">
                            <Tag className="w-4 h-4 text-primary" />
                            {groupObj.name}
                          </span>
                          <button
                            onClick={() =>
                              handleRemoveUserFromGroup(groupObj.id)
                            }
                            className="p-1 hover:bg-destructive/10 text-destructive rounded-md transition-colors"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      );
                    })}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground py-2">
                    No groups assigned
                  </p>
                )}
              </div>

              {/* Add Group */}
              <div>
                <label className="block text-sm font-medium mb-1">
                  Add to Group
                </label>
                {(() => {
                  const availableGroups = groups.filter(
                    (g) => !userGroups.some((ug) => isUserInGroup(ug, g)),
                  );

                  if (availableGroups.length === 0) {
                    return (
                      <p className="text-sm text-muted-foreground py-2">
                        {groups.length === 0
                          ? "No groups available. Create a group first."
                          : "User is already in all available groups."}
                      </p>
                    );
                  }

                  return (
                    <div className="flex gap-2">
                      <select
                        value={selectedGroupId}
                        onChange={(e) => setSelectedGroupId(e.target.value)}
                        className="flex-1 px-3 py-2 border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                      >
                        <option value="">Select a group...</option>
                        {availableGroups.map((group) => (
                          <option key={group.id} value={group.id}>
                            {group.name}
                          </option>
                        ))}
                      </select>
                      <Button
                        onClick={handleAddUserToGroup}
                        disabled={!selectedGroupId || isSubmitting}
                        size="icon"
                      >
                        {isSubmitting ? (
                          <Loader2 className="w-4 h-4 animate-spin" />
                        ) : (
                          <Plus className="w-4 h-4" />
                        )}
                      </Button>
                    </div>
                  );
                })()}
              </div>

              <div className="flex justify-end pt-4">
                <Button
                  onClick={() => {
                    setShowGroupModal(false);
                    setSelectedUser(null);
                    setSelectedGroupId("");
                    loadUsers();
                  }}
                >
                  Done
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Manage Groups Modal */}
      {showManageGroupsModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-card rounded-lg border max-w-md w-full p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold">Manage Groups</h2>
              <button
                onClick={() => setShowManageGroupsModal(false)}
                className="p-1 hover:bg-accent rounded-md"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="space-y-6">
              {/* Create Group Form */}
              <form onSubmit={handleCreateGroup} className="flex gap-2">
                <input
                  type="text"
                  placeholder="New group name..."
                  value={newGroupName}
                  onChange={(e) => setNewGroupName(e.target.value)}
                  className="flex-1 px-3 py-2 border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary"
                />
                <Button
                  type="submit"
                  disabled={!newGroupName.trim() || isSubmitting}
                  size="icon"
                >
                  {isSubmitting ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <Plus className="w-4 h-4" />
                  )}
                </Button>
              </form>

              {/* Groups List */}
              <div className="space-y-2">
                <label className="block text-sm font-medium">
                  Existing Groups
                </label>
                {groups.length > 0 ? (
                  <div className="max-h-60 overflow-y-auto space-y-2 pr-2">
                    {groups.map((group) => (
                      <div
                        key={group.id}
                        className="flex items-center justify-between p-3 border rounded-md bg-muted/30"
                      >
                        <span className="font-medium flex items-center gap-2">
                          <Tag className="w-4 h-4 text-primary" />
                          {group.name}
                        </span>
                        <button
                          onClick={() => handleDeleteGroup(group.id)}
                          className="p-1.5 hover:bg-destructive/10 text-destructive rounded-md transition-colors"
                          title="Delete group"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-center py-8 border rounded-md border-dashed">
                    <Tag className="w-8 h-8 mx-auto text-muted-foreground mb-2" />
                    <p className="text-sm text-muted-foreground">
                      No groups found
                    </p>
                  </div>
                )}
              </div>

              <div className="flex justify-end pt-2">
                <Button
                  variant="outline"
                  onClick={() => setShowManageGroupsModal(false)}
                >
                  Close
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Feature Flags Modal */}
      {showFeatureFlagsModal && selectedUser && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-card rounded-lg border max-w-2xl w-full p-6 max-h-[80vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-xl font-bold flex items-center gap-2">
                  <Settings className="w-5 h-5" />
                  Feature Flags
                </h2>
                <p className="text-sm text-muted-foreground mt-1">
                  {selectedUser.first_name} {selectedUser.last_name} (
                  {selectedUser.email})
                </p>
              </div>
              <button
                onClick={() => {
                  setShowFeatureFlagsModal(false);
                  setSelectedUser(null);
                }}
                className="p-1 hover:bg-accent rounded-md transition-colors"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="space-y-6">
              {/* User's Current Flags */}
              <div>
                <h3 className="font-medium mb-3 flex items-center gap-2">
                  <Shield className="w-4 h-4" />
                  Active Feature Flags
                </h3>

                {(() => {
                  const userFlags = getUserFeatureFlags(selectedUser);
                  return userFlags.length > 0 ? (
                    <div className="space-y-2">
                      {userFlags.map((flag) => (
                        <div
                          key={flag}
                          className="p-3 border rounded-md bg-muted/30"
                        >
                          <div className="flex items-start justify-between">
                            <div className="flex-1">
                              <div className="font-medium text-sm flex items-center gap-2">
                                <Settings className="w-4 h-4 text-blue-600" />
                                {flag}
                              </div>
                            </div>
                            <span className="inline-flex items-center gap-1 px-2 py-1 rounded-md bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 text-xs font-medium">
                              Active
                            </span>
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="text-center py-8 border rounded-md border-dashed">
                      <Settings className="w-8 h-8 mx-auto text-muted-foreground mb-2" />
                      <p className="text-sm text-muted-foreground">
                        No feature flags active
                      </p>
                    </div>
                  );
                })()}
              </div>
            </div>

            <div className="flex justify-end pt-4 mt-4 border-t">
              <Button
                onClick={() => {
                  setShowFeatureFlagsModal(false);
                  setSelectedUser(null);
                }}
              >
                Close
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Group Feature Flags Management Modal */}
      {selectedGroupForFlags && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[60] p-4">
          <div className="bg-card rounded-lg border max-w-xl w-full p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-bold flex items-center gap-2">
                <Tag className="w-5 h-5" />
                Manage Feature Flags - {selectedGroupForFlags.name}
              </h3>
              <button
                onClick={() => setSelectedGroupForFlags(null)}
                className="p-1 hover:bg-accent rounded-md transition-colors"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="space-y-4">
              <p className="text-sm text-muted-foreground">
                Select feature flags to enable for all members of this group.
              </p>

              <div className="space-y-2 max-h-96 overflow-y-auto">
                {featureFlags.map((flag) => {
                  const isEnabled =
                    selectedGroupForFlags.feature_flags?.includes(flag.key) ||
                    false;

                  return (
                    <label
                      key={flag.id}
                      className="flex items-start gap-3 p-3 border rounded-md hover:bg-muted/50 cursor-pointer transition-colors"
                    >
                      <input
                        type="checkbox"
                        checked={isEnabled}
                        onChange={(e) =>
                          handleToggleGroupFlag(
                            selectedGroupForFlags,
                            flag.key,
                            e.target.checked,
                          )
                        }
                        className="mt-0.5 rounded border-gray-300"
                      />
                      <div className="flex-1">
                        <div className="font-medium text-sm">{flag.name}</div>
                        <div className="text-xs text-muted-foreground">
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

              <div className="flex justify-end pt-2 border-t">
                <Button
                  variant="outline"
                  onClick={() => setSelectedGroupForFlags(null)}
                >
                  Done
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
