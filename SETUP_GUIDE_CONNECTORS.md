# Connectors Setup Guide

This guide provides step-by-step instructions for setting up OAuth applications and configuring the Connectors feature for GitHub, Gmail, Google Drive, and Google Calendar.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [GitHub OAuth App Setup](#1-github-oauth-app-setup)
3. [Google Cloud Console Setup](#2-google-cloud-console-setup)
4. [Environment Configuration](#3-environment-configuration)
5. [API Reference](#4-api-reference)
6. [Testing the Setup](#5-testing-the-setup)
7. [Troubleshooting](#6-troubleshooting)

---

## Prerequisites

Before starting, ensure you have:

- [ ] GitHub account with access to create OAuth Apps
- [ ] Google Cloud account with billing enabled (free tier works)
- [ ] Access to your application's environment configuration
- [ ] SSL/HTTPS configured for production (required by Google)

---

## 1. GitHub OAuth App Setup

### 1.1 Create OAuth Application

1. **Navigate to GitHub Developer Settings**
   - Go to [github.com](https://github.com) → Click your profile picture (top-right)
   - Select **Settings** → **Developer settings** (left sidebar, bottom)
   - Click **OAuth Apps** → **New OAuth App**

2. **Fill in Application Details**

   | Field | Development Value | Production Value |
   |-------|-------------------|------------------|
   | **Application name** | `Jan Server Dev` | `Jan Server` |
   | **Homepage URL** | `http://localhost:3001` | `https://jan.ai` |
   | **Application description** | `Jan Server connector for GitHub integration` | Same |
   | **Authorization callback URL** | `http://localhost:8000/api/v1/connectors/github/callback` | `https://api.jan.ai/v1/connectors/github/callback` |

3. **Register Application**
   - Click **Register application**
   - You'll see your **Client ID** immediately
   - Click **Generate a new client secret** → Copy and save it securely

### 1.2 GitHub OAuth Endpoints

| Purpose | Endpoint |
|---------|----------|
| Authorization | `https://github.com/login/oauth/authorize` |
| Token Exchange | `https://github.com/login/oauth/access_token` |
| API Base URL | `https://api.github.com` |

### 1.3 GitHub OAuth Flow

```
┌─────────┐                              ┌─────────┐                              ┌─────────┐
│  User   │                              │ Jan API │                              │ GitHub  │
└────┬────┘                              └────┬────┘                              └────┬────┘
     │                                        │                                        │
     │ 1. Click "Connect GitHub"              │                                        │
     │───────────────────────────────────────>│                                        │
     │                                        │                                        │
     │ 2. Redirect to GitHub auth URL         │                                        │
     │<───────────────────────────────────────│                                        │
     │                                        │                                        │
     │ 3. User authorizes app                 │                                        │
     │────────────────────────────────────────────────────────────────────────────────>│
     │                                        │                                        │
     │ 4. Redirect to callback with code      │                                        │
     │<────────────────────────────────────────────────────────────────────────────────│
     │                                        │                                        │
     │ 5. Send code to Jan API                │                                        │
     │───────────────────────────────────────>│                                        │
     │                                        │                                        │
     │                                        │ 6. Exchange code for token             │
     │                                        │───────────────────────────────────────>│
     │                                        │                                        │
     │                                        │ 7. Return access_token                 │
     │                                        │<───────────────────────────────────────│
     │                                        │                                        │
     │ 8. Connection successful               │                                        │
     │<───────────────────────────────────────│                                        │
```

### 1.4 Authorization URL Construction

```
GET https://github.com/login/oauth/authorize
  ?client_id={GITHUB_CLIENT_ID}
  &redirect_uri=http://localhost:8000/api/v1/connectors/github/callback
  &scope=repo read:user user:email
  &state={random_state}
  &code_challenge={PKCE_challenge}
  &code_challenge_method=S256
```

### 1.5 Token Exchange Request

```bash
POST https://github.com/login/oauth/access_token
Content-Type: application/x-www-form-urlencoded
Accept: application/json

client_id={GITHUB_CLIENT_ID}
&client_secret={GITHUB_CLIENT_SECRET}
&code={authorization_code}
&redirect_uri=http://localhost:8000/api/v1/connectors/github/callback
&code_verifier={PKCE_verifier}
```

**Response:**
```json
{
  "access_token": "gho_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "token_type": "bearer",
  "scope": "repo,read:user,user:email"
}
```

### 1.6 GitHub Scopes Reference

| Scope | Description | Use Case |
|-------|-------------|----------|
| `repo` | **Full access** to public and private repositories including read/write to code, commit statuses, PRs, issues | Core scope for all GitHub operations |
| `public_repo` | Access to public repositories only | If private repos not needed |
| `read:user` | Read user profile data | Get username, avatar |
| `user:email` | Read user email addresses | Get user email |
| `workflow` | Update GitHub Actions workflows | CI/CD management |
| `write:discussion` | Read/write discussions | Community engagement |

**Recommended scopes for Jan Connectors (Full Access - like Claude Code):**
```
repo read:user user:email workflow
```

**Scope Capabilities with `repo`:**

| Operation | Supported | API Endpoint |
|-----------|-----------|--------------|
| Read repositories | ✅ | `GET /repos/{owner}/{repo}` |
| Read file contents | ✅ | `GET /repos/{owner}/{repo}/contents/{path}` |
| Create/update files | ✅ | `PUT /repos/{owner}/{repo}/contents/{path}` |
| Create branches | ✅ | `POST /repos/{owner}/{repo}/git/refs` |
| Create commits | ✅ | `POST /repos/{owner}/{repo}/git/commits` |
| Create pull requests | ✅ | `POST /repos/{owner}/{repo}/pulls` |
| Review pull requests | ✅ | `POST /repos/{owner}/{repo}/pulls/{pull_number}/reviews` |
| Merge pull requests | ✅ | `PUT /repos/{owner}/{repo}/pulls/{pull_number}/merge` |
| Create/update issues | ✅ | `POST /repos/{owner}/{repo}/issues` |
| Add comments | ✅ | `POST /repos/{owner}/{repo}/issues/{issue_number}/comments` |

---

## 2. Google Cloud Console Setup

Google OAuth is used for Gmail, Google Drive, and Google Calendar. All three use the same OAuth credentials.

### 2.1 Create Google Cloud Project

1. **Go to Google Cloud Console**
   - Navigate to [console.cloud.google.com](https://console.cloud.google.com)
   - Click **Select a project** (top bar) → **New Project**

2. **Create Project**
   | Field | Value |
   |-------|-------|
   | Project name | `jan-server` |
   | Organization | (select if applicable) |
   | Location | (select if applicable) |

3. Click **Create** and wait for project creation

### 2.2 Enable Required APIs

1. Go to **APIs & Services** → **Library**
2. Search and enable each API:

   | API | Search Term | Enable |
   |-----|-------------|--------|
   | Gmail API | `Gmail API` | Click → **Enable** |
   | Google Drive API | `Google Drive API` | Click → **Enable** |
   | Google Calendar API | `Google Calendar API` | Click → **Enable** |

### 2.3 Configure OAuth Consent Screen

1. Go to **APIs & Services** → **OAuth consent screen**
2. Select **External** (for public apps) or **Internal** (for organization only)
3. Click **Create**

4. **Fill in App Information:**

   | Field | Value |
   |-------|-------|
   | App name | `Jan Server` |
   | User support email | `support@jan.ai` |
   | App logo | (optional) Upload logo |
   | Application home page | `https://jan.ai` |
   | Application privacy policy | `https://jan.ai/privacy` |
   | Application terms of service | `https://jan.ai/terms` |
   | Authorized domains | `jan.ai` |
   | Developer contact email | `dev@jan.ai` |

5. Click **Save and Continue**

6. **Add Scopes:**
   - Click **Add or Remove Scopes**
   - Add the following scopes:

   ```
   https://www.googleapis.com/auth/gmail.readonly
   https://www.googleapis.com/auth/drive.readonly
   https://www.googleapis.com/auth/calendar.readonly
   https://www.googleapis.com/auth/calendar.events.readonly
   https://www.googleapis.com/auth/userinfo.email
   https://www.googleapis.com/auth/userinfo.profile
   ```

7. Click **Save and Continue**

8. **Add Test Users** (while in testing mode):
   - Add email addresses of users who can test the app
   - Click **Save and Continue**

9. **Review and Submit** (for production):
   - After testing, submit for Google verification
   - Required for apps with sensitive scopes

### 2.4 Create OAuth Credentials

1. Go to **APIs & Services** → **Credentials**
2. Click **Create Credentials** → **OAuth client ID**
3. Select **Web application**

4. **Configure OAuth Client:**

   | Field | Value |
   |-------|-------|
   | Name | `Jan Server Web Client` |
   | Authorized JavaScript origins | `http://localhost:3001` (dev), `https://jan.ai` (prod) |
   | Authorized redirect URIs | See table below |

   **Authorized Redirect URIs:**
   ```
   # Development
   http://localhost:8000/api/v1/connectors/gmail/callback
   http://localhost:8000/api/v1/connectors/google_drive/callback
   http://localhost:8000/api/v1/connectors/google_calendar/callback

   # Production
   https://api.jan.ai/v1/connectors/gmail/callback
   https://api.jan.ai/v1/connectors/google_drive/callback
   https://api.jan.ai/v1/connectors/google_calendar/callback
   ```

5. Click **Create**
6. **Download JSON** or copy **Client ID** and **Client Secret**

### 2.5 Google OAuth Endpoints

| Purpose | Endpoint |
|---------|----------|
| Authorization | `https://accounts.google.com/o/oauth2/v2/auth` |
| Token Exchange | `https://oauth2.googleapis.com/token` |
| Token Revocation | `https://oauth2.googleapis.com/revoke` |
| Token Info | `https://oauth2.googleapis.com/tokeninfo` |

### 2.6 Google OAuth Flow

```
┌─────────┐                              ┌─────────┐                              ┌─────────┐
│  User   │                              │ Jan API │                              │ Google  │
└────┬────┘                              └────┬────┘                              └────┬────┘
     │                                        │                                        │
     │ 1. Click "Connect Gmail"               │                                        │
     │───────────────────────────────────────>│                                        │
     │                                        │                                        │
     │ 2. Redirect to Google auth URL         │                                        │
     │<───────────────────────────────────────│                                        │
     │                                        │                                        │
     │ 3. User signs in & grants consent      │                                        │
     │────────────────────────────────────────────────────────────────────────────────>│
     │                                        │                                        │
     │ 4. Redirect to callback with code      │                                        │
     │<────────────────────────────────────────────────────────────────────────────────│
     │                                        │                                        │
     │ 5. Send code to Jan API                │                                        │
     │───────────────────────────────────────>│                                        │
     │                                        │                                        │
     │                                        │ 6. Exchange code for tokens            │
     │                                        │───────────────────────────────────────>│
     │                                        │                                        │
     │                                        │ 7. Return access_token + refresh_token │
     │                                        │<───────────────────────────────────────│
     │                                        │                                        │
     │ 8. Connection successful               │                                        │
     │<───────────────────────────────────────│                                        │
```

### 2.7 Authorization URL Construction

```
GET https://accounts.google.com/o/oauth2/v2/auth
  ?client_id={GOOGLE_CLIENT_ID}
  &redirect_uri=http://localhost:8000/api/v1/connectors/gmail/callback
  &response_type=code
  &scope=https://www.googleapis.com/auth/gmail.readonly https://www.googleapis.com/auth/userinfo.email
  &access_type=offline
  &prompt=consent
  &state={random_state}
  &code_challenge={PKCE_challenge}
  &code_challenge_method=S256
```

**Key Parameters:**
| Parameter | Value | Description |
|-----------|-------|-------------|
| `access_type` | `offline` | **Required** to get refresh token |
| `prompt` | `consent` | Forces consent screen to get refresh token |
| `include_granted_scopes` | `true` | Supports incremental authorization |

### 2.8 Token Exchange Request

```bash
POST https://oauth2.googleapis.com/token
Content-Type: application/x-www-form-urlencoded

client_id={GOOGLE_CLIENT_ID}
&client_secret={GOOGLE_CLIENT_SECRET}
&code={authorization_code}
&code_verifier={PKCE_verifier}
&grant_type=authorization_code
&redirect_uri=http://localhost:8000/api/v1/connectors/gmail/callback
```

**Response:**
```json
{
  "access_token": "ya29.a0AfH6SMBxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "expires_in": 3599,
  "refresh_token": "1//0exxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "scope": "https://www.googleapis.com/auth/gmail.readonly https://www.googleapis.com/auth/userinfo.email",
  "token_type": "Bearer"
}
```

### 2.9 Token Refresh Request

```bash
POST https://oauth2.googleapis.com/token
Content-Type: application/x-www-form-urlencoded

client_id={GOOGLE_CLIENT_ID}
&client_secret={GOOGLE_CLIENT_SECRET}
&refresh_token={refresh_token}
&grant_type=refresh_token
```

**Response:**
```json
{
  "access_token": "ya29.a0AfH6SMBnewtoken...",
  "expires_in": 3599,
  "scope": "https://www.googleapis.com/auth/gmail.readonly",
  "token_type": "Bearer"
}
```

### 2.10 Google Scopes Reference

#### Gmail Scopes
| Scope | Description | Sensitivity |
|-------|-------------|-------------|
| `gmail.readonly` | View email messages and settings | Sensitive |
| `gmail.metadata` | View email metadata (headers, labels) | Sensitive |
| `gmail.modify` | Read, compose, send emails | Restricted |
| `gmail.send` | Send email on behalf of user | Restricted |

**Recommended for Jan:** `gmail.readonly`

#### Google Drive Scopes
| Scope | Description | Sensitivity |
|-------|-------------|-------------|
| `drive.readonly` | See and download all Drive files | Sensitive |
| `drive.metadata.readonly` | View file metadata only | Non-sensitive |
| `drive.file` | Access files created by app only | Non-sensitive |
| `drive` | Full access to Drive | Restricted |

**Recommended for Jan:** `drive.readonly`

#### Google Calendar Scopes
| Scope | Description | Sensitivity |
|-------|-------------|-------------|
| `calendar.readonly` | View all calendars | Sensitive |
| `calendar.events.readonly` | View events on calendars | Sensitive |
| `calendar.events` | View and edit events | Sensitive |
| `calendar` | Full calendar access | Restricted |

**Recommended for Jan:** `calendar.readonly` + `calendar.events.readonly`

---

## 3. Environment Configuration

### 3.1 Required Environment Variables

Add these to your `.env` file:

```bash
# =============================================================================
# CONNECTORS CONFIGURATION
# =============================================================================

# -----------------------------------------------------------------------------
# GitHub Connector
# -----------------------------------------------------------------------------
GITHUB_CLIENT_ID=Ov23lixxxxxxxxxxxxxxxxxx
GITHUB_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
GITHUB_CONNECTOR_ENABLED=true

# -----------------------------------------------------------------------------
# Google Connectors (shared credentials for Gmail, Drive, Calendar)
# -----------------------------------------------------------------------------
GOOGLE_CLIENT_ID=xxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
GOOGLE_CONNECTOR_ENABLED=true

# -----------------------------------------------------------------------------
# OAuth Configuration
# -----------------------------------------------------------------------------
# Base URL for OAuth callbacks (no trailing slash)
OAUTH_REDIRECT_BASE_URL=http://localhost:8000

# Frontend URL for post-auth redirects
OAUTH_FRONTEND_URL=http://localhost:3001

# -----------------------------------------------------------------------------
# Security: Token Encryption
# -----------------------------------------------------------------------------
# Generate with: openssl rand -hex 32
CONNECTOR_TOKEN_ENCRYPTION_KEY=your_64_character_hex_string_here

# Current encryption key ID (for rotation support)
CONNECTOR_TOKEN_ENCRYPTION_KEY_ID=v1

# Previous encryption key (optional, for rotation)
# CONNECTOR_TOKEN_ENCRYPTION_KEY_PREVIOUS=old_key_here
# CONNECTOR_TOKEN_ENCRYPTION_KEY_ID_PREVIOUS=v0

# -----------------------------------------------------------------------------
# Security: OAuth State
# -----------------------------------------------------------------------------
# HMAC secret for state parameter signing
# Generate with: openssl rand -hex 32
OAUTH_STATE_SECRET=your_64_character_hex_string_here

# State expiration in seconds (default: 300 = 5 minutes)
OAUTH_STATE_EXPIRATION_SECONDS=300

# -----------------------------------------------------------------------------
# Rate Limiting
# -----------------------------------------------------------------------------
# OAuth initiation rate limit (per user per hour)
OAUTH_RATE_LIMIT_INITIATE=10

# Connector API rate limit (per user per minute)
CONNECTOR_API_RATE_LIMIT=100
```

### 3.2 Generate Encryption Keys

```bash
# Generate token encryption key (32 bytes = 64 hex chars)
openssl rand -hex 32
# Example output: a1b2c3d4e5f6...

# Generate OAuth state secret (32 bytes = 64 hex chars)
openssl rand -hex 32
# Example output: f6e5d4c3b2a1...
```

### 3.3 Configuration Struct (Go)

```go
// internal/infrastructure/config/connector_config.go

type ConnectorConfig struct {
    // GitHub
    GitHubClientID       string `env:"GITHUB_CLIENT_ID"`
    GitHubClientSecret   string `env:"GITHUB_CLIENT_SECRET"`
    GitHubEnabled        bool   `env:"GITHUB_CONNECTOR_ENABLED" envDefault:"false"`

    // Google (shared for Gmail, Drive, Calendar)
    GoogleClientID       string `env:"GOOGLE_CLIENT_ID"`
    GoogleClientSecret   string `env:"GOOGLE_CLIENT_SECRET"`
    GoogleEnabled        bool   `env:"GOOGLE_CONNECTOR_ENABLED" envDefault:"false"`

    // OAuth URLs
    OAuthRedirectBaseURL string `env:"OAUTH_REDIRECT_BASE_URL" envDefault:"http://localhost:8000"`
    OAuthFrontendURL     string `env:"OAUTH_FRONTEND_URL" envDefault:"http://localhost:3001"`

    // Security
    TokenEncryptionKey   string `env:"CONNECTOR_TOKEN_ENCRYPTION_KEY" required:"true"`
    TokenEncryptionKeyID string `env:"CONNECTOR_TOKEN_ENCRYPTION_KEY_ID" envDefault:"v1"`
    OAuthStateSecret     string `env:"OAUTH_STATE_SECRET" required:"true"`
    OAuthStateExpiration int    `env:"OAUTH_STATE_EXPIRATION_SECONDS" envDefault:"300"`

    // Rate Limiting
    OAuthRateLimitInitiate int `env:"OAUTH_RATE_LIMIT_INITIATE" envDefault:"10"`
    ConnectorAPIRateLimit  int `env:"CONNECTOR_API_RATE_LIMIT" envDefault:"100"`
}

// Validate checks required fields
func (c *ConnectorConfig) Validate() error {
    if c.GitHubEnabled && (c.GitHubClientID == "" || c.GitHubClientSecret == "") {
        return errors.New("GitHub enabled but credentials not configured")
    }
    if c.GoogleEnabled && (c.GoogleClientID == "" || c.GoogleClientSecret == "") {
        return errors.New("Google enabled but credentials not configured")
    }
    if len(c.TokenEncryptionKey) != 64 {
        return errors.New("CONNECTOR_TOKEN_ENCRYPTION_KEY must be 64 hex characters (32 bytes)")
    }
    if len(c.OAuthStateSecret) != 64 {
        return errors.New("OAUTH_STATE_SECRET must be 64 hex characters (32 bytes)")
    }
    return nil
}
```

---

## 4. API Reference

### 4.1 GitHub API Calls

#### Get Authenticated User
```bash
GET https://api.github.com/user
Authorization: Bearer {access_token}
Accept: application/vnd.github+json
X-GitHub-Api-Version: 2022-11-28
```

**Response:**
```json
{
  "login": "octocat",
  "id": 1,
  "avatar_url": "https://github.com/images/error/octocat_happy.gif",
  "email": "octocat@github.com",
  "name": "The Octocat"
}
```

#### Search Repositories
```bash
GET https://api.github.com/search/repositories?q={query}+user:{username}
Authorization: Bearer {access_token}
```

#### List User Repositories
```bash
GET https://api.github.com/user/repos?sort=updated&per_page=30
Authorization: Bearer {access_token}
```

#### Search Issues
```bash
GET https://api.github.com/search/issues?q={query}+is:issue+user:{username}
Authorization: Bearer {access_token}
```

#### Get File Content
```bash
GET https://api.github.com/repos/{owner}/{repo}/contents/{path}
Authorization: Bearer {access_token}
```

#### List Pull Requests
```bash
GET https://api.github.com/repos/{owner}/{repo}/pulls?state=all
Authorization: Bearer {access_token}
```

#### Create a Branch
```bash
# First, get the SHA of the base branch
GET https://api.github.com/repos/{owner}/{repo}/git/ref/heads/{base_branch}
Authorization: Bearer {access_token}

# Response: { "object": { "sha": "abc123..." } }

# Then create the new branch
POST https://api.github.com/repos/{owner}/{repo}/git/refs
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "ref": "refs/heads/{new_branch_name}",
  "sha": "{base_branch_sha}"
}
```

#### Create or Update File (Commit)
```bash
PUT https://api.github.com/repos/{owner}/{repo}/contents/{path}
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "message": "feat: add new feature",
  "content": "{base64_encoded_content}",
  "branch": "{branch_name}",
  "sha": "{existing_file_sha}"  // Required only for updates, not new files
}
```

**Response:**
```json
{
  "content": {
    "name": "file.txt",
    "path": "path/to/file.txt",
    "sha": "new_file_sha..."
  },
  "commit": {
    "sha": "commit_sha...",
    "message": "feat: add new feature",
    "html_url": "https://github.com/owner/repo/commit/..."
  }
}
```

#### Create Pull Request
```bash
POST https://api.github.com/repos/{owner}/{repo}/pulls
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "title": "feat: add new feature",
  "body": "## Summary\n\nThis PR adds...\n\n## Changes\n- Added X\n- Updated Y",
  "head": "{branch_with_changes}",
  "base": "main",
  "draft": false
}
```

**Response:**
```json
{
  "number": 123,
  "html_url": "https://github.com/owner/repo/pull/123",
  "state": "open",
  "title": "feat: add new feature",
  "head": { "ref": "feature-branch" },
  "base": { "ref": "main" }
}
```

#### Add PR Review Comment
```bash
POST https://api.github.com/repos/{owner}/{repo}/pulls/{pull_number}/reviews
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "body": "LGTM! Great work on this feature.",
  "event": "APPROVE"  // or "REQUEST_CHANGES", "COMMENT"
}
```

#### Merge Pull Request
```bash
PUT https://api.github.com/repos/{owner}/{repo}/pulls/{pull_number}/merge
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "commit_title": "feat: add new feature (#123)",
  "merge_method": "squash"  // or "merge", "rebase"
}
```

#### Create Issue
```bash
POST https://api.github.com/repos/{owner}/{repo}/issues
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "title": "Bug: Something is broken",
  "body": "## Description\n\nWhen I do X, Y happens instead of Z.",
  "labels": ["bug", "priority:high"],
  "assignees": ["username"]
}
```

#### Add Comment to Issue/PR
```bash
POST https://api.github.com/repos/{owner}/{repo}/issues/{issue_number}/comments
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "body": "Thanks for reporting this! I'll look into it."
}
```

### 4.2 Gmail API Calls

**Base URL:** `https://gmail.googleapis.com/gmail/v1`

#### List Messages
```bash
GET https://gmail.googleapis.com/gmail/v1/users/me/messages
  ?q={search_query}
  &maxResults=10
Authorization: Bearer {access_token}
```

**Search Query Examples:**
- `from:boss@company.com` - Emails from specific sender
- `subject:meeting` - Subject contains "meeting"
- `after:2024/01/01` - Emails after date
- `has:attachment` - Emails with attachments
- `is:unread` - Unread emails

**Response:**
```json
{
  "messages": [
    {"id": "18d5a1b2c3d4e5f6", "threadId": "18d5a1b2c3d4e5f6"},
    {"id": "18d5a1b2c3d4e5f7", "threadId": "18d5a1b2c3d4e5f7"}
  ],
  "nextPageToken": "...",
  "resultSizeEstimate": 42
}
```

#### Get Message
```bash
GET https://gmail.googleapis.com/gmail/v1/users/me/messages/{id}
  ?format=full
Authorization: Bearer {access_token}
```

**Response:**
```json
{
  "id": "18d5a1b2c3d4e5f6",
  "threadId": "18d5a1b2c3d4e5f6",
  "snippet": "Hello, this is the email preview...",
  "payload": {
    "headers": [
      {"name": "From", "value": "sender@example.com"},
      {"name": "To", "value": "recipient@example.com"},
      {"name": "Subject", "value": "Meeting Tomorrow"},
      {"name": "Date", "value": "Mon, 27 Jan 2025 10:00:00 -0800"}
    ],
    "body": {
      "data": "base64_encoded_content..."
    }
  }
}
```

#### List Labels
```bash
GET https://gmail.googleapis.com/gmail/v1/users/me/labels
Authorization: Bearer {access_token}
```

### 4.3 Google Drive API Calls

**Base URL:** `https://www.googleapis.com/drive/v3`

#### List Files
```bash
GET https://www.googleapis.com/drive/v3/files
  ?q={search_query}
  &pageSize=10
  &fields=files(id,name,mimeType,modifiedTime,webViewLink)
Authorization: Bearer {access_token}
```

**Search Query Examples:**
- `name contains 'report'` - Files with "report" in name
- `mimeType='application/vnd.google-apps.document'` - Google Docs only
- `modifiedTime > '2024-01-01T00:00:00'` - Modified after date
- `'me' in owners` - Files owned by user
- `trashed = false` - Exclude trashed files

**Response:**
```json
{
  "files": [
    {
      "id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
      "name": "Q4 Report",
      "mimeType": "application/vnd.google-apps.document",
      "modifiedTime": "2025-01-27T15:30:00.000Z",
      "webViewLink": "https://docs.google.com/document/d/1BxiMV.../edit"
    }
  ],
  "nextPageToken": "..."
}
```

#### Get File Metadata
```bash
GET https://www.googleapis.com/drive/v3/files/{fileId}
  ?fields=id,name,mimeType,size,modifiedTime,webViewLink,owners
Authorization: Bearer {access_token}
```

#### Export Google Doc Content
```bash
GET https://www.googleapis.com/drive/v3/files/{fileId}/export
  ?mimeType=text/plain
Authorization: Bearer {access_token}
```

**Supported Export Types:**
| Google Format | Export MIME Type |
|---------------|------------------|
| Google Docs | `text/plain`, `text/html`, `application/pdf` |
| Google Sheets | `text/csv`, `application/pdf` |
| Google Slides | `application/pdf`, `text/plain` |

#### Download Binary File
```bash
GET https://www.googleapis.com/drive/v3/files/{fileId}
  ?alt=media
Authorization: Bearer {access_token}
```

### 4.4 Google Calendar API Calls

**Base URL:** `https://www.googleapis.com/calendar/v3`

#### List Calendars
```bash
GET https://www.googleapis.com/calendar/v3/users/me/calendarList
Authorization: Bearer {access_token}
```

**Response:**
```json
{
  "items": [
    {
      "id": "primary",
      "summary": "john@example.com",
      "timeZone": "America/Los_Angeles",
      "accessRole": "owner"
    },
    {
      "id": "team@group.calendar.google.com",
      "summary": "Team Calendar",
      "accessRole": "reader"
    }
  ]
}
```

#### List Events
```bash
GET https://www.googleapis.com/calendar/v3/calendars/{calendarId}/events
  ?timeMin={ISO_datetime}
  &timeMax={ISO_datetime}
  &maxResults=10
  &singleEvents=true
  &orderBy=startTime
Authorization: Bearer {access_token}
```

**Parameters:**
| Parameter | Format | Example |
|-----------|--------|---------|
| `timeMin` | RFC3339 | `2025-01-27T00:00:00Z` |
| `timeMax` | RFC3339 | `2025-02-27T00:00:00Z` |
| `calendarId` | string | `primary` or calendar ID |

**Response:**
```json
{
  "items": [
    {
      "id": "event123",
      "summary": "Team Meeting",
      "description": "Weekly sync",
      "start": {
        "dateTime": "2025-01-28T10:00:00-08:00",
        "timeZone": "America/Los_Angeles"
      },
      "end": {
        "dateTime": "2025-01-28T11:00:00-08:00",
        "timeZone": "America/Los_Angeles"
      },
      "attendees": [
        {"email": "alice@example.com", "responseStatus": "accepted"},
        {"email": "bob@example.com", "responseStatus": "tentative"}
      ],
      "htmlLink": "https://www.google.com/calendar/event?eid=..."
    }
  ],
  "nextPageToken": "..."
}
```

#### Search Events
```bash
GET https://www.googleapis.com/calendar/v3/calendars/{calendarId}/events
  ?q={search_query}
  &maxResults=10
Authorization: Bearer {access_token}
```

#### Get Event
```bash
GET https://www.googleapis.com/calendar/v3/calendars/{calendarId}/events/{eventId}
Authorization: Bearer {access_token}
```

---

## 5. Testing the Setup

### 5.1 Test GitHub Connection

```bash
# 1. Get authorization URL
curl -X GET "http://localhost:8000/api/v1/connectors/github/auth-url" \
  -H "Authorization: Bearer {your_jwt_token}"

# Response:
# {
#   "auth_url": "https://github.com/login/oauth/authorize?client_id=...",
#   "state": "abc123..."
# }

# 2. Open auth_url in browser, authorize the app

# 3. After redirect, complete connection with the code
curl -X POST "http://localhost:8000/api/v1/connectors/github/connect" \
  -H "Authorization: Bearer {your_jwt_token}" \
  -H "Content-Type: application/json" \
  -d '{"code": "code_from_callback", "state": "abc123..."}'

# 4. Verify connection
curl -X GET "http://localhost:8000/api/v1/connectors/github/status" \
  -H "Authorization: Bearer {your_jwt_token}"
```

### 5.2 Test Google Connection

```bash
# 1. Get authorization URL for Gmail
curl -X GET "http://localhost:8000/api/v1/connectors/gmail/auth-url" \
  -H "Authorization: Bearer {your_jwt_token}"

# 2. Open auth_url in browser, sign in with Google, grant consent

# 3. Complete connection
curl -X POST "http://localhost:8000/api/v1/connectors/gmail/connect" \
  -H "Authorization: Bearer {your_jwt_token}" \
  -H "Content-Type: application/json" \
  -d '{"code": "4/0AfJohX...", "state": "xyz789..."}'

# 4. Verify all Google connections
curl -X GET "http://localhost:8000/api/v1/connectors" \
  -H "Authorization: Bearer {your_jwt_token}"
```

### 5.3 Test API Calls

```bash
# Test GitHub API through connector
curl -X POST "http://localhost:8000/api/v1/mcp/tools/call" \
  -H "Authorization: Bearer {your_jwt_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "tool_name": "github_search_repositories",
    "arguments": {"query": "react"}
  }'

# Test Gmail API through connector
curl -X POST "http://localhost:8000/api/v1/mcp/tools/call" \
  -H "Authorization: Bearer {your_jwt_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "tool_name": "gmail_search_emails",
    "arguments": {"query": "from:support@github.com", "max_results": 5}
  }'
```

### 5.4 Verify Token Encryption

```bash
# Connect to database and verify tokens are encrypted
psql -U postgres -d jandb

# Check that tokens look encrypted (should be base64-encoded ciphertext)
SELECT id, connector_type,
       LEFT(access_token_encrypted, 50) as token_preview,
       encryption_key_id
FROM connector_connections;

# Tokens should NOT be readable as plain text
# Expected format: "djE6..." (base64 encoded: keyID:nonce:ciphertext)
```

---

## 6. Troubleshooting

### 6.1 Common GitHub Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| "redirect_uri mismatch" | Callback URL doesn't match | Verify callback URL in GitHub OAuth App settings matches exactly |
| "Bad credentials" | Invalid or expired token | Check token encryption/decryption, verify token not revoked |
| "Not Found" | Insufficient scopes | Request `repo` scope for private repos |
| "API rate limit exceeded" | Too many requests | Implement rate limiting, use conditional requests |

### 6.2 Common Google Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| "redirect_uri_mismatch" | URI not in allowlist | Add exact URI to Google Cloud Console |
| "access_denied" | User denied consent | Handle gracefully, show retry option |
| "invalid_grant" | Refresh token expired/revoked | Re-authenticate user |
| "deleted_client" | OAuth app deleted | Create new credentials |
| "Insufficient Permission" | Missing scope | Request additional scopes |

### 6.3 Google Verification Requirements

For production apps with sensitive scopes:

1. **Verification Required** for:
   - `gmail.readonly` (sensitive)
   - `drive.readonly` (sensitive)
   - `calendar.readonly` (sensitive)

2. **Verification Process:**
   - Submit app in Google Cloud Console
   - Provide privacy policy
   - Demonstrate scope justification
   - Complete security assessment (for restricted scopes)

3. **While Unverified:**
   - Limited to 100 test users
   - Shows "unverified app" warning
   - 7-day expiration on refresh tokens

### 6.4 Debug Logging

Enable debug logging for troubleshooting:

```go
// In connector service
func (s *ConnectorService) Connect(ctx context.Context, ...) error {
    s.logger.Debug().
        Str("connector_type", string(connectorType)).
        Str("user_id", fmt.Sprint(userID)).
        Msg("initiating connector OAuth flow")

    // ... connection logic

    if err != nil {
        s.logger.Error().
            Err(err).
            Str("connector_type", string(connectorType)).
            // NEVER log tokens!
            Msg("connector OAuth failed")
    }
}
```

### 6.5 Health Check Endpoint

```bash
# Check connector service health
curl -X GET "http://localhost:8000/api/v1/connectors/health"

# Response:
{
  "status": "healthy",
  "connectors": {
    "github": {"enabled": true, "configured": true},
    "gmail": {"enabled": true, "configured": true},
    "google_drive": {"enabled": true, "configured": true},
    "google_calendar": {"enabled": true, "configured": true}
  }
}
```

---

## Quick Reference Card

### OAuth URLs

| Provider | Authorization | Token | Revoke |
|----------|---------------|-------|--------|
| GitHub | `github.com/login/oauth/authorize` | `github.com/login/oauth/access_token` | N/A |
| Google | `accounts.google.com/o/oauth2/v2/auth` | `oauth2.googleapis.com/token` | `oauth2.googleapis.com/revoke` |

### API Base URLs

| Service | Base URL |
|---------|----------|
| GitHub | `https://api.github.com` |
| Gmail | `https://gmail.googleapis.com/gmail/v1` |
| Google Drive | `https://www.googleapis.com/drive/v3` |
| Google Calendar | `https://www.googleapis.com/calendar/v3` |

### Recommended Scopes

| Connector | Scopes |
|-----------|--------|
| GitHub | `repo read:user user:email` |
| Gmail | `gmail.readonly` |
| Google Drive | `drive.readonly` |
| Google Calendar | `calendar.readonly calendar.events.readonly` |

---

## Sources

- [GitHub: Creating an OAuth App](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/creating-an-oauth-app)
- [GitHub: Authorizing OAuth Apps](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps)
- [GitHub: Scopes for OAuth Apps](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/scopes-for-oauth-apps)
- [Google: OAuth 2.0 for Web Server Applications](https://developers.google.com/identity/protocols/oauth2/web-server)
- [Google: OAuth 2.0 Scopes](https://developers.google.com/identity/protocols/oauth2/scopes)
- [Gmail API Reference](https://developers.google.com/gmail/api/reference/rest)
- [Google Drive API Reference](https://developers.google.com/drive/api/reference/rest/v3)
- [Google Calendar API Reference](https://developers.google.com/calendar/api/v3/reference)
