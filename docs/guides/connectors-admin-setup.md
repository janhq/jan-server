# Connectors Admin Setup Guide

This guide provides step-by-step instructions for administrators to configure OAuth connectors for GitHub and Google services (Gmail, Google Drive, Google Calendar).

## Prerequisites

- Admin access to Jan Server deployment
- Access to create OAuth applications on GitHub and Google Cloud Console
- Ability to modify environment variables on the server

---

## Quick Setup Checklist

- [ ] Generate encryption key for token storage
- [ ] Create GitHub OAuth App
- [ ] Create Google Cloud Project and OAuth credentials
- [ ] Enable required Google APIs
- [ ] Configure environment variables
- [ ] Restart services
- [ ] Verify configuration

---

## Step 1: Generate Encryption Key

OAuth tokens are encrypted at rest. Generate a secure encryption key:

```bash
# Generate a 32-byte base64-encoded key
openssl rand -base64 32
```

**Example output:**
```
K7gNU3sdo+OL0wNhqoVWhr3g6s1xYv72ol/pe/Unols=
```

Save this key securely - you'll need it for the `CONNECTOR_ENCRYPTION_KEY` environment variable.

> **Warning:** If you lose this key, all stored OAuth tokens will become unreadable and users will need to reconnect their accounts.

---

## Step 2: Create GitHub OAuth App

### 2.1 Navigate to GitHub Developer Settings

1. Log in to GitHub with an account that has admin access
2. Go to **Settings** → **Developer settings** → **OAuth Apps**
3. Or visit directly: https://github.com/settings/developers

### 2.2 Create New OAuth App

Click **"New OAuth App"** and fill in:

| Field | Value |
|-------|-------|
| **Application name** | `Jan Server` (or your preferred name) |
| **Homepage URL** | Your frontend URL (e.g., `https://jan.example.com`) |
| **Application description** | Optional description |
| **Authorization callback URL** | `https://your-api-domain.com/api/v1/connectors/github/callback` |

**Examples for different environments:**

| Environment | Homepage URL | Callback URL |
|-------------|--------------|--------------|
| Local Development | `http://localhost:3001` | `http://localhost:8000/api/v1/connectors/github/callback` |
| Staging | `https://staging.jan.example.com` | `https://api-staging.jan.example.com/api/v1/connectors/github/callback` |
| Production | `https://jan.example.com` | `https://api.jan.example.com/api/v1/connectors/github/callback` |

### 2.3 Save Credentials

After creating the app:

1. Copy the **Client ID** (visible immediately)
2. Click **"Generate a new client secret"**
3. Copy the **Client Secret** (only shown once!)

Store these securely:
```
GITHUB_CLIENT_ID=Iv1.xxxxxxxxxxxxxxxx
GITHUB_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### 2.4 Configure App Settings (Optional)

For organization-wide deployment:

1. Go to the OAuth App settings
2. Under **"Organization access"**, request access for your organization
3. An organization admin must approve the request

---

## Step 3: Create Google Cloud Project

### 3.1 Create or Select Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click the project dropdown at the top
3. Click **"New Project"** or select an existing project

**New Project Settings:**
- **Project name:** `Jan Server` (or your preferred name)
- **Organization:** Select your organization (if applicable)
- **Location:** Select folder (if applicable)

### 3.2 Enable Required APIs

Navigate to **APIs & Services** → **Library** and enable:

1. **Gmail API**
   - Search for "Gmail API"
   - Click **Enable**

2. **Google Drive API**
   - Search for "Google Drive API"
   - Click **Enable**

3. **Google Calendar API**
   - Search for "Google Calendar API"
   - Click **Enable**

**Quick enable via gcloud CLI:**
```bash
gcloud services enable gmail.googleapis.com
gcloud services enable drive.googleapis.com
gcloud services enable calendar-json.googleapis.com
```

### 3.3 Configure OAuth Consent Screen

1. Go to **APIs & Services** → **OAuth consent screen**
2. Select **User Type**:
   - **Internal**: Only users in your Google Workspace organization
   - **External**: Any Google account (requires verification for production)

3. Fill in the **App information**:

| Field | Value |
|-------|-------|
| **App name** | `Jan Server` |
| **User support email** | Your support email |
| **App logo** | Optional - upload your logo |
| **Application home page** | Your frontend URL |
| **Application privacy policy link** | Your privacy policy URL |
| **Application terms of service link** | Your terms URL |
| **Authorized domains** | Add your domain (e.g., `example.com`) |
| **Developer contact email** | Your developer email |

4. Click **Save and Continue**

### 3.4 Configure Scopes

Add the following scopes:

**Gmail Scopes:**
```
https://www.googleapis.com/auth/gmail.readonly
```

**Google Drive Scopes:**
```
https://www.googleapis.com/auth/drive.readonly
https://www.googleapis.com/auth/drive.metadata.readonly
```

**Google Calendar Scopes:**
```
https://www.googleapis.com/auth/calendar.readonly
https://www.googleapis.com/auth/calendar.events
```

Click **Save and Continue**

### 3.5 Add Test Users (External Apps Only)

For external apps in testing mode:
1. Add email addresses of users who can test the integration
2. Maximum 100 test users before verification

### 3.6 Create OAuth Credentials

1. Go to **APIs & Services** → **Credentials**
2. Click **"Create Credentials"** → **"OAuth client ID"**
3. Select **Application type**: **Web application**
4. Enter **Name**: `Jan Server Web Client`

5. Add **Authorized JavaScript origins**:
   ```
   https://your-frontend-domain.com
   http://localhost:3001  (for development)
   ```

6. Add **Authorized redirect URIs**:
   ```
   https://your-api-domain.com/api/v1/connectors/gmail/callback
   https://your-api-domain.com/api/v1/connectors/google_drive/callback
   https://your-api-domain.com/api/v1/connectors/google_calendar/callback
   ```

   For local development, also add:
   ```
   http://localhost:8000/api/v1/connectors/gmail/callback
   http://localhost:8000/api/v1/connectors/google_drive/callback
   http://localhost:8000/api/v1/connectors/google_calendar/callback
   ```

7. Click **Create**

### 3.7 Save Credentials

Copy the credentials:
```
GOOGLE_CLIENT_ID=xxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxxxxxxxxxxxxxxxxxx
```

---

## Step 4: Configure Environment Variables

Add the following to your `.env` file:

```bash
# ============================================
# CONNECTOR CONFIGURATION
# ============================================

