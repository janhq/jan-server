# OAuth Connectors Guide

This guide covers the OAuth connector integration in Jan Server, enabling users to connect external services like GitHub, Gmail, Google Drive, and Google Calendar to enhance AI assistant capabilities.

## Overview

Connectors allow users to securely authorize Jan Server to access their external accounts. Once connected, the AI assistant can use MCP (Model Context Protocol) tools to interact with these services on behalf of the user.

### Supported Connectors

| Connector | Description | MCP Tools | Capabilities |
|-----------|-------------|-----------|--------------|
| **GitHub** | Repository and code management | 16 tools | Repos, issues, PRs, commits, search, file operations |
| **Gmail** | Email access | 3 tools | Search and read emails |
| **Google Drive** | File storage access | 3 tools | Search, list, and read files |
| **Google Calendar** | Calendar management | 8 tools | Events, calendars, scheduling |

**Total: 30 MCP tools across 4 connectors**

---

## Architecture

### System Components

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Web Frontend  │────▶│    LLM API      │────▶│   MCP Tools     │
│  (React + Vite) │     │   (Go/Gin)      │     │    (Go/Gin)     │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │                       │
        │                       ▼                       │
        │               ┌─────────────────┐             │
        │               │   PostgreSQL    │             │
        │               │  (Token Store)  │             │
        │               └─────────────────┘             │
        │                       │                       │
        ▼                       ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    External OAuth Providers                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐ │
│  │  GitHub  │  │  Gmail   │  │  Drive   │  │ Google Calendar  │ │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **User initiates connection** via frontend
2. **LLM API generates OAuth URL** with PKCE challenge
3. **User authorizes** on provider's consent screen
4. **Provider redirects** back with authorization code
5. **LLM API exchanges code** for tokens (with PKCE verifier)
6. **Tokens encrypted** and stored in database
7. **MCP Tools retrieve tokens** from LLM API when executing tools

---

## Configuration

### Environment Variables

Add these variables to your `.env` file:

```bash
# ============================================
# CONNECTOR OAUTH CONFIGURATION
# ============================================

# --- GitHub OAuth App ---
# Create at: https://github.com/settings/developers
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GITHUB_ENABLED=true

# --- Google OAuth (Gmail, Drive, Calendar) ---
# Create at: https://console.cloud.google.com/apis/credentials
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GOOGLE_ENABLED=true

# --- Token Encryption ---
# Generate with: openssl rand -base64 32
CONNECTOR_ENCRYPTION_KEY=your_32_byte_base64_encoded_key

# --- OAuth URLs ---
# Base URL for OAuth callbacks (your server's public URL)
OAUTH_REDIRECT_BASE_URL=http://localhost:8000
# Frontend URL for post-OAuth redirect
OAUTH_FRONTEND_URL=http://localhost:3001
```

### Creating OAuth Applications

#### GitHub OAuth App

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click "New OAuth App"
3. Fill in:
   - **Application name:** Jan Server
   - **Homepage URL:** `http://localhost:3001`
   - **Authorization callback URL:** `http://localhost:8000/api/v1/connectors/github/callback`
4. Copy Client ID and Client Secret

#### Google OAuth Credentials

