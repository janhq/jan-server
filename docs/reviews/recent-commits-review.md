# Recent Commits Review — 2026-04-07

> Automated review of commits merged to main (PRs #484–#486).
> Phase 1 fixes have already been applied for critical security issues.
> This report documents remaining findings that require team discussion or larger refactors.

---

## Finding 1: API Key Access to Inactive Models Lacks Per-Model Authorization

| Field | Value |
|---|---|
| **Severity** | CRITICAL (design concern) |
| **Source PR** | #485 |
| **Priority** | P1 — requires team decision before fix |

### Affected Files
- `services/llm-api/internal/interfaces/httpserver/handlers/chathandler/chat_handler.go` (lines 195–202)
- `services/llm-api/internal/interfaces/httpserver/handlers/messageshandler/messages_handler.go` (lines 78–86, 462–469)
- `services/llm-api/internal/interfaces/httpserver/handlers/modelhandler/provider_handler.go` (lines 196–220)

### Description

When a request arrives with API key authentication (`X-Auth-Method: apikey`), the chat and messages handlers bypass the active-model filter and call `SelectProviderModelForModelPublicIDIncludingInactive`, which uses `FindByModelKey` (no active filter) instead of `FindActiveByModelKey`. This allows any valid API key to access any model — including inactive/disabled models — regardless of which project, organization, or scope the API key belongs to.

The regular (non-API-key) code path correctly restricts access to active models only via `SelectProviderModelForModelPublicID`, which calls `FindActiveByModelKey`. However, the "including inactive" variant performs no additional authorization: it does not verify that the API key has permission to use the specific model or that the model belongs to the same project/org as the key.

This pattern is duplicated identically in three locations: the OpenAI-compatible chat handler, the Anthropic messages handler's `CreateMessage`, and its `CountTokens` method.

### Evidence

**`chathandler/chat_handler.go` (lines 195–202):**
```go
isAPIKeyAuth := strings.EqualFold(reqCtx.GetHeader("X-Auth-Method"), "apikey")
var selectedProviderModel *domainmodel.ProviderModel
var selectedProvider *domainmodel.Provider
if isAPIKeyAuth {
    selectedProviderModel, selectedProvider, err = h.providerHandler.SelectProviderModelForModelPublicIDIncludingInactive(ctx, request.Model)
} else {
    selectedProviderModel, selectedProvider, err = h.providerHandler.SelectProviderModelForModelPublicID(ctx, request.Model)
}
```

**`messageshandler/messages_handler.go` (lines 78–86) — identical pattern in `CreateMessage`:**
```go
isAPIKeyAuth := strings.EqualFold(reqCtx.GetHeader("X-Auth-Method"), "apikey")
var selectedProviderModel *domainmodel.ProviderModel
var selectedProvider *domainmodel.Provider
var err error
if isAPIKeyAuth {
    selectedProviderModel, selectedProvider, err = h.providerHandler.SelectProviderModelForModelPublicIDIncludingInactive(ctx, request.Model)
} else {
    selectedProviderModel, selectedProvider, err = h.providerHandler.SelectProviderModelForModelPublicID(ctx, request.Model)
}
```

**`messageshandler/messages_handler.go` (lines 462–469) — identical pattern in `CountTokens`:**
```go
isAPIKeyAuth := strings.EqualFold(reqCtx.GetHeader("X-Auth-Method"), "apikey")
var selectedProviderModel *domainmodel.ProviderModel
var err error
if isAPIKeyAuth {
    selectedProviderModel, _, err = h.providerHandler.SelectProviderModelForModelPublicIDIncludingInactive(ctx, request.Model)
} else {
    selectedProviderModel, _, err = h.providerHandler.SelectProviderModelForModelPublicID(ctx, request.Model)
}
```

**`modelhandler/provider_handler.go` (lines 196–220) — the "including inactive" method has no authorization check:**
```go
func (providerHandler *ProviderHandler) SelectProviderModelForModelPublicIDIncludingInactive(ctx context.Context, modelPublicID string) (*domainmodel.ProviderModel, *domainmodel.Provider, error) {
    if strings.TrimSpace(modelPublicID) == "" {
        return nil, nil, platformerrors.NewError(...)
    }

    providerModels, err := providerHandler.providerModelService.FindByModelKey(ctx, modelPublicID)
    // ... no ownership/scope check — returns any matching model regardless of API key identity
}
```

Compare with the active-only variant (line 149) which calls `FindActiveByModelKey` — but neither variant checks API key scope.

### Recommended Fix

This requires a team decision on the authorization model before implementation. Options include:

1. **Project-scoped API keys:** Verify that the API key belongs to the same project as the requested model. This requires associating API keys with projects and filtering models by project ownership.
2. **Explicit model allowlists per API key:** Each API key has a list of model IDs it can access, checked at selection time.
3. **Remove inactive model access for API keys:** If there is no business requirement for API keys to access inactive models, use the same `FindActiveByModelKey` path for all auth methods.

Until a decision is made, consider adding an audit log entry whenever an API key accesses an inactive model to measure the scope of this gap.

---

## Finding 2: Duplicate Redirect Logic Across Frontend Files

| Field | Value |
|---|---|
| **Severity** | MEDIUM |
| **Source PR** | #486 |
| **Priority** | P3 — cleanup, no security risk after Phase 1 fix |

### Affected Files
- `apps/web/src/components/form/login.tsx`
- `apps/web/src/routes/__root.tsx`
- `apps/web/src/routes/auth/callback.tsx`
- `apps/web/src/routes/login.tsx`
- `apps/web/src/lib/redirect-utils.ts` (shared utility, added in Phase 1)

### Description

Phase 1 successfully extracted `isAllowedExternalRedirect` into `apps/web/src/lib/redirect-utils.ts`, and all four files now import from that shared location. The security-sensitive URL validation is no longer duplicated.

However, **redirect handling logic** is still duplicated across the files. The same three-step redirect pattern — (1) check for external localhost redirect and append encoded token, (2) check for internal path redirect starting with `/`, (3) fall back to home — is implemented independently in three places with slight variations. Additionally, `getRedirectUrl` in `login.tsx` (form component) and `getRedirectTarget` in `__root.tsx` perform similar URL-parameter extraction.

### Evidence

**External redirect + token pattern — duplicated in 3 files:**

`apps/web/src/routes/login.tsx` (lines 26–36):
```tsx
if (redirectUrl && isAllowedExternalRedirect(redirectUrl)) {
  if (!accessToken) {
    return;
  }
  const bearerToken = `Bearer ${accessToken}`;
  const encodedToken = btoa(bearerToken);
  const target = new URL(redirectUrl);
  target.searchParams.set("token", encodedToken);
  window.location.href = target.toString();
  return;
}
```

`apps/web/src/routes/__root.tsx` (lines 142–149, inside `handleCloseModal`):
```tsx
if (redirectUrl && isAllowedExternalRedirect(redirectUrl)) {
  if (!accessToken) return;
  const bearerToken = `Bearer ${accessToken}`;
  const encodedToken = btoa(bearerToken);
  const target = new URL(redirectUrl);
  target.searchParams.set("token", encodedToken);
  window.location.href = target.toString();
  return;
}
```

`apps/web/src/routes/auth/callback.tsx` (lines 59–66):
```tsx
if (oauthData.redirectUrl && isAllowedExternalRedirect(oauthData.redirectUrl)) {
  const bearerToken = `Bearer ${tokens.access_token}`;
  const encodedToken = btoa(bearerToken);
  const target = new URL(oauthData.redirectUrl);
  target.searchParams.set("token", encodedToken);
  window.location.href = target.toString();
  return;
}
```

**Redirect URL extraction — duplicated in 2 files:**

`apps/web/src/components/form/login.tsx` (lines 34–57, `getRedirectUrl`):
```tsx
const getRedirectUrl = () => {
  const url = new URL(window.location.href);
  const redirectParam = url.searchParams.get(URL_PARAM.REDIRECT);
  if (redirectParam && (redirectParam.startsWith("/") || isAllowedExternalRedirect(redirectParam))) {
    return redirectParam;
  }
  // ... additional cases
};
```

`apps/web/src/routes/__root.tsx` (lines 83–90, `getRedirectTarget`):
```tsx
const getRedirectTarget = () => {
  const url = new URL(window.location.href);
  const redirectParam = url.searchParams.get(URL_PARAM.REDIRECT);
  if (redirectParam && redirectParam.startsWith("/")) {
    return redirectParam;
  }
  return url.pathname + url.search;
};
```

### Recommended Fix

Consolidate the duplicated logic into `apps/web/src/lib/redirect-utils.ts`:

1. Add a `performExternalRedirect(redirectUrl: string, accessToken: string)` function that handles the token-encoding-and-redirect pattern.
2. Add a `getRedirectUrl()` function that extracts and validates the redirect parameter from the current URL.
3. Update all four files to use the shared functions, eliminating the duplicated code.

This is a cleanup task with no security risk (Phase 1 already secured the validation). Priority is P3.

---

## Finding 3: LoginRoute Renders Blank Page During Redirect

| Field | Value |
|---|---|
| **Severity** | LOW |
| **Source PR** | #486 |
| **Priority** | P4 — UX polish |

### Affected Files
- `apps/web/src/routes/login.tsx` (line 47)

### Description

The `LoginRoute` component returns `null` when the user is already authenticated and a redirect is being processed. During the brief period between rendering and the `useEffect` firing to perform the redirect, users see a completely blank page. This is particularly noticeable on slower connections or when the redirect involves an external URL (which requires a full page navigation).

The OAuth callback page (`auth/callback.tsx`) already handles this correctly by showing a spinner with "Completing authentication..." text.

### Evidence

`apps/web/src/routes/login.tsx` (lines 11–48):
```tsx
function LoginRoute() {
  const navigate = useNavigate();
  const isAuthenticated = useAuth((state) => state.isAuthenticated);
  const isGuest = useAuth((state) => state.isGuest);
  const accessToken = useAuth((state) => state.accessToken);

  useEffect(() => {
    if (!isAuthenticated || isGuest) {
      return;
    }
    // ... redirect logic
  }, [accessToken, isAuthenticated, isGuest, navigate]);

  return null;  // <-- blank page during redirect
}
```

For comparison, `apps/web/src/routes/auth/callback.tsx` (lines 102–108) shows the correct pattern:
```tsx
return (
  <div className="flex h-screen items-center justify-center">
    <div className="text-center">
      <div className="mb-4 inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent motion-reduce:animate-[spin_1.5s_linear_infinite]" />
      <p className="text-sm text-gray-600">Completing authentication...</p>
    </div>
  </div>
);
```

### Recommended Fix

Replace `return null` with a loading indicator. The spinner pattern from `auth/callback.tsx` can be reused directly:

```tsx
return (
  <div className="flex h-screen items-center justify-center">
    <div className="text-center">
      <div className="mb-4 inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent motion-reduce:animate-[spin_1.5s_linear_infinite]" />
      <p className="text-sm text-gray-600">Redirecting...</p>
    </div>
  </div>
);
```

Alternatively, extract the spinner into a shared `LoadingScreen` component to avoid further duplication (no such shared component currently exists in the codebase).