# Token Encryption Key (from Step 1)
CONNECTOR_ENCRYPTION_KEY=K7gNU3sdo+OL0wNhqoVWhr3g6s1xYv72ol/pe/Unols=

# GitHub OAuth (from Step 2)
GITHUB_CLIENT_ID=Iv1.xxxxxxxxxxxxxxxx
GITHUB_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
GITHUB_ENABLED=true

# Google OAuth (from Step 3)
GOOGLE_CLIENT_ID=xxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxxxxxxxxxxxxxxxxxx
GOOGLE_ENABLED=true

# OAuth URLs
# API base URL where callbacks are received
OAUTH_REDIRECT_BASE_URL=https://api.jan.example.com
# Frontend URL for post-auth redirect
OAUTH_FRONTEND_URL=https://jan.example.com
```

### Environment-Specific Configurations

**Local Development:**
```bash
OAUTH_REDIRECT_BASE_URL=http://localhost:8000
OAUTH_FRONTEND_URL=http://localhost:3001
```

**Docker Compose:**
```bash
OAUTH_REDIRECT_BASE_URL=http://localhost:8000
OAUTH_FRONTEND_URL=http://localhost:3001
```

**Kubernetes/Production:**
```bash
OAUTH_REDIRECT_BASE_URL=https://api.jan.example.com
OAUTH_FRONTEND_URL=https://jan.example.com
```

---

## Step 5: Restart Services

After updating environment variables, restart the services:

**Docker Compose:**
```bash
make down
make up-full
```

**Kubernetes:**
```bash
kubectl rollout restart deployment/llm-api -n jan
kubectl rollout restart deployment/mcp-tools -n jan
```

**Verify services are running:**
```bash
make health-check
```

---

## Step 6: Verify Configuration

### 6.1 Check Connector Status via API

```bash
# Get an access token
TOKEN=$(curl -s -X POST http://localhost:8000/auth/guest-login \
  -H "Content-Type: application/json" -d '{}' | jq -r '.access_token')

# List connectors - should show enabled connectors
curl -s http://localhost:8000/v1/connectors \
  -H "Authorization: Bearer $TOKEN" | jq .
```

**Expected output:**
```json
{
  "connectors": [
    {
      "type": "github",
      "display_name": "GitHub",
      "is_connected": false,
      "enabled": true
    },
    {
      "type": "gmail",
      "display_name": "Gmail",
      "is_connected": false,
      "enabled": true
    }
  ]
}
```

### 6.2 Test OAuth Flow

1. Open the web app: `http://localhost:3001`
2. Navigate to **Profile** → **Connectors** tab
3. Click **Connect** on GitHub
4. You should be redirected to GitHub authorization page
5. After authorizing, you should be redirected back with success message

### 6.3 Check Logs for Errors

```bash
# LLM API logs
docker logs server-llm-api-1 2>&1 | grep -i "connector\|oauth"

# MCP Tools logs
docker logs server-mcp-tools-1 2>&1 | grep -i "github\|google"
```

---

## Troubleshooting

### GitHub Issues

#### "redirect_uri_mismatch" Error
- **Cause:** Callback URL in GitHub app doesn't match server configuration
- **Fix:** Ensure `OAUTH_REDIRECT_BASE_URL` matches the callback URL in GitHub OAuth App settings

#### "Bad credentials" Error
- **Cause:** Invalid or expired client secret
- **Fix:** Generate a new client secret in GitHub OAuth App settings

### Google Issues

