import { useEffect, useState } from "react";
import { AlertTriangle, Copy, Loader2, Plus, Trash2, X } from "lucide-react";
import { apiKeyService } from "@/services/admin-service";
import { Button } from "@/components/ui/button";

export function ApiKeysTab() {
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isCreating, setIsCreating] = useState(false);
  const [newKeyName, setNewKeyName] = useState("");
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState<string | null>(null);

  useEffect(() => {
    loadApiKeys();
  }, []);

  async function loadApiKeys() {
    try {
      setIsLoading(true);
      const response = await apiKeyService.listApiKeys();
      setApiKeys(response.items || []);
    } catch (error) {
      console.error("Failed to list API keys:", error);
      setApiKeys([]);
    } finally {
      setIsLoading(false);
    }
  }

  async function handleCreateKey() {
    if (!newKeyName.trim()) return;

    try {
      setIsCreating(true);
      const response = await apiKeyService.createApiKey({ name: newKeyName });
      setCreatedKey(response.key);
      setNewKeyName("");
      loadApiKeys();
    } catch (error) {
      console.error("Failed to create API key:", error);
    } finally {
      setIsCreating(false);
    }
  }

  async function handleDeleteKey(id: string) {
    try {
      await apiKeyService.deleteApiKey(id);
      setShowDeleteModal(null);
      loadApiKeys();
    } catch (error) {
      console.error("Failed to delete API key:", error);
    }
  }

  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text);
  }

  function formatDate(dateString: string) {
    try {
      return new Date(dateString).toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
      });
    } catch {
      return "Invalid Date";
    }
  }

  function getStatusBadge(status?: string) {
    if (status === "active") {
      return (
        <span className="inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400">
          <span className="w-1.5 h-1.5 rounded-full bg-green-600 dark:bg-green-400"></span>
          Active
        </span>
      );
    } else if (status === "revoked") {
      return (
        <span className="inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400">
          <span className="w-1.5 h-1.5 rounded-full bg-red-600 dark:bg-red-400"></span>
          Revoked
        </span>
      );
    } else if (status === "expired") {
      return (
        <span className="inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-700 dark:bg-gray-900/30 dark:text-gray-400">
          <span className="w-1.5 h-1.5 rounded-full bg-gray-600 dark:bg-gray-400"></span>
          Expired
        </span>
      );
    }
    return null;
  }

  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <h3 className="text-lg font-medium">User API keys</h3>
        <p className="text-sm text-muted-foreground">
          Your secret API keys are listed below. Please note that we do not
          display your secret API keys again after you generate them.
        </p>
        <p className="text-sm text-muted-foreground">
          Do not share your API key with others, or expose it in the browser or
          other client-side code.
        </p>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : (
        <div className="rounded-md border">
          <div className="grid grid-cols-12 gap-4 border-b bg-muted/50 p-4 text-xs font-medium text-muted-foreground uppercase tracking-wider">
            <div className="col-span-3">Name</div>
            <div className="col-span-3">Secret Key</div>
            <div className="col-span-2">Created</div>
            <div className="col-span-2">Status</div>
            <div className="col-span-2 text-right">Actions</div>
          </div>
          {apiKeys.length === 0 ? (
            <div className="p-8 text-center text-sm text-muted-foreground">
              You don&apos;t have any API keys yet.
            </div>
          ) : (
            <div className="divide-y">
              {apiKeys.map((key) => (
                <div
                  key={key.id}
                  className="grid grid-cols-12 gap-4 p-4 text-sm items-center"
                >
                  <div className="col-span-3 font-medium truncate" title={key.name}>
                    {key.name}
                  </div>
                  <div className="col-span-3 font-mono text-xs text-muted-foreground truncate">
                    {key.prefix && key.suffix
                      ? `${key.prefix}...${key.suffix}`
                      : key.key
                        ? key.key
                        : "sk-..." + key.id.slice(-4)}
                  </div>
                  <div className="col-span-2 text-muted-foreground">
                    {formatDate(key.created_at)}
                  </div>
                  <div className="col-span-2">{getStatusBadge(key.status)}</div>
                  <div className="col-span-2 flex justify-end gap-2">
                    <button
                      onClick={() => setShowDeleteModal(key.id)}
                      className="p-2 text-muted-foreground hover:text-red-500 transition-colors rounded-md hover:bg-muted"
                      title="Revoke key"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      <div>
        <Button onClick={() => setShowCreateModal(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create new secret key
        </Button>
      </div>

      {/* Create Key Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
          <div className="w-full max-w-md rounded-lg border bg-background p-6 shadow-lg">
            {!createdKey ? (
              <>
                <div className="flex items-center justify-between">
                  <h3 className="text-lg font-semibold">
                    Create new secret key
                  </h3>
                  <button
                    onClick={() => setShowCreateModal(false)}
                    className="text-muted-foreground hover:text-foreground"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
                <div className="space-y-6 mt-6">
                  <div className="space-y-2">
                    <label
                      htmlFor="key-name"
                      className="block text-sm font-medium mb-2"
                    >
                      Name
                    </label>
                    <input
                      id="key-name"
                      type="text"
                      placeholder="My Test Key"
                      value={newKeyName}
                      onChange={(e) => setNewKeyName(e.target.value)}
                      className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                      autoFocus
                    />
                    <p className="text-xs text-muted-foreground">
                      Optional: Name your key for easy identification.
                    </p>
                  </div>
                  <div className="flex justify-end gap-2 pt-2">
                    <Button
                      variant="outline"
                      onClick={() => setShowCreateModal(false)}
                    >
                      Cancel
                    </Button>
                    <Button
                      onClick={handleCreateKey}
                      disabled={isCreating || !newKeyName.trim()}
                    >
                      {isCreating ? (
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      ) : null}
                      Create secret key
                    </Button>
                  </div>
                </div>
              </>
            ) : (
              <>
                <div>
                  <h3 className="text-lg font-semibold text-green-600">
                    API Key Created
                  </h3>
                </div>
                <div className="space-y-6 mt-6">
                  <p className="text-sm text-muted-foreground">
                    Please save this secret key somewhere safe and accessible.
                    For security reasons,{" "}
                    <strong>you will not be able to view it again</strong>{" "}
                    through your account. If you lose this secret key, you will
                    need to generate a new one.
                  </p>
                  <div className="relative">
                    <div className="w-full min-h-[40px] rounded-md border border-input bg-muted px-3 py-2 text-sm font-mono break-all pr-10">
                      {createdKey}
                    </div>
                    <button
                      onClick={() => copyToClipboard(createdKey)}
                      className="absolute right-2 top-2 p-1 text-muted-foreground hover:text-foreground"
                      title="Copy to clipboard"
                    >
                      <Copy className="h-4 w-4" />
                    </button>
                  </div>

                  <div className="space-y-3 pt-4 border-t">
                    <div>
                      <p className="text-sm font-medium mb-2">API Base URL</p>
                      <div className="relative">
                        <div className="w-full rounded-md border border-input bg-muted px-3 py-2 text-sm font-mono pr-10">
                          https://api.jan.ai
                        </div>
                        <button
                          onClick={() => copyToClipboard("https://api.jan.ai")}
                          className="absolute right-2 top-2 p-1 text-muted-foreground hover:text-foreground"
                          title="Copy to clipboard"
                        >
                          <Copy className="h-4 w-4" />
                        </button>
                      </div>
                    </div>

                    <div>
                      <p className="text-sm font-medium mb-2">Example Usage</p>
                      <div className="relative">
                        <div className="w-full rounded-md border border-input bg-muted px-3 py-2 text-xs font-mono break-all pr-10 overflow-x-auto">
                          curl https://api.jan.ai/v1/models -H
                          &quot;Authorization: Bearer {createdKey}&quot;
                        </div>
                        <button
                          onClick={() =>
                            copyToClipboard(
                              `curl https://api.jan.ai/v1/models -H "Authorization: Bearer ${createdKey}"`,
                            )
                          }
                          className="absolute right-2 top-2 p-1 text-muted-foreground hover:text-foreground"
                          title="Copy to clipboard"
                        >
                          <Copy className="h-4 w-4" />
                        </button>
                      </div>
                    </div>
                  </div>
                  <div className="flex justify-end pt-2">
                    <Button
                      onClick={() => {
                        setCreatedKey(null);
                        setShowCreateModal(false);
                      }}
                    >
                      Done
                    </Button>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* Delete Confirmation Modal */}
      {showDeleteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
          <div className="w-full max-w-md rounded-lg border bg-background p-6 shadow-lg">
            <div className="mb-4 flex items-center gap-2 text-red-600">
              <AlertTriangle className="h-5 w-5" />
              <h3 className="text-lg font-semibold">Revoke secret key</h3>
            </div>
            <div className="space-y-4">
              <p className="text-sm text-muted-foreground">
                This API key will immediately be disabled. API requests made
                using this key will be rejected, which could cause any systems
                still relying on it to break. Once revoked, you&apos;ll no
                longer be able to view or modify this API key.
              </p>
              <div className="flex justify-end gap-2">
                <Button
                  variant="outline"
                  onClick={() => setShowDeleteModal(null)}
                >
                  Cancel
                </Button>
                <Button
                  variant="destructive"
                  onClick={() => handleDeleteKey(showDeleteModal)}
                >
                  Revoke key
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