1. Go to [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. Create a new project or select existing
3. Enable APIs:
   - Gmail API
   - Google Drive API
   - Google Calendar API
4. Create OAuth 2.0 Client ID:
   - **Application type:** Web application
   - **Authorized redirect URIs:**
     - `http://localhost:8000/api/v1/connectors/gmail/callback`
     - `http://localhost:8000/api/v1/connectors/google_drive/callback`
     - `http://localhost:8000/api/v1/connectors/google_calendar/callback`
5. Copy Client ID and Client Secret

---

## API Reference

### Base URL
```
http://localhost:8000/v1/connectors
```

### Authentication
All endpoints require Bearer token authentication:
```
Authorization: Bearer <access_token>
```

### Endpoints

#### List Connectors
```http
GET /v1/connectors
```

**Response:**
```json
{
  "connectors": [
    {
      "type": "github",
      "display_name": "GitHub",
      "description": "Access repositories, issues, pull requests, and code",
      "icon_url": "/icons/github.svg",
      "is_connected": false,
      "has_write": true
    },
    {
      "type": "gmail",
      "display_name": "Gmail",
      "description": "Search and read email messages",
      "icon_url": "/icons/gmail.svg",
      "is_connected": true,
      "has_write": false
    }
  ]
}
```

#### Get Connector Status
```http
GET /v1/connectors/{type}/status
```

**Path Parameters:**
- `type`: `github`, `gmail`, `google_drive`, or `google_calendar`

**Response:**
```json
{
  "connected": true,
  "enabled": true,
  "username": "octocat",
  "email": "octocat@github.com",
  "connected_at": "2024-01-15T10:30:00Z",
  "expires_at": "2024-01-15T11:30:00Z"
}
```

#### Initiate OAuth Connection
```http
POST /v1/connectors/{type}/connect
```

**Response:**
```json
{
  "auth_url": "https://github.com/login/oauth/authorize?client_id=...&state=...&code_challenge=..."
}
```

#### Disconnect Connector
```http
DELETE /v1/connectors/{type}/disconnect
```

**Response:**
```json
{
  "disconnected": true
}
```

#### Refresh Tokens
```http
POST /v1/connectors/{type}/refresh
```

**Response:**
```json
{
  "refreshed": true
}
```

#### OAuth Callback (Public)
```http
GET /v1/connectors/{type}/callback?code=...&state=...
```

Redirects to frontend with status:
```
http://localhost:3001/connectors/callback?status=success&connector=github
```

---

## MCP Tools Reference

### GitHub Tools (16)

| Tool | Description | Parameters |
|------|-------------|------------|
| `github_list_repos` | List user's repositories | `visibility`, `sort`, `per_page` |
| `github_get_repo` | Get repository details | `owner`, `repo` |
| `github_list_issues` | List repository issues | `owner`, `repo`, `state`, `labels` |
| `github_get_issue` | Get issue details | `owner`, `repo`, `issue_number` |
| `github_create_issue` | Create a new issue | `owner`, `repo`, `title`, `body`, `labels` |
| `github_update_issue` | Update an issue | `owner`, `repo`, `issue_number`, `title`, `body`, `state` |
| `github_list_pull_requests` | List pull requests | `owner`, `repo`, `state`, `sort` |
| `github_get_pull_request` | Get PR details | `owner`, `repo`, `pull_number` |
| `github_create_pull_request` | Create a pull request | `owner`, `repo`, `title`, `body`, `head`, `base` |
| `github_list_commits` | List commits | `owner`, `repo`, `sha`, `per_page` |
| `github_get_commit` | Get commit details | `owner`, `repo`, `ref` |
| `github_search_code` | Search code | `query`, `per_page` |
| `github_search_repos` | Search repositories | `query`, `sort`, `per_page` |
| `github_get_file_content` | Get file content | `owner`, `repo`, `path`, `ref` |
| `github_create_or_update_file` | Create/update file | `owner`, `repo`, `path`, `message`, `content`, `sha` |
| `github_list_branches` | List branches | `owner`, `repo`, `per_page` |

### Gmail Tools (3)

| Tool | Description | Parameters |
|------|-------------|------------|
| `gmail_search_emails` | Search emails | `query`, `max_results` |
| `gmail_get_email` | Get email details | `message_id`, `format` |
| `gmail_list_labels` | List email labels | - |

### Google Drive Tools (3)

| Tool | Description | Parameters |
|------|-------------|------------|
| `drive_search_files` | Search files | `query`, `page_size`, `order_by` |
| `drive_list_files` | List files in folder | `folder_id`, `page_size` |
| `drive_get_file_content` | Get file content | `file_id` |

### Google Calendar Tools (8)

| Tool | Description | Parameters |
|------|-------------|------------|
| `calendar_list_calendars` | List calendars | - |
| `calendar_get_calendar` | Get calendar details | `calendar_id` |
| `calendar_list_events` | List events | `calendar_id`, `time_min`, `time_max`, `max_results` |
| `calendar_get_event` | Get event details | `calendar_id`, `event_id` |
| `calendar_create_event` | Create event | `calendar_id`, `summary`, `start`, `end`, `description` |
| `calendar_update_event` | Update event | `calendar_id`, `event_id`, `summary`, `start`, `end` |
| `calendar_delete_event` | Delete event | `calendar_id`, `event_id` |
| `calendar_quick_add` | Quick add event | `calendar_id`, `text` |

---

## Frontend Integration

### Connectors Tab Component

The connectors UI is available in the user profile page:

```
http://localhost:3001/profile?tab=connectors
```

### Zustand Store

```typescript
import { useConnectorStore } from '@/stores/connector-store';

// In your component
const {
  connectors,
  statuses,
  isLoading,
  fetchConnectors,
  initiateConnect,
  disconnect,
  refreshTokens,
} = useConnectorStore();
```

### OAuth Callback Handling

The frontend handles OAuth callbacks at `/connectors/callback`:

```typescript
// URL: /connectors/callback?status=success&connector=github
// Or:  /connectors/callback?status=error&message=...
```

The callback page:
1. Displays success/error message
2. Notifies parent window (if opened as popup)
3. Auto-closes after brief delay

---

## Security

### Token Encryption

OAuth tokens are encrypted at rest using AES-256-GCM:

```go
// Encryption flow
plaintext := accessToken
ciphertext, keyID := encryptor.Encrypt(plaintext)
// Store ciphertext and keyID in database

// Decryption flow
plaintext := encryptor.Decrypt(ciphertext, keyID)
```

### PKCE Flow

All OAuth flows use PKCE (Proof Key for Code Exchange) for enhanced security:

1. Generate random `code_verifier` (43-128 chars)
2. Create `code_challenge` = BASE64URL(SHA256(code_verifier))
3. Send `code_challenge` in authorization request
4. Send `code_verifier` in token exchange

### State Parameter

OAuth state includes:
- User ID
- Connector type
- HMAC signature for validation
- Expiration timestamp

### Token Refresh

Tokens are automatically refreshed when:
- Access token is within 5 minutes of expiry
- User explicitly requests refresh
- MCP tool execution detects expired token

---

## Database Schema

### Tables

```sql
-- User connector connections
CREATE TABLE llm_api.connector_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INTEGER NOT NULL,
    connector_type VARCHAR(32) NOT NULL,
    access_token_encrypted TEXT NOT NULL,
    refresh_token_encrypted TEXT,
    encryption_key_id VARCHAR(32) NOT NULL DEFAULT 'v1',
    token_type VARCHAR(32) DEFAULT 'Bearer',
    expires_at TIMESTAMP WITH TIME ZONE,
    provider_user_id VARCHAR(255),
    provider_username VARCHAR(255),
    provider_email VARCHAR(255),
    provider_avatar_url TEXT,
    scopes TEXT[],
    is_connected BOOLEAN NOT NULL DEFAULT true,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, connector_type)
);

-- OAuth state for PKCE flow
CREATE TABLE llm_api.connector_oauth_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INTEGER NOT NULL,
    connector_type VARCHAR(32) NOT NULL,
    state VARCHAR(128) NOT NULL UNIQUE,
    state_hash VARCHAR(64) NOT NULL,
    code_verifier VARCHAR(128),
    redirect_uri TEXT,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Audit log for connector operations
CREATE TABLE llm_api.connector_audit_log (
    id BIGSERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    connector_type VARCHAR(32) NOT NULL,
    action VARCHAR(64) NOT NULL,
    tool_name VARCHAR(128),
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    ip_address VARCHAR(45),
    user_agent TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

---

## Testing

### API Tests

Run the Postman collection:
```bash
make test-connectors
```

Or use newman:
```bash
newman run tests/e2e/automation/collections/connectors.postman.json \
  --env-var gateway_url=http://localhost:8000 \
  --env-var access_token=$TOKEN
```

### Playwright Tests

```bash
cd tests/e2e/playwright
npm install
npx playwright test
```

### Manual Testing

```bash
# Get access token
TOKEN=$(curl -s -X POST http://localhost:8000/auth/guest-login \
  -H "Content-Type: application/json" -d '{}' | jq -r '.access_token')

# List connectors
curl -s http://localhost:8000/v1/connectors \
  -H "Authorization: Bearer $TOKEN" | jq .

# Get connector status
curl -s http://localhost:8000/v1/connectors/github/status \
  -H "Authorization: Bearer $TOKEN" | jq .
```

---

## Troubleshooting

### Common Issues

#### "Connector not enabled"
- Check that `GITHUB_ENABLED=true` or `GOOGLE_ENABLED=true` is set
- Verify client ID and secret are configured

#### "OAuth state expired"
- States expire after 10 minutes
- User took too long to authorize
- Retry the connection flow

#### "Token refresh failed"
- Refresh token may have been revoked
- User needs to reconnect

#### "Invalid connector type"
- Valid types: `github`, `gmail`, `google_drive`, `google_calendar`

### Logs

Check LLM API logs for connector operations:
```bash
docker logs server-llm-api-1 2>&1 | grep -i connector
```

Check MCP Tools logs for tool execution:
```bash
docker logs server-mcp-tools-1 2>&1 | grep -i "github\|gmail\|drive\|calendar"
```

---

## Development

### Adding a New Connector

1. **Define connector type** in `entity.go`:
   ```go
   const ConnectorTypeSlack ConnectorType = "slack"
   ```

2. **Add OAuth configuration** in `oauth_provider.go`

3. **Create MCP tools** in `services/mcp-tools/internal/interfaces/httpserver/routes/mcp/`

4. **Update frontend** `connectors-tab.tsx` with icon and description

5. **Add environment variables** to `.env.template`

6. **Update tests** in Postman collection and Playwright

### Local Development

```bash
# Start infrastructure
make up-infra

# Run LLM API locally
cd services/llm-api && go run ./cmd/server

# Run MCP Tools locally
cd services/mcp-tools && go run ./cmd/server

# Run frontend
cd apps/web && npm run dev
```

---

## Related Documentation

- [Architecture Overview](../architecture/README.md)
- [MCP Tools Guide](./mcp-tools.md)
- [Authentication Guide](./authentication.md)
- [API Reference](../api/README.md)
