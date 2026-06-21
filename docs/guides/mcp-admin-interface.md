# MCP Admin Interface Guide

Complete guide for administrators managing MCP (Model Context Protocol) tools without code changes.

## Overview

The MCP Admin Interface allows administrators to dynamically configure MCP tools through the LLM API, without requiring service restarts or code deployments. This guide covers:

- Viewing and listing MCP tools
- Editing tool configurations
- Enabling/disabling tools
- Setting content filtering rules
- Managing tool access and permissions

## Architecture

```
Admin client (authenticated with an admin JWT)
    ↓
LLM API  (/v1/admin/mcp-tools, RequireAdmin)
    ↓
Database (admin_mcp_tools table)
    ↓
MCP Tools Service (Tool Registration)
```

## Quick Start

### List Available MCP Tools

**GET** `/v1/admin/mcp-tools`

View all MCP tools available in your system.

```bash
curl -H "Authorization: Bearer <admin-token>" \
  http://localhost:8000/v1/admin/mcp-tools
```

Response:

```json
{
  "data": [
    {
      "id": "tool_1",
      "tool_key": "google_search",
      "name": "Google Search",
      "description": "Search the web using Google",
      "category": "search",
      "enabled": true,
      "disallowed_keywords": []
    },
    {
      "id": "tool_2",
      "tool_key": "python_exec",
      "name": "Python Executor",
      "description": "Execute Python code securely",
      "category": "code",
      "enabled": true,
      "disallowed_keywords": ["delete", "drop", "rm"]
    }
  ]
}
```

### Enable/Disable a Tool

**PATCH** `/v1/admin/mcp-tools/{tool_id}`

Enable or disable a tool for all users.

```bash
curl -X PATCH http://localhost:8000/v1/admin/mcp-tools/tool_1 \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": false
  }'
```

## Managing MCP Tools

### Get Tool Details

**GET** `/v1/admin/mcp-tools/{tool_id}`

View complete configuration for a specific tool.

```bash
curl -H "Authorization: Bearer <admin-token>" \
  http://localhost:8000/v1/admin/mcp-tools/tool_2
```

Response:

```json
{
  "id": "tool_2",
  "tool_key": "python_exec",
  "name": "Python Executor",
  "description": "Execute Python code",
  "category": "code",
  "enabled": true,
  "disallowed_keywords": ["delete", "drop", "format", "rm.*recursive"],
  "created_at": "2025-12-20T10:00:00Z",
  "updated_at": "2025-12-23T11:00:00Z"
}
```

### Update Tool Configuration

**PATCH** `/v1/admin/mcp-tools/{tool_id}`

Modify tool name, description, or enable/disable status.

```bash
curl -X PATCH http://localhost:8000/v1/admin/mcp-tools/tool_2 \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Safe Python Executor",
    "description": "Execute sandboxed Python code",
    "enabled": true
  }'
```

### Set Content Filtering Rules

Content filters are part of the tool configuration - set them with the same **PATCH** `/v1/admin/mcp-tools/{tool_id}` endpoint by sending `disallowed_keywords`. There is no separate `/filters` endpoint.

Configure regex patterns to block certain keywords/patterns in tool arguments:

```bash
curl -X PATCH http://localhost:8000/v1/admin/mcp-tools/tool_2 \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "disallowed_keywords": [
      "^delete\\s+",
      "^drop\\s+",
      "^rm\\s+-rf",
      "os\\.remove",
      "__import__\\(.*os.*\\)"
    ]
  }'
```

## Content Filtering

### Filter Syntax

Disallowed keywords use **regex patterns** for flexible matching:

| Pattern       | Matches                            | Use Case                  |
| ------------- | ---------------------------------- | ------------------------- |
| `^delete\\s+` | "delete" at start of string        | Block SQL DELETE commands |
| `drop`        | "drop" anywhere (case-insensitive) | Block DROP statements     |
| `os\\.remove` | "os.remove" in Python code         | Block file deletion       |
| `__import__`  | Python import statements           | Restrict imports          |
| `eval\\(`     | "eval(" function calls             | Block dangerous eval      |
| `exec\\(`     | "exec(" function calls             | Block exec calls          |

### Common Filter Sets

#### Python Code Execution - Safe Mode

Block potentially dangerous operations:

```json
{
  "disallowed_keywords": [
    "^import os",
    "^import subprocess",
    "^import socket",
    "^import requests",
    "os\\.system",
    "os\\.remove",
    "subprocess\\.call",
    "subprocess\\.run",
    "open\\(",
    "__import__"
  ]
}
```

