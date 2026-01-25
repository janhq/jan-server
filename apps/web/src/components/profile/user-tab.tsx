import { useAuth } from "@/stores/auth-store";

export function UserTab() {
  const user = useAuth((state) => state.user);

  if (!user) return null;

  return (
    <div className="space-y-8">
      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <label className="text-sm font-medium leading-none">Name</label>
          <div className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm">
            {user.name}
          </div>
          <p className="text-[0.8rem] text-muted-foreground">
            The name associated with this account
          </p>
        </div>
        <div className="space-y-2">
          <label className="text-sm font-medium leading-none">
            Email address
          </label>
          <div className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm">
            {user.email || "Not provided"}
          </div>
          <p className="text-[0.8rem] text-muted-foreground">
            The email address associated with this account
          </p>
        </div>
      </div>

      <div className="space-y-2">
        <label className="text-sm font-medium leading-none">User ID</label>
        <div className="flex h-10 w-full max-w-md rounded-md border border-input bg-background px-3 py-2 text-sm font-mono">
          {user.id}
        </div>
        <p className="text-[0.8rem] text-muted-foreground">
          Unique identifier for this user
        </p>
      </div>
    </div>
  );
}
