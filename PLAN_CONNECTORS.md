# Connectors Feature Implementation Plan

## Overview

Implement a Connectors feature similar to OpenAI's "Apps/Connectors" and Claude's GitHub integration. This enables users to connect their GitHub, Gmail, Google Drive, and Google Calendar accounts, allowing the AI to search and retrieve data from these services.

**Reference implementations:**
- [OpenAI Connectors/Apps](https://platform.openai.com/docs/guides/tools-connectors-mcp) - MCP wrappers for services
- [Claude GitHub Connect](https://docs.anthropic.com/en/docs/claude-code/github-actions) - Enterprise GitHub integration
- [Google OAuth 2.0](https://developers.google.com/identity/protocols/oauth2)
- [GitHub OAuth Apps](https://docs.github.com/en/apps/oauth-apps/using-oauth-apps/authorizing-oauth-apps)

---

## Architecture Design

### High-Level Flow

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Web App   │───▶│  Kong API   │───▶│   LLM API   │───▶│  Connector  │
│  (React)    │    │   Gateway   │    │   Service   │    │  Providers  │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                                             │
                                             ▼
                                      ┌─────────────┐
                                      │  PostgreSQL │
                                      │ (Tokens DB) │
                                      └─────────────┘
```

### Connector Integration with MCP

```
User Chat → Response API → MCP Service → Connector MCP Client → External API
                                              │
                                              ▼
                                    Token from DB (per-user OAuth)
```

---

## Phase 1: Database & Core Infrastructure

### 1.1 Database Schema

**Migration: `000027_create_connectors.up.sql`**

```sql
-- Connector types enum
CREATE TYPE connector_type AS ENUM ('github', 'gmail', 'google_drive', 'google_calendar');

-- User connector connections
CREATE TABLE connector_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connector_type connector_type NOT NULL,

    -- OAuth tokens (encrypted)
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_type VARCHAR(32) DEFAULT 'Bearer',
    expires_at TIMESTAMP WITH TIME ZONE,

    -- Provider-specific user info
    provider_user_id VARCHAR(255),
    provider_username VARCHAR(255),
    provider_email VARCHAR(255),
    provider_avatar_url TEXT,

    -- Scopes granted
    scopes TEXT[], -- e.g., ['repo', 'read:user'] for GitHub

    -- Status
    is_connected BOOLEAN NOT NULL DEFAULT true,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Ensure one connection per connector type per user
    UNIQUE(user_id, connector_type)
);

-- Index for quick lookups
CREATE INDEX idx_connector_connections_user_type ON connector_connections(user_id, connector_type);
CREATE INDEX idx_connector_connections_connected ON connector_connections(is_connected) WHERE is_connected = true;

-- Connector OAuth states (for CSRF protection during OAuth flow)
CREATE TABLE connector_oauth_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connector_type connector_type NOT NULL,
    state VARCHAR(128) NOT NULL UNIQUE,
    code_verifier VARCHAR(128), -- For PKCE
    redirect_uri TEXT,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Auto-cleanup expired states
CREATE INDEX idx_connector_oauth_states_expires ON connector_oauth_states(expires_at);
```

### 1.2 Domain Layer

**Location: `services/llm-api/internal/domain/connector/`**

```
connector/
├── entity.go          # ConnectorConnection, ConnectorType
├── service.go         # ConnectorService (business logic)
├── repository.go      # Repository interface
├── oauth_service.go   # OAuth flow handling
└── dto.go             # Data transfer objects
```

**entity.go:**
```go
package connector

import "time"

type ConnectorType string

const (
    ConnectorTypeGitHub         ConnectorType = "github"
    ConnectorTypeGmail          ConnectorType = "gmail"
    ConnectorTypeGoogleDrive    ConnectorType = "google_drive"
    ConnectorTypeGoogleCalendar ConnectorType = "google_calendar"
)

type ConnectorConnection struct {
    ID               string
    UserID           uint
    ConnectorType    ConnectorType
    AccessToken      string
    RefreshToken     string
    TokenType        string
    ExpiresAt        *time.Time
    ProviderUserID   string
    ProviderUsername string
    ProviderEmail    string
    ProviderAvatar   string
    Scopes           []string
    IsConnected      bool
    LastSyncAt       *time.Time
    LastError        string
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

type OAuthState struct {
    ID            string
    UserID        uint
    ConnectorType ConnectorType
    State         string
    CodeVerifier  string
    RedirectURI   string
    ExpiresAt     time.Time
    CreatedAt     time.Time
}
```

### 1.3 Infrastructure Layer

**Location: `services/llm-api/internal/infrastructure/connector/`**

```
connector/
├── config.go           # Connector configuration
├── github_client.go    # GitHub API client
├── google_client.go    # Google APIs client (Gmail, Drive, Calendar)
├── token_encryptor.go  # AES encryption for tokens
└── oauth_provider.go   # OAuth flow helpers
```

**config.go (add to existing config):**
```go
// Connector OAuth Configurations
GitHubClientID       string `env:"GITHUB_CLIENT_ID"`
GitHubClientSecret   string `env:"GITHUB_CLIENT_SECRET"`
GitHubEnabled        bool   `env:"GITHUB_CONNECTOR_ENABLED" envDefault:"false"`

GoogleClientID       string `env:"GOOGLE_CLIENT_ID"`
GoogleClientSecret   string `env:"GOOGLE_CLIENT_SECRET"`
GoogleEnabled        bool   `env:"GOOGLE_CONNECTOR_ENABLED" envDefault:"false"`

// Encryption key for storing OAuth tokens (32 bytes for AES-256)
ConnectorTokenEncryptionKey string `env:"CONNECTOR_TOKEN_ENCRYPTION_KEY"`

// OAuth redirect base URL
OAuthRedirectBaseURL string `env:"OAUTH_REDIRECT_BASE_URL" envDefault:"http://localhost:8000"`
```

---

## Phase 2: Backend API Implementation

### 2.1 HTTP Routes

**Location: `services/llm-api/internal/interfaces/httpserver/routes/v1/connectors/`**

**Endpoints:**

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/v1/connectors` | List all available connectors with user's connection status |
| GET | `/v1/connectors/:type` | Get specific connector details and connection status |
| GET | `/v1/connectors/:type/auth-url` | Get OAuth authorization URL |
| GET | `/v1/connectors/:type/callback` | OAuth callback handler |
| POST | `/v1/connectors/:type/connect` | Connect using authorization code |
| DELETE | `/v1/connectors/:type/disconnect` | Disconnect/revoke connector |
| POST | `/v1/connectors/:type/refresh` | Force refresh OAuth tokens |
| GET | `/v1/connectors/:type/status` | Check connection health |

### 2.2 Route Implementation

**connectors_route.go:**
```go
package connectors

import (
    "github.com/gin-gonic/gin"
    "jan-server/services/llm-api/internal/domain/connector"
)

type ConnectorRoute struct {
    service *connector.Service
}

func NewConnectorRoute(service *connector.Service) *ConnectorRoute {
    return &ConnectorRoute{service: service}
}

func (r *ConnectorRoute) RegisterRouter(router gin.IRouter, protectedRouter gin.IRouter) {
    // Public callback route (OAuth redirects here)
    router.GET("/v1/connectors/:type/callback", r.HandleOAuthCallback)

    // Protected routes
    connectors := protectedRouter.Group("/v1/connectors")
    {
        connectors.GET("", r.ListConnectors)
        connectors.GET("/:type", r.GetConnector)
        connectors.GET("/:type/auth-url", r.GetAuthURL)
        connectors.POST("/:type/connect", r.Connect)
        connectors.DELETE("/:type/disconnect", r.Disconnect)
        connectors.POST("/:type/refresh", r.RefreshTokens)
        connectors.GET("/:type/status", r.GetStatus)
    }
}
```

### 2.3 OAuth Flow Implementation

**GitHub OAuth Client:**
```go
package github

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strings"
    "time"
)

const (
    GitHubAuthURL  = "https://github.com/login/oauth/authorize"
    GitHubTokenURL = "https://github.com/login/oauth/access_token"
    GitHubAPIURL   = "https://api.github.com"
)

type GitHubClient struct {
    clientID     string
    clientSecret string
    httpClient   *http.Client
}

type OAuthTokens struct {
    AccessToken  string     `json:"access_token"`
    TokenType    string     `json:"token_type"`
    Scope        string     `json:"scope"`
    RefreshToken string     `json:"refresh_token,omitempty"`
    ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

type GitHubUser struct {
    ID        int64  `json:"id"`
    Login     string `json:"login"`
    Email     string `json:"email"`
    Name      string `json:"name"`
    AvatarURL string `json:"avatar_url"`
}

func NewGitHubClient(clientID, clientSecret string) *GitHubClient {
    return &GitHubClient{
        clientID:     clientID,
        clientSecret: clientSecret,
        httpClient:   &http.Client{Timeout: 30 * time.Second},
    }
}

// GetAuthURL generates the GitHub OAuth authorization URL with PKCE
// Scopes include full repo access for read/write operations (like Claude Code)
func (c *GitHubClient) GetAuthURL(state, redirectURI, codeChallenge string) string {
    params := url.Values{
        "client_id":             {c.clientID},
        "redirect_uri":          {redirectURI},
        "scope":                 {"repo read:user user:email workflow"}, // Full access for code operations
        "state":                 {state},
        "code_challenge":        {codeChallenge},
        "code_challenge_method": {"S256"},
    }
    return GitHubAuthURL + "?" + params.Encode()
}

// ExchangeCode exchanges the authorization code for an access token
func (c *GitHubClient) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*OAuthTokens, error) {
    data := url.Values{
        "client_id":     {c.clientID},
        "client_secret": {c.clientSecret},
        "code":          {code},
        "redirect_uri":  {redirectURI},
        "code_verifier": {codeVerifier},
    }

    req, err := http.NewRequestWithContext(ctx, "POST", GitHubTokenURL, strings.NewReader(data.Encode()))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("Accept", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var tokens OAuthTokens
    if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
        return nil, err
    }

    return &tokens, nil
}

// GetUser fetches the authenticated user's profile
func (c *GitHubClient) GetUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", GitHubAPIURL+"/user", nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+accessToken)
    req.Header.Set("Accept", "application/vnd.github+json")
    req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
    }

    var user GitHubUser
    if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
        return nil, err
    }

    return &user, nil
}
```

**Google OAuth Client (Gmail, Drive, Calendar):**
```go
package google

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strings"
    "time"
)

const (
    GoogleAuthURL   = "https://accounts.google.com/o/oauth2/v2/auth"
    GoogleTokenURL  = "https://oauth2.googleapis.com/token"
    GoogleRevokeURL = "https://oauth2.googleapis.com/revoke"
    GoogleUserURL   = "https://www.googleapis.com/oauth2/v2/userinfo"
)

// Scopes for each Google connector
var GoogleScopes = map[ConnectorType][]string{
    ConnectorTypeGmail: {
        "https://www.googleapis.com/auth/gmail.readonly",
        "https://www.googleapis.com/auth/userinfo.email",
        "https://www.googleapis.com/auth/userinfo.profile",
    },
    ConnectorTypeGoogleDrive: {
        "https://www.googleapis.com/auth/drive.readonly",
        "https://www.googleapis.com/auth/userinfo.email",
        "https://www.googleapis.com/auth/userinfo.profile",
    },
    ConnectorTypeGoogleCalendar: {
        "https://www.googleapis.com/auth/calendar",              // Full calendar access (read/write)
        "https://www.googleapis.com/auth/calendar.events",       // Read/write events
        "https://www.googleapis.com/auth/userinfo.email",
        "https://www.googleapis.com/auth/userinfo.profile",
    },
}

type GoogleClient struct {
    clientID     string
    clientSecret string
    httpClient   *http.Client
}

type GoogleTokens struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token,omitempty"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int    `json:"expires_in"`
    Scope        string `json:"scope"`
}

type GoogleUser struct {
    ID            string `json:"id"`
    Email         string `json:"email"`
    VerifiedEmail bool   `json:"verified_email"`
    Name          string `json:"name"`
    Picture       string `json:"picture"`
}

func NewGoogleClient(clientID, clientSecret string) *GoogleClient {
    return &GoogleClient{
        clientID:     clientID,
        clientSecret: clientSecret,
        httpClient:   &http.Client{Timeout: 30 * time.Second},
    }
}

// GetAuthURL generates the Google OAuth authorization URL with PKCE
func (c *GoogleClient) GetAuthURL(connectorType ConnectorType, state, redirectURI, codeChallenge string) string {
    scopes := GoogleScopes[connectorType]
    params := url.Values{
        "client_id":             {c.clientID},
        "redirect_uri":          {redirectURI},
        "response_type":         {"code"},
        "scope":                 {strings.Join(scopes, " ")},
        "state":                 {state},
        "access_type":           {"offline"},  // REQUIRED for refresh token
        "prompt":                {"consent"},  // Force consent to get refresh token
        "code_challenge":        {codeChallenge},
        "code_challenge_method": {"S256"},
        "include_granted_scopes": {"true"},
    }
    return GoogleAuthURL + "?" + params.Encode()
}

// ExchangeCode exchanges the authorization code for tokens
func (c *GoogleClient) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*GoogleTokens, error) {
    data := url.Values{
        "client_id":     {c.clientID},
        "client_secret": {c.clientSecret},
        "code":          {code},
        "code_verifier": {codeVerifier},
        "grant_type":    {"authorization_code"},
        "redirect_uri":  {redirectURI},
    }

    req, err := http.NewRequestWithContext(ctx, "POST", GoogleTokenURL, strings.NewReader(data.Encode()))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("Google token exchange failed: %d", resp.StatusCode)
    }

    var tokens GoogleTokens
    if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
        return nil, err
    }

    return &tokens, nil
}

// RefreshToken refreshes an expired access token
func (c *GoogleClient) RefreshToken(ctx context.Context, refreshToken string) (*GoogleTokens, error) {
    data := url.Values{
        "client_id":     {c.clientID},
        "client_secret": {c.clientSecret},
        "refresh_token": {refreshToken},
        "grant_type":    {"refresh_token"},
    }

    req, err := http.NewRequestWithContext(ctx, "POST", GoogleTokenURL, strings.NewReader(data.Encode()))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("Google token refresh failed: %d", resp.StatusCode)
    }

    var tokens GoogleTokens
    if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
        return nil, err
    }

    return &tokens, nil
}

// RevokeToken revokes an access or refresh token
func (c *GoogleClient) RevokeToken(ctx context.Context, token string) error {
    data := url.Values{"token": {token}}
    req, err := http.NewRequestWithContext(ctx, "POST", GoogleRevokeURL, strings.NewReader(data.Encode()))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Google returns 200 on successful revocation
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("Google token revocation failed: %d", resp.StatusCode)
    }
    return nil
}

// GetUser fetches the authenticated user's profile
func (c *GoogleClient) GetUser(ctx context.Context, accessToken string) (*GoogleUser, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", GoogleUserURL, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+accessToken)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("Google user API error: %d", resp.StatusCode)
    }

    var user GoogleUser
    if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
        return nil, err
    }

    return &user, nil
}
```

---

## Phase 3: MCP Tool Integration

### 3.1 Connector MCP Clients

Each connector exposes MCP-compatible tools that the AI can use to search/retrieve data.

**Location: `services/mcp-tools/internal/domain/connectors/`**

**GitHub MCP Tools with API Implementation:**
```go
package connectors

import (
    "context"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"

    "github.com/mark3labs/mcp-go/mcp"
)

const GitHubAPIURL = "https://api.github.com"

type GitHubMCPClient struct {
    httpClient    *http.Client
    tokenProvider TokenProvider // Gets user's OAuth token
}

func (c *GitHubMCPClient) GetTools() []mcp.Tool {
    return []mcp.Tool{
        // ============ READ OPERATIONS ============
        {
            Name:        "github_search_repositories",
            Description: "Search GitHub repositories the user has access to",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "query": map[string]string{"type": "string", "description": "Search query (e.g., 'react language:typescript')"},
                    "sort":  map[string]interface{}{"type": "string", "enum": []string{"stars", "forks", "updated", "best-match"}, "default": "best-match"},
                    "per_page": map[string]interface{}{"type": "integer", "default": 10, "maximum": 30},
                },
                Required: []string{"query"},
            },
        },
        {
            Name:        "github_search_issues",
            Description: "Search issues and pull requests across repositories",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "query":    map[string]string{"type": "string", "description": "Search query (e.g., 'bug is:open label:help-wanted')"},
                    "type":     map[string]interface{}{"type": "string", "enum": []string{"issue", "pr", "all"}, "default": "all"},
                    "state":    map[string]interface{}{"type": "string", "enum": []string{"open", "closed", "all"}, "default": "open"},
                    "per_page": map[string]interface{}{"type": "integer", "default": 10, "maximum": 30},
                },
                Required: []string{"query"},
            },
        },
        {
            Name:        "github_get_file_content",
            Description: "Get the content of a file from a GitHub repository",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "owner": map[string]string{"type": "string", "description": "Repository owner (username or org)"},
                    "repo":  map[string]string{"type": "string", "description": "Repository name"},
                    "path":  map[string]string{"type": "string", "description": "File path (e.g., 'src/main.ts')"},
                    "ref":   map[string]interface{}{"type": "string", "description": "Branch, tag, or commit SHA", "default": "main"},
                },
                Required: []string{"owner", "repo", "path"},
            },
        },
        {
            Name:        "github_list_pull_requests",
            Description: "List pull requests for a repository",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "owner":    map[string]string{"type": "string", "description": "Repository owner"},
                    "repo":     map[string]string{"type": "string", "description": "Repository name"},
                    "state":    map[string]interface{}{"type": "string", "enum": []string{"open", "closed", "all"}, "default": "open"},
                    "per_page": map[string]interface{}{"type": "integer", "default": 10, "maximum": 30},
                },
                Required: []string{"owner", "repo"},
            },
        },
        {
            Name:        "github_list_user_repos",
            Description: "List repositories for the authenticated user",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "type":     map[string]interface{}{"type": "string", "enum": []string{"all", "owner", "member"}, "default": "all"},
                    "sort":     map[string]interface{}{"type": "string", "enum": []string{"created", "updated", "pushed", "full_name"}, "default": "updated"},
                    "per_page": map[string]interface{}{"type": "integer", "default": 10, "maximum": 30},
                },
            },
        },
        {
            Name:        "github_get_pull_request",
            Description: "Get details of a specific pull request including diff stats",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "owner":       map[string]string{"type": "string", "description": "Repository owner"},
                    "repo":        map[string]string{"type": "string", "description": "Repository name"},
                    "pull_number": map[string]interface{}{"type": "integer", "description": "Pull request number"},
                },
                Required: []string{"owner", "repo", "pull_number"},
            },
        },
        {
            Name:        "github_list_branches",
            Description: "List branches in a repository",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "owner":    map[string]string{"type": "string", "description": "Repository owner"},
                    "repo":     map[string]string{"type": "string", "description": "Repository name"},
                    "per_page": map[string]interface{}{"type": "integer", "default": 30, "maximum": 100},
                },
                Required: []string{"owner", "repo"},
            },
        },

        // ============ WRITE OPERATIONS (like Claude Code) ============
        {
            Name:        "github_create_branch",
            Description: "Create a new branch from an existing branch or commit",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "owner":       map[string]string{"type": "string", "description": "Repository owner"},
                    "repo":        map[string]string{"type": "string", "description": "Repository name"},
                    "branch_name": map[string]string{"type": "string", "description": "Name for the new branch (e.g., 'feature/add-login')"},
                    "from_branch": map[string]interface{}{"type": "string", "description": "Base branch to create from", "default": "main"},
                },
                Required: []string{"owner", "repo", "branch_name"},
            },
        },
        {
            Name:        "github_create_or_update_file",
            Description: "Create a new file or update an existing file in a repository (commits the change)",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "owner":   map[string]string{"type": "string", "description": "Repository owner"},
                    "repo":    map[string]string{"type": "string", "description": "Repository name"},
                    "path":    map[string]string{"type": "string", "description": "File path (e.g., 'src/utils/helper.ts')"},
                    "content": map[string]string{"type": "string", "description": "File content (plain text, will be base64 encoded)"},
                    "message": map[string]string{"type": "string", "description": "Commit message (e.g., 'feat: add helper function')"},
                    "branch":  map[string]interface{}{"type": "string", "description": "Branch to commit to", "default": "main"},
                },
                Required: []string{"owner", "repo", "path", "content", "message"},
            },
        },
        {
            Name:        "github_delete_file",
            Description: "Delete a file from a repository (commits the deletion)",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "owner":   map[string]string{"type": "string", "description": "Repository owner"},
                    "repo":    map[string]string{"type": "string", "description": "Repository name"},
                    "path":    map[string]string{"type": "string", "description": "File path to delete"},
                    "message": map[string]string{"type": "string", "description": "Commit message"},
                    "branch":  map[string]interface{}{"type": "string", "description": "Branch to commit to", "default": "main"},
                },
                Required: []string{"owner", "repo", "path", "message"},
            },
        },
        {
            Name:        "github_create_pull_request",
            Description: "Create a new pull request to merge changes from one branch to another",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "owner": map[string]string{"type": "string", "description": "Repository owner"},
                    "repo":  map[string]string{"type": "string", "description": "Repository name"},
                    "title": map[string]string{"type": "string", "description": "PR title (e.g., 'feat: add user authentication')"},
                    "body":  map[string]string{"type": "string", "description": "PR description in markdown"},
                    "head":  map[string]string{"type": "string", "description": "Branch containing changes (e.g., 'feature/auth')"},
                    "base":  map[string]interface{}{"type": "string", "description": "Branch to merge into", "default": "main"},
                    "draft": map[string]interface{}{"type": "boolean", "description": "Create as draft PR", "default": false},
                },
                Required: []string{"owner", "repo", "title", "head"},
            },
        },
        {
            Name:        "github_merge_pull_request",
            Description: "Merge a pull request",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "owner":        map[string]string{"type": "string", "description": "Repository owner"},
                    "repo":         map[string]string{"type": "string", "description": "Repository name"},
                    "pull_number":  map[string]interface{}{"type": "integer", "description": "Pull request number"},
                    "commit_title": map[string]interface{}{"type": "string", "description": "Custom merge commit title (optional)"},
                    "merge_method": map[string]interface{}{"type": "string", "enum": []string{"merge", "squash", "rebase"}, "default": "squash"},
                },
                Required: []string{"owner", "repo", "pull_number"},
            },
        },
        {
            Name:        "github_add_pr_review",
            Description: "Add a review to a pull request (approve, request changes, or comment)",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "owner":       map[string]string{"type": "string", "description": "Repository owner"},
                    "repo":        map[string]string{"type": "string", "description": "Repository name"},
                    "pull_number": map[string]interface{}{"type": "integer", "description": "Pull request number"},
                    "body":        map[string]string{"type": "string", "description": "Review comment"},
                    "event":       map[string]interface{}{"type": "string", "enum": []string{"APPROVE", "REQUEST_CHANGES", "COMMENT"}, "description": "Review action"},
                },
                Required: []string{"owner", "repo", "pull_number", "event"},
            },
        },
        {
            Name:        "github_create_issue",
            Description: "Create a new issue in a repository",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "owner":     map[string]string{"type": "string", "description": "Repository owner"},
                    "repo":      map[string]string{"type": "string", "description": "Repository name"},
                    "title":     map[string]string{"type": "string", "description": "Issue title"},
                    "body":      map[string]string{"type": "string", "description": "Issue description in markdown"},
                    "labels":    map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}, "description": "Labels to add (e.g., ['bug', 'priority:high'])"},
                    "assignees": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}, "description": "Usernames to assign"},
                },
                Required: []string{"owner", "repo", "title"},
            },
        },
        {
            Name:        "github_add_comment",
            Description: "Add a comment to an issue or pull request",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "owner":        map[string]string{"type": "string", "description": "Repository owner"},
                    "repo":         map[string]string{"type": "string", "description": "Repository name"},
                    "issue_number": map[string]interface{}{"type": "integer", "description": "Issue or PR number"},
                    "body":         map[string]string{"type": "string", "description": "Comment body in markdown"},
                },
                Required: []string{"owner", "repo", "issue_number", "body"},
            },
        },
        {
            Name:        "github_update_issue",
            Description: "Update an existing issue (title, body, state, labels, assignees)",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "owner":        map[string]string{"type": "string", "description": "Repository owner"},
                    "repo":         map[string]string{"type": "string", "description": "Repository name"},
                    "issue_number": map[string]interface{}{"type": "integer", "description": "Issue number"},
                    "title":        map[string]interface{}{"type": "string", "description": "New title (optional)"},
                    "body":         map[string]interface{}{"type": "string", "description": "New body (optional)"},
                    "state":        map[string]interface{}{"type": "string", "enum": []string{"open", "closed"}, "description": "Issue state"},
                    "labels":       map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}, "description": "Labels to set"},
                },
                Required: []string{"owner", "repo", "issue_number"},
            },
        },
    }
}

// CallTool executes a GitHub tool
func (c *GitHubMCPClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    token, err := c.tokenProvider.GetToken(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get GitHub token: %w", err)
    }

    switch toolName {
    case "github_search_repositories":
        return c.searchRepositories(ctx, token, args)
    case "github_search_issues":
        return c.searchIssues(ctx, token, args)
    case "github_get_file_content":
        return c.getFileContent(ctx, token, args)
    case "github_list_pull_requests":
        return c.listPullRequests(ctx, token, args)
    case "github_list_user_repos":
        return c.listUserRepos(ctx, token, args)
    default:
        return nil, fmt.Errorf("unknown tool: %s", toolName)
    }
}

// searchRepositories calls GET /search/repositories
func (c *GitHubMCPClient) searchRepositories(ctx context.Context, token string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    query := args["query"].(string)
    sort := getStringOrDefault(args, "sort", "best-match")
    perPage := getIntOrDefault(args, "per_page", 10)

    url := fmt.Sprintf("%s/search/repositories?q=%s&sort=%s&per_page=%d",
        GitHubAPIURL, url.QueryEscape(query), sort, perPage)

    result, err := c.makeRequest(ctx, token, "GET", url)
    if err != nil {
        return nil, err
    }

    return &mcp.CallToolResult{
        Content: []mcp.Content{{Type: "text", Text: result}},
    }, nil
}

// getFileContent calls GET /repos/{owner}/{repo}/contents/{path}
func (c *GitHubMCPClient) getFileContent(ctx context.Context, token string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    owner := args["owner"].(string)
    repo := args["repo"].(string)
    path := args["path"].(string)
    ref := getStringOrDefault(args, "ref", "main")

    url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s",
        GitHubAPIURL, owner, repo, path, ref)

    result, err := c.makeRequest(ctx, token, "GET", url)
    if err != nil {
        return nil, err
    }

    // Decode base64 content if present
    var content struct {
        Content  string `json:"content"`
        Encoding string `json:"encoding"`
        Name     string `json:"name"`
        Path     string `json:"path"`
    }
    if err := json.Unmarshal([]byte(result), &content); err == nil && content.Encoding == "base64" {
        decoded, _ := base64.StdEncoding.DecodeString(content.Content)
        return &mcp.CallToolResult{
            Content: []mcp.Content{{Type: "text", Text: string(decoded)}},
        }, nil
    }

    return &mcp.CallToolResult{
        Content: []mcp.Content{{Type: "text", Text: result}},
    }, nil
}

func (c *GitHubMCPClient) makeRequest(ctx context.Context, token, method, url string) (string, error) {
    req, err := http.NewRequestWithContext(ctx, method, url, nil)
    if err != nil {
        return "", err
    }
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Accept", "application/vnd.github+json")
    req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return "", fmt.Errorf("GitHub API error: %d", resp.StatusCode)
    }

    var result json.RawMessage
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }

    return string(result), nil
}

// ============ WRITE OPERATION IMPLEMENTATIONS ============

// createBranch creates a new branch from an existing branch
// API: POST /repos/{owner}/{repo}/git/refs
func (c *GitHubMCPClient) createBranch(ctx context.Context, token string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    owner := args["owner"].(string)
    repo := args["repo"].(string)
    branchName := args["branch_name"].(string)
    fromBranch := getStringOrDefault(args, "from_branch", "main")

    // First, get the SHA of the base branch
    refURL := fmt.Sprintf("%s/repos/%s/%s/git/ref/heads/%s", GitHubAPIURL, owner, repo, fromBranch)
    refResp, err := c.makeRequest(ctx, token, "GET", refURL)
    if err != nil {
        return nil, fmt.Errorf("failed to get base branch: %w", err)
    }

    var refData struct {
        Object struct {
            SHA string `json:"sha"`
        } `json:"object"`
    }
    json.Unmarshal([]byte(refResp), &refData)

    // Create the new branch
    createURL := fmt.Sprintf("%s/repos/%s/%s/git/refs", GitHubAPIURL, owner, repo)
    payload := map[string]string{
        "ref": fmt.Sprintf("refs/heads/%s", branchName),
        "sha": refData.Object.SHA,
    }

    result, err := c.makeRequestWithBody(ctx, token, "POST", createURL, payload)
    if err != nil {
        return nil, err
    }

    return &mcp.CallToolResult{
        Content: []mcp.Content{{Type: "text", Text: fmt.Sprintf("Branch '%s' created successfully from '%s'\n%s", branchName, fromBranch, result)}},
    }, nil
}

// createOrUpdateFile creates or updates a file in a repository
// API: PUT /repos/{owner}/{repo}/contents/{path}
func (c *GitHubMCPClient) createOrUpdateFile(ctx context.Context, token string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    owner := args["owner"].(string)
    repo := args["repo"].(string)
    path := args["path"].(string)
    content := args["content"].(string)
    message := args["message"].(string)
    branch := getStringOrDefault(args, "branch", "main")

    // Check if file exists to get its SHA (required for updates)
    var existingSHA string
    fileURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", GitHubAPIURL, owner, repo, path, branch)
    if fileResp, err := c.makeRequest(ctx, token, "GET", fileURL); err == nil {
        var fileData struct {
            SHA string `json:"sha"`
        }
        json.Unmarshal([]byte(fileResp), &fileData)
        existingSHA = fileData.SHA
    }

    // Create/update the file
    putURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s", GitHubAPIURL, owner, repo, path)
    payload := map[string]interface{}{
        "message": message,
        "content": base64.StdEncoding.EncodeToString([]byte(content)),
        "branch":  branch,
    }
    if existingSHA != "" {
        payload["sha"] = existingSHA
    }

    result, err := c.makeRequestWithBody(ctx, token, "PUT", putURL, payload)
    if err != nil {
        return nil, err
    }

    action := "created"
    if existingSHA != "" {
        action = "updated"
    }

    return &mcp.CallToolResult{
        Content: []mcp.Content{{Type: "text", Text: fmt.Sprintf("File '%s' %s successfully\n%s", path, action, result)}},
    }, nil
}

// createPullRequest creates a new pull request
// API: POST /repos/{owner}/{repo}/pulls
func (c *GitHubMCPClient) createPullRequest(ctx context.Context, token string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    owner := args["owner"].(string)
    repo := args["repo"].(string)
    title := args["title"].(string)
    head := args["head"].(string)
    base := getStringOrDefault(args, "base", "main")
    body := getStringOrDefault(args, "body", "")
    draft := getBoolOrDefault(args, "draft", false)

    prURL := fmt.Sprintf("%s/repos/%s/%s/pulls", GitHubAPIURL, owner, repo)
    payload := map[string]interface{}{
        "title": title,
        "head":  head,
        "base":  base,
        "body":  body,
        "draft": draft,
    }

    result, err := c.makeRequestWithBody(ctx, token, "POST", prURL, payload)
    if err != nil {
        return nil, err
    }

    var prData struct {
        Number  int    `json:"number"`
        HTMLURL string `json:"html_url"`
    }
    json.Unmarshal([]byte(result), &prData)

    return &mcp.CallToolResult{
        Content: []mcp.Content{{Type: "text", Text: fmt.Sprintf("Pull request #%d created: %s\n%s", prData.Number, prData.HTMLURL, result)}},
    }, nil
}

// mergePullRequest merges a pull request
// API: PUT /repos/{owner}/{repo}/pulls/{pull_number}/merge
func (c *GitHubMCPClient) mergePullRequest(ctx context.Context, token string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    owner := args["owner"].(string)
    repo := args["repo"].(string)
    pullNumber := int(args["pull_number"].(float64))
    mergeMethod := getStringOrDefault(args, "merge_method", "squash")
    commitTitle := getStringOrDefault(args, "commit_title", "")

    mergeURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/merge", GitHubAPIURL, owner, repo, pullNumber)
    payload := map[string]interface{}{
        "merge_method": mergeMethod,
    }
    if commitTitle != "" {
        payload["commit_title"] = commitTitle
    }

    result, err := c.makeRequestWithBody(ctx, token, "PUT", mergeURL, payload)
    if err != nil {
        return nil, err
    }

    return &mcp.CallToolResult{
        Content: []mcp.Content{{Type: "text", Text: fmt.Sprintf("Pull request #%d merged successfully\n%s", pullNumber, result)}},
    }, nil
}

// addPRReview adds a review to a pull request
// API: POST /repos/{owner}/{repo}/pulls/{pull_number}/reviews
func (c *GitHubMCPClient) addPRReview(ctx context.Context, token string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    owner := args["owner"].(string)
    repo := args["repo"].(string)
    pullNumber := int(args["pull_number"].(float64))
    event := args["event"].(string) // APPROVE, REQUEST_CHANGES, COMMENT
    body := getStringOrDefault(args, "body", "")

    reviewURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/reviews", GitHubAPIURL, owner, repo, pullNumber)
    payload := map[string]interface{}{
        "event": event,
        "body":  body,
    }

    result, err := c.makeRequestWithBody(ctx, token, "POST", reviewURL, payload)
    if err != nil {
        return nil, err
    }

    return &mcp.CallToolResult{
        Content: []mcp.Content{{Type: "text", Text: fmt.Sprintf("Review added to PR #%d: %s\n%s", pullNumber, event, result)}},
    }, nil
}

// createIssue creates a new issue
// API: POST /repos/{owner}/{repo}/issues
func (c *GitHubMCPClient) createIssue(ctx context.Context, token string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    owner := args["owner"].(string)
    repo := args["repo"].(string)
    title := args["title"].(string)
    body := getStringOrDefault(args, "body", "")

    issueURL := fmt.Sprintf("%s/repos/%s/%s/issues", GitHubAPIURL, owner, repo)
    payload := map[string]interface{}{
        "title": title,
        "body":  body,
    }

    if labels, ok := args["labels"].([]interface{}); ok {
        payload["labels"] = labels
    }
    if assignees, ok := args["assignees"].([]interface{}); ok {
        payload["assignees"] = assignees
    }

    result, err := c.makeRequestWithBody(ctx, token, "POST", issueURL, payload)
    if err != nil {
        return nil, err
    }

    var issueData struct {
        Number  int    `json:"number"`
        HTMLURL string `json:"html_url"`
    }
    json.Unmarshal([]byte(result), &issueData)

    return &mcp.CallToolResult{
        Content: []mcp.Content{{Type: "text", Text: fmt.Sprintf("Issue #%d created: %s\n%s", issueData.Number, issueData.HTMLURL, result)}},
    }, nil
}

// addComment adds a comment to an issue or PR
// API: POST /repos/{owner}/{repo}/issues/{issue_number}/comments
func (c *GitHubMCPClient) addComment(ctx context.Context, token string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    owner := args["owner"].(string)
    repo := args["repo"].(string)
    issueNumber := int(args["issue_number"].(float64))
    body := args["body"].(string)

    commentURL := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", GitHubAPIURL, owner, repo, issueNumber)
    payload := map[string]string{
        "body": body,
    }

    result, err := c.makeRequestWithBody(ctx, token, "POST", commentURL, payload)
    if err != nil {
        return nil, err
    }

    return &mcp.CallToolResult{
        Content: []mcp.Content{{Type: "text", Text: fmt.Sprintf("Comment added to #%d\n%s", issueNumber, result)}},
    }, nil
}

// Helper to make requests with JSON body
func (c *GitHubMCPClient) makeRequestWithBody(ctx context.Context, token, method, url string, payload interface{}) (string, error) {
    body, _ := json.Marshal(payload)
    req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
    if err != nil {
        return "", err
    }
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Accept", "application/vnd.github+json")
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    respBody, _ := io.ReadAll(resp.Body)

    if resp.StatusCode >= 400 {
        return "", fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(respBody))
    }

    return string(respBody), nil
}
```

**Gmail MCP Tools with API Implementation:**

API Base URL: `https://gmail.googleapis.com/gmail/v1`

| Tool | API Endpoint | Description |
|------|--------------|-------------|
| `gmail_search_emails` | `GET /users/me/messages?q={query}` | Search emails using Gmail search syntax |
| `gmail_get_email` | `GET /users/me/messages/{id}` | Get full email content |
| `gmail_list_labels` | `GET /users/me/labels` | List all labels/folders |

```go
// Gmail search query examples:
// - "from:boss@company.com" - Emails from specific sender
// - "subject:meeting" - Subject contains "meeting"
// - "after:2024/01/01 before:2024/12/31" - Date range
// - "has:attachment" - Emails with attachments
// - "is:unread" - Unread emails only
// - "in:inbox" - Inbox only
// - "label:important" - With specific label

func (c *GmailMCPClient) searchEmails(ctx context.Context, token string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    query := args["query"].(string)
    maxResults := getIntOrDefault(args, "max_results", 10)

    // API: GET https://gmail.googleapis.com/gmail/v1/users/me/messages?q={query}&maxResults={max}
    searchURL := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages?q=%s&maxResults=%d",
        url.QueryEscape(query), maxResults)

    req, _ := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
    req.Header.Set("Authorization", "Bearer "+token)

    // Returns list of message IDs, then fetch each message's metadata
    // ...
}
```

**Google Drive MCP Tools with API Implementation:**

API Base URL: `https://www.googleapis.com/drive/v3`

| Tool | API Endpoint | Description |
|------|--------------|-------------|
| `drive_search_files` | `GET /files?q={query}` | Search files with Drive query syntax |
| `drive_get_file_content` | `GET /files/{id}/export` or `GET /files/{id}?alt=media` | Get file content |
| `drive_list_recent` | `GET /files?orderBy=modifiedTime desc` | List recently modified |

```go
// Drive query syntax examples:
// - "name contains 'report'" - Name contains text
// - "mimeType='application/vnd.google-apps.document'" - Google Docs only
// - "mimeType='application/vnd.google-apps.spreadsheet'" - Google Sheets only
// - "modifiedTime > '2024-01-01T00:00:00'" - Modified after date
// - "'me' in owners" - Files owned by user
// - "trashed = false" - Exclude trashed files

// Google Workspace MIME types:
var DriveMimeTypes = map[string]string{
    "document":     "application/vnd.google-apps.document",
    "spreadsheet":  "application/vnd.google-apps.spreadsheet",
    "presentation": "application/vnd.google-apps.presentation",
    "folder":       "application/vnd.google-apps.folder",
}

// Export formats for Google Workspace files:
// - Google Docs → text/plain, text/html, application/pdf
// - Google Sheets → text/csv, application/pdf
// - Google Slides → application/pdf, text/plain

func (c *DriveMCPClient) getFileContent(ctx context.Context, token string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    fileID := args["file_id"].(string)

    // For Google Workspace files, use export endpoint
    // API: GET https://www.googleapis.com/drive/v3/files/{fileId}/export?mimeType=text/plain
    exportURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s/export?mimeType=text/plain", fileID)

    // For binary files, use alt=media
    // API: GET https://www.googleapis.com/drive/v3/files/{fileId}?alt=media

    req, _ := http.NewRequestWithContext(ctx, "GET", exportURL, nil)
    req.Header.Set("Authorization", "Bearer "+token)
    // ...
}
```

**Google Calendar MCP Tools with API Implementation:**

API Base URL: `https://www.googleapis.com/calendar/v3`

**Read Operations:**
| Tool | API Endpoint | Description |
|------|--------------|-------------|
| `calendar_list_events` | `GET /calendars/{id}/events` | List events in time range |
| `calendar_search_events` | `GET /calendars/{id}/events?q={query}` | Search events by keyword |
| `calendar_get_event` | `GET /calendars/{id}/events/{eventId}` | Get event details |
| `calendar_list_calendars` | `GET /users/me/calendarList` | List user's calendars |

**Write Operations:**
| Tool | API Endpoint | Description |
|------|--------------|-------------|
| `calendar_create_event` | `POST /calendars/{id}/events` | Create a new event |
| `calendar_update_event` | `PATCH /calendars/{id}/events/{eventId}` | Update an existing event |
| `calendar_delete_event` | `DELETE /calendars/{id}/events/{eventId}` | Delete an event |
| `calendar_quick_add` | `POST /calendars/{id}/events/quickAdd` | Create event from text (e.g., "Meeting tomorrow 3pm") |

```go
const CalendarAPIURL = "https://www.googleapis.com/calendar/v3"

func (c *GoogleCalendarMCPClient) GetTools() []mcp.Tool {
    return []mcp.Tool{
        // ============ READ OPERATIONS ============
        {
            Name:        "calendar_list_events",
            Description: "List upcoming calendar events within a time range",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "time_min":    map[string]string{"type": "string", "description": "Start time (RFC3339, e.g., '2025-01-28T00:00:00Z'). Defaults to now."},
                    "time_max":    map[string]string{"type": "string", "description": "End time (RFC3339). Defaults to 7 days from now."},
                    "calendar_id": map[string]interface{}{"type": "string", "default": "primary"},
                    "max_results": map[string]interface{}{"type": "integer", "default": 10, "maximum": 50},
                },
            },
        },
        {
            Name:        "calendar_search_events",
            Description: "Search calendar events by keyword in title or description",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "query":       map[string]string{"type": "string", "description": "Search query"},
                    "calendar_id": map[string]interface{}{"type": "string", "default": "primary"},
                    "max_results": map[string]interface{}{"type": "integer", "default": 10},
                },
                Required: []string{"query"},
            },
        },
        {
            Name:        "calendar_get_event",
            Description: "Get detailed information about a specific event",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "event_id":    map[string]string{"type": "string", "description": "The event ID"},
                    "calendar_id": map[string]interface{}{"type": "string", "default": "primary"},
                },
                Required: []string{"event_id"},
            },
        },
        {
            Name:        "calendar_list_calendars",
            Description: "List all calendars the user has access to",
            InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}},
        },

        // ============ WRITE OPERATIONS ============
        {
            Name:        "calendar_create_event",
            Description: "Create a new calendar event",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "summary":     map[string]string{"type": "string", "description": "Event title"},
                    "description": map[string]interface{}{"type": "string", "description": "Event description"},
                    "location":    map[string]interface{}{"type": "string", "description": "Event location"},
                    "start_time":  map[string]string{"type": "string", "description": "Start time (RFC3339, e.g., '2025-01-28T10:00:00-08:00')"},
                    "end_time":    map[string]string{"type": "string", "description": "End time (RFC3339)"},
                    "timezone":    map[string]interface{}{"type": "string", "description": "Timezone (e.g., 'America/Los_Angeles')", "default": "UTC"},
                    "attendees":   map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}, "description": "List of attendee emails"},
                    "calendar_id": map[string]interface{}{"type": "string", "default": "primary"},
                    "send_notifications": map[string]interface{}{"type": "boolean", "default": true, "description": "Send email notifications to attendees"},
                },
                Required: []string{"summary", "start_time", "end_time"},
            },
        },
        {
            Name:        "calendar_update_event",
            Description: "Update an existing calendar event",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "event_id":    map[string]string{"type": "string", "description": "The event ID to update"},
                    "summary":     map[string]interface{}{"type": "string", "description": "New event title"},
                    "description": map[string]interface{}{"type": "string", "description": "New description"},
                    "location":    map[string]interface{}{"type": "string", "description": "New location"},
                    "start_time":  map[string]interface{}{"type": "string", "description": "New start time (RFC3339)"},
                    "end_time":    map[string]interface{}{"type": "string", "description": "New end time (RFC3339)"},
                    "calendar_id": map[string]interface{}{"type": "string", "default": "primary"},
                    "send_notifications": map[string]interface{}{"type": "boolean", "default": true},
                },
                Required: []string{"event_id"},
            },
        },
        {
            Name:        "calendar_delete_event",
            Description: "Delete a calendar event",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "event_id":    map[string]string{"type": "string", "description": "The event ID to delete"},
                    "calendar_id": map[string]interface{}{"type": "string", "default": "primary"},
                    "send_notifications": map[string]interface{}{"type": "boolean", "default": true, "description": "Notify attendees about cancellation"},
                },
                Required: []string{"event_id"},
            },
        },
        {
            Name:        "calendar_quick_add",
            Description: "Quickly create an event using natural language (e.g., 'Meeting with Bob tomorrow at 3pm')",
            InputSchema: mcp.ToolInputSchema{
                Type: "object",
                Properties: map[string]interface{}{
                    "text":        map[string]string{"type": "string", "description": "Natural language event description"},
                    "calendar_id": map[string]interface{}{"type": "string", "default": "primary"},
                    "send_notifications": map[string]interface{}{"type": "boolean", "default": true},
                },
                Required: []string{"text"},
            },
        },
    }
}

// ============ WRITE OPERATION IMPLEMENTATIONS ============

// createEvent creates a new calendar event
// API: POST /calendars/{calendarId}/events
func (c *GoogleCalendarMCPClient) createEvent(ctx context.Context, token string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    calendarID := getStringOrDefault(args, "calendar_id", "primary")
    sendNotifications := getBoolOrDefault(args, "send_notifications", true)

    event := map[string]interface{}{
        "summary": args["summary"],
        "start": map[string]string{
            "dateTime": args["start_time"].(string),
            "timeZone": getStringOrDefault(args, "timezone", "UTC"),
        },
        "end": map[string]string{
            "dateTime": args["end_time"].(string),
            "timeZone": getStringOrDefault(args, "timezone", "UTC"),
        },
    }

    if desc, ok := args["description"]; ok {
        event["description"] = desc
    }
    if loc, ok := args["location"]; ok {
        event["location"] = loc
    }
    if attendees, ok := args["attendees"].([]interface{}); ok {
        attendeeList := make([]map[string]string, len(attendees))
        for i, email := range attendees {
            attendeeList[i] = map[string]string{"email": email.(string)}
        }
        event["attendees"] = attendeeList
    }

    createURL := fmt.Sprintf("%s/calendars/%s/events?sendUpdates=%s",
        CalendarAPIURL, url.PathEscape(calendarID),
        map[bool]string{true: "all", false: "none"}[sendNotifications])

    result, err := c.makeRequestWithBody(ctx, token, "POST", createURL, event)
    if err != nil {
        return nil, err
    }

    var created struct {
        ID      string `json:"id"`
        HTMLURL string `json:"htmlLink"`
        Summary string `json:"summary"`
    }
    json.Unmarshal([]byte(result), &created)

    return &mcp.CallToolResult{
        Content: []mcp.Content{{Type: "text", Text: fmt.Sprintf("Event '%s' created: %s\n%s", created.Summary, created.HTMLURL, result)}},
    }, nil
}

// updateEvent updates an existing calendar event
// API: PATCH /calendars/{calendarId}/events/{eventId}
func (c *GoogleCalendarMCPClient) updateEvent(ctx context.Context, token string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    calendarID := getStringOrDefault(args, "calendar_id", "primary")
    eventID := args["event_id"].(string)
    sendNotifications := getBoolOrDefault(args, "send_notifications", true)

    // Build partial update payload
    event := map[string]interface{}{}
    if summary, ok := args["summary"]; ok {
        event["summary"] = summary
    }
    if desc, ok := args["description"]; ok {
        event["description"] = desc
    }
    if loc, ok := args["location"]; ok {
        event["location"] = loc
    }
    if startTime, ok := args["start_time"]; ok {
        event["start"] = map[string]string{"dateTime": startTime.(string)}
    }
    if endTime, ok := args["end_time"]; ok {
        event["end"] = map[string]string{"dateTime": endTime.(string)}
    }

    updateURL := fmt.Sprintf("%s/calendars/%s/events/%s?sendUpdates=%s",
        CalendarAPIURL, url.PathEscape(calendarID), url.PathEscape(eventID),
        map[bool]string{true: "all", false: "none"}[sendNotifications])

    result, err := c.makeRequestWithBody(ctx, token, "PATCH", updateURL, event)
    if err != nil {
        return nil, err
    }

    return &mcp.CallToolResult{
        Content: []mcp.Content{{Type: "text", Text: fmt.Sprintf("Event updated successfully\n%s", result)}},
    }, nil
}

// deleteEvent deletes a calendar event
// API: DELETE /calendars/{calendarId}/events/{eventId}
func (c *GoogleCalendarMCPClient) deleteEvent(ctx context.Context, token string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    calendarID := getStringOrDefault(args, "calendar_id", "primary")
    eventID := args["event_id"].(string)
    sendNotifications := getBoolOrDefault(args, "send_notifications", true)

    deleteURL := fmt.Sprintf("%s/calendars/%s/events/%s?sendUpdates=%s",
        CalendarAPIURL, url.PathEscape(calendarID), url.PathEscape(eventID),
        map[bool]string{true: "all", false: "none"}[sendNotifications])

    req, _ := http.NewRequestWithContext(ctx, "DELETE", deleteURL, nil)
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("failed to delete event: %d", resp.StatusCode)
    }

    return &mcp.CallToolResult{
        Content: []mcp.Content{{Type: "text", Text: fmt.Sprintf("Event '%s' deleted successfully", eventID)}},
    }, nil
}

// quickAdd creates an event from natural language
// API: POST /calendars/{calendarId}/events/quickAdd?text={text}
func (c *GoogleCalendarMCPClient) quickAdd(ctx context.Context, token string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    calendarID := getStringOrDefault(args, "calendar_id", "primary")
    text := args["text"].(string)
    sendNotifications := getBoolOrDefault(args, "send_notifications", true)

    quickAddURL := fmt.Sprintf("%s/calendars/%s/events/quickAdd?text=%s&sendUpdates=%s",
        CalendarAPIURL, url.PathEscape(calendarID), url.QueryEscape(text),
        map[bool]string{true: "all", false: "none"}[sendNotifications])

    req, _ := http.NewRequestWithContext(ctx, "POST", quickAddURL, nil)
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)

    var created struct {
        ID      string `json:"id"`
        HTMLURL string `json:"htmlLink"`
        Summary string `json:"summary"`
        Start   struct {
            DateTime string `json:"dateTime"`
        } `json:"start"`
    }
    json.Unmarshal(body, &created)

    return &mcp.CallToolResult{
        Content: []mcp.Content{{Type: "text", Text: fmt.Sprintf("Event '%s' created from '%s': %s\nStart: %s", created.Summary, text, created.HTMLURL, created.Start.DateTime)}},
    }, nil
}
```

### 3.2 MCP Service Integration

Update the MCP service to dynamically register connector clients based on user's active connections:

```go
// In response-api or mcp-tools service
func (s *MCPService) GetToolsForUser(ctx context.Context, userID uint) ([]mcp.Tool, error) {
    // Get base tools (search, browse, etc.)
    tools := s.getBaseTools()

    // Get user's active connector connections
    connections, err := s.connectorRepo.GetActiveConnections(ctx, userID)
    if err != nil {
        return tools, err
    }

    // Add connector-specific tools for each active connection
    for _, conn := range connections {
        switch conn.ConnectorType {
        case connector.ConnectorTypeGitHub:
            client := NewGitHubMCPClient(conn.AccessToken)
            tools = append(tools, client.GetTools()...)
        case connector.ConnectorTypeGmail:
            client := NewGmailMCPClient(conn.AccessToken)
            tools = append(tools, client.GetTools()...)
        // ... other connectors
        }
    }

    return tools, nil
}
```

---

## Phase 4: Frontend Implementation

### 4.1 Types

**Location: `apps/web/src/types/connector.d.ts`**

```typescript
type ConnectorType = 'github' | 'gmail' | 'google_drive' | 'google_calendar';

interface Connector {
  type: ConnectorType;
  name: string;
  description: string;
  icon: string;
  enabled: boolean;
  scopes: string[];
}

interface ConnectorConnection {
  id: string;
  connectorType: ConnectorType;
  isConnected: boolean;
  providerUsername?: string;
  providerEmail?: string;
  providerAvatar?: string;
  scopes: string[];
  lastSyncAt?: string;
  lastError?: string;
  connectedAt: string;
}

interface ConnectorStatus {
  type: ConnectorType;
  available: boolean;
  connected: boolean;
  connection?: ConnectorConnection;
}
```

### 4.2 Service

**Location: `apps/web/src/services/connector-service.ts`**

```typescript
import { fetchJsonWithAuth } from "@/lib/api-client";

declare const JAN_API_BASE_URL: string;

export const connectorService = {
  // List all connectors with connection status
  listConnectors: async (): Promise<ConnectorStatus[]> => {
    return fetchJsonWithAuth<ConnectorStatus[]>(
      `${JAN_API_BASE_URL}v1/connectors`
    );
  },

  // Get OAuth authorization URL
  getAuthURL: async (type: ConnectorType): Promise<{ auth_url: string; state: string }> => {
    return fetchJsonWithAuth<{ auth_url: string; state: string }>(
      `${JAN_API_BASE_URL}v1/connectors/${type}/auth-url`
    );
  },

  // Connect using OAuth code (after redirect)
  connect: async (type: ConnectorType, code: string, state: string): Promise<ConnectorConnection> => {
    return fetchJsonWithAuth<ConnectorConnection>(
      `${JAN_API_BASE_URL}v1/connectors/${type}/connect`,
      {
        method: "POST",
        body: JSON.stringify({ code, state }),
      }
    );
  },

  // Disconnect a connector
  disconnect: async (type: ConnectorType): Promise<void> => {
    await fetchJsonWithAuth(
      `${JAN_API_BASE_URL}v1/connectors/${type}/disconnect`,
      { method: "DELETE" }
    );
  },

  // Check connection status
  getStatus: async (type: ConnectorType): Promise<ConnectorStatus> => {
    return fetchJsonWithAuth<ConnectorStatus>(
      `${JAN_API_BASE_URL}v1/connectors/${type}/status`
    );
  },
};
```

### 4.3 Zustand Store

**Location: `apps/web/src/stores/connector-store.ts`**

```typescript
import { create } from "zustand";
import { persist } from "zustand/middleware";
import { connectorService } from "@/services/connector-service";

interface ConnectorState {
  connectors: ConnectorStatus[];
  isLoading: boolean;
  error: string | null;
  pendingOAuth: { type: ConnectorType; state: string } | null;

  // Actions
  fetchConnectors: () => Promise<void>;
  initiateConnect: (type: ConnectorType) => Promise<void>;
  completeConnect: (type: ConnectorType, code: string, state: string) => Promise<void>;
  disconnect: (type: ConnectorType) => Promise<void>;
  clearError: () => void;
}

export const useConnectors = create<ConnectorState>()(
  persist(
    (set, get) => ({
      connectors: [],
      isLoading: false,
      error: null,
      pendingOAuth: null,

      fetchConnectors: async () => {
        set({ isLoading: true, error: null });
        try {
          const connectors = await connectorService.listConnectors();
          set({ connectors, isLoading: false });
        } catch (error) {
          set({ error: (error as Error).message, isLoading: false });
        }
      },

      initiateConnect: async (type: ConnectorType) => {
        set({ isLoading: true, error: null });
        try {
          const { auth_url, state } = await connectorService.getAuthURL(type);
          set({ pendingOAuth: { type, state }, isLoading: false });
          // Redirect to OAuth provider
          window.location.href = auth_url;
        } catch (error) {
          set({ error: (error as Error).message, isLoading: false });
        }
      },

      completeConnect: async (type: ConnectorType, code: string, state: string) => {
        const { pendingOAuth } = get();

        // Verify state matches
        if (!pendingOAuth || pendingOAuth.state !== state) {
          set({ error: "Invalid OAuth state", pendingOAuth: null });
          return;
        }

        set({ isLoading: true, error: null });
        try {
          await connectorService.connect(type, code, state);
          await get().fetchConnectors(); // Refresh list
          set({ pendingOAuth: null, isLoading: false });
        } catch (error) {
          set({ error: (error as Error).message, isLoading: false, pendingOAuth: null });
        }
      },

      disconnect: async (type: ConnectorType) => {
        set({ isLoading: true, error: null });
        try {
          await connectorService.disconnect(type);
          await get().fetchConnectors(); // Refresh list
          set({ isLoading: false });
        } catch (error) {
          set({ error: (error as Error).message, isLoading: false });
        }
      },

      clearError: () => set({ error: null }),
    }),
    {
      name: "connector-storage",
      partialize: (state) => ({ pendingOAuth: state.pendingOAuth }), // Only persist OAuth state
    }
  )
);
```

### 4.4 UI Components

**Location: `apps/web/src/components/connectors/`**

```
connectors/
├── ConnectorList.tsx          # Grid of available connectors
├── ConnectorCard.tsx          # Individual connector card
├── ConnectorSettings.tsx      # Settings page for connectors
├── ConnectorOAuthCallback.tsx # OAuth callback handler page
└── icons/                     # Connector brand icons
    ├── GitHubIcon.tsx
    ├── GmailIcon.tsx
    ├── GoogleDriveIcon.tsx
    └── GoogleCalendarIcon.tsx
```

**ConnectorCard.tsx:**
```tsx
interface ConnectorCardProps {
  connector: ConnectorStatus;
  onConnect: () => void;
  onDisconnect: () => void;
}

export function ConnectorCard({ connector, onConnect, onDisconnect }: ConnectorCardProps) {
  const { type, available, connected, connection } = connector;

  return (
    <Card className="p-4">
      <div className="flex items-center gap-3">
        <ConnectorIcon type={type} className="h-10 w-10" />
        <div className="flex-1">
          <h3 className="font-medium">{getConnectorName(type)}</h3>
          <p className="text-sm text-muted-foreground">
            {getConnectorDescription(type)}
          </p>
        </div>
      </div>

      {connected && connection && (
        <div className="mt-3 flex items-center gap-2 text-sm text-muted-foreground">
          <Avatar className="h-5 w-5">
            <AvatarImage src={connection.providerAvatar} />
          </Avatar>
          <span>{connection.providerUsername || connection.providerEmail}</span>
        </div>
      )}

      <div className="mt-4">
        {!available ? (
          <Button disabled variant="outline" className="w-full">
            Coming Soon
          </Button>
        ) : connected ? (
          <Button variant="outline" onClick={onDisconnect} className="w-full">
            Disconnect
          </Button>
        ) : (
          <Button onClick={onConnect} className="w-full">
            Connect
          </Button>
        )}
      </div>
    </Card>
  );
}
```

### 4.5 OAuth Callback Route

**Location: `apps/web/src/routes/connectors/callback.tsx`**

```tsx
import { useEffect } from "react";
import { useNavigate, useSearchParams } from "@tanstack/react-router";
import { useConnectors } from "@/stores/connector-store";

export function ConnectorOAuthCallback() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { completeConnect, error } = useConnectors();

  useEffect(() => {
    const code = searchParams.get("code");
    const state = searchParams.get("state");
    const type = searchParams.get("type") as ConnectorType;

    if (code && state && type) {
      completeConnect(type, code, state).then(() => {
        navigate({ to: "/settings/connectors" });
      });
    }
  }, [searchParams]);

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen">
        <p className="text-destructive">Connection failed: {error}</p>
        <Button onClick={() => navigate({ to: "/settings/connectors" })}>
          Go Back
        </Button>
      </div>
    );
  }

  return (
    <div className="flex items-center justify-center min-h-screen">
      <Spinner />
      <span className="ml-2">Connecting...</span>
    </div>
  );
}
```

---

## Phase 5: Security Architecture (Comprehensive)

This section covers all security aspects of the Connectors feature, organized by security domain.

### 5.1 Threat Model

**Assets to Protect:**
- OAuth access tokens (high value - grants access to user's external accounts)
- OAuth refresh tokens (critical - long-lived credentials)
- User connection metadata (PII - email, username, avatar)
- Client secrets (critical - application credentials)

**Threat Actors:**
- External attackers (credential theft, CSRF, injection)
- Malicious insiders (unauthorized data access)
- Compromised dependencies (supply chain attacks)

**Attack Vectors:**
| Vector | Risk | Mitigation |
|--------|------|------------|
| Token theft from DB | Critical | Encryption at rest, key rotation |
| CSRF during OAuth | High | State parameter with HMAC, PKCE |
| Token leakage in logs | High | Structured logging, redaction |
| SSRF via connector | Medium | URL allowlisting, request validation |
| Privilege escalation | High | Ownership validation, RBAC |
| Replay attacks | Medium | Nonce validation, short expiry |

---

### 5.2 Token Storage Security

#### 5.2.1 Encryption at Rest (AES-256-GCM)

All OAuth tokens MUST be encrypted before database storage:

```go
package encryption

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "io"
)

type TokenEncryptor struct {
    key       []byte // 32 bytes for AES-256
    keyID     string // For key rotation tracking
}

func NewTokenEncryptor(keyHex string, keyID string) (*TokenEncryptor, error) {
    key, err := hex.DecodeString(keyHex)
    if err != nil || len(key) != 32 {
        return nil, errors.New("encryption key must be 32 bytes (64 hex chars)")
    }
    return &TokenEncryptor{key: key, keyID: keyID}, nil
}

// Encrypt returns: base64(keyID:nonce:ciphertext:tag)
func (e *TokenEncryptor) Encrypt(plaintext string) (string, error) {
    if plaintext == "" {
        return "", nil
    }

    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    // Generate random nonce
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }

    // Encrypt with authentication
    ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

    // Format: keyID:nonce:ciphertext (all base64)
    result := fmt.Sprintf("%s:%s:%s",
        e.keyID,
        base64.StdEncoding.EncodeToString(nonce),
        base64.StdEncoding.EncodeToString(ciphertext),
    )

    return base64.StdEncoding.EncodeToString([]byte(result)), nil
}

func (e *TokenEncryptor) Decrypt(encrypted string) (string, error) {
    if encrypted == "" {
        return "", nil
    }

    // Decode outer base64
    data, err := base64.StdEncoding.DecodeString(encrypted)
    if err != nil {
        return "", err
    }

    // Parse keyID:nonce:ciphertext
    parts := strings.SplitN(string(data), ":", 3)
    if len(parts) != 3 {
        return "", errors.New("invalid encrypted token format")
    }

    keyID := parts[0]
    if keyID != e.keyID {
        return "", errors.New("token encrypted with different key version")
    }

    nonce, err := base64.StdEncoding.DecodeString(parts[1])
    if err != nil {
        return "", err
    }

    ciphertext, err := base64.StdEncoding.DecodeString(parts[2])
    if err != nil {
        return "", err
    }

    // Decrypt
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", errors.New("decryption failed: authentication error")
    }

    return string(plaintext), nil
}
```

#### 5.2.2 Key Management

```go
// Multi-key support for rotation
type KeyManager struct {
    currentKey    *TokenEncryptor
    previousKeys  map[string]*TokenEncryptor // keyID -> encryptor
    mu            sync.RWMutex
}

func (km *KeyManager) Encrypt(plaintext string) (string, error) {
    km.mu.RLock()
    defer km.mu.RUnlock()
    return km.currentKey.Encrypt(plaintext)
}

func (km *KeyManager) Decrypt(encrypted string) (string, error) {
    km.mu.RLock()
    defer km.mu.RUnlock()

    // Try current key first
    plaintext, err := km.currentKey.Decrypt(encrypted)
    if err == nil {
        return plaintext, nil
    }

    // Try previous keys for rotation support
    for _, key := range km.previousKeys {
        plaintext, err = key.Decrypt(encrypted)
        if err == nil {
            return plaintext, nil
        }
    }

    return "", errors.New("unable to decrypt with any known key")
}
```

#### 5.2.3 Database Schema Security

```sql
-- Enhanced schema with security fields
CREATE TABLE connector_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connector_type connector_type NOT NULL,

    -- Encrypted tokens (NEVER store plaintext)
    access_token_encrypted TEXT NOT NULL,
    refresh_token_encrypted TEXT,
    encryption_key_id VARCHAR(32) NOT NULL, -- Track which key version

    -- Token metadata (not sensitive)
    token_type VARCHAR(32) DEFAULT 'Bearer',
    expires_at TIMESTAMP WITH TIME ZONE,
    scopes TEXT[],

    -- Provider info (consider if PII needs encryption)
    provider_user_id VARCHAR(255),
    provider_username VARCHAR(255),
    provider_email_hash VARCHAR(64), -- SHA-256 hash for lookup
    provider_avatar_url TEXT,

    -- Security tracking
    is_connected BOOLEAN NOT NULL DEFAULT true,
    last_used_at TIMESTAMP WITH TIME ZONE,
    last_refresh_at TIMESTAMP WITH TIME ZONE,
    refresh_failure_count INTEGER DEFAULT 0,
    last_error TEXT,

    -- Revocation support
    revoked_at TIMESTAMP WITH TIME ZONE,
    revocation_reason VARCHAR(255),

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    UNIQUE(user_id, connector_type)
);

-- Separate audit table for sensitive operations
CREATE TABLE connector_audit_log (
    id BIGSERIAL PRIMARY KEY,
    connection_id UUID REFERENCES connector_connections(id) ON DELETE SET NULL,
    user_id BIGINT NOT NULL,
    connector_type connector_type NOT NULL,
    action VARCHAR(50) NOT NULL, -- 'connected', 'disconnected', 'refreshed', 'revoked', 'used', 'error'
    ip_address INET,
    user_agent TEXT,
    metadata JSONB, -- Additional context (no sensitive data)
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_connector_audit_log_user ON connector_audit_log(user_id, created_at DESC);
CREATE INDEX idx_connector_audit_log_connection ON connector_audit_log(connection_id, created_at DESC);
```

---

### 5.3 OAuth Flow Security

#### 5.3.1 PKCE (Proof Key for Code Exchange)

PKCE is REQUIRED for all OAuth flows to prevent authorization code interception:

```go
package oauth

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
)

// GeneratePKCE creates a code verifier and challenge pair
func GeneratePKCE() (verifier, challenge string, err error) {
    // Generate 32 random bytes for verifier
    verifierBytes := make([]byte, 32)
    if _, err := rand.Read(verifierBytes); err != nil {
        return "", "", err
    }

    // URL-safe base64 encode (no padding)
    verifier = base64.RawURLEncoding.EncodeToString(verifierBytes)

    // SHA-256 hash for challenge
    hash := sha256.Sum256([]byte(verifier))
    challenge = base64.RawURLEncoding.EncodeToString(hash[:])

    return verifier, challenge, nil
}

// OAuth URL with PKCE
func (c *GoogleClient) GetAuthURL(connectorType ConnectorType, state, redirectURI, codeChallenge string) string {
    scopes := GoogleScopes[connectorType]
    params := url.Values{
        "client_id":             {c.clientID},
        "redirect_uri":          {redirectURI},
        "response_type":         {"code"},
        "scope":                 {strings.Join(scopes, " ")},
        "state":                 {state},
        "access_type":           {"offline"},
        "prompt":                {"consent"},
        "code_challenge":        {codeChallenge},
        "code_challenge_method": {"S256"},
    }
    return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

// Token exchange with PKCE verifier
func (c *GoogleClient) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*OAuthTokens, error) {
    params := url.Values{
        "client_id":     {c.clientID},
        "client_secret": {c.clientSecret},
        "code":          {code},
        "code_verifier": {codeVerifier},
        "grant_type":    {"authorization_code"},
        "redirect_uri":  {redirectURI},
    }
    // ... POST to token endpoint
}
```

#### 5.3.2 State Parameter with HMAC

State must be cryptographically bound to the user session:

```go
package oauth

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "time"
)

type StateManager struct {
    secret []byte
    repo   StateRepository
}

type OAuthStateData struct {
    ID            string
    UserID        uint
    ConnectorType ConnectorType
    State         string
    CodeVerifier  string // PKCE verifier
    RedirectURI   string
    Nonce         string // Additional entropy
    ExpiresAt     time.Time
    CreatedAt     time.Time
    IPAddress     string // Bind to client IP
    UserAgent     string // Bind to client
}

// GenerateState creates a cryptographically secure state
func (sm *StateManager) GenerateState(ctx context.Context, userID uint, connectorType ConnectorType, ip, userAgent string) (*OAuthStateData, error) {
    // Generate random nonce
    nonceBytes := make([]byte, 16)
    if _, err := rand.Read(nonceBytes); err != nil {
        return nil, err
    }
    nonce := hex.EncodeToString(nonceBytes)

    // Create state payload
    timestamp := time.Now().Unix()
    payload := fmt.Sprintf("%d:%s:%d:%s", userID, connectorType, timestamp, nonce)

    // HMAC signature
    mac := hmac.New(sha256.New, sm.secret)
    mac.Write([]byte(payload))
    signature := hex.EncodeToString(mac.Sum(nil))

    // State = base64(payload:signature)
    state := base64.RawURLEncoding.EncodeToString(
        []byte(fmt.Sprintf("%s:%s", payload, signature)),
    )

    // Generate PKCE
    verifier, challenge, err := GeneratePKCE()
    if err != nil {
        return nil, err
    }

    stateData := &OAuthStateData{
        ID:            uuid.New().String(),
        UserID:        userID,
        ConnectorType: connectorType,
        State:         state,
        CodeVerifier:  verifier,
        Nonce:         nonce,
        ExpiresAt:     time.Now().Add(5 * time.Minute), // Short expiry
        CreatedAt:     time.Now(),
        IPAddress:     ip,
        UserAgent:     userAgent,
    }

    // Store in database
    if err := sm.repo.Create(ctx, stateData); err != nil {
        return nil, err
    }

    return stateData, nil
}

// ValidateState verifies and consumes the state (one-time use)
func (sm *StateManager) ValidateState(ctx context.Context, state, ip, userAgent string) (*OAuthStateData, error) {
    // Decode state
    decoded, err := base64.RawURLEncoding.DecodeString(state)
    if err != nil {
        return nil, errors.New("invalid state encoding")
    }

    parts := strings.SplitN(string(decoded), ":", 5)
    if len(parts) != 5 {
        return nil, errors.New("invalid state format")
    }

    payload := strings.Join(parts[:4], ":")
    signature := parts[4]

    // Verify HMAC
    mac := hmac.New(sha256.New, sm.secret)
    mac.Write([]byte(payload))
    expectedSig := hex.EncodeToString(mac.Sum(nil))

    if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
        return nil, errors.New("invalid state signature")
    }

    // Lookup in database
    stateData, err := sm.repo.GetByState(ctx, state)
    if err != nil {
        return nil, errors.New("state not found or already used")
    }

    // Check expiration
    if time.Now().After(stateData.ExpiresAt) {
        sm.repo.Delete(ctx, stateData.ID) // Cleanup
        return nil, errors.New("state expired")
    }

    // Optionally verify IP/UserAgent binding (may be too strict)
    // if stateData.IPAddress != ip {
    //     return nil, errors.New("state IP mismatch")
    // }

    // Delete state (one-time use)
    if err := sm.repo.Delete(ctx, stateData.ID); err != nil {
        // Log but don't fail - state might be deleted by cleanup
    }

    return stateData, nil
}
```

#### 5.3.3 Redirect URI Validation

```go
// AllowedRedirectURIs whitelist per environment
var AllowedRedirectURIs = map[string][]string{
    "development": {
        "http://localhost:8000/api/v1/connectors/*/callback",
        "http://localhost:3001/connectors/callback",
    },
    "production": {
        "https://api.jan.ai/v1/connectors/*/callback",
        "https://jan.ai/connectors/callback",
    },
}

func ValidateRedirectURI(uri string, env string) error {
    allowed := AllowedRedirectURIs[env]
    for _, pattern := range allowed {
        if matchesPattern(uri, pattern) {
            return nil
        }
    }
    return errors.New("redirect URI not in allowlist")
}

// Prevent open redirect attacks
func (h *ConnectorHandler) HandleOAuthCallback(c *gin.Context) {
    redirectURI := c.Query("redirect_uri")

    // ALWAYS validate redirect URI
    if err := ValidateRedirectURI(redirectURI, h.env); err != nil {
        c.JSON(400, gin.H{"error": "invalid redirect_uri"})
        return
    }
    // ... continue with callback handling
}
```

---

### 5.4 Access Control & Authorization

#### 5.4.1 Connection Ownership Validation

```go
// CRITICAL: Always verify user owns the connection
func (s *ConnectorService) GetConnection(ctx context.Context, userID uint, connectorType ConnectorType) (*ConnectorConnection, error) {
    conn, err := s.repo.GetByUserAndType(ctx, userID, connectorType)
    if err != nil {
        return nil, err
    }

    // Defense in depth: verify ownership even though query filters by userID
    if conn.UserID != userID {
        s.auditLog.Log(ctx, AuditEvent{
            Action:  "unauthorized_access_attempt",
            UserID:  userID,
            Details: fmt.Sprintf("attempted to access connection %s", conn.ID),
        })
        return nil, ErrUnauthorized
    }

    return conn, nil
}

// Middleware for all connector routes
func RequireConnectionOwnership() gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := GetUserID(c)
        connectorType := c.Param("type")

        // Validate connector type
        if !IsValidConnectorType(connectorType) {
            c.AbortWithStatusJSON(400, gin.H{"error": "invalid connector type"})
            return
        }

        c.Set("userID", userID)
        c.Set("connectorType", connectorType)
        c.Next()
    }
}
```

#### 5.4.2 Role-Based Access Control

```go
// Admin-only operations
type ConnectorPermission string

const (
    PermissionConnectorConnect    ConnectorPermission = "connector:connect"
    PermissionConnectorDisconnect ConnectorPermission = "connector:disconnect"
    PermissionConnectorView       ConnectorPermission = "connector:view"
    PermissionConnectorAdmin      ConnectorPermission = "connector:admin" // View all users' connections
)

func RequirePermission(perm ConnectorPermission) gin.HandlerFunc {
    return func(c *gin.Context) {
        user := GetUser(c)
        if !user.HasPermission(string(perm)) {
            c.AbortWithStatusJSON(403, gin.H{"error": "insufficient permissions"})
            return
        }
        c.Next()
    }
}
```

---

### 5.5 Rate Limiting

#### 5.5.1 Multi-Layer Rate Limiting

```go
// Rate limit configuration
type RateLimitConfig struct {
    // OAuth flow limits (prevent brute force)
    OAuthInitiate   RateLimit // e.g., 10 per hour per user
    OAuthCallback   RateLimit // e.g., 20 per hour per user

    // API limits
    ConnectorAPI    RateLimit // e.g., 100 per minute per user

    // Per-connector limits (respect provider rate limits)
    GitHubAPI       RateLimit // GitHub: 5000/hour authenticated
    GoogleAPI       RateLimit // Google: varies by API
}

type RateLimit struct {
    Requests int
    Window   time.Duration
}

// Redis-based rate limiter
type RateLimiter struct {
    redis  *redis.Client
    config RateLimitConfig
}

func (rl *RateLimiter) CheckOAuthInitiate(ctx context.Context, userID uint) error {
    key := fmt.Sprintf("ratelimit:oauth:initiate:%d", userID)
    return rl.check(ctx, key, rl.config.OAuthInitiate)
}

func (rl *RateLimiter) CheckConnectorAPI(ctx context.Context, userID uint, connectorType string) error {
    // User-level limit
    userKey := fmt.Sprintf("ratelimit:connector:%d", userID)
    if err := rl.check(ctx, userKey, rl.config.ConnectorAPI); err != nil {
        return err
    }

    // Connector-specific limit
    connectorKey := fmt.Sprintf("ratelimit:connector:%d:%s", userID, connectorType)
    var limit RateLimit
    switch connectorType {
    case "github":
        limit = rl.config.GitHubAPI
    case "gmail", "google_drive", "google_calendar":
        limit = rl.config.GoogleAPI
    }
    return rl.check(ctx, connectorKey, limit)
}

func (rl *RateLimiter) check(ctx context.Context, key string, limit RateLimit) error {
    count, err := rl.redis.Incr(ctx, key).Result()
    if err != nil {
        return err
    }

    if count == 1 {
        rl.redis.Expire(ctx, key, limit.Window)
    }

    if count > int64(limit.Requests) {
        return ErrRateLimitExceeded
    }

    return nil
}
```

#### 5.5.2 Exponential Backoff for Failures

```go
func (s *ConnectorService) GetValidToken(ctx context.Context, userID uint, connectorType ConnectorType) (string, error) {
    conn, err := s.repo.GetConnection(ctx, userID, connectorType)
    if err != nil {
        return "", err
    }

    // Check if in backoff period due to repeated failures
    if conn.RefreshFailureCount > 0 {
        backoffDuration := time.Duration(math.Pow(2, float64(conn.RefreshFailureCount))) * time.Minute
        if conn.RefreshFailureCount > 5 {
            backoffDuration = time.Hour // Cap at 1 hour
        }
        if time.Since(*conn.LastRefreshAt) < backoffDuration {
            return "", ErrTokenRefreshBackoff
        }
    }

    // ... proceed with token refresh
}
```

---

### 5.6 Audit Logging

#### 5.6.1 Comprehensive Audit Events

```go
type AuditAction string

const (
    AuditConnectorConnected     AuditAction = "connector.connected"
    AuditConnectorDisconnected  AuditAction = "connector.disconnected"
    AuditConnectorTokenRefreshed AuditAction = "connector.token_refreshed"
    AuditConnectorTokenUsed     AuditAction = "connector.token_used"
    AuditConnectorRevoked       AuditAction = "connector.revoked"
    AuditConnectorError         AuditAction = "connector.error"
    AuditOAuthInitiated         AuditAction = "oauth.initiated"
    AuditOAuthCompleted         AuditAction = "oauth.completed"
    AuditOAuthFailed            AuditAction = "oauth.failed"
    AuditUnauthorizedAccess     AuditAction = "security.unauthorized_access"
)

type AuditEvent struct {
    ID            string
    Timestamp     time.Time
    UserID        uint
    ConnectionID  *string
    ConnectorType ConnectorType
    Action        AuditAction
    IPAddress     string
    UserAgent     string
    RequestID     string
    Success       bool
    ErrorMessage  string
    Metadata      map[string]interface{}
}

type AuditLogger struct {
    repo   AuditRepository
    logger zerolog.Logger
}

func (a *AuditLogger) Log(ctx context.Context, event AuditEvent) {
    event.ID = uuid.New().String()
    event.Timestamp = time.Now()
    event.RequestID = GetRequestID(ctx)

    // Store in database
    if err := a.repo.Create(ctx, &event); err != nil {
        a.logger.Error().Err(err).Msg("failed to store audit event")
    }

    // Also log for real-time monitoring
    a.logger.Info().
        Str("audit_id", event.ID).
        Str("action", string(event.Action)).
        Uint("user_id", event.UserID).
        Str("connector_type", string(event.ConnectorType)).
        Bool("success", event.Success).
        Msg("connector audit event")
}
```

#### 5.6.2 Sensitive Data Redaction

```go
// NEVER log tokens or secrets
func (a *AuditLogger) LogTokenUsage(ctx context.Context, userID uint, connectorType ConnectorType, toolName string) {
    a.Log(ctx, AuditEvent{
        UserID:        userID,
        ConnectorType: connectorType,
        Action:        AuditConnectorTokenUsed,
        Metadata: map[string]interface{}{
            "tool_name": toolName,
            // DO NOT include: access_token, refresh_token, API responses with sensitive data
        },
    })
}

// Redact tokens from error messages
func RedactTokensFromError(err error) string {
    msg := err.Error()
    // Redact anything that looks like a token
    tokenPatterns := []string{
        `Bearer [A-Za-z0-9\-_\.]+`,
        `gho_[A-Za-z0-9]+`,        // GitHub tokens
        `ya29\.[A-Za-z0-9\-_]+`,   // Google access tokens
    }
    for _, pattern := range tokenPatterns {
        re := regexp.MustCompile(pattern)
        msg = re.ReplaceAllString(msg, "[REDACTED]")
    }
    return msg
}
```

---

### 5.7 Token Lifecycle Management

#### 5.7.1 Secure Token Refresh

```go
func (s *ConnectorService) RefreshTokens(ctx context.Context, conn *ConnectorConnection) (*OAuthTokens, error) {
    // Decrypt refresh token
    refreshToken, err := s.encryptor.Decrypt(conn.RefreshTokenEncrypted)
    if err != nil {
        s.auditLog.Log(ctx, AuditEvent{
            UserID:        conn.UserID,
            ConnectionID:  &conn.ID,
            ConnectorType: conn.ConnectorType,
            Action:        AuditConnectorError,
            Success:       false,
            ErrorMessage:  "failed to decrypt refresh token",
        })
        return nil, ErrTokenDecryptionFailed
    }

    // Call provider's token endpoint
    var newTokens *OAuthTokens
    switch conn.ConnectorType {
    case ConnectorTypeGitHub:
        newTokens, err = s.githubClient.RefreshToken(ctx, refreshToken)
    case ConnectorTypeGmail, ConnectorTypeGoogleDrive, ConnectorTypeGoogleCalendar:
        newTokens, err = s.googleClient.RefreshToken(ctx, refreshToken)
    }

    if err != nil {
        // Track failure for backoff
        s.repo.IncrementRefreshFailure(ctx, conn.ID)
        s.auditLog.Log(ctx, AuditEvent{
            UserID:        conn.UserID,
            ConnectionID:  &conn.ID,
            ConnectorType: conn.ConnectorType,
            Action:        AuditConnectorError,
            Success:       false,
            ErrorMessage:  RedactTokensFromError(err),
        })
        return nil, err
    }

    // Encrypt new tokens
    accessEncrypted, err := s.encryptor.Encrypt(newTokens.AccessToken)
    if err != nil {
        return nil, err
    }

    var refreshEncrypted string
    if newTokens.RefreshToken != "" {
        refreshEncrypted, err = s.encryptor.Encrypt(newTokens.RefreshToken)
        if err != nil {
            return nil, err
        }
    } else {
        // Keep existing refresh token if not rotated
        refreshEncrypted = conn.RefreshTokenEncrypted
    }

    // Update database
    err = s.repo.UpdateTokens(ctx, conn.ID, UpdateTokensParams{
        AccessTokenEncrypted:  accessEncrypted,
        RefreshTokenEncrypted: refreshEncrypted,
        ExpiresAt:             newTokens.ExpiresAt,
        RefreshFailureCount:   0, // Reset on success
        LastRefreshAt:         time.Now(),
    })
    if err != nil {
        return nil, err
    }

    s.auditLog.Log(ctx, AuditEvent{
        UserID:        conn.UserID,
        ConnectionID:  &conn.ID,
        ConnectorType: conn.ConnectorType,
        Action:        AuditConnectorTokenRefreshed,
        Success:       true,
    })

    return newTokens, nil
}
```

#### 5.7.2 Token Revocation

```go
func (s *ConnectorService) Disconnect(ctx context.Context, userID uint, connectorType ConnectorType, reason string) error {
    conn, err := s.GetConnection(ctx, userID, connectorType)
    if err != nil {
        return err
    }

    // Revoke tokens at the provider
    accessToken, _ := s.encryptor.Decrypt(conn.AccessTokenEncrypted)
    switch connectorType {
    case ConnectorTypeGitHub:
        // GitHub doesn't have a revocation endpoint for OAuth tokens
        // Token is deleted from our DB which prevents further use
    case ConnectorTypeGmail, ConnectorTypeGoogleDrive, ConnectorTypeGoogleCalendar:
        // Revoke at Google
        if err := s.googleClient.RevokeToken(ctx, accessToken); err != nil {
            // Log but continue - we still want to remove from our DB
            s.logger.Warn().Err(err).Msg("failed to revoke token at Google")
        }
    }

    // Mark as revoked (soft delete for audit trail)
    err = s.repo.Revoke(ctx, conn.ID, RevokeParams{
        RevokedAt:        time.Now(),
        RevocationReason: reason,
        // Clear encrypted tokens
        AccessTokenEncrypted:  "",
        RefreshTokenEncrypted: "",
    })
    if err != nil {
        return err
    }

    s.auditLog.Log(ctx, AuditEvent{
        UserID:        userID,
        ConnectionID:  &conn.ID,
        ConnectorType: connectorType,
        Action:        AuditConnectorDisconnected,
        Success:       true,
        Metadata: map[string]interface{}{
            "reason": reason,
        },
    })

    return nil
}
```

#### 5.7.3 Automated Cleanup Jobs

```go
// Scheduled job to clean up expired states and connections
func (s *ConnectorService) CleanupExpiredData(ctx context.Context) error {
    // Delete expired OAuth states (older than 10 minutes)
    deleted, err := s.stateRepo.DeleteExpired(ctx, time.Now().Add(-10*time.Minute))
    if err != nil {
        return err
    }
    s.logger.Info().Int("deleted", deleted).Msg("cleaned up expired OAuth states")

    // Disable connections with too many refresh failures
    disabled, err := s.repo.DisableFailedConnections(ctx, 10) // 10+ failures
    if err != nil {
        return err
    }
    s.logger.Info().Int("disabled", disabled).Msg("disabled connections with repeated failures")

    // Archive old audit logs (older than 90 days) - move to cold storage
    archived, err := s.auditRepo.ArchiveOldEvents(ctx, time.Now().AddDate(0, 0, -90))
    if err != nil {
        return err
    }
    s.logger.Info().Int("archived", archived).Msg("archived old audit events")

    return nil
}
```

---

### 5.8 Data Protection & Privacy

#### 5.8.1 Scope Configuration

**GitHub Scopes (Full Access - Claude Code Style):**

GitHub connector requires write access for code operations (create branches, commits, PRs):

| Scope | Access Level | Capabilities |
|-------|--------------|--------------|
| `repo` | **Full** | Read/write code, issues, PRs, branches, commits |
| `read:user` | Read | User profile information |
| `user:email` | Read | User email addresses |
| `workflow` | Write | Update GitHub Actions workflows |

**Google Scopes:**

| Connector | Required Scopes | Access Level | Justification |
|-----------|-----------------|--------------|---------------|
| Gmail | `gmail.readonly` | Read-only | Email is sensitive - read only |
| Google Drive | `drive.readonly` | Read-only | Documents are sensitive - read only |
| Google Calendar | `calendar`, `calendar.events` | **Read + Write** | Manage events, schedule meetings |

```go
// Scope definitions by connector
var ConnectorScopes = map[ConnectorType][]string{
    // GitHub: Full access for code operations (like Claude Code)
    ConnectorTypeGitHub: {
        "repo",          // Full repository access (read/write)
        "read:user",     // Read user profile
        "user:email",    // Read user email
        "workflow",      // GitHub Actions (optional)
    },
    // Gmail: Read-only for security
    ConnectorTypeGmail: {
        "https://www.googleapis.com/auth/gmail.readonly",
        "https://www.googleapis.com/auth/userinfo.email",
        "https://www.googleapis.com/auth/userinfo.profile",
    },
    // Google Drive: Read-only for security
    ConnectorTypeGoogleDrive: {
        "https://www.googleapis.com/auth/drive.readonly",
        "https://www.googleapis.com/auth/userinfo.email",
        "https://www.googleapis.com/auth/userinfo.profile",
    },
    // Google Calendar: Read + Write for event management
    ConnectorTypeGoogleCalendar: {
        "https://www.googleapis.com/auth/calendar",        // Full calendar access
        "https://www.googleapis.com/auth/calendar.events", // Read/write events
        "https://www.googleapis.com/auth/userinfo.email",
        "https://www.googleapis.com/auth/userinfo.profile",
    },
}
```

#### 5.8.2 Write Operation Security (GitHub)

Since GitHub connector has write access, additional security measures are required:

```go
// Write operation audit logging
func (c *GitHubMCPClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
    // All write operations must be logged
    isWriteOp := isWriteOperation(toolName)

    if isWriteOp {
        c.auditLog.Log(ctx, AuditEvent{
            Action:        AuditGitHubWrite,
            ConnectorType: ConnectorTypeGitHub,
            Metadata: map[string]interface{}{
                "tool":       toolName,
                "owner":      args["owner"],
                "repo":       args["repo"],
                "branch":     args["branch"],
            },
        })
    }

    // Execute the operation
    result, err := c.executeToolCall(ctx, toolName, args)

    if isWriteOp {
        c.auditLog.Log(ctx, AuditEvent{
            Action:  AuditGitHubWriteComplete,
            Success: err == nil,
            Metadata: map[string]interface{}{
                "tool":  toolName,
                "error": err,
            },
        })
    }

    return result, err
}

func isWriteOperation(toolName string) bool {
    writeOps := map[string]bool{
        "github_create_branch":         true,
        "github_create_or_update_file": true,
        "github_delete_file":           true,
        "github_create_pull_request":   true,
        "github_merge_pull_request":    true,
        "github_add_pr_review":         true,
        "github_create_issue":          true,
        "github_add_comment":           true,
        "github_update_issue":          true,
    }
    return writeOps[toolName]
}

// Rate limiting for write operations (stricter than read)
var WriteOperationRateLimits = RateLimitConfig{
    // Per user per hour
    CreateBranch:      20,
    CreateFile:        50,
    CreatePR:          10,
    MergePR:           10,
    CreateIssue:       20,
    AddComment:        50,
}
```

#### 5.8.3 User Consent for Write Operations

Users should explicitly consent to write operations when connecting:

```typescript
// Frontend: Show write permissions clearly during OAuth
const GitHubPermissionsDisplay = () => (
  <div className="permissions-list">
    <h4>GitHub will grant Jan access to:</h4>
    <ul>
      <li className="read">✓ Read your repositories, issues, and pull requests</li>
      <li className="read">✓ Read your profile and email</li>
      <li className="write">✓ Create and update files in repositories</li>
      <li className="write">✓ Create branches and pull requests</li>
      <li className="write">✓ Add comments and reviews</li>
      <li className="write">✓ Create and update issues</li>
    </ul>
    <p className="warning">
      Write operations will be logged and can be revoked at any time.
    </p>
  </div>
);
```

#### 5.8.2 Data Retention Policy

```go
// Retention policy configuration
type RetentionPolicy struct {
    // How long to keep connection after disconnection (for potential re-connection)
    DisconnectedConnectionRetention time.Duration // e.g., 30 days

    // How long to keep audit logs
    AuditLogRetention time.Duration // e.g., 90 days for compliance

    // How long to keep cached data from connectors
    CachedDataRetention time.Duration // e.g., 24 hours
}

// GDPR-compliant data export
func (s *ConnectorService) ExportUserData(ctx context.Context, userID uint) (*UserDataExport, error) {
    connections, err := s.repo.GetAllByUser(ctx, userID)
    if err != nil {
        return nil, err
    }

    auditLogs, err := s.auditRepo.GetByUser(ctx, userID)
    if err != nil {
        return nil, err
    }

    return &UserDataExport{
        Connections: sanitizeConnectionsForExport(connections),
        AuditLogs:   auditLogs,
        ExportedAt:  time.Now(),
    }, nil
}

// GDPR-compliant data deletion
func (s *ConnectorService) DeleteAllUserData(ctx context.Context, userID uint) error {
    // Disconnect all connectors first (revoke tokens)
    connections, _ := s.repo.GetAllByUser(ctx, userID)
    for _, conn := range connections {
        s.Disconnect(ctx, userID, conn.ConnectorType, "user_data_deletion")
    }

    // Hard delete connections
    if err := s.repo.HardDeleteByUser(ctx, userID); err != nil {
        return err
    }

    // Delete audit logs (or anonymize if required for compliance)
    if err := s.auditRepo.DeleteByUser(ctx, userID); err != nil {
        return err
    }

    return nil
}
```

---

### 5.9 Network Security

#### 5.9.1 SSRF Prevention

```go
// Allowlist for external API calls
var AllowedHosts = map[ConnectorType][]string{
    ConnectorTypeGitHub: {
        "api.github.com",
        "github.com",
        "raw.githubusercontent.com",
    },
    ConnectorTypeGmail: {
        "gmail.googleapis.com",
        "www.googleapis.com",
        "oauth2.googleapis.com",
    },
    ConnectorTypeGoogleDrive: {
        "www.googleapis.com",
        "oauth2.googleapis.com",
    },
    ConnectorTypeGoogleCalendar: {
        "www.googleapis.com",
        "oauth2.googleapis.com",
    },
}

// Secure HTTP client that prevents SSRF
func NewSecureHTTPClient(connectorType ConnectorType) *http.Client {
    return &http.Client{
        Timeout: 30 * time.Second,
        Transport: &ssrfSafeTransport{
            allowedHosts: AllowedHosts[connectorType],
            base:         http.DefaultTransport,
        },
    }
}

type ssrfSafeTransport struct {
    allowedHosts []string
    base         http.RoundTripper
}

func (t *ssrfSafeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    host := req.URL.Hostname()

    allowed := false
    for _, h := range t.allowedHosts {
        if host == h {
            allowed = true
            break
        }
    }

    if !allowed {
        return nil, fmt.Errorf("host %s not in allowlist for connector", host)
    }

    // Prevent internal IP access
    ips, err := net.LookupIP(host)
    if err == nil {
        for _, ip := range ips {
            if isPrivateIP(ip) {
                return nil, fmt.Errorf("private IP addresses not allowed")
            }
        }
    }

    return t.base.RoundTrip(req)
}

func isPrivateIP(ip net.IP) bool {
    privateRanges := []string{
        "10.0.0.0/8",
        "172.16.0.0/12",
        "192.168.0.0/16",
        "127.0.0.0/8",
        "169.254.0.0/16",
        "::1/128",
        "fc00::/7",
        "fe80::/10",
    }
    for _, cidr := range privateRanges {
        _, network, _ := net.ParseCIDR(cidr)
        if network.Contains(ip) {
            return true
        }
    }
    return false
}
```

#### 5.9.2 TLS Configuration

```go
// Enforce TLS 1.2+ for all external connections
func NewTLSConfig() *tls.Config {
    return &tls.Config{
        MinVersion: tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        },
    }
}
```

---

### 5.10 Frontend Security

#### 5.10.1 Secure State Storage

```typescript
// Use sessionStorage for OAuth state (not localStorage)
// sessionStorage is cleared when tab closes, reducing attack window

const OAUTH_STATE_KEY = 'connector_oauth_state';

export const secureStateStorage = {
  setOAuthState: (state: { type: ConnectorType; state: string; timestamp: number }) => {
    // Include timestamp for client-side expiry check
    sessionStorage.setItem(OAUTH_STATE_KEY, JSON.stringify({
      ...state,
      timestamp: Date.now(),
    }));
  },

  getOAuthState: (): { type: ConnectorType; state: string; timestamp: number } | null => {
    const data = sessionStorage.getItem(OAUTH_STATE_KEY);
    if (!data) return null;

    const parsed = JSON.parse(data);

    // Client-side expiry check (5 minutes)
    if (Date.now() - parsed.timestamp > 5 * 60 * 1000) {
      sessionStorage.removeItem(OAUTH_STATE_KEY);
      return null;
    }

    return parsed;
  },

  clearOAuthState: () => {
    sessionStorage.removeItem(OAUTH_STATE_KEY);
  },
};
```

#### 5.10.2 XSS Prevention

```typescript
// Never render connector data without sanitization
import DOMPurify from 'dompurify';

function ConnectorCard({ connector }: { connector: ConnectorStatus }) {
  // Sanitize any user-provided data from the connector
  const sanitizedUsername = DOMPurify.sanitize(connector.connection?.providerUsername || '');

  return (
    <div>
      {/* Safe: React escapes by default */}
      <span>{connector.connection?.providerEmail}</span>

      {/* If using dangerouslySetInnerHTML (avoid if possible) */}
      <div dangerouslySetInnerHTML={{ __html: sanitizedUsername }} />
    </div>
  );
}
```

#### 5.10.3 OAuth Callback Security

```typescript
// Secure OAuth callback handler
export function ConnectorOAuthCallback() {
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const handleCallback = async () => {
      const params = new URLSearchParams(window.location.search);
      const code = params.get('code');
      const state = params.get('state');
      const errorParam = params.get('error');

      // Handle OAuth errors
      if (errorParam) {
        setError(`OAuth error: ${errorParam}`);
        secureStateStorage.clearOAuthState();
        return;
      }

      if (!code || !state) {
        setError('Missing OAuth parameters');
        return;
      }

      // Verify state matches what we stored
      const storedState = secureStateStorage.getOAuthState();
      if (!storedState || storedState.state !== state) {
        setError('Invalid OAuth state - possible CSRF attack');
        secureStateStorage.clearOAuthState();
        return;
      }

      // Clear state immediately (one-time use)
      secureStateStorage.clearOAuthState();

      // Clear URL parameters (don't leave code in browser history)
      window.history.replaceState({}, '', window.location.pathname);

      try {
        await connectorService.connect(storedState.type, code, state);
        navigate('/settings/connectors');
      } catch (err) {
        setError('Failed to complete connection');
      }
    };

    handleCallback();
  }, []);

  // ... render
}
```

---

### 5.11 Incident Response

#### 5.11.1 Token Compromise Response

```go
// Emergency token revocation for compromised accounts
func (s *ConnectorService) EmergencyRevokeAllUserTokens(ctx context.Context, userID uint, reason string) error {
    connections, err := s.repo.GetAllByUser(ctx, userID)
    if err != nil {
        return err
    }

    var errs []error
    for _, conn := range connections {
        if err := s.Disconnect(ctx, userID, conn.ConnectorType, reason); err != nil {
            errs = append(errs, err)
        }
    }

    // Alert security team
    s.alerting.SendSecurityAlert(ctx, SecurityAlert{
        Type:     AlertTokenCompromise,
        UserID:   userID,
        Message:  fmt.Sprintf("Emergency revocation of %d connectors for user %d: %s", len(connections), userID, reason),
        Severity: SeverityHigh,
    })

    if len(errs) > 0 {
        return fmt.Errorf("failed to revoke some tokens: %v", errs)
    }
    return nil
}

// Bulk revocation for security incidents
func (s *ConnectorService) RevokeAllConnectionsByConnectorType(ctx context.Context, connectorType ConnectorType, reason string) error {
    // Use with extreme caution - affects all users
    connections, err := s.repo.GetAllByType(ctx, connectorType)
    if err != nil {
        return err
    }

    s.logger.Warn().
        Str("connector_type", string(connectorType)).
        Int("affected_users", len(connections)).
        Str("reason", reason).
        Msg("SECURITY: Bulk revocation initiated")

    for _, conn := range connections {
        s.Disconnect(ctx, conn.UserID, connectorType, reason)
    }

    return nil
}
```

#### 5.11.2 Security Monitoring

```go
// Anomaly detection for connector usage
type SecurityMonitor struct {
    auditRepo AuditRepository
    alerting  AlertingService
}

func (sm *SecurityMonitor) CheckForAnomalies(ctx context.Context) {
    // Check for unusual patterns

    // 1. Multiple failed OAuth attempts
    failedAttempts, _ := sm.auditRepo.CountByActionSince(ctx,
        AuditOAuthFailed,
        time.Now().Add(-time.Hour),
    )
    if failedAttempts > 100 {
        sm.alerting.SendSecurityAlert(ctx, SecurityAlert{
            Type:     AlertBruteForce,
            Message:  fmt.Sprintf("%d failed OAuth attempts in last hour", failedAttempts),
            Severity: SeverityMedium,
        })
    }

    // 2. Unusual token refresh patterns (potential token theft)
    // ... check for tokens being used from multiple IPs simultaneously

    // 3. Mass disconnections
    disconnections, _ := sm.auditRepo.CountByActionSince(ctx,
        AuditConnectorDisconnected,
        time.Now().Add(-time.Hour),
    )
    if disconnections > 50 {
        sm.alerting.SendSecurityAlert(ctx, SecurityAlert{
            Type:     AlertMassDisconnection,
            Message:  fmt.Sprintf("%d disconnections in last hour", disconnections),
            Severity: SeverityMedium,
        })
    }
}
```

---

### 5.12 Security Checklist

#### Pre-Launch Security Review

- [ ] **Token Storage**
  - [ ] All tokens encrypted with AES-256-GCM
  - [ ] Encryption keys stored securely (not in code)
  - [ ] Key rotation mechanism in place
  - [ ] Database column-level encryption for extra security

- [ ] **OAuth Flow**
  - [ ] PKCE implemented for all flows
  - [ ] State parameter uses HMAC and server-side validation
  - [ ] State expires in 5 minutes
  - [ ] Redirect URIs strictly validated against allowlist
  - [ ] Code exchange happens server-side only

- [ ] **Access Control**
  - [ ] All endpoints require authentication
  - [ ] Connection ownership verified on every access
  - [ ] Admin operations require elevated permissions
  - [ ] Rate limiting on OAuth and API endpoints

- [ ] **Data Protection**
  - [ ] Minimal scopes requested
  - [ ] PII handling compliant with GDPR
  - [ ] Data retention policies implemented
  - [ ] User data export/deletion available

- [ ] **Logging & Monitoring**
  - [ ] All sensitive operations logged
  - [ ] No tokens or secrets in logs
  - [ ] Anomaly detection alerts configured
  - [ ] Incident response procedures documented

- [ ] **Network Security**
  - [ ] SSRF protection via host allowlisting
  - [ ] TLS 1.2+ enforced
  - [ ] Private IP access blocked

- [ ] **Frontend Security**
  - [ ] OAuth state in sessionStorage (not localStorage)
  - [ ] XSS protection on connector data
  - [ ] CSRF tokens on state-changing requests
  - [ ] URL parameters cleared after OAuth callback

---

## Phase 6: Environment Configuration

### 6.1 Required Environment Variables

Add to `.env.template`:

```bash
# =============================================================================
# CONNECTORS CONFIGURATION
# =============================================================================

# GitHub Connector
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GITHUB_CONNECTOR_ENABLED=false

# Google Connectors (Gmail, Drive, Calendar share the same credentials)
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GOOGLE_CONNECTOR_ENABLED=false

# OAuth Configuration
OAUTH_REDIRECT_BASE_URL=http://localhost:8000

# Token Encryption (generate with: openssl rand -hex 32)
CONNECTOR_TOKEN_ENCRYPTION_KEY=your_32_byte_hex_key
```

### 6.2 Google Cloud Console Setup

1. Create a new project in Google Cloud Console
2. Enable APIs: Gmail API, Google Drive API, Google Calendar API
3. Configure OAuth consent screen
4. Create OAuth 2.0 credentials (Web application)
5. Add authorized redirect URIs:
   - `http://localhost:8000/api/v1/connectors/gmail/callback`
   - `http://localhost:8000/api/v1/connectors/google_drive/callback`
   - `http://localhost:8000/api/v1/connectors/google_calendar/callback`

### 6.3 GitHub OAuth App Setup

1. Go to GitHub Settings > Developer settings > OAuth Apps
2. Create new OAuth App
3. Set callback URL: `http://localhost:8000/api/v1/connectors/github/callback`

---

## Phase 7: Implementation Order

### Sprint 1: Foundation (Backend Core)
1. [ ] Create database migration for connector tables
2. [ ] Implement domain entities and repository interface
3. [ ] Implement token encryption service
4. [ ] Create OAuth state management
5. [ ] Add environment configuration

### Sprint 2: GitHub Connector (Full Vertical Slice)
1. [ ] Implement GitHub OAuth client
2. [ ] Create HTTP routes for GitHub connector
3. [ ] Implement GitHub MCP tools (search repos, issues, files)
4. [ ] Frontend: connector service and store
5. [ ] Frontend: ConnectorCard and settings UI
6. [ ] Frontend: OAuth callback handler
7. [ ] Integration testing

### Sprint 3: Google Connectors
1. [ ] Implement Google OAuth client (shared for all Google services)
2. [ ] Gmail MCP tools (search, read emails)
3. [ ] Google Drive MCP tools (search, read files)
4. [ ] Google Calendar MCP tools (list, search events)
5. [ ] Frontend UI updates for Google connectors
6. [ ] Integration testing

### Sprint 4: Polish & Production
1. [ ] Token refresh automation
2. [ ] Connection health monitoring
3. [ ] Error handling and user notifications
4. [ ] Rate limiting per connector
5. [ ] Logging and observability
6. [ ] Documentation
7. [ ] Production deployment configuration

---

## File Structure Summary

### Backend (Go)

```
services/llm-api/
├── internal/
│   ├── domain/connector/
│   │   ├── entity.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── oauth_service.go
│   │   └── dto.go
│   ├── infrastructure/
│   │   ├── connector/
│   │   │   ├── config.go
│   │   │   ├── github_client.go
│   │   │   ├── google_client.go
│   │   │   ├── token_encryptor.go
│   │   │   └── oauth_provider.go
│   │   └── database/
│   │       ├── dbschema/
│   │       │   └── connector_connection.go
│   │       └── repository/
│   │           └── connectorrepo/
│   │               └── connector_repository.go
│   └── interfaces/httpserver/
│       └── routes/v1/connectors/
│           ├── connectors_route.go
│           └── handlers.go
├── migrations/
│   ├── 000027_create_connectors.up.sql
│   └── 000027_create_connectors.down.sql
└── docs/swagger/
    └── swagger.yaml (updated)

services/mcp-tools/
├── internal/
│   └── domain/connectors/
│       ├── github_mcp.go
│       ├── gmail_mcp.go
│       ├── drive_mcp.go
│       └── calendar_mcp.go
```

### Frontend (React)

```
apps/web/src/
├── types/
│   └── connector.d.ts
├── services/
│   └── connector-service.ts
├── stores/
│   └── connector-store.ts
├── components/connectors/
│   ├── ConnectorList.tsx
│   ├── ConnectorCard.tsx
│   ├── ConnectorSettings.tsx
│   └── icons/
│       ├── GitHubIcon.tsx
│       ├── GmailIcon.tsx
│       ├── GoogleDriveIcon.tsx
│       └── GoogleCalendarIcon.tsx
└── routes/
    ├── settings/
    │   └── connectors.tsx
    └── connectors/
        └── callback.tsx
```

---

## Testing Strategy

### Unit Tests
- Token encryption/decryption
- OAuth state validation
- Domain service logic

### Integration Tests
- OAuth flow (mock OAuth providers)
- MCP tool execution
- Token refresh flow

### E2E Tests
- Full connector connection flow
- Using connector tools in chat

---

## API Response Examples

### GET /v1/connectors
```json
{
  "data": [
    {
      "type": "github",
      "name": "GitHub",
      "description": "Access your repositories, issues, and pull requests",
      "available": true,
      "connected": true,
      "connection": {
        "id": "conn_abc123",
        "providerUsername": "johndoe",
        "providerAvatar": "https://avatars.githubusercontent.com/u/123",
        "scopes": ["repo", "read:user"],
        "connectedAt": "2025-01-15T10:30:00Z"
      }
    },
    {
      "type": "gmail",
      "name": "Gmail",
      "description": "Search and read your emails",
      "available": true,
      "connected": false
    },
    {
      "type": "google_drive",
      "name": "Google Drive",
      "description": "Access your files and documents",
      "available": true,
      "connected": false
    },
    {
      "type": "google_calendar",
      "name": "Google Calendar",
      "description": "View your calendar events",
      "available": true,
      "connected": false
    }
  ]
}
```

### GET /v1/connectors/github/auth-url
```json
{
  "auth_url": "https://github.com/login/oauth/authorize?client_id=xxx&redirect_uri=xxx&scope=repo%20read:user&state=abc123",
  "state": "abc123"
}
```

---

## Sources

- [OpenAI Connectors and MCP servers](https://platform.openai.com/docs/guides/tools-connectors-mcp)
- [OpenAI Apps/Connectors in ChatGPT](https://help.openai.com/en/articles/11487775-connectors-in-chatgpt)
- [Claude Code GitHub Actions](https://docs.anthropic.com/en/docs/claude-code/github-actions)
- [Google OAuth 2.0](https://developers.google.com/identity/protocols/oauth2)
- [Google API Scopes](https://developers.google.com/identity/protocols/oauth2/scopes)
- [GitHub OAuth Apps](https://docs.github.com/en/apps/oauth-apps/using-oauth-apps/authorizing-oauth-apps)
- [GitHub Apps vs OAuth Apps](https://nango.dev/blog/github-app-vs-github-oauth)