#### "Access blocked: Authorization Error"
- **Cause:** OAuth consent screen not configured or app not verified
- **Fix:**
  1. Complete OAuth consent screen setup
  2. Add test users for unverified apps
  3. Submit for verification for production use

#### "redirect_uri_mismatch" Error
- **Cause:** Redirect URI not in authorized list
- **Fix:** Add the exact redirect URI to Google Cloud Console credentials

#### "Access Not Configured" Error
- **Cause:** Required API not enabled
- **Fix:** Enable Gmail API, Drive API, and Calendar API in Google Cloud Console

### General Issues

#### "Connector not enabled"
- **Cause:** Environment variable not set
- **Fix:** Ensure `GITHUB_ENABLED=true` or `GOOGLE_ENABLED=true` is set

#### "Token encryption failed"
- **Cause:** Invalid encryption key
- **Fix:** Ensure `CONNECTOR_ENCRYPTION_KEY` is a valid 32-byte base64 string

#### Tokens Not Persisting After Restart
- **Cause:** Database not persisted or encryption key changed
- **Fix:**
  1. Ensure PostgreSQL data volume is persisted
  2. Never change encryption key after deployment

---

## Security Best Practices

### 1. Protect OAuth Credentials

- Never commit OAuth secrets to version control
- Use secret management tools (Vault, AWS Secrets Manager, etc.)
- Rotate secrets periodically

### 2. Restrict OAuth App Permissions

**GitHub:**
- Only request necessary scopes
- Use organization OAuth app policies to restrict access

**Google:**
- Use minimal scopes (readonly where possible)
- Enable domain-wide delegation only if required

### 3. Monitor OAuth Usage

- Enable audit logging
- Monitor for unusual access patterns
- Set up alerts for failed authentication attempts

### 4. Production Checklist

- [ ] Use HTTPS for all OAuth redirect URIs
- [ ] Verify Google OAuth app for production
- [ ] Configure proper CORS settings
- [ ] Enable rate limiting on OAuth endpoints
- [ ] Set up monitoring and alerting
- [ ] Document recovery procedures for key rotation

---

## Updating Credentials

### Rotating GitHub Client Secret

1. Go to GitHub OAuth App settings
2. Click **"Generate a new client secret"**
3. Update `GITHUB_CLIENT_SECRET` in environment
4. Restart services
5. Delete old secret from GitHub

### Rotating Google Client Secret

1. Go to Google Cloud Console → Credentials
2. Edit the OAuth 2.0 Client ID
3. Click **"Add Secret"** to create new secret
4. Update `GOOGLE_CLIENT_SECRET` in environment
5. Restart services
6. Delete old secret from Google Console

### Rotating Encryption Key

> **Warning:** Rotating the encryption key will invalidate all existing OAuth tokens. Users will need to reconnect their accounts.

1. Generate new encryption key
2. Plan maintenance window
3. Update `CONNECTOR_ENCRYPTION_KEY`
4. Restart services
5. Notify users to reconnect accounts

---

## Support

For issues with connector setup:

1. Check the [Troubleshooting](#troubleshooting) section above
2. Review logs for specific error messages
3. Consult the [Connectors Guide](./connectors.md) for technical details
4. Open an issue on GitHub with:
   - Error messages from logs
   - Environment configuration (redact secrets)
   - Steps to reproduce

---

## Quick Reference

### Environment Variables Summary

| Variable | Required | Description |
|----------|----------|-------------|
| `CONNECTOR_ENCRYPTION_KEY` | Yes | 32-byte base64 key for token encryption |
| `GITHUB_CLIENT_ID` | For GitHub | GitHub OAuth App client ID |
| `GITHUB_CLIENT_SECRET` | For GitHub | GitHub OAuth App client secret |
| `GITHUB_ENABLED` | No | Enable GitHub connector (default: false) |
| `GOOGLE_CLIENT_ID` | For Google | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | For Google | Google OAuth client secret |
| `GOOGLE_ENABLED` | No | Enable Google connectors (default: false) |
| `OAUTH_REDIRECT_BASE_URL` | Yes | Base URL for OAuth callbacks |
| `OAUTH_FRONTEND_URL` | Yes | Frontend URL for post-auth redirect |

### Callback URLs

| Connector | Callback URL Path |
|-----------|-------------------|
| GitHub | `/api/v1/connectors/github/callback` |
| Gmail | `/api/v1/connectors/gmail/callback` |
| Google Drive | `/api/v1/connectors/google_drive/callback` |
| Google Calendar | `/api/v1/connectors/google_calendar/callback` |

### Google API Scopes

| Service | Scope |
|---------|-------|
| Gmail (read) | `https://www.googleapis.com/auth/gmail.readonly` |
| Drive (read) | `https://www.googleapis.com/auth/drive.readonly` |
| Drive (metadata) | `https://www.googleapis.com/auth/drive.metadata.readonly` |
| Calendar (read) | `https://www.googleapis.com/auth/calendar.readonly` |
| Calendar (write) | `https://www.googleapis.com/auth/calendar.events` |