#### Web Search - Content Filter

Block adult/NSFW searches:

```json
{
  "disallowed_keywords": [
    "adult",
    "nsfw",
    "porn",
    "xxx",
    "sex.*site",
    "18\\+.*content"
  ]
}
```

#### General Safety Rules

```json
{
  "disallowed_keywords": [
    "delete",
    "drop",
    "truncate",
    "destroy",
    "rm\\s+-rf",
    "format.*drive"
  ]
}
```

## JavaScript Examples

### Bulk Tool Status Update

```javascript
const adminToken = "your_admin_token";
const headers = {
  Authorization: `Bearer ${adminToken}`,
  "Content-Type": "application/json",
};

// Fetch all tools
const listResponse = await fetch("http://localhost:8000/v1/admin/mcp-tools", {
  headers,
});

const tools = await listResponse.json();

// Disable all code execution tools
for (const tool of tools.data) {
  if (tool.category === "code") {
    await fetch(`http://localhost:8000/v1/admin/mcp-tools/${tool.id}`, {
      method: "PATCH",
      headers,
      body: JSON.stringify({ enabled: false }),
    });
    console.log(`Disabled: ${tool.name}`);
  }
}
```

## Access Control

### Admin Authentication

The MCP Admin endpoints require an admin JWT. Authentication is a Keycloak OAuth2 redirect, not an email/password POST. Initiate it via:

```bash
# Returns the Keycloak authorization URL (+ state) for the frontend to redirect to
curl "http://localhost:8000/auth/login?redirect_url=http://localhost:3001"
```

The user signs in on Keycloak (authorization-code flow with PKCE) and the resulting access token is used as the Bearer token for admin operations.

### Role-Based Access Control (RBAC)

MCP Admin endpoints are guarded by `middleware.RequireAdmin()`, which requires the Keycloak **realm role `admin`** (or an `is_admin` attribute on the principal). There is no `mcp:admin` scope or `mcp_admin` group.

To grant access in Keycloak:

1. Define (or use) the realm role `admin` in the `jan` realm.
2. Assign that role to the admin user.
3. The role appears in the token's realm-access claims, and `RequireAdmin` allows the request.

## Troubleshooting

### "Permission Denied" Error

**Problem**: Receiving 403 Forbidden when accessing admin endpoints

**Solutions**:

1. Verify the token carries the realm role `admin`
2. Check token hasn't expired: `jwt decode YOUR_TOKEN`
3. Ensure the user has the `admin` realm role assigned in Keycloak

### Filter Not Applied

**Problem**: Content filter isn't blocking expected keywords

**Solutions**:

1. Verify regex syntax is correct
2. Test regex pattern independently
3. Remember regexes are case-sensitive (use `(?i)` for case-insensitive)
4. Check filter is applied to correct tool

### Tool Not Appearing

**Problem**: Tool exists but doesn't show in admin list

**Solutions**:

1. Verify MCP Tools service is running
2. Check service has registered the tool
3. Ensure tool registration completed (may need restart)

## Best Practices

### 1. Enable Only Needed Tools

Start with minimal tool set, enable others as needed:

```bash
# Disable all tools initially
curl -X PATCH http://localhost:8000/v1/admin/mcp-tools/web_scraper \
  -H "Authorization: Bearer <token>" \
  -d '{"enabled": false}'
```

### 2. Test Filters Before Deployment

Test filter patterns against actual use cases:

```bash
# Pattern to test: "os\\.system"
# Test inputs:
# - "os.system('ls')"     → BLOCKED ✓
# - "my_os.system_call()" → ALLOWED ✓
```

### 3. Audit Tool Usage

Monitor which tools users are accessing:

```bash
# Check MCP Tools service logs
docker logs mcp-tools | grep "tool_call"
```

### 4. Regular Security Updates

Review tool filters quarterly as new security threats emerge.

## Related Documentation

- [MCP Tools API Reference](../api/mcp-tools/README.md) - Tool execution API
- [Authentication Guide](authentication.md) - Admin authentication
- [Security Guide](../architecture/security.md) - Security best practices
- [Deployment Guide](deployment.md) - Production configuration

## Support

For admin-related issues:

- Check [Troubleshooting Guide](troubleshooting.md)
- Review [MCP Tools Documentation](../api/mcp-tools/README.md)
- Contact your Jan Server administrator

---

**Role Required**: Keycloak realm role `admin`
